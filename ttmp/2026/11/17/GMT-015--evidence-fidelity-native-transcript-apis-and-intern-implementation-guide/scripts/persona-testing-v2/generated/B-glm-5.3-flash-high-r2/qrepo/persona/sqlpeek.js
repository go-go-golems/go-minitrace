function sqlPeek() {
  const mt = require("minitrace");
  return {
    tool: mt.query().ToolRows().Build().sql(),
    usage: mt.query().TokenUsageRows().Build().sql(),
    counts: mt.query().SessionToolCounts().Build().sql(),
  };
}
__verb__("sqlPeek", { name: "sqlpeek", short: "Peek recipe SQL", fields: {} });
