package minitracejs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestJSNegativeResultContractsAndDiagnostics(t *testing.T) {
	vm := goja.New()
	raw, err := json.Marshal(minitrace.BuildSessionSkeleton("negative", "codex", "test", "test"))
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.Set("archive", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err = vm.Set("newBuilder", func() *goja.Object { return builderObject(vm, NewDBBuilder(context.Background())) }); err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`
 function check(ok, message) { if (!ok) throw new Error(message); }
 const db = newBuilder().Content(archive).Build();
 try {
  for (const sql of ['', 'DELETE FROM sessions', 'SELECT missing FROM sessions', 'SELECT name FROM sqlite_master']) {
   for (const result of [db.queryResult(sql), db.execute({sql:sql || 'SELECT missing FROM sessions',args:[]})]) {
    check(result.error && !result.complete && result.count===0,'error completeness');
    check(Array.isArray(result.rows)&&result.rows.length===0,'rows array');
    check(Array.isArray(result.columns)&&result.columns.length===0,'columns array');
    const roundtrip=JSON.parse(JSON.stringify(result));
    check(Array.isArray(roundtrip.rows)&&Array.isArray(roundtrip.columns),'JSON arrays');
   }
  }
  let hint=''; try { db.execute('SELECT 1'); } catch(e) { hint=String(e); }
  check(hint.includes('queryResult(sql, ...args)')&&hint.includes('execute({sql:'),'actionable string hint');
  check(db.execute({sql:'SELECT ? AS n',args:[7]}).rows[0].n===7,'recipe still works');
  const lazyRecipe={sql:'SELECT 1 AS n', get unrelated(){throw new Error('must not export entire recipe');}};
  check(db.execute(lazyRecipe).rows[0].n===1,'only recipe fields accessed');
  for (const method of ['query','queryOne']) {let thrown=false;try{db[method]('SELECT missing FROM sessions');}catch(e){thrown=true;}check(thrown,method+' still throws');}
 } finally {db.close();}
 for (const auto of [false,true]) {
  let message='';try{newBuilder().Content('{"id":"bad","turns":[],"metrics":[]}').AutoConvert(auto).Build();}catch(e){message=String(e);}
  check(message.includes('metrics')&&message.includes('cannot unmarshal array')&&!message.includes('AutoConvert is disabled'),'typed normalized decode diagnostic');
 }
 `)
	if err != nil {
		t.Fatal(err)
	}
}
