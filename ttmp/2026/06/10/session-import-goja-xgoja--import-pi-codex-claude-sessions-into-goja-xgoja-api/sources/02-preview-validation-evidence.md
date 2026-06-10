---
Title: Preview Validation Evidence
Ticket: session-import-goja-xgoja
Status: active
Topics:
    - minitrace
    - goja
    - xgoja
    - transcript
DocType: reference
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Phase 4 preview validation over latest local Pi, Codex, Claude parent, and Claude subagent sessions."
LastUpdated: 2026-06-10T18:30:52.478395442-04:00
WhatFor: "Evidence that latest local sessions load through go-minitrace preview session."
WhenToUse: "Use when reviewing Phase 4 validation outcomes for session import support."
---

# Preview Validation Evidence

All previews use `--privacy structural --sample-limit 2 --output json` unless otherwise noted.

## Source Files

- Pi: `/home/manuel/.pi/agent/sessions/--home-manuel-workspaces-2026-06-07-club-meetup-site--/2026-06-10T18-27-34-754Z_019eb2ca-2ce2-7332-ba1f-2dd453caf5af.jsonl`
- Codex: `/home/manuel/.codex/sessions/2026/06/10/rollout-2026-06-10T10-58-46-019eb20b-0253-7882-9999-36979739f196.jsonl`
- Claude parent: `/home/manuel/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a.jsonl`
- Claude subagent: `/home/manuel/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a/subagents/agent-ab9117a7e97626a37.jsonl`

## Pi Preview

```json
[
{
  "adapter": "pi",
  "agent_framework": "pi",
  "diagnostics": [
    {
      "source": "2026-06-10T18-27-34-754Z_019eb2ca-2ce2-7332-ba1f-2dd453caf5af.jsonl",
      "severity": "info",
      "format": "pi-jsonl",
      "message": "converted source into minitrace session",
      "adapter": "pi",
      "recordRows": 361
    }
  ],
  "error": "",
  "format": "pi-jsonl",
  "has_image_signals": true,
  "has_system_prompt": false,
  "has_thinking": true,
  "model": "gpt-5.5",
  "role_counts": {
    "assistant": 169,
    "user": 5
  },
  "sample_tools": [
    {
      "id": "call_fabb9ZN2oTnQNVlPdWYRCrbU|fc_0a8ad42086ee5fdf016a29ad4021d08197b0545b52d59751f0",
      "turnIndex": 1,
      "toolName": "read",
      "operationType": "READ",
      "success": true,
      "hasResult": true,
      "hasError": false,
      "truncated": false,
      "outputContentBytes": 3362
    },
    {
      "id": "call_nOqnHX3ZPVjyUqAGUxiTXKHG|fc_0a8ad42086ee5fdf016a29ad4021e08197a13936ca9df81460",
      "turnIndex": 1,
      "toolName": "read",
      "operationType": "READ",
      "success": true,
      "hasResult": true,
      "hasError": false,
      "truncated": false,
      "outputContentBytes": 1174
    }
  ],
  "sample_turns": [
    {
      "index": 0,
      "role": "user",
      "source": "human",
      "model": "gpt-5.5",
      "hasContent": true,
      "hasThinking": false
    },
    {
      "index": 1,
      "role": "assistant",
      "source": "model",
      "model": "gpt-5.5",
      "hasContent": false,
      "hasThinking": false,
      "toolCalls": [
        "call_fabb9ZN2oTnQNVlPdWYRCrbU|fc_0a8ad42086ee5fdf016a29ad4021d08197b0545b52d59751f0",
        "call_nOqnHX3ZPVjyUqAGUxiTXKHG|fc_0a8ad42086ee5fdf016a29ad4021e08197a13936ca9df81460",
        "call_GMD5kJi0cV6yw4H1uuXEHVST|fc_0a8ad42086ee5fdf016a29ad4021e88197bd17ceb96eb86723"
      ]
    }
  ],
  "session_id": "019eb2ca-2ce2-7332-ba1f-2dd453caf5af",
  "source_path": "/home/manuel/.pi/agent/sessions/--home-manuel-workspaces-2026-06-07-club-meetup-site--/2026-06-10T18-27-34-754Z_019eb2ca-2ce2-7332-ba1f-2dd453caf5af.jsonl",
  "subagent_count": 0,
  "title": "Create a new ticket in go-minitrace/ttmp using `docmgr --root go-minitrace/ttmp ",
  "tool_call_count": 178,
  "tool_counts": {
    "bash": 89,
    "edit": 30,
    "kagi_web_search": 2,
    "read": 47,
    "write": 10
  },
  "turn_count": 174,
  "working_directory": "~/workspaces/2026-06-07/club-meetup-site"
}
]
```

## Codex Preview

```json
[
{
  "adapter": "codex",
  "agent_framework": "codex",
  "diagnostics": [
    {
      "source": "rollout-2026-06-10T10-58-46-019eb20b-0253-7882-9999-36979739f196.jsonl",
      "severity": "info",
      "format": "codex-jsonl",
      "message": "converted source into minitrace session",
      "adapter": "codex",
      "recordRows": 198
    }
  ],
  "error": "",
  "format": "codex-jsonl",
  "has_image_signals": true,
  "has_system_prompt": true,
  "has_thinking": false,
  "model": "gpt-5.4-mini",
  "role_counts": {
    "assistant": 23,
    "user": 4
  },
  "sample_tools": [
    {
      "id": "call_NnoknAkhq9xzWdfZm1VeQcdV",
      "turnIndex": 2,
      "toolName": "exec_command",
      "operationType": "EXECUTE",
      "success": true,
      "hasResult": true,
      "hasError": false,
      "truncated": false,
      "outputContentBytes": 11
    },
    {
      "id": "call_jZJIS89njk4nEHT8Crn7KlEI",
      "turnIndex": 2,
      "toolName": "exec_command",
      "operationType": "READ",
      "success": false,
      "hasResult": true,
      "hasError": true,
      "truncated": false,
      "outputContentBytes": 101
    }
  ],
  "sample_turns": [
    {
      "index": 0,
      "role": "user",
      "source": "human",
      "hasContent": true,
      "hasThinking": false
    },
    {
      "index": 1,
      "role": "assistant",
      "source": "model",
      "model": "gpt-5.4-mini",
      "hasContent": true,
      "hasThinking": false
    }
  ],
  "session_id": "019eb20b-0253-7882-9999-36979739f196",
  "source_path": "/home/manuel/.codex/sessions/2026/06/10/rollout-2026-06-10T10-58-46-019eb20b-0253-7882-9999-36979739f196.jsonl",
  "subagent_count": 0,
  "title": "Create a little demo application just so I have a nice session content with thin",
  "tool_call_count": 36,
  "tool_counts": {
    "apply_patch": 4,
    "exec_command": 18,
    "spawn_agent": 7,
    "view_image": 1,
    "wait_agent": 4,
    "write_stdin": 2
  },
  "turn_count": 27,
  "working_directory": "/tmp/foobar"
}
]
```

## Claude Parent Preview

```json
[
{
  "adapter": "claude-code",
  "agent_framework": "claude-code",
  "diagnostics": [
    {
      "source": "e431a533-9e12-4300-b192-c52da630439a.jsonl",
      "severity": "info",
      "format": "claude-code-jsonl",
      "message": "converted source into minitrace session",
      "adapter": "claude-code",
      "recordRows": 90
    }
  ],
  "error": "",
  "format": "claude-code-jsonl",
  "has_image_signals": false,
  "has_system_prompt": false,
  "has_thinking": false,
  "model": "claude-haiku-4-5-20251001",
  "role_counts": {
    "assistant": 35,
    "system": 3,
    "user": 5
  },
  "sample_tools": [
    {
      "id": "toolu_01GvjeHKnU3rkasNXbQvaahm",
      "turnIndex": 6,
      "toolName": "Read",
      "operationType": "READ",
      "success": false,
      "hasResult": true,
      "hasError": true,
      "truncated": false,
      "outputContentBytes": 147
    },
    {
      "id": "toolu_019RpRTJCShNyvDzYhGgFE97",
      "turnIndex": 9,
      "toolName": "Agent",
      "operationType": "DELEGATE",
      "success": true,
      "hasResult": true,
      "hasError": false,
      "truncated": false,
      "spawnedAgentType": "general-purpose",
      "outputContentBytes": 317
    }
  ],
  "sample_turns": [
    {
      "index": 0,
      "role": "user",
      "source": "human",
      "hasContent": true,
      "hasThinking": false
    },
    {
      "index": 1,
      "role": "user",
      "source": "framework",
      "hasContent": true,
      "hasThinking": false
    }
  ],
  "session_id": "e431a533-9e12-4300-b192-c52da630439a",
  "source_path": "/home/manuel/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a.jsonl",
  "subagent_count": 6,
  "title": "Demonstrate Claude Code capabilities and features",
  "tool_call_count": 17,
  "tool_counts": {
    "Agent": 6,
    "Bash": 6,
    "Read": 1,
    "Write": 4
  },
  "turn_count": 43,
  "working_directory": ""
}
]
```

## Claude Subagent Preview

```json
[
{
  "adapter": "claude-code",
  "agent_framework": "claude-code",
  "diagnostics": [
    {
      "source": "agent-ab9117a7e97626a37.jsonl",
      "severity": "info",
      "format": "claude-code-jsonl",
      "message": "converted source into minitrace session",
      "adapter": "claude-code",
      "recordRows": 4
    }
  ],
  "error": "",
  "format": "claude-code-jsonl",
  "has_image_signals": false,
  "has_system_prompt": false,
  "has_thinking": false,
  "model": "claude-haiku-4-5-20251001",
  "role_counts": {
    "assistant": 1,
    "user": 1
  },
  "sample_tools": null,
  "sample_turns": [
    {
      "index": 0,
      "role": "user",
      "source": "human",
      "hasContent": true,
      "hasThinking": false
    },
    {
      "index": 1,
      "role": "assistant",
      "source": "model",
      "model": "claude-haiku-4-5-20251001",
      "hasContent": true,
      "hasThinking": false
    }
  ],
  "session_id": "agent-ab9117a7e97626a37",
  "source_path": "/home/manuel/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a/subagents/agent-ab9117a7e97626a37.jsonl",
  "subagent_count": 0,
  "title": "Say hello and describe what you specialize in searching for. Keep it concise.",
  "tool_call_count": 0,
  "tool_counts": {},
  "turn_count": 2,
  "working_directory": "~/workspaces/2026-06-07/club-meetup-site/2026-05-27--rag-evaluation-system"
}
]
```
