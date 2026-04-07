#!/usr/bin/env python3
"""Show one example each of edit and write tool calls with their full argument shapes."""
import json, sys

with open(sys.argv[1]) as f:
    data = json.load(f)

for target in ["edit", "write"]:
    for tc in data["tool_calls"]:
        if tc["tool_name"] == target:
            args = tc["input"].get("arguments", {})
            print(f"=== {target} ===")
            print(f"file_path: {tc['input'].get('file_path')}")
            print(f"arg keys: {list(args.keys())}")
            if target == "edit":
                edits = args.get("edits", [])
                if edits:
                    e = edits[0]
                    print(f"edits[0].oldText ({len(e.get('oldText',''))} chars): {e.get('oldText','')[:80]}...")
                    print(f"edits[0].newText ({len(e.get('newText',''))} chars): {e.get('newText','')[:80]}...")
            elif target == "write":
                content = args.get("content", "")
                print(f"content: {len(content)} chars")
                print(f"content preview: {content[:200]}...")
            print()
            break

# Usage: python3 07-show-edit-write-structure.py session.minitrace.json
