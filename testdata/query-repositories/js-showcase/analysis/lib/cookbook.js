function _safeNumber(value) {
  const n = Number(value || 0);
  return Number.isFinite(n) ? n : 0;
}

function _roundTo(value, decimals) {
  const factor = Math.pow(10, decimals || 1);
  return Math.round(_safeNumber(value) * factor) / factor;
}

function _firstBy(rows, key) {
  const ret = {};
  for (const row of rows || []) {
    const value = row?.[key];
    if (value == null || ret[value]) {
      continue;
    }
    ret[value] = row;
  }
  return ret;
}

function _groupBy(rows, key) {
  const ret = {};
  for (const row of rows || []) {
    const value = row?.[key];
    if (value == null) {
      continue;
    }
    ret[value] = ret[value] || [];
    ret[value].push(row);
  }
  return ret;
}

function _shortWorkspace(value) {
  const source = String(value || "");
  if (!source) {
    return "(none)";
  }
  const parts = source.split("/").filter(Boolean);
  return parts.slice(-2).join("/") || source;
}

function _joinTopValues(rows, field, limit) {
  return (rows || [])
    .slice(0, limit || 3)
    .map((row) => row?.[field])
    .filter(Boolean)
    .join(", ");
}

function _pairCounts(sessionToolRows) {
  const grouped = _groupBy(sessionToolRows, "id");
  const counts = {};

  for (const rows of Object.values(grouped)) {
    const tools = Array.from(new Set(rows.map((row) => row.tool_name).filter(Boolean))).sort();
    for (let i = 0; i < tools.length; i += 1) {
      for (let j = i + 1; j < tools.length; j += 1) {
        const left = tools[i];
        const right = tools[j];
        const key = `${left}__${right}`;
        counts[key] = (counts[key] || 0) + 1;
      }
    }
  }

  return Object.entries(counts)
    .map(([key, session_count]) => {
      const [left_tool, right_tool] = key.split("__");
      return {
        left_tool,
        right_tool,
        pair: `${left_tool} + ${right_tool}`,
        session_count,
      };
    })
    .sort((a, b) => b.session_count - a.session_count || a.pair.localeCompare(b.pair));
}

function _classifySessionShape(row) {
  const toolCalls = _safeNumber(row.tool_call_count);
  const turnCount = _safeNumber(row.turn_count);
  const uniqueTools = _safeNumber(row.unique_tools);
  const userTurns = _safeNumber(row.user_turns);
  const assistantTurns = _safeNumber(row.assistant_turns);
  const ratio = userTurns === 0 ? assistantTurns : assistantTurns / userTurns;

  if (toolCalls >= 12 && uniqueTools >= 4) {
    return "tool-orchestrator";
  }
  if (ratio >= 8) {
    return "assistant-monologue";
  }
  if (turnCount >= 20 && toolCalls <= 4) {
    return "conversation-heavy";
  }
  if (toolCalls >= 6 && turnCount <= 12) {
    return "surgical-executor";
  }
  return "balanced-builder";
}

exports.safeNumber = _safeNumber;
exports.round1 = (value) => _roundTo(value, 1);
exports.round2 = (value) => _roundTo(value, 2);
exports.firstBy = _firstBy;
exports.groupBy = _groupBy;
exports.shortWorkspace = _shortWorkspace;
exports.joinTopValues = _joinTopValues;
exports.pairCounts = _pairCounts;
exports.classifySessionShape = _classifySessionShape;
