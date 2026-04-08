#!/usr/bin/env python3
"""Show one example of each tool_name with its argument keys, file_path, command."""
import json, sys

with open(sys.argv[1]) as f:
    data = json.load(f)

seen = {}
for tc in data["tool_calls"]:
    name = tc["tool_name"]
    if name not in seen:
        seen[name] = tc

for name in sorted(seen):
    tc = seen[name]
    args = tc.get("input", {}).get("arguments", {})
    fp = tc.get("input", {}).get("file_path", "")
    cmd = tc.get("input", {}).get("command", "")
    print(f"{name:20s} file_path={str(fp):30s} command={str(cmd):30s} arg_keys={list(args.keys())[:8]}")

# Usage: python3 05-tool-call-argument-shapes.py session.minitrace.json
