# Live persona evaluation

This lab adds a stakeholder conversation to the file-attribution [brief](../../reference/16-open-ended-persona-briefs-for-real-transcript-analysis.md). It follows the Playback Asset Studio pattern of separate stakeholder, product agent, host verification and judge.

1. The stakeholder opens with the scenario's file-change request.
2. The Pi analyst writes JS and SQL, queries the pinned go-minitrace executable, and submits a reusable `ledger.js` command.
3. The host replays the command and compares a multiset of file-target evidence rows against an independent SQL query.
4. A separate stakeholder model asks a follow-up based on the reply and bounded ledger preview. A third analyst turn applies a fixed scope change to Codex-only. All three analyst turns use the same Pi session.
5. The judge receives the conversation and host results and assesses grounding, usefulness, uncertainty and efficiency. Replays check the ledger contract. The judge assesses the narrative against the evidence it was shown.

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

This uses the synthetic fixtures with real Pi models. It requires macOS Seatbelt (`sandbox-exec`), the Lunaroute extension, and working Lunaroute authentication for Pi. The models default to GLM-5.3 Flash and DeepSeek 4.1 Flash at low thinking. GLM-5.3 Flash also plays the stakeholder and judge. Set `--models`, `--reviewer` and `--thinking` to run a specific configuration.

Use `--offline` for a fresh batch with a scripted analyst and no model calls. The fixture asserts two write-like targets across all archives and one within Codex, including a failed attempted edit. Use `--corpus /path/to/approved/manifest.json` for the frozen nine-session corpus. Use a fresh batch directory each time. The manifest format comes from [the corpus selection script](../persona-testing/select_corpus.py). Only normalized archive copies are staged and their hashes are checked. Historical runs can send selected archive evidence to Lunaroute through analyst tool outputs and conversation previews, so use corpus material cleared for that provider.

The analyst has only the three tools defined in [analyst.ts](analyst.ts). It can save scripts and run SQL or custom JS verbs. A separate Seatbelt process runs generated code with network, subprocess and filesystem-write denials and access to the staged inputs and saved artifacts. The trusted Pi/provider process makes the model requests. Results delivered to the model carry a transport truncation flag; inspect query completeness separately.

## Replay a specific failure

```sh
python3 "$lab/replay.py" "$batch" glm-5.3-flash \
  --batch /private/tmp/gmt015-persona-prompt-variant --guidance /path/to/guidance.md
```

This reruns the opening analyst turn with the pinned inputs and saved prompt. Omit guidance for a repeat. Use a second viewer for the second batch. Later-turn and provider-request replay require recording additional state. Treat differences between single model responses as observations and repeat before attributing them to a prompt change.

## Read the evidence

The viewer polls local append-only logs and displays role, turn, tool call, generated script, result, replay and verdict. Switch model runs or filter by role. The viewer listens on loopback and serves only the timeline, so private raw receipts stay on disk. A batch directory contains model stdout, prompts, Pi session history, scripts, expected rows and candidate rows. Keep these artifacts local. End the viewer with `tmux kill-session -t minitrace-persona-viewer`.

If the host shows missing or unexpected rows, inspect the row-key samples and replay the script with the same archive selection. Identical counts can still conceal wrong rows. A failed model request or tool transport is recorded separately from a ledger disagreement.
