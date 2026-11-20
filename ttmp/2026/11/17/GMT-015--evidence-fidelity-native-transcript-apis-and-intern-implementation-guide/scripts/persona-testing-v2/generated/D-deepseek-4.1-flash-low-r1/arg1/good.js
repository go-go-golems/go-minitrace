function good() { return [{argc: arguments.length, a0: arguments[0] === undefined ? null : arguments[0], keys: arguments[0] ? Object.keys(arguments[0]) : []}]; }
__verb__("good", { name: "good", short: "Good", params: { source: { type: "string", required: true, help: "Source" } } });
