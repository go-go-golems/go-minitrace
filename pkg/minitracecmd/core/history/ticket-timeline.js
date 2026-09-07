// history ticket-timeline — when was a docmgr ticket created, and when were
// its tasks.md/changelog.md/diary files touched, across all converted
// archives.
//
// See design-doc/01 (GOGO-MINITRACE-HISTORY-VERBS-2026-07-20). Two channels:
// (1) shell/exec commands mentioning both "docmgr" and the ticket fragment
// (classified by docmgr subcommand), (2) any tool call whose file_path falls
// under the ticket's ttmp directory (classified by filename). A "task"/
// "changelog" classification means the command ran or the file was touched —
// not that the edit succeeded; verify against the actual file/git history.

const openDb = function() {
  const mt = require("minitrace");
  return mt.db().RuntimeArchives().QueryCommandDefaults()
    .Limits(mt.limits().Rows(200000).CellChars(4000).Build())
    .Build();
}

const effectiveCommands = function(row) {
  if (row.framework === "codex") return row.command ? [row.command] : [];
  if (row.command && row.command.length) return [row.command];
  const aj = row.arguments_json || "";
  const out = [];
  const re = /(?:\\?"?cmd\\?"?|\\?"command\\?")\s*:\s*\\?"((?:[^"\\]|\\.)*)\\?"/g;
  let m;
  while ((m = re.exec(aj)) !== null) {
    out.push(m[1].replace(/\\n/g, "\n").replace(/\\"/g, '"').slice(0, 500));
  }
  return out;
};

// docmgr's live subcommand surface, confirmed via `docmgr ticket --help` /
// `docmgr doc --help` while designing this verb (2026-07-20). If docmgr's
// CLI surface changes, unmatched commands fall back to 'other-docmgr'
// instead of erroring.
const classifyDocmgrCommand = function(cmd) {
  const c = cmd.toLowerCase();
  if (!c.includes("docmgr")) return null;
  if (/\bticket\s+create\b/.test(c)) return "ticket-create";
  if (/\bticket\s+close\b/.test(c)) return "ticket-close";
  if (/\btask\b/.test(c)) return "task";
  if (/\bchangelog\b/.test(c)) return "changelog";
  if (/\bdoc\s+add\b/.test(c)) return "doc-add";
  if (/\bdoc\s+relate\b/.test(c)) return "doc-relate";
  return "other-docmgr";
}

const classifyTicketPath = function(path) {
  const p = (path || "").toLowerCase();
  if (p.endsWith("/tasks.md")) return "task";
  if (p.endsWith("/changelog.md")) return "changelog";
  if (p.includes("diary")) return "diary";
  if (p.endsWith(".md")) return "doc";
  return "other-file";
}

__section__("tickettimelineopts", {
  fields: {
    ticket: { type: "string", help: "Ticket ID or slug fragment, e.g. DIARY-MINING-2026-07-19", required: true },
    limit: { type: "int", default: 500, help: "Max timeline rows" },
  },
});

function ticketTimeline(tickettimelineopts) {
  const mt = require("minitrace");
  const frag = tickettimelineopts.ticket;
  if (!frag) throw new Error("--ticket is required");
  const like = mt.sql.like(frag);
  const db = openDb();
  try {
    const cmdRows = db.query(`
      SELECT tc.session_id, s.agent_framework AS framework, emitting_turn_index AS turn_index, timestamp, tool_name,
             substr(COALESCE(command,''),1,400) AS command,
             substr(COALESCE(arguments_json,''),1,2000) AS arguments_json
      FROM tool_calls tc JOIN sessions s USING (session_id)
      WHERE ((s.agent_framework='codex' AND tc.record_kind='execution') OR (COALESCE(s.agent_framework,'')!='codex' AND tool_name IN ('Bash','bash','shell','exec','run_terminal_cmd')))
        AND (COALESCE(command,'') LIKE '%docmgr%' OR COALESCE(arguments_json,'') LIKE '%docmgr%')
        AND (COALESCE(command,'') LIKE ${like} OR COALESCE(arguments_json,'') LIKE ${like})
      ORDER BY timestamp
      LIMIT ${tickettimelineopts.limit || 500}
    `);

    const fileRows = db.query(`
      SELECT tc.session_id, emitting_turn_index AS turn_index, timestamp, tc.tool_name,
             f.operation_type, f.path AS file_path, f.evidence_status, f.success
      FROM files f JOIN tool_calls tc ON f.session_id=tc.session_id AND f.tool_call_id=tc.tool_call_id
      JOIN sessions s ON s.session_id=tc.session_id
      WHERE f.path LIKE ${like} AND (COALESCE(s.agent_framework,'')!='codex' OR f.evidence_kind!='legacy_scalar')
      ORDER BY timestamp
      LIMIT ${tickettimelineopts.limit || 500}
    `);

    const events = [];
    for (const r of cmdRows) {
      for (const cmd of effectiveCommands(r)) {
        const cat = classifyDocmgrCommand(cmd);
        if (!cat) continue;
        events.push({
          session_id: r.session_id, turn_index: r.turn_index, timestamp: r.timestamp,
          channel: "command", category: cat, tool_name: r.tool_name,
          evidence_kind: "command_text_candidate", verified_subcommand_execution: false,
          detail: cmd.slice(0, 240),
        });
      }
    }
    for (const r of fileRows) {
      events.push({
        session_id: r.session_id, turn_index: r.turn_index, timestamp: r.timestamp,
        channel: "file", category: classifyTicketPath(r.file_path), tool_name: r.tool_name,
        evidence_status: r.evidence_status, success: r.success,
        detail: `${r.operation_type || "?"} ${r.file_path}`,
      });
    }
    events.sort((a, b) => (a.timestamp || "").localeCompare(b.timestamp || ""));

    const byCategory = function(cat) { return events.filter((e) => e.category === cat); }
    const ticketCreateEvents = byCategory("ticket-create");
    // Fallback creation candidate: the chronologically-first file event for
    // this ticket, when no explicit `docmgr ticket create` command was
    // captured in this archive's window (e.g. it ran in an earlier,
    // unconverted session).
    const firstFileEvent = events.find((e) => e.channel === "file") || null;

    return {
      query_ticket: frag,
      timeline: events,
      summary: {
        first_event: events[0] || null,
        ticket_create_events: ticketCreateEvents,
        ticket_create_fallback_candidate: ticketCreateEvents.length ? null : firstFileEvent,
        task_edits: byCategory("task"),
        changelog_edits: byCategory("changelog"),
        diary_edits: byCategory("diary"),
        doc_add_events: byCategory("doc-add"),
        doc_relate_events: byCategory("doc-relate"),
        other_docmgr_events: byCategory("other-docmgr"),
        sessions_touching_ticket: [...new Set(events.map((e) => e.session_id))],
        event_count: events.length,
      },
    };
  } finally {
    db.close();
  }
}

__verb__("ticketTimeline", {
  name: "ticket-timeline",
  short: "When was a docmgr ticket created and its tasks/changelog/diary touched",
  fields: { tickettimelineopts: { bind: "tickettimelineopts" } },
});
