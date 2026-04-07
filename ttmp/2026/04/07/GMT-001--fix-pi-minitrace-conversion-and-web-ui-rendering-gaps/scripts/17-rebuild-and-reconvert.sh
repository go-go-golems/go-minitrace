#!/bin/bash
# After fixing the converter (skip tool result turns), rebuild and re-convert
set -euo pipefail

cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace

# Rebuild
go build ./cmd/go-minitrace

# Run tests
go test ./pkg/adapters/pi/ -v

# Install
go install ./cmd/go-minitrace

# Re-convert (from the session output dir)
cd /home/manuel/.pi/agent/sessions/--home-manuel-code-wesen-2026-04-06--paper-pro-pen-prob--/output
rm -f ./analysis/active/2026-04/f6498c9d-3c41-4850-8f9c-667eca2ee271.minitrace.json
go-minitrace convert pi \
  --source-session ../2026-04-06T22-07-00-864Z_f6498c9d-3c41-4850-8f9c-667eca2ee271.jsonl \
  --output-dir ./analysis

# Re-sync annotations
go-minitrace annotate sync --output-dir ./analysis --session f6498c9d-3c41-4850-8f9c-667eca2ee271

# Validate
go-minitrace validate --path ./analysis --recursive
