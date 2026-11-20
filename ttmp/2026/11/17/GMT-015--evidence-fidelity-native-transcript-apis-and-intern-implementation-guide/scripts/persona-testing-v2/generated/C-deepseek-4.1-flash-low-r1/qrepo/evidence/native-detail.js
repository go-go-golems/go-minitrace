__section__("inputs", {
  title: "Inputs",
  fields: { path: { type: "string", help: "Absolute path to native JSONL file" } }
});
function detail(inputs) {
  const mt = require("minitrace");
  const insp = mt.native().File(inputs.path).Inspect();
  return [{
    scan: insp.scan,
    limitations: insp.limitations,
    requests: insp.requests.map(r => ({ id: r.id, status: r.status, querySource: r.querySource, source: r.source, diagnostics: r.diagnostics })),
    compactions: insp.compactions.map(c => ({ boundary: c.boundary, logicalParentUuid: c.logicalParentUuid, summaries: c.summaries, beforeMessage: c.beforeMessage, firstFollowingMessage: c.firstFollowingMessage, firstFollowingAssistant: c.firstFollowingAssistant }))
  }];
}
__verb__("detail", { name: "detail", short: "Detailed native inspection", fields: { inputs: { bind: "inputs" } } });
