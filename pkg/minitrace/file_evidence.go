package minitrace

const (
	RecordKindToolCall      = "tool_call"
	RecordKindOrchestration = "orchestration"
	RecordKindExecution     = "execution"
	RecordKindFileChange    = "file_change"
)

// FileTarget records evidence about a target, not a claim that every statement
// of a shell script ran. Attempted targets have unknown success. Only explicit
// native file-effect evidence can establish a confirmed effect.
type FileTarget struct {
	Path            string `json:"path"`
	NativePath      string `json:"native_path,omitempty"`
	OperationType   string `json:"operation_type"`
	EvidenceKind    string `json:"evidence_kind"`
	Status          string `json:"status"`
	Success         *bool  `json:"success"`
	CWD             string `json:"cwd,omitempty"`
	Resolved        bool   `json:"resolved"`
	SourceReference string `json:"source_reference,omitempty"`
}

func (call ToolCall) EffectiveRecordKind() string {
	if call.RecordKind != "" {
		return call.RecordKind
	}
	if metadata, ok := call.FrameworkMetadata.(map[string]any); ok {
		if kind, ok := metadata["record_kind"].(string); ok && kind != "" {
			return kind
		}
	}
	return RecordKindToolCall
}

// EffectiveFileTargets preserves pre-structural adapters' scalar reporting.
// A non-nil empty list means explicitly no structural evidence; it must not
// fall back to a scalar convenience path or scan opaque tool arguments.
func (call ToolCall) EffectiveFileTargets() []FileTarget {
	if call.Input.FileTargets != nil {
		return call.Input.FileTargets
	}
	if call.Input.FilePath == nil || *call.Input.FilePath == "" {
		return nil
	}
	return []FileTarget{{Path: *call.Input.FilePath, OperationType: call.OperationType,
		EvidenceKind: "legacy_scalar", Status: "reported", Success: call.Output.Success}}
}
