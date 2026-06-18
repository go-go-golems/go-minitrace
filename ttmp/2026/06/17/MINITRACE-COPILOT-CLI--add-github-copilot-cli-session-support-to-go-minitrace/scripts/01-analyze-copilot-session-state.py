#!/usr/bin/env python3
"""Summarize GitHub Copilot CLI session-state without printing message bodies.

Usage:
  python scripts/01-analyze-copilot-session-state.py ~/.copilot/session-state/<id> > sources/local-copilot-session-analysis.md
"""
from __future__ import annotations

import collections
import json
import sqlite3
import sys
from pathlib import Path

try:
    import yaml  # type: ignore
except Exception:  # pragma: no cover
    yaml = None


def load_yaml(path: Path):
    if not path.exists():
        return None
    text = path.read_text(errors="replace")
    if yaml is not None:
        return yaml.safe_load(text)
    return {"raw_preview": text[:1000]}


def shape(value, depth=0):
    if depth > 3:
        return type(value).__name__
    if isinstance(value, dict):
        return {k: shape(v, depth + 1) for k, v in sorted(value.items())[:30]}
    if isinstance(value, list):
        if not value:
            return []
        return [shape(value[0], depth + 1)]
    return type(value).__name__


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: analyze-copilot-session-state.py SESSION_DIR", file=sys.stderr)
        return 2
    root = Path(sys.argv[1]).expanduser().resolve()
    print(f"# Local Copilot CLI session-state structural analysis\n")
    print(f"- Session directory: `{root}`")
    print(f"- Exists: `{root.exists()}`")
    if not root.exists():
        return 1
    print("\n## Directory entries\n")
    for path in sorted(root.rglob("*")):
        if path.is_file():
            try:
                size = path.stat().st_size
            except OSError:
                size = -1
            rel = path.relative_to(root)
            print(f"- `{rel}` ({size} bytes)")

    workspace = load_yaml(root / "workspace.yaml")
    print("\n## workspace.yaml keys\n")
    if isinstance(workspace, dict):
        for key, value in sorted(workspace.items()):
            if isinstance(value, (str, int, float, bool)) or value is None:
                display = value if key.lower() not in {"prompt", "message", "content"} else "<redacted>"
                print(f"- `{key}`: `{display}`")
            else:
                print(f"- `{key}`: `{type(value).__name__}`")
    else:
        print("- no parseable workspace.yaml")

    events_path = root / "events.jsonl"
    print("\n## events.jsonl structural summary\n")
    type_counts = collections.Counter()
    top_keys = collections.Counter()
    payload_type_counts = collections.Counter()
    examples = {}
    bad_lines = []
    if events_path.exists():
        with events_path.open(errors="replace") as f:
            for lineno, line in enumerate(f, 1):
                line = line.strip()
                if not line:
                    continue
                try:
                    obj = json.loads(line)
                except Exception as e:
                    bad_lines.append((lineno, str(e)))
                    continue
                for key in obj.keys():
                    top_keys[key] += 1
                typ = obj.get("type", "<missing>")
                type_counts[str(typ)] += 1
                payload = obj.get("payload")
                if isinstance(payload, dict):
                    ptyp = payload.get("type")
                    if ptyp is not None:
                        payload_type_counts[str(ptyp)] += 1
                examples.setdefault(str(typ), shape(obj))
        print(f"- JSON records: `{sum(type_counts.values())}`")
        print(f"- Bad JSON lines: `{len(bad_lines)}`")
        if bad_lines[:5]:
            print(f"- First bad lines: `{bad_lines[:5]}`")
        print("\n### Top-level keys\n")
        for key, count in top_keys.most_common():
            print(f"- `{key}`: {count}")
        print("\n### Record `type` counts\n")
        for key, count in type_counts.most_common():
            print(f"- `{key}`: {count}")
        print("\n### Payload `type` counts\n")
        for key, count in payload_type_counts.most_common():
            print(f"- `{key}`: {count}")
        print("\n### Redacted example shapes by record type\n")
        for key, val in examples.items():
            print(f"- `{key}`: `{json.dumps(val, sort_keys=True)}`")
    else:
        print("- events.jsonl missing")

    db_path = root / "session.db"
    print("\n## session.db schema summary\n")
    if db_path.exists():
        conn = sqlite3.connect(str(db_path))
        try:
            tables = [r[0] for r in conn.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")]
            for table in tables:
                print(f"\n### Table `{table}`")
                cols = conn.execute(f"PRAGMA table_info({table})").fetchall()
                for _, name, typ, notnull, default, pk in cols:
                    print(f"- `{name}` {typ} notnull={notnull} pk={pk} default={default}")
                try:
                    count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                    print(f"- row_count: `{count}`")
                except Exception as e:
                    print(f"- row_count_error: `{e}`")
        finally:
            conn.close()
    else:
        print("- session.db missing")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
