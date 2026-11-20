# Live persona evaluation

For an operator handoff, follow the [persona cosplay playbook](../../reference/18-playbook-for-running-and-reviewing-persona-cosplay.md).

This lab adds a stakeholder conversation to the file-attribution [brief](../../reference/16-open-ended-persona-briefs-for-real-transcript-analysis.md). It follows the Playback Asset Studio pattern of separate stakeholder, product agent, host verification and judge.

1. The stakeholder opens with the scenario's file-change request.
2. The Pi analyst writes JS and SQL, queries the pinned go-minitrace executable, and submits a reusable `ledger.js` command.
3. The host replays the self-contained submission in its own command directory and compares a multiset of file-target evidence rows against an independent SQL query.
4. A separate stakeholder model asks a follow-up based on the reply and bounded ledger preview. A third analyst turn applies a fixed scope change to Codex-only. All three analyst turns use the same Pi session.
5. The Codex judge receives a bounded tool-friction summary and exact host replay receipts. It distinguishes CLI, documentation, script and harness friction. Replays independently check the ledger contract.

## Start the viewer and run a pilot

From the repository root:

```sh
ticket=$(find ttmp/2026/11/17 -maxdepth 1 -name 'GMT-015--*')
lab="$ticket/scripts/persona-live"
batch=/private/tmp/gmt015-persona-pilot-example
binary=/private/tmp/gmt015-persona-pinned

go build -o "$binary" ./cmd/go-minitrace
tmux new-session -d -s minitrace-persona-viewer python3 "$lab/serve.py" "$batch" --port 8794
open http://127.0.0.1:8794/
python3 "$lab/run.py" --binary "$binary" --batch "$batch"
```

This uses the synthetic fixtures with real Pi models. It requires macOS Seatbelt (`sandbox-exec`), Lunaroute authentication for Pi and ChatGPT login for the Codex CLI judge. The models default to GLM-5.3 Flash and DeepSeek 4.1 Flash at low thinking. GLM-5.3 Flash plays the stakeholder. The judge defaults to `openai-codex/gpt-6.2-sol` with medium thinking. Set `--models`, `--reviewer`, `--thinking`, `--judge-model`, and `--judge-thinking` to run a specific configuration.

Use `--offline` for a fresh batch with a scripted analyst and no model calls. The fixture asserts two write-like targets across all archives and one within Codex, including a failed attempted edit. Use `--corpus /path/to/approved/manifest.json` for the frozen nine-session corpus. Use a fresh batch directory each time. The manifest format comes from [the corpus selection script](../persona-testing/select_corpus.py). Only normalized archive copies are staged and their hashes are checked. Historical runs can send selected archive evidence to Lunaroute through analyst tool outputs and conversation previews, so use corpus material cleared for that provider.

The analyst has only the three tools defined in [analyst.ts](analyst.ts). It can save scripts and run SQL or custom JS verbs. Saving a JS file returns its registered command paths and scanner diagnostics, resolved by the same catalog as the invocation. For schema, pass `{kind: "schema"}` for table names or `{kind: "schema", text: "tool_calls"}` for columns. For general help, pass `{kind: "help", text: "js-api-reference"}`. For JS discovery, pass `{kind: "help", path: []}` to list the catalog or `{kind: "help", path: ["probe_sandbox"]}` to list a group's commands. Run a collapsed verb with `{kind: "js", path: ["ledger"]}` or a grouped one with `{kind: "js", path: ["probe_sandbox", "probe-sandbox"]}`. A separate Seatbelt process runs generated code with network, subprocess and filesystem-write denials and access to the staged inputs and saved artifacts. The trusted Pi/provider process makes the model requests. The model receives stdout/stderr previews and a receipt path under the run’s `tool-results/` directory. `queryComplete` reports an explicit query envelope or null for other outputs. `previewTruncated` describes the model-facing copy. `captureComplete` describes the saved local output, and `processError` distinguishes an exit, timeout/signal, capture limit or startup error. The harness retains up to 2 MiB on each subprocess stream.

## Run the recovery and efficiency scenarios

Pass `--scenario recovery` or `--scenario efficiency` with `--corpus` and a fresh batch directory. Both keep the adaptive follow-up and end with a Codex-only scope change. The recovery verifier checks failed tool call identities and the first later successful call with the same tool name. The efficiency verifier checks recorded token totals and coverage counts per session, preserving missing values. Each scenario's JSON defines the prompt and output fields, and its SQL file defines the independent oracle.

## Replay a specific failure

```sh
python3 "$lab/replay.py" "$batch" glm-5.3-flash \
  --batch /private/tmp/gmt015-persona-prompt-variant --guidance /path/to/guidance.md
```

Recheck a ledger without a provider call when you want to isolate a harness problem:

```sh
python3 "$lab/replay.py" "$batch" glm-5.3-flash \
  --ledger-turn 1 --batch /private/tmp/gmt015-persona-ledger-recheck
```

The recheck uses the ledger saved at the specified turn and the original input hashes. It loads only that command, so duplicate scratch verbs in other artifacts cannot affect the result.

The guidance command reruns the opening analyst turn with the pinned inputs and saved prompt. Omit guidance for a repeat. Use a second viewer for the second batch. Later-turn and provider-request replay require recording additional state. Treat differences between single model responses as observations and repeat before attributing them to a prompt change.

## Rejudge a completed batch

```sh
python3 "$lab/rejudge.py" "$batch" --output /private/tmp/gmt015-persona-api-rejudge
```

The rejudge copies the timeline to a fresh viewer directory and appends a separate Codex verdict. It sends only categorized tool counts and the exact replay checks. Command text, archive rows, paths and analyst prose stay local. Use `--timeline` to copy an existing chronological projection and `--recheck` to include an isolated opening-ledger recheck as separate evidence.

## Read the evidence

The viewer polls local append-only logs. **Conversation summary** groups the stakeholder requests, analyst replies, host checks and judge verdict. **Errors and code** surfaces saved scripts and command diagnostics. **Timeline** opens by default and follows event order with public replies and tool result previews inline. Expand a tool event for its full recorded arguments or result. It shows thinking start and end markers, plus elapsed time when captured live. It filters by role. Switch models with the run picker and enable **Follow latest** when you want new events to scroll into view.

For a completed batch created before the chronology markers were added, project the saved Pi event order into a fresh viewer directory:

```sh
python3 "$lab/trace.py" "$batch" /private/tmp/gmt015-persona-chronology
python3 "$lab/serve.py" /private/tmp/gmt015-persona-chronology --port 8795
```

The historical projection preserves sequence and tool call IDs. Its thinking markers have no wall-clock timestamps. The viewer listens on loopback and serves only the timeline, so private raw receipts stay on disk. A batch directory contains model stdout, prompts, Pi session history, scripts, expected rows and candidate rows. Keep these artifacts local. End the viewer with `tmux kill-session -t minitrace-persona-viewer`.

If the host shows missing or unexpected rows, inspect the row-key samples and replay the script with the same archive selection. Identical counts can still conceal wrong rows. A failed model request or tool transport is recorded separately from a ledger disagreement.

## Test command authoring across workloads

Repeat `--scenario efficiency` against the same nine-archive manifest and add `--scenario coverage` in a separate batch. Run both analysts in each. Coverage asks the analyst to prioritize missing outcome evidence and defend the choice, then narrow to Pi. Its SQL checks four counts per session and tool, including zero counts. The judge sees tool-use signals and exact replay receipts. Review the analyst's final answer separately for overclaims about missing outcomes.

Run `compare.py BASELINE CANDIDATE` for efficiency. Inspect `first_ledger_save_registered` (whether the first completed save returned the root ledger path), first successful ledger call ordinal, and exact replay at every turn. A first-save success alone says nothing about what the command calculates.

### Run the full thinking-level matrix

From the repository root, with the lab, scratch, binary and corpus paths set:

```sh
python3 "$lab/matrix.py" --root "$scratch/matrix" --binary "$binary" --corpus "$corpus"
python3 "$lab/matrix_results.py" "$scratch"/matrix/* > "$scratch/matrix-summary.json"
```

The launcher pairs both Lunaroute analysts at low and high thinking on file-change attribution, recovery, cost and coverage. The stakeholder stays at low thinking. The judge stays at medium. Each batch contains two conversations. The default runs two batches concurrently. Use a fresh root for each full repetition. If some cells already exist from the same revision and inputs, pass their task-effort names to `--omit` and give their batch directories to the summarizer alongside the new ones. The summary checks input hashes, per-task prompt-definition and oracle hashes, judge configuration, terminal state and matrix membership. Separate timeouts from row mismatches when comparing outcomes.
