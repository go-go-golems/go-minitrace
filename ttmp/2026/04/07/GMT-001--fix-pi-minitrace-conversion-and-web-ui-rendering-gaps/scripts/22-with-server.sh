#!/bin/bash
# Reusable: start go-minitrace serve, run a command, shut down.
# Usage: ./22-with-server.sh <command...>
# Example: ./22-with-server.sh curl -s http://localhost:$PORT/v2/sessions/ID/blocks | python3 ...
#
# Sets PORT env var for the command to use.
set -euo pipefail

PORT="${MINITRACE_PORT:-9877}"
export PORT

ARCHIVE_GLOB="${MINITRACE_GLOB:-./analysis/active/*/*.minitrace.json}"
export ARCHIVE_GLOB

# Start server in background
go-minitrace serve --archive-glob "$ARCHIVE_GLOB" --port "$PORT" 2>/tmp/mt-serve.log &
SERVER_PID=$!

cleanup() {
    kill $SERVER_PID 2>/dev/null
    wait $SERVER_PID 2>/dev/null
}
trap cleanup EXIT

# Wait for server to be ready
for i in $(seq 1 20); do
    if curl -sf "http://localhost:$PORT/api/sessions" > /dev/null 2>&1; then
        break
    fi
    sleep 0.25
done

# Run the user's command
"$@"
