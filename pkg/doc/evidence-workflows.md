---
Title: Evidence workflows and command scope
Slug: evidence-workflows
Short: Choose commands, preserve completeness, and count zero-tool sessions
Topics:
- minitrace
- javascript
IsTemplate: false
IsTopLevel: true
ShowPerDefault: false
SectionType: Tutorial
---

## Choose the command before choosing flags

| Command | Inputs and controls | Evidence output |
| --- | --- | --- |
| `query run` | `--archive-glob`, `--sql`/`--sql-file`/`--preset`, `--max-rows`, `--max-cell-bytes`, `--cache-mode`, `--strict-evidence` | Rows by default; `--result-envelope` preserves metadata; `--run-record` records failures |
| `query commands ... group verb` | `--query-repository`, leaf `--archive-glob`, declared custom fields | JS chooses `MaxRows`, `MaxCellBytes`, `StrictEvidence`, and output shape |
| `convert codex` | Native Codex input; see its `--help` | Supports `--run-record` |
| `convert claude-code`, `convert pi` | Native input; see the specific command's `--help` | Do not accept the Codex `--run-record` flag |

Flags belong to their command. Configure JS query policies through its database builder. A multi-verb JS filename creates a group and `__verb__` names each leaf. A single verb whose name matches its filename is collapsed: `sessions.js` with `__verb__("sessions", ...)` runs as `query commands sessions`, not `sessions sessions`. A single verb with a different name retains the filename group. Run the checked-in example:

```bash
go-minitrace query commands --query-repository ./testdata/query-repositories/evidence-workflows analysis session-tools counts --archive-glob './output/**/*.minitrace.json' --output json
```

Use the complete path before requesting `--help`; group help does not list a child's runtime flags. This example uses `SessionToolCounts()` to count actual `tool_calls` rows while retaining sessions with zero tools. Stored metrics are separate evidence and need not equal those rows.

## Retain query fidelity metadata

```bash
go-minitrace query run --archive-glob './output/**/*.minitrace.json' --sql 'SELECT session_id FROM sessions ORDER BY session_id' --max-rows 10 --cache-mode memory --result-envelope --output json
```

JSON output is a one-element array. Inspect `.[0].complete`, `.[0].rowsTruncated`, and `.[0].cellDiagnostics` before consuming `.[0].rows`. A successful empty query emits one envelope with `rows: []` and `count: 0`. A permissive clipped query emits diagnostics. `--strict-evidence` rejects incomplete results before emission; add `--run-record receipt.json` to preserve failure metadata. SQL errors exit nonzero and do not become successful envelope rows.

Limits are UTF-8 bytes, not tokens or character counts. Omitted `truncated`, `rowsTruncated`, or `cellDiagnostics` means false, false, or no diagnostics respectively. Default rows-only output has no metadata envelope.

## Bind values and discover the schema

```js
const result = db.queryResult(
  "SELECT session_id FROM sessions WHERE agent_framework = ? ORDER BY session_id",
  "claude-code"
);
if (result.error || !result.complete) throw new Error(result.error || "Incomplete evidence");
```

`queryResult(sql, ...args)` accepts SQL strings; `execute(recipe)` accepts a recipe or `{sql, args}`. Bind values instead of interpolating them. `queryOne` returns `null` when no row matches. Error envelopes always contain array-valued `rows` and `columns`, but callers must check `error` before interpreting an empty result.

Run `go-minitrace help minitrace-schema` for normalized columns. In JS, inspect `db.tables()` and `db.schema()` rather than guessing names. `go-minitrace help js-api-reference` includes native compaction projection and typed coordinate examples. Native `Inspect()` uses nested `boundary`, `summaries`, and `firstFollowingAssistant` coordinates; absent optional fields are not zero-valued coordinates.
