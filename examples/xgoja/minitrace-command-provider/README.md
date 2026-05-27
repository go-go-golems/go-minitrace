# go-minitrace xgoja command-provider smoke

This example builds a generated xgoja binary that mounts the `go-minitrace.queries`
command provider as `traces`.

The local query repository contains a JavaScript query command that uses:

- `require("minitrace")` to query loaded `.minitrace.json` archives;
- `require("fs")` to create `dist/report/minitrace-summary.md`.

Run:

```bash
make smoke
```
