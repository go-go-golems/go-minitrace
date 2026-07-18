---
Title: Activity-Based Session Discovery Design and Implementation Guide
Ticket: GMT-014
Status: active
Topics:
    - architecture
    - go-minitrace
    - minitrace
    - transcript-analysis
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/discover/filters.go
      Note: Defines current --since semantics and shared discovery filter boundary
    - Path: repo://pkg/adapters/claudecode/convert.go
      Note: Evidence for Claude Code timestamp semantics
    - Path: repo://pkg/adapters/codex/convert.go
      Note: Evidence for Codex timestamp semantics
    - Path: repo://pkg/adapters/head.go
      Note: Defines bounded header scan contrasted with opt-in full activity scan
    - Path: repo://pkg/adapters/pi/convert.go
      Note: Evidence for Pi timestamp semantics
    - Path: repo://pkg/adapters/types.go
      Note: Defines SessionLocator metadata proposed to carry last activity
ExternalSources: []
Summary: Intern-oriented design and implementation guide for exact activity-based discovery of long-lived Pi, Codex, and Claude Code sessions.
LastUpdated: 2026-07-18T18:30:00-04:00
WhatFor: Explain and guide implementation of the --active-since discovery filter.
WhenToUse: Use when implementing, reviewing, or troubleshooting activity-window session discovery.
---


# Activity-Based Session Discovery Design and Implementation Guide

## Executive Summary

`go-minitrace discover` is the inventory stage of the transcript-analysis pipeline. It identifies native transcripts before conversion writes `.minitrace.json` archives and before the normalized SQLite query engine answers higher-level questions. Today, discovery can filter with `--since`, but that flag intentionally means **the session began on or after a timestamp**. It does not mean **the session had recorded activity on or after a timestamp**.

That distinction produced a real false negative. A Pi session titled **“Real-Provider RAG and Geppetto Reranking”** began on 2026-07-15 and still had activity on 2026-07-18. `discover pi --since 2026-07-18` omitted it because the initial session timestamp predated the boundary. The feature in this ticket adds a distinct, opt-in `--active-since` filter for Pi, Codex, and Claude Code discovery. It retains `--since` unchanged and makes activity scanning explicit because it is materially more expensive than header discovery.

The implementation should stream native JSONL records, collect the maximum valid framework activity timestamp, place that value in `adapters.SessionLocator.LastActivityAt`, and apply it as an additional filter. The scan uses constant memory and reuses the timestamp parsing contract already used by conversion (`minitrace.ParseTimestamp`), but reads the entire candidate transcript. The command output will expose `last_activity_at`, making the decision inspectable instead of a hidden filter.

## 1. Problem Statement and Scope

### 1.1 Definitions

A **session start timestamp** is the timestamp in an early native session/header record. A **last activity timestamp** is the maximum valid timestamp associated with a native record that the adapter regards as transcript activity. A session is **active since `T`** when its last activity timestamp is greater than or equal to `T`. This definition answers the common recovery question, “which session still did work after the boundary?” It does not claim that the session was continuously active during that interval.

The intended command surface is:

```text
go-minitrace discover pi --active-since 2026-07-18
go-minitrace discover codex --cwd-contains rag-eval --active-since 2026-07-18T00:00:00Z
go-minitrace discover claude-code --since 2026-07-01 --active-since 2026-07-18
```

The two filters are conjunctive:

- `--since T`: the session began at or after `T`.
- `--active-since T`: the session has a last activity timestamp at or after `T`.
- both: both claims must be true.

### 1.2 In scope

- Pi JSONL v3 discovery.
- Codex persisted and exec JSONL discovery.
- Claude Code JSONL v2 discovery.
- Shared CLI parsing and filtering behavior.
- Discovery output containing `last_activity_at`.
- Tests for start-time/activity-time divergence, timestamp variants, invalid data, and combined filters.
- Documentation, ticket bookkeeping, and operator guidance.

### 1.3 Out of scope

- Changing the semantics of `--since`.
- Rewriting native transcript formats.
- Converting every native transcript before discovery.
- Persistent scan caching. Caching is a possible future optimization after exact behavior is tested and accepted.
- Claude Code legacy `dir-v1` activity inference without a native timestamp-bearing JSONL source.
- Guessing activity from filesystem mtime. Mtime is useful only as a future conservative optimization, never as timestamp evidence.

## 2. System Orientation for a New Intern

### 2.1 The reduction pipeline

The repository turns bulky, framework-specific transcript stores into progressively more useful representations:

```text
native stores              conversion archives                 query engine
─────────────              ───────────────────                 ────────────
Pi JSONL        ─┐
Codex JSONL      ├─> discover ─> source list ─> convert ─> .minitrace.json ─> SQLite ─> SQL / JS / web
Claude JSONL    ─┘
                       ^
                       │ this ticket: select by actual activity, not only start time
```

Discovery must be cheap enough to inventory a large home directory. Conversion may do expensive normalization because it processes a deliberately selected shortlist. This ticket adds a more expensive discovery mode, but keeps it opt-in so ordinary `discover` calls retain their bounded-header behavior.

### 2.2 Command layer

The executable begins at `cmd/go-minitrace/main.go`. The discover command group is rooted at `cmd/go-minitrace/cmds/discover/root.go`. Each framework command is a Glazed command wired into Cobra through `cmd/go-minitrace/cmds/common`:

- `cmd/go-minitrace/cmds/discover/pi.go`
- `cmd/go-minitrace/cmds/discover/codex.go`
- `cmd/go-minitrace/cmds/discover/claude_code.go`
- `cmd/go-minitrace/cmds/discover/filters.go`

Each command follows the same sequence: decode Glazed fields into a settings struct, parse date flags, ask an adapter for `[]adapters.SessionLocator`, filter the locators, and emit Glazed rows. The command layer owns user-facing flag semantics and output columns. It should not understand framework record layouts.

### 2.3 Adapter layer

`pkg/adapters/types.go` defines `SessionLocator`, the small cross-framework discovery record. Before this ticket it carries the native source path, framework format hint, optional working directory, and `StartedAt` header timestamp.

The adapters own native-format knowledge:

- `pkg/adapters/pi/discover.go` recognizes Pi session files and reads the leading `type: "session"` record.
- `pkg/adapters/codex/discover.go` recognizes Codex stores, persisted/exec formats, and `session_meta` headers.
- `pkg/adapters/claudecode/discover.go` finds JSONL v2 primary transcripts and legacy directory sessions; it skips subagents in top-level discovery.
- `pkg/adapters/head.go` provides `ScanJSONLHead`, bounded to 50 lines and 256 KiB. It deliberately avoids full transcript reads.

Conversion code is useful evidence for timestamp semantics. `pkg/adapters/pi/convert.go`, `pkg/adapters/codex/convert.go`, and `pkg/adapters/claudecode/convert.go` parse native records into normalized turns, tool calls, events, and timing. They already use `minitrace.ParseTimestamp` and `minitrace.ComputeTiming`; activity discovery should use the same timestamp parser to avoid two competing notions of valid time.

### 2.4 Normalized model and queries

Conversion produces a `minitrace.Session` with timing values. `pkg/minitracedb/materialize.go` materializes archives into SQLite `sessions`, `turns`, and `tool_calls` tables. That model can answer activity questions **after conversion**, but cannot efficiently select unknown native sources. `--active-since` closes the native-inventory gap; it does not replace the query engine.

## 3. Observed Current Behavior and Gap

### 3.1 Evidence-backed path

`filters.go` defines one shared `--since` field and documents it as “Only keep sessions started at or after this time.” `parseSince` accepts RFC3339 or `YYYY-MM-DD` (UTC midnight). `keepLocator` parses only `locator.StartedAt`; when it is before the boundary, the locator is rejected.

Each framework command calls its adapter first and then calls `keepLocator`. The Pi adapter reads only a leading `session` record. Codex reads a leading `session_meta` record. Claude Code reads until it finds a record with `cwd`, because the first record can be a `file-history-snapshot`. None exposes a final transcript timestamp.

### 3.2 Why the gap matters

Long-lived sessions are ordinary: sessions can be resumed across days, and their original start time does not communicate their most recent activity. A query such as “what was active today?” needs an event-time predicate. Using only a filename date, cwd, title, or start time is weak candidate evidence and causes the exact false negative observed in the motivating incident.

## 4. Proposed Architecture and API

### 4.1 Public CLI contract

Add this shared discovery flag for the JSONL-backed framework commands:

```text
--active-since <RFC3339|YYYY-MM-DD>
    Only keep sessions whose latest valid transcript activity timestamp is at
    or after this time. This scans candidate native transcripts and may be
    slower than --since. Sessions without a valid activity timestamp are excluded.
```

Add `last_activity_at` to rows emitted by Pi, Codex, and Claude Code discovery. An empty value means that activity was not scanned or no valid activity timestamp was available. The value lets JSON, YAML, and table consumers audit why a locator passed.

### 4.2 Locator contract

Extend `adapters.SessionLocator`:

```go
type SessionLocator struct {
    ID             string
    FormatHint     string
    SourcePath     string
    Cwd            string
    StartedAt      string
    LastActivityAt string
    Identity       *SourceIdentity
}
```

`LastActivityAt` is discovery metadata, not archive provenance. It must not alter source identity or conversion behavior.

### 4.3 Adapter API sketch

Expose a small framework-local helper instead of embedding format switches in the CLI:

```go
// LastActivityAt streams one JSONL transcript and returns the maximum valid
// activity timestamp in canonical RFC3339 text. Empty means no valid timestamp.
func LastActivityAt(path string) (string, error)
```

A shared primitive in `pkg/adapters/head.go` (renamed or placed in a new file because it is not a head scan) should open a JSONL file, scan with the same 10 MiB token limit used by converters, decode records one at a time, and ask an adapter callback for timestamp candidates. It should retain only the maximum parsed `time.Time`, so memory usage is O(1).

### 4.4 Timestamp extraction policy

Use records that the existing converter treats as timing-bearing. The implementation must keep the policy narrow enough to avoid arbitrary embedded strings and broad enough to capture actual tool, user, assistant, lifecycle, and session events.

| Framework | Candidate extraction | Evidence |
| --- | --- | --- |
| Pi | record `timestamp`; for `type: message`, fall back to `message.timestamp` | `pkg/adapters/pi/convert.go` collects exactly those candidates into `allTimestamps`. |
| Codex | top-level record `timestamp`; for session header compatibility, payload timestamp fallback when top-level is absent | `pkg/adapters/codex/discover.go` already uses this fallback for the start timestamp; converters parse timing per format. |
| Claude Code v2 | top-level record `timestamp`, ignoring the same discarded snapshot/progress records used by conversion | `pkg/adapters/claudecode/convert.go` filters `file-history-snapshot`, `last-prompt`, and `progress` before collecting timestamps. |

For missing, malformed, or unsupported source data, return an empty activity timestamp and let `--active-since` exclude the source. Do not fabricate a timestamp from mtime.

## 5. Key Flow and Pseudocode

```text
+-------------------+       +------------------------+       +-------------------+
| native filesystem | ----> | bounded header discover | ----> | apply cheap CWD / |
|   many JSONL      |       | (current fast path)     |       | start filters     |
+-------------------+       +------------------------+       +---------+---------+
                                                                         |
                                           --active-since absent         | present
                                              return rows                v
                                                               +-------------------+
                                                               | stream candidate  |
                                                               | JSONL, retain max |
                                                               | valid timestamp   |
                                                               +---------+---------+
                                                                         |
                                                                         v
                                                               +-------------------+
                                                               | last >= boundary? |
                                                               | emit / reject     |
                                                               +-------------------+
```

```go
func filterForActivity(locator SessionLocator, activeSince *time.Time, scan ScanFn) (SessionLocator, bool, error) {
    if activeSince == nil {
        return locator, true, nil
    }
    // A valid start at/after the boundary is already proof of activity.
    if started, ok := ParseTimestamp(locator.StartedAt); ok && !started.Before(*activeSince) {
        locator.LastActivityAt = locator.StartedAt
        return locator, true, nil
    }

    latest, err := scan(locator.SourcePath)
    if err != nil {
        return locator, false, err
    }
    locator.LastActivityAt = latest
    last, ok := ParseTimestamp(latest)
    return locator, ok && !last.Before(*activeSince), nil
}
```

The actual command may choose to scan every CWD-matching source for straightforward control flow. The optimization above is valid only because a start timestamp at or after the boundary is itself a timestamped session record and therefore satisfies the activity predicate.

## 6. Decisions

### Decision: Add `--active-since`; do not redefine `--since`

- **Context:** Existing users can reasonably depend on `--since` selecting sessions created in a period.
- **Options considered:** Redefine `--since`; add an activity flag; require conversion before querying.
- **Decision:** Add `--active-since` and retain `--since` start-time semantics.
- **Rationale:** The two questions are both useful and non-equivalent. A separate name prevents silent semantic breakage.
- **Consequences:** Documentation and help must distinguish the flags; callers can combine them.
- **Status:** accepted.

### Decision: Exact streaming scan, opt-in

- **Context:** Header discovery reads at most 50 lines / 256 KiB, whereas activity is generally near the end of a transcript.
- **Options considered:** file mtime; tail-only JSONL read; full conversion; exact streaming scan.
- **Decision:** Stream full candidate JSONL only when `--active-since` is requested.
- **Rationale:** It is correct for append-only transcript files, uses constant memory, and avoids archive writes. Tail-only data can end in snapshots or truncated records; mtime is not event evidence.
- **Consequences:** Activity queries cost O(total candidate bytes); ordinary discovery remains fast. A fingerprint/size/mtime cache can be added later without changing semantics.
- **Status:** accepted.

### Decision: Adapter-owned timestamp extraction

- **Context:** Framework timestamp fields and ignored records differ.
- **Options considered:** generic recursive JSON search; command-layer switches; adapter-local functions backed by a shared scanner.
- **Decision:** Keep record interpretation in adapters and share only JSONL streaming mechanics.
- **Rationale:** Conversion already demonstrates adapter-specific semantics. Generic recursive searching risks treating quoted timestamps as activity.
- **Consequences:** Three small adapter tests are required, but no command package needs native-format conditionals.
- **Status:** accepted.

## 7. Implementation Plan

### Phase 1: Shared contracts and exact scanner

1. Extend `SessionLocator` with `LastActivityAt`.
2. Add a full-file JSONL timestamp scanner under `pkg/adapters/`.
3. Preserve malformed-line tolerance used by discovery/conversion, but return real I/O/scanner errors.
4. Unit-test maximum-time selection, invalid records, and timestamps in non-monotonic order.

### Phase 2: Framework extractors

1. Implement Pi extraction with message timestamp fallback.
2. Implement Codex extraction with top-level/payload fallback.
3. Implement Claude Code JSONL v2 extraction with converter-consistent ignored record types.
4. Define legacy Claude directory behavior as “no activity timestamp” until a native timing source exists.

### Phase 3: Command wiring

1. Add `ActiveSince` to Pi, Codex, and Claude settings structs.
2. Parse it using the existing RFC3339/date parser (or rename that parser to a neutral timestamp-boundary name).
3. Apply cheap cwd/start logic before invoking scans where possible.
4. Emit `last_activity_at` in each row.
5. Update command long help with performance and semantics.

### Phase 4: Tests and real-source smoke checks

1. Add shared filter tests for old-start/new-activity inclusion.
2. Add adapter fixtures with activity after the header date.
3. Run focused packages, then `go test ./...` and `go build ./...`.
4. Use the known long-lived RAG/Geppetto Pi transcript as a manual smoke input; verify it appears under `--active-since 2026-07-18` but not under `--since 2026-07-18`.

## 8. Test Strategy

### Unit tests

- Timestamp parser accepts RFC3339 and date boundaries exactly as before.
- `--active-since` includes a session that began before but has later activity.
- It excludes a session with only older activity.
- Both time filters are ANDed.
- Empty/malformed activity timestamp does not accidentally pass.
- The maximum timestamp is selected even when native records are out of order.
- Framework-specific fallback fields match converter behavior.

### Command-level tests

Construct temporary source directories using existing adapter fixture patterns and execute Glazed command logic or `go run ./cmd/go-minitrace discover ... --output json`. Assert columns and filtering, not only helper functions.

### Regression evidence

The motivating real source is the Pi session `019f66db-bc32-79cc-8ba9-da2b6286e24b`, title “Real-Provider RAG and Geppetto Reranking.” It starts on 2026-07-15 and has last activity on 2026-07-18. It is a smoke test only; committed tests must use small synthetic fixtures with no private transcript content.

## 9. Risks, Alternatives, and Open Questions

### Performance

Exact scanning is O(total bytes) for sources surviving cheap filters. That is intentional but can be expensive on years of session history. A later cache keyed by normalized path, size, and mtime can store `last_activity_at`; cache invalidation must force a rescan when either size or mtime changes.

### Timestamp semantics

“Last timestamp” proves a record exists after a boundary; it does not prove continuous execution. This is the correct and useful definition for session recovery. Documentation must avoid saying a session was continuously active.

### Error policy

A source unreadable during an explicitly exact scan should surface an error rather than silently claim a complete result. Malformed individual JSON lines should be skipped, matching current tolerant adapters. Reviewers should confirm this policy is uniform across the three extractors.

### Why not use conversion?

Conversion has the most complete semantic knowledge but is too heavy for initial inventory: it reads whole files, builds full turn/tool objects, writes archives, and may perform identity handling. The new scan extracts only timestamp evidence.

## 10. References

### Primary implementation files

- `cmd/go-minitrace/cmds/discover/filters.go` — shared flag definitions, boundary parsing, and locator filtering.
- `cmd/go-minitrace/cmds/discover/pi.go` — Pi command settings, execution, and output rows.
- `cmd/go-minitrace/cmds/discover/codex.go` — Codex command equivalent.
- `cmd/go-minitrace/cmds/discover/claude_code.go` — Claude Code command equivalent.
- `pkg/adapters/types.go` — cross-framework locator contract.
- `pkg/adapters/head.go` — bounded header scanning helper and its limits.
- `pkg/adapters/pi/discover.go` — Pi native header discovery.
- `pkg/adapters/codex/discover.go` — Codex header discovery and format detection.
- `pkg/adapters/claudecode/discover.go` — Claude primary-session and legacy directory discovery.
- `pkg/adapters/pi/convert.go` — Pi timing collection semantics.
- `pkg/adapters/codex/convert.go` — Codex timing collection semantics.
- `pkg/adapters/claudecode/convert.go` — Claude ignored-record and timing semantics.
- `cmd/go-minitrace/cmds/discover/filters_test.go` — existing start-time filter tests to extend.

### Operator references

- `README.md` — overview of native discovery, conversion, and query pipeline.
- `skills/go-minitrace-transcript-analysis/SKILL.md` — operator guidance; it must state that `--since` is start-time filtering and recommend `--active-since` for activity-window recovery.
