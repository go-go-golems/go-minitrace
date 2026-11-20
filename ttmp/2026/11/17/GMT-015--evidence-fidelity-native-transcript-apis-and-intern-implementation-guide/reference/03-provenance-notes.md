---
Title: Additive archive ordering, token provenance, and corpus identity
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
Summary: Implementation and focused validation of M04/M08/M10 analytical provenance.
---

# Implementation

`tool_calls.source_ordinal` records the zero-based position in the normalized archive's `tool_calls` array. It is scoped to the session and is not a native JSONL line number, tool-result arrival coordinate, or causal replay boundary. The existing ToolRows chronological order is unchanged; consumers can explicitly order by `session_id, source_ordinal` to recover archive order. SQLite schema v6 changes the cache key, so prior v5 materializations are rebuilt rather than reused with missing columns. Directly supplied legacy SQLite databases are not migrated by this change.

`ToolRows()` directly exposes `source_ordinal`, `arguments_json` and `output_status`. The last is the archive model's explicit output status; `outcome_status` remains its derived outcome, which can be inferred from success or other evidence. Absent status is SQL null rather than a synthesized success. Nullable `success` is unchanged. Because the existing Go archive model represents absent/empty output status as the same empty string, those two input encodings remain indistinguishable; no new parser was introduced. Tool IDs remain the existing session-scoped primary key.

`TokenUsageRows()` adds `token_origin`, `token_unit`, `token_method`, and `reported_usage_fields`. Session/role/turn aggregates retain their numeric zero-fill behavior for compatibility but now distinguish missing accounting (`unknown`) from explicitly recorded zero (`recorded_usage`). `reported_usage_fields` counts non-null archive usage fields, summed in grouped rows; it is not a count of requests or complete records. Recorded usage describes the archive's reported fields, not a guarantee of independent provider measurement or absence of adapter estimation.

Tool rows use `estimated` / `estimated_tokens` / `round_utf8_bytes_div_4`; rows without output evidence say `unknown`. The estimate now consistently measures bytes using SQLite `LENGTH(CAST(... AS BLOB))`, aligning the fallback text measurement with `full_bytes`. This intentionally corrects the old mixed character/byte arithmetic for non-ASCII output. It is a heuristic, not tokenization or billing. `TranscriptRows()` carries equivalent per-row labels, and `TimelineRows()` labels its recorded usage. Consumers must not sum message/reasoning transcript views as disjoint provider usage: presentation can show the same accounting in more than one row.

`minitracedb.BuildCorpusManifest(paths)` explicitly inventories normalized archives without changing ingestion or silently selecting a winner. It reads and hashes the same bytes once per file, reports normalized and original identities, source format/fingerprint, absolute path, byte hash and size, and groups normalized session IDs. `duplicateOf` refers to the zero-based first entry with identical bytes. A session with multiple byte hashes is a conflict, marked on all its entries; metadata/whitespace differences therefore conservatively count as conflicts. This is not semantic equivalence and does not infer identity from filenames or strip child prefixes. Callers decide how to resolve conflicting variants before ingestion.

The JS helper is `mt.corpusManifest(paths)`. It returns `{entries, sessions, duplicateCount, conflictCount}` and throws a GoError on read/decode/missing-ID failures. Empty input returns empty arrays/counts. This standalone preflight is deliberately opt-in: legacy duplicate session ingestion behavior is unchanged. No recursive glob expansion or native-source classification is hidden inside it; `ExpandArchiveGlobs` remains available to Go callers and callers can supply already-selected paths.

# Validation diary

Inspected the SQLite schema/materializer, cache-key version, typed query recipes and archive model before editing. No repository AGENTS.md or CLAUDE.md exists at this worktree root. Changed only owned schema/materializer/typed recipe files and added dedicated manifest/provenance files.

The first focused run exposed two test-fixture mistakes: `Query` returns rows rather than a result envelope (`QueryResult` is the envelope method), and the actual derived success enum is `succeeded`, not `success`. Corrected the tests. The next `go test ./pkg/minitracedb ./pkg/minitracejs` passed. Tests exercise source order despite conflicting ID order, direct arguments, unknown success/original status, derived outcome distinction, schema cache-key invalidation, identical copies plus divergent same-ID variants, missing identity, all token grouping modes, known zero vs missing usage, and UTF-8 estimate units. A Goja contract test additionally exercises manifest JSON field casing, duplicate index zero, and thrown read errors.

No source transcript contents or credentials are copied into test fixtures. No changes to query.go, db_builder.go, or module.go are owned here; module wiring is coordinated with the parent agent. Native request reconstruction, automatic corpus conflict enforcement, explicit result-arrival chronology, per-provider tokenizer estimates, and source version migration for arbitrary old SQLite files remain outside this additive change.
