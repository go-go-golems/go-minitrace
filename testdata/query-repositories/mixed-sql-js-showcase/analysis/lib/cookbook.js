function _safeNumber(value) {
  const n = Number(value || 0);
  return Number.isFinite(n) ? n : 0;
}

function _round1(value) {
  return Math.round(_safeNumber(value) * 10) / 10;
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

function _shortWorkspace(value) {
  const source = String(value || "");
  if (!source) return "(none)";
  const parts = source.split("/").filter(Boolean);
  return parts.slice(-2).join("/") || source;
}

exports.safeNumber = _safeNumber;
exports.round1 = _round1;
exports.firstBy = _firstBy;
exports.shortWorkspace = _shortWorkspace;
