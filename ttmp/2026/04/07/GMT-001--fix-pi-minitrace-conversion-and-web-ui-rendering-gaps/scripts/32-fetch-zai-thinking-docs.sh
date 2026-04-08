#!/bin/bash
# Fetch z.ai thinking mode documentation
set -euo pipefail

echo "=== Z.ai Thinking Mode Docs ==="
defuddle https://docs.z.ai/guides/capabilities/thinking-mode 2>/dev/null || curl -s https://docs.z.ai/guides/capabilities/thinking-mode

echo ""
echo "=== GLM-5 API Docs ==="
defuddle https://docs.z.ai/guides/llm/glm-5 2>/dev/null || curl -s https://docs.z.ai/guides/llm/glm-5
