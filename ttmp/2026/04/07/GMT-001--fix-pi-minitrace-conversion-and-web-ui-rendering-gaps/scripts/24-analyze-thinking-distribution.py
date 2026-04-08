#!/usr/bin/env python3
"""Analyze thinking block distribution across the session.
Usage: python3 24-analyze-thinking-distribution.py [minitrace.json]
"""
import json, sys

path = sys.argv[1] if len(sys.argv) > 1 else "./analysis/active/2026-04/f6498c9d-3c41-4850-8f9c-667eca2ee271.minitrace.json"
with open(path) as f:
    data = json.load(f)

turns = data["turns"]
assistant = [t for t in turns if t["role"] == "assistant"]
with_thinking = [t for t in assistant if t.get("thinking")]

print(f"Total turns:       {len(turns)}")
print(f"Assistant turns:   {len(assistant)}")
print(f"With thinking:     {len(with_thinking)}  ({100*len(with_thinking)//len(assistant)}%)")
print()

# Histogram by decile
deciles = 10
bucket_size = len(assistant) / deciles
buckets = [0] * deciles
for t in with_thinking:
    idx = assistant.index(t)
    b = min(int(idx / bucket_size), deciles - 1)
    buckets[b] += 1

print("Thinking distribution (by session decile):")
for i, count in enumerate(buckets):
    bar = "█" * count
    pct = count * 100 // max(len(with_thinking), 1)
    start = int(i * bucket_size)
    end = int((i + 1) * bucket_size)
    print(f"  {start:3d}-{end:3d}: {count:3d} {bar}")
