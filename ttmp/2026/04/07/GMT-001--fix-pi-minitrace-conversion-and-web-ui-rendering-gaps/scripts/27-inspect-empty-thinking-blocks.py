#!/usr/bin/env python3
"""Show the raw structure of empty thinking blocks in Pi JSONL.
Usage: python3 27-inspect-empty-thinking-blocks.py [pi-session.jsonl]
"""
import json, sys

path = sys.argv[1] if len(sys.argv) > 1 else "../2026-04-06T22-07-00-864Z_f6498c9d-3c41-4850-8f9c-667eca2ee271.jsonl"

shown = 0
with open(path) as f:
    for line in f:
        obj = json.loads(line)
        if obj.get("type") != "message":
            continue
        msg = obj.get("message", {})
        role = msg.get("role", "")
        if role != "assistant":
            continue
        content = msg.get("content", [])
        if not isinstance(content, list):
            continue
        for block in content:
            if isinstance(block, dict) and block.get("type") == "thinking":
                text = block.get("thinking", "")
                if not text.strip():
                    print(f"=== Empty thinking block (turn ~{shown}) ===")
                    print(json.dumps(block, indent=2))
                    # Show the next block too for context
                    idx = content.index(block)
                    if idx + 1 < len(content):
                        next_block = content[idx + 1]
                        print(f"Next block: type={next_block.get('type','?')}")
                        if next_block.get("type") == "text":
                            print(f"  text: {next_block.get('text','')[:80]}")
                    print()
                    shown += 1
                    if shown >= 5:
                        sys.exit(0)
