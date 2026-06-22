# go-minitrace xgoja jsverbs smoke

This example builds a small generated xgoja binary with embedded JavaScript verbs that call `require("minitrace")` directly.

It demonstrates the lightweight path for generated binaries that do not need the `go-minitrace` CLI command-provider layer:

- `verbs inspect summary <file>` reads one `.minitrace.json` archive and prints session metadata.
- `verbs inspect tools <file>` reads the same archive and prints grouped tool-call counts.

Run:

```bash
make smoke
```

The smoke target builds `dist/minitrace-jsverbs`, runs both verbs against `data/session-a.minitrace.json`, and checks the JSON output.
