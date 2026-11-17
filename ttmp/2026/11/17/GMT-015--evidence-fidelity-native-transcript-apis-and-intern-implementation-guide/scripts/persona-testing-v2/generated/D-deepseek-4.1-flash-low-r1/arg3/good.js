function good(v) { return [{argc: arguments.length, a0: v === undefined ? null : v, keys: v ? Object.keys(v) : []}]; }
__verb__("good", { name: "good", short: "Good", fields: { source: { type: "string", required: true, help: "Source" } } });
