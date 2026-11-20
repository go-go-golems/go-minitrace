---
Title: Playbook for running and reviewing persona cosplay
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/rejudge.py
      Note: Rejudge an existing batch
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/replay.py
      Note: Replay the opening prompt or recheck an artifact
    - Path: repo://ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-live/run.py
      Note: Build the local batch and run the persona turns
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-18T17:58:04.465641-05:00
WhatFor: ""
WhenToUse: ""
---

# Playbook for running and reviewing persona cosplay

Use a persona run when you want to see how a coding agent actually learns and uses a product surface. Give it a job, let it choose its method, and record its attempts. An independent executable check establishes the result. The trace explains what got in the way.

## 1. Set up the roles

| Role | What it sees and does |
| --- | --- |
| Stakeholder | Plays the user. Starts with a fixed opening from the scenario card, then asks a follow-up based on the analyst's answer and bounded evidence. |
| Analyst | Runs in Pi. Writes its own SQL and JS using a pinned go-minitrace CLI. Carries the same Pi session across turns. |
| Host verifier | Replays the submitted command in isolation and compares exact rows against independent SQL. Holds the expected result. |
| Judge | Reviews measured tool friction and replay receipts. The default is `openai-codex/gpt-6.2-sol` at medium effort through the authenticated Codex CLI. |

The working scenario is a file-change handoff. The analyst writes a self-contained `ledger.js`. The maintainer challenges an evidence claim, then asks for the same command on Codex-only archives. The host compares the exact target keys and evidence statuses on every turn. An attempted edit, a tool report, and a confirmed effect are different evidence.

## 2. Build and verify the fixture

Run from a go-minitrace worktree on macOS. Install Go as specified by `go.mod`, Python 3, and tmux. Model runs need Pi with the Lunaroute extension and an authenticated Codex CLI. The generated JS runs under macOS Seatbelt. Use the offline fixture to verify the host before troubleshooting model access.

```sh
repo=$(git rev-parse --show-toplevel)
cd "$repo"
ticket=$(find "$repo/ttmp/2026/11/17" -maxdepth 1 -name 'GMT-015--*')
lab="$ticket/scripts/persona-live"
scratch=$(mktemp -d /private/tmp/minitrace-persona.XXXXXX)
binary="$scratch/go-minitrace"

git status --short
git rev-parse HEAD > "$scratch/revision.txt"
go build -o "$binary" ./cmd/go-minitrace
python3 "$lab/run.py" --offline --binary "$binary" --batch "$scratch/fixture"
```

The scripted fixture should match two write targets on each all-archive turn and one on the Codex-only turn. It runs the actual CLI and independent verifier. To exercise the model and provider path without historical data, run the next section without `--corpus`. The analysts receive the same small fixture.

## 3. Run the historical case and watch live

Request access to a frozen, approved corpus from the person handing off the experiment. The original nine archives came from Claude Code, Codex, and Pi. A manifest has a `rows` array containing `framework`, absolute `archive_path`, and `archive_sha256` per normalized archive. Use framework values `claude-code`, `codex`, and `pi`. For a new corpus, hash the archived files with `shasum -a 256` and use the same manifest format. This scenario expects the Codex subset to contain some but fewer qualifying targets than the full corpus. Its preflight enforces that condition.

Archived evidence visible in analyst tools and stakeholder previews goes to Lunaroute. The API judge receives categorized tool signals, checked counts, and bounded cited target references. Raw tool output and archive rows remain in the local batch directory.

```sh
manifest=/absolute/path/to/approved/manifest.json
batch="$scratch/historical"
session="persona-$(basename "$scratch")"
port=8794

tmux new-session -d -s "$session" python3 "$lab/serve.py" "$batch" --port "$port"
open "http://127.0.0.1:$port/"
python3 "$lab/run.py" --batch "$batch" --binary "$binary" --corpus "$manifest" \
  --models deepseek-4.1-flash glm-5.3-flash --thinking low --timeout 300 \
  --judge-model gpt-6.2-sol --judge-thinking medium
```

The viewer can start before the batch exists. Choose a free port for each viewer and a fresh directory for each batch. The default stakeholder is GLM-5.3 Flash. The runner copies and hashes the executable and selected normalized archives. Preserve `batch.json`, the worktree revision, and the exact harness source used for a comparison. Record any uncommitted diff alongside the run.

Use **Timeline** to see the stakeholder request, thinking activity markers, tool calls and results, analyst replies, host replays, and verdict in order. **Conversation summary** is faster for reading the exchanges. **Errors and code** is useful for finding repeated diagnostics and script versions. Turn on **Follow latest** during a live run. The server listens on loopback. Stop it with `tmux kill-session -t "$session"` when review is done.

## 4. Review a run from evidence

Start with the host replay. The frozen nine-archive case expects 34, 34, and one target. Check missing and unexpected row identities when totals match. Compare the submitted script with its local exact replay.

| Question | Where to look |
| --- | --- |
| What did the model see and try? | `events.jsonl` and the phase prompt in the model directory. |
| What did the CLI actually emit? | `tool-results/<receipt>/stdout`, `stderr`, and `receipt.json`. |
| Which script was graded? | `ledger-turn-N.js`, `expected-N-SCOPE.json`, and `observed-N-SCOPE.json`. |
| What evidence did the judge get? | `api-judgment.prompt.md` and `api-judgment.receipt.json`. |

Read the tool statuses independently. `queryComplete` describes an explicit result envelope. A null means the tool could not infer query completeness. `previewTruncated` describes the copy sent to the analyst. `captureComplete` describes the locally saved subprocess stream. `processError` identifies errors such as timeout, nonzero exit or capture limit.

For each wasted step, write down the exact attempt, diagnostic, next action and responsible layer. Useful categories are API constraint, misleading error, missing help, script mistake, unavailable source, harness defect and provider interruption. Check whether the error points to a next action the analyst can perform. Treat a provider-interrupted attempt as its own record and retry that model in a new batch on the same binary and corpus.

## 5. Isolate, repeat, and compare

Use the smallest replay that can answer the question.

```sh
# Recheck a saved ledger with the same binary and archives, without a model.
python3 "$lab/replay.py" "$batch" glm-5.3-flash --ledger-turn 1 \
  --batch "$scratch/ledger-recheck"

# Repeat the opening analyst turn with one help or prompt change.
python3 "$lab/replay.py" "$batch" glm-5.3-flash --guidance /absolute/path/to/guidance.md \
  --batch "$scratch/opening-variant"

# Change or repeat the judge on saved analyst runs.
python3 "$lab/rejudge.py" "$batch" --output "$scratch/rejudge"

# Compare two full runs.
baseline=/absolute/path/to/baseline-batch
candidate="$batch"
python3 "$lab/compare.py" "$baseline" "$candidate" > "$scratch/comparison.json"
```

Opening-turn replay reruns the saved prompt. A specific later provider request needs additional state capture. The host-only recheck proves the submitted artifact against the frozen scope. It does not change what the stakeholder saw live.

For a full rerun, preserve the models, thinking levels, scenario and archive hashes. Compare first successful JS call, first successful submitted ledger, repeated error classes and exact results after the scope change. Review harder scenarios more than once before comparing rates. Keep fixture failures, host replay and judge observations separate. Recording the judge's input matters because an evaluation can only verify claims it can see.

## 6. Add a scenario or hand off a batch

The same role split works with a browser UI or another CLI. Give the analyst the real product entry point. Replace the ledger replay with a product-specific check, such as compiling the submitted Asset Studio source and inspecting its captured output. Keep the persona's view limited to the evidence a real stakeholder would see.

Start with a job the stakeholder can explain in one paragraph. Pick an opening that leaves the analyst free to choose its method. Give the stakeholder room to challenge one claim. Include a fixed scope shift and one executable artifact. Define the artifact's objective checks before writing the persona prompt.

The default file-change scenario lives in [scenario.json](../scripts/persona-live/scenario.json) and [oracle.sql](../scripts/persona-live/oracle.sql). Use `--scenario recovery` or `--scenario efficiency` with a historical corpus to run two additional cases. Each pairs a prompt and output field list with independent SQL. The [open-ended briefs](16-open-ended-persona-briefs-for-real-transcript-analysis.md) include a handoff scenario you can add using the same pattern.

Give the next reviewer the revision, archived-input hashes, scenario and oracle hashes, analyst and judge settings, run status, comparison receipt and the best traces for each proposed fix. Raw private archives and receipts can stay with the authorized operator. The [latest compact results](../sources/live-persona-discovery-follow-up.json) show the intended handoff size.

## What to improve next

1. **Give the analyst the resolved command path as early as possible.** For a saved JS command, surface its registered path or the scanner reason using the real catalog. For an unresolved invocation, return the available command path as structured data the analyst can use on the next call. Measure whether that next call succeeds. Both analysts still had routing retries even after the text diagnostics improved.
2. **Expose schema in the SQL tool's own discovery surface.** Offer one compact lookup backed by the existing normalized schema. Return table names by default and columns for a named table. Both analysts still tried `sqlite_master` once, despite the improved help.
3. **Expand the evidence beyond one familiar ledger task.** Run recovery and efficiency next, with one held-out task and matched repeats on harder cases. Use results and traces to decide whether any fluent API change is necessary.

The Asset Studio team first reproduced a specific failing call, then tested a targeted correction before repeating the full scenario. Use the same order here. [Asset Studio playbook](</Users/manuel.odendahl/code/tulip/playback/.worktrees/feat-asset-studio/docs/operations/agent-prompt-playbook.md:285>) · [Minitrace and Asset Studio review](17-live-persona-api-friction-and-asset-studio-lessons.md)
