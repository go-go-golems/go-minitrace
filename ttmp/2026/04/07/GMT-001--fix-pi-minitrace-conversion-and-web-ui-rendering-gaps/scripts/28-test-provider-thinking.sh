#!/bin/bash
# Test a provider's thinking output with pi --mode json
# Usage: ./28-test-provider-thinking.sh [provider] [model] ["prompt"]
# Example: ./28-test-provider-thinking.sh zai glm-5.1 "What is 17 * 23?"
set -euo pipefail

PROVIDER="${1:-zai}"
MODEL="${2:-glm-5.1}"
PROMPT="${3:-What is 17 times 23? Think step by step.}"

echo "=== Provider: $PROVIDER / Model: $MODEL ==="
echo "=== Prompt: $PROMPT ==="
echo ""

pi --mode json --provider "$PROVIDER" --model "$MODEL" --thinking high -p "$PROMPT" 2>/dev/null | python3 -c "
import json, sys

provider = '$PROVIDER'
thinking_parts = []
thinking_events = 0
final_thinking = ''
final_text = ''

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
            thinking_events += 1
            # delta content can be string or dict
            partial = evt.get('partial', '')
            content = evt.get('content', '')
            delta = partial if isinstance(partial, str) else str(partial)
            if not delta and isinstance(content, str):
                delta = content
            thinking_parts.append(delta)

    elif t == 'message_end':
        msg = obj.get('message', {})
        content = msg.get('content', [])
        if isinstance(content, list):
            for block in content:
                if isinstance(block, dict):
                    bt = block.get('type', '')
                    if bt == 'thinking':
                        final_thinking = block.get('thinking', '')
                        has_sig = 'thinkingSignature' in block
                        redacted = block.get('redacted', False)
                        print(f'Thinking block:')
                        print(f'  len:           {len(final_thinking)}')
                        print(f'  has_signature: {has_sig}')
                        print(f'  redacted:      {redacted}')
                        if final_thinking:
                            print(f'  text:          {final_thinking[:300]}')
                    elif bt == 'text':
                        final_text = block.get('text', '')
                        print(f'Text block: {final_text[:200]}')

streamed = ''.join(thinking_parts)
print(f'')
print(f'Summary:')
print(f'  Provider:          {provider}')
print(f'  Thinking events:   {thinking_events}')
print(f'  Streamed chars:    {len(streamed)}')
print(f'  Final thinking:    {len(final_thinking)} chars')
if final_thinking:
    print(f'  Stream == Final:   {streamed == final_thinking}')
"
