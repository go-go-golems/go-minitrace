package minitracejs

import (
	"encoding/json"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestJSNegativeResultContractsAndDiagnostics(t *testing.T) {
	vm := newRelationalTestVM()
	raw, err := json.Marshal(minitrace.BuildSessionSkeleton("negative", "codex", "test", "test"))
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.Set("archive", string(raw)); err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`
 const mt=require("minitrace");
 function check(ok,message) { if (!ok) throw Error(message); }
 const db=mt.db().sources(mt.sources().content(archive).build()).build();
 try {
  for (const plan of [mt.sql` + "`" + `SELECT missing FROM sessions` + "`" + `.build(),
     mt.sql` + "`" + `SELECT name FROM sqlite_master` + "`" + `.build(),
     {sql:"SELECT 1"}, "SELECT 1"]) {
   const result=db.execute(plan);
   check(result.error&&!result.complete&&result.count===0,'error completeness');
   check(Array.isArray(result.rows)&&result.rows.length===0,'rows array');
   check(Array.isArray(result.columns)&&result.columns.length===0,'columns array');
   const roundtrip=JSON.parse(JSON.stringify(result));
   check(Array.isArray(roundtrip.rows)&&Array.isArray(roundtrip.columns),'JSON arrays');
  }
  for (const name of ["query","queryOne","queryResult","_handle"]) check(!(name in db),name+" still exported");
  for (const name of ["cache","limits","importPolicy","query","view","session"]) check(!(name in mt),name+" still exported");
  check(!("Build" in mt.db())&&!("File" in mt.sources()),"uppercase methods still exported");
  let rejected=false;
  try{mt.sql` + "`" + `DELETE FROM sessions` + "`" + `;}catch(e){rejected=String(e).includes("only SELECT");}
  check(rejected,"writes rejected at build");
 } finally {db.close();}
 for (const auto of [false,true]) {
  const sources=mt.sources().content('{"id":"bad","turns":[],"metrics":[]}').build();
  let message='';
  try{mt.db().sources(sources).import({autoConvert:auto}).build();}catch(e){message=String(e);}
  check(message.includes('metrics')&&message.includes('cannot unmarshal array')&&!message.includes('AutoConvert is disabled'),'typed normalized decode diagnostic');
 }
 `)
	if err != nil {
		t.Fatal(err)
	}
}
