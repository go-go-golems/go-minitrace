#!/usr/bin/env bash
set -euo pipefail

# Discover Pi sessions using the current Go CLI.

REPO_DIR="/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
SOURCE_DIR="${1:-$HOME/.pi/agent/sessions}"

cd "$REPO_DIR"
go run ./cmd/go-minitrace discover pi --source-dir "$SOURCE_DIR" --output json
