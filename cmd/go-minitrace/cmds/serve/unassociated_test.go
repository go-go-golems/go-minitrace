package serve

import (
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"testing"
)

func TestUnassociatedRecordsRemainVisibleWithoutInventingTurns(t *testing.T) {
	session := minitrace.BuildSessionSkeleton("id", "codex", "test", "test")
	session.ToolCalls = []minitrace.ToolCall{{ID: "native", RecordKind: "execution", FrameworkMetadata: map[string]any{"parent_association": "unknown", "argv": []string{"sh", "-lc", "pwd"}}}}
	detail := normalizeSessionDetail(session)
	if len(detail.UnassociatedToolCalls) != 1 || len(detail.Blocks) != 0 {
		t.Fatal("unassociated execution lost or fake turn created")
	}
	wire, err := protoSessionDetail(detail)
	if err != nil {
		t.Fatal(err)
	}
	if len(wire.UnassociatedToolCalls) != 1 || wire.UnassociatedToolCalls[0].FrameworkMetadata.AsMap()["parent_association"] != "unknown" {
		t.Fatal("unassociated provenance lost in proto")
	}
}
