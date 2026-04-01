# Tasks

- [x] Review the Python `turns.db` adapter and the alternate ChatGPT transcript design docs
- [x] Add Go support for the alternate per-conversation ChatGPT JSON transcript format
- [x] Extract tool calls from alternate ChatGPT transcript exchanges
- [x] Add a `go-minitrace convert chatgpt-json` command
- [x] Port the `turns.db` canonical snapshot + LCS delta strategy to Go
- [x] Add a `go-minitrace convert turnsdb` command
- [x] Add focused adapter tests for both new paths
- [x] Run real-data smoke conversions against `/tmp/chatgpt-exports` and `/tmp/turns.db`
- [x] Inspect the generated session output with `jq`
- [x] Commit the alternate ChatGPT transcript checkpoint
- [x] Commit the turns.db checkpoint
