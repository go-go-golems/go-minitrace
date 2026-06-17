package copilot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertParsedMapsTurnsToolsPermissionsAndShutdown(t *testing.T) {
	parsed := parseJSONLBytes([]byte(minimalCopilotJSONL()))
	workspace := &WorkspaceMetadata{
		ID:         "workspace-id",
		Name:       "Session Initialization",
		CWD:        "/tmp/project",
		Branch:     "main",
		Repository: "example/repo",
		Raw:        map[string]any{"id": "workspace-id", "cwd": "/tmp/project"},
	}

	session, err := ConvertParsed(parsed, workspace, "fallback", "/tmp/session/events.jsonl")
	if err != nil {
		t.Fatalf("ConvertParsed returned error: %v", err)
	}

	if session.ID != "sess-1" {
		t.Fatalf("expected session.start ID, got %q", session.ID)
	}
	if session.Environment.AgentFramework == nil || *session.Environment.AgentFramework != "copilot" {
		t.Fatalf("expected copilot framework, got %+v", session.Environment.AgentFramework)
	}
	if session.Environment.AgentVersion == nil || *session.Environment.AgentVersion != "1.0.0" {
		t.Fatalf("expected agent version, got %+v", session.Environment.AgentVersion)
	}
	if session.Environment.Model == nil || *session.Environment.Model != "gpt-test" {
		t.Fatalf("expected latest model gpt-test, got %+v", session.Environment.Model)
	}
	if session.OperationalContext.WorkingDirectory == nil || *session.OperationalContext.WorkingDirectory != "/tmp/project" {
		t.Fatalf("unexpected working directory: %+v", session.OperationalContext.WorkingDirectory)
	}
	if len(session.Turns) != 2 {
		t.Fatalf("expected user+assistant turns, got %d", len(session.Turns))
	}
	if session.Turns[0].Role != "user" || session.Turns[0].Content != "Inspect README.md" {
		t.Fatalf("unexpected user turn: %+v", session.Turns[0])
	}
	if session.Turns[1].Role != "assistant" || session.Turns[1].Content != "The README has a title." {
		t.Fatalf("unexpected assistant turn: %+v", session.Turns[1])
	}
	if len(session.Turns[1].ToolCallsInTurn) != 1 || session.Turns[1].ToolCallsInTurn[0] != "tool-1" {
		t.Fatalf("expected assistant turn to reference tool-1, got %+v", session.Turns[1].ToolCallsInTurn)
	}
	if len(session.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(session.ToolCalls))
	}
	tool := session.ToolCalls[0]
	if tool.ToolName != "bash" || tool.OperationType != "READ" {
		t.Fatalf("unexpected tool classification: %+v", tool)
	}
	if tool.Input.Command == nil || *tool.Input.Command != "cat README.md" {
		t.Fatalf("expected command, got %+v", tool.Input.Command)
	}
	if tool.Output.Result == nil || *tool.Output.Result != "# Example" || !tool.Output.Success {
		t.Fatalf("unexpected tool output: %+v", tool.Output)
	}
	if tool.EmittingTurnIndex == nil || *tool.EmittingTurnIndex != 1 {
		t.Fatalf("expected emitting turn index 1, got %+v", tool.EmittingTurnIndex)
	}
	if len(session.Events) < 4 {
		t.Fatalf("expected lifecycle/permission events, got %d", len(session.Events))
	}
	if session.Metrics.TotalInputTokens == nil || *session.Metrics.TotalInputTokens != 100 {
		t.Fatalf("expected shutdown input tokens, got %+v", session.Metrics.TotalInputTokens)
	}
	if session.Metrics.TotalOutputTokens == nil || *session.Metrics.TotalOutputTokens != 20 {
		t.Fatalf("expected shutdown output tokens, got %+v", session.Metrics.TotalOutputTokens)
	}
	if session.Metrics.TotalCacheReadTokens == nil || *session.Metrics.TotalCacheReadTokens != 5 {
		t.Fatalf("expected shutdown cache read tokens, got %+v", session.Metrics.TotalCacheReadTokens)
	}
}

func TestConvertParsedAnnotatesMalformedJSONLines(t *testing.T) {
	parsed := parseJSONLBytes([]byte(minimalCopilotJSONL() + "{not json}\n"))
	if len(parsed.BadLines) != 1 {
		t.Fatalf("expected one bad line, got %#v", parsed.BadLines)
	}
	session, err := ConvertParsed(parsed, nil, "sess-bad", "/tmp/events.jsonl")
	if err != nil {
		t.Fatalf("ConvertParsed returned error: %v", err)
	}
	found := false
	for _, annotation := range session.Annotations {
		if annotation.Content.Category == "data-quality" && annotation.Scope.Type == "session" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected data-quality annotation for malformed line, got %+v", session.Annotations)
	}
}

func TestConvertLocatorReadsWorkspaceNextToEvents(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "session-state", "sess-1")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "workspace.yaml"), []byte("id: workspace-id\ncwd: /tmp/project\nbranch: feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte(minimalCopilotJSONL()), 0o644); err != nil {
		t.Fatal(err)
	}
	locators, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	session, err := ConvertLocator(locators[0])
	if err != nil {
		t.Fatalf("ConvertLocator returned error: %v", err)
	}
	if session.OperationalContext.GitBranch == nil || *session.OperationalContext.GitBranch != "feature" {
		t.Fatalf("expected workspace branch, got %+v", session.OperationalContext.GitBranch)
	}
}

func minimalCopilotJSONL() string {
	return `{"type":"session.start","id":"ev-1","timestamp":"2026-06-17T10:00:00Z","parentId":null,"data":{"sessionId":"sess-1","copilotVersion":"1.0.0","context":{"cwd":"/tmp/project","branch":"main","repository":"example/repo"}}}
{"type":"user.message","id":"ev-2","timestamp":"2026-06-17T10:00:01Z","parentId":"ev-1","data":{"content":"Inspect README.md","transformedContent":"Inspect README.md","interactionId":"int-1","attachments":[]}}
{"type":"assistant.turn_start","id":"ev-3","timestamp":"2026-06-17T10:00:02Z","parentId":"ev-2","data":{"turnId":"turn-1","interactionId":"int-1"}}
{"type":"permission.requested","id":"ev-4","timestamp":"2026-06-17T10:00:03Z","parentId":"ev-3","data":{"requestId":"perm-1","permissionRequest":{"toolCallId":"tool-1","fullCommandText":"cat README.md","intention":"Read the README","kind":"command","possiblePaths":["README.md"],"hasWriteFileRedirection":false}}}
{"type":"tool.execution_start","id":"ev-5","timestamp":"2026-06-17T10:00:04Z","parentId":"ev-4","data":{"toolCallId":"tool-1","toolName":"bash","turnId":"turn-1","model":"gpt-test","arguments":{"command":"cat README.md","description":"Read the README","mode":"default"}}}
{"type":"tool.execution_complete","id":"ev-6","timestamp":"2026-06-17T10:00:05Z","parentId":"ev-5","data":{"toolCallId":"tool-1","turnId":"turn-1","success":true,"result":{"content":"# Example","detailedContent":"# Example"},"toolTelemetry":{"metrics":{"commandTimeout":0}}}}
{"type":"permission.completed","id":"ev-7","timestamp":"2026-06-17T10:00:06Z","parentId":"ev-6","data":{"requestId":"perm-1","toolCallId":"tool-1","result":{"kind":"allow"}}}
{"type":"assistant.message","id":"ev-8","timestamp":"2026-06-17T10:00:07Z","parentId":"ev-7","data":{"turnId":"turn-1","interactionId":"int-1","messageId":"msg-1","model":"gpt-test","phase":"final","content":"The README has a title.","encryptedContent":"secret","reasoningOpaque":"opaque","outputTokens":7,"toolRequests":[]}}
{"type":"assistant.turn_end","id":"ev-9","timestamp":"2026-06-17T10:00:08Z","parentId":"ev-8","data":{"turnId":"turn-1"}}
{"type":"session.shutdown","id":"ev-10","timestamp":"2026-06-17T10:00:09Z","parentId":"ev-9","data":{"currentModel":"gpt-test","tokenDetails":{"input":{"tokenCount":100},"output":{"tokenCount":20},"cache_read":{"tokenCount":5}},"shutdownType":"user"}}
`
}
