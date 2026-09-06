// history file-history — when was a file created / edited / read, in which
// session, turn, and at what timestamp, across all converted archives.
//
// Codex uses only normalized structural rows from files, including evidence
// status and independent file outcome. Never scan its JavaScript wrappers or
// quoted arguments. Older Codex archives need reconversion for this ledger.
// Existing non-Codex scalar/argument extraction remains supported below.
// A NEW classification is a target operation, not proof of prior nonexistence;
// evidence_status distinguishes attempted targets from confirmed effects.

const openDb = function() {
  const mt = require("minitrace");
  return mt.db().RuntimeArchives().QueryCommandDefaults()
    // CellChars must exceed the arguments_json substr window below, otherwise
    // the runtime truncates the payload before extractCandidatePaths ever sees
    // it and every file after the truncation point in a multi-file patch is
    // lost — the exact bug this command exists to avoid.
    .Limits(mt.limits().Rows(200000).CellChars(100000).Build())
    .Build();
}

// Legacy non-Codex command extraction. Codex never enters this inference path.
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

// Shell redirection targets (`cat >> path/to/file <<'EOF'`, `echo x > path`).
// Structural, not a substring search: a redirect target is always a single
// whitespace-delimited token immediately after `>`/`>>` in valid shell
// syntax, so this cannot match free-text prose the way a blanket search
// over the whole command string could.
// `2>&1` and `>&2` are file-descriptor duplications, not paths; `>(cmd)` is a
// process substitution. None of them name a file, so they are skipped.
const extractRedirectTargets = function(cmdText) {
  const out = [];
  const re = />>?\s*(\S+)/g;
  let m;
  while ((m = re.exec(cmdText)) !== null) {
    const t = m[1];
    if (t.startsWith("&") || t.startsWith("(")) continue;
    out.push(t);
  }
  return out;
};

// Some adapters normalize the home directory to `~` in tc.file_path while
// the raw arguments_json payload keeps the absolute path (or vice versa) —
// without normalizing, both forms of the same file independently satisfy a
// fragment match and produce two timeline rows for one tool call.
const canonicalizePath = function(p) {
  if (!p) return p;
  if (p.startsWith("~/")) return p.slice(2);
  const m = p.match(/^\/home\/[^/]+\/(.*)$/);
  return m ? m[1] : p;
};

// Structurally file-path-shaped candidates from a tool call: the file_path
// column itself, every Codex apply_patch "*** Update/Add/Delete File:"
// header (there can be several in one multi-file patch — matches the exact
// three prefixes extractFilePathFromPatch recognizes, in the same order),
// and any JSON file_path/path key found in arguments_json (other tool
// call shapes that carry a target path outside the patch-header format).
// Deliberately does NOT do a raw substring search over the whole payload —
// that would match prose that merely mentions a path-like string.
// Deduplicated by canonical (home-normalized) form, file_path column first,
// so the adapter-normalized representation wins when both are present.
const extractCandidatePaths = function(row) {
  const paths = [];
  if (row.file_path) paths.push(row.file_path);
  const aj = row.arguments_json || "";
  const patchRe = /\*\*\* (?:Update|Add|Delete) File:\s*(.+?)(?:\\n|\n|$)/g;
  let m;
  while ((m = patchRe.exec(aj)) !== null) {
    const p = m[1].trim();
    if (p) paths.push(p);
  }
  const jsonPathRe = /\\?"(?:file_path|path)\\?"\s*:\s*\\?"([^"\\]+)\\?"/g;
  while ((m = jsonPathRe.exec(aj)) !== null) {
    if (m[1]) paths.push(m[1]);
  }
  for (const cmd of effectiveCommands(row)) {
    for (const t of extractRedirectTargets(cmd)) paths.push(t);
  }
  const seen = new Set();
  const out = [];
  for (const p of paths) {
    const canon = canonicalizePath(p);
    if (seen.has(canon)) continue;
    seen.add(canon);
    out.push(p);
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
             CASE WHEN s.agent_framework='codex' THEN f.operation_type ELSE tc.operation_type END AS operation_type,
             CASE WHEN s.agent_framework='codex' THEN f.success ELSE tc.success END AS success,
             CASE WHEN s.agent_framework='codex' THEN f.path ELSE tc.file_path END AS file_path,
             f.evidence_kind, f.evidence_status, f.cwd AS evidence_cwd, f.source_reference,
             substr(COALESCE(tc.command,''),1,2000) AS command,
             substr(COALESCE(tc.arguments_json,''),1,64000) AS arguments_json
      FROM tool_calls tc
      JOIN sessions s USING (session_id)
      LEFT JOIN files f ON s.agent_framework='codex' AND f.session_id=tc.session_id AND f.tool_call_id=tc.tool_call_id
      WHERE (s.agent_framework='codex' AND f.evidence_kind!='legacy_scalar' AND f.path LIKE ${like})
         OR (COALESCE(s.agent_framework,'')!='codex' AND (COALESCE(tc.file_path,'') LIKE ${like}
         OR COALESCE(tc.command,'') LIKE ${like}
         OR COALESCE(tc.arguments_json,'') LIKE ${like}))
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
      if (turn == null) return null;
      const list = bySession[sid];
      if (!list) return null;
      let best = null;
      for (const u of list) {
        if (u.turn_index <= turn) best = u; else break;
      }
      return best ? best.content : null;
    }

    // Every genuinely file-path-shaped candidate that matches the search
    // fragment becomes its own timeline row. A single multi-file patch that
    // touches both a matching and a non-matching file yields one row (for
    // the matching file only); a patch touching two matching files yields
    // two rows, correctly attributing the tool call to both.
    const fragLower = frag.toLowerCase();
    const timeline = [];
    for (const r of rows) {
      const candidates = r.framework === "codex" ? [r.file_path] : extractCandidatePaths(r);
      const matched = candidates.filter((p) => p.toLowerCase().includes(fragLower));
      if (!matched.length) continue; // coarse SQL LIKE hit arguments_json prose, not a real path
      for (const path of matched) {
        timeline.push({
          session_id: r.session_id,
          framework: r.framework,
          working_directory: r.working_directory,
          turn_index: r.turn_index,
          timestamp: r.timestamp,
          tool_name: r.tool_name,
          op: classifyOp(r),
          raw_operation_type: r.operation_type,
          success: r.success,
          file_path: path,
          match_source: r.framework === "codex" ? "structural_file_target" : path === r.file_path ? "file_path" : "arguments_json-extracted-path",
          evidence_kind: r.evidence_kind,
          evidence_status: r.evidence_status,
          evidence_cwd: r.evidence_cwd,
          source_reference: r.source_reference,
          preceding_instruction: precedingInstruction(r.session_id, r.turn_index),
        });
      }
    }

    // Group into a per-distinct-path summary. The key is the *canonical*
    // (home-normalized) path: per-row dedup only collapses `~/x` and
    // `/home/me/x` within one tool call, so without canonicalizing here the
    // same file still splits into two summary groups when different rows
    // happen to record it in different forms.
    const groups = {};
    for (const t of timeline) {
      const key = canonicalizePath(t.file_path) || "(unresolved)";
      const g = groups[key] || (groups[key] = {
        file_path: key, first_seen: t.timestamp, first_op: t.op, first_session: t.session_id,
        last_seen: t.timestamp, creates: 0, modifies: 0, reads: 0, other: 0,
        sessions: new Set(), structural: false, attempted_targets: 0, confirmed_effects: 0,
      });
      if (t.timestamp < g.first_seen) { g.first_seen = t.timestamp; g.first_op = t.op; g.first_session = t.session_id; }
      if (t.timestamp > g.last_seen) g.last_seen = t.timestamp;
      g.sessions.add(t.session_id);
      g.structural = g.structural || t.framework === "codex";
      if (t.evidence_status === "attempted") g.attempted_targets++;
      if (t.evidence_status === "confirmed" && t.success === 1) g.confirmed_effects++;
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
      attempted_targets: g.attempted_targets, confirmed_effects: g.confirmed_effects,
      created_before_visible_history: g.structural ? null : !g.first_op.startsWith("NEW"),
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
