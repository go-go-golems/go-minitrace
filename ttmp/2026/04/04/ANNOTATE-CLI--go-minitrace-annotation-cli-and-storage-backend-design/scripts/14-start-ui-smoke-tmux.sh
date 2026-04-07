#!/usr/bin/env bash
# Start backend + frontend in tmux against tmp/ui-smoke fixture.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../../../../../../.." && pwd)"
cd "$REPO"

TMUX_SESSION="${TMUX_SESSION:-go-minitrace-ui-smoke}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

(tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true)

tmux new-session -d -s "$TMUX_SESSION" -n backend

tmux send-keys -t "$TMUX_SESSION":backend \
  "cd $REPO && ./go-minitrace serve --archive-glob './tmp/ui-smoke/output/active/*/*.minitrace.json' --db-path ./tmp/ui-smoke/output/analysis.ui.db --port $BACKEND_PORT --dev 2>&1 | tee /tmp/ui-smoke-backend.log" Enter

tmux split-window -h -t "$TMUX_SESSION":backend

tmux send-keys -t "$TMUX_SESSION":backend.1 \
  "cd $REPO/web && npm run dev -- --host 0.0.0.0 --port $FRONTEND_PORT 2>&1 | tee /tmp/ui-smoke-frontend.log" Enter

sleep 5

echo "tmux session: $TMUX_SESSION"
echo "backend: http://127.0.0.1:$BACKEND_PORT"
echo "frontend: http://127.0.0.1:$FRONTEND_PORT"
echo "attach: tmux attach -t $TMUX_SESSION"
