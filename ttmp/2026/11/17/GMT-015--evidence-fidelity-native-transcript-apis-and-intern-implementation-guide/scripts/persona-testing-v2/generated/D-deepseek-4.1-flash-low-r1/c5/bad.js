__section__("other_filters", { title: "Other", fields: { x: { type: "string", help: "x" } } });
function bad(f) { return [f]; }
__verb__("bad", { name: "bad", short: "Bad", fields: { other_filters: { bind: "other_filters" } } });
