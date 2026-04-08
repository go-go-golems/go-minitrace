#!/usr/bin/env python3
"""Compare last thinking_delta partial vs final message_end thinking.
Usage: pi --mode json --provider zai --model glm-5.1 --thinking high -p "What is 17*23?" 2>/dev/null | python3 31-compare-stream-vs-final.py
"""
import json, sys

last_streamed_thinking = ""
thinking_delta_count = 0
final_thinking = ""
final_has_signature = False

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
        if evt.get('type') == 'thinking_delta':
            thinking_delta_count += 1
            partial = evt.get('partial', '')
            if isinstance(partial, dict):
                content = partial.get('content', [])
                if isinstance(content, list) and len(content) > 0:
                    block = content[0]
                    if isinstance(block, dict):
                        last_streamed_thinking = block.get('thinking', '')

    elif t == 'message_end':
        msg = obj.get('message', {})
        content = msg.get('content', [])
        if isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get('type') == 'thinking':
                    final_thinking = block.get('thinking', '')
                    final_has_signature = 'thinkingSignature' in block

print(f"Thinking deltas:        {thinking_delta_count}")
print(f"Last streamed thinking: {len(last_streamed_thinking)} chars")
print(f"Final thinking:         {len(final_thinking)} chars")
print(f"Final has signature:    {final_has_signature}")
print(f"Stream == Final:        {last_streamed_thinking == final_thinking}")
print()
print(f"--- Last streamed (last 300 chars) ---")
print(last_streamed_thinking[-300:] if len(last_streamed_thinking) > 300 else last_streamed_thinking)
print()
print(f"--- Final thinking (full) ---")
print(final_thinking)
