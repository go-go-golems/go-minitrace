function good(v) { return [{source: (v && v.source) || null}]; }
__verb__("good", { name: "good", short: "Good", flags: { source: { type: "string", required: true, help: "Source path" } } });
