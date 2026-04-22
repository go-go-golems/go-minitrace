# Changelog

## 2026-04-22

- Created a new repo-local ticket to capture the narrowed follow-up from the external tool_calls investigation.
- Chose a documentation/troubleshooting ticket shape instead of a schema-bug ticket.
- Recorded the proposed scope, non-goals, evidence base, and acceptance criteria.
- Updated `pkg/doc/writing-duckdb-queries.md` with clearer JSON[] guidance, 1-based indexing notes, safer tool-input access patterns, and a `LIKE`/`->>` parser warning.
- Updated `pkg/doc/troubleshooting.md` with a dedicated DuckDB JSON[] sharp-edges section for `tool_calls`.
- Updated `pkg/doc/minitrace-schema.md` to clarify `input.file_path`, `input.arguments`, and adapter-dependent tool-name casing.
- Updated `pkg/doc/analysis-guide.md` and `pkg/doc/query.md` so adjacent examples match the verified current tool-call access model.
- Validated the ticket docs and confirmed `docmgr doctor` passes.
