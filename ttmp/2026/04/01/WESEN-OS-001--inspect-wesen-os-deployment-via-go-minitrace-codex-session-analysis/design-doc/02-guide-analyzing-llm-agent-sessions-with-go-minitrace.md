---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/10-human-blocks.sql
      Note: Core query - human activity blocks
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/14-autopilot-blocks.py
      Note: Core script - autopilot detection
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/20-run-all-wesen-os-analysis.sh
      Note: Master runner script
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/21-doc-creation-timeline.sql
      Note: Ticket and document creation events
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/22-diary-writes.sql
      Note: Diary content extraction from write calls
    - Path: ttmp/2026/04/01/WESEN-OS-001--inspect-wesen-os-deployment-via-go-minitrace-codex-session-analysis/scripts/24-git-commits.sql
      Note: Git commit log extraction
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Guide: Analyzing LLM Agent Sessions with go-minitrace

## Purpose

This document describes a practical methodology for analyzing LLM coding agent sessions (Codex, Claude Code, Pi) using `go-minitrace`. The goal is to reconstruct what happened in a set of sessions, understand the interaction patterns between human and agent, and produce a narrative timeline of work that can be used for retrospectives, onboarding, and process improvement.

The methodology was developed during the WESEN-OS-001 investigation, where we analyzed three Codex sessions spanning 113 hours of wall-clock time, 1,903 turns, and 4,622 tool calls across a two-week wesen-os deployment effort. All queries referenced here are in this ticket's `scripts/` directory.

---

## Overview of the Pipeline

```
  ┌──────────┐     ┌──────────┐     ┌──────────────────────┐
  │ discover │ ──▶ │ convert  │ ──▶ │ query (DuckDB SQL)   │
  │          │     │          │     │                      │
  │ list     │     │ to       │     │ human blocks         │
  │ sessions │     │ .mini-   │     │ artifact timeline    │
  │          │     │ trace.   │     │ diary content        │
  │          │     │ json     │     │ commit log           │
  └──────────┘     └──────────┘     └──────────────────────┘
```

The three phases:
1. **Discover** — enumerate available sessions, understand date ranges and format support.
2. **Convert** — transform native session JSONL into minitrace archives queryable by DuckDB.
3. **Query** — run SQL against the `sessions_base` table to extract structured data, then post-process with Python scripts.

---

## Phase 1: Discover and Convert

### Discover

```bash
go-minitrace discover codex
```

This lists all sessions with their ID, format hint, and source path. Only `session-jsonl-v1` sessions are convertible; older `unknown-jsonl` sessions will cause convert to abort.

**Practical issue:** `convert` fails on the first unsupported session with no `--skip-unsupported` flag. The workaround is to copy only the supported date range to a scratch directory:

```bash
mkdir -p /tmp/codex-recent/sessions/2026
for day in 18 19 20 21 22 23 24 25 26 27 28 29 30 31; do
  [ -d ~/.codex/sessions/2026/03/$day ] && \
    mkdir -p /tmp/codex-recent/sessions/2026/03 && \
    cp -r ~/.codex/sessions/2026/03/$day /tmp/codex-recent/sessions/2026/03/
done
```

Note: symlinks do not work — the discover walk does not follow symlinked directories.

### Convert

```bash
go-minitrace convert codex \
  --source-dir /tmp/codex-recent \
  --output-dir /tmp/minitrace-output
```

The output is one `.minitrace.json` file per session under `output/active/YYYY-MM/`. A `manifest.json` is written with aggregate statistics.

---

## Phase 2: Understand the Schema

Before writing queries, probe the schema:

```bash
go-minitrace query duckdb \
  --archive-glob '/tmp/minitrace-output/active/*/*.minitrace.json' \
  --sql "DESCRIBE sessions_base"
```

### Key columns and JSON paths

| What you want | How to get it |
|---|---|
| Session ID | `id` |
| Title (truncated prompt) | `title` |
| Start time | `timing->>'started_at'` |
| Wall-clock duration | `timing->>'duration_seconds'` |
| Active (non-idle) duration | `timing->>'active_duration_seconds'` |
| Turn count | `metrics->>'turn_count'` |
| Tool call count | `metrics->>'tool_call_count'` |
| Working directory | `operational_context->>'working_directory'` |
| First user prompt | `turns[1]->>'content'` (1-indexed!) |
| Turn role | `turns[N]->>'role'` (`user` or `assistant`) |
| Turn timestamp | `turns[N]->>'timestamp'` |
| Tool call command | `tool_calls[N]->'input'->'arguments'->>'cmd'` |
| Tool call output | `tool_calls[N]->'output'->>'result'` |
| Tool call success | `tool_calls[N]->'output'->>'success'` |
| Tool name | `tool_calls[N]->>'tool_name'` |
| Tool calls in a turn | `turns[N]->'tool_calls_in_turn'` (array of call IDs) |

**DuckDB JSON gotchas:**
- Arrays are **1-indexed**: `turns[1]` is the first turn.
- Use `->>'field'` for scalar extraction (returns VARCHAR), `->'field'` for nested JSON.
- UNNEST with ordinality: `CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)`.
- Always cast: `CAST(metrics->>'turn_count' AS INT)`.

---

## Phase 3: Finding Target Sessions

### Step 1: Broad session list

Use the `session-list` preset for an overview:

```bash
go-minitrace query duckdb \
  --archive-glob '...' \
  --preset session-list
```

### Step 2: Filter by topic

Search by title, working directory, and first-turn content simultaneously. Title alone misses sessions that start with generic prompts or `cd` commands:

```sql
-- scripts/04-wesen-os-strict.sql
SELECT id, timing->>'started_at' AS started_at, title,
  operational_context->>'working_directory' AS workdir
FROM sessions_base
WHERE
  LOWER(operational_context->>'working_directory') LIKE '%wesen-os%'
  OR LOWER(title) LIKE '%wesen-os%'
  OR LOWER(CAST(turns[1]->>'content' AS VARCHAR)) LIKE '%wesen-os%'
ORDER BY timing->>'started_at';
```

The working directory is often the most precise signal: if a session ran from a `wesen-os` directory, it is wesen-os work regardless of what the title says.

---

## Phase 4: The Analysis Layers

Once you have the target session IDs, the analysis proceeds in layers, each building on the previous.

### Layer 1: Human Activity Blocks

**Purpose:** Decompose each session into blocks where the human gave an instruction and the agent ran autonomously until the next human input. This is the structural spine of the narrative.

```sql
-- scripts/10-human-blocks.sql (replace SESSION_ID)
WITH numbered AS (
  SELECT t.idx,
    CAST(t.turn->>'role' AS VARCHAR) AS role,
    CAST(t.turn->>'content' AS VARCHAR) AS content,
    CAST(t.turn->>'timestamp' AS VARCHAR) AS ts,
    json_array_length(COALESCE(t.turn->'tool_calls_in_turn', '[]'::JSON)) AS tc_count
  FROM sessions_base
  CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
  WHERE id = 'SESSION_ID'
),
blocks AS (
  SELECT *,
    SUM(CASE WHEN role = 'user' THEN 1 ELSE 0 END)
      OVER (ORDER BY idx) AS block_num
  FROM numbered
)
SELECT
  block_num AS blk,
  MIN(CASE WHEN role = 'user' THEN idx END) AS turn,
  MIN(CASE WHEN role = 'user' THEN ts END) AS user_ts,
  LEFT(MIN(CASE WHEN role = 'user' THEN content END), 200) AS user_prompt,
  COUNT(*) FILTER (WHERE role != 'user') AS agent_turns,
  SUM(tc_count) AS tool_calls
FROM blocks
GROUP BY block_num
ORDER BY block_num;
```

**What this tells you:**
- How many times the human intervened (number of blocks).
- How much autonomous work followed each intervention (agent_turns, tool_calls).
- Whether the human was actively steering or just saying "continue" (prompt content).
- The timestamps let you compute gaps between human inputs.

**Autopilot detection:** Post-process the JSON output with a Python script (scripts/14-autopilot-blocks.py) to identify blocks where the user prompt is just "continue", "ok", "go ahead", "yes", etc. In our wesen-os analysis, 24-28% of all tool calls happened in autopilot blocks.

### Layer 2: Artifact Creation Timeline

**Purpose:** Identify when new tickets and documents were created during the session. These are the structural landmarks — each ticket creation is a scope event, each document creation is a deliverable.

```sql
-- scripts/21-doc-creation-timeline.sql
SELECT s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) AS cmd
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN ('SESSION_ID_1', 'SESSION_ID_2')
  AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
  AND (
    CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%docmgr doc add%'
    OR CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%docmgr ticket create%'
  )
ORDER BY CAST(tc->>'timestamp' AS VARCHAR);
```

Parse the `--ticket`, `--title`, `--doc-type` flags from the command string to extract structured metadata.

Also capture raw `mkdir ttmp` calls (scripts/19-ticket-creation-timeline.sql), since not all sessions use `docmgr` — some create ticket directories manually.

### Layer 3: Git Commits

**Purpose:** Ground truth of what code was actually shipped. Commit messages are the best short summary of each unit of work.

```sql
-- scripts/24-git-commits.sql
SELECT s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  LEFT(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR), 300) AS cmd,
  LEFT(CAST(tc->'output'->>'result' AS VARCHAR), 200) AS result
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN ('SESSION_ID_1', ...)
  AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
  AND CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%git commit%'
  AND CAST(tc->'output'->>'success' AS BOOLEAN) = true
ORDER BY s.id, CAST(tc->>'timestamp' AS VARCHAR);
```

Extract the commit message from the `-m "..."` flag in the command string with a regex. In our analysis this yielded 146 commits, each a concrete work artifact.

### Layer 4: Diary Content from Write Calls

**Purpose:** Read the actual narrative the agent wrote about its own work. Diary files are the richest information source — they contain step-by-step records of decisions, failures, and lessons learned.

**Important:** Do not rely on reading diary files from disk. The files may have been moved, renamed, or deleted since the session ran (in our analysis, 3 of 8 diary files were missing from their original paths because tickets were moved between repos mid-session). Instead, extract the diary content from the tool calls that wrote it.

There are two approaches:

#### Approach A: Extract diary content from `write_stdin` / `apply_patch` / `cat >` / `tee` calls

Find all tool calls that wrote to diary paths and extract the content from the command or from the `output.result` field:

```sql
-- scripts/22-diary-writes.sql
SELECT s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  CAST(tc->'output'->>'success' AS VARCHAR) AS success,
  CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) AS cmd
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN ('SESSION_ID_1', ...)
  AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
  AND LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%diary%'
  AND (
    LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%apply_patch%'
    OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%tee %'
    OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%cat >%'
    OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%cat <<_%'
  )
ORDER BY s.id, CAST(tc->>'timestamp' AS VARCHAR);
```

The `cmd` field for `cat >` and heredoc writes contains the full file content inline. For `apply_patch` calls, the patch content is in the command arguments and shows the diff applied.

This approach captures the content as it was at the time of writing — which is what you want for a retrospective, since it reflects the agent's understanding at that moment, not the final state after subsequent edits.

#### Approach B: Find diary file paths, then selectively read from disk

Use the path-extraction query (scripts/23-find-diary-files.sql) to get all distinct diary paths referenced in tool calls, then check which still exist:

```bash
for f in $(go-minitrace query duckdb ... --select diary_path); do
  [ -f "$f" ] && echo "EXISTS $(wc -l < "$f") lines  $f"
done
```

This gives you the final state of each diary. It's useful when the files haven't moved, but it misses deleted or relocated content and conflates all edits into the final version.

**Recommendation:** Use Approach A (write-call extraction) as the primary source, with Approach B as a convenience for reading the final polished version when files are still in place.

### Layer 5: docmgr Operations

**Purpose:** Quantify documentation management overhead and understand the ticket lifecycle within each session.

```sql
-- scripts/15-docmgr-and-ttmp-ops.sql
SELECT s.id AS session_id,
  CAST(tc->>'timestamp' AS VARCHAR) AS ts,
  CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) AS cmd
FROM sessions_base s
CROSS JOIN UNNEST(tool_calls) AS t(tc)
WHERE s.id IN ('SESSION_ID_1', ...)
  AND CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'
  AND (
    LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%docmgr%'
    OR LOWER(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR)) LIKE '%ttmp%'
  )
ORDER BY s.id, CAST(tc->>'timestamp' AS VARCHAR);
```

Post-process with a Python classifier (scripts/16-classify-docmgr-ttmp.py) that breaks down:
- **docmgr subcommands**: `validate frontmatter`, `doc relate`, `changelog update`, `doc add`, `ticket create`, `task check`, etc.
- **ttmp file operations**: git (commits/status), edit (apply_patch/sed), read (cat/head), list (ls/find), create (mkdir/cp).

In our analysis, `validate frontmatter` was the single most common docmgr call (37 out of 110) — the agent repeatedly validates YAML after each edit, which is useful to know for optimizing the agent's workflow.

### Layer 6: Time Gap Analysis

**Purpose:** Identify when the human was away and for how long. This distinguishes "the session ran for 87 hours" from "the human actively worked for 13 hours across 4 days."

```sql
-- scripts/17-user-input-gaps.sql
WITH user_turns AS (
  SELECT s.id AS session_id, t.idx AS turn_idx,
    CAST(t.turn->>'timestamp' AS TIMESTAMP) AS ts,
    LEFT(CAST(t.turn->>'content' AS VARCHAR), 120) AS content
  FROM sessions_base s
  CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
  WHERE s.id IN ('SESSION_ID_1', ...)
    AND CAST(t.turn->>'role' AS VARCHAR) = 'user'
),
gaps AS (
  SELECT *, ts - LAG(ts) OVER (PARTITION BY session_id ORDER BY turn_idx) AS gap
  FROM user_turns
)
SELECT session_id, turn_idx, ts,
  ROUND(EXTRACT(EPOCH FROM gap) / 60, 1) AS gap_minutes, content
FROM gaps
WHERE EXTRACT(EPOCH FROM gap) > 1800  -- >30 min
ORDER BY session_id, ts;
```

This reveals the session's real structure: overnight gaps (10-20h), lunch breaks (1-2h), and "waiting for human action" gaps (merging PRs, creating tokens, resolving conflicts).

---

## Composing the Timeline Report

With all six layers extracted, the report is assembled by interleaving them chronologically. For each human block:

```
[2026-03-22T20:47] HUMAN: "Read the docs in geppetto and create
  a new ticket to bring wesen-os/ up to the new profiles..."
  ├── TICKET CREATED: APP-30-WESEN-OS-PINOCCHIO-PROFILE-BOOTSTRAP
  ├── DOC ADDED: "Investigation diary" (reference)
  ├── DOC ADDED: "Intern guide to migrating..." (design-doc)
  ├── DIARY WRITTEN: reference/01-investigation-diary.md
  │     (content from write call: "## Step 1: Audit current
  │      profile loading in wesen-os launcher...")
  ├── agent_turns=23, tool_calls=130
  └── duration: ~6h until next human input

[2026-03-23T02:36] HUMAN: "how does this all impact go-go-os-chat?"
  ├── agent_turns=2, tool_calls=4 (Q&A only, no artifacts)
  └── duration: 6min

  ... (5.8h gap — human asleep) ...

[2026-03-23T02:45] HUMAN: "ok, we can kill the profile CRUD.
  Design a step by step implementation plan."
  ├── COMMITS: docs(ticket): plan APP-30 read-only profile migration
  ├── agent_turns=7, tool_calls=13
  └── duration: 11min

[2026-03-23T02:56] HUMAN: "No backwards compatibility, no wrappers."
  ├── COMMITS:
  │     refactor(profilechat): resolve engine profiles with pinocchio runtime
  │     refactor(pinoweb): align inventory runtime wrappers
  │     feat(profilechat): support configured default profile selection
  │     test(wesen-os): add pinocchio profile fixture
  ├── DIARY WRITTEN: 4 updates to investigation diary
  ├── agent_turns=56, tool_calls=286
  └── duration: ~55min (then 42min gap)
```

This format makes several things immediately visible:
- **Where scope expanded:** ticket creation events.
- **Where real work happened:** blocks with commits.
- **Where time was spent waiting:** gaps between blocks.
- **Where the agent ran unsupervised:** large autopilot blocks.
- **What the agent's own narrative says:** diary write content, extracted from the session data itself, not from potentially-stale files on disk.

---

## Practical Tips

### Start narrow, expand as needed

Begin with the human-blocks query (Layer 1) for your target sessions. This gives you the structural overview in one query. Only pull in artifact timelines, commits, and diary content for the blocks that look interesting.

### Use `--output json` and Python for post-processing

The DuckDB table output truncates columns. Always pipe to JSON for analysis:

```bash
go-minitrace query duckdb --archive-glob '...' \
  --sql "..." --output json > /tmp/result.json
python3 analyze.py /tmp/result.json
```

### Diary content is in the write calls, not on disk

Files get moved, renamed, or deleted. The session data is the authoritative record. For `cat > file <<'EOF'` heredocs, the full content is in `tc->'input'->'arguments'->>'cmd'`. For `apply_patch`, the diff is in the command. For `write_stdin` calls (Codex's file-write tool), the content is in the input arguments.

### The `emitting_turn_index` field links tool calls back to turns

Each tool call has `emitting_turn_index` which tells you which turn in the `turns[]` array triggered it. This lets you correlate a commit or diary write back to the specific assistant turn that decided to do it, and from there back to the human block that prompted it.

### Filtering out noise

Many sessions contain safety-assessment subagent stubs (title starts with "The following is the Codex agent history whose request action you are assessing"). Filter these out early:

```sql
WHERE title NOT LIKE 'The following is the Codex agent history%'
```

Also filter by `CAST(metrics->>'turn_count' AS INT) > 10` to skip trivially short sessions.

### The master runner pattern

Write individual `.sql` queries and `.py` post-processors, then tie them together with a shell script (see scripts/20-run-all-wesen-os-analysis.sh). This makes the analysis reproducible and lets you re-run it against new session data without rebuilding queries.

---

## Reference: Script Inventory

| # | Script | Purpose |
|---|---|---|
| 01 | schema-probe.sql | Probe sessions_base schema |
| 02 | session-list.sql | Full session list with metrics |
| 03 | wesen-os-deploy-filter.sql | Broad filter: wesen-os/deploy/coolify in title or content |
| 04 | wesen-os-strict.sql | Strict filter: workdir or content references wesen-os |
| 05 | deep-read-session.sql | Extract user turns for a specific session (template) |
| 06 | wesen-os-tool-breakdown.sql | Tool call breakdown per session |
| 07 | assistant-summary-turns.sql | Assistant turn reader for session deep-reads |
| 08 | all-workdirs.sql | All working directories with session volumes |
| 09 | deploy-timeline.sql | Chronological deploy/wesen-os/k3s/hetzner timeline |
| 10 | human-blocks.sql | **Core:** human activity blocks (template) |
| 11 | session-size-buckets.sql | Session categorization by size |
| 12 | marathon-sessions.sql | 500+ turn sessions with active/wall metrics |
| 13 | wesen-os-active-vs-wall.sql | Active vs wall-clock for 3 wesen-os sessions |
| 14 | autopilot-blocks.py | **Core:** classify autopilot vs directed blocks |
| 15 | docmgr-and-ttmp-ops.sql | All docmgr and ttmp tool calls |
| 16 | classify-docmgr-ttmp.py | Classify docmgr subcommands and ttmp op types |
| 17 | user-input-gaps.sql | Time gaps >30min between user inputs |
| 18 | analyze-gaps.py | Per-session gap summary |
| 19 | ticket-creation-timeline.sql | Ticket directory creation events |
| 20 | run-all-wesen-os-analysis.sh | **Master runner:** executes all analysis scripts |
| 21 | doc-creation-timeline.sql | docmgr doc add / ticket create timeline |
| 22 | diary-writes.sql | All writes to diary files with timestamps |
| 23 | find-diary-files.sql | Extract distinct diary paths from tool calls |
| 24 | git-commits.sql | All successful git commits with messages |
