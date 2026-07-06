---
Title: Annotation Playbook
Slug: annotation-playbook
Short: Operator playbook for adding, reviewing, syncing, and querying annotations correctly from the CLI.
Topics:
- minitrace
- annotations
- tutorial
- cli
- sqlite
- transcript-analysis
Commands:
- annotate
- annotate add
- annotate list
- annotate edit
- annotate delete
- annotate sync
- annotate import
- query run
- validate
Flags:
- --output-dir
- --session
- --scope
- --target-id
- --category
- --title
- --detail
- --tags
- --taxonomy-minitrace
- --classification
- --archive-glob
- --dry-run
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This playbook explains how to add annotations to `go-minitrace` archives correctly from the command line. It is written for an operator or LLM agent that is actively reviewing transcripts and needs a reliable workflow rather than just a flag reference.

The central rule is simple: **write annotations into the SQLite working store first, then sync them back into `.minitrace.json` when you want the canonical file archive updated.**

That one rule determines which command to call at each step and prevents most mistakes.

## The mental model you should keep in mind

This section explains the storage model because it affects every command choice.

There are three places annotations can matter:

1. **SQLite working store**
   - file: `output/annotations.db`
   - used by `go-minitrace annotate ...`
   - best place for add/edit/delete workflows
2. **`.minitrace.json` session files**
   - portable archive format
   - only updated by `go-minitrace annotate sync`
3. **Normalized SQL query layer**
   - `go-minitrace serve` ATTACHes the SQLite working store as schema `anno`, so SQL run in the web Query Editor can read live annotations as `anno.annotations`
   - `go-minitrace query run` reads annotations from the JSON archives it loads (the `annotations` table)

The practical consequence is important:

- if you are using the **annotation CLI**, you are editing SQLite
- if you want **JSON files updated**, you must run `annotate sync`
- if you want `query run` to see your latest annotations, **sync first**
- if you are using the **web UI served by `go-minitrace serve`**, SQL against `anno.annotations` sees the live store without a sync step

## When to call which command

This section is the shortest decision guide for an agent.

| Goal | Command | Why this is the right command |
|------|---------|-------------------------------|
| Add one annotation | `annotate add` | Writes directly to the SQLite working store |
| Review existing annotations | `annotate list` | Fast filtered listing from SQLite |
| Fix text or category | `annotate edit` | Patch only the fields you set |
| Remove a mistaken annotation | `annotate delete` | Deletes from SQLite |
| Bulk-load prepared annotations | `annotate import` | Insert a JSON array into SQLite |
| Persist changes into session JSON | `annotate sync` | Writes SQLite annotations back into `.minitrace.json` |
| Analyze synced annotations in CLI SQL | `query run` | Reads the archive after sync |
| Check resulting JSON structure | `validate` | Confirms the written files are structurally valid |

If you only remember one workflow, remember this one:

```bash
# inspect target session
go-minitrace annotate list --output-dir ./output --session <SESSION_ID>

# add or change annotations
go-minitrace annotate add  ...
go-minitrace annotate edit ...

# preview the file writeback
go-minitrace annotate sync --output-dir ./output --session <SESSION_ID> --dry-run

# write to canonical JSON
go-minitrace annotate sync --output-dir ./output --session <SESSION_ID>

# verify archive integrity
go-minitrace validate --path ./output --recursive
```

## Before you annotate: identify the right target

This section covers the target IDs you need before calling `annotate add`.

Every annotation must attach to exactly one of these scopes:

- `session`
- `turn`
- `tool_call`

### Session scope

Use session scope when the judgment is about the whole run.

Examples:

- “Good representative failure case”
- “This session is a false positive and should not be counted”
- “Worth revisiting for a taxonomy review”

For session scope:

- `--scope` can be omitted because it defaults to `session`
- `--target-id` can be omitted because it defaults to the session ID

### Turn scope

Use turn scope when the note is about a specific conversational move.

Examples:

- a misleading user prompt
- an assistant hallucination
- a turn that clearly changes strategy
- a turn that should be labeled as a question or improvement point

For turn scope:

- pass `--scope turn`
- pass `--target-id <TURN_INDEX>`
- the target ID is the turn index as a string, for example `0`, `14`, or `27`

### Tool-call scope

Use tool-call scope when the note is about one concrete tool invocation.

Examples:

- a failing shell command
- a bad file edit attempt
- a tool result that the model ignored
- a tool call that reveals an environment issue rather than model failure

For tool-call scope:

- pass `--scope tool_call`
- pass `--target-id <TOOL_CALL_ID>`
- the target ID is the actual tool-call ID, not the tool name

A common mistake is to use `bash`, `read`, or `edit` as the target. That is wrong. The target must be the unique tool-call identifier from the transcript.

## How to find session IDs before annotating

This section focuses on the simplest CLI path.

If you need a session ID, start with the session list preset:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

If you want machine-readable output for an agent loop:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list \
  --output json
```

If you already know part of the title and want a narrower match:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT session_id, title, agent_framework
    FROM sessions
    WHERE LOWER(title) LIKE '%auth%'
    ORDER BY session_id DESC
  "
```

If you need a tool-call ID for a `tool_call` annotation, the easiest path is usually the Transcript Viewer in `go-minitrace serve`, because it shows the tool call in context. If you are staying CLI-only, sync first if needed and then inspect the session JSON directly or run a query that expands tool calls.

A CLI example that lists tool-call IDs for one session is:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT session_id, tool_call_id, tool_name
    FROM tool_calls
    WHERE session_id = '019bb3f6-3c71-7013-b585-4f16d9bdceb6'
    ORDER BY emitting_turn_index
  "
```

## How to add annotations: simple examples first

This section starts with the smallest valid commands and then adds detail.

### Example 1: minimal session annotation

Use this when you only need to mark the whole session.

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --category observation \
  --title "Interesting session to review later"
```

This writes one annotation row into `./output/annotations.db`. It does **not** yet modify the `.minitrace.json` file.

### Example 2: session annotation with detail and tags

Use this when you want a short title plus richer review notes.

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --category to-discuss \
  --title "Needs taxonomy review" \
  --detail "Session mixes environment issues with model planning mistakes." \
  --tags review,taxonomy,mixed-failure \
  --annotator manuel
```

### Example 3: turn-scoped annotation

Use this when the important judgment belongs to one turn rather than the whole session.

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --scope turn \
  --target-id 14 \
  --category ai-failure \
  --title "Model ignored earlier evidence" \
  --detail "The assistant retried a stale hypothesis even though the previous tool result contradicted it."
```

### Example 4: tool-call annotation

Use this when the note is about one concrete invocation.

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --scope tool_call \
  --target-id call_Y70XEopD3Ef1mGctwTXG2CEq \
  --category environment-issue \
  --title "Shell command failed because dependency is missing" \
  --detail "This is not a planning failure. The environment lacked the required executable."
```

### Example 5: annotation with taxonomy and classification

Use this when the annotation is intended for later failure analysis and not just casual note-taking.

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --scope turn \
  --target-id 9 \
  --category ai-failure \
  --title "Autonomy failure during diagnosis" \
  --detail "The model switched tactics without grounding the decision in the observed output." \
  --taxonomy-minitrace F-AUT,O-INT \
  --tags autonomy,diagnosis \
  --classification internal \
  --annotator review-agent
```

## A recommended agent workflow for reviewing one session

This section gives a complete, reliable review loop.

### Step 1: check what already exists

Always check for existing annotations before adding new ones. This avoids duplicate notes and helps an agent continue a previous review rather than restarting it.

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6
```

If you want JSON output for downstream reasoning:

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --format json
```

### Step 2: add the new judgments

Add one or more annotations. For example:

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --category question \
  --title "Unclear whether user or model caused the failure"

go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --scope turn \
  --target-id 22 \
  --category to-improve \
  --title "Would benefit from explicit plan recap"
```

### Step 3: inspect again and capture annotation IDs

After adding, list annotations again so you can record the IDs for later editing or deletion.

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --format json
```

### Step 4: patch mistakes instead of re-adding

If the category or title is wrong, patch the existing annotation.

```bash
go-minitrace annotate edit \
  --output-dir ./output \
  --id 69d7b1fc-b2df-4dcd-a639-b1c0b40da45b \
  --category user-error \
  --title "Prompt ambiguity caused the detour"
```

`annotate edit` only applies the flags you explicitly set. That makes it safe for partial correction.

### Step 5: dry-run the writeback

Before updating the archive files, preview the sync step.

```bash
go-minitrace annotate sync \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --dry-run
```

### Step 6: sync the canonical JSON

Once the annotations look right, write them back.

```bash
go-minitrace annotate sync \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6
```

### Step 7: validate the archive

Run validation after sync if the archive is intended for further processing or handoff.

```bash
go-minitrace validate --path ./output --recursive
```

## Listing and filtering annotations correctly

This section shows how to inspect the SQLite working store without touching JSON.

### List one session

```bash
go-minitrace annotate list --output-dir ./output --session <SESSION_ID>
```

### List only AI failures

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --category ai-failure
```

### List only turn-scoped annotations

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --scope turn
```

### List by taxonomy code

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --taxonomy F-AUT
```

### Get machine-readable results for an agent

```bash
go-minitrace annotate list \
  --output-dir ./output \
  --format json \
  --limit 200
```

This is the best listing form when another agent step needs to inspect IDs, categories, titles, timestamps, or target scopes programmatically.

## Editing and deleting annotations safely

This section explains how to correct mistakes without creating extra noise.

### Change only the title

```bash
go-minitrace annotate edit \
  --output-dir ./output \
  --id <ANNOTATION_ID> \
  --title "Better title"
```

### Change category and detail together

```bash
go-minitrace annotate edit \
  --output-dir ./output \
  --id <ANNOTATION_ID> \
  --category environment-issue \
  --detail "Root cause was missing toolchain dependency, not model reasoning."
```

### Add or replace tags

```bash
go-minitrace annotate edit \
  --output-dir ./output \
  --id <ANNOTATION_ID> \
  --tags shell,dependency,missing-binary
```

### Delete a mistaken annotation

```bash
go-minitrace annotate delete \
  --output-dir ./output \
  --id <ANNOTATION_ID>
```

Use deletion sparingly. In many cases `edit` is better because it preserves the annotation identity and review continuity.

## Importing prepared annotations in bulk

This section is for cases where an agent has already generated a JSON array of annotation objects.

Example file:

```json
[
  {
    "id": "11111111-1111-1111-1111-111111111111",
    "session_id": "sess-001",
    "timestamp": "2026-04-04T12:00:00Z",
    "annotator": "review-agent",
    "scope_type": "session",
    "target_id": "sess-001",
    "category": "observation",
    "title": "Imported example",
    "detail": "Created outside the CLI and then imported",
    "tags": ["imported"],
    "taxonomy_minitrace": ["F-AUT"],
    "taxonomy_mast": [],
    "taxonomy_toolemu": [],
    "classification": "internal"
  }
]
```

Import it:

```bash
go-minitrace annotate import \
  --output-dir ./output \
  --file ./annotations.json
```

Or stream from stdin:

```bash
cat ./annotations.json | go-minitrace annotate import \
  --output-dir ./output \
  --file -
```

Use import when the annotation objects already exist as JSON. Use `annotate add` when you are authoring one annotation interactively.

## Querying annotations after sync

This section covers the correct CLI-only analysis flow.

Remember the rule from earlier: `query run` reads the archive files it loads. It does **not** read the SQLite annotation store directly. That means your latest annotations must be synced first.

### First sync

```bash
go-minitrace annotate sync --output-dir ./output
```

### Then use the built-in annotations preset

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset annotations
```

### Then use ad hoc SQL for more specific questions

Annotations are rows in the normalized `annotations` table, so no JSON unnesting is needed.

Count annotations by category:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT category, COUNT(*) AS n
    FROM annotations
    GROUP BY category
    ORDER BY n DESC
  "
```

Find all tool-call annotations:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT session_id, target_id AS tool_call_id, category, title
    FROM annotations
    WHERE scope_type = 'tool_call'
  "
```

Join annotations with session metadata:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT a.session_id, s.agent_framework, s.model, a.category, a.title
    FROM annotations a
    JOIN sessions s USING (session_id)
    ORDER BY a.session_id
  "
```

### Querying the live store from the web UI

In `go-minitrace serve`, the SQLite working store is ATTACHed as schema `anno`, so SQL run in the web Query Editor can read annotations that have **not** been synced yet:

```sql
SELECT s.session_id, s.agent_framework, a.category, a.title
FROM anno.annotations a
JOIN sessions s ON s.session_id = a.session_id;
```

This is the fastest way to review labels while you are still annotating. The archive-backed `annotations` table only catches up after `annotate sync`.

## Advanced example: review a session and classify three different failure types

This section gives a realistic, multi-step operator sequence.

Imagine a session where:

- the overall run is worth keeping for study
- one turn contains a model reasoning failure
- one tool call failed because of the environment

A good annotation sequence would look like this:

```bash
# 1. Mark the session as worth keeping
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --category observation \
  --title "Representative debugging session" \
  --tags reference,review

# 2. Mark the turn-level reasoning problem
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --scope turn \
  --target-id 14 \
  --category ai-failure \
  --title "Reasoning drift after contradictory evidence" \
  --detail "The assistant continued with the old plan despite a tool result that should have forced replanning." \
  --taxonomy-minitrace F-AUT,O-INT \
  --tags planning,evidence

# 3. Mark the tool-call-level environment problem
go-minitrace annotate add \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --scope tool_call \
  --target-id call_Y70XEopD3Ef1mGctwTXG2CEq \
  --category environment-issue \
  --title "Missing binary in runtime environment" \
  --detail "This should not be counted as a model failure in aggregate analytics." \
  --tags shell,env

# 4. Inspect current state
go-minitrace annotate list \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6 \
  --format json

# 5. Sync only that session
go-minitrace annotate sync \
  --output-dir ./output \
  --session 019bb3f6-3c71-7013-b585-4f16d9bdceb6

# 6. Validate the archive
go-minitrace validate --path ./output --recursive
```

That sequence is a good model because it keeps the three scopes distinct instead of collapsing everything into a single session-level label.

## Advanced example: batch-review several sessions with one consistent rubric

This section is useful when an agent is reviewing a queue of sessions with the same labeling logic.

Suppose the rubric is:

- session-level `to-discuss` if the whole session is interesting
- turn-level `ai-failure` for reasoning failures
- tool-call-level `environment-issue` for external failures

A robust pattern is:

1. get candidate session IDs with `query run`
2. for each session, inspect existing annotations with `annotate list --format json`
3. add only missing annotations
4. sync at the end of the batch
5. validate once after the batch

Example candidate query:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT session_id, title
    FROM sessions
    WHERE tool_call_count > 20
    ORDER BY session_id DESC
    LIMIT 10
  " \
  --output json
```

Then annotate each chosen session in the same style. This is preferable to improvising category meanings session by session.

## Common mistakes and how to avoid them

This section covers the errors that show up most often in real use.

### Mistake: forgetting `--output-dir`

If you read from one output tree and write to another, your annotations will appear to vanish.

Rule: use the same `--output-dir` consistently across `add`, `list`, `edit`, `delete`, `import`, and `sync`.

### Mistake: using tool name instead of tool-call ID

Wrong:

```bash
--scope tool_call --target-id bash
```

Right:

```bash
--scope tool_call --target-id call_Y70XEopD3Ef1mGctwTXG2CEq
```

### Mistake: expecting `query run` to see unsynced annotations

`query run` loads `.minitrace.json`. If the annotations only exist in SQLite, the query results will not include them yet.

Rule: `annotate sync` first, then `query run`. (Or query `anno.annotations` in the `serve` web UI, which sees the live store.)

### Mistake: overusing session scope

If the issue belongs to one turn or one tool call, use the narrower scope. Session scope is best for whole-run judgments.

### Mistake: deleting when editing would be better

If the annotation is basically right but poorly worded, use `annotate edit`. Save deletion for genuinely mistaken entries.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `annotate add` fails because category is invalid | Category must be one of the known hardcoded values | Use one of: `observation`, `ai-failure`, `user-error`, `environment-issue`, `success`, `question`, `to-discuss`, `to-improve` |
| `annotate add` with `--scope turn` or `--scope tool_call` behaves oddly | Missing or wrong `--target-id` | Pass the exact turn index or tool-call ID |
| `annotate list` shows nothing | Wrong `--output-dir` or wrong session filter | Re-run with the correct output root and verify the session ID |
| `query run` does not show new annotations | You did not sync SQLite back to JSON | Run `go-minitrace annotate sync --output-dir ...` first, or query `anno.annotations` in the web UI |
| `annotate sync` misses files | Nonstandard archive layout | Override `--archive-glob` explicitly |
| `validate` reports annotation structure issues | Invalid category, scope, tags, taxonomy arrays, or classification | Fix the data with `annotate edit` or remove the bad row and re-add it correctly |

## See also

- `go-minitrace help getting-started` — basic archive workflow from discovery to query
- `go-minitrace help query-commands` — `query run` flags and preset list
- `go-minitrace help writing-queries` — how to write custom SQL against the normalized schema
- `go-minitrace help validate-command` — validation behavior and flags
- `go-minitrace annotate add --help` — exact flag reference for creation
- `go-minitrace annotate sync --help` — exact flag reference for sync behavior
