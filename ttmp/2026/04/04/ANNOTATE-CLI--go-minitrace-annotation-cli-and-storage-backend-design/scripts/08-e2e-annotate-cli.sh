#!/usr/bin/env bash
# scripts/08-e2e-annotate-cli.sh
# End-to-end test: CLI annotate add → sqlite3 verify → sync → validate
set -euo pipefail

BIN="${BIN:-./go-minitrace}"
DIR=$(mktemp -d)
SESSION_FILE="$DIR/sess-001.minitrace.json"
DB="$DIR/annotations.db"

cleanup() {
  rm -rf "$DIR"
}
trap cleanup EXIT

echo "=== E2E: annotate CLI ==="
echo "DIR: $DIR"

# Create a minimal .minitrace.json session file.
cat > "$SESSION_FILE" << 'EOF'
{
  "id": "sess-001",
  "schema_version": "0.2.0",
  "profile": "organic",
  "title": "E2E test session",
  "classification": "internal",
  "provenance": {
    "source_format": "e2e-test",
    "source_path": "e2e-test"
  },
  "flags": {},
  "environment": {
    "agent_framework": "claude-code",
    "model": "claude-opus-4"
  },
  "operational_context": {
    "working_directory": "/tmp"
  },
  "timing": {
    "started_at": "2026-04-04T00:00:00Z"
  },
  "turns": [],
  "tool_calls": [],
  "annotations": []
}
EOF

echo "--- Adding annotation ---"
$BIN annotate add \
  --output-dir "$DIR" \
  --session "sess-001" \
  --category "ai-failure" \
  --title "Test failure annotation" \
  --detail "This is a test failure for E2E validation" \
  --tags "auth,e2e" \
  --taxonomy-minitrace "F-AUT" \
  --annotator "e2e-test"

echo ""
echo "--- Verifying in SQLite ---"
sqlite3 "$DB" "SELECT id, session_id, category, title, scope_type FROM annotations;"

ANNOTATION_COUNT=$(sqlite3 "$DB" "SELECT COUNT(*) FROM annotations;")
if [ "$ANNOTATION_COUNT" -ne 1 ]; then
  echo "FAIL: expected 1 annotation, got $ANNOTATION_COUNT"
  exit 1
fi
echo "Annotation count: $ANNOTATION_COUNT ✓"

echo ""
echo "--- Syncing to JSON ---"
$BIN annotate sync \
  --output-dir "$DIR" \
  --archive-glob "$DIR/*.minitrace.json" \
  --dry-run

$BIN annotate sync \
  --output-dir "$DIR" \
  --archive-glob "$DIR/*.minitrace.json"

echo ""
echo "--- Verifying JSON was updated ---"
ANNOTATIONS_IN_JSON=$(python3 -c "
import json, sys
with open('$SESSION_FILE') as f:
    data = json.load(f)
anns = data.get('annotations', 'MISSING')
print(f'type={type(anns).__name__}, count={len(anns)}')
if not isinstance(anns, list):
    print('FAIL: annotations is not a list')
    sys.exit(1)
if len(anns) != 1:
    print(f'FAIL: expected 1 annotation, got {len(anns)}')
    sys.exit(1)
print(f'category={anns[0][\"content\"][\"category\"]}')
print('annotations[] present and non-null ✓')
")

echo "$ANNOTATIONS_IN_JSON"

echo ""
echo "--- Listing annotations ---"
$BIN annotate list --output-dir "$DIR" --format table

echo ""
echo "--- Editing annotation ---"
ANN_ID=$(sqlite3 "$DB" "SELECT id FROM annotations LIMIT 1;")
$BIN annotate edit \
  --output-dir "$DIR" \
  --id "$ANN_ID" \
  --title "Updated: Test failure annotation"

echo ""
echo "--- Deleting annotation ---"
$BIN annotate delete --output-dir "$DIR" --id "$ANN_ID"

REMAINING=$(sqlite3 "$DB" "SELECT COUNT(*) FROM annotations;")
if [ "$REMAINING" -ne 0 ]; then
  echo "FAIL: expected 0 annotations after delete, got $REMAINING"
  exit 1
fi
echo "Delete OK ✓"

echo ""
echo "--- Validating JSON ---"
$BIN validate --path "$SESSION_FILE"
echo "Validate OK ✓"

echo ""
echo "=== ALL E2E TESTS PASSED ==="
