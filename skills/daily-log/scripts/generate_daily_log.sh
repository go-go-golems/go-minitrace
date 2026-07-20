#!/usr/bin/env bash
#
# generate_daily_log.sh — Stages 1-3 of the daily-log skill.
#
# Discovers all Pi and Codex sessions active on a target day, converts them
# to normalized minitrace archives, and runs the session-list overview query.
#
# This script does NOT write the report or verify against git. Those stages
# require judgment and are done manually. See the daily-log SKILL.md.
#
# Usage:
#   generate_daily_log.sh <TARGET_DAY> [INVEST_DIR]
#
#   TARGET_DAY   The day to report on, in YYYY-MM-DD format.
#   INVEST_DIR   Optional. Where to store archives/queries/results.
#                Defaults to scripts/<TODAY>/daily-report-<TARGET_DAY>
#                relative to the current working directory.
#
# Run from the claw-stuff repo root (or any repo with a scripts/ dir).

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: generate_daily_log.sh <TARGET_DAY> [INVEST_DIR]" >&2
  echo "  TARGET_DAY  e.g. 2026-07-19" >&2
  exit 1
fi

TARGET_DAY="$1"
TODAY="$(date +%Y/%m/%d)"

if [[ $# -ge 2 ]]; then
  INVEST_DIR="$2"
else
  INVEST_DIR="scripts/${TODAY}/daily-report-${TARGET_DAY}"
fi

mkdir -p "$INVEST_DIR"/{archives,queries,results}

echo "=== Daily log investigation ==="
echo "Target day:  $TARGET_DAY"
echo "Invest dir:  $INVEST_DIR"
echo ""

# ---- Stage 1: Discover ----
echo "=== Stage 1: Discover candidate sessions ==="

go-minitrace discover pi \
  --source-dir ~/.pi/agent/sessions \
  --active-since "$TARGET_DAY" \
  --output json > "$INVEST_DIR/results/pi-discovery.json" 2>&1 || {
    echo "ERROR: pi discovery failed" >&2
    exit 1
  }

go-minitrace discover codex \
  --source-dir ~/.codex \
  --active-since "$TARGET_DAY" \
  --output json > "$INVEST_DIR/results/codex-discovery.json" 2>&1 || {
    echo "ERROR: codex discovery failed" >&2
    exit 1
  }

go-minitrace discover claude-code \
  --source-dir ~/.claude/projects \
  --active-since "$TARGET_DAY" \
  --output json > "$INVEST_DIR/results/claude-code-discovery.json" 2>&1 || {
    echo "ERROR: claude-code discovery failed" >&2
    exit 1
  }

PI_COUNT=$(python3 -c "import json; print(len(json.load(open('$INVEST_DIR/results/pi-discovery.json'))))" 2>/dev/null || echo "?")
CODEX_COUNT=$(python3 -c "import json; print(len(json.load(open('$INVEST_DIR/results/codex-discovery.json'))))" 2>/dev/null || echo "?")
CLAUDE_COUNT=$(python3 -c "import json; print(len(json.load(open('$INVEST_DIR/results/claude-code-discovery.json'))))" 2>/dev/null || echo "?")
echo "Pi candidates:          $PI_COUNT"
echo "Codex candidates:       $CODEX_COUNT"
echo "Claude Code candidates: $CLAUDE_COUNT"
echo ""

# ---- Build per-framework source lists ----
# Two rules are enforced here, both of which the earlier single mixed list got
# wrong:
#
#   1. Each adapter converts only its own framework's sessions. Feeding a Codex
#      or Claude transcript to `convert pi` makes the Pi adapter publish it as
#      an empty or misclassified Pi session, and a directory-form Claude session
#      can fail the whole staged batch. So we write one list per framework, each
#      from its own discovery file, and convert each with its own adapter.
#
#   2. Sessions that started after the target day are dropped. `discover
#      --active-since` is a lower bound only ("active at or after"); with no
#      upper bound, a report generated days later would pull in every session
#      from the intervening days and inflate the totals and timelines for the
#      reported day.
build_source_list() {
  local discovery="$1" out="$2"
  python3 -c "
import json
target = '$TARGET_DAY'
paths = []
try:
    for s in json.load(open('$discovery')):
        started = (s.get('started_at') or '')[:10]
        last = (s.get('last_activity_at') or '')[:10]
        if started and started > target:   # started after the reported day
            continue
        if last and last < target:         # last active before the reported day
            continue
        p = s.get('source_path')
        if p:
            paths.append(p)
except Exception:
    pass
for p in sorted(set(paths)):
    print(p)
" > "$out"
}

build_source_list "$INVEST_DIR/results/pi-discovery.json"          "$INVEST_DIR/pi-sources.txt"
build_source_list "$INVEST_DIR/results/codex-discovery.json"       "$INVEST_DIR/codex-sources.txt"
build_source_list "$INVEST_DIR/results/claude-code-discovery.json" "$INVEST_DIR/claude-code-sources.txt"

echo "Source lists (own-framework only, sessions active on $TARGET_DAY):"
echo "  pi:          $(wc -l < "$INVEST_DIR/pi-sources.txt")          -> $INVEST_DIR/pi-sources.txt"
echo "  codex:       $(wc -l < "$INVEST_DIR/codex-sources.txt")       -> $INVEST_DIR/codex-sources.txt"
echo "  claude-code: $(wc -l < "$INVEST_DIR/claude-code-sources.txt") -> $INVEST_DIR/claude-code-sources.txt"
echo ""

# ---- Stage 2: Convert ----
echo "=== Stage 2: Convert to archives ==="

# Convert each framework from its own source list. On preflight failures
# (for example "missing native session ID"), pass the offending sessions
# explicitly with repeatable --source-session flags rather than suppressing
# the error — a failure means a bad input, not a reason to skip the day.
convert_framework() {
  local fw="$1" list="$2"
  if [[ ! -s "$list" ]]; then
    echo "  $fw: no sessions active on $TARGET_DAY, skipping"
    return
  fi
  go-minitrace convert "$fw" \
    --source-list "$list" \
    --output-dir "$INVEST_DIR/archives/$fw" 2>&1 | tail -5 || {
      echo "WARNING: $fw convert had issues; retry failing sessions with --source-session" >&2
    }
}

convert_framework pi          "$INVEST_DIR/pi-sources.txt"
convert_framework codex       "$INVEST_DIR/codex-sources.txt"
convert_framework claude-code "$INVEST_DIR/claude-code-sources.txt"

echo ""
GLOB="$INVEST_DIR/archives/*/active/*/*.minitrace.json"
ARCHIVE_COUNT=$(ls $GLOB 2>/dev/null | wc -l)
echo "Archives created: $ARCHIVE_COUNT"
echo ""

# ---- Stage 3: Query overview ----
echo "=== Stage 3: Session-list overview ==="

go-minitrace query run \
  --archive-glob "$GLOB" \
  --preset session-list 2>&1 || {
    echo "WARNING: session-list query failed" >&2
  }

echo ""
echo "=== Stages 1-3 complete ==="
echo ""
echo "Next steps (manual):"
echo "  1. Identify repositories and tickets from the session-list above."
echo "  2. Run history file-history per repo path fragment:"
echo "     go-minitrace query commands history file-history --archive-glob '$GLOB' --path '<fragment>' --output json"
echo "  3. Run history ticket-timeline per ticket fragment:"
echo "     go-minitrace query commands history ticket-timeline --archive-glob '$GLOB' --ticket '<FRAGMENT>' --output json"
echo "  4. Verify commit counts against git:"
echo "     git -C <repo> log --since='$TARGET_DAY 00:00:00' --until='$TARGET_DAY 23:59:59' --oneline | wc -l"
echo "  5. Read full changelog entries from disk (the verb truncates detail)."
echo "  6. Write the report to the vault using references/report-template.md."
echo ""
echo "Investigation artifacts: $INVEST_DIR"
