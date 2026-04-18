#!/usr/bin/env bash
set -euo pipefail

DAY="${1:-$(date -d 'yesterday' +%F)}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_ROOT="${NIGHTLY_TRANSCRIPT_REVIEW_WORKDIR:-${TMPDIR:-/tmp}/nightly-transcript-review}"
RUN_DIR="$WORK_ROOT/$DAY"
OUTPUT_DIR="$RUN_DIR/output"
REPORT_DIR="$RUN_DIR/report"
PI_SOURCES_FILE="$RUN_DIR/pi-sources.txt"
CODEX_SOURCES_FILE="$RUN_DIR/codex-sources.txt"
CODEX_STAGE_HOME="$RUN_DIR/codex-stage/.codex"
ARCHIVE_GLOB="$OUTPUT_DIR/active/*/*.minitrace.json"
LOG_FILE="$RUN_DIR/nightly-review.log"

REPO_ROOT="$SCRIPT_DIR"
while [[ "$REPO_ROOT" != "/" && ! -f "$REPO_ROOT/go.mod" ]]; do
  REPO_ROOT="$(dirname "$REPO_ROOT")"
done
if [[ ! -f "$REPO_ROOT/go.mod" ]]; then
  echo "could not locate repo root from $SCRIPT_DIR" >&2
  exit 1
fi
GO_MINITRACE_BIN="${GO_MINITRACE_BIN:-$REPO_ROOT/go-minitrace}"

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required executable: $1" >&2
    exit 1
  }
}

require "$GO_MINITRACE_BIN"
require jq
require python3

mkdir -p "$OUTPUT_DIR" "$REPORT_DIR"
: > "$LOG_FILE"

# Discover yesterday's Pi sessions from the native Pi store.
"$GO_MINITRACE_BIN" discover pi \
  --source-dir "$HOME/.pi/agent/sessions" \
  --output json \
  | jq -r --arg day "$DAY" '.[] | select(.source_path | contains($day)) | .source_path' \
  | tee "$PI_SOURCES_FILE" >/dev/null

# Discover yesterday's Codex sessions from the native Codex store.
"$GO_MINITRACE_BIN" discover codex \
  --source-dir "$HOME/.codex" \
  --output json \
  | jq -r --arg day "$DAY" '.[] | select(.source_path | contains("/sessions/" + $day[0:4] + "/" + $day[5:7] + "/" + $day[8:10] + "/")) | .source_path' \
  | tee "$CODEX_SOURCES_FILE" >/dev/null

# Convert Pi sessions one by one; convert Codex sessions after staging them into
# a temporary ~/.codex-shaped tree.
if [[ -s "$PI_SOURCES_FILE" ]]; then
  while IFS= read -r source_path; do
    [[ -n "$source_path" ]] || continue
    "$GO_MINITRACE_BIN" convert pi \
      --source-session "$source_path" \
      --output-dir "$OUTPUT_DIR" \
      >> "$LOG_FILE" 2>&1
  done < "$PI_SOURCES_FILE"
fi

if [[ -s "$CODEX_SOURCES_FILE" ]]; then
  rm -rf "$RUN_DIR/codex-stage"
  while IFS= read -r source_path; do
    [[ -n "$source_path" ]] || continue
    rel_path="${source_path#${HOME}/.codex/}"
    dest_path="$CODEX_STAGE_HOME/$rel_path"
    mkdir -p "$(dirname "$dest_path")"
    cp "$source_path" "$dest_path"
  done < "$CODEX_SOURCES_FILE"

  "$GO_MINITRACE_BIN" convert codex \
    --source-dir "$RUN_DIR/codex-stage/.codex" \
    --output-dir "$OUTPUT_DIR" \
    >> "$LOG_FILE" 2>&1
fi

if [[ ! -d "$OUTPUT_DIR/active" ]]; then
  echo "no converted sessions were produced for $DAY" >&2
  exit 1
fi

# Run the reusable query-command bundle and persist the raw outputs for
# follow-up analysis in later context windows.
"$GO_MINITRACE_BIN" query commands nightly session-inventory \
  --archive-glob "$ARCHIVE_GLOB" \
  --day "$DAY" \
  --output json \
  > "$REPORT_DIR/session-inventory.json"

"$GO_MINITRACE_BIN" query commands nightly workspace-summary \
  --archive-glob "$ARCHIVE_GLOB" \
  --day "$DAY" \
  --output json \
  > "$REPORT_DIR/workspace-summary.json"

"$GO_MINITRACE_BIN" query commands nightly tool-breakdown \
  --archive-glob "$ARCHIVE_GLOB" \
  --day "$DAY" \
  --output json \
  > "$REPORT_DIR/tool-breakdown.json"

"$GO_MINITRACE_BIN" query commands nightly followup-candidates \
  --archive-glob "$ARCHIVE_GLOB" \
  --day "$DAY" \
  --output json \
  > "$REPORT_DIR/followup-candidates.json"

"$GO_MINITRACE_BIN" query commands nightly annotation-summary \
  --archive-glob "$ARCHIVE_GLOB" \
  --day "$DAY" \
  --output json \
  > "$REPORT_DIR/annotation-summary.json"

python3 "$SCRIPT_DIR/render-nightly-report.py" \
  --day "$DAY" \
  --pi-sources "$PI_SOURCES_FILE" \
  --codex-sources "$CODEX_SOURCES_FILE" \
  --session-inventory "$REPORT_DIR/session-inventory.json" \
  --workspace-summary "$REPORT_DIR/workspace-summary.json" \
  --tool-breakdown "$REPORT_DIR/tool-breakdown.json" \
  --followup-candidates "$REPORT_DIR/followup-candidates.json" \
  --annotation-summary "$REPORT_DIR/annotation-summary.json" \
  --output "$RUN_DIR/nightly-review.md"

echo "nightly review ready: $RUN_DIR/nightly-review.md"
