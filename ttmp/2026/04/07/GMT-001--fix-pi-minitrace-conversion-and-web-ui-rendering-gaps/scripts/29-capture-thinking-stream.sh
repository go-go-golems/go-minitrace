#!/bin/bash
# Capture raw thinking_delta events to see what the provider actually sends
# vs what ends up in the final thinking block
# Usage: ./29-capture-thinking-stream.sh [provider] [model] ["prompt"]
set -euo pipefail

PROVIDER="${1:-zai}"
MODEL="${2:-glm-5.1}"
PROMPT="${3:-What is 17 times 23? Think step by step.}"

echo "=== Raw thinking stream capture ==="
echo "=== Provider: $PROVIDER / Model: $MODEL ==="
echo ""

pi --mode json --provider "$PROVIDER" --model "$MODEL" --thinking high -p "$PROMPT" 2>/dev/null | python3 -c "
import json, sys

provider = '$PROVIDER'
all_thinking_deltas = []

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
            # Dump the raw event structure
            partial = evt.get('partial', '')
            content = evt.get('content', '')
            output = evt.get('output', '')
            print(f'  [{len(all_thinking_deltas):3d}] partial type={type(partial).__name__} content type={type(content).__name__}')
            if isinstance(partial, str) and partial:
                all_thinking_deltas.append(partial)
            elif isinstance(content, str) and content:
                all_thinking_deltas.append(content)
            elif isinstance(partial, dict):
                print(f'        partial keys: {sorted(partial.keys())}')
                all_thinking_deltas.append(str(partial))

    elif t == 'message_end':
        msg = obj.get('message', {})
        content = msg.get('content', [])
        final_thinking = ''
        if isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get('type') == 'thinking':
                    final_thinking = block.get('thinking', '')

        full_streamed = ''.join(all_thinking_deltas)
        print(f'')
        print(f'Result:')
        print(f'  Deltas captured:     {len(all_thinking_deltas)}')
        print(f'  Full streamed text:  {len(full_streamed)} chars')
        print(f'  Final thinking:      {len(final_thinking)} chars')
        print(f'  Ratio:               {len(final_thinking)/max(len(full_streamed),1)*100:.1f}%')
        print(f'')
        print(f'--- Streamed (first 500 chars) ---')
        print(full_streamed[:500])
        print(f'')
        print(f'--- Final thinking (full) ---')
        print(final_thinking)
"
