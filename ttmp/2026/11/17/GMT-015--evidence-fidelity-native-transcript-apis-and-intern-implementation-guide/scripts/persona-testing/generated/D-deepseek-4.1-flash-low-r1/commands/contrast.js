// contrast.js — compare array-return CLI formatting with raw JS values.
// array-return returns an array of objects; Glazed treats each element as a
// row. raw-values returns one object containing scalar, array and nested
// object values, so the CLI must flatten/stringify the non-scalar members.
// Both return values are pure JavaScript values; no SQL is involved.

function arrayReturn() {
  return [
    { rank: 1, label: "alpha", tags: ["a", "b"], meta: { n: 1 } },
    { rank: 2, label: "beta", tags: [], meta: { n: 2 } },
  ];
}

function rawValues() {
  return {
    label: "single-object",
    rank: 1,
    tags: ["a", "b"],
    meta: { nested: true, n: 2 },
    flag: false,
    missing: null,
  };
}

__verb__("arrayReturn", {
  name: "array-return",
  short: "Return an array of JS objects so the CLI emits one row per element",
});

__verb__("rawValues", {
  name: "raw-values",
  short: "Return one object with array/object/null members as raw JS values",
});
