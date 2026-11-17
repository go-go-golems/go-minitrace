function good(v) { return [{source: (v && v.source) || null}]; }
__verb__("good", { name: "good", short: "Good", parameters: { source: { type: "string", required: true, help: "Source path" } } });
