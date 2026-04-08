#!/usr/bin/env python3
"""Dump the raw structure of one thinking_delta partial from pi --mode json.
Usage: pi --mode json --provider zai --model glm-5.1 --thinking high -p "What is 17*23?" 2>/dev/null | python3 30-dump-one-thinking-delta.py
"""
import json, sys

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        obj = json.loads(line)
    except:
        continue
    t = obj.get('type', '')
    if t == 'message_update':
        evt = obj.get('assistantMessageEvent', {})
        et = evt.get('type', '')
        if et == 'thinking_delta':
            partial = evt.get('partial', '')
            if isinstance(partial, dict):
                # Show the thinking content from the partial message
                content = partial.get('content', [])
                if isinstance(content, list) and len(content) > 0:
                    block = content[0]
                    if isinstance(block, dict):
                        thinking = block.get('thinking', '')
                        print(f"Partial thinking text ({len(thinking)} chars):")
                        print(thinking[:500])
                        print()
                        print(f"Full partial keys: {sorted(partial.keys())}")
                        print(f"Content blocks: {len(content)}")
                        print(f"Block[0] keys: {sorted(block.keys())}")
            elif isinstance(partial, str):
                print(f"Partial is string: {partial[:200]}")
            sys.exit(0)

print("No thinking_delta found")
