#!/usr/bin/env python3
"""Generate local-query commands from a fixed synthetic corpus and replay them."""
import argparse
import concurrent.futures
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys

TASKS = {
    "inventory": (
        "Use fluent catalog queries, correlated count(), a scoped fragment that "
        "adds a minimum-tool filter to the projected query, and stable session ID "
        "ordering. The minimum is zero. Return fields session_id and tool_count. "
        "Count physical tool_calls rows for each session, including zero-tool sessions.",
        [{"session_id": "empty", "tool_count": 0},
         {"session_id": "single", "tool_count": 1},
         {"session_id": "two", "tool_count": 2}],
    ),
    "report": (
        "Use tagged SQL with a named integer parameter called minimum, declared "
        "columns, then mapRows() to format labels. Use a session-rooted count of "
        "physical tool_calls and apply the minimum to the result. Execute with "
        "minimum=0 and return fields session_id and label. Format the label as "
        "'0 tools', '1 tools', or '2 tools'. Preserve session ID order.",
        [{"session_id": "empty", "label": "0 tools"},
         {"session_id": "single", "label": "1 tools"},
         {"session_id": "two", "label": "2 tools"}],
    ),
}
MODELS = ("glm-5.3-flash", "deepseek-4.1-flash")
LEVELS = ("low", "high")


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def run_logged(argv, cwd, output, timeout):
    with output.open("wb") as stdout, output.with_suffix(".stderr").open("wb") as stderr:
        try:
            return subprocess.run(argv, cwd=cwd, stdout=stdout, stderr=stderr,
                                  timeout=timeout, check=False).returncode
        except subprocess.TimeoutExpired:
            return 124


def sandbox(profile, root, run, binary):
    # The generated JS executes without network and cannot read user files.
    readable = [str(binary), str(root / "inputs"), str(run)]
    protected_reads = ["/Users", "/Volumes", "/private/tmp", "/private/var/folders"]
    rules = ["(version 1)", "(allow default)", "(deny network*)",
             "(deny file-write*)", "(deny process-exec)",
             "(allow process-exec (literal " + json.dumps(str(binary)) + "))"]
    rules += ["(deny file-read-data (subpath " + json.dumps(path) + "))"
              for path in protected_reads]
    rules += ["(allow file-read-data (subpath " + json.dumps(path) + "))" for path in readable]
    rules += ["(allow file-write* (subpath " + json.dumps(str(run)) + "))",
              '(allow file-write* (literal "/dev/null"))']
    profile.write_text("\n".join(rules) + "\n")


def replay(work, root, binary):
    policy = work / "replay.sb"
    sandbox(policy, root, work, binary)
    output = work / "replay.json"
    argv = ["/usr/bin/sandbox-exec", "-f", str(policy), str(binary), "query", "commands",
            "--query-repository", str(work / "commands"), "report", "--archive-glob",
            str(root / "inputs" / "*.minitrace.json"), "--output", "json"]
    return argv, run_logged(argv, work, output, 20), output


def grade_replay(work, root, binary, task):
    argv, code, output = replay(work, root, binary)
    result = {"argv": argv, "replay_exit": code, "replay_sha256": digest(output), "passed": False}
    try:
        rows = json.loads(output.read_text())
        result["rows"] = rows
        result["passed"] = code == 0 and rows == TASKS[task][1]
    except ValueError:
        result["error"] = "replay did not emit JSON"
    return result


def probe(root, binary, run):
    """Use an ordinary JS command to verify private reads and input writes fail."""
    work = root / run / "sandbox-probe"
    commands = work / "commands"
    commands.mkdir(parents=True)
    (commands / "report.js").write_text(
        "function report() { const fs = require('fs');"
        " let readDenied = false, writeDenied = false;"
        f" try {{ fs.readFileSync({json.dumps(str(root / 'pin.json'))}, 'utf8'); }}"
        " catch (error) { readDenied = true; }"
        f" try {{ fs.writeFileSync({json.dumps(str(root / 'inputs/probe'))}, 'blocked'); }}"
        " catch (error) { writeDenied = true; }"
        " return {readDenied, writeDenied}; }"
        " __verb__('report', {name: 'report'});"
    )
    _, code, output = replay(work, root, binary)
    if code != 0 or json.loads(output.read_text()) != [{"readDenied": True, "writeDenied": True}]:
        raise RuntimeError("replay sandbox probe failed: " + str(work))
    if (root / "inputs/probe").exists():
        raise RuntimeError("replay sandbox modified a protected input directory")


def run_one(spec, root, binary, reference, extension, cohort):
    task, model, thinking = spec
    description, expected = TASKS[task]
    work = root / cohort / f"{task}-{model}-{thinking}"
    work.mkdir(parents=True)
    prompt = work / "prompt.md"
    prompt.write_text(
        f"You are a transcript analysis developer using the published go-minitrace JS API.\n"
        "Output one complete JavaScript script in a single fenced block, with no prose.\n"
        "Define one top-level function named report and register it with "
        '__verb__("report", {name:"report"}). The filename will be report.js.\n'
        "Use require('minitrace'), runtimeArchives(), db.execute(plan), check the "
        "result envelope, return rows, and close the database. The input corpus "
        "contains normalized synthetic archives with some zero-tool sessions.\n"
        f"Task: {description}\n\n"
        f"Published API reference:\n{reference}"
    )
    model_output = work / "answer.txt"
    model_argv = [
        "pi", "--provider", "lunaroute", "--model", model, "--thinking", thinking,
        "--print", "--mode", "text", "--no-tools", "--no-context-files",
        "--no-skills", "--no-extensions", "-e", str(extension), "--no-prompt-templates",
        "--session-dir", str(work / "sessions"), "@" + str(prompt),
    ]
    model_exit = run_logged(model_argv, work, model_output, 180)
    result = {"task": task, "model": model, "thinking": thinking, "model_exit": model_exit,
              "prompt_sha256": digest(prompt), "answer_sha256": digest(model_output), "passed": False}
    if model_exit == 0:
        response = model_output.read_text().strip()
        fence = re.fullmatch(r"```(?:javascript|js)?\s*\n(.*?)\n```", response, re.S)
        if fence is None:
            result["error"] = "expected a single fenced script"
        else:
            commands = work / "commands"
            commands.mkdir()
            script = commands / "report.js"
            script.write_text(fence[1] + "\n")
            result["script_sha256"] = digest(script)
            result.update(grade_replay(work, root, binary, task))
    save(work / "receipt.json", result)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--cohort", choices=["pilot", "pilot-replay", "matrix", "followup"], required=True)
    parser.add_argument("--workers", type=int, default=4)
    args = parser.parse_args()
    root = args.root.resolve()
    root.mkdir(parents=True, exist_ok=True)
    (root / args.cohort).mkdir(exist_ok=False)
    extension = Path.home() / ".pi/agent/npm/node_modules/@lunaroute/pi-extension/src/index.ts"
    if not extension.is_file() or not Path("/usr/bin/sandbox-exec").is_file():
        parser.error("local Lunaroute extension and macOS sandbox-exec are required")
    binary = root / "go-minitrace"
    inputs = root / "inputs"
    pin = root / "pin.json"
    if args.cohort in ("pilot", "followup"):
        inputs.mkdir(exist_ok=False)
        subprocess.run(["go", "build", "-o", str(binary), "./cmd/go-minitrace"], check=True)
        for name, count in (("empty", 0), ("single", 1), ("two", 2)):
            save(inputs / f"{name}.minitrace.json", {
                "id": name, "profile": "organic", "turns": [],
                "tool_calls": [{"id": f"{name}-{i}", "tool_name": "read"} for i in range(count)],
            })
        (root / "api.md").write_bytes(Path("pkg/doc/js-api-reference.md").read_bytes())
        paths = [binary, root / "api.md", *sorted(inputs.iterdir())]
        save(pin, {"revision": subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip(),
                   "hashes": {str(path): digest(path) for path in paths}})
    pinned = json.loads(pin.read_text())
    for path, expected in pinned["hashes"].items():
        if digest(Path(path)) != expected:
            raise RuntimeError("pinned input drift: " + path)
    probe(root, binary, args.cohort)
    reference = (root / "api.md").read_text()
    if args.cohort == "pilot-replay":
        results = []
        for task in TASKS:
            name = f"{task}-{MODELS[0]}-low"
            original = root / "pilot" / name
            work = root / args.cohort / name
            work.mkdir()
            shutil.copytree(original / "commands", work / "commands")
            receipt = json.loads((original / "receipt.json").read_text())
            receipt.pop("error", None)
            receipt.update(grade_replay(work, root, binary, task))
            save(work / "receipt.json", receipt)
            results.append(receipt)
    else:
        specs = [(task, MODELS[0], "low") for task in TASKS]
        if args.cohort == "followup":
            specs = [("report", MODELS[0], thinking) for thinking in LEVELS]
        if args.cohort == "matrix":
            specs = [(task, model, thinking) for task in TASKS for model in MODELS for thinking in LEVELS]
        with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as executor:
            futures = [executor.submit(run_one, spec, root, binary, reference, extension, args.cohort) for spec in specs]
            results = [future.result() for future in futures]
    for path, expected in pinned["hashes"].items():
        if digest(Path(path)) != expected:
            raise RuntimeError("pinned input drift: " + path)
    summary = {"cohort": args.cohort, "pin": pinned, "runs": results,
               "passed": sum(run["passed"] for run in results), "attempted": len(results)}
    save(root / args.cohort / "summary.json", summary)
    print(json.dumps({key: summary[key] for key in ("cohort", "passed", "attempted")}))
    return 0 if summary["passed"] == summary["attempted"] else 1


if __name__ == "__main__":
    sys.exit(main())
