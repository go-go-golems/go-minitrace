#!/usr/bin/env python3
"""Generate the synthetic Claude Code fixture tree used by go-minitrace tests.

Usage: python3 01-generate-claude-code-fixture.py <repo-root>
Writes testdata/claude-code/projects/-tmp-demo/. Every value is synthetic.
"""
import json
import os
import sys

root = os.path.join(sys.argv[1], "testdata", "claude-code", "projects", "-tmp-demo")
PARENT = "11111111-1111-4111-8111-111111111111"
STUB = "22222222-2222-4222-8222-222222222222"
CWD = "/tmp/demo"
CLAUDE_MD = "# Demo\n\nRun tests with make test.\n"
RULE = "---\npaths: [\"src/**\"]\n---\nPrefer small functions.\n"
SKILL = "# Diary\n\nKeep a diary.\n"


def base(uuid, parent, ts, **extra):
    record = {"parentUuid": parent, "isSidechain": False, "uuid": uuid, "timestamp": ts,
              "sessionId": PARENT, "cwd": CWD, "version": "2.1.330", "gitBranch": "main",
              "userType": "external", "entrypoint": "cli"}
    record.update(extra)
    return record


def user_text(uuid, parent, ts, text):
    return base(uuid, parent, ts, type="user", message={"role": "user", "content": text})


def assistant(uuid, parent, ts, blocks, model="claude-test-1"):
    return base(uuid, parent, ts, type="assistant", message={
        "role": "assistant", "model": model, "content": blocks,
        "usage": {"input_tokens": 10, "output_tokens": 5, "cache_read_input_tokens": 100}})


def tool_use(tool_id, name, tool_input):
    return {"type": "tool_use", "id": tool_id, "name": name, "input": tool_input}


def tool_result(uuid, parent, ts, tool_id, text, is_error=False, tur=None):
    record = base(uuid, parent, ts, type="user", message={"role": "user", "content": [
        {"type": "tool_result", "tool_use_id": tool_id, "content": text, "is_error": is_error}]})
    if tur is not None:
        record["toolUseResult"] = tur
    return record


def attachment(uuid, parent, ts, payload):
    return base(uuid, parent, ts, type="attachment", attachment=payload)


def system(uuid, parent, ts, subtype, **extra):
    return base(uuid, parent, ts, type="system", subtype=subtype, isMeta=False, **extra)


parent = [
    {"type": "permission-mode", "permissionMode": "auto", "sessionId": PARENT},
    attachment("a-instr", None, "2026-11-20T10:00:00.000Z", {"type": "instructions", "files": [
        {"path": CWD + "/CLAUDE.md", "type": "Project", "content": CLAUDE_MD},
        {"path": "/tmp/home/.claude/projects/-tmp-demo/memory/MEMORY.md", "type": "AutoMem", "content": "- note\n"}]}),
    user_text("u-1", "a-instr", "2026-11-20T10:00:01.000Z", "Fix the bug and run tests."),
    assistant("as-1", "u-1", "2026-11-20T10:00:05.000Z", [
        tool_use("toolu_agent1", "Agent", {"subagent_type": "Explore", "description": "Find bug", "prompt": "Find the bug"}),
        tool_use("toolu_agent2", "Agent", {"subagent_type": "general-purpose", "description": "Review", "prompt": "Review the fix"}),
        tool_use("toolu_bash1", "Bash", {"command": "make test FILE=src/foo.test.ts"})]),
    attachment("a-hook", "as-1", "2026-11-20T10:00:05.500Z", {
        "type": "hook_success", "hookName": "PreToolUse:Bash", "toolUseID": "toolu_bash1", "hookEvent": "PreToolUse",
        "content": "", "stdout": json.dumps({"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": "use the test target"}}),
        "stderr": "", "exitCode": 0, "command": "guard.py", "durationMs": 42}),
    tool_result("u-tr-agent1", "as-1", "2026-11-20T10:01:00.000Z", "toolu_agent1", "Async agent launched",
                tur={"isAsync": True, "status": "async_launched", "agentId": "a1111111111111111", "description": "Find bug"}),
    tool_result("u-tr-agent2", "u-tr-agent1", "2026-11-20T10:02:00.000Z", "toolu_agent2", "The fix looks right."),
    tool_result("u-tr-bash1", "u-tr-agent2", "2026-11-20T10:02:01.000Z", "toolu_bash1",
                "The command did not run. A PreToolUse hook blocked it before execution, with this message:\n\n> use the test target",
                tur={"stdout": "The command did not run.", "stderr": "", "interrupted": False, "isImage": False}),
    attachment("a-nested-1", "u-tr-bash1", "2026-11-20T10:02:01.100Z", {
        "type": "nested_memory", "path": CWD + "/src/CLAUDE.md", "displayPath": "src/CLAUDE.md",
        "content": {"path": CWD + "/src/CLAUDE.md", "type": "Project", "content": CLAUDE_MD, "contentDiffersFromDisk": False}}),
    attachment("a-nested-2", "a-nested-1", "2026-11-20T10:02:01.200Z", {
        "type": "nested_memory", "path": CWD + "/.claude/rules/style.md", "displayPath": ".claude/rules/style.md",
        "content": {"path": CWD + "/.claude/rules/style.md", "type": "Project", "content": RULE}}),
    assistant("as-2", "a-nested-2", "2026-11-20T10:02:10.000Z", [{"type": "text", "text": "The hook blocked the test run."}]),
    system("s-td-1", "as-2", "2026-11-20T10:02:11.000Z", "turn_duration", durationMs=130000, messageCount=12),
    system("s-cb", None, "2026-11-20T10:30:00.000Z", "compact_boundary", content="Conversation compacted", level="info",
           logicalParentUuid="s-td-1", compactMetadata={"trigger": "auto", "preTokens": 400000, "postTokens": 50000, "durationMs": 23}),
    user_text("u-2", "s-cb", "2026-11-20T10:30:00.100Z", "This session is being continued from a previous conversation."),
    attachment("a-skills", "u-2", "2026-11-20T10:30:00.200Z", {"type": "invoked_skills", "skills": [
        {"name": "diary", "path": "userSettings:diary", "content": SKILL}]}),
    assistant("as-3", "a-skills", "2026-11-20T10:30:05.000Z", [tool_use("toolu_bash2", "Bash", {"command": "cat /etc/secret"})]),
    tool_result("u-tr-bash2", "as-3", "2026-11-20T10:30:06.000Z", "toolu_bash2",
                "Permission for this action was denied by the Claude Code auto mode classifier. Reason: [Unauthorized Access].",
                is_error=True, tur="Error: Permission for this action was denied by the Claude Code auto mode classifier."),
    assistant("as-4", "u-tr-bash2", "2026-11-20T10:30:10.000Z", [tool_use("toolu_grep", "Bash", {"command": "grep -rn 'did not run' docs"})]),
    tool_result("u-tr-grep", "as-4", "2026-11-20T10:30:11.000Z", "toolu_grep",
                "docs/a.md:3:The command did not run. A PreToolUse hook blocked it", tur={"stdout": "docs/a.md:3", "stderr": "", "interrupted": False}),
    system("s-stop", "u-tr-grep", "2026-11-20T10:30:12.000Z", "stop_hook_summary", hookCount=1,
           hookInfos=[{"command": "stop-check.sh"}], hookErrors=[], preventedContinuation=False, stopReason="", toolUseID="toolu_grep", level="suggestion"),
    assistant("as-5", "s-stop", "2026-11-20T10:30:20.000Z", [{"type": "text", "text": "Done."}]),
    system("s-td-2", "as-5", "2026-11-20T10:30:21.000Z", "turn_duration", durationMs=21000, messageCount=8),
]


def sidechain(records, agent_id):
    for record in records:
        record["isSidechain"] = True
        record["agentId"] = agent_id
    return records


sub1 = sidechain([
    user_text("s1-u", None, "2026-11-20T10:00:10.000Z", "Find the bug"),
    assistant("s1-a1", "s1-u", "2026-11-20T10:00:20.000Z", [tool_use("toolu_s1_read", "Read", {"file_path": CWD + "/src/foo.ts"})]),
    tool_result("s1-tr", "s1-a1", "2026-11-20T10:00:21.000Z", "toolu_s1_read", "export const foo = 1"),
    assistant("s1-a2", "s1-tr", "2026-11-20T10:00:30.000Z", [{"type": "text", "text": "Found it."}]),
], "a1111111111111111")
sub2 = sidechain([
    user_text("s2-u", None, "2026-11-20T10:01:10.000Z", "Review the fix"),
    assistant("s2-a1", "s2-u", "2026-11-20T10:01:50.000Z", [{"type": "text", "text": "The fix looks right."}]),
], "a2222222222222222")
journal = [{"type": "launched"}, {"type": "started", "agentId": "a3333333333333333", "label": "phase:1"}]
stub = [
    {"type": "mode", "mode": "normal", "sessionId": STUB},
    {"type": "permission-mode", "permissionMode": "auto", "sessionId": STUB},
    {"parentUuid": None, "isSidechain": False, "type": "system", "subtype": "local_command",
     "content": "<command-name>/resume</command-name>", "level": "info", "timestamp": "2026-11-21T09:00:00.000Z",
     "uuid": "stub-1", "isMeta": False, "cwd": CWD, "sessionId": STUB, "version": "2.1.330"},
    {"type": "cost-state", "sessionId": STUB, "totalCostUSD": 0, "totalAPIDuration": 0, "totalToolDuration": 0},
    {"type": "last-prompt", "leafUuid": "stub-1", "sessionId": STUB},
]


def write_jsonl(path, records):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as handle:
        for record in records:
            handle.write(json.dumps(record, separators=(",", ":")) + "\n")


def write_json(path, payload):
    with open(path, "w") as handle:
        json.dump(payload, handle, separators=(",", ":"))
        handle.write("\n")


subagents = os.path.join(root, PARENT, "subagents")
write_jsonl(os.path.join(root, PARENT + ".jsonl"), parent)
write_jsonl(os.path.join(subagents, "agent-a1111111111111111.jsonl"), sub1)
write_json(os.path.join(subagents, "agent-a1111111111111111.meta.json"),
           {"agentType": "Explore", "description": "Find bug", "toolUseId": "toolu_agent1"})
write_jsonl(os.path.join(subagents, "agent-a2222222222222222.jsonl"), sub2)
write_json(os.path.join(subagents, "agent-a2222222222222222.meta.json"),
           {"agentType": "general-purpose", "description": "Review", "toolUseId": "toolu_agent2"})
write_jsonl(os.path.join(subagents, "workflows", "wf_1", "journal.jsonl"), journal)
write_jsonl(os.path.join(root, STUB + ".jsonl"), stub)
