function good(v) {
  const mt = require("minitrace");
  let r = {};
  try { r.runtimeKeys = Object.keys(mt.runtime); r.runtimeJSON = JSON.parse(JSON.stringify(mt.runtime)); } catch(e) { r.runtimeError = String(e); }
  return [{argc: arguments.length, a0Type: typeof v, a0: v === undefined ? null : v, keys: v ? Object.keys(v) : [], r}];
}
__verb__("good", { name: "good", short: "Good", fields: { source: { type: "string", required: true, help: "Source" } } });
