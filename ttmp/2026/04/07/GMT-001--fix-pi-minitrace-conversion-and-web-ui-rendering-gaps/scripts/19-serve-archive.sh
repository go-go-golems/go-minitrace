#!/bin/bash
# Serve the minitrace archive with the web UI
# Run from the session output directory

SESSION_DIR=/home/manuel/.pi/agent/sessions/--home-manuel-code-wesen-2026-04-06--paper-pro-pen-prob--/output

go-minitrace serve \
  --archive-glob "$SESSION_DIR/analysis/active/*/*.minitrace.json" \
  --query-dir "$SESSION_DIR/queries" \
  --port 8080
