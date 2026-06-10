---
Title: Agent Session Shape Survey
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
Summary: "Structural survey of recent local Pi, Codex, and Claude Code JSONL session formats."
LastUpdated: 2026-06-10T18:30:52.478395442-04:00
WhatFor: "Evidence for latest session format support gaps."
WhenToUse: "Use when updating adapters or adding preview tests for recent session shapes."
---
# Agent Session Shape Survey

Structural preview only; message text and prompt bodies are intentionally omitted.

## pi: `~/.pi/agent/sessions/--home-manuel-workspaces-2026-06-10-add-docs-deploy--/2026-06-10T15-08-40-711Z_019eb214-1387-7aa4-8e32-b5c1363445ae.jsonl`

- Bytes: 2154919
- Records sampled: 680
- Subagent path: False
- Session IDs: `001f0fdc, 007683fc, 009e0013, 013b696c, 019eb214-1387-7aa4-8e32-b5c1363445ae`
- Models: `gpt-5.5`
- Record types:
  - `message`: 660
  - `custom`: 15
  - `session`: 1
  - `model_change`: 1
  - `thinking_level_change`: 1
  - `session_info`: 1
  - `compaction`: 1
- Payload types:
  - none observed
- Content block types:
  - `text`: 357
  - `toolCall`: 332
  - `thinking`: 246
- Tool names:
  - `bash`: 223
  - `read`: 66
  - `write`: 24
  - `edit`: 19
- Top-level key shapes:
  - `['id', 'message', 'parentId', 'timestamp', 'type']`: 660
  - `['customType', 'data', 'id', 'parentId', 'timestamp', 'type']`: 15
  - `['cwd', 'id', 'timestamp', 'type', 'version']`: 1
  - `['id', 'modelId', 'parentId', 'provider', 'timestamp', 'type']`: 1
  - `['id', 'parentId', 'thinkingLevel', 'timestamp', 'type']`: 1
- Message key shapes:
  - `['api', 'content', 'model', 'provider', 'responseId', 'role', 'stopReason', 'timestamp', 'usage']`: 298
  - `['content', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 294
  - `['content', 'details', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 37
  - `['content', 'role', 'timestamp']`: 15
  - `['api', 'content', 'errorMessage', 'model', 'provider', 'role', 'stopReason', 'timestamp', 'usage']`: 13
- Tool-result key shapes:
  - `['content', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 294
  - `['content', 'details', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 37
- Image/blob indicators:
  - none observed

## pi: `~/.pi/agent/sessions/--home-manuel-workspaces-2026-06-07-club-meetup-site--/2026-06-09T20-33-18-780Z_019eae16-edbc-74a6-ac75-a963c9980fa9.jsonl`

- Bytes: 4925351
- Records sampled: 1327
- Subagent path: False
- Session IDs: `0060de03, 008e65b4, 0097180d, 009a905b, 00d0ea6c`
- Models: `glm-5.1, gpt-5.5, umans-qwen3.6-35b-a3b`
- Record types:
  - `message`: 1271
  - `custom`: 35
  - `thinking_level_change`: 11
  - `model_change`: 4
  - `compaction`: 3
  - `session_info`: 2
  - `session`: 1
- Payload types:
  - none observed
- Content block types:
  - `text`: 1122
  - `toolCall`: 653
  - `thinking`: 365
- Tool names:
  - `bash`: 264
  - `read`: 169
  - `edit`: 85
  - `write`: 69
  - `playwright_browser_take_screenshot`: 14
  - `playwright_browser_evaluate`: 14
  - `playwright_browser_navigate`: 13
  - `playwright_browser_wait_for`: 13
- Top-level key shapes:
  - `['id', 'message', 'parentId', 'timestamp', 'type']`: 1271
  - `['customType', 'data', 'id', 'parentId', 'timestamp', 'type']`: 35
  - `['id', 'parentId', 'thinkingLevel', 'timestamp', 'type']`: 11
  - `['id', 'modelId', 'parentId', 'provider', 'timestamp', 'type']`: 4
  - `['details', 'firstKeptEntryId', 'fromHook', 'id', 'parentId', 'summary', 'timestamp', 'tokensBefore', 'type']`: 3
- Message key shapes:
  - `['api', 'content', 'model', 'provider', 'responseId', 'role', 'stopReason', 'timestamp', 'usage']`: 562
  - `['content', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 471
  - `['content', 'details', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 181
  - `['content', 'role', 'timestamp']`: 40
  - `['api', 'content', 'errorMessage', 'model', 'provider', 'role', 'stopReason', 'timestamp', 'usage']`: 11
- Tool-result key shapes:
  - `['content', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 471
  - `['content', 'details', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 181
- Image/blob indicators:
  - none observed

## pi: `~/.pi/agent/sessions/--home-manuel-workspaces-2026-06-07-club-meetup-site--/2026-06-10T18-27-34-754Z_019eb2ca-2ce2-7332-ba1f-2dd453caf5af.jsonl`

- Bytes: 1428022
- Records sampled: 359
- Subagent path: False
- Session IDs: `0129c62c, 019eb2ca-2ce2-7332-ba1f-2dd453caf5af, 01f86ed4, 024f137d, 029ef867`
- Models: `gpt-5.5`
- Record types:
  - `message`: 349
  - `custom`: 6
  - `thinking_level_change`: 2
  - `session`: 1
  - `model_change`: 1
- Payload types:
  - none observed
- Content block types:
  - `text`: 233
  - `toolCall`: 177
  - `thinking`: 138
- Tool names:
  - `bash`: 88
  - `read`: 47
  - `edit`: 30
  - `write`: 10
  - `kagi_web_search`: 2
- Top-level key shapes:
  - `['id', 'message', 'parentId', 'timestamp', 'type']`: 349
  - `['customType', 'data', 'id', 'parentId', 'timestamp', 'type']`: 6
  - `['id', 'parentId', 'thinkingLevel', 'timestamp', 'type']`: 2
  - `['cwd', 'id', 'timestamp', 'type', 'version']`: 1
  - `['id', 'modelId', 'parentId', 'provider', 'timestamp', 'type']`: 1
- Message key shapes:
  - `['api', 'content', 'model', 'provider', 'responseId', 'role', 'stopReason', 'timestamp', 'usage']`: 159
  - `['content', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 130
  - `['content', 'details', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 46
  - `['api', 'content', 'errorMessage', 'model', 'provider', 'role', 'stopReason', 'timestamp', 'usage']`: 8
  - `['content', 'role', 'timestamp']`: 5
- Tool-result key shapes:
  - `['content', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 130
  - `['content', 'details', 'isError', 'role', 'timestamp', 'toolCallId', 'toolName']`: 46
- Image/blob indicators:
  - none observed

## codex: `~/.codex/sessions/2026/06/10/rollout-2026-06-10T11-04-24-019eb210-2ac8-77f1-a5a2-204e2ac77f0e.jsonl`

- Bytes: 112211
- Records sampled: 38
- Subagent path: False
- Session IDs: `019eb210-2ac8-77f1-a5a2-204e2ac77f0e`
- Models: `gpt-5.4-mini`
- Record types:
  - `response_item`: 25
  - `event_msg`: 11
  - `session_meta`: 1
  - `turn_context`: 1
- Payload types:
  - `message`: 7
  - `function_call`: 7
  - `function_call_output`: 7
  - `reasoning`: 4
  - `agent_message`: 4
  - `token_count`: 4
  - `<missing>`: 2
  - `task_started`: 1
- Content block types:
  - `input_text`: 7
  - `output_text`: 4
- Tool names:
  - `exec_command`: 7
- Top-level key shapes:
  - `['payload', 'timestamp', 'type']`: 38
- Message key shapes:
  - none observed
- Tool-result key shapes:
  - `['arguments', 'call_id', 'name', 'type']`: 7
  - `['call_id', 'output', 'type']`: 7
- Image/blob indicators:
  - none observed

## codex: `~/.codex/sessions/2026/06/10/rollout-2026-06-10T11-04-21-019eb210-1fd0-7d93-8c61-7f99c0dfe463.jsonl`

- Bytes: 126124
- Records sampled: 36
- Subagent path: False
- Session IDs: `019eb210-1fd0-7d93-8c61-7f99c0dfe463`
- Models: `gpt-5.4-mini`
- Record types:
  - `response_item`: 23
  - `event_msg`: 11
  - `session_meta`: 1
  - `turn_context`: 1
- Payload types:
  - `message`: 7
  - `function_call`: 6
  - `function_call_output`: 6
  - `reasoning`: 4
  - `agent_message`: 4
  - `token_count`: 4
  - `<missing>`: 2
  - `task_started`: 1
- Content block types:
  - `input_text`: 7
  - `output_text`: 4
- Tool names:
  - `exec_command`: 6
- Top-level key shapes:
  - `['payload', 'timestamp', 'type']`: 36
- Message key shapes:
  - none observed
- Tool-result key shapes:
  - `['arguments', 'call_id', 'name', 'type']`: 6
  - `['call_id', 'output', 'type']`: 6
- Image/blob indicators:
  - none observed

## codex: `~/.codex/sessions/2026/06/10/rollout-2026-06-10T10-58-46-019eb20b-0253-7882-9999-36979739f196.jsonl`

- Bytes: 3088183
- Records sampled: 198
- Subagent path: False
- Session IDs: `019eb20b-0253-7882-9999-36979739f196`
- Models: `gpt-5.4-mini`
- Record types:
  - `response_item`: 132
  - `event_msg`: 64
  - `session_meta`: 1
  - `turn_context`: 1
- Payload types:
  - `function_call`: 32
  - `function_call_output`: 32
  - `message`: 31
  - `token_count`: 31
  - `reasoning`: 27
  - `agent_message`: 23
  - `user_message`: 4
  - `custom_tool_call`: 4
- Content block types:
  - `output_text`: 23
  - `input_text`: 13
- Tool names:
  - `exec_command`: 18
  - `spawn_agent`: 7
  - `apply_patch`: 4
  - `wait_agent`: 4
  - `write_stdin`: 2
  - `view_image`: 1
- Top-level key shapes:
  - `['payload', 'timestamp', 'type']`: 198
- Message key shapes:
  - none observed
- Tool-result key shapes:
  - `['call_id', 'output', 'type']`: 36
  - `['arguments', 'call_id', 'name', 'type']`: 21
  - `['arguments', 'call_id', 'name', 'namespace', 'type']`: 11
  - `['call_id', 'input', 'name', 'status', 'type']`: 4
  - `['call_id', 'changes', 'status', 'stderr', 'stdout', 'success', 'turn_id', 'type']`: 4
- Image/blob indicators:
  - none observed

## claude: `~/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a/subagents/agent-a1d3474b55660fcc3.jsonl`

- Bytes: 16462
- Records sampled: 4
- Subagent path: True
- Session IDs: `e431a533-9e12-4300-b192-c52da630439a`
- Agent IDs: `a1d3474b55660fcc3`
- Models: `claude-haiku-4-5-20251001`
- Record types:
  - `attachment`: 2
  - `user`: 1
  - `assistant`: 1
- Payload types:
  - none observed
- Content block types:
  - `text`: 1
- Tool names:
  - none observed
- Top-level key shapes:
  - `['agentId', 'attachment', 'cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'parentUuid', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 2
  - `['agentId', 'cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'message', 'parentUuid', 'promptId', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 1
  - `['agentId', 'attributionAgent', 'cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'message', 'parentUuid', 'requestId', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 1
- Message key shapes:
  - `['content', 'role']`: 1
  - `['content', 'diagnostics', 'id', 'model', 'role', 'stop_details', 'stop_reason', 'stop_sequence', 'type', 'usage']`: 1
- Tool-result key shapes:
  - none observed
- Image/blob indicators:
  - none observed

## claude: `~/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a/subagents/agent-ab9117a7e97626a37.jsonl`

- Bytes: 16569
- Records sampled: 4
- Subagent path: True
- Session IDs: `e431a533-9e12-4300-b192-c52da630439a`
- Agent IDs: `ab9117a7e97626a37`
- Models: `claude-haiku-4-5-20251001`
- Record types:
  - `attachment`: 2
  - `user`: 1
  - `assistant`: 1
- Payload types:
  - none observed
- Content block types:
  - `text`: 1
- Tool names:
  - none observed
- Top-level key shapes:
  - `['agentId', 'attachment', 'cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'parentUuid', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 2
  - `['agentId', 'cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'message', 'parentUuid', 'promptId', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 1
  - `['agentId', 'attributionAgent', 'cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'message', 'parentUuid', 'requestId', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 1
- Message key shapes:
  - `['content', 'role']`: 1
  - `['content', 'diagnostics', 'id', 'model', 'role', 'stop_details', 'stop_reason', 'stop_sequence', 'type', 'usage']`: 1
- Tool-result key shapes:
  - none observed
- Image/blob indicators:
  - none observed

## claude: `~/.claude/projects/-home-manuel-workspaces-2026-06-07-club-meetup-site-2026-05-27--rag-evaluation-system/e431a533-9e12-4300-b192-c52da630439a.jsonl`

- Bytes: 119631
- Records sampled: 90
- Subagent path: False
- Session IDs: `e431a533-9e12-4300-b192-c52da630439a`
- Models: `claude-haiku-4-5-20251001`
- Record types:
  - `assistant`: 35
  - `user`: 22
  - `file-history-snapshot`: 7
  - `mode`: 5
  - `permission-mode`: 5
  - `ai-title`: 5
  - `attachment`: 4
  - `last-prompt`: 4
- Payload types:
  - none observed
- Content block types:
  - `tool_use`: 17
  - `tool_result`: 17
  - `thinking`: 9
  - `text`: 9
- Tool names:
  - `Agent`: 6
  - `Bash`: 6
  - `Write`: 4
  - `Read`: 1
- Top-level key shapes:
  - `['cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'message', 'parentUuid', 'requestId', 'sessionId', 'timestamp', 'type', 'userType', 'uuid', 'version']`: 35
  - `['cwd', 'entrypoint', 'gitBranch', 'isSidechain', 'message', 'parentUuid', 'promptId', 'sessionId', 'sourceToolAssistantUUID', 'timestamp', 'toolUseResult', 'type', 'userType', 'uuid', 'version']`: 17
  - `['isSnapshotUpdate', 'messageId', 'snapshot', 'type']`: 7
  - `['mode', 'sessionId', 'type']`: 5
  - `['permissionMode', 'sessionId', 'type']`: 5
- Message key shapes:
  - `['content', 'diagnostics', 'id', 'model', 'role', 'stop_details', 'stop_reason', 'stop_sequence', 'type', 'usage']`: 35
  - `['content', 'role']`: 22
- Tool-result key shapes:
  - none observed
- Image/blob indicators:
  - none observed

