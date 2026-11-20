function good() {
  const mt = require("minitrace");
  let r = {};
  try { r.runtimeKeys = Object.keys(mt.runtime); r.runtimeJSON = JSON.parse(JSON.stringify(mt.runtime)); } catch(e) { r.runtimeError = String(e); }
  return [{argc: arguments.length, a0: arguments[0] === undefined ? null : arguments[0], keys: arguments[0] ? Object.keys(arguments[0]) : [], r}];
}
__verb__("good", { name: "good", short: "Good", params: { source: { type: "string", required: true, help: "Source" } } });
