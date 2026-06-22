#!/usr/bin/env bash
set -euo pipefail

WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)"
EXAMPLE_DIR="$WORKSPACE_ROOT/go-minitrace/examples/xgoja/minitrace-command-provider"
XGOJA_ROOT="$WORKSPACE_ROOT/go-go-goja"
TMP_SPEC="$EXAMPLE_DIR/xgoja.v2.tmp.yaml"
TMP_BIN="$EXAMPLE_DIR/dist/x.tmp"

cleanup() {
  rm -f "$TMP_SPEC" "$TMP_BIN"
}
trap cleanup EXIT

cd "$EXAMPLE_DIR"
echo "== current make smoke =="
if make smoke; then
  echo "make smoke: ok"
else
  echo "make smoke: failed as expected before example refresh"
fi

echo "== temporary v2 migration and build =="
(
  cd "$XGOJA_ROOT"
  GOWORK=off go run ./cmd/xgoja migrate-spec \
    -f "$EXAMPLE_DIR/xgoja.yaml" \
    --out "$TMP_SPEC"
  GOWORK=off go run ./cmd/xgoja build \
    -f "$TMP_SPEC" \
    --output "$TMP_BIN" \
    --xgoja-replace "$XGOJA_ROOT" \
    --keep-work
)
