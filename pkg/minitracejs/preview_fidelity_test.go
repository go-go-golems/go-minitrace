package minitracejs

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

func TestFileTargetPreviewPreservesEvidenceAndPrivacy(t *testing.T) {
	session := minitrace.BuildSessionSkeleton("preview", "codex", "test", "test")
	original := minitrace.FileTarget{Path: "/private/target", NativePath: "private-target", CWD: "/private", SourceReference: "/private/source#L1", OperationType: "MODIFY", EvidenceKind: "shell_redirect", Status: "attempted", Resolved: true}
	session.ToolCalls = []minitrace.ToolCall{{ID: "native", RecordKind: "execution", Input: minitrace.ToolCallInput{FileTargets: []minitrace.FileTarget{original}}, Output: minitrace.ToolCallOutput{Status: minitrace.ToolOutcomePending}}}
	loaded := &minitracedb.LoadedSession{Session: &session}
	full := PreviewLoadedSessionWithOptions(loaded, PreviewOptions{Privacy: "full"})
	if full.SampleTools[0].RecordKind != "execution" || full.SampleTools[0].OutcomeStatus != "pending" || full.SampleTools[0].FileTargets[0] != original {
		t.Fatalf("preview evidence lost: %+v", full.SampleTools)
	}
	structural := PreviewLoadedSessionWithOptions(loaded, PreviewOptions{Privacy: "structural"})
	payload, err := json.Marshal(structural.SampleTools)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "private") {
		t.Fatalf("structural preview leaked paths: %s", payload)
	}
	target := structural.SampleTools[0].FileTargets[0]
	if target.Status != "attempted" || target.Success != nil || target.EvidenceKind != "shell_redirect" {
		t.Fatal("privacy erased structural evidence")
	}
	if session.ToolCalls[0].Input.FileTargets[0] != original {
		t.Fatal("preview mutated original source")
	}
}
