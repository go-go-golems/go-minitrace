#!/usr/bin/env python3
"""Find empty thinking blocks in Pi JSONL and check if they correlate with model/provider.
Usage: python3 25-analyze-empty-thinking.py [pi-session.jsonl]
"""
import json, sys

path = sys.argv[1] if len(sys.argv) > 1 else "../2026-04-06T22-07-00-864Z_f6498c9d-3c41-4850-8f9c-667eca2ee271.jsonl"

current_model = None
current_provider = None
results = []

with open(path) as f:
    for line in f:
        obj = json.loads(line)
        if obj.get("type") == "model_change":
            current_provider = obj.get("provider", current_provider)
            current_model = obj.get("modelId", current_model)
        if obj.get("type") != "message":
            continue
        msg = obj.get("message", {})
        role = msg.get("role", "")
        if role != "assistant":
            continue
        content = msg.get("content", [])
        if not isinstance(content, list):
            continue
        for block in content:
            if isinstance(block, dict) and block.get("type") == "thinking":
                text = block.get("thinking", "")
                results.append((current_model, current_provider, True, len(text)))
                break
        else:
            results.append((current_model, current_provider, False, 0))

by_model = {}
for model, provider, has_thinking, tlen in results:
    if model not in by_model:
        by_model[model] = {"provider": provider, "total": 0, "with_thinking": 0, "nonempty": 0, "empty": 0}
    by_model[model]["total"] += 1
    if has_thinking:
        by_model[model]["with_thinking"] += 1
        if tlen > 0:
            by_model[model]["nonempty"] += 1
        else:
            by_model[model]["empty"] += 1

print(f"{'Model':20s} {'Provider':20s} {'Total':>6s} {'Think':>6s} {'NonEmpty':>8s} {'Empty':>6s}")
for model in sorted(by_model):
    m = by_model[model]
    print(f"{model:20s} {m['provider']:20s} {m['total']:6d} {m['with_thinking']:6d} {m['nonempty']:8d} {m['empty']:6d}")
