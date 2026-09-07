package minitrace

// ActivityCounts separates model invocations, observed effects and file-target
// evidence. FileTouchCount counts target rows, not unique paths or proven writes.
type ActivityCounts struct {
	ToolCallRecordCount      int `json:"tool_call_record_count"`
	OrchestrationCount       int `json:"orchestration_count"`
	ExecutionRecordCount     int `json:"execution_record_count"`
	FileChangeCount          int `json:"file_change_count"`
	ModelInvocationCount     int `json:"model_invocation_count"`
	FileTouchCount           int `json:"file_touch_count"`
	ConfirmedFileTargetCount int `json:"confirmed_file_target_count"`
}

func CountToolActivity(calls []ToolCall) ActivityCounts {
	counts := ActivityCounts{}
	for _, call := range calls {
		switch call.EffectiveRecordKind() {
		case RecordKindOrchestration:
			counts.OrchestrationCount++
			counts.ModelInvocationCount++
		case RecordKindExecution:
			counts.ExecutionRecordCount++
			if metadata, ok := call.FrameworkMetadata.(map[string]any); ok && metadata["response_call_id"] == call.ID {
				counts.ModelInvocationCount++
			}
		case RecordKindFileChange:
			counts.FileChangeCount++
		default:
			counts.ToolCallRecordCount++
			counts.ModelInvocationCount++
		}
		for _, target := range call.EffectiveFileTargets() {
			counts.FileTouchCount++
			if target.Status == "confirmed" && target.Success != nil && *target.Success {
				counts.ConfirmedFileTargetCount++
			}
		}
	}
	return counts
}
