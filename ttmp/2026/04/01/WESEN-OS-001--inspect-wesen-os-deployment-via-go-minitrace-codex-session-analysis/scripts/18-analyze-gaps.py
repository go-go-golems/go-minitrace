#!/usr/bin/env python3
"""18-analyze-gaps.py

Reads the output of 17-user-input-gaps.sql (as JSON) and produces a
per-session summary of idle periods.

Usage:
  GLOB='/tmp/minitrace-output/active/*/*.minitrace.json'
  go-minitrace query duckdb --archive-glob "$GLOB" \
    --sql-file 17-user-input-gaps.sql --output json > /tmp/gaps.json
  python3 18-analyze-gaps.py /tmp/gaps.json
"""
import json, sys

path = sys.argv[1] if len(sys.argv) > 1 else "/tmp/gaps.json"
data = json.load(open(path))

print(f"Gaps > 30 min: {len(data)}")
print()

SESSIONS = [
    ("019d174c", "019d174c-fc68-7c00-8f1b-7fcc067c1fd6"),
    ("019d376d", "019d376d-0103-7dc3-a96d-650c7c2e1cf7"),
    ("019d4a35", "019d4a35-9c8d-7f10-8fef-ef0650432725"),
]

for sid_short, sid in SESSIONS:
    rows = [d for d in data if d["session_id"] == sid]
    if not rows:
        continue
    total_gap_h = sum(r["gap_minutes"] for r in rows) / 60
    print(f"=== {sid_short} ({len(rows)} gaps, total idle={total_gap_h:.1f}h) ===")
    for r in rows:
        content = (r["content"] or "")[:80]
        print(
            f"  turn {r['turn_idx']:4d}"
            f"  gap={r['gap_minutes']:6.0f}min ({r['gap_minutes']/60:.1f}h)"
            f"  ts={r['ts'][:16]}  {content}"
        )
    print()
