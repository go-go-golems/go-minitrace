# Codex fidelity and evidence contract

The adapter supports persisted session JSONL, legacy rollout JSONL and exec JSONL. Reconvert older archives to populate the new structural evidence ledger; native sources are never modified.

## Messages and identity

Paginated UserMessage/AgentMessage items and response messages are reconciled by native identity. Adjacent complementary mirrors may reconcile only with the same nonempty native turn and exact native text. Repeated messages remain distinct. Source lines survive reconciliation. Tool emitters are null unless an explicit identity establishes an assistant message; shared time/turn context is not parent proof.

Native CommandExecution start/completion notifications reconcile by ID, not command text. Missing IDs use collision-safe source-line identities and diagnostics. Separate native IDs remain separate even with identical commands. Direct response calls are enriched only with one-to-one explicit linkage. Opaque JavaScript, quoted histories and false branches are never evaluated or scanned into execution records.

## Outcomes, outputs and provenance

`output.success` is nullable. `status` is unknown, pending, cancelled, succeeded or failed. Command completion without a numeric exit is not success. Conflicting terminal evidence stays unknown, including after replay. Native FileChange completion represents applied file effects; failed/cancelled groups do not establish every target's outcome.

Typed text/image blocks are decoded before truncation; images become signals, not copied base64. Only recognized transport envelopes supply exit/duration metadata. Native stdout is never reparsed as a transport envelope. Child results do not become orchestration success. Source references, full bytes and SHA-256 describe pre-truncation output. Execution argv/cwd, native IDs, lifecycle sources and explicit-or-unknown parentage remain in framework metadata and API provenance.

## File evidence

`input.file_targets` is the ledger; `file_path` is only its first-target convenience. Each target has operation, evidence kind, attempted/confirmed status, independent nullable success, native path, cwd, lexical resolution and source reference. Empty lists forbid fallback inference.

- Direct patch Add/Update/Delete/Move headers establish attempts.
- Native FileChange maps establish observed effects; rename source and destination both appear. Replayed events are deduplicated; conflicting contents/statuses retain uncertainty and source hashes.
- Literal shell `<`, `>` and `>>` are attempts, never confirmed by exit 0. The bounded grammar supports straight-line commands, quotes, escapes and comments. It deliberately rejects expansions, compound control flow, pipelines, descriptor duplication, heredocs, process substitutions and cwd-changing/eval/source constructs with diagnostics. The script limit is 256 KiB; patch analysis limit is 1 MiB.
- Relative paths resolve only against explicit tool/execution cwd, never final/session cwd. `resolved` means lexical absolute resolution, not filesystem/symlink canonicalization. Unknown cwd leaves paths relative.
- Search operands and opaque wrapper source are not file reads/writes. Existing non-Codex scalar reporting is retained as `legacy_scalar` / `reported`.

## Counts and consumers

`tool_call_count` remains total normalized records. `tool_call_record_count`, `orchestration_count`, `execution_record_count` and `file_change_count` partition them. `model_invocation_count` includes ordinary/orchestration calls and explicitly enriched direct invocations, not every native child. `file_touch_count` counts target evidence rows, not unique paths; `confirmed_file_target_count` counts explicitly confirmed successful effects. Wrapper records do not inflate execution operation counts.

SQLite schema v5 projects kinds, counts and all file targets. History/file presets consume structural Codex targets. Ticket command-text matches are candidates, not proof a quoted/conditional subcommand ran. Context-window queries include only tools explicitly linked into the requested message range; unknown associations are not guessed.

## Validation and limitations

Fidelity warnings are bounded and exposed in framework config, warning events and cleaning flags. Unsupported evidence is not a clean archive. Conversion receipts validate selection/identity/publication, not semantic fidelity. The archive validator may report `source-unavailable` for literal `~` paths; independently expand paths and compare hashes. Do not change native files to satisfy validation.

See ticket CODEX-FIDELITY-001 for synthetic regressions, read-only independent native audits, physical phase receipts and final smoke evidence. Verify repository commits/files independently before attributing implementation; attempts and command mentions alone are not authorship proof.
