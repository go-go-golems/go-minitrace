// Contrast helper: returns raw JS values (not SQL rows) to show how the
// Glazed CLI formats an object return versus an array-of-object returns.
__section__("inputs", {
  title: "Inputs",
  fields: {
    source: { type: "string", required: true, help: "Required. Label echoed back" },
  },
});

function raw(inputs) {
  const mt = require("minitrace");
  return {
    source: inputs.source,
    int_value: 42,
    float_value: 1.5,
    bool_value: true,
    null_value: null,
    undefined_value: undefined,
    list_value: [1, 2, 3],
    nested_value: { a: 1 },
    utf8_bytes: mt.utf8ByteLength("abé"),
  };
}

__verb__("raw", {
  name: "raw",
  short: "Return raw JS values for CLI formatting contrast",
  fields: { inputs: { bind: "inputs" } },
});
