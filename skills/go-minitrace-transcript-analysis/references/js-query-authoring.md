# JavaScript query command authoring

For a one-off investigation, prefer saved SQL and `go-minitrace query run`. Use a JavaScript command for reusable typed flags, several queries, ordinary JS analysis, or scoped domain vocabulary. Check the installed binary with `go-minitrace help js-api-reference` and `go-minitrace help structured-query-commands`.

```js
__section__("filters", {
  fields: {
    framework: {type: "string", default: "pi", help: "Framework to match"},
    limit: {type: "int", default: 25, help: "Maximum rows to select"},
  },
});

function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db().sources(mt.sources().runtimeArchives().build())
    .import({autoConvert: true, strict: true})
    .limits({maxCellBytes: 8192, strict: true}).build();
  try {
    const plan = mt.sql`
      SELECT session_id, title, agent_framework
      FROM sessions WHERE agent_framework = ${filters.framework}
      ORDER BY started_at DESC, session_id LIMIT ${filters.limit}
    `.build();
    const result = db.execute(plan);
    if (result.error || !result.complete) throw Error(result.error || "Incomplete result");
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("sessionList", {
  name: "session-list", short: "List minitrace sessions",
  fields: {filters: {bind: "filters"}},
});
```

Put the script at `my-commands/overview/session-list.js` and run:

```sh
go-minitrace query commands --query-repository ./my-commands overview session-list \
  --archive-glob './archives/active/*/*.minitrace.json' --framework pi --output json
```

Use `mt.from(db.table("sessions"))` to compose a catalog-backed query. Use symbolic callbacks for filters and projections, for example `.where(row => row.agent_framework.eq("pi"))`. Construct actual per-session tool counts with a correlated `.count()` over `tool_calls` so sessions with zero tool calls remain present. A named `mt.param.integer("minimum").default(0)` can be bound both in fluent filters and in tagged SQL through `db.execute(plan, {params: {minimum: 3}})`.

Prefer ordinary functions for reusable query transformations. Attach scoped fragment methods only when they clarify several related operations. A `mapRows` stage runs once per row after complete SQL execution and requires declared output columns. For multiquery reductions, check every input envelope before combining them in JS.

Keep interpolations scalar. Use explicit statement text for identifiers. Treat safety row caps, UTF-8 clipping, and SQL-level selection as distinct. For raw record inspection, use `mt.native().file(path).inspect()` or `.forEach(callback)`. Preserve physical coordinates and provenance in evidence claims. The repository's `examples/query-library/analysis.js` provides composable tool and token projections, timelines, and frames.
