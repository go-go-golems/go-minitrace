#!/usr/bin/env python3
import json
import os
from collections import defaultdict

CANDIDATES = [
    "exit_code",
    "justification",
    "approval_policy",
    "sandbox_policy",
    "collaboration_mode",
    "truncation_policy",
    "rate_limits",
    "turn_id",
    "phase",
    "memory_citation",
    "source",
    "parsed_cmd",
    "stdout",
    "stderr",
    "caller",
    "entrypoint",
    "stop_reason",
    "stop_sequence",
    "slug",
    "parentUuid",
    "isSidechain",
    "cache_creation",
    "diff",
    "stopReason",
    "errorMessage",
]

PATHS = {
    "pi": os.path.expanduser("~/.pi/agent/sessions/--home-manuel-code-wesen-crib-k3s--/2026-04-16T01-34-34-242Z_2035dd97-cfb1-47ba-a90d-41096ae624d5.jsonl"),
    "codex-2026-04": os.path.expanduser("~/.codex/sessions/2026/04/14/rollout-2026-04-14T20-17-00-019d8e7f-99a5-7243-b19a-845778ff2b5a.jsonl"),
    "codex-2026-01": os.path.expanduser("~/.codex/sessions/2026/01/12/rollout-2026-01-12T15-46-57-019bb3f6-3c71-7013-b585-4f16d9bdceb6.jsonl"),
    "claude": os.path.expanduser("~/.claude/projects/-home-manuel-code-others-pretext/3e25fe06-537b-40cf-9903-2e7bf1b1cd0d.jsonl"),
}


def walk(value, path=""):
    if isinstance(value, dict):
        for key, child in value.items():
            child_path = f"{path}.{key}" if path else key
            yield child_path, child
            yield from walk(child, child_path)
    elif isinstance(value, list):
        for idx, child in enumerate(value):
            child_path = f"{path}[{idx}]"
            yield from walk(child, child_path)


def scan(path):
    found = defaultdict(set)
    snippets = defaultdict(list)
    with open(path) as handle:
        for line in handle:
            line = line.rstrip("\n")
            try:
                obj = json.loads(line)
            except Exception:
                continue
            for p, _ in walk(obj):
                leaf = p.split(".")[-1].split("[")[0]
                if leaf in CANDIDATES:
                    found[leaf].add(p)
                    if len(snippets[leaf]) < 2:
                        snippets[leaf].append(line[:500])
    return found, snippets


for name, path in PATHS.items():
    print(f"=== {name} ===")
    found, snippets = scan(path)
    for candidate in CANDIDATES:
        if candidate not in found:
            continue
        print(f"\n[{candidate}]")
        for p in sorted(found[candidate]):
            print(f"- {p}")
        for snippet in snippets[candidate]:
            print(f"  snippet: {snippet}")
    print()
