"""Regression probes for the ledger comparison and incremental viewer cursor."""
import argparse
import json
from pathlib import Path
import shutil
from tempfile import TemporaryDirectory

from run import HERE, command_json, fixture, input_glob, verify
from serve import timeline
from trace import Thinking, project_run


def test_ledger(binary, root):
    archives = root / "input/archives"
    archives.mkdir(parents=True)
    fixture(archives)
    work = root / "fixture"
    artifacts = work / "artifacts"
    artifacts.mkdir(parents=True)
    source = (HERE / "fixture-ledger.js").read_text()
    command = artifacts / "ledger.js"
    command.write_text(source)
    for name in ("probe_test.js", "probe-test.js"):
        (artifacts / name).write_text('function probe() { return []; } __verb__("probe", {name:"probe"});')
    try:
        command_json([str(binary), "query", "commands", "--query-repository", str(artifacts),
            "ledger", "--archive-glob", input_glob(root, "all"), "--output", "json"], work)
    except RuntimeError as error:
        assert "duplicate js verb path" in str(error), error
    else:
        raise AssertionError("Scratch command collision did not reproduce")
    original = verify(root, work, binary, "all", offline=True)
    assert original["passed"] and original["observed_rows"] == 2, original

    # Preserve the row count while replacing a genuine record with a duplicate.
    command.write_text(source.replace("return result.rows;", "return [result.rows[0], result.rows[0]];"))
    duplicate = verify(root, work, binary, "all", offline=True)
    assert duplicate["observed_rows"] == 2 and not duplicate["passed"], duplicate
    assert duplicate["unexpected_count"] == duplicate["missing_count"] == 1, duplicate

    command.write_text(source.replace("return result.rows;", "return result.rows.slice(0, 1);"))
    missing = verify(root, work, binary, "all", offline=True)
    assert not missing["passed"] and missing["missing_count"] == 1, missing

    command.write_text(source)
    narrowed = verify(root, work, binary, "codex", offline=True)
    assert narrowed["passed"] and narrowed["observed_rows"] == 1, narrowed
    assert narrowed["preview"][0]["evidence_status"] == "attempted", narrowed


def test_cursor(root):
    work = root / "events-test"
    work.mkdir()
    log = work / "events.jsonl"
    log.write_bytes(b'{"kind":"start"}\n{"kind":"')
    first = timeline(root, {})["events-test"]
    assert first["events"] == [{"kind": "start"}], first
    with log.open("ab") as handle:
        handle.write(b'finish"}\n')
    second = timeline(root, {"events-test": first["offset"]})["events-test"]
    assert second["events"] == [{"kind": "finish"}], second
    assert not timeline(root, {"events-test": second["offset"]})["events-test"]["events"]


def test_trace(root):
    work = root / "trace-test"
    work.mkdir()
    begin = {"type": "message_update", "assistantMessageEvent": {"type": "thinking_start", "contentIndex": 0}}
    delta = {"type": "message_update", "assistantMessageEvent": {"type": "thinking_delta", "delta": "private body"}}
    end = {"type": "message_update", "assistantMessageEvent": {"type": "thinking_end", "contentIndex": 0,
                                                                    "content": "private body"}}
    thinking = Thinking()
    assert thinking.event(begin, 2.0) == ("thinking start", "Thinking started", None)
    assert thinking.event(delta, 2.2) is None
    assert thinking.event(end, 2.8) == ("thinking end", "Thinking ended", {"elapsed_seconds": 0.8})
    tool_open = {"type": "tool_execution_start", "toolName": "minitrace", "toolCallId": "example"}
    tool_close = {**tool_open, "type": "tool_execution_end"}
    message = {"type": "message_end", "message": {"role": "assistant",
                 "content": [{"type": "thinking", "thinking": "private body"},
                             {"type": "text", "text": "Public reply"}]}}

    def write(name, rows):
        (work / name).write_text("".join(json.dumps(row) + "\n" for row in rows))

    def observed(kind, text):
        return {"role": "analyst", "run": "example", "kind": kind, "text": text, "time": 10, "data": None}

    write("analyst-1.pi.jsonl", [begin, delta, end, tool_open, tool_close, begin, end, message])
    write("events.jsonl", [observed("waiting", "model · analyst-1"), observed("tool call", "minitrace"),
                         observed("tool result", "minitrace"), observed("message", "Public reply"),
                         observed("replay", "Host result")])
    rows = list(project_run(work))
    assert [row["kind"] for row in rows] == [
        "waiting", "thinking start", "thinking end", "tool call", "tool result",
        "thinking start", "thinking end", "message", "replay",
    ]
    assert rows[3]["tool_call_id"] == rows[4]["tool_call_id"] == "example"
    assert rows[1]["time"] is None and "private body" not in json.dumps(rows)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    args = parser.parse_args()
    with TemporaryDirectory(prefix="persona-lab-test-") as tmp:
        root = Path(tmp).resolve()
        binary = root / "go-minitrace"
        shutil.copy2(args.binary, binary)
        test_ledger(binary, root)
        test_cursor(root)
        test_trace(root)
    print("Ledger, partial-event cursor and chronology checks passed")
