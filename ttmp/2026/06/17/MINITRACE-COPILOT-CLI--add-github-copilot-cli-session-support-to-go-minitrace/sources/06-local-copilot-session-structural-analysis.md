---
Title: Local Copilot Session Structural Analysis
Ticket: MINITRACE-COPILOT-CLI
DocType: reference
Topics: [go-minitrace, copilot, conversion]
Status: active
Intent: source-evidence
---

# Local Copilot CLI session-state structural analysis

- Session directory: `/home/manuel/.copilot/session-state/e5b2d4a3-1027-4b0c-a6c4-fb5955855b2a`
- Exists: `True`

## Directory entries

- `checkpoints/index.md` (172 bytes)
- `events.jsonl` (250841 bytes)
- `rewind-snapshots/backups/1d8a198d707ef715-1781721336668` (23 bytes)
- `rewind-snapshots/index.json` (2769 bytes)
- `session.db` (28672 bytes)
- `workspace.yaml` (596 bytes)

## workspace.yaml keys

- `branch`: `task/club-meetup-site`
- `client_name`: `github/cli`
- `created_at`: `datetime`
- `cwd`: `/home/manuel/workspaces/2026-06-07/club-meetup-site/glazed`
- `git_root`: `/home/manuel/workspaces/2026-06-07/club-meetup-site/glazed`
- `host_type`: `github`
- `id`: `e5b2d4a3-1027-4b0c-a6c4-fb5955855b2a`
- `mc_last_event_id`: `01862be1-14e2-4042-9e88-1e97e8b80d75`
- `mc_session_id`: `5dd3bc24-186f-4ec5-a089-14608a4eaf3b`
- `mc_task_id`: `de695e22-417d-4db5-8357-ecfe12daa02a`
- `name`: `Session Initialization`
- `remote_steerable`: `False`
- `repository`: `go-go-golems/glazed`
- `summary_count`: `0`
- `updated_at`: `datetime`
- `user_named`: `False`

## events.jsonl structural summary

- JSON records: `72`
- Bad JSON lines: `0`

### Top-level keys

- `type`: 72
- `data`: 72
- `id`: 72
- `timestamp`: 72
- `parentId`: 72

### Record `type` counts

- `assistant.turn_start`: 9
- `assistant.message`: 9
- `assistant.turn_end`: 9
- `tool.execution_start`: 7
- `hook.start`: 7
- `hook.end`: 7
- `tool.execution_complete`: 7
- `system.message`: 4
- `user.message`: 4
- `session.model_change`: 2
- `permission.requested`: 2
- `permission.completed`: 2
- `session.start`: 1
- `session.info`: 1
- `session.shutdown`: 1

### Payload `type` counts


### Redacted example shapes by record type

- `session.start`: `{"data": {"alreadyInUse": "bool", "context": {"baseCommit": "str", "branch": "str", "cwd": "str", "gitRoot": "str", "headCommit": "str", "hostType": "str", "repository": "str", "repositoryHost": "str"}, "contextTier": "NoneType", "copilotVersion": "str", "producer": "str", "remoteSteerable": "bool", "sessionId": "str", "startTime": "str", "version": "int"}, "id": "str", "parentId": "NoneType", "timestamp": "str", "type": "str"}`
- `session.model_change`: `{"data": {"newModel": "str", "previousReasoningEffort": "str", "reasoningEffort": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `session.info`: `{"data": {"infoType": "str", "message": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `system.message`: `{"data": {"content": "str", "role": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `user.message`: `{"data": {"attachments": [], "content": "str", "interactionId": "str", "parentAgentTaskId": "str", "supportedNativeDocumentMimeTypes": [], "transformedContent": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `assistant.turn_start`: `{"data": {"interactionId": "str", "turnId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `assistant.message`: `{"data": {"apiCallId": "str", "content": "str", "encryptedContent": "str", "interactionId": "str", "messageId": "str", "model": "str", "outputTokens": "int", "phase": "str", "reasoningOpaque": "str", "requestId": "str", "serviceRequestId": "str", "toolRequests": [], "turnId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `assistant.turn_end`: `{"data": {"turnId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `tool.execution_start`: `{"data": {"arguments": {"command": "str", "description": "str", "initial_wait": "int", "mode": "str"}, "model": "str", "toolCallId": "str", "toolName": "str", "turnId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `hook.start`: `{"data": {"hookInvocationId": "str", "hookType": "str", "input": {"cwd": "str", "sessionId": "str", "timestamp": "int", "toolArgs": "str", "toolName": "str", "toolResult": {"resultType": "str", "sessionLog": "str", "textResultForLlm": "str", "toolTelemetry": "dict"}}}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `hook.end`: `{"data": {"hookInvocationId": "str", "hookType": "str", "success": "bool"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `tool.execution_complete`: `{"data": {"interactionId": "str", "model": "str", "result": {"content": "str", "detailedContent": "str"}, "success": "bool", "toolCallId": "str", "toolTelemetry": {"metrics": {"commandTimeout": "int"}, "properties": {"customTimeout": "str", "detached": "str", "executionMode": "str"}}, "turnId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `permission.requested`: `{"data": {"permissionRequest": {"canOfferSessionApproval": "bool", "commands": ["dict"], "fullCommandText": "str", "hasWriteFileRedirection": "bool", "intention": "str", "kind": "str", "possiblePaths": ["str"], "possibleUrls": [], "toolCallId": "str"}, "promptRequest": {"canOfferSessionApproval": "bool", "commandIdentifiers": ["str"], "fullCommandText": "str", "intention": "str", "kind": "str", "toolCallId": "str"}, "requestId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `permission.completed`: `{"data": {"requestId": "str", "result": {"kind": "str"}, "toolCallId": "str"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`
- `session.shutdown`: `{"data": {"codeChanges": {"filesModified": ["str"], "linesAdded": "int", "linesRemoved": "int"}, "conversationTokens": "int", "currentModel": "str", "currentTokens": "int", "eventsFileSizeBytes": "int", "modelMetrics": {"gpt-5.4-mini": {"requests": "dict", "tokenDetails": "dict", "totalNanoAiu": "int", "usage": "dict"}}, "sessionStartTime": "int", "shutdownType": "str", "systemTokens": "int", "tokenDetails": {"cache_read": {"tokenCount": "int"}, "input": {"tokenCount": "int"}, "output": {"tokenCount": "int"}}, "toolDefinitionsTokens": "int", "totalApiDurationMs": "int", "totalNanoAiu": "int", "totalPremiumRequests": "float"}, "id": "str", "parentId": "str", "timestamp": "str", "type": "str"}`

## session.db schema summary


### Table `inbox_entries`
- `id` TEXT notnull=0 pk=1 default=None
- `recipient_session_id` TEXT notnull=1 pk=0 default=None
- `sender_id` TEXT notnull=1 pk=0 default=None
- `sender_name` TEXT notnull=1 pk=0 default=None
- `sender_type` TEXT notnull=1 pk=0 default=None
- `interaction_id` TEXT notnull=1 pk=0 default=None
- `sequence` INTEGER notnull=1 pk=0 default=0
- `summary` TEXT notnull=1 pk=0 default=None
- `content` TEXT notnull=1 pk=0 default=None
- `unread` INTEGER notnull=1 pk=0 default=1
- `sent_at` INTEGER notnull=1 pk=0 default=None
- `read_at` INTEGER notnull=0 pk=0 default=None
- `notified_at` INTEGER notnull=0 pk=0 default=None
- row_count: `0`

### Table `todo_deps`
- `todo_id` TEXT notnull=1 pk=1 default=None
- `depends_on` TEXT notnull=1 pk=2 default=None
- row_count: `0`

### Table `todos`
- `id` TEXT notnull=0 pk=1 default=None
- `title` TEXT notnull=1 pk=0 default=None
- `description` TEXT notnull=0 pk=0 default=None
- `status` TEXT notnull=0 pk=0 default='pending'
- `created_at` TEXT notnull=0 pk=0 default=datetime('now')
- `updated_at` TEXT notnull=0 pk=0 default=datetime('now')
- row_count: `0`
