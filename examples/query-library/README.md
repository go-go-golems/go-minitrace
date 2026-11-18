# Local transcript analysis helpers

Copy `analysis.js` beside a command script and load it with
`require("./analysis")`. The module contains ordinary functions that use the
public minitrace API. A function returns a query builder unless its purpose
requires processing several result sets.

```javascript
const mt = require("minitrace");
const analysis = require("./analysis");

function countTools() {
  const db = mt.db().sources(mt.sources().runtimeArchives().build()).build();
  try {
    const query = analysis.sessionToolCounts(db);
    const result = db.execute(query.build());
    if (result.error || !result.complete) throw Error(result.error || "Incomplete result");
    return result.rows;
  } finally {
    db.close();
  }
}
```

`toolRows` and the token usage helpers declare column types for composition.
For example, `analysis.toolRows().orderBy(t => [t.session_id.asc(),
t.source_ordinal.asc()])` requests archive order. The tool usage helper estimates
output tokens and labels the estimate separately from provider-reported usage.
`turnFrames(db)` combines two complete result sets into grouped JS objects.
