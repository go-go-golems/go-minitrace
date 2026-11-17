// bad.js — intentionally invalid for the scanner-under-test.
// The field named `metadata` is numeric with a signed default (-1).
// If the catalog loader cannot parse the signed default it should isolate
// this file while keeping the valid sibling `good.js` runnable.
// Do not delete this file: it is left in place for grading.

function bad(metadata) {
  return { metadata };
}

__verb__("bad", {
  name: "bad",
  short: "Intentionally invalid: numeric metadata field defaults to -1",
  fields: {
    metadata: {
      type: "int",
      default: -1,
      help: "Signed numeric default used to test scanner support",
    },
  },
});
