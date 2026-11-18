# Local JS API persona acceptance

Run from the repository root on macOS with Python, Go, Pi and the installed
Lunaroute extension. Choose a fresh scratch root for the pilot and matrix.

```sh
python3 path/to/run.py --root /private/tmp/minitrace-persona-new --cohort pilot
python3 path/to/run.py --root /private/tmp/minitrace-persona-new --cohort matrix
```

The pilot builds the binary and pins its hash and the input hashes. Pi receives
only the public JS API reference and a task. Built-in and extension tools are
disabled for Pi. Two tasks generate commands against a synthetic zero-tool
session and two synthetic sessions with tools. The fixed matrix uses two
models and two thinking levels. The host invokes commands with fixed flags
and compares the rows with the task oracle.

The replay policy denies network, reads of other user/scratch data, writes
outside the run directory and execution of other programs. A JS probe verifies
that the replay cannot read the pin or write to the fixture directory before
generated scripts run. Runtime outputs, full Pi sessions and scratch receipts
stay outside Git.

Use `--cohort pilot-replay` to replay preserved pilot scripts without a model
call. Use `--cohort followup` with a fresh scratch root to retest the mapped-report
task on GLM with both thinking levels after a documentation or API change.
