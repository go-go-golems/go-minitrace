package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// Exercise the public JavaScript API on materialized archives, not SQL mocks.
func TestGojaFidelityViews(t *testing.T) {
	var paths []string
	for _, id := range []string{"a", "b"} {
		session := minitrace.BuildSessionSkeleton(id, "codex", "test", "test")
		zero, five := 0, 5
		session.Turns = []minitrace.Turn{minitrace.BuildTurn(0, nil, "assistant", nil, "first")}
		for _, entry := range []struct {
			id     string
			status minitrace.ToolOutcomeStatus
			turn   *int
		}{
			{"failed", minitrace.ToolOutcomeFailed, &zero},
			{"unknown", minitrace.ToolOutcomeUnknown, &zero},
			{"pending", minitrace.ToolOutcomePending, &zero},
			{"cancelled", minitrace.ToolOutcomeCancelled, &zero},
			{"ok", minitrace.ToolOutcomeSucceeded, &five},
			{"detached", minitrace.ToolOutcomePending, nil},
		} {
			call := minitrace.ToolCall{ID: entry.id, RecordKind: "execution", ToolName: "exec_command", OperationType: "EXECUTE", EmittingTurnIndex: entry.turn, Output: minitrace.ToolCallOutput{Status: entry.status}, FrameworkMetadata: map[string]any{"parent_association": "unknown"}}
			if entry.status == minitrace.ToolOutcomeFailed || entry.status == minitrace.ToolOutcomeSucceeded {
				call.Output.SetSuccess(entry.status == minitrace.ToolOutcomeSucceeded)
			}
			if entry.id == "failed" {
				call.Input.FileTargets = []minitrace.FileTarget{{Path: "/private/first", NativePath: "first", CWD: "/private", SourceReference: "/private/source#L4", Status: "attempted", EvidenceKind: "shell_redirect", OperationType: "MODIFY"}, {Path: "/private/second", Status: "attempted", EvidenceKind: "shell_redirect", OperationType: "MODIFY"}}
				ref, hash, size := "source#L4", "sha256:test", 12000
				call.Output.FullReference = &ref
				call.Output.FullHash = &hash
				call.Output.FullBytes = &size
			}
			session.ToolCalls = append(session.ToolCalls, call)
		}
		session.Turns[0].ToolCallsInTurn = []string{"failed", "unknown", "pending", "cancelled", "failed"}
		last := minitrace.BuildTurn(5, nil, "assistant", nil, "last")
		last.ToolCallsInTurn = []string{"ok"}
		session.Turns = append(session.Turns, last)
		session.Annotations = []minitrace.Annotation{
			{ID: "note1", Scope: minitrace.AnnotationScope{Type: "turn", TargetID: "0"}},
			{ID: "note2", Scope: minitrace.AnnotationScope{Type: "turn", TargetID: "0"}},
			{ID: "tool-note", Scope: minitrace.AnnotationScope{Type: "tool_call", TargetID: "5"}},
		}
		session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, 0, nil)
		payload, err := json.Marshal(session)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), id+".minitrace.json")
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	mod := resolveModule(t)
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	module := vm.NewObject()
	exports := vm.NewObject()
	if err := module.Set("exports", exports); err != nil {
		t.Fatal(err)
	}
	loader(vm, module)
	if err := vm.Set("mt", exports); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("paths", paths); err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`
 const check=(condition,message)=>{if(!condition)throw new Error(message)};
 const db=mt.db().File(paths[0]).File(paths[1]).Build();
 try {
  const recipe=builder=>{const r=builder.Build();return db.query(r.sql(),...r.args())};
  const tools=recipe(mt.query().ToolRows().SessionID('a'));
  check(tools.length===6,'tool rows lost');
  const failed=tools.find(t=>t.tool_call_id==='failed');
  check(failed.record_kind==='execution' && failed.outcome_status==='failed','tool kind/status missing');
  check(failed.full_hash==='sha256:test' && failed.full_reference==='source#L4','output provenance lost');
  check(JSON.parse(failed.framework_metadata_json).parent_association==='unknown','framework metadata lost');
  check(JSON.parse(failed.file_targets_json).length===2,'nested targets lost');
  const files=recipe(mt.query().FileRows().SessionID('a'));
  check(files.length===2 && files.every(f=>f.success===null && f.evidence_status==='attempted'),'file evidence lost');
  const summary=mt.view().DB(db).SessionID('a').SessionSummary().Run();
  check(summary.execution_record_count===6 && summary.file_touch_count===2 && summary.confirmed_file_target_count===0,'summary counters lost');
  const timeline=mt.view().DB(db).SessionID('a').Timeline().Run();
  const first=timeline.find(t=>t.turn_index===0),last=timeline.find(t=>t.turn_index===5);
  check(first.tool_call_count===4 && first.failed_tool_count===1 && first.unknown_tool_count===3,'join multiplied tool outcomes');
  check(first.file_count===2 && first.file_touch_count===2 && first.confirmed_file_target_count===0,'join multiplied targets');
  check(first.has_annotations===1 && last.has_annotations===0,'annotation scopes conflated');
  const transcript=recipe(mt.query().TranscriptRows().IncludeTools());
  check(transcript.filter(r=>r.role==='tool').length===12,'memberships duplicated transcript tools');
  check(transcript.filter(r=>r.tool_call_id==='detached').every(r=>r.turn_index===null),'transcript fabricated turn zero');
  const frames=mt.view().DB(db).TurnFrames().Run();
  check(frames.length===6,'sessions or null turns conflated');
  for(const id of ['a','b']) {
   const frame=frames.find(f=>f.sessionId===id && f.turnIndex===0);
   check(frame.stats.toolCalls===4 && frame.stats.failedToolCalls===1 && frame.stats.unknownToolCalls===3,'frame outcomes wrong');
   const detached=frames.find(f=>f.sessionId===id && f.unassociated);
   check(detached.turnIndex===null && detached.toolCalls.length===1 && detached.stats.unknownToolCalls===1,'unassociated frame lost');
  }
  const preview=mt.importer().File(paths[0]).Preview();
  check(preview.sampleTools[0].recordKind==='execution' && preview.sampleTools[0].fileTargets.length===2,'preview evidence lost');
 } finally {db.close()}
 `)
	if err != nil {
		t.Fatal(err)
	}
}
