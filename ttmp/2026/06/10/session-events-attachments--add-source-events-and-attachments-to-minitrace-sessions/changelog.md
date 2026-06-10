# Changelog

## 2026-06-10

- Initial workspace created


## 2026-06-10

Created events/attachments ticket, intern-oriented design guide, diary, and phased task list.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/10/session-events-attachments--add-source-events-and-attachments-to-minitrace-sessions/design-doc/01-session-events-and-attachments-design-and-implementation-guide.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/10/session-events-attachments--add-source-events-and-attachments-to-minitrace-sessions/reference/01-diary.md — Chronological implementation diary
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/10/session-events-attachments--add-source-events-and-attachments-to-minitrace-sessions/tasks.md — Sequential implementation task list


## 2026-06-10

Step 2: Added first-class Event and Attachment structs, Session fields, builders, and minitrace tests (commit d612015).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitrace/builders.go — Builder/default initialization additions
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitrace/minitrace_test.go — Validation of defaults and helpers
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitrace/schema.go — Canonical schema additions


## 2026-06-10

Step 3: Added attachments table, explicit event insertion, attachment insertion, schema version bump, and materialization tests (commit 07f4ba5).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracedb/materialize.go — Materialization path
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracedb/materialize_test.go — Coverage for new rows
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracedb/schema.go — Normalized schema additions


## 2026-06-10

Step 4: Added native JSON validation for events/attachments and updated query/adapter docs (commit adca28f).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/doc/query.md — Query documentation
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/validate/json.go — Validation logic


## 2026-06-10

Step 5: Mapped Pi non-message records to first-class source events and framework config (commit fa73b81).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/adapters/pi/convert.go — Pi adapter mappings
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/adapters/pi/convert_test.go — Pi adapter coverage


## 2026-06-10

Step 6: Mapped Claude Code lifecycle records to events and attachment records to first-class attachments (commit 673489f).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/adapters/claudecode/convert.go — Claude adapter mappings
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/adapters/claudecode/convert_test.go — Claude adapter coverage


## 2026-06-10

Step 7: Added Codex subagent/image/rate-limit events and image attachments (commit cf00d11).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/adapters/codex/convert.go — Codex adapter mappings
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/adapters/codex/convert_test.go — Codex adapter coverage


## 2026-06-10

Step 8: Extended Goja and CLI session previews with event/attachment counts and samples (commit 72dd30f).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/cmd/go-minitrace/cmds/preview/session.go — CLI output
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/import_builder.go — Preview API


## 2026-06-10

Step 9: Updated JS API preview docs and design implementation status (commit 5e24a54).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/doc/js-api-reference.md — JS API docs
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/ttmp/2026/06/10/session-events-attachments--add-source-events-and-attachments-to-minitrace-sessions/design-doc/01-session-events-and-attachments-design-and-implementation-guide.md — Ticket design guide

