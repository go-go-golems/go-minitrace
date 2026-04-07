#!/bin/bash
# Convert the single Pi session to minitrace JSON
set -euo pipefail
go-minitrace convert pi \
  --source-session ../2026-04-06T22-07-00-864Z_f6498c9d-3c41-4850-8f9c-667eca2ee271.jsonl \
  --output-dir ./analysis
