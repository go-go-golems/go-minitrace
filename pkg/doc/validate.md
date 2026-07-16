---
Title: Validate Command
Slug: validate-command
Short: Validate minitrace JSON, archives, manifests, source identity, and conversion receipts
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

The `validate` command supports file-level JSON/session checks and native archive-integrity checks.

## File validation

```bash
go-minitrace validate --path ./session.minitrace.json
go-minitrace validate --path ./output --recursive --output json
```

File mode reports `path`, `valid_json`, and `error`. It checks JSON syntax plus the session fields currently covered by `pkg/validate`, including annotations, events, and attachments.

## Archive validation

```bash
go-minitrace validate --path ./output --archive --output json
```

Archive mode locates the archive root even when `--path` points to a period directory or session file. It emits stable machine-readable findings with `code`, `severity`, `path`, `session_id`, and `detail`.

Checks include:

- archive filename versus payload ID;
- archive period versus `timing.started_at`;
- root and period manifest syntax and consistency;
- orphan archives and orphan manifest entries;
- recorded manifest file sizes;
- source SHA-256 when the recorded source path still exists;
- conversion receipt timestamps and completion state.

Select checks by repeating `--check`:

```bash
go-minitrace validate --path ./output --archive \
  --check archive --check manifest
```

Valid check names are `archive`, `manifest`, `source`, and `receipt`. With no `--check`, all checks run.

Error findings make the command return non-zero by default. Use `--fail-on-error=false` only when collecting diagnostics without gating a workflow.

## Rebuild manifests

Root and period manifests are written atomically. Rebuild them from archive files before validation with:

```bash
go-minitrace validate --path ./output --archive --rebuild-manifests
```

This is the supported replacement for directory-shape-sensitive manifest audit scripts.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | | File, directory, archive root, or path below an archive root |
| `--recursive` | `false` | Recursively scan JSON files in file mode |
| `--archive` | `false` | Enable archive integrity validation |
| `--check` | all | Repeatable archive check selector |
| `--rebuild-manifests` | `false` | Atomically rebuild manifests before validation |
| `--fail-on-error` | `true` | Return non-zero for error findings |

## Recommended conversion gate

```bash
go-minitrace convert codex \
  --source-list ./sources.txt \
  --output-dir ./output \
  --run-record ./output/conversion-run.json

go-minitrace validate --path ./output --archive --output json
```

## See also

- `go-minitrace help convert-commands`
- `go-minitrace help minitrace-schema`
