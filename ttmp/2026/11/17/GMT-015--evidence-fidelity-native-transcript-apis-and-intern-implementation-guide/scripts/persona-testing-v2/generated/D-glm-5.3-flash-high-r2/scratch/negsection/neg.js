__section__("f", { fields: { n: { type: "int", default: -1, help: "negative in section" } } });
function neg(f) { return [{ n: f.n }]; }
__verb__("neg", { name: "neg", short: "negative default in section metadata", fields: { f: { bind: "f" } } });
