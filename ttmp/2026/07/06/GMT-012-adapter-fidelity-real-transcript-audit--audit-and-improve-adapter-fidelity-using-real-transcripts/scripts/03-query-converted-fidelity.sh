#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
TICKET="$ROOT/ttmp/2026/07/06/GMT-012-adapter-fidelity-real-transcript-audit--audit-and-improve-adapter-fidelity-using-real-transcripts"
OUT="$TICKET/sources/converted-corpus"
LOGS="$TICKET/scripts/logs"
mkdir -p "$LOGS"
cd "$ROOT"

ARCHIVE_GLOB="$OUT/*/active/*/*.minitrace.json"
if ! compgen -G "$ARCHIVE_GLOB" >/dev/null; then
  echo "no converted archives match $ARCHIVE_GLOB" | tee "$LOGS/03-query-converted-fidelity.log"
  exit 0
fi

run_sql() {
  local name="$1"
  local sql="$2"
  local sql_file="$LOGS/03-${name}.sql"
  local out_file="$LOGS/03-${name}.json"
  printf '%s\n' "$sql" > "$sql_file"
  echo "## ${name}" | tee -a "$LOGS/03-query-converted-fidelity.log"
  GOWORK=off go run ./cmd/go-minitrace query run \
    --archive-glob "$ARCHIVE_GLOB" \
    --sql-file "$sql_file" \
    --output json | tee "$out_file"
}

: > "$LOGS/03-query-converted-fidelity.log"

run_sql sessions_by_framework "SELECT agent_framework, COUNT(*) AS sessions, SUM(tool_call_count) AS tool_calls, SUM(turn_count) AS turns FROM sessions GROUP BY agent_framework ORDER BY agent_framework;"

run_sql tool_fidelity "SELECT s.agent_framework, COUNT(*) AS tool_calls, SUM(tc.duration_ms IS NULL) AS missing_duration, SUM(tc.exit_code IS NULL) AS missing_exit_code, SUM(COALESCE(tc.error, '') <> '') AS error_outputs, SUM(tc.truncated = 1) AS truncated_outputs FROM tool_calls tc JOIN sessions s USING (session_id) GROUP BY s.agent_framework ORDER BY s.agent_framework;"

run_sql turn_fidelity "SELECT s.agent_framework, COUNT(*) AS turns, SUM(COALESCE(t.thinking, '') <> '') AS turns_with_thinking, SUM(t.input_tokens IS NOT NULL OR t.output_tokens IS NOT NULL OR t.cache_read_tokens IS NOT NULL OR t.reasoning_tokens IS NOT NULL) AS turns_with_usage FROM turns t JOIN sessions s USING (session_id) GROUP BY s.agent_framework ORDER BY s.agent_framework;"

run_sql event_kinds "SELECT s.agent_framework, e.kind, COUNT(*) AS events FROM events e JOIN sessions s USING (session_id) GROUP BY s.agent_framework, e.kind ORDER BY s.agent_framework, events DESC, e.kind LIMIT 100;"

run_sql attachments "SELECT s.agent_framework, a.kind, COUNT(*) AS attachments FROM attachments a JOIN sessions s USING (session_id) GROUP BY s.agent_framework, a.kind ORDER BY s.agent_framework, attachments DESC;"

echo "Logs in $LOGS/03-*.json"
