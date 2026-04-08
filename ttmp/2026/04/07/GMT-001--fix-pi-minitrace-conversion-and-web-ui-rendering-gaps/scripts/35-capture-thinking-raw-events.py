#!/usr/bin/env python3
"""Capture ALL thinking-related events from pi --mode json to find where thinking gets truncated.
Usage: pi --mode json --provider zai --model glm-5.1 --thinking high -p "What is 17*23?" 2>/dev/null | python3 35-capture-thinking-raw-events.py
"""
import json, sys

thinking_deltas = []
message_end_thinking = None
message_end_len = 0

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        obj = json.loads(line)
    except:
        continue

    t = obj.get('type', '')

    # Capture thinking_delta from message_update events
    if t == 'message_update':
        evt = obj.get('assistantMessageEvent', {})
        if evt.get('type') == 'thinking_delta':
            delta = evt.get('delta', '')
            partial = evt.get('partial', '')
            if isinstance(partial, dict):
                content = partial.get('content', [])
                for block in content:
                    if isinstance(block, dict) and block.get('type') == 'thinking':
                        thinking_deltas.append({
                            'delta_len': len(delta),
                            'accumulated_len': len(block.get('thinking', '')),
                            'delta_preview': delta[:80],
                        })
    
    # Capture the final message_end thinking
    elif t == 'message_end':
        msg = obj.get('message', {})
        content = msg.get('content', [])
        if isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get('type') == 'thinking':
                    message_end_thinking = block.get('thinking', '')
                    message_end_len = len(message_end_thinking)

    # Also capture done event
    elif t == 'done':
        msg = obj.get('message', {})
        content = msg.get('content', [])
        if isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get('type') == 'thinking':
                    print(f"[done] thinking: {len(block.get('thinking', ''))} chars")

print(f"Thinking deltas:         {len(thinking_deltas)}")
print(f"message_end thinking:    {message_end_len} chars")
print()

if thinking_deltas:
    print(f"Delta accumulation:")
    print(f"  first delta:   len={thinking_deltas[0]['delta_len']}, accumulated={thinking_deltas[0]['accumulated_len']}")
    print(f"  last delta:    len={thinking_deltas[-1]['delta_len']}, accumulated={thinking_deltas[-1]['accumulated_len']}")
    max_acc = max(d['accumulated_len'] for d in thinking_deltas)
    print(f"  max accumulated: {max_acc}")
    print()
    print(f"  Ratio: message_end / max_accumulated = {message_end_len}/{max_acc} = {message_end_len/max_acc:.3f}")
    print()

if message_end_thinking:
    print(f"--- message_end thinking (full) ---")
    print(message_end_thinking)
    print()

# Show if the last delta's accumulated text is different from message_end
if thinking_deltas and message_end_thinking:
    print(f"message_end == last accumulated? checking...")
    # We don't have the actual accumulated text from the last delta, just its length
    print(f"  last accumulated len: {thinking_deltas[-1]['accumulated_len']}")
    print(f"  message_end len:     {message_end_len}")
    if thinking_deltas[-1]['accumulated_len'] != message_end_len:
        print(f"  *** MISMATCH! Thinking was truncated from {thinking_deltas[-1]['accumulated_len']} to {message_end_len} chars ***")
    else:
        print(f"  Match - no truncation in pi")
