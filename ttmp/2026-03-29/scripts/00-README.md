# Scripts

This folder records reproducible helper scripts for the `go-minitrace` porting work.

The goal is traceability: each script corresponds to a concrete exploration,
validation, or conversion step that was run during implementation.

Conventions:

- Files are numbered `XX-...` in rough chronological order.
- Shell scripts use `bash` and `set -euo pipefail`.
- Scripts prefer explicit absolute paths where the investigation depended on
  local user data such as `~/.pi` or `~/Downloads`.
- Some scripts are retroactive reconstructions of commands that were originally
  run ad hoc in the shell. They exist so the same step can be repeated later.

Current coverage:

- Pi discovery and real-session conversion
- claude.ai export ZIP validation and inspection
- claude.ai dry-run and filtered conversion
- ChatGPT smoke harness using a non-export JSON source to prove it is not the right input shape
- local download scanning for candidate ChatGPT and claude.ai exports
