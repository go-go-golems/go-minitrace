# go-minitrace xgoja jsverbs smoke

This example builds a small generated xgoja binary with embedded JavaScript verbs that call `require("minitrace")` directly.

It demonstrates the lightweight path for generated binaries that do not need the `go-minitrace` CLI command-provider layer. The example intentionally includes both an already-converted `.minitrace.json` archive and a raw Pi JSONL export so the verbs can tour several JavaScript APIs:

- `verbs inspect summary <file>` reads one `.minitrace.json` archive and prints session metadata.
- `verbs inspect tools <file>` reads the same archive and prints grouped tool-call counts.
- `verbs inspect preview <raw-jsonl>` uses `mt.importer().File(...).AutoDetect().Preview()` to inspect a raw export before saving it.
- `verbs inspect auto-convert <raw-jsonl>` uses `mt.sources()`, `mt.importPolicy()`, `mt.cache()`, `mt.limits()`, and `mt.db().Sources(...).Import(...).Cache(...).Limits(...)` to auto-convert and query a raw export.
- `verbs inspect save-converted <raw-jsonl>` uses `mt.importer().File(...).AutoDetect().Into(...).Save()` to write a converted archive directory.

Run:

```bash
make smoke
```

The smoke target builds `dist/minitrace-jsverbs`, runs all verbs against the checked-in sample data, checks the JSON output, and verifies that the save verb wrote `dist/converted/pi-jsverbs-tour/session.minitrace.json`.
