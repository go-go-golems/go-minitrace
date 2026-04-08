#!/bin/bash
# Raw curl test against z.ai Coding Plan API to see full reasoning_content.
# Compares raw API output vs what pi captures.
# Usage: ./34-curl-zai-raw-thinking.sh ["prompt"]
set -euo pipefail

PROMPT="${1:-What is 17 * 23? Think step by step.}"

ZAI_TOKEN=$(python3 -c "
import json
with open('$HOME/.pi/agent/auth.json') as f:
    data = json.load(f)
print(data['zai']['access'])
")

echo "=== Raw z.ai API (curl) ==="
echo "Prompt: $PROMPT"
echo ""

curl -s "https://api.z.ai/api/coding/paas/v4/chat/completions" \
  -H "Authorization: Bearer $ZAI_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$(python3 -c "
import json
print(json.dumps({
    'model': 'glm-5.1',
    'messages': [{'role': 'user', 'content': '''$PROMPT'''}],
    'stream': True,
    'enable_thinking': True
}))
")" 2>&1 | python3 -c "
import json, sys

thinking_chunks = []
text_chunks = []
total_lines = 0

for line in sys.stdin:
    line = line.strip()
    if not line or not line.startswith('data: '):
        if line.startswith('{'):
            try:
                obj = json.loads(line)
                if 'error' in obj:
                    print(f'ERROR: {obj[\"error\"]}')
            except: pass
        continue
    data = line[6:]
    if data == '[DONE]': break
    try: obj = json.loads(data)
    except: continue
    total_lines += 1
    choices = obj.get('choices', [])
    if not choices: continue
    delta = choices[0].get('delta', {})
    for field in ['reasoning_content', 'reasoning', 'reasoning_text']:
        val = delta.get(field)
        if val: thinking_chunks.append(val)
    content = delta.get('content')
    if content: text_chunks.append(content)

full_thinking = ''.join(thinking_chunks)
full_text = ''.join(text_chunks)
print(f'SSE lines:       {total_lines}')
print(f'Thinking chunks: {len(thinking_chunks)}')
print(f'Thinking text:   {len(full_thinking)} chars')
print(f'Output text:     {len(full_text)} chars')
print()
if full_thinking:
    print('--- Raw reasoning_content (full) ---')
    print(full_thinking)
    print()
print('--- Output text (first 300 chars) ---')
print(full_text[:300])
"

echo ""
echo "=== Through pi (pi --mode json) ==="
SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"
"$SCRIPTS_DIR/28-test-provider-thinking.sh" zai glm-5.1 "$PROMPT"
