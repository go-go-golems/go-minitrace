"""Build bounded API-friction evidence and run the independent Codex judge."""
from collections import Counter
import json
from pathlib import Path
import subprocess

JUDGE_MODEL = "gpt-6.2-sol"
JUDGE_THINKING = "medium"


def records(path):
    with path.open() as handle:
        for line in handle:
            try:
                yield json.loads(line)
            except ValueError:
                continue


def tool_evidence(work):
    tool_counts = Counter()
    signals = Counter()
    by_phase = {}
    for path in sorted(work.glob("analyst-*.pi.jsonl")):
        calls = {}
        phase = Counter()
        for event in records(path):
            if event.get("type") == "tool_execution_start":
                tool = event.get("toolName")
                tool_counts[tool] += 1
                calls[event.get("toolCallId")] = (event.get("args") or {}).get("kind")
                continue
            if event.get("type") != "tool_execution_end" or event.get("toolName") not in ("minitrace", "save_artifact"):
                continue
            blocks = (event.get("result") or {}).get("content", [])
            try:
                result = json.loads(blocks[0]["text"])
            except (KeyError, IndexError, TypeError, ValueError):
                if event.get("isError"):
                    phase["Pi tool validation failure"] += 1
                continue
            if not isinstance(result, dict):
                continue
            if event.get("toolName") == "save_artifact":
                if result.get("paths"):
                    phase["save returned registered command paths"] += 1
                if result.get("diagnostics"):
                    phase["save returned catalog diagnostics"] += 1
                continue
            if "exit" not in result and ("tables" in result or "columns" in result):
                phase["successful schema lookup"] += 1
                continue
            if result.get("previewTruncated", result.get("deliveryTruncated")):
                phase["delivery preview shortened"] += 1
            if result.get("queryComplete") is False:
                phase["query envelope incomplete"] += 1
            status = result.get("exit")
            if status == 0:
                phase[f"successful {calls.get(event.get('toolCallId')) or 'query'} invocation"] += 1
                continue
            phase["nonzero or interrupted CLI invocation"] += 1
            text = result.get("stderr", "")
            for label, pattern in (
                ("unresolved command reported as --archive-glob", "unknown flag: --archive-glob"),
                ("unresolved command with actionable diagnostic", "available commands:"),
                ("scanner skipped invalid command declaration", "query catalog: skipped"),
                ("SQLite schema introspection denied", 'disallowed table/view "sqlite_master"'),
                ("duplicate command registration", "duplicate js verb path"),
                ("strict query incomplete", "strict evidence"),
                ("operation denied by environment", "operation not permitted"),
            ):
                if pattern in text:
                    phase[label] += 1
            if status == "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" or result.get("processError") == "capture_limit":
                phase["tool subprocess output exceeded cap"] += 1
        signals.update(phase)
        by_phase[path.stem] = dict(phase)
    return {"tool_calls": dict(tool_counts), "signals": dict(signals), "by_phase": by_phase}


def evidence_prompt(analyst, tools, checks, scenario="file-change-ledger"):
    # Full archive rows, command strings, and free-form analyst outputs stay local.
    payload = {
        "analyst": analyst,
        "scenario": scenario,
        "observed_tool_evidence": tools,
        "host_checks": [{key: check[key] for key in (
            "turn", "scope", "passed", "expected_rows", "observed_rows", "missing_count", "unexpected_count", "check_type", "cited_targets",
        ) if key in check} for check in checks],
    }
    return """Judge the go-minitrace developer experience in this transcript-analysis task. Use only the supplied
evidence and return under 230 words. The analyst investigated the scenario, submitted a reusable JS command, answered a
stakeholder follow-up, and reran it on a narrower archive selection. The host independently checked exact returned rows.

Distinguish CLI/API defects, documentation or discoverability problems, analyst script mistakes, and harness
limitations. Prioritize at most three specific improvements and identify what needs a rerun. Repeated attempts
in one run do not establish prevalence. Signal counts may overlap for one tool result. Successful execution
does not imply complete evidence. A later isolated check does not change the original live check.

Product contracts: go-minitrace scans static __verb__("ledger", {name: "ledger", short: "Run the ledger"}) metadata and binds a
top-level function. A matching JS filename and sole verb collapse to one CLI word. A grouped leaf takes two
or more words. For the historical runs the wrapper accepted a single word and supplied --archive-glob.
The CLI has since gained missing-command diagnostics and the wrapper accepts grouped paths and catalog help.
Normalized SQLite denies sqlite_master. Supported alternatives include schema help and JS db.schema().
Current batches have structured schema lookups and return registered paths when saving JS artifacts.
The harness has preview and subprocess output caps. Historical trials staged normalized archives only.
Use no tools. Do not infer rows or claims that are absent from the supplied evidence.

Evidence:
""" + json.dumps(payload, indent=2) + "\n"


def codex_judge(args, work, prompt, emit):
    phase = "api-judgment"
    profile = {"provider": "openai-codex", "model": args.judge_model, "thinking": args.judge_thinking}
    (work / f"{phase}.prompt.md").write_text(prompt)
    cwd = work / "judge-cwd"
    cwd.mkdir(exist_ok=True)
    argv = [
        args.codex, "-a", "never", "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral",
        "--skip-git-repo-check", "--sandbox", "read-only", "--cd", str(cwd), "--model", args.judge_model,
        "-c", "model_reasoning_effort=" + json.dumps(args.judge_thinking), "--json", "-",
    ]
    emit("judge", "waiting", f"openai-codex/{args.judge_model} · {args.judge_thinking}", profile)
    raw = work / f"{phase}.codex.jsonl"
    stderr = work / f"{phase}.stderr"
    with raw.open("w") as stdout, stderr.open("w") as errors:
        result = subprocess.run(argv, cwd=cwd, input=prompt, text=True, stdout=stdout, stderr=errors,
                                timeout=args.timeout)
    reply = ""
    receipt = {"requested": profile, "exit": result.returncode, "turn_completed": False, "tool_items": []}
    for event in records(raw):
        kind = event.get("type")
        item = event.get("item") or {}
        item_type = item.get("type")
        if kind == "item.completed" and item_type == "agent_message":
            reply = item.get("text", "")
        elif item_type in ("command_execution", "mcp_tool_call", "web_search", "file_change"):
            receipt["tool_items"].append(item_type)
        elif kind == "turn.completed":
            receipt.update(turn_completed=True, usage=event.get("usage"))
    (work / f"{phase}.receipt.json").write_text(json.dumps(receipt, indent=2) + "\n")
    if result.returncode or not receipt["turn_completed"] or not reply or receipt["tool_items"]:
        raise RuntimeError(f"Codex judge failed its run contract: {receipt}. {stderr.read_text()[-900:]}")
    return reply
