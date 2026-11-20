__section__("inputs", { title: "Inputs", fields: { path: { type: "string", help: "Path" } } });
function early(inputs) {
  const mt = require("minitrace");
  let seen = 0;
  const scan = mt.native().File(inputs.path).ForEach(record => {
    if (record.diagnostic) return;
    if (record.value && record.value.type === "api-request") { seen++; return false; }
  });
  return [{ seen, scan }];
}
__verb__("early", { name: "early", short: "Early stop scan", fields: { inputs: { bind: "inputs" } } });
