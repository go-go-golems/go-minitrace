#!/bin/bash
# Check if the /api/sessions/:id endpoint returns thinking/model/usage on turns
# Run from the session output directory with a minitrace server on port 9877
set -euo pipefail

SESSION=f6498c9d-3c41-4850-8f9c-667eca2ee271
PORT=${1:-9877}

curl -s http://localhost:$PORT/api/sessions/$SESSION | \
  python3 -c "
import json, sys
data = json.load(sys.stdin)
for block in data.get('blocks', []):
    for turn in block.get('turns', []):
        if turn.get('thinking') or turn.get('model'):
            print('Raw turn keys:', sorted(turn.keys()))
            print('thinking:', repr(turn.get('thinking'))[:80])
            print('model:', repr(turn.get('model')))
            print('usage:', turn.get('usage'))
            sys.exit(0)
print('NO THINKING/MODEL FOUND IN API RESPONSE')
"
