#!/bin/bash
# Trace why thinking/model/usage were not reaching the frontend
# Root cause: v2 API uses protobuf, proto Turn message didn't have the fields
#
# Investigation steps:
# 1. Check API response: curl localhost:PORT/api/sessions/ID | python3 -c "..."
# 2. Check which endpoint the frontend uses: grep "v2/sessions" web/src/api/
# 3. Check proto definition: cat proto/.../sessions.proto | grep -A10 "message Turn"
# 4. Check adapter: grep "adaptTurn" web/src/api/sessionProtoAdapters.ts

# Step 1: Verify the v2 blocks endpoint now returns thinking
SESSION=f6498c9d-3c41-4850-8f9c-667eca2ee271
PORT=${1:-9877}

echo "=== Checking v2 blocks API ==="
curl -s http://localhost:$PORT/v2/sessions/$SESSION/blocks | \
  python3 -c "
import json, sys
data = json.load(sys.stdin)
blocks = data.get('blocks', [])
for block in blocks[:3]:
    for turn in block.get('turns', []):
        if turn.get('thinking'):
            print(f'✓ Thinking found in block {block[\"block_num\"]}, turn {turn[\"idx\"]}')
            print(f'  model: {turn.get(\"model\")}')
            print(f'  usage: {turn.get(\"usage\")}')
            print(f'  thinking: {turn[\"thinking\"][:100]}...')
            sys.exit(0)
print('✗ No thinking found in v2 API response')
"
