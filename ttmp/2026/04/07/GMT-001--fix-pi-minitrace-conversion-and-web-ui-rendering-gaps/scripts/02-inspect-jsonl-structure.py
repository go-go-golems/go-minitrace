#!/usr/bin/env python3
"""Dump the first 20 lines of a Pi JSONL showing type, role, content structure."""
import json, sys

with open(sys.argv[1]) as f:
    for i, line in enumerate(f):
        if i >= 20:
            break
        obj = json.loads(line)
        typ = obj.get("type", "NO_TYPE")
        role = obj.get("message", {}).get("role", "")
        content = obj.get("message", {}).get("content", "")
        preview = ""
        if isinstance(content, str):
            preview = content[:80]
        elif isinstance(content, list):
            parts = []
            for c in content[:2]:
                if isinstance(c, dict):
                    t = c.get("type", "")
                    if t == "text": parts.append(f"text:{c.get('text','')[:40]}")
                    elif t == "toolCall": parts.append(f"toolCall:{c.get('name','')}")
                    elif t == "toolResult": parts.append(f"toolResult:{c.get('toolCallId','')[:8]}")
                    elif t == "thinking": parts.append(f"thinking:{c.get('thinking','')[:30]}")
                    else: parts.append(t)
            preview = " | ".join(parts)
        print(f"[{i:3d}] type={typ:25s} role={role:15s} {preview}")

# Usage: python3 02-inspect-jsonl-structure.py session.jsonl
