const mt = require("minitrace");
const db = mt.db().sources(mt.sources().content(JSON.stringify({
  id: "test", profile: "organic", turns: [], tool_calls: [],
})).build()).build();

const expect = (condition, message) => {
  if (!condition) throw Error(message);
};
const rejects = (fn, message) => {
  try { fn(); } catch (error) {
    if (String(error).includes(message)) return;
    throw error;
  }
  throw Error("expected " + message);
};
const resultError = (result, message) =>
  expect(!result.complete && result.rows.length === 0 && result.columns.length === 0 &&
    result.error.includes(message), JSON.stringify(result));

try {
  const base = mt.param.integer("minimum");
  const minimum = base.default(2).range(0, 5).ui({widget: "slider", label: "Minimum"});
  let calls = 0;
  const query = mt.sql`SELECT ${minimum} AS n, ${"bound"} AS label`
    .columns({n: mt.type.integer(), label: mt.type.string()})
    .where(r => { calls++; return r.n.gte(minimum); });
  const plan = query.build();
  const metadata = plan.parameters();
  expect(metadata.length === 1 && metadata[0].default === 2 &&
    metadata[0].ui.widget === "slider", JSON.stringify(metadata));
  metadata[0].default = 5;
  expect(db.execute(plan).rows[0].n === 2, "default lost");
  expect(db.execute(plan, {params: {minimum: 5}}).rows[0].n === 5, "override lost");
  expect(calls === 1, "execution reran callback");
  resultError(db.execute(plan, {params: {minimum: 6}}), "integer in [0, 5]");
  resultError(db.execute(plan, {params: {minimum: 1.2}}), "expects integer");
  resultError(db.execute(plan, {params: {minimum: null}}), "expects integer");
  resultError(db.execute(plan, {params: {minimum: 1, unexpected: 2}}), "unknown parameter");
  resultError(db.execute(mt.sql`SELECT ${base} AS n`.build()), "missing parameter");
  rejects(() => mt.sql`SELECT ${base}, ${minimum}`.build(), "conflicting declarations");
  rejects(() => minimum.range(3, 5), "integer in [3, 5]");

  const config = {
    name: "session-filter", target: "query", requires: ["session_id"],
    methods: {withId(q, id) { return q.where(s => s.session_id.eq(id)); }},
  };
  const fragment = mt.fragment(config);
  config.requires.push("nonexistent");
  config.methods.withId = () => { throw Error("mutable fragment"); };
  const sessions = mt.from(db.table("sessions"));
  const extended = sessions.extend(fragment);
  expect(typeof sessions.withId === "undefined", "fragment mutated original builder");
  const result = db.execute(extended.withId("test").withId("test").build());
  expect(result.complete && result.rows[0].session_id === "test", JSON.stringify(result));
  expect(db.execute(extended.withId("missing").build()).rows.length === 0, "fragment branch mutated");
  rejects(() => extended.select(() => ({alias: "x"})).withId("test"), "session_id");
  rejects(() => extended.extend(fragment), "collides");
  rejects(() => sessions.extend(mt.fragment({
    name: "collision", target: "query", methods: {where(q) { return q; }},
  })), "collides");
  rejects(() => sessions.extend(mt.fragment({
    name: "invalid", target: "query", methods: {escape() { return mt.from(db.table("turns")); }},
  })).escape(), "derived from its input");
} finally {
  db.close();
}
