package minitracejs

import (
	"encoding/json"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestJSQueryFidelityBuildersAndExecute(t *testing.T) {
	vm := newRelationalTestVM()
	session := minitrace.BuildSessionSkeleton("fidelity", "codex", "test", "test")
	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("archive", string(raw)); err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`
 const mt = require("minitrace");
 const builder = mt.db().sources(mt.sources().content(archive).build());
 const bound = mt.sql` + "`" + `SELECT ${"abé"} AS evidence` + "`" + `.build();
 const check = (value, message) => { if (!value) throw Error(message); };
 const db = builder.limits({maxCellBytes:3}).build();
 try {
  const r=db.execute(bound);
  check(r.rows[0].evidence==='ab'&&!r.complete&&r.truncated,'UTF-8 fidelity');
  check(r.cellDiagnostics[0].originalBytes===4&&r.cellDiagnostics[0].emittedBytes===2,'byte metadata');
  const nullValue=db.execute(mt.sql` + "`" + `SELECT NULL AS value` + "`" + `.build());
  check(nullValue.complete&&nullValue.rows[0].value===null,'null is complete');
  const serialized=JSON.parse(JSON.stringify(r));
  check(serialized.complete===false&&serialized.rows[0].evidence==='ab','JSON roundtrip');
 } finally {db.close();}
 const strict=builder.limits({maxCellBytes:3,maxRows:1,strict:true}).build();
 try {
  const r=strict.execute(mt.sql` + "`" + `SELECT 'abé' AS text UNION ALL SELECT 'xyz' ORDER BY text` + "`" + `.build());
  check(r.error&&r.rows.length===0&&r.count===0&&!r.complete&&r.rowsTruncated,'strict envelope');
  const single=strict.execute(bound);
  check(single.error&&single.rows.length===0,'execute strict');
 } finally {strict.close();}
 const exact=builder.limits({maxCellBytes:4,strict:true}).build();
 try {check(exact.execute(bound).complete,'exact byte cap');} finally {exact.close();}
 `)
	if err != nil {
		t.Fatal(err)
	}
}
