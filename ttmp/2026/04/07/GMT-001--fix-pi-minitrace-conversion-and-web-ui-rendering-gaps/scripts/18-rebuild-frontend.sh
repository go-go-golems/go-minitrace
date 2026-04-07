#!/bin/bash
# After fixing ToolCallRow.tsx and BlockBody.tsx, rebuild the frontend and embed
set -euo pipefail

cd /home/manuel/code/wesen/corporate-headquarters/go-minitrace

# Build frontend
cd web
npm run build

# Copy dist into embed directory
cd ..
rm -rf cmd/go-minitrace/cmds/serve/frontend/*
cp -r web/dist/* cmd/go-minitrace/cmds/serve/frontend/

# Rebuild go-minitrace with embedded frontend
go install ./cmd/go-minitrace
