// contrast.js -- compare array-return CLI rendering with raw JS values.
function arrayReturn(inputs) {
  const mt = require("minitrace");
  const rows = mt.query().SessionSummary().Build();
  // Return a real array of row objects: the CLI's row/array path.
  return [
    { kind: "array", session_count: 3, labels: ["pi", "codex", "claude-code"], nested: [1, [2, 3]] },
    { kind: "array", session_count: 9, labels: [], nested: [] },
  ];
}

function rawValue(inputs) {
  const mt = require("minitrace");
  // Return a plain JS object/scalar, not an array of rows.
  return {
    kind: "raw-object",
    session_count: 3,
    labels: ["pi", "codex", "claude-code"],
    nested: [1, [2, 3]],
    archiveGlobs: mt.runtime.archiveGlob,
  };
}

__verb__("arrayReturn", {
  name: "array-return",
  short: "Return an array of row objects",
  fields: {},
});

__verb__("rawValue", {
  name: "raw-value",
  short: "Return a raw JS object value",
  fields: {},
});
