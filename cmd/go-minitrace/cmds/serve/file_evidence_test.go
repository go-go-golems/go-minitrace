package serve

import (
	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

func TestFileEvidenceSurvivesAPIAndProto(t *testing.T) {
	call := minitrace.ToolCall{ID: "execution", RecordKind: minitrace.RecordKindExecution, Input: minitrace.ToolCallInput{FileTargets: []minitrace.FileTarget{{Path: "/cwd/one", NativePath: "one", OperationType: "MODIFY", EvidenceKind: "shell_redirect", Status: "attempted", CWD: "/cwd", Resolved: true, SourceReference: "native#L4"}}}}
	call.Output.SetSuccess(true)
	normalized := normalizeToolCall(call)
	wire, err := protoToolCall(normalized)
	if err != nil {
		t.Fatal(err)
	}
	data, err := protojson.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &apiv1.ToolCall{}
	if err := protojson.Unmarshal(data, decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RecordKind != "execution" || len(decoded.Input.FileTargets) != 1 {
		t.Fatal("record kind or targets lost")
	}
	target := decoded.Input.FileTargets[0]
	if target.Success != nil || target.Status != "attempted" || target.SourceReference != "native#L4" || !target.Resolved {
		t.Fatalf("file evidence changed: %v", target)
	}
	session := minitrace.BuildSessionSkeleton("id", "codex", "test", "test")
	session.ToolCalls = []minitrace.ToolCall{call}
	summary := normalizeSessionSummaryDetail(session)
	if summary.Metrics.ExecutionRecordCount != 1 || summary.Metrics.FileTouchCount != 1 || summary.Metrics.ConfirmedFileTargetCount != 0 {
		t.Fatalf("summary counted serialized zero instead of records: %+v", summary.Metrics)
	}
}
