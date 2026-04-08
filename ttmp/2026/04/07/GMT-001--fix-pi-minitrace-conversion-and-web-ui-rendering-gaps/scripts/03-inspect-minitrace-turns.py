#!/usr/bin/env python3
"""Dump first N turns from a minitrace JSON showing role, source, content preview."""
import json, sys

with open(sys.argv[1]) as f:
    data = json.load(f)

turns = data.get("turns", [])
n = int(sys.argv[2]) if len(sys.argv) > 2 else 30
print(f"Total turns: {len(turns)}")
print()
for i, t in enumerate(turns[:n]):
    role = t.get("role", "NO_ROLE")
    source = t.get("source", "")
    thinking = t.get("thinking") is not None
    model = t.get("model", "")
    content = t.get("content", "")
    preview = content[:80].replace("\n", "\\n") if isinstance(content, str) else ""
    print(f"[{i:3d}] role={role:12s} src={str(source):8s} think={thinking!s:5s} model={model:12s} {preview}")

# Usage: python3 03-inspect-minitrace-turns.py session.minitrace.json [N]
