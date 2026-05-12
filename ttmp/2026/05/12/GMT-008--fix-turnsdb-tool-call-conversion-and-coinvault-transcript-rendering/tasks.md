# Tasks

## TODO

- [x] Port GMINI-0002 root-cause design into go-minitrace ticket docs
- [ ] Add failing turnsdb regression tests for metadata churn, duplicate tool calls, ToolCallsInTurn linking, and whitespace text payload normalization
- [ ] Make turnsdb LCS block fingerprint stable across volatile tool-call metadata
- [ ] Deduplicate and merge top-level tool calls by ID without downgrading successful results to pending failures
- [ ] Populate turns[].ToolCallsInTurn while converting tool calls
- [ ] Unwrap payload["text"] even when empty or whitespace-only
- [ ] Evaluate and document ordered text/tool/text interleaving behavior after the converter fix
- [ ] Run gofmt and targeted go test ./pkg/adapters/turnsdb
- [ ] Regenerate or smoke-check Coinvault archives/API if local fixture data is available
