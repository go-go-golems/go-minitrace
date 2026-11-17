package minitracejs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestJSQueryFidelityBuildersAndExecute(t *testing.T) {
	vm := goja.New()
	session := minitrace.BuildSessionSkeleton("fidelity", "codex", "test", "test")
	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("archive", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("newBuilder", func() *goja.Object { return builderObject(vm, NewDBBuilder(context.Background())) }); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("newLimits", func() *goja.Object { return queryLimitsBuilderObject(vm, NewQueryLimitsBuilder()) }); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("boundRecipe", queryRecipeObject(vm, &QueryRecipe{SQL: "SELECT ? AS evidence", Args: []any{"abé"}})); err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`
 function check(v,message){if(!v)throw new Error(message);}
 const db=newBuilder().Content(archive).MaxCellBytes(3).Build();
 try {
  const r=db.execute(boundRecipe);
  check(r.rows[0].evidence==='ab'&&!r.complete&&r.truncated,'UTF-8 fidelity');
  check(r.cellDiagnostics[0].originalBytes===4&&r.cellDiagnostics[0].emittedBytes===2,'byte metadata');
  const bound=db.execute({sql:'SELECT ? AS value',args:["' OR 1=1 --"]});
  check(bound.cellDiagnostics[0].originalBytes===11,'bound recipe argument');
  const nullValue=db.execute({sql:'SELECT NULL AS value'});
  check(nullValue.complete&&nullValue.rows[0].value===null,'null is complete');
  let invalid=false;try{db.execute({args:[]});}catch(e){invalid=true;}check(invalid,'invalid recipe rejected');
  const serialized=JSON.parse(JSON.stringify(r));check(serialized.complete===false&&serialized.rows[0].evidence==='ab','JSON roundtrip');
 }finally{db.close();}
 const strict=newBuilder().Content(archive).Limits(newLimits().CellBytes(3).Rows(1).StrictEvidence().Build()).Build();
 try {
  const r=strict.queryResult("SELECT 'abé' AS text UNION ALL SELECT 'xyz' ORDER BY text");
  check(r.error&&r.rows.length===0&&r.count===0&&!r.complete&&r.rowsTruncated,'strict envelope');
  let thrown=false;try{strict.query("SELECT 'abcd'");}catch(e){thrown=true;}check(thrown,'strict query throws');
  let thrownOne=false;try{strict.queryOne("SELECT 'abcd'");}catch(e){thrownOne=true;}check(thrownOne,'strict queryOne throws');
  const recipe=strict.execute(boundRecipe);check(recipe.error&&recipe.rows.length===0,'execute strict');
 }finally{strict.close();}
 const exact=newBuilder().Content(archive).MaxCellChars(2).MaxCellBytes(4).StrictEvidence(true).Build();
 try{check(exact.execute(boundRecipe).complete,'byte setter precedence');}finally{exact.close();}
 const legacy=newBuilder().Content(archive).MaxCellBytes(4).MaxCellChars(3).Build();
 try{check(!legacy.execute(boundRecipe).complete,'last setter legacy alias');}finally{legacy.close();}
 const limitsLegacy=newBuilder().Content(archive).Limits(newLimits().CellBytes(4).CellChars(3).Build()).Build();
 try{check(!limitsLegacy.execute(boundRecipe).complete,'limits alias');}finally{limitsLegacy.close();}
 `)
	if err != nil {
		t.Fatal(err)
	}
}
