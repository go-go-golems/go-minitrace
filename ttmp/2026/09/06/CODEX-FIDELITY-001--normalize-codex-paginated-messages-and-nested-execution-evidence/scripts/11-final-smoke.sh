#!/usr/bin/env bash
# Run from repository root. Only checkout commands and read-only source audits.
set -u
T="$PWD/ttmp/2026/09/06/CODEX-FIDELITY-001--normalize-codex-paginated-messages-and-nested-execution-evidence"
E="$T/various/final"
mkdir -p "$E"
failed=0
run() {
 local name="$1"; shift
 printf '%s ... ' "$name"
 if "$@" > "$E/$name.log" 2>&1; then echo PASS; echo 0 > "$E/$name.exit"; else local code=$?; echo "FAIL ($code)"; echo "$code" > "$E/$name.exit"; failed=1; fi
}
run make-all make all
run logcopter make logcopter-check
run race go test ./pkg/adapters/codex ./pkg/minitracedb ./cmd/go-minitrace/cmds/serve -race -count=1
run web-build bash -c 'cd web && pnpm build'
run web-lint bash -c 'cd web && pnpm lint'
run browser bash -c 'cd web && pnpm exec vitest run --project storybook'
run help go run ./cmd/go-minitrace help adapter-reference
run schema-help go run ./cmd/go-minitrace help minitrace-schema
run convert-private go run ./cmd/go-minitrace convert codex --source-list "$T/various/local-validation/sources.txt" --output-dir /tmp/codex-fidelity-final-private --run-record /tmp/codex-fidelity-final-private/receipt.json
run execution-audit python3 "$T/scripts/08-audit-execution-coverage.py" "$T/various/local-validation/sources.txt" /tmp/codex-fidelity-final-private
run message-audit python3 "$T/scripts/05-audit-message-coverage.py" "$T/various/local-validation/sources.txt" /tmp/codex-fidelity-final-private
run file-audit python3 "$T/scripts/09-audit-file-changes.py" "$T/various/local-validation/sources.txt" /tmp/codex-fidelity-final-private
run private-sql go run ./cmd/go-minitrace query run --archive-glob '/tmp/codex-fidelity-final-private/active/*/*.minitrace.json' --sql "SELECT session_id,turn_count,tool_call_count,tool_call_record_count,orchestration_count,execution_record_count,file_change_count,model_invocation_count,file_touch_count,confirmed_file_target_count FROM sessions ORDER BY session_id" --output json
run validator go run ./cmd/go-minitrace validate --path /tmp/codex-fidelity-final-private --archive --output json
run synthetic-oracle python3 "$T/scripts/04-check-synthetic-baseline.py" /tmp/codex-fidelity-final-synthetic/active/2026-09/fidelity-synthetic.minitrace.json
run history python3 "$T/scripts/10-check-history-consumers.py" '/tmp/codex-fidelity-final-synthetic/active/*/*.minitrace.json'
run proto-roundtrip bash -o pipefail -c "go run '$T/scripts/06-proto-outcomes.go' | node '$T/scripts/07-check-proto-outcomes.mjs' '$PWD'"
run doctor docmgr doctor --ticket CODEX-FIDELITY-001 --stale-after 30
run diff-check git diff --check
exit "$failed"
