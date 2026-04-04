#!/usr/bin/env bash
# scripts/09-e2e-duckdb-sqlite-live.sh
# End-to-end test: DuckDB sqlite_scanner — annotations are live in DuckDB.
# Starts serve in background, adds annotation via CLI, then verifies via DuckDB CLI.
set -euo pipefail

BIN="${BIN:-./go-minitrace}"
DUCKDB_BIN="${DUCKDB_BIN:-duckdb}"
DIR=$(mktemp -d)
DUCKDB_FILE="$DIR/analysis.duckdb"
SESS_FILE="$DIR/sessions/sess-001.minitrace.json"
PORT=18765

cleanup() {
  kill "${SERVER_PID:-}" 2>/dev/null || true
  rm -rf "$DIR"
}
trap cleanup EXIT
mkdir -p "$DIR/sessions"

echo "=== E2E: DuckDB sqlite_scanner — annotations are live ==="
echo "DIR: $DIR"

# Create a minimal .minitrace.json session.
cat > "$SESS_FILE" << 'EOF'
{
  "id": "sess-001",
  "schema_version": "0.2.0",
  "profile": "organic",
  "title": "DuckDB live query test",
  "classification": "internal",
  "provenance": {"source_format": "e2e", "source_path": "e2e"},
  "flags": {},
  "environment": {"agent_framework": "claude-code", "model": "claude-opus-4"},
  "operational_context": {"working_directory": "/tmp"},
  "timing": {"started_at": "2026-04-04T00:00:00Z"},
  "turns": [], "tool_calls": [],
  "annotations": []
}
EOF

echo "--- Building go-minitrace ---"
go build -o "$BIN" ./cmd/go-minitrace/ 2>/dev/null || \
  [ -x "$BIN" ] || { echo "BIN not found: $BIN"; exit 1; }

echo "--- Starting serve in background ---"
$BIN serve \
  --archive-glob "$SESS_FILE" \
  --db-path "$DUCKDB_FILE" \
  --port "$PORT" \
  > /dev/null 2>&1 &
SERVER_PID=$!
sleep 3

# Verify server is up.
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
  echo "FAIL: server failed to start (PID=$SERVER_PID)"
  exit 1
fi
echo "Server up on port $PORT (PID=$SERVER_PID) ✓"

echo ""
echo "--- Adding annotation via CLI ---"
$BIN annotate add \
  --output-dir "$DIR" \
  --session "sess-001" \
  --category "ai-failure" \
  --title "DuckDB live query test" \
  --tags "auth,regression"

# Use -set to pass the DB path as a DuckDB variable (avoids heredoc escaping issues).
ANNO_DB="$DIR/annotations.db"
DUCKDB_FILE_ESCAPED="${DUCKDB_FILE@Q}"
ANNO_DB_ESCAPED="${ANNO_DB@Q}"

echo ""
echo "--- Querying via DuckDB CLI (annotations should be live) ---"
$DUCKDB_BIN "$DUCKDB_FILE" -set 'anno_db' "$ANNO_DB" << 'SQL'
INSTALL sqlite_scanner;
LOAD sqlite_scanner;
CALL sqlite_attach(:anno_db, overwrite => true);

SELECT
    a.session_id,
    a.category,
    a.title,
    a.annotator
FROM annotations a
ORDER BY a.created_at;
SQL

echo ""
echo "--- Cross-session query (DuckDB sessions + SQLite annotations) ---"
$DUCKDB_BIN "$DUCKDB_FILE" -set 'anno_db' "$ANNO_DB" << 'SQL'
INSTALL sqlite_scanner;
LOAD sqlite_scanner;
CALL sqlite_attach(:anno_db, overwrite => true);

SELECT
    a.session_id,
    sb.environment->>'agent_framework' AS framework,
    a.category,
    a.title,
    COUNT(*) AS count
FROM annotations a
JOIN sessions_base sb ON sb.id = a.session_id
GROUP BY a.session_id, sb.environment->>'agent_framework', a.category, a.title;
SQL

echo ""
echo "--- Deleting annotation via CLI ---"
ANN_ID=$($DUCKDB_BIN "$ANNO_DB" -noheader -list "SELECT id FROM annotations LIMIT 1;" 2>/dev/null || echo "")
if [ -n "$ANN_ID" ]; then
  $BIN annotate delete --output-dir "$DIR" --id "$ANN_ID"
  echo "Deleted annotation $ANN_ID ✓"
fi

echo ""
echo "--- Verifying annotation removed from DuckDB ---"
REMAINING=$($DUCKDB_BIN "$DUCKDB_FILE" -set 'anno_db' "$ANNO_DB" -noheader -list << 'SQL'
INSTALL sqlite_scanner;
LOAD sqlite_scanner;
CALL sqlite_attach(:anno_db, overwrite => true);
SELECT COUNT(*) FROM annotations;
SQL
)
echo "Remaining in SQLite: $REMAINING"
[ "$REMAINING" -eq 0 ] || { echo "FAIL: annotation should be gone"; exit 1; }
echo "Live removal verified ✓"

echo ""
echo "=== ALL DUCKDB LIVE QUERY TESTS PASSED ==="
