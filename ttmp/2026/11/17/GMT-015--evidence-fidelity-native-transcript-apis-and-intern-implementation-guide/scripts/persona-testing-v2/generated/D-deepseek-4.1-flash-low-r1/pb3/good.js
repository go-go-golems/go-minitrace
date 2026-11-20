function good(fields) { return [{argc: arguments.length, v: fields === undefined ? null : fields}]; }
__verb__("good", { name: "good", short: "Good", params: { source: { type: "string", required: true, bind: "fields", help: "Source" } } });
