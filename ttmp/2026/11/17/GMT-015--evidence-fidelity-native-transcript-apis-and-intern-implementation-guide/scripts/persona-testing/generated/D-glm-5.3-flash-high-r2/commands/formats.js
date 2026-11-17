// formats.js — contrast array-return CLI formatting with raw JS values.
//
// Returns two rows that carry the "same" list value in two forms:
//   - value_raw:    a real JavaScript array returned as-is
//   - value_string: the same array pre-serialized with JSON.stringify
// This makes the difference between how the CLI renders raw JS values
// versus pre-stringified JSON observable in both table and --output json.

function contrast() {
  return [
    { label: "raw-js-array", value_raw: ["a", "b", "c"], value_string: JSON.stringify(["a", "b", "c"]) },
    { label: "nested-object", value_raw: { k: 1, tags: ["x", "y"] }, value_string: JSON.stringify({ k: 1, tags: ["x", "y"] }) },
  ];
}

__verb__("contrast", {
  name: "contrast",
  short: "Show how the CLI formats raw JS arrays/objects versus strings",
  fields: {},
});
