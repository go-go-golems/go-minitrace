package minitracejs

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestSessionToolCountsGoja(t *testing.T) {
	vm := goja.New()
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
	_ = vm.Set("newDB", func() *goja.Object { return builderObject(vm, NewDBBuilder(context.Background())) })
	_ = vm.Set("newQuery", func() *goja.Object { return queryRecipeBuilderObject(vm, NewQueryRecipeBuilder()) })
	_, err := vm.RunString(`
 const builder=newDB().SQLiteMemory();
 archives.forEach((archive,i)=>builder.Content(archive,'fixture-'+i));
 const db=builder.Build();
 function check(v,m){if(!v)throw Error(m)}
 try {
  const recipe=newQuery().SessionToolCounts().Build();
  const all=db.execute(recipe);
  check(all.complete&&!all.error,'complete envelope');
  check(JSON.stringify(all.rows.map(r=>({session_id:r.session_id,tool_count:r.tool_count})))==='[{"session_id":"a","tool_count":0},{"session_id":"m","tool_count":1},{"session_id":"z","tool_count":2}]','all identities/counts/order');
  const zero=newQuery().SessionToolCounts().SessionID('a').Build();
  check(JSON.stringify(zero.args())==='["a","a"]','bound filter');
  check(db.execute(zero).rows[0].tool_count===0,'filtered zero');
  const absent=newQuery().SessionToolCounts().SessionID("' OR 1=1 --").Build();
  check(db.execute(absent).rows.length===0,'filter must remain bound');
 }finally{db.close()}
 `)
	if err != nil {
		t.Fatal(err)
	}
}
