# Investigation: What Happened to jellyfin-001 Diary Files

## Summary

The docmgr ticket `jellyfin-001` was created **3 times** during the session and the diary was written **9 times**, but the files **never persisted to disk**. The files kept disappearing between 2-12 minutes after creation. This pattern repeated 3 full cycles within the same session.

## Timeline (from go-minitrace analysis)

### Cycle 1: Turns 2-40 (01:35 - 01:48 UTC)
| Turn | Time | Tool | Action | Result |
|------|------|------|--------|--------|
| 2 | 01:35 | bash | `docmgr ticket create-ticket jellyfin-001` | ✅ "Ticket Workspace Created" |
| 3 | 01:35 | bash | `docmgr doc add --ticket jellyfin-001 --title Diary` | ✅ "Document Created" |
| 4 | 01:36 | write | `01-diary.md` (2810 bytes) | ✅ "Successfully wrote" |
| 13 | 01:37 | edit | `01-diary.md` (environment update) | ✅ "Successfully replaced" |
| 14 | 01:37 | bash | `docmgr changelog update` | ✅ "Changelog updated" |
| **40** | **01:48** | **edit** | **`01-diary.md`** | **❌ "File not found"** |
| 41 | 01:48 | bash | `find ... -name "01-diary.md"` | ❌ "(no output)" |
| 42 | 01:48 | bash | `ls ... jellyfin-001/reference/` | ❌ "No such file or directory" |
| 43 | 01:48 | bash | `docmgr ticket list --ticket jellyfin-001` | ❌ "No tickets found." |

**Gap: ~11 minutes** between last successful write (turn 14) and disappearance (turn 40).

### Cycle 2: Turns 46-111 (01:49 - 02:06 UTC)
| Turn | Time | Tool | Action | Result |
|------|------|------|--------|--------|
| 46 | 01:49 | bash | `docmgr ticket create-ticket jellyfin-001` (with `cd`) | ✅ Created |
| 47 | 01:49 | bash | `docmgr doc add --ticket jellyfin-001 --title Diary` | ✅ Created |
| 48 | 01:49 | write | `01-diary.md` (5528 bytes) | ✅ Wrote OK |
| 69 | 01:52 | edit | diary update | ✅ Replaced |
| 74 | 01:53 | edit | diary update | ✅ Replaced |
| **111** | **02:06** | **edit** | **`01-diary.md`** | **❌ "File not found"** |

**Gap: ~13 minutes** between last successful write (turn 74) and disappearance (turn 111).

### Cycle 3: Turns 255-252+ (02:22 - 02:25 UTC)
| Turn | Time | Tool | Action | Result |
|------|------|------|--------|--------|
| 228 | 02:22 | write | `01-diary.md` (3258 bytes) | ✅ Wrote OK |
| **252** | **02:25** | **edit** | **`01-diary.md`** | **❌ "File not found"** |
| 255 | 02:26 | bash | `docmgr ticket create-ticket jellyfin-001` | ✅ Created (3rd time!) |
| 257 | 02:26 | write | `01-diary.md` (4330 bytes) | ✅ Wrote OK |

**Gap: ~3 minutes** this time.

## Root Cause Analysis

### What it's NOT:
- **Not git operations**: No `git checkout`, `git clean`, or `git reset` ran between writes and disappearances
- **Not explicit deletion**: No `rm` commands targeting the ttmp directory
- **Not wrong working directory**: docmgr correctly reported `Docs root: /home/manuel/code/wesen/crib-k3s/ttmp` every time
- **Not docmgr bug in isolation**: Running `docmgr ticket create-ticket` manually right now works fine and persists

### Most Likely Cause: Pi's Bash Sandbox / Ephemeral Filesystem

The pi agent runs bash commands and write/edit operations in a sandboxed environment where:
1. The `write` tool and `edit` tool operate on a **virtual filesystem** or **overlay**
2. Changes are **not immediately synced** to the real filesystem
3. Periodic syncs or session state transitions can **discard** unsynced files
4. The `bash` tool (for docmgr) operates on the **real filesystem**, explaining why docmgr's own output showed success (it ran in the real FS) but the diary edits by the `write`/`edit` tool vanished (they were in the overlay)

**Evidence supporting this:**
- The `write` tool reported "Successfully wrote" 9 times but files weren't on disk when checked later
- The `bash` tool (real FS) could verify files existed right after docmgr created them
- But `edit` (overlay FS?) couldn't find files that `bash` (real FS) created via docmgr
- The timing gaps (3-13 minutes) suggest periodic cleanup or cache eviction

### Alternative Theory: docmgr Writes Are Reported But Not Persisted

docmgr might be reporting success without actually writing files, perhaps due to a dry-run mode or a permissions issue specific to the pi agent's execution environment.

## Impact

- **9 diary entries** were written with valuable session context but lost
- The agent spent significant time (~15 turns across 3 cycles) recreating tickets and rewriting diaries
- The compaction summary at turn 423 references `jellyfin-001` as existing, but by the current session it was gone

## Recommendations

1. **Pi agent**: Investigate whether `write`/`edit` tools persist to real filesystem or overlay
2. **docmgr**: Add a `--verify` flag that reads back the created file to confirm persistence
3. **Session workflow**: After docmgr ticket creation, immediately verify with `ls` via bash tool

## Queries Used

All SQL queries are in `scripts/`:
- `query-jellyfin-file-operations.sql` - All write/edit/read on jellyfin files
- `query-docmgr-commands.sql` - All docmgr bash commands
- `query-all-failures.sql` - All failed tool calls
- `query-jellyfin-timeline.sql` - Combined timeline of operations
- `query-gap-14-48.sql` - Operations between first write and first disappearance
- `query-gap-turns-30-50.sql` - Conversation turns during gap period
- `query-git-operations.sql` - All git operations in session
- `query-deletion-operations.sql` - Deletion-like operations
- `query-docmgr-create-results.sql` - Full docmgr create-ticket results
- `query-first-bash-calls.sql` - First bash calls to check cwd
- `query-schema.sql` - Schema discovery
- `query-tool-call-structure.sql` - Tool call JSON structure inspection
- `query-tool-name-check.sql` - Tool name type check
