#!/usr/bin/env python3
"""Report how many turns have thinking, model, and usage fields (diagnostic for Issue 4)."""
import json, sys

with open(sys.argv[1]) as f:
    data = json.load(f)

turns = data["turns"]
print(f"Total turns: {len(turns)}")
print(f"  with thinking: {sum(1 for t in turns if t.get('thinking'))}")
print(f"  with model:    {sum(1 for t in turns if t.get('model'))}")
print(f"  with usage:    {sum(1 for t in turns if t.get('usage'))}")
print()

# Show models used
models = {}
for t in turns:
    m = t.get("model", "")
    if m:
        models[m] = models.get(m, 0) + 1
print("Models used:")
for m, count in sorted(models.items(), key=lambda x: -x[1]):
    print(f"  {m}: {count} turns")

# Show first thinking example
print()
for t in turns:
    if t.get("thinking"):
        print(f"First thinking example (turn #{t['index']}):")
        print(f"  model: {t.get('model')}")
        print(f"  usage: {json.dumps(t.get('usage'))}")
        print(f"  thinking: {t['thinking'][:200]}...")
        break

# Usage: python3 06-check-thinking-model-usage.py session.minitrace.json
