---
Title: Validate Command
Slug: validate-command
Short: Check JSON syntax of minitrace files and directories
Topics:
- minitrace
- validate
Commands:
- validate
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `validate` command checks that minitrace JSON files are syntactically valid. It walks files or directories and reports whether each file parses as valid JSON.

## Current scope

This is a bootstrap validation implementation. It performs JSON syntax checks only — it does not validate against the minitrace schema (field presence, types, or value constraints). Full schema and semantic validation from the Python reference validator will be ported into `pkg/validate` in a future release.

For now, use `validate` to catch truncated files, encoding issues, or accidental corruption after conversion.

## Usage

Validate a single file:

```bash
go-minitrace validate --path ./output/active/2026-03/abc123.minitrace.json
```

Validate an entire directory recursively:

```bash
go-minitrace validate --path ./output --recursive
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | | File or directory to validate (required) |
| `--recursive` | `false` | Recursively scan directories for JSON files |

## Output

Each file produces one row with these fields:

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Absolute path to the validated file |
| `valid_json` | bool | Whether the file is syntactically valid JSON |
| `error` | string | Parse error message (empty if valid) |

The output goes through Glazed:

```bash
# Table output (default)
go-minitrace validate --path ./output --recursive

# JSON for scripting
go-minitrace validate --path ./output --recursive --output json

# Count valid files
go-minitrace validate --path ./output --recursive --output json \
  | jq '[.[] | select(.valid_json)] | length'
```

## Typical workflow

Run validation after conversion to check that all output files are well-formed:

```bash
go-minitrace convert claude-code --output-dir ./output
go-minitrace validate --path ./output --recursive
```

If validation reports errors, the affected files may have been truncated during conversion (e.g., due to disk space) or corrupted afterward. Re-run the conversion for those sessions.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| All files show `valid_json: true` but queries fail | JSON is syntactically valid but has unexpected schema | This is expected; `validate` currently checks syntax only |
| `valid_json: false` on some files | File was truncated or corrupted | Re-convert the source session |
| Validate is slow on large archives | Reads and parses every file | Validate a single period directory instead: `--path ./output/active/2026-03/` |

## See also

- `go-minitrace help convert-commands` — conversion that produces the files to validate
- `go-minitrace help minitrace-schema` — the schema these files should conform to
