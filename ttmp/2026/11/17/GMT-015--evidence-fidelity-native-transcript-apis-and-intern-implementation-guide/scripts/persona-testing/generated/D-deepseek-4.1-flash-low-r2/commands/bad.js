// bad.js -- INTENTIONAL signed-default probe. Kept in place for grading.
// The numeric default below is -1 (a negative/signed literal in scanner metadata).
__section__("inputs", {
  title: "Bad inputs",
  fields: {
    offset: {
      type: "int",
      default: -1,
      help: "Numeric offset whose default is -1",
    },
  },
});

function bad(inputs) {
  const mt = require("minitrace");
  return [{ offset: inputs.offset, sessionCount: mt.runtime.archiveGlob.length }];
}

__verb__("bad", {
  name: "bad",
  short: "Signed-default probe command",
  fields: { inputs: { bind: "inputs" } },
});
