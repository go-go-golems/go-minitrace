package minitracedb

import (
	"context"
	"database/sql"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"testing"
)

func TestStructuralFileTargetsDoNotInheritShellSuccess(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	session := minitrace.BuildSessionSkeleton("files", "codex", "synthetic", "test")
	path := "unsafe-convenience.txt"
	call := minitrace.ToolCall{ID: "exec", RecordKind: minitrace.RecordKindExecution, ToolName: "exec_command", OperationType: "EXECUTE", Input: minitrace.ToolCallInput{FilePath: &path, FileTargets: []minitrace.FileTarget{
		{Path: "/cwd/first", NativePath: "first", OperationType: "MODIFY", EvidenceKind: "shell_redirect", Status: "attempted", CWD: "/cwd", Resolved: true, SourceReference: "source#L3"},
		{Path: "/cwd/second", NativePath: "second", OperationType: "MODIFY", EvidenceKind: "shell_redirect", Status: "attempted", CWD: "/cwd", Resolved: true, SourceReference: "source#L3"},
	}}}
	call.Output.SetSuccess(true)
	session.ToolCalls = []minitrace.ToolCall{call, {ID: "wrapper", RecordKind: minitrace.RecordKindOrchestration, Input: minitrace.ToolCallInput{FilePath: &path, FileTargets: []minitrace.FileTarget{}}}}
	if err := MaterializeSession(ctx, db, &session, MaterializeOptions{}); err != nil {
		t.Fatal(err)
	}
	var count, known int
	if err := db.QueryRowContext(ctx, "SELECT count(*),count(success) FROM files").Scan(&count, &known); err != nil {
		t.Fatal(err)
	}
	if count != 2 || known != 0 {
		t.Fatalf("targets/count/outcomes corrupted: count=%d known=%d", count, known)
	}
	var kind, status, source string
	if err := db.QueryRowContext(ctx, "SELECT tc.record_kind,f.evidence_status,f.source_reference FROM files f JOIN tool_calls tc ON tc.tool_call_id=f.tool_call_id WHERE f.target_ordinal=1").Scan(&kind, &status, &source); err != nil {
		t.Fatal(err)
	}
	if kind != "execution" || status != "attempted" || source != "source#L3" {
		t.Fatalf("provenance lost: %s %s %s", kind, status, source)
	}
}
