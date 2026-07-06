#!/usr/bin/env python3
"""Redaction-safe source-shape inventory for go-minitrace adapter audits.

The script samples local transcript stores, counts event/key/tool shapes, and
writes JSON summaries plus source-list files for follow-up conversion scripts.
It deliberately does not copy raw transcript content into the ticket.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import random
import sqlite3
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any, Iterable

TICKET = Path(__file__).resolve().parents[1]
SOURCES = TICKET / "sources" / "source-shape-inventory"
LOGS = TICKET / "scripts" / "logs"

JSONL_ADAPTERS = {
    "pi": [Path("~/.pi/agent/sessions").expanduser()],
    "codex": [Path("~/.codex/sessions").expanduser()],
    "claude-code": [Path("~/.claude/projects").expanduser()],
    "copilot": [Path("~/.copilot").expanduser(), Path("~/.config/github-copilot").expanduser()],
}

EXPORT_ADAPTERS = {
    "chatgpt": [Path("~/Downloads").expanduser(), Path("~/Downloads/chatgpt").expanduser()],
    "claude-ai": [Path("~/Downloads").expanduser(), Path("~/Downloads/claude").expanduser()],
}

TURNSDB_CANDIDATES = [
    Path("~/code").expanduser(),
    Path("~/workspaces").expanduser(),
    Path("~/.config").expanduser(),
]

TOOL_KEY_HINTS = (
    "tool", "tool_name", "name", "function", "function_name", "command", "cmd",
)

INTERESTING_KEY_HINTS = (
    "type", "role", "timestamp", "created_at", "model", "cwd", "git", "branch",
    "session", "parent", "thread", "agent", "subagent", "usage", "token",
    "tool", "duration", "exit", "stderr", "stdout", "error", "image", "attachment",
    "permission", "mode", "title", "summary", "fork", "rate",
)


def sha256_text(value: str) -> str:
    return "sha256:" + hashlib.sha256(value.encode("utf-8", "replace")).hexdigest()[:16]


def file_hash(path: Path, limit: int = 1024 * 1024) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        remaining = limit
        while remaining > 0:
            chunk = f.read(min(65536, remaining))
            if not chunk:
                break
            h.update(chunk)
            remaining -= len(chunk)
    return "sha256:" + h.hexdigest()[:16]


def flatten_keys(obj: Any, prefix: str = "", depth: int = 0, out: set[str] | None = None) -> set[str]:
    if out is None:
        out = set()
    if depth > 4:
        return out
    if isinstance(obj, dict):
        for k, v in obj.items():
            key = f"{prefix}.{k}" if prefix else str(k)
            out.add(key)
            flatten_keys(v, key, depth + 1, out)
    elif isinstance(obj, list):
        for item in obj[:8]:
            flatten_keys(item, prefix + "[]" if prefix else "[]", depth + 1, out)
    return out


def nested_get(obj: Any, path: str) -> Any:
    cur = obj
    for part in path.split("."):
        if not isinstance(cur, dict):
            return None
        cur = cur.get(part)
    return cur


def find_tool_names(obj: Any) -> list[str]:
    ret: list[str] = []
    if isinstance(obj, dict):
        for k, v in obj.items():
            lk = str(k).lower()
            if any(h == lk or lk.endswith("_" + h) or lk.endswith("." + h) for h in TOOL_KEY_HINTS):
                if isinstance(v, str) and 0 < len(v) <= 80:
                    # Avoid counting ordinary prose command strings as a tool name.
                    if lk in {"tool", "tool_name", "name", "function", "function_name"} or not any(c.isspace() for c in v):
                        ret.append(v)
            ret.extend(find_tool_names(v))
    elif isinstance(obj, list):
        for item in obj[:32]:
            ret.extend(find_tool_names(item))
    return ret


def safe_sample(values: Iterable[str], limit: int = 50) -> list[str]:
    values = sorted(set(v for v in values if v))
    return values[:limit]


def summarize_jsonl(adapter: str, path: Path, max_lines: int) -> dict[str, Any]:
    stat = path.stat()
    event_types: Counter[str] = Counter()
    role_types: Counter[str] = Counter()
    tool_names: Counter[str] = Counter()
    key_counts: Counter[str] = Counter()
    interesting_keys: set[str] = set()
    timestamp_keys: set[str] = set()
    invalid_lines = 0
    sampled_lines = 0
    first_type = ""

    with path.open("r", encoding="utf-8", errors="replace") as f:
        for i, line in enumerate(f):
            if i >= max_lines:
                break
            line = line.strip()
            if not line:
                continue
            sampled_lines += 1
            try:
                obj = json.loads(line)
            except Exception:
                invalid_lines += 1
                continue
            typ = str(obj.get("type") or obj.get("event") or obj.get("kind") or obj.get("message", {}).get("type") or "<missing>")
            if not first_type:
                first_type = typ
            event_types[typ] += 1
            role = nested_get(obj, "message.role") or obj.get("role")
            if isinstance(role, str):
                role_types[role] += 1
            for name in find_tool_names(obj):
                tool_names[name] += 1
            keys = flatten_keys(obj)
            for key in keys:
                key_counts[key] += 1
                lk = key.lower()
                if any(h in lk for h in INTERESTING_KEY_HINTS):
                    interesting_keys.add(key)
                if "time" in lk or lk.endswith(".ts") or lk.endswith(".timestamp"):
                    timestamp_keys.add(key)

    return {
        "adapter": adapter,
        "source_path_hash": sha256_text(str(path)),
        "file_hash_prefix": file_hash(path),
        "size_bytes": stat.st_size,
        "mtime_ns": stat.st_mtime_ns,
        "sampled_lines": sampled_lines,
        "max_lines": max_lines,
        "invalid_lines": invalid_lines,
        "first_type": first_type,
        "event_types": dict(event_types.most_common(50)),
        "roles": dict(role_types.most_common(20)),
        "tool_names": dict(tool_names.most_common(50)),
        "interesting_keys": safe_sample(interesting_keys, 120),
        "timestamp_keys": safe_sample(timestamp_keys, 60),
        "top_keys": dict(key_counts.most_common(80)),
        "signals": {
            "has_errors": any("error" in k.lower() for k in interesting_keys) or any("error" in t.lower() for t in event_types),
            "has_subagents": any("subagent" in k.lower() or "agent" in k.lower() for k in interesting_keys),
            "has_attachments": any("image" in k.lower() or "attachment" in k.lower() for k in interesting_keys),
            "has_usage": any("usage" in k.lower() or "token" in k.lower() for k in interesting_keys),
        },
    }


def discover_jsonl(adapter: str, roots: list[Path]) -> list[Path]:
    paths: list[Path] = []
    for root in roots:
        if not root.exists():
            continue
        if adapter == "copilot":
            candidates = list(root.rglob("*.jsonl")) + list(root.rglob("events.jsonl"))
        else:
            candidates = list(root.rglob("*.jsonl"))
        paths.extend(p for p in candidates if p.is_file())
    # De-dupe while preserving deterministic order.
    return sorted(set(paths), key=lambda p: str(p))


def discover_exports(adapter: str, roots: list[Path]) -> list[Path]:
    paths: list[Path] = []
    needles = ["chatgpt", "conversations"] if adapter == "chatgpt" else ["claude", "data-"]
    for root in roots:
        if not root.exists():
            continue
        for ext in ("*.zip", "*.json"):
            for path in root.glob(ext):
                name = path.name.lower()
                if any(n in name for n in needles):
                    paths.append(path)
    return sorted(set(paths), key=lambda p: str(p))


def discover_turnsdb() -> list[Path]:
    paths: list[Path] = []
    for root in TURNSDB_CANDIDATES:
        if not root.exists():
            continue
        # Keep this bounded; turns.db can be anywhere under project trees.
        for path in root.rglob("turns.db"):
            if path.is_file():
                paths.append(path)
                if len(paths) >= 100:
                    return sorted(set(paths), key=lambda p: str(p))
    return sorted(set(paths), key=lambda p: str(p))


def summarize_file_candidate(adapter: str, path: Path) -> dict[str, Any]:
    stat = path.stat()
    ret: dict[str, Any] = {
        "adapter": adapter,
        "source_path_hash": sha256_text(str(path)),
        "file_hash_prefix": file_hash(path),
        "size_bytes": stat.st_size,
        "mtime_ns": stat.st_mtime_ns,
        "suffix": path.suffix,
        "signals": {},
    }
    if adapter == "turnsdb":
        try:
            conn = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
            tables = [row[0] for row in conn.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").fetchall()]
            ret["tables"] = tables[:50]
            counts = {}
            for table in tables[:20]:
                try:
                    counts[table] = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                except Exception:
                    pass
            ret["table_counts"] = counts
            conn.close()
        except Exception as e:
            ret["error"] = str(e)
    return ret


def pick_samples(paths: list[Path], max_per_adapter: int, seed: int) -> list[Path]:
    if len(paths) <= max_per_adapter:
        return paths
    rng = random.Random(seed)
    # Prefer a mix of newest, oldest, and random middle samples.
    by_mtime = sorted(paths, key=lambda p: p.stat().st_mtime_ns)
    picked = []
    picked.extend(by_mtime[: max_per_adapter // 4])
    picked.extend(by_mtime[-max_per_adapter // 4 :])
    remaining = [p for p in paths if p not in set(picked)]
    rng.shuffle(remaining)
    picked.extend(remaining[: max_per_adapter - len(picked)])
    return sorted(set(picked), key=lambda p: str(p))


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--max-per-adapter", type=int, default=30)
    ap.add_argument("--max-lines", type=int, default=2500)
    ap.add_argument("--seed", type=int, default=12)
    args = ap.parse_args()

    SOURCES.mkdir(parents=True, exist_ok=True)
    LOGS.mkdir(parents=True, exist_ok=True)

    all_summaries: list[dict[str, Any]] = []
    inventory: dict[str, Any] = {"adapters": {}}

    for adapter, roots in JSONL_ADAPTERS.items():
        paths = discover_jsonl(adapter, roots)
        samples = pick_samples(paths, args.max_per_adapter, args.seed)
        inventory["adapters"][adapter] = {"discovered": len(paths), "sampled": len(samples), "roots": [str(r) for r in roots]}
        (SOURCES / f"{adapter}-sample-list.txt").write_text("\n".join(str(p) for p in samples) + ("\n" if samples else ""))
        summaries = [summarize_jsonl(adapter, p, args.max_lines) for p in samples]
        all_summaries.extend(summaries)
        (SOURCES / f"{adapter}-source-shapes.json").write_text(json.dumps(summaries, indent=2, sort_keys=True))

    for adapter, roots in EXPORT_ADAPTERS.items():
        paths = discover_exports(adapter, roots)
        samples = pick_samples(paths, args.max_per_adapter, args.seed)
        inventory["adapters"][adapter] = {"discovered": len(paths), "sampled": len(samples), "roots": [str(r) for r in roots]}
        (SOURCES / f"{adapter}-sample-list.txt").write_text("\n".join(str(p) for p in samples) + ("\n" if samples else ""))
        summaries = [summarize_file_candidate(adapter, p) for p in samples]
        all_summaries.extend(summaries)
        (SOURCES / f"{adapter}-source-shapes.json").write_text(json.dumps(summaries, indent=2, sort_keys=True))

    turns = discover_turnsdb()
    turns_samples = pick_samples(turns, args.max_per_adapter, args.seed)
    inventory["adapters"]["turnsdb"] = {"discovered": len(turns), "sampled": len(turns_samples), "roots": [str(r) for r in TURNSDB_CANDIDATES]}
    (SOURCES / "turnsdb-sample-list.txt").write_text("\n".join(str(p) for p in turns_samples) + ("\n" if turns_samples else ""))
    summaries = [summarize_file_candidate("turnsdb", p) for p in turns_samples]
    all_summaries.extend(summaries)
    (SOURCES / "turnsdb-source-shapes.json").write_text(json.dumps(summaries, indent=2, sort_keys=True))

    inventory["summary_count"] = len(all_summaries)
    (SOURCES / "inventory-summary.json").write_text(json.dumps(inventory, indent=2, sort_keys=True))
    print(json.dumps(inventory, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
