function _roundToTenth(value) {
  return Math.round(value * 10) / 10;
}

function _addSharePercent(rows, countField) {
  const field = countField || "count";
  const total = rows.reduce((sum, row) => sum + Number(row[field] || 0), 0);
  return rows.map((row, index) => {
    const count = Number(row[field] || 0);
    return {
      ...row,
      rank: index + 1,
      share_percent: total === 0 ? 0 : _roundToTenth((count / total) * 100),
    };
  });
}

exports.addSharePercent = _addSharePercent;
