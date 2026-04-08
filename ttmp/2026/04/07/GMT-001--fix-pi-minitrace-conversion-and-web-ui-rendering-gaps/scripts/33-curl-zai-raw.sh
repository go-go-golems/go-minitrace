#!/bin/bash
# Raw curl test against z.ai API to see what reasoning_content looks like
# Usage: ./33-curl-zai-raw.sh
set -euo pipefail

# Get the API key from pi config
ZAI_KEY=$(grep -o '"apiKey":"[^"]*"' ~/.pi/config/settings.json 2>/dev/null | head -1 | cut -d'"' -f4)
if [ -z "$ZAI_KEY" ]; then
    ZAI_KEY=$(printenv ZAI_API_KEY 2>/dev/null)
fi
if [ -z "$ZAI_KEY" ]; then
    echo "No ZAI_API_KEY found. Set it or check ~/.pi/config/settings.json"
    exit 1
fi

curl -s "https://api.z.ai/api/coding/paas/v4/chat/completions" \
  -H "Authorization: Bearer $ZAI_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "glm-5.1",
    "messages": [
      {"role": "user", "content": "What is 17 * 23? Think step by step."}
    ],
    "stream": true,
    "enable_thinking": true
  }' 2>/dev/null | python3 -c "
import json, sys

thinking_chunks = []
text_chunks = []
total_lines = 0

for line in sys.stdin:
    line = line.strip()
    if not line or not line.startswith('data: '):
        continue
    data = line[6:]
    if data == '[DONE]':
        break
    try:
        obj = json.loads(data)
    except:
        continue
    total_lines += 1
    choices = obj.get('choices', [])
    if not choices:
        continue
    delta = choices[0].get('delta', {})
    
    # Check for reasoning fields
    for field in ['reasoning_content', 'reasoning', 'reasoning_text']:
        val = delta.get(field)
        if val:
            thinking_chunks.append(val)
    
    # Check for content
    content = delta.get('content')
    if content:
        text_chunks.append(content)

full_thinking = ''.join(thinking_chunks)
full_text = ''.join(text_chunks)
print(f'SSE lines:          {total_lines}')
print(f'Thinking chunks:    {len(thinking_chunks)}')
print(f'Thinking text:      {len(full_thinking)} chars')
print(f'Output text:        {len(full_text)} chars')
print()
print(f'--- Thinking (first 500 chars) ---')
print(full_thinking[:500])
print()
print(f'--- Text (first 200 chars) ---')
print(full_text[:200])
"
