function good(v) { return [{argc: arguments.length, v: v === undefined ? null : v}]; }
__verb__("good", { name: "good", short: "Good", params: { source: { type: "string", required: true, bind: "v", help: "Source" } } });
