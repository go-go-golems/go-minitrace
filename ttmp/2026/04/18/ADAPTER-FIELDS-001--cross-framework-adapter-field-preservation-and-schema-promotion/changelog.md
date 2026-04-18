# Changelog

## 2026-04-18

- Initial workspace created

## 2026-04-18

Created follow-up ticket `ADAPTER-FIELDS-001` to separate cross-framework field preservation/schema work from the Pi `isError` bug ticket. Added a source-backed field matrix covering Pi, Codex, and Claude Code, plus a ticket-local scanner script and scan output so the findings can be re-run against representative raw transcripts. Recorded in commit `1da7a17065d19c283e88f4154fa6db1fac5bdb1f` (`Add adapter field analysis ticket`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/analysis/01-cross-framework-field-matrix.md - Source-backed field matrix and promotion recommendations
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/scripts/01-scan-field-representations.py - Reproducible raw-field scanner
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/18/ADAPTER-FIELDS-001--cross-framework-adapter-field-preservation-and-schema-promotion/sources/01-field-scan.txt - Captured scanner output for the sampled raw transcripts

## 2026-04-18

Implemented the first-wave schema promotions from the new ticket: `tool_calls[].output.exit_code` and `tool_calls[].input.justification`. Updated the minitrace schema/builders, populated both fields in the Codex adapter, added regression coverage, and refreshed the schema/adapter docs. Validation included `gofmt -w ...`, `go test ./pkg/adapters/codex ./pkg/minitrace -count=1`, and `go test ./... -count=1`. Recorded in commit `ffbd6a1d62512a01250d9a780b5dcc6be2b73f3a` (`Promote Codex exit code and justification`).

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/schema.go - Added first-class `exit_code` and `justification` fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/builders.go - Initialized the new tool input/output fields in the shared builder
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert.go - Populated `exit_code` and `justification` during Codex conversion
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert_test.go - Added assertions covering the promoted fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/minitrace-schema.md - Documented the new schema fields
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/doc/adapter-reference.md - Documented Codex preservation of command metadata and justification
