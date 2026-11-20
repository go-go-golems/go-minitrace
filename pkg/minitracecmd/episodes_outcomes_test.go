package minitracecmd

import (
	"github.com/dop251/goja"
	"os"
	"testing"
)

func TestEpisodesDistinguishesUnknownOutcomesAndUnassociatedCalls(t *testing.T) {
	source, err := os.ReadFile("../../skills/transcript-doc-friction-analysis/query-commands/docmetrics/episodes.js")
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	script := `
 let closed=false;
 const users=[{session_id:'s',turn_index:0,content:'Start work'},{session_id:'s',turn_index:5,content:'Next task'}];
 const calls=[true,1,false,0,null,null,null].map(success=>({session_id:'s',turn:1,tool_name:'shell',timestamp:'2026-09-01T00:00:00Z',success}));
 calls.push({session_id:'s',turn:null,tool_name:'shell',success:false});
 calls.push({session_id:'s',turn:6,tool_name:'shell',success:0});
 const sourceBuilder={runtimeArchives(){return this},build(){return {}}};
 const db={sources(){return this},import(){return this},limits(){return this},build(){return this},
   execute(sql){return {complete:true,rows:sql.includes('FROM turns')?users:calls}},
   close(){closed=true}};
 const require=()=>({db:()=>db,sources:()=>sourceBuilder,sql:parts=>({build:()=>parts[0]})});
 const __verb__=()=>{};
 `
	result, err := vm.RunString(script + string(source) + "\nepisodes();")
	if err != nil {
		t.Fatal(err)
	}
	rows := result.ToObject(vm)
	if rows.Get("length").ToInteger() != 2 {
		t.Fatal("wrong episode count")
	}
	first := rows.Get("0").ToObject(vm)
	second := rows.Get("1").ToObject(vm)
	if first.Get("tool_calls").ToInteger() != 7 || first.Get("failures").ToInteger() != 2 || first.Get("unknown_outcomes").ToInteger() != 3 {
		t.Fatalf("unknown outcomes or null turn miscounted: %v", result.Export())
	}
	if second.Get("failures").ToInteger() != 1 || second.Get("unknown_outcomes").ToInteger() != 0 {
		t.Fatalf("episode boundary lost: %v", result.Export())
	}
	closed, err := vm.RunString("closed")
	if err != nil || !closed.ToBoolean() {
		t.Fatal("database not closed")
	}
}
