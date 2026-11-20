// bad.js — sibling structured JS query command (scanner-first).
// Single self-named verb "bad", so the CLI path collapses to:
//   go-minitrace query commands bad
// Exercises a NEGATIVE numeric default (-1) defined purely in the JS
// __verb__ metadata (no CLI value, no YAML alias default).
// The handler only echoes the resolved parameter so the effective default
// is observable without touching any archive data.

function bad(params) {
  const limit = params.limit;
  return [{
    command: "bad",
    limit: limit,
    limit_is_negative: (typeof limit === "number" && limit < 0),
  }];
}

__verb__("bad", {
  name: "bad",
  short: "Sibling command with a negative numeric metadata default (-1)",
  long: "Verifies that a negative numeric default declared in __verb__ metadata is accepted and applied when no CLI value is given.",
  fields: {
    limit: {
      type: "int",
      default: -1,
      help: "Numeric limit with signed metadata default -1",
    },
  },
});
