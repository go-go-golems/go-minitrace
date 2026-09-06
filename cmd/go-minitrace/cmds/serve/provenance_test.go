package serve

import (
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"math"
	"testing"
)

func TestToolProvenanceAndTruncationSurviveAPI(t *testing.T) {
	ref, hash, size := "native#L12", "sha256:test", 20000
	call := minitrace.ToolCall{ID: "native", FrameworkMetadata: map[string]any{"argv": []string{"sh", "-lc", "pwd"}, "parent_association": "unknown", "source_line": 12}, Output: minitrace.ToolCallOutput{FullReference: &ref, FullHash: &hash, FullBytes: &size, Truncated: true}}
	wire, err := protoToolCall(normalizeToolCall(call))
	if err != nil {
		t.Fatal(err)
	}
	if wire.FrameworkMetadata.AsMap()["parent_association"] != "unknown" || len(wire.FrameworkMetadata.AsMap()["argv"].([]any)) != 3 {
		t.Fatal("provenance lost")
	}
	if wire.Output.GetFullReference() != ref || wire.Output.GetFullHash() != hash || wire.Output.GetFullBytes() != 20000 {
		t.Fatal("truncation references lost")
	}
	call.FrameworkMetadata = map[string]any{"invalid": math.NaN()}
	if _, err := protoToolCall(normalizeToolCall(call)); err == nil {
		t.Fatal("invalid provenance silently dropped")
	}
}
