#!/usr/bin/env bash
# full-workflow-demo.sh
# Demonstrates the complete go-minitrace workflow: discover → convert → query → validate
# Used during documentation investigation to understand the end-to-end flow.

set -euo pipefail
OUT_DIR="/tmp/minitrace-docs-test"

echo "=== Step 1: Discover Claude Code sessions ==="
go run ./cmd/go-minitrace discover claude-code --source-dir ~/.claude/projects --output json \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Found {len(d)} Claude Code sessions')"

echo ""
echo "=== Step 2: Discover Codex sessions ==="
go run ./cmd/go-minitrace discover codex --source-dir ~/.codex --output json \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Found {len(d)} Codex sessions')"

echo ""
echo "=== Step 3: Discover Pi sessions ==="
go run ./cmd/go-minitrace discover pi --source-dir ~/.pi/agent/sessions --output json \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Found {len(d)} Pi sessions')"

echo ""
echo "=== Step 4: Convert Claude Code ==="
go run ./cmd/go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir "$OUT_DIR" 2>&1 | tail -3

echo ""
echo "=== Step 5: Convert turnsdb ==="
go run ./cmd/go-minitrace convert turnsdb --source /tmp/turns.db --output-dir "$OUT_DIR" 2>&1 | tail -3

echo ""
echo "=== Step 6: Convert Pi ==="
go run ./cmd/go-minitrace convert pi --source-dir ~/.pi/agent/sessions --output-dir "$OUT_DIR" 2>&1 | tail -3

echo ""
echo "=== Step 7: Query — session-list (first 5) ==="
go run ./cmd/go-minitrace query duckdb \
  --archive-glob "$OUT_DIR/active/*/*.minitrace.json" \
  --preset session-list --output json \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Total: {len(d)} sessions'); [print(json.dumps(r)) for r in d[:5]]"

echo ""
echo "=== Step 8: Query — framework-summary ==="
go run ./cmd/go-minitrace query duckdb \
  --archive-glob "$OUT_DIR/active/*/*.minitrace.json" \
  --preset framework-summary --output json

echo ""
echo "=== Step 9: Query — custom SQL ==="
go run ./cmd/go-minitrace query duckdb \
  --archive-glob "$OUT_DIR/active/*/*.minitrace.json" \
  --sql "SELECT environment->>'agent_framework' AS framework, environment->>'model' AS model, COUNT(*) AS cnt FROM sessions_base GROUP BY ALL ORDER BY cnt DESC LIMIT 10" \
  --output json

echo ""
echo "=== Step 10: Validate sample ==="
go run ./cmd/go-minitrace validate --path "$OUT_DIR/active/2026-03/" --recursive --output json \
  | python3 -c "import json,sys; d=json.load(sys.stdin); ok=sum(1 for r in d if r.get('valid_json')); print(f'Validated: {len(d)}, valid JSON: {ok}')"
