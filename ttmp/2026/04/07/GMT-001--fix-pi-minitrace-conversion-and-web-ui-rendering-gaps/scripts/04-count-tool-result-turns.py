#!/usr/bin/env python3
"""Count how many turns are tool results disguised as user messages (pre-fix diagnostic)."""
import json, sys

with open(sys.argv[1]) as f:
    data = json.load(f)

turns = data["turns"]
user_turns = [t for t in turns if t["role"] == "user"]
assistant_turns = [t for t in turns if t["role"] == "assistant"]
framework_user = [t for t in turns if t["role"] == "user" and t.get("source") == "framework"]
human_user = [t for t in turns if t["role"] == "user" and t.get("source") == "human"]

print(f"Total turns:         {len(turns)}")
print(f"  assistant:         {len(assistant_turns)}")
print(f"  user (all):        {len(user_turns)}")
print(f"    user (human):    {len(human_user)}")
print(f"    user (framework):{len(framework_user)}")
print()
if framework_user:
    print(f"⚠ {len(framework_user)} tool result turns incorrectly appear as 'user' (source=framework)")
    print("  First 5:")
    for t in framework_user[:5]:
        print(f"    #{t['index']}: {t.get('content','')[:60]}...")
else:
    print("✓ No framework-sourced user turns (tool results correctly excluded)")

# Usage: python3 04-count-tool-result-turns.py session.minitrace.json
