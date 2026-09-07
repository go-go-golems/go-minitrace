package minitrace

import "testing"

func TestActivityCountsSeparateInvocationsExecutionsAndEffects(t *testing.T) {
	success := true
	calls := []ToolCall{
		{ID: "ordinary"},
		{ID: "wrapper", RecordKind: RecordKindOrchestration, OperationType: "EXECUTE"},
		{ID: "child", RecordKind: RecordKindExecution, OperationType: "EXECUTE", Input: ToolCallInput{FileTargets: []FileTarget{{Status: "attempted"}, {Status: "attempted"}}}},
		{ID: "direct", RecordKind: RecordKindExecution, OperationType: "EXECUTE", FrameworkMetadata: map[string]any{"response_call_id": "direct"}},
		{ID: "effect", RecordKind: RecordKindFileChange, Input: ToolCallInput{FileTargets: []FileTarget{{Status: "confirmed", Success: &success}}}},
	}
	metrics := ComputeMetrics(nil, calls, Timing{}, 0, nil)
	if metrics.ToolCallCount != 5 || metrics.ExecuteCount != 2 {
		t.Fatalf("wrapper inflated executions: %#v", metrics)
	}
	want := ActivityCounts{ToolCallRecordCount: 1, OrchestrationCount: 1, ExecutionRecordCount: 2, FileChangeCount: 1, ModelInvocationCount: 3, FileTouchCount: 3, ConfirmedFileTargetCount: 1}
	if metrics.ActivityCounts != want {
		t.Fatalf("got %+v want %+v", metrics.ActivityCounts, want)
	}
}
