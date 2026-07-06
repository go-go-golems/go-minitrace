---
Title: Output Formats and Pipelines
Slug: output-formats-and-pipelines
Short: Using Glazed output options and piping go-minitrace results to other tools
Topics:
- minitrace
- glazed
Flags:
- output
- fields
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---

All go-minitrace commands produce structured output through Glazed, which means you can choose the output format and select specific fields. This page shows how to use these options and how to pipe results to external tools.

## Output formats

Every command that produces tabular output supports the `--output` flag:

| Format | Flag | Best for |
|--------|------|----------|
| Table | `--output table` (default) | Terminal viewing |
| JSON | `--output json` | Piping to jq or scripts |
| YAML | `--output yaml` | Human-readable structured output |
| CSV | `--output csv` | Spreadsheets, data import |
| Markdown | `--output markdown` | Documentation, pasting into reports |

Examples:

```bash
# JSON for scripting
go-minitrace query run --archive-glob '...' --preset session-list --output json

# CSV for Excel/Sheets
go-minitrace query run --archive-glob '...' --preset framework-summary --output csv

# YAML for reading
go-minitrace discover claude-code --output yaml
```

## Field selection

Use `--fields` to include only specific columns in the output:

```bash
go-minitrace query run \
  --archive-glob '...' \
  --preset session-list \
  --fields id,framework,turns,tools
```

This reduces output width and is especially useful with table and CSV formats.

## Piping to jq

JSON output works well with jq for filtering and transforming:

```bash
# Count sessions
go-minitrace discover claude-code --output json | jq length

# Filter to sessions with many tool calls
go-minitrace query run --archive-glob '...' --preset session-list --output json \
  | jq '[.[] | select(.tools > 50)]'

# Extract just the models used
go-minitrace query run --archive-glob '...' --preset session-list --output json \
  | jq '[.[].model] | unique'

# Group by framework
go-minitrace query run --archive-glob '...' --preset session-list --output json \
  | jq 'group_by(.framework) | .[] | {framework: .[0].framework, count: length}'
```

## Piping to Python

For more complex analysis, pipe JSON to a Python script:

```bash
go-minitrace query run --archive-glob '...' --preset session-list --output json \
  | python3 -c "
import json, sys
sessions = json.load(sys.stdin)
total = len(sessions)
avg_turns = sum(s.get('turns', 0) for s in sessions) / max(total, 1)
print(f'{total} sessions, avg {avg_turns:.1f} turns')
"
```

## CSV to spreadsheet workflows

Export data for spreadsheet analysis:

```bash
# Full session list
go-minitrace query run --archive-glob '...' --preset session-list --output csv > sessions.csv

# Framework summary
go-minitrace query run --archive-glob '...' --preset framework-summary --output csv > summary.csv
```

Open the CSV directly in your spreadsheet application, or import it for charts and pivot tables.

## Scripting the full pipeline

Combine discover, convert, and query in a shell script:

```bash
#!/bin/bash
set -euo pipefail

OUTDIR="./analysis-$(date +%Y%m%d)"

echo "Discovering..."
COUNT=$(go-minitrace discover claude-code --output json | jq length)
echo "Found $COUNT sessions"

echo "Converting..."
go-minitrace convert claude-code --output-dir "$OUTDIR"

echo "Summary:"
go-minitrace query run \
  --archive-glob "$OUTDIR/active/*/*.minitrace.json" \
  --preset framework-summary

echo "Exporting..."
go-minitrace query run \
  --archive-glob "$OUTDIR/active/*/*.minitrace.json" \
  --preset session-list --output csv > "$OUTDIR/sessions.csv"

echo "Done. CSV at $OUTDIR/sessions.csv"
```

## The manifest file

After conversion, the root `manifest.json` contains aggregate statistics:

```bash
cat ./output/manifest.json | jq '.statistics'
```

Output includes:
- `total_sessions` — total number of converted sessions
- `by_profile` — breakdown by profile (organic, synthetic)
- `by_quality` — breakdown by quality tier (A, B, C)
- `by_classification` — breakdown by classification
- `date_range` — earliest and latest session timestamps

Period manifests under `active/YYYY-MM/manifest.json` list every session in that month with key metadata.

## See also

- `go-minitrace help query-commands` — query flags and modes
- `go-minitrace help getting-started` — tutorial using these output features
