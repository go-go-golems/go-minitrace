#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def load_json(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    text = path.read_text().strip()
    if not text:
        return []
    data = json.loads(text)
    if isinstance(data, list):
        return data
    return [data]


def md_escape(value: Any) -> str:
    text = "" if value is None else str(value)
    return text.replace("|", "\\|").replace("\n", " ")


def markdown_table(headers: list[str], rows: list[list[Any]]) -> str:
    if not rows:
        return "_No rows._"
    out = ["| " + " | ".join(headers) + " |", "| " + " | ".join("---" for _ in headers) + " |"]
    for row in rows:
        out.append("| " + " | ".join(md_escape(cell) for cell in row) + " |")
    return "\n".join(out)


def session_rows(items: list[dict[str, Any]]) -> list[list[Any]]:
    rows = []
    for item in items:
        rows.append([
            item.get("started_at", ""),
            item.get("working_directory", ""),
            item.get("framework", ""),
            item.get("model", ""),
            item.get("title", ""),
            item.get("hours", ""),
            item.get("turns", ""),
            item.get("tools", ""),
            item.get("read_ratio", ""),
        ])
    return rows


def summary_rows(items: list[dict[str, Any]]) -> list[list[Any]]:
    rows = []
    for item in items:
        rows.append([
            item.get("working_directory", ""),
            item.get("sessions", ""),
            item.get("hours", ""),
            item.get("tools", ""),
            item.get("turns", ""),
            item.get("avg_read_ratio", ""),
        ])
    return rows


def tool_rows(items: list[dict[str, Any]]) -> list[list[Any]]:
    rows = []
    for item in items:
        rows.append([
            item.get("operation", ""),
            item.get("count", ""),
        ])
    return rows


def followup_rows(items: list[dict[str, Any]]) -> list[list[Any]]:
    rows = []
    for item in items:
        rows.append([
            item.get("working_directory", ""),
            item.get("started_at", ""),
            item.get("title", ""),
            item.get("turns", ""),
            item.get("tools", ""),
            item.get("hours", ""),
            item.get("reason", ""),
        ])
    return rows


def annotation_rows(items: list[dict[str, Any]]) -> list[list[Any]]:
    rows = []
    for item in items:
        rows.append([
            item.get("session_id", ""),
            item.get("scope_type", ""),
            item.get("category", ""),
            item.get("title", ""),
            item.get("created_at", ""),
        ])
    return rows


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--day", required=True)
    parser.add_argument("--pi-sources", type=Path, required=True)
    parser.add_argument("--codex-sources", type=Path, required=True)
    parser.add_argument("--session-inventory", type=Path, required=True)
    parser.add_argument("--workspace-summary", type=Path, required=True)
    parser.add_argument("--tool-breakdown", type=Path, required=True)
    parser.add_argument("--followup-candidates", type=Path, required=True)
    parser.add_argument("--annotation-summary", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    sessions = load_json(args.session_inventory)
    workspaces = load_json(args.workspace_summary)
    tools = load_json(args.tool_breakdown)
    followups = load_json(args.followup_candidates)
    annotations = load_json(args.annotation_summary)

    pi_sources = [line.strip() for line in args.pi_sources.read_text().splitlines() if line.strip()] if args.pi_sources.exists() else []
    codex_sources = [line.strip() for line in args.codex_sources.read_text().splitlines() if line.strip()] if args.codex_sources.exists() else []

    lines: list[str] = []
    lines.append(f"# Nightly transcript review for {args.day}")
    lines.append("")
    lines.append("## Goal")
    lines.append("")
    lines.append("Build a resumable daily writeup from the prior day's Pi and Codex transcripts using go-minitrace discovery, conversion, structured query commands, and optional annotations.")
    lines.append("")
    lines.append("## Scope")
    lines.append("")
    lines.append(f"- Pi sessions discovered: **{len(pi_sources)}**")
    lines.append(f"- Codex sessions discovered: **{len(codex_sources)}**")
    if codex_sources:
        lines.append("- Codex source paths are staged into a temporary `.codex` tree before conversion.")
    else:
        lines.append("- No Codex sessions were present for this day.")
    lines.append("")
    lines.append("### Pi session sources")
    lines.append("")
    if pi_sources:
        lines.extend([f"- `{path}`" for path in pi_sources])
    else:
        lines.append("- _None found._")
    lines.append("")
    lines.append("### Codex session sources")
    lines.append("")
    if codex_sources:
        lines.extend([f"- `{path}`" for path in codex_sources])
    else:
        lines.append("- _None found._")
    lines.append("")
    lines.append("## Workspace summary")
    lines.append("")
    lines.append(markdown_table(["working_directory", "sessions", "hours", "tools", "turns", "avg_read_ratio"], summary_rows(workspaces)))
    lines.append("")
    lines.append("## Session inventory")
    lines.append("")
    lines.append(markdown_table(["started_at", "working_directory", "framework", "model", "title", "hours", "turns", "tools", "read_ratio"], session_rows(sessions)))
    lines.append("")
    lines.append("## Follow-up candidates")
    lines.append("")
    lines.append(markdown_table(["working_directory", "started_at", "title", "turns", "tools", "hours", "reason"], followup_rows(followups)))
    lines.append("")
    lines.append("## Tool-operation breakdown")
    lines.append("")
    lines.append(markdown_table(["operation", "count"], tool_rows(tools)))
    lines.append("")
    lines.append("## Annotation summary")
    lines.append("")
    lines.append(markdown_table(["session_id", "scope_type", "category", "title", "created_at"], annotation_rows(annotations)))
    lines.append("")
    lines.append("## Observations for the next context window")
    lines.append("")

    if workspaces:
        top = workspaces[0]
        lines.append(f"- The heaviest workspace was `{top.get('working_directory', '')}` with {top.get('hours', '')} hours and {top.get('tools', '')} tool calls.")
    if sessions:
        longest = max(sessions, key=lambda item: float(item.get("hours", 0) or 0))
        lines.append(
            f"- The longest session was `{longest.get('title', '')}` in `{longest.get('working_directory', '')}` at {longest.get('hours', '')} hours and {longest.get('tools', '')} tool calls."
        )
    if not codex_sources:
        lines.append("- There were no Codex sessions for the target day, so the report is Pi-only for this pass.")
    if followups:
        lines.append(f"- The first follow-up candidate is `{followups[0].get('title', '')}`; use it as the starting point for the next review window.")
    if not annotations:
        lines.append("- No synced annotations were present yet; once you annotate sessions, rerun the annotation summary before finalizing the writeup.")

    lines.append("")
    lines.append("## How to continue")
    lines.append("")
    lines.append("- Re-run the same command with a narrower `--day` or a different `--framework` filter if you need to drill into a single workspace.")
    lines.append("- Add session-level annotations for anything worth revisiting, then sync and rerun the annotation summary.")
    lines.append("- If the writeup is too large for one window, use the workspace summary first, then inspect the top follow-up candidates.")

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text("\n".join(lines).rstrip() + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
