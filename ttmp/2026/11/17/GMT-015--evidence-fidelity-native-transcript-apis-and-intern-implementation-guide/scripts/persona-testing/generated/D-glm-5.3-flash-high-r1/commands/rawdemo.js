// rawdemo.js — contrast array-return CLI formatting with raw JS values.
// Verb "rawdemo" returns a plain object containing:
//   - db.tables()  (array of strings)
//   - db.schema()  raw value (shape documented as "schema introspection" without a stated type)
//   - a scalar, a nested object, and a one-row array for side-by-side formatting comparison.
function rawdemo() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    return {
      scalar_value: 42,
      tables_value: db.tables(),
      schema_raw: db.schema(),
      nested_object: { a: 1, b: [1, 2, 3] },
      one_row_array: [{ col_a: "x", col_b: 2 }],
    };
  } finally {
    db.close();
  }
}

__verb__("rawdemo", {
  name: "rawdemo",
  short: "Contrast array-return CLI formatting with raw JS values",
});
