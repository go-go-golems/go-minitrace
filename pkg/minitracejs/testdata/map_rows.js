const mt = require("minitrace");
const fixture = JSON.stringify({id: "test", profile: "organic", turns: [], tool_calls: []});
const builder = mt.db().sources(mt.sources().content(fixture).build());
const columns = {label: mt.type.string()};
const source = mt.sql`SELECT 1 AS n UNION ALL SELECT 2 AS n`.columns({n: mt.type.integer()});
const expect = (condition, message) => {
  if (!condition) throw Error(message);
};
const resultError = (result, message) =>
  expect(!result.complete && result.rows.length === 0 && result.error.includes(message),
    JSON.stringify(result));

const db = builder.limits({maxCellBytes: 3, timeoutMs: 250}).build();
try {
  let prefix = "a";
  let calls = 0;
  const stage = source.mapRows(row => { calls++; return {label: prefix + row.n}; }, {columns});
  const plan = stage.build();
  expect(calls === 0 && typeof stage.where === "undefined", "stage ran early or can be composed");
  prefix = "b";
  const mapped = db.execute(plan);
  expect(mapped.complete && mapped.rows[1].label === "b2" && calls === 2, JSON.stringify(mapped));

  const empty = db.execute(source.where(row => row.n.lt(0)).mapRows(() => {
    throw Error("unexpected callback");
  }, {columns}).build());
  expect(empty.complete && empty.rows.length === 0 && empty.columns[0] === "label", JSON.stringify(empty));

  const clipped = db.execute(source.mapRows(() => ({label: "abé"}), {columns}).build());
  expect(!clipped.complete && clipped.rows[0].label === "ab" &&
    clipped.cellDiagnostics[0].originalBytes === 4 && clipped.cellDiagnostics[0].column === "label",
    JSON.stringify(clipped));

  const failed = db.execute(source.mapRows(row => {
    if (row.n === 2) throw Error("private callback message");
    return {label: "ok"};
  }, {columns}).build());
  resultError(failed, "mapRows row 1: callback failed");
  expect(!failed.error.includes("private callback message"), "callback error leaked its value");

  resultError(db.execute(source.mapRows(() => ({label: 1}), {columns}).build()), "expects string");
  resultError(db.execute(source.mapRows(async () => ({label: "x"}), {columns}).build()), "synchronous object");
  resultError(db.execute(source.mapRows(() => ({get label() {
    throw Error("private getter message");
  }}), {columns}).build()), "mapRows row 0: callback failed");

  const deadline = db.execute(source.mapRows(() => {
    while (true) {}
  }, {columns}).build());
  resultError(deadline, "deadline exceeded");
  expect(db.execute(plan).complete, "interrupt survived into the next plan");
} finally {
  db.close();
}

const strict = builder.limits({maxCellBytes: 3, strict: true}).build();
try {
  const result = strict.execute(source.mapRows(() => ({label: "abé"}), {columns}).build());
  resultError(result, "strict evidence: mapRows");
  expect(result.cellDiagnostics[0].column === "label", JSON.stringify(result));
} finally {
  strict.close();
}

const limited = builder.limits({maxRows: 1}).build();
try {
  let callbacks = 0;
  const input = limited.execute(source.mapRows(() => {
    callbacks++;
    return {label: "x"};
  }, {columns}).build());
  resultError(input, "requires complete SQL input");
  expect(callbacks === 0 && input.rowsTruncated, JSON.stringify(input));
} finally {
  limited.close();
}
