// history file-history — when was a file created / edited / read, in which
// session, turn, and at what timestamp, across all converted archives.
//
// See design-doc/01 (GOGO-MINITRACE-HISTORY-VERBS-2026-07-20) for the full
// design rationale. Short version: `tool_calls.file_path` is the primary
// signal; Codex patch wrappers sometimes leave file_path empty and put the
// target path only in `arguments_json`, so that is matched too. A "created"
// classification is a candidate, never proof the file didn't already exist
// before this archive's window — see `created_before_visible_history`.

const openDb = function() {
  const mt = require("minitrace");
  return mt.db().RuntimeArchives().QueryCommandDefaults()
    .Limits(mt.limits().Rows(200000).CellChars(4000).Build())
    .Build();
}

// Cross-framework command extraction: claude-code populates `command`;
// pi uses lowercase tool names with `command`; codex buries shell commands
// as JS strings (tools.exec_command({cmd: "..."})) inside arguments_json.
const effectiveCommands = function(row) {
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

const isRealInstruction = function(content) {
  const c = (content || "").trim();
  if (!c) return false;
  if (c.startsWith("<")) return false;
  if (c.startsWith("[Image")) return false;
  if (c.startsWith("Base directory for this skill")) return false;
  if (c.startsWith("This session is being continued")) return false;
  if (c.startsWith("Caveat:")) return false;
  return true;
}

// operation_type is authoritative when present (NEW/MODIFY/READ/OTHER).
// When OTHER or empty (common for Codex exec/patch wrappers), fall back to
// a tool-name guess. This is a weaker signal and is reported as such.
const classifyOp = function(row) {
  const known = { NEW: 1, MODIFY: 1, READ: 1 };
  if (row.operation_type && known[row.operation_type]) return row.operation_type;
  const tn = (row.tool_name || "").toLowerCase();
  if (tn === "read") return "READ-inferred";
  if (tn === "write") return "NEW-or-MODIFY-inferred";
  if (tn === "edit" || tn === "multiedit" || tn === "notebookedit") return "MODIFY-inferred";
  if (tn === "exec" || tn === "bash" || tn === "shell") return "OTHER-exec";
  return row.operation_type || "OTHER";
}

__section__("filehistoryopts", {
  fields: {
    path: { type: "string", help: "File path fragment to match (LIKE %fragment%), e.g. vocabulary.yaml or pkg/scripting/engine.go", required: true },
    limit: { type: "int", default: 500, help: "Max timeline rows" },
  },
});

function fileHistory(filehistoryopts) {
  const mt = require("minitrace");
  const frag = filehistoryopts.path;
  if (!frag) throw new Error("--path is required");
  const like = mt.sql.like(frag);
  const db = openDb();
  try {
    const rows = db.query(`
      SELECT tc.session_id, s.agent_framework AS framework, s.working_directory,
             s.started_at AS session_started_at,
             tc.emitting_turn_index AS turn_index, tc.timestamp, tc.tool_name,
             tc.operation_type, tc.success, tc.file_path,
             substr(COALESCE(tc.command,''),1,300) AS command,
             substr(COALESCE(tc.arguments_json,''),1,1500) AS arguments_json
      FROM tool_calls tc
      JOIN sessions s USING (session_id)
      WHERE COALESCE(tc.file_path,'') LIKE ${like}
         OR (COALESCE(tc.file_path,'') = '' AND COALESCE(tc.arguments_json,'') LIKE ${like})
      ORDER BY tc.timestamp
      LIMIT ${filehistoryopts.limit || 500}
    `);

    // Preceding real user instruction, per session, for context enrichment.
    const userTurns = db.query(`
      SELECT session_id, turn_index, substr(COALESCE(content,''),1,200) AS content
      FROM turns
      WHERE role = 'user' AND COALESCE(content_type,'') != 'tool_result'
      ORDER BY session_id, turn_index
    `);
    const bySession = {};
    for (const u of userTurns) {
      if (!isRealInstruction(u.content)) continue;
      (bySession[u.session_id] = bySession[u.session_id] || []).push(u);
    }
    const precedingInstruction = function(sid, turn) {
      const list = bySession[sid];
      if (!list) return null;
      let best = null;
      for (const u of list) {
        if (u.turn_index <= turn) best = u; else break;
      }
      return best ? best.content : null;
    }

    const timeline = rows.map((r) => {
      const hasFilePath = r.file_path && r.file_path.length;
      const cmds = effectiveCommands(r);
      const resolvedPath = hasFilePath ? r.file_path
        : (cmds.find((c) => c.includes(frag)) || "").slice(0, 300);
      return {
        session_id: r.session_id,
        framework: r.framework,
        working_directory: r.working_directory,
        turn_index: r.turn_index,
        timestamp: r.timestamp,
        tool_name: r.tool_name,
        op: classifyOp(r),
        raw_operation_type: r.operation_type,
        success: r.success,
        file_path: resolvedPath || r.file_path,
        match_source: hasFilePath ? "file_path" : "arguments_json-fallback",
        preceding_instruction: precedingInstruction(r.session_id, r.turn_index),
      };
    });

    // Group into a per-distinct-path summary. Group key: exact file_path when
    // present, else the resolved arguments_json-derived path.
    const groups = {};
    for (const t of timeline) {
      const key = t.file_path || "(unresolved)";
      const g = groups[key] || (groups[key] = {
        file_path: key, first_seen: t.timestamp, first_op: t.op, first_session: t.session_id,
        last_seen: t.timestamp, creates: 0, modifies: 0, reads: 0, other: 0,
        sessions: new Set(),
      });
      if (t.timestamp < g.first_seen) { g.first_seen = t.timestamp; g.first_op = t.op; g.first_session = t.session_id; }
      if (t.timestamp > g.last_seen) g.last_seen = t.timestamp;
      g.sessions.add(t.session_id);
      if (t.op.startsWith("NEW")) g.creates++;
      else if (t.op.startsWith("MODIFY")) g.modifies++;
      else if (t.op.startsWith("READ")) g.reads++;
      else g.other++;
    }
    const summary = Object.values(groups).map((g) => ({
      file_path: g.file_path,
      first_seen: g.first_seen,
      first_op: g.first_op,
      first_session: g.first_session,
      last_seen: g.last_seen,
      creates: g.creates, modifies: g.modifies, reads: g.reads, other: g.other,
      sessions: [...g.sessions],
      created_before_visible_history: !g.first_op.startsWith("NEW"),
    })).sort((a, b) => (a.first_seen || "").localeCompare(b.first_seen || ""));

    return { query_path: frag, timeline, summary };
  } finally {
    db.close();
  }
}

__verb__("fileHistory", {
  name: "file-history",
  short: "When was a file created/edited/read — timeline + per-file summary across sessions",
  fields: { filehistoryopts: { bind: "filehistoryopts" } },
});
