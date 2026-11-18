package minitracejs

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestSessionToolCountsGoja(t *testing.T) {
	vm := newRelationalTestVM()
	archives := []string{}
	// Deliberately out of ID order; a has zero tools and forged summary metadata.
	for i, id := range []string{"z", "a", "m"} {
		s := minitrace.BuildSessionSkeleton(id, "synthetic", "test", "test")
		s.Metrics.ToolCallCount = 999
		n := []int{2, 0, 1}[i]
		for j := 0; j < n; j++ {
			s.ToolCalls = append(s.ToolCalls, minitrace.ToolCall{ID: fmt.Sprintf("%s-%d", id, j), ToolName: "read"})
		}
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		archives = append(archives, string(raw))
	}
	_ = vm.Set("archives", archives)
	_, err := vm.RunString(`
 const mt=require("minitrace");
 const sources=archives.reduce((builder, archive, i) =>
   builder.content(archive, {name: 'fixture-'+i}), mt.sources()).build();
 const db=mt.db().sources(sources).build();
 function check(v,m){if(!v)throw Error(m)}
 try {
  const sessions=mt.from(db.table("sessions"));
  const tools=mt.from(db.table("tool_calls"));
  const counts=sessions.select(s=>({
   session_id:s.session_id,
   tool_count:tools.where(t=>t.session_id.eq(s.session_id)).count()
  })).orderBy(s=>s.session_id.asc());
  const recipe=counts.build();
  const all=db.execute(recipe);
  check(all.complete&&!all.error,'complete envelope');
  check(JSON.stringify(all.rows.map(r=>({session_id:r.session_id,tool_count:r.tool_count})))==='[{"session_id":"a","tool_count":0},{"session_id":"m","tool_count":1},{"session_id":"z","tool_count":2}]','all identities/counts/order');
  const zero=counts.where(s=>s.session_id.eq("a")).build();
  check(db.execute(zero).rows[0].tool_count===0,'filtered zero');
  const absent=counts.where(s=>s.session_id.eq("' OR 1=1 --")).build();
  check(db.execute(absent).rows.length===0,'filter must remain bound');
 }finally{db.close()}
 `)
	if err != nil {
		t.Fatal(err)
	}
}
