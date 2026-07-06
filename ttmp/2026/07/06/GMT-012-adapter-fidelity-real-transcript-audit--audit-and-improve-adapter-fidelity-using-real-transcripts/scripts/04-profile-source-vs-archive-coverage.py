#!/usr/bin/env python3
"""Profile source-vs-archive coverage without copying transcript text.

This script reads ticket sample lists produced by 01-inventory-source-shapes.py,
profiles structural facts from raw sources, profiles the converted minitrace
archives, and writes aggregate JSON/Markdown reports. It is intentionally
redaction-safe: it records counts, keys, block types, and payload shapes, not
message contents or tool outputs.
"""
from __future__ import annotations

import argparse
import json
import sqlite3
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

TICKET = Path(__file__).resolve().parents[1]
INV = TICKET / "sources" / "source-shape-inventory"
CORPUS = TICKET / "sources" / "converted-corpus"
OUT = TICKET / "sources" / "coverage-profile"
LOGS = TICKET / "scripts" / "logs"

JSONL_ADAPTERS = ["pi", "codex", "claude-code", "copilot"]

KEY_HINTS = (
    "thinking", "reason", "summary", "encrypted", "usage", "token", "tool",
    "toolUseResult", "duration", "exit", "stdout", "stderr", "interrupted",
    "attachment", "image", "file", "parent", "sidechain", "agent", "subagent",
    "fork", "thread", "rate", "limit", "permission", "mode", "title", "git",
    "branch", "cwd", "version", "model", "sandbox", "approval",
)

CONTENT_KEYS = {"text", "content", "output", "stdout", "stderr", "command", "prompt", "message"}


def safe_shape(value: Any, depth: int = 0) -> Any:
    """Return a structural shape only: types, keys, list item shapes, and string length buckets."""
    if depth > 4:
        return type(value).__name__
    if isinstance(value, dict):
        return {str(k): safe_shape(v, depth + 1) for k, v in sorted(value.items())[:50]}
    if isinstance(value, list):
        if not value:
            return []
        return [safe_shape(value[0], depth + 1)]
    if isinstance(value, str):
        if len(value) == 0:
            return "str:empty"
        if len(value) < 80:
            return "str:short"
        if len(value) < 1000:
            return "str:medium"
        return "str:long"
    if value is None:
        return "null"
    return type(value).__name__


def safe_key(k: Any) -> str:
    s = str(k)
    # Some transcript metadata uses file paths as object keys (for example
    # snapshot backup maps). Keep the structural fact without copying local paths
    # into generated reports.
    if "/" in s or "\\" in s or len(s) > 80:
        return "<dynamic-key>"
    return s


def walk(obj: Any, prefix: str = ""):
    if isinstance(obj, dict):
        for k, v in obj.items():
            sk = safe_key(k)
            key = f"{prefix}.{sk}" if prefix else sk
            yield key, v
            yield from walk(v, key)
    elif isinstance(obj, list):
        for item in obj[:200]:
            yield from walk(item, prefix + "[]" if prefix else "[]")


def counter_dict(c: Counter, limit: int = 100) -> dict[str, int]:
    return dict(c.most_common(limit))


def inc_if(counter: Counter, name: str, condition: bool, amount: int = 1) -> None:
    if condition:
        counter[name] += amount


def profile_jsonl(adapter: str, path: Path, max_lines: int) -> dict[str, Any]:
    facts: Counter[str] = Counter()
    types: Counter[str] = Counter()
    payload_types: Counter[str] = Counter()
    roles: Counter[str] = Counter()
    block_types: Counter[str] = Counter()
    tool_names: Counter[str] = Counter()
    tool_result_shapes: Counter[str] = Counter()
    interesting_keys: Counter[str] = Counter()
    shape_examples: dict[str, Any] = {}
    invalid = 0
    lines = 0

    with path.open("r", encoding="utf-8", errors="replace") as f:
        for line_no, line in enumerate(f):
            if line_no >= max_lines:
                break
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except Exception:
                invalid += 1
                continue
            lines += 1
            typ = str(obj.get("type") or obj.get("record_type") or obj.get("event") or obj.get("kind") or obj.get("message", {}).get("type") or "<missing>")
            types[typ] += 1
            if isinstance(obj.get("payload"), dict):
                payload_types[str(obj["payload"].get("type") or "<missing>")] += 1

            # Generic key coverage.
            for key, value in walk(obj):
                lk = key.lower()
                if any(h.lower() in lk for h in KEY_HINTS):
                    interesting_keys[key] += 1
                    if key not in shape_examples and not any(k in lk.split(".")[-1] for k in CONTENT_KEYS):
                        shape_examples[key] = safe_shape(value)
                inc_if(facts, "keys.thinking_or_reasoning", "thinking" in lk or "reason" in lk)
                inc_if(facts, "keys.usage_or_tokens", "usage" in lk or "token" in lk)
                inc_if(facts, "keys.attachments_or_images", "attachment" in lk or "image" in lk)
                if key.lower().endswith("attachments") and isinstance(value, list) and len(value) > 0:
                    facts["attachments.nonempty"] += 1
                inc_if(facts, "keys.parent_or_lineage", "parent" in lk or "fork" in lk or "thread" in lk)

            # Common message roles and content blocks.
            msg = obj.get("message") if isinstance(obj.get("message"), dict) else obj
            if isinstance(msg, dict):
                role = msg.get("role")
                if isinstance(role, str):
                    roles[role] += 1
                content = msg.get("content")
                if isinstance(content, list):
                    for block in content:
                        if isinstance(block, dict):
                            bt = str(block.get("type") or "<missing>")
                            block_types[bt] += 1
                            if bt == "image":
                                facts["content_block.image"] += 1
                            if bt in {"thinking", "reasoning", "redacted_thinking"}:
                                facts[f"content_block.{bt}"] += 1
                                if string_value := block.get("thinking") or block.get("text") or block.get("content"):
                                    if isinstance(string_value, str) and string_value:
                                        facts[f"content_block.{bt}.nonempty"] += 1
                                if block.get("signature"):
                                    facts[f"content_block.{bt}.signature"] += 1
                            if bt in {"tool_use", "function_call"}:
                                name = block.get("name") or block.get("tool_name")
                                if isinstance(name, str):
                                    tool_names[name] += 1
                            if bt in {"tool_result", "function_result"}:
                                facts["content_block.tool_result"] += 1

            # Claude Code toolUseResult shape.
            if "toolUseResult" in obj:
                tur = obj.get("toolUseResult")
                if isinstance(tur, dict):
                    keys = tuple(sorted(str(k) for k in tur.keys()))
                    tool_result_shapes["object:" + ",".join(keys[:20])] += 1
                    inc_if(facts, "tool_result.has_stdout", "stdout" in tur)
                    inc_if(facts, "tool_result.has_stderr", "stderr" in tur)
                    inc_if(facts, "tool_result.has_exit_code", "exitCode" in tur or "exit_code" in tur)
                    inc_if(facts, "tool_result.has_duration", any(k in tur for k in ("durationMs", "duration_ms", "totalDurationMs", "total_duration_ms")))
                    inc_if(facts, "tool_result.has_interrupted", "interrupted" in tur)
                elif isinstance(tur, str):
                    tool_result_shapes["string"] += 1
                    inc_if(facts, "tool_result.string_mentions_exit_code", "exit code" in tur.lower())
                else:
                    tool_result_shapes[type(tur).__name__] += 1

            # Codex exec/output shapes.
            if typ in {"exec_command_begin", "exec_command_end", "function_call", "function_call_output"}:
                facts[f"codex.{typ}"] += 1
            payload = obj.get("payload") if isinstance(obj.get("payload"), dict) else {}
            if isinstance(payload, dict):
                ptype = payload.get("type")
                if isinstance(ptype, str):
                    facts[f"codex.payload.{ptype}"] += 1
                if ptype == "token_count":
                    facts["codex.token_count_events"] += 1
                if ptype in {"agent_reasoning", "reasoning"}:
                    facts["codex.reasoning_events"] += 1

            # Lifecycle/source metadata.
            for k in ("cwd", "gitBranch", "version", "sessionId", "parentUuid", "isSidechain", "userType", "uuid"):
                inc_if(facts, f"top.{k}", k in obj)
            if isinstance(obj.get("git"), dict):
                facts["top.git"] += 1
            if isinstance(payload.get("git"), dict):
                facts["payload.git"] += 1
            if isinstance(payload.get("rate_limits"), dict) or "rate_limits" in obj:
                facts["rate_limits"] += 1

    return {
        "adapter": adapter,
        "source_path_hash": "sha256:" + __import__("hashlib").sha256(str(path).encode()).hexdigest()[:16],
        "file_name": path.name,
        "lines_profiled": lines,
        "invalid_lines": invalid,
        "record_types": counter_dict(types, 80),
        "payload_types": counter_dict(payload_types, 80),
        "roles": counter_dict(roles, 20),
        "content_block_types": counter_dict(block_types, 80),
        "tool_names": counter_dict(tool_names, 80),
        "tool_use_result_shapes": counter_dict(tool_result_shapes, 80),
        "facts": counter_dict(facts, 200),
        "interesting_keys": counter_dict(interesting_keys, 150),
        "shape_examples": shape_examples,
    }


def merge_profiles(profiles: list[dict[str, Any]]) -> dict[str, Any]:
    merged: dict[str, Any] = {"files": len(profiles)}
    for field in ["record_types", "payload_types", "roles", "content_block_types", "tool_names", "tool_use_result_shapes", "facts", "interesting_keys"]:
        c: Counter[str] = Counter()
        for p in profiles:
            c.update(p.get(field, {}))
        merged[field] = counter_dict(c, 200)
    merged["lines_profiled"] = sum(int(p.get("lines_profiled", 0)) for p in profiles)
    merged["invalid_lines"] = sum(int(p.get("invalid_lines", 0)) for p in profiles)
    return merged


def profile_archives() -> dict[str, Any]:
    by_adapter: dict[str, Counter[str]] = defaultdict(Counter)
    event_kinds: dict[str, Counter[str]] = defaultdict(Counter)
    attachment_kinds: dict[str, Counter[str]] = defaultdict(Counter)
    framework_keys: dict[str, Counter[str]] = defaultdict(Counter)

    for path in CORPUS.rglob("*.minitrace.json"):
        try:
            obj = json.loads(path.read_text())
        except Exception:
            continue
        adapter = (((obj.get("environment") or {}).get("agent_framework")) or path.parts[-4] if len(path.parts) > 3 else "unknown")
        c = by_adapter[str(adapter)]
        c["sessions"] += 1
        c["turns"] += len(obj.get("turns") or [])
        c["tools"] += len(obj.get("tool_calls") or [])
        c["events"] += len(obj.get("events") or [])
        c["attachments"] += len(obj.get("attachments") or [])
        cfg = (obj.get("operational_context") or {}).get("framework_config")
        if isinstance(cfg, dict):
            for k in cfg:
                framework_keys[str(adapter)]["session.framework_config." + str(k)] += 1
        if (obj.get("operational_context") or {}).get("git_branch"):
            c["session.git_branch"] += 1
        if (obj.get("operational_context") or {}).get("working_directory"):
            c["session.cwd"] += 1
        for t in obj.get("turns") or []:
            if t.get("thinking"):
                c["turn.thinking"] += 1
            if t.get("usage"):
                c["turn.usage"] += 1
            if t.get("model"):
                c["turn.model"] += 1
            fm = t.get("framework_metadata")
            if isinstance(fm, dict):
                if fm.get("thinking_signature_present"):
                    c["turn.signed_thinking"] += 1
                for k in fm:
                    framework_keys[str(adapter)]["turn.framework_metadata." + str(k)] += 1
        for tc in obj.get("tool_calls") or []:
            out = tc.get("output") or {}
            if out.get("duration_ms") is not None:
                c["tool.duration"] += 1
            if out.get("exit_code") is not None:
                c["tool.exit_code"] += 1
            if out.get("error"):
                c["tool.error"] += 1
            if out.get("result"):
                c["tool.result"] += 1
            if out.get("truncated"):
                c["tool.truncated"] += 1
            if tc.get("spawned_agent"):
                c["tool.spawned_agent"] += 1
            fm = tc.get("framework_metadata")
            if isinstance(fm, dict):
                for k in fm:
                    framework_keys[str(adapter)]["tool.framework_metadata." + str(k)] += 1
        for e in obj.get("events") or []:
            kind = str(e.get("kind") or "<missing>")
            event_kinds[str(adapter)][kind] += 1
        for a in obj.get("attachments") or []:
            kind = str(a.get("kind") or "<missing>")
            attachment_kinds[str(adapter)][kind] += 1

    return {
        "by_adapter": {k: counter_dict(v, 200) for k, v in by_adapter.items()},
        "event_kinds": {k: counter_dict(v, 200) for k, v in event_kinds.items()},
        "attachment_kinds": {k: counter_dict(v, 200) for k, v in attachment_kinds.items()},
        "framework_keys": {k: counter_dict(v, 200) for k, v in framework_keys.items()},
    }


def classify(source: dict[str, Any], archive: dict[str, Any]) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    arch = archive.get("by_adapter", {})
    for adapter, src in sorted(source.items()):
        a = arch.get(adapter, {})
        facts = src.get("facts", {})
        blocks = src.get("content_block_types", {})
        recs = src.get("record_types", {})
        payloads = src.get("payload_types", {})

        def add(fact: str, src_count: int, out_count: int, severity: str, note: str):
            findings.append({"adapter": adapter, "fact": fact, "source_count": src_count, "archive_count": out_count, "severity": severity, "note": note})

        thinking_blocks = sum(v for k, v in blocks.items() if k in {"thinking", "reasoning", "redacted_thinking"})
        nonempty_thinking = sum(v for k, v in facts.items() if k.startswith("content_block.") and k.endswith(".nonempty"))
        reasoning_events = facts.get("codex.reasoning_events", 0) + recs.get("reasoning", 0) + recs.get("agent_reasoning", 0) + payloads.get("agent_reasoning", 0)
        thinking_src = nonempty_thinking + reasoning_events
        thinking_out = a.get("turn.thinking", 0)
        if thinking_src and not thinking_out:
            add("thinking/reasoning", thinking_src, thinking_out, "high", "source has non-empty reasoning/thinking structures but converted turns have no thinking")
        elif thinking_src and thinking_out < thinking_src:
            add("thinking/reasoning", thinking_src, thinking_out, "medium", "source reasoning count exceeds archive thinking count; inspect mapping granularity")
        elif thinking_blocks and not thinking_src:
            signed_out = a.get("turn.signed_thinking", 0)
            if signed_out:
                add("thinking/reasoning", thinking_blocks, signed_out, "info", "source thinking blocks are signature/encrypted-only and archive preserves signed-thinking presence in turn metadata")
            else:
                add("thinking/reasoning", thinking_blocks, thinking_out, "info", "source has thinking blocks but no cleartext thinking payload; blocks appear signature/encrypted-only")
        elif not thinking_src and thinking_out == 0:
            add("thinking/reasoning", 0, 0, "info", "not observed in profiled source sample")

        usage_src = facts.get("keys.usage_or_tokens", 0) + facts.get("codex.token_count_events", 0)
        usage_out = a.get("turn.usage", 0)
        if usage_src and not usage_out:
            add("usage/tokens", usage_src, usage_out, "high", "source has usage/token structures but archive has no turn usage")
        elif usage_src and usage_out:
            add("usage/tokens", usage_src, usage_out, "info", "usage exists in both source and archive; compare exact token fields if needed")

        attach_src = facts.get("attachments.nonempty", 0) + facts.get("content_block.image", 0)
        attach_out = a.get("attachments", 0)
        if attach_src and not attach_out:
            add("attachments/images", attach_src, attach_out, "medium", "source has attachment/image signals but archive has no attachments")

        lineage_src = facts.get("keys.parent_or_lineage", 0) + facts.get("top.parentUuid", 0) + facts.get("top.isSidechain", 0)
        lineage_out = a.get("tool.spawned_agent", 0)
        if lineage_src and not lineage_out and adapter in {"claude-code", "codex"}:
            add("lineage/subagents", lineage_src, lineage_out, "medium", "source has lineage/sidechain/parent signals; archive has no spawned_agent links in sampled output")

        if adapter == "claude-code":
            tur = sum(src.get("tool_use_result_shapes", {}).values())
            exits = facts.get("tool_result.has_exit_code", 0) + facts.get("tool_result.string_mentions_exit_code", 0)
            out_exit = a.get("tool.exit_code", 0)
            if tur and out_exit < exits:
                add("tool exit code", exits, out_exit, "medium", "some source exit-code signals may not be reflected as output.exit_code")
            elif tur and not exits:
                add("tool exit code", 0, out_exit, "info", "many toolUseResult objects do not expose native exit codes; null exit_code can be correct")

        if adapter == "codex" and a.get("sessions", 0) < src.get("files", 0):
            add("convertibility", src.get("files", 0), a.get("sessions", 0), "high", "some sampled Codex files did not convert, likely old unknown-jsonl shapes")
    return findings


def write_markdown(source: dict[str, Any], archive: dict[str, Any], findings: list[dict[str, Any]], path: Path) -> None:
    lines = [
        "---",
        "Title: Generated source-vs-archive coverage profile",
        "Ticket: GMT-012-adapter-fidelity-real-transcript-audit",
        "Status: active",
        "Topics:",
        "    - tooling",
        "    - cli",
        "    - diagnostics",
        "DocType: reference",
        "Intent: short-term",
        "Owners: []",
        "RelatedFiles: []",
        "Summary: Generated structural coverage profile comparing sampled native transcript facts to converted minitrace archive facts.",
        "LastUpdated: 2026-07-06T19:00:00-04:00",
        "WhatFor: Evidence artifact for GMT-012 adapter coverage investigation.",
        "WhenToUse: Use with the coverage investigation guide and missing functionality report.",
        "---",
        "",
        "# Source-vs-archive coverage profile",
        "",
        "This report is generated by `scripts/04-profile-source-vs-archive-coverage.py`. It records structural counts only; it does not copy raw transcript messages, tool output, prompts, or private paths.",
        "",
        "## Aggregate source facts",
        "",
    ]
    for adapter, src in sorted(source.items()):
        lines += [f"### {adapter}", "", f"- Files profiled: {src.get('files', 0)}", f"- Lines profiled: {src.get('lines_profiled', 0)}", ""]
        for label, field in [("Record types", "record_types"), ("Payload types", "payload_types"), ("Content block types", "content_block_types"), ("ToolUseResult shapes", "tool_use_result_shapes")]:
            vals = src.get(field, {})
            if vals:
                lines.append(f"**{label}:**")
                for k, v in list(vals.items())[:20]:
                    lines.append(f"- `{k}`: {v}")
                lines.append("")
    lines += ["## Archive facts", ""]
    for adapter, vals in sorted(archive.get("by_adapter", {}).items()):
        lines += [f"### {adapter}", ""]
        for k, v in vals.items():
            lines.append(f"- `{k}`: {v}")
        ev = archive.get("event_kinds", {}).get(adapter, {})
        if ev:
            lines.append("- Event kinds: " + ", ".join(f"`{k}`={v}" for k, v in list(ev.items())[:15]))
        att = archive.get("attachment_kinds", {}).get(adapter, {})
        if att:
            lines.append("- Attachment kinds: " + ", ".join(f"`{k}`={v}" for k, v in list(att.items())[:15]))
        lines.append("")
    lines += ["## Classified coverage findings", ""]
    for f in findings:
        lines.append(f"- **{f['severity']}** `{f['adapter']}` / {f['fact']}: source={f['source_count']}, archive={f['archive_count']} — {f['note']}")
    lines.append("")
    path.write_text("\n".join(lines), encoding="utf-8")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--max-lines", type=int, default=20000)
    ap.add_argument("--adapters", default=",".join(JSONL_ADAPTERS))
    args = ap.parse_args()

    OUT.mkdir(parents=True, exist_ok=True)
    LOGS.mkdir(parents=True, exist_ok=True)

    source_by_adapter: dict[str, Any] = {}
    per_file: list[dict[str, Any]] = []
    for adapter in [a.strip() for a in args.adapters.split(",") if a.strip()]:
        sample = INV / f"{adapter}-sample-list.txt"
        if not sample.exists():
            continue
        profiles = []
        for raw in sample.read_text().splitlines():
            if not raw.strip():
                continue
            p = Path(raw)
            if not p.exists() or not p.is_file():
                continue
            prof = profile_jsonl(adapter, p, args.max_lines)
            profiles.append(prof)
            per_file.append(prof)
        source_by_adapter[adapter] = merge_profiles(profiles)

    archive = profile_archives()
    findings = classify(source_by_adapter, archive)

    (OUT / "source-aggregate.json").write_text(json.dumps(source_by_adapter, indent=2, sort_keys=True), encoding="utf-8")
    (OUT / "source-per-file.json").write_text(json.dumps(per_file, indent=2, sort_keys=True), encoding="utf-8")
    (OUT / "archive-aggregate.json").write_text(json.dumps(archive, indent=2, sort_keys=True), encoding="utf-8")
    (OUT / "coverage-findings.json").write_text(json.dumps(findings, indent=2, sort_keys=True), encoding="utf-8")
    write_markdown(source_by_adapter, archive, findings, OUT / "01-coverage-profile.md")
    print(f"Wrote coverage profile to {OUT}")
    for f in findings:
        print(f"{f['severity']:>6} {f['adapter']:12} {f['fact']:24} source={f['source_count']} archive={f['archive_count']} {f['note']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
