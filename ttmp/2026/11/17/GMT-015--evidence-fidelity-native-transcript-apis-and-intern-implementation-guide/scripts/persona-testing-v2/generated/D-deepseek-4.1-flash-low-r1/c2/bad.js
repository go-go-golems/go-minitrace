__section__("bad_filters", { title: "Bad", fields: { offset: { type: "int", default: -1, help: "neg" } } });
function bad(f) { return [f]; }
__verb__("bad", { name: "bad", short: "Bad", fields: { bad_filters: { bind: "bad_filters" } } });
