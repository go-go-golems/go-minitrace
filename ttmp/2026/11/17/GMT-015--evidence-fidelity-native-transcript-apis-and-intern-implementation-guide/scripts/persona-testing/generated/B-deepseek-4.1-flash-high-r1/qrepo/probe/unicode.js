// Byte fidelity probes: exact UTF-8 length, permissive clipping, strict rejection.
function utf8() {
  const mt = require("minitrace");
  return {
    text: "abé",
    utf8ByteLength: mt.utf8ByteLength("abé"),
    latin1Length: "abé".length,
  };
}

__verb__("utf8", {
  name: "utf8",
  short: "Exact UTF-8 byte length of abé",
  fields: {},
});
