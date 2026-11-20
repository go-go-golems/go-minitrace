function good(fields) { return [{ source: fields.source }]; }
__verb__("good", { name: "good", short: "Good", fields: { source: { type: "string", required: true, help: "Source path" } } });
