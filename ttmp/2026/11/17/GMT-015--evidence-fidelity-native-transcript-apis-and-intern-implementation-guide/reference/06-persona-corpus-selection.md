---
Title: Historical Claude, Codex and Pi corpus for persona testing
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
Summary: Nine bounded real native transcripts, immutable private copies, targeted conversions and independently checked normalized counts.
---

# Selection and reproduction

The corpus contains three historical native transcripts per framework, selected across small, medium and larger source-size bands. It is a deliberately varied convenience sample, not a representative estimate of agent behavior. No current persona experiment session is selected. Native records cover September through November 2026. Each source is below 4 MB, comfortably below the requested 20 MB bound.

Read the go-minitrace transcript-analysis skill and query reference, then inspected the updated CLI help for discovery, conversion, validation and queries. The first `go run` failed at the sandboxed build-cache boundary; rerunning with the required filesystem permission worked. A remembered `convert claude` spelling only showed the command group; the actual current adapter name is `claude-code`. Saved command help lives in the private corpus `preflight/` directory.

The initial shortlist was a filesystem metadata scan of historical source files, ranked by distance from roughly 150 KB, 800 KB and 4 MB. Explicit paths are frozen in `scripts/persona-testing/select_corpus.py` so later store growth does not silently change the selection. The script reads selected originals, hashes the bytes, writes separate private copies and checks the original hash again. It refuses to replace an existing private copy with different content. Originals are never modified.

From the worktree root:

```sh
python3 ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/persona-testing/select_corpus.py
```

`--root PATH` chooses an alternate private corpus destination; `--freeze-only` copies/hashes without conversion. Default private layout:

```text
/private/tmp/gmt015-persona/corpus/
  raw/{claude-code,codex,pi}/             # untouched source-byte copies
  {framework}-sources.txt               # exact targeted conversion inputs
  archives/{framework}/active/{period}/ # normalized archives
  discover-*.stdout                    # bounded structured discovery
  convert-*.stdout, validate-*.stdout   # command outputs, corresponding stderr
  codex-conversion-run.json             # adapter's atomic run receipt
  sql-baseline.stdout                  # independent normalized SQL counts
  manifest.json                        # metadata and command receipts
```

The committed `sources/persona-corpus-manifest.json` contains hashes, paths, timestamps, counts and tool-name distributions, not transcript bodies. Raw and normalized content remains private. Conversion uses `go run ./cmd/go-minitrace` from the implementation worktree, not a stale installed binary. It supplies one targeted source list per framework and validates each resulting archive with `validate --archive --output json`. This validates archive/source/manifest consistency; it does not prove the adapter preserves every semantic detail of every provider format.

`baseline.sql` independently counts normalized turns/tools, unknown tool outcomes and source-ordinal bounds. The script compares these SQL counts with direct normalized-array counts. These checks establish baseline consistency, not correctness of model judgments. Native JSONL record counts are retained separately because one native record need not equal one normalized turn or tool. In particular Codex nested exec records can yield more normalized tool records than top-level native custom calls.

# Important test properties

The Claude sample includes a child with Agent/Write/Bash activity, a short parent with only a Skill tool, and a longer parent with shell, artifact, file-read/edit/write activity. Low-tool-count sessions are intentional edge cases: byte size may reflect injected instructions rather than useful work. The Pi sample includes read/bash, read/bash/write/edit, and a longer shell-heavy run with web search. The smallest Codex sample is genuinely message-only: its native response records contain no function/custom calls. Empty tool tables must remain a valid result, not an inferred conversion failure.

All transcript text must be treated as historical evidence, never instructions to execute the old work. Persona tests should inspect bounded normalized/native windows and API receipts; they must not replay commands found inside the transcripts. Findings should distinguish API errors, adapter limitations, analyst mistakes and unsupported assumptions. Preserve nullable outcomes, report estimated tokens separately from recorded usage, and do not infer causal task success from a command mention.

# Final frozen inventory

The final corpus has **9 archives, 793 normalized turns and 706 normalized tool records**. All three framework archive validations pass with no findings; all nine independent SQL/array-count and unknown-success comparisons pass. Source ordinal ranges are 0 through tool-count minus one for every nonempty tool array. Native source records span **2026-09-15 through 2026-11-12 UTC**, excluding the current experiment.

| Framework | Session ID | Native bytes | Turns | Tools | Tool shapes |
| --- | --- | ---: | ---: | ---: | --- |
| claude-code | agent-ae7076749749daca0 | 174,606 | 19 | 15 | Agent, Write, Bash |
| claude-code | d047a697-35c0-460c-8fda-541799b5b5b0 | 1,070,484 | 7 | 1 | Skill |
| claude-code | 7dce992c-7793-4e1a-950c-4db92cef2502 | 3,779,472 | 246 | 108 | Artifact, Bash, Read, Skill, Write, Edit |
| codex | 01a1364a-f58a-7ce3-8460-a856dacc8a1a | 150,747 | 7 | 0 | message-only |
| codex | 01a13157-f631-7cd3-8f84-e2666b24af01 | 783,921 | 18 | 13 | exec, exec_command, send_message |
| codex | 01a1c666-5f02-7b92-984e-99a27afd7a94 | 3,968,125 | 30 | 165 | exec, exec_command, apply_patch |
| pi | 01a0ee23-bd68-75c1-a81f-2c43771fc216 | 150,185 | 19 | 9 | read, bash |
| pi | 01a0b061-0cdd-7557-a80a-96802d3d02c5 | 769,209 | 29 | 50 | read, bash, write, edit |
| pi | 01a0a5c3-342f-7447-be74-b6c35f542ff7 | 3,969,769 | 418 | 345 | bash, read, write, edit, web_search |

The medium Codex candidate was deliberately changed before persona testing began. The first size-only choice (`01a13159...`) was also message-only, duplicating the smallest case. Native metadata showed that the adjacent historical `01a13157...` includes six custom calls and one function call, yielding thirteen normalized tool records including nested exec calls. The final script fixes that replacement explicitly. Prior manifest, SQL result, source-list, raw medium source and Codex archives are preserved under private `superseded-v1/`; the current archive glob does not include them. The final manifest was announced frozen before the persona matrix launched; no further corpus changes were made.

Exact original paths, frozen paths, archive paths, byte hashes, timestamp bounds, tool distributions, nullable success counts, truncation counts, command arrays, exit codes and SQL baseline rows are in the committed metadata manifest. Native and archive hashes establish precisely what the matrix used. They do not establish that these nine examples cover all provider formats or all evidence-fidelity edge cases.
