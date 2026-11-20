// Intentionally invalid: the metadata default for a numeric parameter is -1.
// Kept in place so the catalog loader's diagnostic for an invalid sibling can be tested.
function bad() {}
__verb__("bad", {
  name: "bad",
  short: "Bad sibling with a signed numeric metadata default",
  fields: { limit: { type: "int", default: -1 } }
});
