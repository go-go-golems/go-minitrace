#!/usr/bin/env python3
"""14-autopilot-blocks.py

Reads the human-blocks JSON output for a session and computes:
  - Total blocks, agent turns, tool calls
  - Top 20 biggest autonomous runs (by tool calls)
  - "Autopilot" blocks where the user just said continue/ok/go ahead
  - Aggregate autopilot stats

Usage:
  GLOB='/tmp/minitrace-output/active/*/*.minitrace.json'
  go-minitrace query duckdb --archive-glob "$GLOB" \
    --sql "$(cat 10-human-blocks.sql | sed 's/SESSION_ID/019d376d-.../')" \
    --output json > /tmp/blocks.json
  python3 14-autopilot-blocks.py /tmp/blocks.json
"""
import json, sys

path = sys.argv[1] if len(sys.argv) > 1 else "/tmp/blocks-376d.json"
data = json.load(open(path))

print(f"Total blocks: {len(data)}")
print(f"Total agent turns: {sum(b['agent_turns'] or 0 for b in data)}")
print(f"Total tool calls: {sum(b['tool_calls'] or 0 for b in data)}")
print()

# Top 20 biggest autonomous runs
biggest = sorted(data, key=lambda b: b["tool_calls"] or 0, reverse=True)[:20]
print("=== BIGGEST AUTONOMOUS RUNS (by tool calls) ===")
for b in biggest:
    prompt = (b["user_prompt"] or "")[:90]
    print(
        f"  blk {b['blk']:3d}  turn {b['turn']:4d}"
        f"  agent_turns={b['agent_turns']:3d}"
        f"  tools={b['tool_calls']:4d}"
        f"  ts={b['user_ts'][:16]}  {prompt}"
    )

print()

# Autopilot blocks
AUTOPILOT = {
    "continue", "ok continue", "ok", "go ahead", "yes",
    "ok, continue", "alright do it", "ok do it", "continue.",
    "ok, go ahead", "alrightgo ahead.", "go ahead.",
}
autopilot = [
    b for b in data
    if b["user_prompt"] and b["user_prompt"].strip().rstrip(".").lower() in AUTOPILOT
       or (b["user_prompt"] and b["user_prompt"].strip().lower() in AUTOPILOT)
]
print(f"=== AUTOPILOT BLOCKS (continue/ok/go ahead) === count={len(autopilot)}")
for b in autopilot:
    print(
        f"  blk {b['blk']:3d}"
        f"  agent_turns={b['agent_turns']:3d}"
        f"  tools={b['tool_calls']:4d}"
        f'  "{b["user_prompt"].strip()[:60]}"'
    )
total_at = sum(b["agent_turns"] or 0 for b in autopilot)
total_tc = sum(b["tool_calls"] or 0 for b in autopilot)
all_at = sum(b["agent_turns"] or 0 for b in data)
all_tc = sum(b["tool_calls"] or 0 for b in data)
print(f"  total agent turns in autopilot: {total_at} ({total_at*100//max(all_at,1)}% of all)")
print(f"  total tool calls in autopilot:  {total_tc} ({total_tc*100//max(all_tc,1)}% of all)")
