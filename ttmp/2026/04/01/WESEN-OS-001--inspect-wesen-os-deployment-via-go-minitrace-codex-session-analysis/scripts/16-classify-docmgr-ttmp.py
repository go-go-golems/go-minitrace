#!/usr/bin/env python3
"""16-classify-docmgr-ttmp.py

Reads the output of 15-docmgr-and-ttmp-ops.sql (as JSON) and classifies:
  - docmgr subcommand breakdown
  - ttmp file op types (read, edit, git, create, list, etc.)
  - Per-session totals

Usage:
  GLOB='/tmp/minitrace-output/active/*/*.minitrace.json'
  go-minitrace query duckdb --archive-glob "$GLOB" \
    --sql-file 15-docmgr-and-ttmp-ops.sql --output json > /tmp/docmgr-calls.json
  python3 16-classify-docmgr-ttmp.py /tmp/docmgr-calls.json
"""
import json, sys
from collections import Counter

path = sys.argv[1] if len(sys.argv) > 1 else "/tmp/docmgr-calls.json"
data = json.load(open(path))
print(f"Total docmgr/ttmp tool calls: {len(data)}")
print()

docmgr_cmds = [d for d in data if "docmgr" in (d["cmd"] or "").lower()]
ttmp_cmds = [
    d for d in data
    if "ttmp" in (d["cmd"] or "").lower() and "docmgr" not in (d["cmd"] or "").lower()
]

print(f"docmgr commands: {len(docmgr_cmds)}")
print(f"ttmp file ops (non-docmgr): {len(ttmp_cmds)}")
print()


def extract_subcmd(cmd):
    if not cmd:
        return "unknown"
    parts = cmd.strip().split()
    for i, p in enumerate(parts):
        if "docmgr" in p:
            return " ".join(parts[i : min(i + 3, len(parts))])
    return cmd[:60]


subcmds = Counter(extract_subcmd(d["cmd"]) for d in docmgr_cmds)
print("=== docmgr subcommand breakdown ===")
for cmd, cnt in subcmds.most_common(20):
    print(f"  {cnt:3d}  {cmd[:80]}")


def classify_ttmp(cmd):
    if not cmd:
        return "unknown"
    c = cmd.lower()
    if any(x in c for x in ["cat ", "head ", "tail ", "less ", "wc "]):
        return "read"
    if any(x in c for x in ["mkdir", "cp ", "mv "]):
        return "create/move"
    if any(x in c for x in ["ls ", "find ", "tree "]):
        return "list"
    if "git" in c:
        return "git"
    if any(x in c for x in ["rm ", "rmdir"]):
        return "delete"
    if "apply_patch" in c or "sed " in c or "tee " in c:
        return "edit"
    if "grep" in c:
        return "search"
    return "other"


print()
ttmp_types = Counter(classify_ttmp(d["cmd"]) for d in ttmp_cmds)
print("=== ttmp file op types ===")
for t, cnt in ttmp_types.most_common():
    print(f"  {cnt:3d}  {t}")

print()
print("=== per session ===")
SESSIONS = [
    "019d174c-fc68-7c00-8f1b-7fcc067c1fd6",
    "019d376d-0103-7dc3-a96d-650c7c2e1cf7",
    "019d4a35-9c8d-7f10-8fef-ef0650432725",
]
for sid in SESSIONS:
    s_data = [d for d in data if d["session_id"] == sid]
    s_docmgr = [d for d in s_data if "docmgr" in (d["cmd"] or "").lower()]
    s_ttmp = [
        d for d in s_data
        if "ttmp" in (d["cmd"] or "").lower() and "docmgr" not in (d["cmd"] or "").lower()
    ]
    print(f"  {sid[:8]}  docmgr={len(s_docmgr):3d}  ttmp_ops={len(s_ttmp):3d}  total={len(s_data):3d}")
