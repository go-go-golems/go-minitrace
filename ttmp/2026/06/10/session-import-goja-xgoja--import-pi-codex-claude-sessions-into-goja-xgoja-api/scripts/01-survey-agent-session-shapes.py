#!/usr/bin/env python3
"""Survey recent Pi, Codex, and Claude Code session JSONL files.

The script intentionally prints structural previews rather than full prompts or
message bodies. It is meant for ticket research: identify current JSONL shapes,
content block kinds, metadata keys, tool-call/result forms, subagent files, and
image/blob indicators without copying private transcript text into docs.
"""
from __future__ import annotations

import argparse
import collections
import json
import os
from pathlib import Path
from typing import Any, Iterable

HOME = Path.home()
DEFAULT_ROOTS = {
    "pi": HOME / ".pi" / "agent" / "sessions",
    "codex": HOME / ".codex" / "sessions",
    "claude": HOME / ".claude" / "projects",
}


def iter_jsonl_files(root: Path, max_files: int) -> list[Path]:
    if not root.exists():
        return []
    files = [p for p in root.rglob("*.jsonl") if p.is_file()]
    files.sort(key=lambda p: (p.stat().st_mtime, str(p)))
    return files[-max_files:]


def load_records(path: Path, max_records: int | None = None) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as f:
        for line_number, line in enumerate(f, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                value = json.loads(line)
            except Exception as e:  # noqa: BLE001 - research script should continue
                records.append({"__parse_error__": f"line {line_number}: {e}"})
                continue
            if isinstance(value, dict):
                records.append(value)
            else:
                records.append({"__non_object__": type(value).__name__})
            if max_records is not None and len(records) >= max_records:
                break
    return records


def keys(value: Any) -> tuple[str, ...]:
    return tuple(sorted(value.keys())) if isinstance(value, dict) else ()


def get_path(value: Any, *path: str) -> Any:
    cur = value
    for part in path:
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
    return cur


def list_value(value: Any) -> list[Any]:
    if isinstance(value, list):
        return value
    if value is None:
        return []
    return [value]


def shorten_path(path: Path) -> str:
    s = str(path)
    home = str(HOME)
    if s.startswith(home + os.sep):
        return "~" + s[len(home):]
    return s


def survey_file(framework: str, path: Path, max_records: int | None) -> dict[str, Any]:
    records = load_records(path, max_records=max_records)
    type_counts: collections.Counter[str] = collections.Counter()
    top_keys: collections.Counter[tuple[str, ...]] = collections.Counter()
    message_keys: collections.Counter[tuple[str, ...]] = collections.Counter()
    payload_types: collections.Counter[str] = collections.Counter()
    content_block_types: collections.Counter[str] = collections.Counter()
    tool_names: collections.Counter[str] = collections.Counter()
    tool_result_keys: collections.Counter[tuple[str, ...]] = collections.Counter()
    image_indicators: collections.Counter[str] = collections.Counter()
    has_subagent_path = "/subagents/" in str(path)
    agent_ids: set[str] = set()
    session_ids: set[str] = set()
    models: set[str] = set()

    for record in records:
        t = str(record.get("type", "<missing>"))
        type_counts[t] += 1
        top_keys[keys(record)] += 1
        payload = record.get("payload")
        if isinstance(payload, dict):
            payload_types[str(payload.get("type", "<missing>"))] += 1
            if payload.get("id"):
                session_ids.add(str(payload.get("id")))
            if payload.get("model"):
                models.add(str(payload.get("model")))
        if record.get("id"):
            session_ids.add(str(record.get("id")))
        if record.get("sessionId"):
            session_ids.add(str(record.get("sessionId")))
        if record.get("agentId"):
            agent_ids.add(str(record.get("agentId")))
        if record.get("model"):
            models.add(str(record.get("model")))

        msg = record.get("message")
        if isinstance(msg, dict):
            message_keys[keys(msg)] += 1
            if msg.get("model"):
                models.add(str(msg.get("model")))
            role = str(msg.get("role", ""))
            if role.lower().startswith("tool"):
                tool_result_keys[keys(msg)] += 1
            for block in list_value(msg.get("content")):
                if isinstance(block, dict):
                    bt = str(block.get("type", "<missing>"))
                    content_block_types[bt] += 1
                    name = block.get("name")
                    if name:
                        tool_names[str(name)] += 1
                    if bt in {"image", "image_url", "input_image"} or any(k in block for k in ("source", "data", "media_type", "mimeType")):
                        image_indicators[bt] += 1
        # Codex event payloads carry messages/tool calls in payload rather than message.
        if isinstance(payload, dict):
            et = str(payload.get("type", ""))
            if et:
                payload_types[et] += 0
            if payload.get("model"):
                models.add(str(payload.get("model")))
            if payload.get("name"):
                tool_names[str(payload.get("name"))] += 1
            if payload.get("call_id"):
                tool_result_keys[keys(payload)] += 1
            for item in list_value(payload.get("items")) + list_value(payload.get("content")):
                if isinstance(item, dict):
                    it = str(item.get("type", "<missing>"))
                    content_block_types[it] += 1
                    if item.get("name"):
                        tool_names[str(item.get("name"))] += 1
                    if it in {"image", "image_url", "input_image"} or any(k in item for k in ("source", "data", "media_type", "mimeType")):
                        image_indicators[it] += 1

    def top(counter: collections.Counter[Any], n: int = 8) -> list[Any]:
        return [{"value": list(k) if isinstance(k, tuple) else k, "count": v} for k, v in counter.most_common(n)]

    return {
        "framework": framework,
        "path": shorten_path(path),
        "mtime": path.stat().st_mtime,
        "bytes": path.stat().st_size,
        "records_sampled": len(records),
        "is_subagent_path": has_subagent_path,
        "session_ids": sorted(session_ids)[:5],
        "agent_ids": sorted(agent_ids)[:5],
        "models": sorted(models)[:5],
        "record_types": top(type_counts),
        "payload_types": top(payload_types),
        "content_block_types": top(content_block_types),
        "tool_names": top(tool_names),
        "top_level_key_shapes": top(top_keys, 5),
        "message_key_shapes": top(message_keys, 5),
        "tool_result_key_shapes": top(tool_result_keys, 5),
        "image_blob_indicators": top(image_indicators),
    }


def render_markdown(items: list[dict[str, Any]]) -> str:
    lines = ["# Agent Session Shape Survey", "", "Structural preview only; message text and prompt bodies are intentionally omitted.", ""]
    for item in items:
        lines.append(f"## {item['framework']}: `{item['path']}`")
        lines.append("")
        lines.append(f"- Bytes: {item['bytes']}")
        lines.append(f"- Records sampled: {item['records_sampled']}")
        lines.append(f"- Subagent path: {item['is_subagent_path']}")
        if item["session_ids"]:
            lines.append(f"- Session IDs: `{', '.join(item['session_ids'])}`")
        if item["agent_ids"]:
            lines.append(f"- Agent IDs: `{', '.join(item['agent_ids'])}`")
        if item["models"]:
            lines.append(f"- Models: `{', '.join(item['models'])}`")
        for label, key in [
            ("Record types", "record_types"),
            ("Payload types", "payload_types"),
            ("Content block types", "content_block_types"),
            ("Tool names", "tool_names"),
            ("Top-level key shapes", "top_level_key_shapes"),
            ("Message key shapes", "message_key_shapes"),
            ("Tool-result key shapes", "tool_result_key_shapes"),
            ("Image/blob indicators", "image_blob_indicators"),
        ]:
            lines.append(f"- {label}:")
            values = item[key]
            if not values:
                lines.append("  - none observed")
            for row in values:
                lines.append(f"  - `{row['value']}`: {row['count']}")
        lines.append("")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--framework", choices=["pi", "codex", "claude", "all"], default="all")
    parser.add_argument("--max-files", type=int, default=3)
    parser.add_argument("--max-records", type=int, default=None)
    parser.add_argument("--format", choices=["markdown", "json"], default="markdown")
    args = parser.parse_args()

    frameworks: Iterable[str] = DEFAULT_ROOTS if args.framework == "all" else [args.framework]
    items: list[dict[str, Any]] = []
    for framework in frameworks:
        for path in iter_jsonl_files(DEFAULT_ROOTS[framework], args.max_files):
            items.append(survey_file(framework, path, args.max_records))

    if args.format == "json":
        print(json.dumps(items, indent=2, sort_keys=True))
    else:
        print(render_markdown(items))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
