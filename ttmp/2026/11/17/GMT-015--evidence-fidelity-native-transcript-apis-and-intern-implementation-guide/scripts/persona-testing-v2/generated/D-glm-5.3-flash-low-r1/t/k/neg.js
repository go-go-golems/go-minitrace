__verb__("neg", {
  name: "neg",
  short: "Neg",
  fields: {
    a: { type: "int", default: "-1", help: "t" },
  },
});
function neg(a) { return [{ a: a }]; }
