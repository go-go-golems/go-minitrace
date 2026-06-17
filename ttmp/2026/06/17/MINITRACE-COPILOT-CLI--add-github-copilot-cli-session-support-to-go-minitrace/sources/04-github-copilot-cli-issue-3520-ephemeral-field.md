---
Title: Github Copilot Cli Issue 3520 Ephemeral Field
Ticket: MINITRACE-COPILOT-CLI
DocType: reference
Topics: [go-minitrace, copilot, conversion]
Status: active
Intent: source-evidence
---

## Summary

Copilot CLI **1.0.54** writes session events to `~/.copilot/session-state/<sessionId>/events.jsonl` 
**without the `ephemeral` field**, while the schema it ships with at 
`pkg/universal/1.0.48/schemas/session-events.schema.json` lists `ephemeral` as **required** on 
every event. The same file is later validated by the VS Code "Copilot CLI" integration on resume, 
which fails with:

```
Failed to open chat session: Session file is corrupted (line N: ephemeral: Invalid literal value, 
expected true)
```

The Zod error message is slightly misleading — it surfaces the last failing union branch 
("expected true"); the underlying issue is that the field is absent entirely, not that it has the 
wrong value.

## Environment

- **Copilot CLI:** 1.0.54
- **OS:** Windows 11 Pro 26100
- **Node.js:** v25.9.0 (also reproduces under v24.16.0)
- **Schema in use:** `~/.copilot/pkg/universal/1.0.48/schemas/session-events.schema.json`

## Reproduction

1. Use the CLI for any non-trivial conversation (any session created on or before this version).
2. Exit the CLI (`/exit`).
3. In VS Code Stable, try to resume the session via the Copilot CLI integration.
4. The "Failed to open chat session" error appears, referencing a line number where the CLI wrote 
events.

## Affected event types

Snapshot from a single ~3-minute slice of one running CLI session — every event written by the 
CLI in that window was missing `ephemeral`:

| Count | Event `type` |
| --- | --- |
| 6 | `assistant.message` |
| 6 | `assistant.turn_start` |
| 6 | `permission.completed` |
| 6 | `permission.requested` |
| 6 | `tool.execution_start` |
| 5 | `assistant.turn_end` |
| 5 | `hook.end` |
| 5 | `hook.start` |
| 5 | `tool.execution_complete` |
| 1 | `session.resume` |
| 1 | `system.message` |
| 1 | `user.message` |

The bundled schema (`session-events.schema.json`) lists `ephemeral` as a required property with a 
discriminated boolean union (`const: true` for transient, `false` for persisted), but no events in 
the file carry the field.

## Attempted workaround (does not fully resolve)

I tried backfilling `"ephemeral": false` onto every event in `events.jsonl` while the CLI was fully 
exited (lock file released, no process appending). Verified afterwards that **every line** in the 
file carried the field. **VS Code still fails to resume with the same error**, on the same line 
number.

That suggests one or more of:

- The validator is reading something other than `events.jsonl` (e.g. a sibling metadata file or 
cached parse).
- The discriminated union has additional constraints beyond just the `ephemeral` field's 
presence/value, and the actual failure is elsewhere on the record but the error pretty-printer 
attributes it to `ephemeral`.
- The "line N" in the error is not a line in the on-disk file but a line in some materialised view 
the validator emits.

For reference, the backfill script that was used:

```powershell
$f = "$env:USERPROFILE\.copilot\session-state\<sessionId>\events.jsonl"
Copy-Item $f "$f.bak" -Force
$out = New-Object System.Collections.Generic.List[string]
foreach ($line in Get-Content $f) {
    if (-not $line.Trim()) { continue }
    $obj = $line | ConvertFrom-Json
    if (-not ($obj.PSObject.Properties.Name -contains 'ephemeral')) {
        $obj | Add-Member -NotePropertyName 'ephemeral' -NotePropertyValue $false
    }
    $out.Add( ($obj | ConvertTo-Json -Compress -Depth 100) )
}
[System.IO.File]::WriteAllLines($f, $out)
```

So this report is really two findings:

1. The CLI writer is emitting events that fail its own bundled schema (`ephemeral` always absent).
2. Even after manually fixing that on disk, VS Code resume still errors with the same message — 
suggesting the error message is at minimum misleading and possibly pointing to a different 
underlying problem.

## Expected behaviour

Either:

1. **The writer always emits `ephemeral`** on every event it appends (preferred — keeps the 
schema strict), or
2. **The schema makes `ephemeral` optional** with a default of `false` for persisted events.

Option 1 is the safer fix: schema strictness has caught a real bug here.

## Impact

- VS Code cannot resume any CLI session created or updated by 1.0.54, and (per the section above) a 
manual backfill of the missing field does not restore resumability either.
- Users who alternate between CLI and VS Code lose access to their session history through the GUI 
path.
- The CLI itself is unaffected — it tolerates the missing field on its own reads — so the 
failure surfaces only via the VS Code integration.

## Additional notes

- `~/.copilot/vscode.session.metadata.cache.json` continues to list the session as resumable, so 
the failure happens at file-parse time, not discovery time.
- No data is lost — the events are well-formed JSON apart from the missing field — and a manual 
rewrite is recoverable.
