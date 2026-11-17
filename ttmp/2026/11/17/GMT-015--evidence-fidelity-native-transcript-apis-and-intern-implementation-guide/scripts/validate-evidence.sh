#!/usr/bin/env bash
# Run from the worktree root. Full suite may expose the documented baseline macOS config failures.
set -u
export TMPDIR="${TMPDIR:-/private/tmp}"
status=0
go test ./pkg/minitracedb ./pkg/minitracejs ./pkg/nativeevidence ./pkg/minitracecmd || status=1
go test ./cmd/go-minitrace/cmds/query -run 'Test(StrictEvidenceCLIRejectsCellsBeforeEmission|RootJSCommandShowsCustomRequiredFieldAndRuntimeFlags|ValidCommandExecutesDespiteInvalidSibling)$' -count=1 || status=1
go build ./... || status=1
go vet ./... || status=1
exit "$status"
