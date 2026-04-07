#!/usr/bin/env python3
"""Check if the v2 blocks API returns thinking/model/usage on turns.
Usage: 22-with-server.sh python3 23-check-v2-thinking.py [SESSION_ID]
"""
import json, sys, urllib.request

SESSION = sys.argv[1] if len(sys.argv) > 1 else "f6498c9d-3c41-4850-8f9c-667eca2ee271"
PORT = sys.argv[2] if len(sys.argv) > 2 else "9877"

url = f"http://localhost:{PORT}/v2/sessions/{SESSION}/blocks"
req = urllib.request.Request(url, headers={"Accept": "application/json"})
with urllib.request.urlopen(req) as resp:
    data = json.load(resp)

blocks = data.get("blocks", [])
print(f"Blocks: {len(blocks)}")

for b in blocks[:5]:
    for t in b.get("turns", []):
        if t.get("thinking"):
            print(f"✓ THINKING FOUND in block {b['block_num']}, turn {t['idx']}")
            print(f"  model: {t.get('model')}")
            print(f"  usage: {t.get('usage')}")
            print(f"  thinking: {t['thinking'][:120]}...")
            sys.exit(0)

print("✗ NO THINKING found in first 5 blocks")
# Debug: show keys of first assistant turn
for b in blocks[:3]:
    for t in b.get("turns", []):
        if t.get("role") == "assistant":
            print(f"  First assistant turn keys: {sorted(t.keys())}")
            sys.exit(0)
