package claudecode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// signalState collects the Claude Code signals that need cross-record context:
// which record uuid carried which tool result (to attribute guidance loads to
// the tool call that triggered them), which tool call ids exist (to scope hook
// events), and which calls a PreToolUse hook blocked.
type signalState struct {
	toolResultByRecord map[string]string
	attachmentParent   map[string]string
	knownToolCalls     map[string]struct{}
	hookBlocked        map[string]struct{}
	contextLoads       []minitrace.ContextLoad
	compactions        int
	turnDurationMS     int
	turnDurations      int
}

func newSignalState() *signalState {
	return &signalState{
		toolResultByRecord: map[string]string{},
		attachmentParent:   map[string]string{},
		knownToolCalls:     map[string]struct{}{},
		hookBlocked:        map[string]struct{}{},
	}
}

// triggeringToolCall follows parentUuid through attachment records to the
// tool-result record that caused an attachment, if any.
func (s *signalState) triggeringToolCall(record map[string]any) *string {
	current := stringValue(record["parentUuid"])
	for steps := 0; current != "" && steps < 256; steps++ {
		if toolCallID, ok := s.toolResultByRecord[current]; ok {
			return &toolCallID
		}
		parent, ok := s.attachmentParent[current]
		if !ok {
			return nil
		}
		current = parent
	}
	return nil
}

func (s *signalState) knownToolCall(id string) *string {
	if _, ok := s.knownToolCalls[id]; ok && id != "" {
		return &id
	}
	return nil
}

// addContextLoads records the guidance files carried by one attachment.
// nextTurnIndex is the index the next emitted turn will get.
func (s *signalState) addContextLoads(record map[string]any, attachmentID string, nextTurnIndex int) {
	attachment := mapValue(record["attachment"])
	timestamp := optionalString(stringValue(record["timestamp"]))
	add := func(load minitrace.ContextLoad) {
		load.ID = fmt.Sprintf("%s:%d", attachmentID, len(s.contextLoads))
		load.Timestamp = timestamp
		turnIndex := nextTurnIndex
		load.TurnIndex = &turnIndex
		id := attachmentID
		load.AttachmentID = &id
		s.contextLoads = append(s.contextLoads, load)
	}
	switch stringValue(attachment["type"]) {
	case "nested_memory":
		content := mapValue(attachment["content"])
		text := stringValue(content["content"])
		if content == nil {
			text = stringValue(attachment["content"])
		}
		load := newContextLoad(minitrace.ContextLoadNestedMemory, firstNonEmpty(stringValue(attachment["path"]), stringValue(content["path"])), text)
		load.DisplayPath = stringValue(attachment["displayPath"])
		load.FileType = stringValue(content["type"])
		if differs, ok := content["contentDiffersFromDisk"].(bool); ok {
			load.DiffersFromDisk = &differs
		}
		load.ToolCallID = s.triggeringToolCall(record)
		add(load)
	case "instructions":
		for _, item := range listValue(attachment["files"]) {
			file := mapValue(item)
			if file == nil {
				continue
			}
			load := newContextLoad(minitrace.ContextLoadInstructions, stringValue(file["path"]), stringValue(file["content"]))
			load.FileType = stringValue(file["type"])
			add(load)
		}
	case "invoked_skills":
		for _, item := range listValue(attachment["skills"]) {
			skill := mapValue(item)
			if skill == nil {
				continue
			}
			load := newContextLoad(minitrace.ContextLoadInvokedSkill, stringValue(skill["path"]), stringValue(skill["content"]))
			if name := stringValue(skill["name"]); name != "" {
				load.Name = name
			}
			add(load)
		}
	}
}

func newContextLoad(kind, path, content string) minitrace.ContextLoad {
	sum := sha256.Sum256([]byte(content))
	load := minitrace.ContextLoad{
		Kind:          kind,
		Path:          normalizeLoadPath(path),
		ContentChars:  utf8.RuneCountInString(content),
		ContentBytes:  len(content),
		ContentSHA256: hex.EncodeToString(sum[:]),
	}
	if path != "" {
		load.Name = filepath.Base(path)
	}
	return load
}

// normalizeLoadPath home-normalizes file paths and keeps non-path identifiers
// such as the skill locator "userSettings:diary" verbatim.
func normalizeLoadPath(path string) string {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~") {
		return minitrace.NormalizePath(path)
	}
	return path
}

// applyHookSignal turns a hook_* attachment event into a first-class hook
// event scoped to the tool call it concerns.
func (s *signalState) applyHookSignal(event *minitrace.Event, attachment map[string]any) {
	hookType := stringValue(attachment["type"])
	if !strings.HasPrefix(hookType, "hook_") {
		return
	}
	hookEvent := stringValue(attachment["hookEvent"])
	hookName := firstNonEmpty(stringValue(attachment["hookName"]), hookEvent, hookType)
	toolUseID := stringValue(attachment["toolUseID"])
	metadata := map[string]any{"hook_type": hookType}
	setNonEmpty(metadata, "hook_event", hookEvent)
	setNonEmpty(metadata, "hook_name", stringValue(attachment["hookName"]))
	setNonEmpty(metadata, "tool_use_id", toolUseID)
	setNonEmpty(metadata, "command", stringValue(attachment["command"]))
	if exitCode, ok := attachment["exitCode"]; ok {
		metadata["exit_code"] = minitrace.SafeInt(exitCode, 0)
	}
	if durationMS, ok := attachment["durationMs"]; ok {
		metadata["duration_ms"] = minitrace.SafeInt(durationMS, 0)
	}
	decision, reason := hookPermissionDecision(stringValue(attachment["stdout"]))
	setNonEmpty(metadata, "permission_decision", decision)
	setNonEmpty(metadata, "permission_decision_reason", reason)

	text := hookText(attachment["content"])
	severity := "info"
	switch hookType {
	case "hook_blocking_error":
		severity = "error"
		blocking := mapValue(attachment["blockingError"])
		text = firstNonEmpty(stringValue(blocking["blockingError"]), text)
		setNonEmpty(metadata, "command", stringValue(blocking["command"]))
		if hookEvent == "PreToolUse" && toolUseID != "" {
			s.hookBlocked[toolUseID] = struct{}{}
		}
	case "hook_non_blocking_error", "hook_error_during_execution", "hook_cancelled":
		severity = "warning"
		text = firstNonEmpty(text, stringValue(attachment["stderr"]))
	}

	event.Kind = "hook"
	event.Title = hookName
	event.Summary = strings.TrimSpace(strings.Join(nonEmpty(hookType, hookEvent, prefixed("decision=", decision)), " "))
	event.Text = text
	event.Severity = severity
	event.ToolCallID = s.knownToolCall(toolUseID)
	event.FrameworkMetadata = mergeMetadataMap(event.FrameworkMetadata, metadata)
}

// hookPermissionDecision extracts hookSpecificOutput.permissionDecision from a
// hook's JSON stdout.
func hookPermissionDecision(stdout string) (string, string) {
	stdout = strings.TrimSpace(stdout)
	if !strings.HasPrefix(stdout, "{") {
		return "", ""
	}
	var payload map[string]any
	if json.Unmarshal([]byte(stdout), &payload) != nil {
		return "", ""
	}
	output := mapValue(payload["hookSpecificOutput"])
	return stringValue(output["permissionDecision"]), stringValue(output["permissionDecisionReason"])
}

func hookText(value any) string {
	if items := listValue(value); items != nil {
		parts := make([]string, 0, len(items))
		for _, item := range items {
			if text := stringValue(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return stringValue(value)
}

// systemMarkerEvent maps the system subtypes that mark session structure
// rather than carry conversation. It returns false for other subtypes, which
// stay system turns.
func (s *signalState) systemMarkerEvent(record map[string]any, ordinal, nextTurnIndex int) (minitrace.Event, bool) {
	subtype := stringValue(record["subtype"])
	timestamp := optionalString(stringValue(record["timestamp"]))
	newEvent := func(kind, title, summary string, metadata map[string]any) minitrace.Event {
		event := minitrace.BuildEvent(fmt.Sprintf("claude-%s-%06d", strings.ReplaceAll(kind, "_", "-"), ordinal), timestamp, kind, title, summary, record)
		event.Role = "system"
		event.FrameworkMetadata = mergeMetadataMap(claudeRecordMetadata(record), metadata)
		return event
	}
	switch subtype {
	case "compact_boundary":
		compact := mapValue(record["compactMetadata"])
		metadata := map[string]any{}
		setNonEmpty(metadata, "trigger", stringValue(compact["trigger"]))
		for source, target := range map[string]string{"preTokens": "pre_tokens", "postTokens": "post_tokens", "durationMs": "duration_ms"} {
			if value, ok := compact[source]; ok {
				metadata[target] = minitrace.SafeInt(value, 0)
			}
		}
		event := newEvent("compaction", "Claude Code compaction", fmt.Sprintf("%s compaction %v -> %v tokens", firstNonEmpty(stringValue(compact["trigger"]), "unknown"), compact["preTokens"], compact["postTokens"]), metadata)
		turnIndex := nextTurnIndex
		event.TurnIndex = &turnIndex
		s.compactions++
		return event, true
	case "turn_duration":
		durationMS := minitrace.SafeInt(record["durationMs"], 0)
		metadata := map[string]any{"duration_ms": durationMS}
		if messageCount, ok := record["messageCount"]; ok {
			metadata["message_count"] = minitrace.SafeInt(messageCount, 0)
		}
		event := newEvent("turn_duration", "Claude Code turn duration", fmt.Sprintf("%d ms", durationMS), metadata)
		if nextTurnIndex > 0 {
			turnIndex := nextTurnIndex - 1
			event.TurnIndex = &turnIndex
		}
		s.turnDurationMS += durationMS
		s.turnDurations++
		return event, true
	case "stop_hook_summary":
		metadata := map[string]any{"hook_type": "stop_hook_summary"}
		commands := []string{}
		for _, item := range listValue(record["hookInfos"]) {
			if command := stringValue(mapValue(item)["command"]); command != "" {
				commands = append(commands, command)
			}
		}
		if len(commands) > 0 {
			metadata["commands"] = commands
		}
		if hookCount, ok := record["hookCount"]; ok {
			metadata["hook_count"] = minitrace.SafeInt(hookCount, 0)
		}
		hookErrors := listValue(record["hookErrors"])
		if len(hookErrors) > 0 {
			metadata["hook_errors"] = hookErrors
		}
		prevented := boolValue(record["preventedContinuation"])
		metadata["prevented_continuation"] = prevented
		setNonEmpty(metadata, "stop_reason", stringValue(record["stopReason"]))
		setNonEmpty(metadata, "tool_use_id", stringValue(record["toolUseID"]))
		event := newEvent("hook", "Stop hooks", "stop_hook_summary", metadata)
		event.Text = stringValue(record["stopReason"])
		event.ToolCallID = s.knownToolCall(stringValue(record["toolUseID"]))
		switch {
		case prevented:
			event.Severity = "error"
		case len(hookErrors) > 0:
			event.Severity = "warning"
		}
		return event, true
	}
	return minitrace.Event{}, false
}

type outcomePrefix struct {
	prefix string
	reason string
	// requiresError guards prefixes ordinary tool output could start with.
	requiresError bool
}

// claudeOutcomePrefixes classify tool results by how Claude Code phrases a
// call it did not run. Prefix matching, not substring matching: a grep over
// these very phrases must not look like a denial.
var claudeOutcomePrefixes = []outcomePrefix{
	{prefix: "The command did not run. A PreToolUse hook blocked it", reason: minitrace.ToolOutcomeReasonHookBlocked},
	{prefix: "Permission for this action was denied by the Claude Code auto mode classifier", reason: minitrace.ToolOutcomeReasonClassifierDenied, requiresError: true},
	{prefix: "Permission to use ", reason: minitrace.ToolOutcomeReasonPermissionDenied, requiresError: true},
	{prefix: "The server-side auto mode classifier gave no verdict", reason: minitrace.ToolOutcomeReasonClassifierNoVerdict, requiresError: true},
	{prefix: "The user doesn't want to proceed with this tool use", reason: minitrace.ToolOutcomeReasonUserRejected, requiresError: true},
	{prefix: "The user doesn't want to take this action", reason: minitrace.ToolOutcomeReasonUserRejected, requiresError: true},
	{prefix: "<tool_use_error>", reason: minitrace.ToolOutcomeReasonToolUseError, requiresError: true},
	{prefix: "This agent is isolated in the worktree", reason: minitrace.ToolOutcomeReasonHarnessBlocked, requiresError: true},
}

// classifyClaudeToolOutcome sets output.reason from the tool-result text. A
// hook-blocked call is a failure even though Claude Code reports it without
// is_error: the command never ran.
func classifyClaudeToolOutcome(toolCall *minitrace.ToolCall, resultText string, isError bool) {
	if toolCall.Output.Reason != "" {
		return
	}
	text := strings.TrimSpace(resultText)
	for _, candidate := range claudeOutcomePrefixes {
		if candidate.requiresError && !isError {
			continue
		}
		if strings.HasPrefix(text, candidate.prefix) {
			toolCall.Output.Reason = candidate.reason
			if !toolCall.Output.Failed() {
				toolCall.Output.SetSuccess(false)
				errText := truncateText(text, 500)
				toolCall.Output.Error = &errText
			}
			return
		}
	}
}

// applyHookBlocks marks calls named by a PreToolUse hook_blocking_error.
func (s *signalState) applyHookBlocks(toolCalls []minitrace.ToolCall) {
	for i := range toolCalls {
		if _, ok := s.hookBlocked[toolCalls[i].ID]; !ok {
			continue
		}
		toolCalls[i].Output.Reason = minitrace.ToolOutcomeReasonHookBlocked
		toolCalls[i].Output.SetSuccess(false)
	}
}

func (s *signalState) finishContextLoads(turnCount int) []minitrace.ContextLoad {
	for i := range s.contextLoads {
		if load := &s.contextLoads[i]; load.TurnIndex != nil && *load.TurnIndex >= turnCount {
			load.TurnIndex = nil
		}
	}
	return s.contextLoads
}

func setNonEmpty(metadata map[string]any, key, value string) {
	if value != "" {
		metadata[key] = value
	}
}

func nonEmpty(values ...string) []string {
	ret := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			ret = append(ret, value)
		}
	}
	return ret
}

func prefixed(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value
}
