# Tasks

## TODO

- [x] Port GMINI-0002 root-cause design into go-minitrace ticket docs
- [x] Add failing turnsdb regression tests for metadata churn, duplicate tool calls, ToolCallsInTurn linking, and whitespace text payload normalization
- [x] Make turnsdb LCS block fingerprint stable across volatile tool-call metadata
- [x] Deduplicate and merge top-level tool calls by ID without downgrading successful results to pending failures
- [x] Populate turns[].ToolCallsInTurn while converting tool calls
- [x] Unwrap payload["text"] even when empty or whitespace-only
- [x] Evaluate and document ordered text/tool/text interleaving behavior after the converter fix
- [x] Run gofmt and targeted go test ./pkg/adapters/turnsdb
- [x] Regenerate or smoke-check Coinvault archives/API if local fixture data is available
- [x] Define a turnsdb semantic block identity helper that prefers block_id and tool payload IDs over metadata-sensitive content hashes
- [x] Refactor turnsdb LCS/delta matching to compare semantic block keys while tracking content/version updates separately
- [x] Add regression tests for metadata-only block version changes on non-tool blocks and tool blocks
- [x] Add regression tests for cumulative snapshots where framework/control blocks disappear but stable block IDs should prevent duplicate transcript output
- [x] Refactor turnsdb conversion to process ordered delta blocks in one pass instead of separate text and tool phases
- [x] Add an ordered text/tool/text fixture and document whether turns plus ToolCallsInTurn is sufficient
- [ ] Design a minitrace ordered transcript-event model/API if exact interleaving cannot be represented by current turns
- [x] Evaluate optional Pinocchio semantic_hash or stable_block_key export only after go-minitrace consumes existing block_id and ordinal fields
- [x] Audit real turnsdb archives for any missing block_id or missing tool payload id failures after fail-fast semantic identity lands
- [x] Expose generic tool arguments clearly in the transcript UI for non-bash/non-file tools
