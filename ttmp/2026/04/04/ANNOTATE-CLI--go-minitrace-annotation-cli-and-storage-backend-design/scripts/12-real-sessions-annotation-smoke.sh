#!/usr/bin/env bash
# test-real-sessions.sh — E-to-end annotation test with real sessions.
set -euo pipefail

BIN="${BIN:-./go-minitrace}"
OUTPUT="./tmp/output"
GLOB="./tmp/output/active/*/*.minitrace.json"
PORT=18767

fail() { echo "FAIL: $1"; exit 1; }

cleanup() {
    pkill -f "go-minitrace serve" 2>/dev/null || true
}

trap cleanup EXIT

 echo "=== Real Session Annotation Test ==="

# Remove stale DBs from previous runs
rm -f "$OUTPUT/annotations.db"* "$OUTPUT/active/annotations.db"*

# --- Collect session IDs (clean, no trailing whitespace) ---
SESS_IDS=()
for f in $(find "$OUTPUT/active" -name "*.minitrace.json" | sort); do
    id=$(python3 -c "import json; print(json.load(open('$f'))['id'])" | tr -d '\n')
    SESS_IDS+=("$id")
done
COUNT=${#SESS_IDS[@]}
echo "Found $COUNT sessions:"
for id in "${SESS_IDS[@]}"; do echo "  $id"; done
echo ""

# --- Build ---
echo "--- Building ---"
go build -o "$BIN" ./cmd/go-minitrace/ || fail "build failed"

# --- Add one annotation per session ---
CATEGORIES=("ai-failure" "observation" "user-error" "success")
ANN_COUNT=0

for ((i=0; i<${#SESS_IDS[@]}; i++)); do
    SESS="${SESS_IDS[$i]}"
    CAT="${CATEGORIES[$((i % ${#CATEGORIES[@]}))]}"
    out=$("$BIN" annotate add \
        --output-dir "$OUTPUT" \
        --session "$SESS" \
        --category "$cat" \
        --title "$cat annotation (E2E test)" \
        --detail "Automated E2E test with real sessions" \
        --tags "e2e,real" \
        --annotator ci 2>&1)
    if echo "$out" | grep -q "Added annotation"; then
        ann_id=$(echo "$out" | grep "Added annotation" | awk '{print $NF}')
        echo "  ✓ $ann_id ($cat) on ${SESS:0:12}..."
        ANN_COUNT=$((ANN_COUNT + 1))
    else
        fail "annotate add failed: $out"
    fi
done
echo "  $ANN_COUNT annotations created"
echo ""

# --- Verify in SQLite ---
DB_ROWS=$(sqlite3 "$OUTPUT/annotations.db" "SELECT COUNT(*) FROM annotations;")
echo "--- SQLite: $DB_ROWS rows ---"
[ "$DB_ROWS" = "$ANN_COUNT" ] || fail "expected $ANN_COUNT rows in SQLite, got $DB_ROWS"

# --- List via CLI ---
echo ""
echo "--- List via CLI ---"
"$BIN" annotate list --output-dir "$OUTPUT" 2>&1 | head -10

# --- Sync to JSON ---
echo ""
echo "--- Sync annotations to .minitrace.json ---"
"$BIN" annotate sync --output-dir "$OUTPUT" 2>&1

# --- Verify annotations appear in JSON ---
echo ""
echo "--- Verify annotations in JSON ---"
SYNC_COUNT=0
for f in $(find "$OUTPUT/active" -name "*.minitrace.json" | sort); do
    id=$(python3 -c "import json; print(json.load(open('$f'))['id'])" | tr -d '\n')
    n=$(python3 -c "import json; print(len(json.load(open('$f')).get('annotations', [])))" | tr -d '\n')
    if [ "$n" = "0" ]; then
        echo "  $id: no annotations"
    else
        echo "  $id: ✓ $n annotation(s)"
        SYNC_COUNT=$((SYNC_COUNT + n))
    fi
done

# --- HTTP API test ---
echo ""
echo "--- Starting serve for HTTP API test ---"
"$BIN" serve \
    --archive-glob "$GLOB" \
    --db-path "$OUTPUT/analysis.dev.db" \
    --port "$PORT" \
    > /tmp/serve-real-test.log 2>&1 &
sleep 3

 echo "  serve started (pid $!)"

# GET all annotations
echo ""
echo "--- GET /api/annotations ---"
HTTP_COUNT=$(curl -s "http://localhost:$PORT/api/annotations" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))")
echo "  → $HTTP_COUNT annotations"
[ "$HTTP_COUNT" = "$ANN_COUNT" ] || fail "HTTP API returned $HTTP_COUNT, annotations, expected $ANN_COUNT"

# POST new annotation
echo ""
echo "--- POST new annotation ---"
FIRST_SESS="${SESS_IDS[0]}"
RESP=$(curl -s -X POST "http://localhost:$PORT/api/sessions/$FIRST_SESS/annotations" \
    -H 'Content-Type: application/json' \
    -d '{"category":"question","title":"HTTP test","detail":"via curl","annotator":"http-test"}')
NEW_ID=$(echo "$RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")
echo "  → created $NEW_ID"

# DELETE it
echo ""
echo "--- DELETE annotation ---"
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "http://localhost:$PORT/api/annotations/$NEW_ID")
echo "  → HTTP $CODE"
[ "$CODE" = "204" ] || fail "expected 204, got $CODE"

# Verify count is back to ANN_COUNT
echo ""
echo "--- Final count ---"
FINAL=$(curl -s "http://localhost:$PORT/api/annotations" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))")
echo "  → $FINAL annotations (expected $ANN_COUNT)"
[ "$FINAL" = "$ANN_COUNT" ] || fail "final count $FINAL != expected $ANN_COUNT"

echo ""
echo "=== ALL TESTS PASSED ==="
