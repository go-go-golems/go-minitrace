__section__("f", { fields: { n: { type: "int", help: "n" } } });
function pos(f) { return [{ n: f.n }]; }
__verb__("pos", { name: "pos", short: "positive default in verb metadata", fields: { f: { bind: "f" }, k: { type: "int", default: 1, help: "positive" } } });
