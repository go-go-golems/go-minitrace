---
Title: Geppetto Turns Identity and Minitrace Delta Handling Guide
Ticket: GMT-008
Status: active
Topics:
    - minitrace
    - turnsdb
    - conversion
    - tool-calls
    - coinvault
    - web-ui
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../geppetto/pkg/events/correlation.go
      Note: New canonical typed correlation contract added after initial go-minitrace turnsdb support.
    - Path: ../../../../../../../../geppetto/pkg/turns/helpers_blocks.go
      Note: Geppetto block constructors and tool-call/tool-use ID behavior.
    - Path: ../../../../../../../../geppetto/pkg/turns/types.go
      Note: Geppetto Turn and Block data model with current block ID fields.
    - Path: ../../../../../../../../pinocchio/pkg/persistence/chatstore/block_hash.go
      Note: Content hash includes metadata, explaining metadata-churn duplicate rows.
    - Path: ../../../../../../../../pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go
      Note: Pinocchio SQLite turns snapshot schema and block membership export.
    - Path: ../../../../../../../geppetto/pkg/events/correlation.go
      Note: Canonical correlation contract that postdates first go-minitrace turnsdb support.
    - Path: ../../../../../../../geppetto/pkg/turns/helpers_blocks.go
      Note: Geppetto constructors showing UUID block IDs and tool payload IDs.
    - Path: ../../../../../../../geppetto/pkg/turns/types.go
      Note: Geppetto Block model with stable ID field.
    - Path: ../../../../../../../pinocchio/pkg/persistence/chatstore/block_hash.go
      Note: Content hash includes metadata
    - Path: ../../../../../../../pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go
      Note: SQLite turns export with block_id/content_hash/ordinal.
    - Path: pkg/adapters/turnsdb/convert.go
      Note: |-
        Current go-minitrace turnsdb converter and delta accumulation implementation.
        Current converter delta logic to simplify using semantic block identity.
    - Path: pkg/adapters/turnsdb/convert_test.go
      Note: |-
        Current regression tests for turnsdb conversion behavior.
        Regression-test target for semantic delta accumulation.
ExternalSources: []
Summary: Research guide on how Geppetto/Pinocchio turns identity evolved and how go-minitrace should simplify delta handling using stable block/tool identities.
LastUpdated: 2026-05-12T00:00:00Z
WhatFor: Guide follow-up work to simplify go-minitrace turnsdb delta accumulation and eventually support ordered transcript events.
WhenToUse: Read before changing go-minitrace turnsdb conversion, Pinocchio turns SQLite export, or Geppetto block/correlation identity contracts.
---


# Geppetto Turns Identity and Minitrace Delta Handling Guide

## Executive summary

The first go-minitrace turnsdb converter predates Geppetto's current robust typed event correlation contract, but it does **not** predate the existence of block IDs in the Geppetto turns model. Current Geppetto and Pinocchio already provide enough persisted identity and ordering information for go-minitrace to simplify and improve its delta accumulation logic without changing Geppetto first.

The main follow-up should be in go-minitrace: replace metadata-sensitive block fingerprinting and two-pass tool/text conversion with a semantic block identity strategy that prefers stable `block_id`, tool call IDs, and snapshot ordinal. Exact text/tool/text interleaving may require a richer minitrace transcript-event model, but source data currently appears sufficient to build one.

## Why this research was needed

The GMT-008 converter fix removed immediate symptoms in Coinvault-derived minitrace archives:

- top-level duplicate tool calls collapsed from 12 to 6 for representative session `8730...`,
- all 6 tool calls are successful,
- all 6 are linked from a transcript turn,
- raw `{"text":"\n"}` / `{"text":""}` artifacts are gone.

However, the converter still attaches all tools in the representative delta to one assistant turn. The remaining question was whether Geppetto/Pinocchio needs a DB export change to expose better ordering and identity, or whether go-minitrace should make better use of existing data.

## Timeline findings

### go-minitrace turnsdb support

The initial go-minitrace turnsdb converter landed in:

```text
387a967 2026-04-01 09:50:58 -0400 Add ChatGPT transcript and turnsdb conversion support
```

This is before the robust typed event correlation work in Geppetto.

### Geppetto canonical event correlation

Geppetto added canonical typed correlation in:

```text
efc38756 2026-05-08 05:47:51 -0400 Add canonical event correlation types
```

File:

```text
geppetto/pkg/events/correlation.go
```

Current contract:

```go
type Correlation struct {
    SessionID   string
    RunID       string
    InferenceID string
    TurnID      string

    ProviderCallID    string
    ProviderCallIndex int32
    Provider          string
    Model             string
    ResponseID        string

    ItemID            string
    OutputIndex       *int32
    SummaryIndex      *int32
    ChoiceIndex       *int32
    ContentBlockIndex *int32

    SegmentID    string
    SegmentIndex int32
    SegmentType  string
    StreamKind   string

    ToolCallID    string
    ToolCallIndex *int32

    CorrelationKey       string
    ParentCorrelationKey string
}
```

Conclusion: the user's hypothesis is correct for canonical correlation IDs. The first turnsdb converter was built before this robust correlation vocabulary existed.

### Geppetto block IDs

Current Geppetto block model:

```text
geppetto/pkg/turns/types.go
```

```go
type Block struct {
    ID      string
    Kind    BlockKind
    Role    string
    Payload map[string]any
    Metadata BlockMetadata
}
```

Block IDs existed before the first go-minitrace turnsdb converter. Historical notes:

```text
b0894e3c 2025-08-10 17:43:12 -0400 First pass at refactoring to have Turn
45f98520 2025-08-11 02:45:38 -0400 Remove order from blocks
8273a49e 2025-08-13 13:53:36 -0400 Reformat all the files
```

The initial turns model included `Block.ID` and `Block.Order`. `Block.Order` was removed, leaving ordering as slice order and, later, DB membership ordinal. Constructors gained UUID IDs for normal text/system/user/tool-use blocks early in the turns model lifetime.

Current constructors:

```text
geppetto/pkg/turns/helpers_blocks.go
```

```go
func NewAssistantTextBlock(text string) Block {
    return Block{
        ID:      uuid.NewString(),
        Kind:    BlockKindLLMText,
        Role:    RoleAssistant,
        Payload: map[string]any{PayloadKeyText: text},
    }
}

func NewToolCallBlock(id string, name string, args any) Block {
    return Block{
        ID:   id,
        Kind: BlockKindToolCall,
        Payload: map[string]any{
            PayloadKeyID:   id,
            PayloadKeyName: name,
            PayloadKeyArgs: args,
        },
    }
}

func NewToolUseBlockWithError(id string, result any, err string) Block {
    return Block{
        ID:   uuid.NewString(),
        Kind: BlockKindToolUse,
        Payload: map[string]any{
            PayloadKeyID:     id,
            PayloadKeyResult: result,
            PayloadKeyError:  err,
        },
    }
}
```

Conclusion: go-minitrace can and should use existing block IDs more directly.

## Current Pinocchio SQLite turns export

The SQLite turns snapshot export lives in Pinocchio:

```text
pinocchio/pkg/persistence/chatstore/turn_store_sqlite.go
```

It persists blocks separately from snapshot membership:

```sql
CREATE TABLE IF NOT EXISTS blocks (
    block_id TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    hash_algorithm TEXT NOT NULL DEFAULT 'sha256-canonical-json-v1',
    kind TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    block_metadata_json TEXT NOT NULL DEFAULT '{}',
    first_seen_at_ms INTEGER NOT NULL,
    PRIMARY KEY (block_id, content_hash)
);

CREATE TABLE IF NOT EXISTS turn_block_membership (
    conv_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    phase TEXT NOT NULL,
    snapshot_created_at_ms INTEGER NOT NULL,
    ordinal INTEGER NOT NULL,
    block_id TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    PRIMARY KEY (conv_id, session_id, turn_id, phase, snapshot_created_at_ms, ordinal),
    FOREIGN KEY (block_id, content_hash) REFERENCES blocks(block_id, content_hash)
);
```

Snapshot storage loops over full ordered turn blocks:

```go
for i, block := range t.Blocks {
    blockID := normalizeBlockID(block.ID, turnID, i)
    ...
    contentHash, err := ComputeBlockContentHash(block.Kind.String(), block.Role, payloadMap, blockMetadata)
    ...
    INSERT INTO blocks(block_id, content_hash, ...)
    INSERT OR REPLACE INTO turn_block_membership(... ordinal, block_id, content_hash)
}
```

Fallback block identity:

```go
func normalizeBlockID(blockID string, turnID string, ordinal int) string {
    id := strings.TrimSpace(blockID)
    if id != "" {
        return id
    }
    return fmt.Sprintf("%s#%d", strings.TrimSpace(turnID), ordinal)
}
```

Conclusion: current DB export gives go-minitrace the important pieces:

1. `block_id` for stable block identity when Geppetto supplied one.
2. `content_hash` for exact content-version identity.
3. `ordinal` for ordering within each full snapshot.
4. `phase` and `snapshot_created_at_ms` for choosing canonical snapshots.

## Why GMT-008 saw duplicate tool calls

Pinocchio content hashes include metadata:

```text
pinocchio/pkg/persistence/chatstore/block_hash.go
```

```go
// The canonical material is JSON over:
//   - kind
//   - role
//   - payload
//   - metadata
func ComputeBlockContentHash(kind, role string, payload, metadata map[string]any) (string, error) {
    b, err := CanonicalBlockMaterialJSON(kind, role, payload, metadata)
    ...
}
```

This is valid for exact row identity, but it is too sensitive for semantic transcript identity. In the Coinvault session, repeated `tool_call` blocks had stable payload/tool IDs but changed metadata. That produced different `content_hash` values and caused go-minitrace's old LCS fingerprint to treat old tool calls as new.

GMT-008 fixed the immediate issue in go-minitrace by ignoring metadata for tool-call/tool-use delta fingerprints and by deduplicating top-level tool calls by ID.

## What `deltaBlocks` are

The DB stores full ordered snapshots. `deltaBlocks` are not a Geppetto export format.

In go-minitrace:

```go
deltaBlocks := lcsDelta(previousBlocks, snapshot.Blocks)
previousBlocks = snapshot.Blocks
```

So `deltaBlocks` means:

> the blocks in this full cumulative snapshot that are not already accounted for by the previous full snapshot.

This is necessary because snapshots are cumulative. Blindly emitting every full snapshot would duplicate the transcript repeatedly.

The remaining issue is not that go-minitrace computes deltas. The issue is that the converter currently does not use the strongest available semantic identity and still processes text turns and tool calls in separate conceptual passes.

## Recommended go-minitrace direction

### 1. Introduce a strict semantic block identity helper

Add a helper in the turnsdb adapter that treats identity as a hard contract, not a fallback chain:

```go
func semanticBlockKey(block Block) (string, error) {
    switch block.Kind {
    case "tool_call", "tool_use":
        toolID := strings.TrimSpace(stringValue(block.Payload["id"]))
        if toolID == "" {
            return "", fmt.Errorf("%s block %q missing payload id", block.Kind, block.ID)
        }
        return block.Kind + ":" + toolID, nil
    default:
        blockID := strings.TrimSpace(block.ID)
        if blockID == "" {
            return "", fmt.Errorf("%s block missing block_id", block.Kind)
        }
        return block.Kind + ":" + block.Role + ":" + blockID, nil
    }
}
```

The key point: separate semantic identity from exact content version, and fail when the source/export does not provide identity. Do not guess from text payloads or metadata. Pinocchio already normalizes missing source block IDs to `turnID#ordinal` at SQLite export time, so go-minitrace does not need a second legacy fallback layer.

### 2. Keep exact content/version tracking separately

For repeated semantic keys, the converter should decide whether the new row is:

- a duplicate carried-forward block,
- a content update to an in-progress block,
- a finalization of a previous block,
- or a true new transcript event.

This avoids forcing `content_hash` to do both semantic identity and exact-version identity.

### 3. Keep LCS only where it helps

The existing LCS delta logic helps when framework/control blocks disappear or snapshots are not pure append-only suffixes. But LCS should compare semantic keys, not metadata-sensitive full fingerprints.

Alternative: maintain a `seenSemanticKeys` set and scan each snapshot in ordinal order. This is simpler, but may behave worse when blocks are removed/reordered. A hybrid is probably best:

- LCS over semantic keys for carried-forward matching.
- Version/update merge logic for blocks with the same semantic key but changed content.

### 4. Process delta blocks in source order once

The converter should avoid the text-first/tools-second shape. Preferred shape:

```text
for block in deltaBlocks ordered by source ordinal:
  if reasoning:
    accumulate thinking
  if llm_text/user/system:
    emit or update transcript turn
  if tool_call:
    create/merge tool call
    attach to current or nearest assistant turn
  if tool_use:
    complete matching tool call
```

This would better preserve the source interleaving already present in `turn_block_membership.ordinal`.

### 5. Decide whether minitrace needs ordered transcript events

`turns[] + ToolCallsInTurn` can show that a turn has tools, but it cannot perfectly express:

```text
assistant text A
tool call 1
tool result 1
assistant text B
tool call 2
tool result 2
assistant text C
```

If the UI needs exact placement, add an ordered transcript-event API/model in go-minitrace rather than changing Geppetto first.

## Do we need Geppetto or Pinocchio changes?

Not first.

Current source data already includes stable block IDs, payload tool IDs, and ordered membership rows. go-minitrace should consume that better before changing producers.

Potential future producer improvement:

- Add `semantic_hash` or `stable_block_key` to Pinocchio's `blocks` or `turn_block_membership` export.
- Keep current `content_hash` as exact row/version identity.
- Compute `semantic_hash` without volatile metadata.

This would make downstream consumers simpler, but it is not required for the next go-minitrace improvement.

## Practical follow-up tasks

1. Refactor go-minitrace turnsdb delta matching to use strict semantic block keys.
2. Add tests for missing block IDs / missing tool payload IDs and metadata-only block version changes.
3. Add tests for repeated snapshots with removed control blocks and stable block IDs.
4. Add tests for ordered text/tool/text/tool/text source blocks.
5. Refactor conversion to process delta blocks in one ordered pass.
6. Decide whether `ToolCallsInTurn` is enough or whether `/blocks` should consume an ordered transcript-event stream.
7. Consider optional Pinocchio `semantic_hash` only after go-minitrace consumes current identity fields well.

## Conclusion

The first go-minitrace converter predates robust Geppetto event correlation, but current persisted turns snapshots already carry enough identity and ordering for a much cleaner converter. The next architectural improvement should be in go-minitrace: use semantic block identity based on `block_id` and tool IDs, separate identity from content version, and process ordered deltas in one pass.
