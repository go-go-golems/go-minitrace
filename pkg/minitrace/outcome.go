package minitrace

// ToolOutcomeStatus distinguishes absence of evidence from a failing tool.
// Success is non-null only for a known succeeded/failed outcome.
type ToolOutcomeStatus string

const (
	ToolOutcomeUnknown   ToolOutcomeStatus = "unknown"
	ToolOutcomePending   ToolOutcomeStatus = "pending"
	ToolOutcomeSucceeded ToolOutcomeStatus = "succeeded"
	ToolOutcomeFailed    ToolOutcomeStatus = "failed"
	ToolOutcomeCancelled ToolOutcomeStatus = "cancelled"
)

// Tool outcome reasons refine a known outcome with why the tool did or did
// not run. They are set only when the source states the reason; most tool
// calls have none. Every reason except interrupted means the tool never
// executed, so the call's duration measures waiting, not work.
const (
	ToolOutcomeReasonHookBlocked         = "hook_blocked"
	ToolOutcomeReasonPermissionDenied    = "permission_denied"
	ToolOutcomeReasonClassifierDenied    = "classifier_denied"
	ToolOutcomeReasonClassifierNoVerdict = "classifier_no_verdict"
	ToolOutcomeReasonUserRejected        = "user_rejected"
	ToolOutcomeReasonInterrupted         = "interrupted"
	ToolOutcomeReasonToolUseError        = "tool_use_error"
	ToolOutcomeReasonHarnessBlocked      = "harness_blocked"
)

// SetSuccess records a known binary outcome and keeps its status consistent.
func (output *ToolCallOutput) SetSuccess(success bool) {
	output.Success = &success
	output.Status = ToolOutcomeFailed
	if success {
		output.Status = ToolOutcomeSucceeded
	}
}

// OutcomeStatus returns the evidence-based state. A nullable success alone
// establishes a binary outcome; without one only pending/cancelled status
// establishes additional lifecycle information. No completion claim invents a
// successful result when the native outcome is absent.
func (output ToolCallOutput) OutcomeStatus() ToolOutcomeStatus {
	if output.Success != nil {
		if *output.Success {
			return ToolOutcomeSucceeded
		}
		return ToolOutcomeFailed
	}
	switch output.Status {
	case ToolOutcomePending, ToolOutcomeCancelled:
		return output.Status
	case ToolOutcomeUnknown, ToolOutcomeSucceeded, ToolOutcomeFailed, "":
		return ToolOutcomeUnknown
	default:
		return ToolOutcomeUnknown
	}
}

func (output ToolCallOutput) Failed() bool {
	return output.Success != nil && !*output.Success
}

func (output ToolCallOutput) Succeeded() bool {
	return output.Success != nil && *output.Success
}
