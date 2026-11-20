package minitracejs

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestAnalysisTokenOriginsAndSourceProjection(t *testing.T) {
	n := 0
	zero := 0
	body := "éééé"
	yes := true
	s := &minitrace.Session{ID: "s", Turns: []minitrace.Turn{{Index: 0, Role: "assistant", Content: "known zero", Usage: &minitrace.Usage{InputTokens: &zero}}, {Index: 1, Role: "user", Content: "unknown"}}, ToolCalls: []minitrace.ToolCall{{ID: "z-first", EmittingTurnIndex: &n, Input: minitrace.ToolCallInput{Arguments: map[string]any{"q": 1}}, Output: minitrace.ToolCallOutput{Result: &body}}, {ID: "a-second", Output: minitrace.ToolCallOutput{Success: &yes, Status: minitrace.ToolOutcomeStatus("succeeded")}}}}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../examples/query-library/analysis.js")
	if err != nil {
		t.Fatal(err)
	}
	vm := newRelationalTestVM()
	if err := vm.Set("archive", string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunScript("analysis.js", "const analysis = (() => { const exports = {};\n"+
		string(source)+"\nreturn exports; })();"); err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`
	 const mt=require("minitrace");
	 const db=mt.db().sources(mt.sources().content(archive).build()).build();
	 const check=(passed,message)=>{if(!passed)throw Error(message)};
	 try {
	  const rows=q=>{
	   const result=db.execute(q.build());
	   check(result.complete&&!result.error,result.error);
	   return result.rows;
	  };
	  const turns=rows(analysis.turnTokenUsage("s"));
	  check(turns[0].token_origin==="recorded_usage"&&turns[1].token_origin==="unknown","known zero vs missing");
	  for(const query of [analysis.sessionTokenUsage("s"),analysis.roleTokenUsage("s")]) {
	   const result=rows(query);
	   check(result.length>0&&result[0].token_unit==="tokens","aggregate provenance");
	  }
	  const tools=rows(analysis.toolTokenUsage("s"));
	  const estimate=tools.find(row=>row.tool_call_id==="z-first");
	  check(estimate.total_tokens===2&&estimate.token_origin==="estimated"&&estimate.token_method==="round_utf8_bytes_div_4","UTF-8 estimate");
	  const sourceOrder=rows(analysis.toolRows("s").orderBy(t=>t.source_ordinal.asc()));
	  check(sourceOrder[0].source_ordinal===0&&sourceOrder[0].arguments_json==='{"q":1}'&&sourceOrder[0].success===null&&sourceOrder[0].output_status===null,"tool provenance");
	  for(const row of rows(analysis.transcriptRows({sessionId:"s",includeTools:true}))) {
	   check(row.role!=="tool"||row.token_unit==="estimated_tokens","unlabeled transcript estimate");
	  }
	  rows(analysis.timelineRows("s"));
	 } finally {db.close();}
	`)
	if err != nil {
		t.Fatal(err)
	}
}
