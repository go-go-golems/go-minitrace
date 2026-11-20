function good(source) { return [{argc: arguments.length, v: source === undefined ? null : source}]; }
__verb__("good", { name: "good", short: "Good", params: { source: { type: "string", required: true, bind: "source", help: "Source" } } });
