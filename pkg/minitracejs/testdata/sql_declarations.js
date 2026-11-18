const mt = require("minitrace");
const db = mt.db().limits({maxRows: 1}).build();
const expect = (condition, message) => {
  if (!condition) throw Error(message);
};

try {
  const expectError = (plan, message, options) => {
    const result = db.execute(plan, options);
    expect(result.error && result.error.includes(message) && !result.complete &&
      result.columns.length === 0 && result.rows.length === 0, JSON.stringify(result));
  };

  const missing = mt.sql`SELECT 1 AS actual WHERE 0`.columns({declared: mt.type.integer()});
  expectError(missing.build(), 'SQLite returned "actual"');
  expectError(missing.select(() => ({kept: 1})).build(), 'SQLite returned "actual"');

  const unexpected = mt.sql`SELECT 'wrong' AS n`.columns({n: mt.type.integer()});
  expectError(unexpected.build(), 'column "n" expects integer');
  expectError(unexpected.where(row => row.n.eq(1)).build(), 'column "n" expects integer');

  const host = mt.sql`SELECT 1 AS keep`.columns({keep: mt.type.integer()});
  expectError(host.select(() => ({total: unexpected.count()})).build(), 'column "n" expects integer');

  const nullable = mt.sql`SELECT NULL AS n`.columns({n: mt.type.integer().nullable()});
  const nullResult = db.execute(nullable.build());
  expect(nullResult.complete && nullResult.rows[0].n === null, JSON.stringify(nullResult));

  const value = mt.param.integer("value");
  const before = mt.param.string("before");
  const source = mt.sql`SELECT ${value} AS n UNION ALL SELECT 2 AS n`
    .columns({n: mt.type.integer()});
  const composed = source.select(row => ({before, n: row.n}))
    .where(row => row.n.eq(value)).build();
  const result = db.execute(composed, {params: {value: 7, before: "prefix"}});
  expect(result.complete && result.rows[0].n === 7 && result.rows[0].before === "prefix",
    JSON.stringify(result));

  const flags = mt.sql`SELECT 1 AS flag`.columns({flag: mt.type.boolean()});
  expect(db.execute(flags.build()).rows[0].flag === true, "declared boolean was returned as an integer");
  const logical = flags.select(r => ({inverted: r.flag.not()}));
  expect(db.execute(logical.build()).rows[0].inverted === false, "logical projection typing lost");

  const failBuild = (fn, code, path) => {
    try { fn(); } catch (error) {
      expect(error.code === code && error.path === path, String(error) + " " + error.code + " " + error.path);
      return;
    }
    throw Error("expected " + code);
  };
  failBuild(() => mt.from(flags.build()), "E_QUERY_SOURCE", "from");
  failBuild(() => mt.from(flags), "E_QUERY_SOURCE", "from");
  failBuild(() => flags.where(r => r.missing), "E_UNKNOWN_COLUMN", "where.missing");
  failBuild(() => flags.where(r => r.flag === true), "E_EXPECTED_PREDICATE", "where");
  let escaped;
  flags.where(r => { escaped = r.flag; return r.flag; });
  failBuild(() => host.where(() => escaped).build(), "E_SCOPE", "build");

  const untyped = db.execute(mt.sql`SELECT 'mixed' AS n`.build());
  expect(untyped.complete && untyped.rows[0].n === "mixed", JSON.stringify(untyped));
} finally {
  db.close();
}
