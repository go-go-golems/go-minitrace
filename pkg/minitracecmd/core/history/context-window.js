// history context-window — given a session and a target turn, reconstruct
// what was "in scope": every tool call, file touched, and skill signal from
// the last compaction boundary up to that turn.
//
// See design-doc/01 (GOGO-MINITRACE-HISTORY-VERBS-2026-07-20). There is no
// single ground-truth "compaction happened here" column, so two independent
// signals are combined: (1) a continuation-summary turn ("This session is
// being continued...", direct textual marker), and (2) a >80% cache_read
// collapse between consecutive assistant API calls (api-calls.js's
// heuristic). In `auto` mode the LATER of the two candidate boundaries wins
// — the most recent compaction is what is actually still live.

const openDb = function() {
  const mt = require("minitrace");
  return mt.db().RuntimeArchives().QueryCommandDefaults()
    .Limits(mt.limits().Rows(200000).CellChars(4000).Build())
    .Build();
}

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

const parseSkillName = function(argsJson) {
  try {
    const a = JSON.parse(argsJson);
    return a.skill || a.name || null;
  } catch (e) {
    return null;
  }
}

// Collapse consecutive assistant turns sharing an (out,cr,cc) tuple within 3
// turns into one logical API call — same algorithm as docmetrics/api-calls.js,
// needed here to find the cache-read-collapse boundary.
const collapseApiCalls = function(rows) {
  const calls = [];
  let last = null;
  for (const r of rows) {
    const key = r.out + "|" + r.cr + "|" + r.cc;
    if (last && last.key === key && r.turn_index - last.end_turn <= 3) {
      last.end_turn = r.turn_index;
    } else {
      const call = { start_turn: r.turn_index, end_turn: r.turn_index, key, cr: r.cr };
      calls.push(call);
      last = call;
    }
  }
  return calls;
}

__section__("contextwindowopts", {
  fields: {
    session: { type: "string", help: "Session ID", required: true },
    turn: { type: "int", help: "Target turn_index", required: true },
    boundaryMethod: { type: "string", default: "auto", help: "auto | summary-only | cache-collapse-only" },
  },
});

function contextWindow(contextwindowopts) {
  const mt = require("minitrace");
  const sid = contextwindowopts.session;
  const target = contextwindowopts.turn;
  const method = contextwindowopts.boundaryMethod || "auto";
  if (!sid) throw new Error("--session is required");
  if (target === undefined || target === null) throw new Error("--turn is required");
  const sidLit = mt.sql.string(sid);
  const db = openDb();
  try {
    let summaryBoundary = null;
    if (method === "auto" || method === "summary-only") {
      const rows = db.query(`
        SELECT turn_index, timestamp FROM turns
        WHERE session_id = ${sidLit} AND role = 'user'
          AND COALESCE(content,'') LIKE 'This session is being continued%'
          AND turn_index < ${target}
        ORDER BY turn_index DESC
        LIMIT 1
      `);
      if (rows.length) summaryBoundary = rows[0].turn_index;
    }

    let collapseBoundary = null;
    if (method === "auto" || method === "cache-collapse-only") {
      const assistantRows = db.query(`
        SELECT turn_index, COALESCE(output_tokens,0) AS out, COALESCE(cache_read_tokens,0) AS cr,
               COALESCE(cache_creation_tokens,0) AS cc
        FROM turns
        WHERE session_id = ${sidLit} AND role = 'assistant' AND turn_index < ${target}
        ORDER BY turn_index
      `);
      const calls = collapseApiCalls(assistantRows);
      for (let i = 1; i < calls.length; i++) {
        const prev = calls[i - 1], cur = calls[i];
        if (prev.cr > 100000 && cur.cr < prev.cr * 0.2) collapseBoundary = cur.start_turn;
      }
    }

    let boundary, boundarySource;
    if (method === "summary-only") { boundary = summaryBoundary ?? 0; boundarySource = summaryBoundary != null ? "summary" : "session-start"; }
    else if (method === "cache-collapse-only") { boundary = collapseBoundary ?? 0; boundarySource = collapseBoundary != null ? "cache-collapse" : "session-start"; }
    else {
      const candidates = [summaryBoundary, collapseBoundary].filter((x) => x != null);
      if (!candidates.length) { boundary = 0; boundarySource = "session-start"; }
      else {
        boundary = Math.max(...candidates);
        boundarySource = (summaryBoundary === boundary && collapseBoundary === boundary) ? "both"
          : (summaryBoundary === boundary ? "summary" : "cache-collapse");
      }
    }

    const calls = db.query(`
      SELECT emitting_turn_index AS turn_index, timestamp, tool_call_id, tool_name, operation_type,
             success, file_path,
             substr(COALESCE(command,''),1,300) AS command,
             substr(COALESCE(arguments_json,''),1,1500) AS arguments_json
      FROM tool_calls
      WHERE session_id = ${sidLit} AND emitting_turn_index BETWEEN ${boundary} AND ${target}
      ORDER BY timestamp
    `);

    const filesMap = {};
    for (const c of calls) {
      if (!c.file_path) continue;
      const f = filesMap[c.file_path] || (filesMap[c.file_path] = {
        file_path: c.file_path, first_op: c.operation_type || c.tool_name, ops: [], last_seen: c.timestamp,
      });
      f.ops.push({ turn: c.turn_index, op: c.operation_type || c.tool_name, tool_name: c.tool_name });
      f.last_seen = c.timestamp;
    }

    const skillLoads = calls.filter((c) => c.tool_name === "Skill")
      .map((c) => ({ turn: c.turn_index, skill: parseSkillName(c.arguments_json) }));
    const skillFileReads = calls.filter((c) =>
      (c.tool_name === "Read" || c.tool_name === "read") &&
      (/\/skills\//.test(c.file_path || "") || /SKILL\.md/.test(c.file_path || "")))
      .map((c) => ({ turn: c.turn_index, file: c.file_path }));
    const skillSideloads = calls.filter((c) => {
      if (!["Bash", "bash", "shell", "exec"].includes(c.tool_name)) return false;
      const text = effectiveCommands(c).join(" ");
      return /\.claude\/skills|\.codex\/skills|\.pi\/skills|\.pi\/agent\/skills/.test(text);
    }).map((c) => ({ turn: c.turn_index, command: effectiveCommands(c).join(" ").slice(0, 200) }));

    return {
      session_id: sid,
      target_turn: target,
      boundary_turn: boundary,
      boundary_source: boundarySource,
      boundary_candidates: { summary: summaryBoundary, cache_collapse: collapseBoundary },
      window_size_turns: target - boundary,
      tool_calls: calls,
      files_touched: Object.values(filesMap),
      skills: { skill_loads: skillLoads, skill_file_reads: skillFileReads, skill_sideloads: skillSideloads },
    };
  } finally {
    db.close();
  }
}

__verb__("contextWindow", {
  name: "context-window",
  short: "Reconstruct files/tool calls/skills in scope since the last compaction, for a given session+turn",
  fields: { contextwindowopts: { bind: "contextwindowopts" } },
});
