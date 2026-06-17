package copilot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

type EventEnvelope struct {
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	ParentID  *string        `json:"parentId"`
	Ephemeral *bool          `json:"ephemeral,omitempty"`
	Raw       map[string]any `json:"-"`
}

type BadJSONLine struct {
	Line  int
	Error string
}

type ParseResult struct {
	Events   []EventEnvelope
	BadLines []BadJSONLine
}

type conversionState struct {
	sessionID    string
	sourcePath   string
	workspace    *WorkspaceMetadata
	turns        []minitrace.Turn
	toolCalls    []minitrace.ToolCall
	events       []minitrace.Event
	annotations  []minitrace.Annotation
	timestamps   []time.Time
	tokenTotals  *minitrace.TokenTotals
	metadata     map[string]any
	latestModel  string
	agentVersion string
	systemPrompt string
	containsErr  bool

	pendingTools       map[string]*pendingTool
	toolIndexByID      map[string]int
	turnIndexByTurnID  map[string]int
	toolIDsByTurnID    map[string][]string
	permissionByToolID map[string]map[string]any
}

type pendingTool struct {
	ID          string
	ToolName    string
	TurnID      string
	Timestamp   *string
	Arguments   map[string]any
	Model       string
	StartRaw    map[string]any
	Permission  map[string]any
	Description string
}

var readCommandPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^cat\s`),
	regexp.MustCompile(`^head\s`),
	regexp.MustCompile(`^tail\s`),
	regexp.MustCompile(`^less\s`),
	regexp.MustCompile(`^more\s`),
	regexp.MustCompile(`^find\s`),
	regexp.MustCompile(`^ls\s`),
	regexp.MustCompile(`^tree\s`),
	regexp.MustCompile(`^wc\s`),
	regexp.MustCompile(`^grep\s`),
	regexp.MustCompile(`^rg\s`),
	regexp.MustCompile(`^git\s+(log|show|diff|status|branch|blame)\b`),
}

var modifyCommandPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^sed\s+-i`),
	regexp.MustCompile(`^perl\s+-i`),
	regexp.MustCompile(`^patch\s`),
	regexp.MustCompile(`^git\s+apply\b`),
	regexp.MustCompile(`^chmod\s`),
	regexp.MustCompile(`^chown\s`),
	regexp.MustCompile(`>`),
}

func ConvertLocator(locator adapters.SessionLocator) (*minitrace.Session, error) {
	parsed, err := parseJSONLFile(locator.SourcePath)
	if err != nil {
		return nil, err
	}
	workspace, _ := ReadWorkspace(filepath.Join(filepath.Dir(locator.SourcePath), "workspace.yaml"))
	session, err := ConvertParsed(parsed, workspace, locator.ID, locator.SourcePath)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func ConvertRecords(records []EventEnvelope, workspace *WorkspaceMetadata, sessionID, sourcePath string) (*minitrace.Session, error) {
	return ConvertParsed(ParseResult{Events: records}, workspace, sessionID, sourcePath)
}

func ConvertRawRecords(records []map[string]any, workspace *WorkspaceMetadata, sessionID, sourcePath string) (*minitrace.Session, error) {
	events := make([]EventEnvelope, 0, len(records))
	for _, record := range records {
		dataMap := mapValue(record["data"])
		if dataMap == nil {
			dataMap = map[string]any{}
		}
		event := EventEnvelope{
			Type:      stringValue(record["type"]),
			Data:      dataMap,
			ID:        stringValue(record["id"]),
			Timestamp: stringValue(record["timestamp"]),
			ParentID:  optionalString(stringValue(record["parentId"])),
			Raw:       record,
		}
		if ephemeral, ok := record["ephemeral"].(bool); ok {
			event.Ephemeral = &ephemeral
		}
		events = append(events, event)
	}
	return ConvertParsed(ParseResult{Events: events}, workspace, sessionID, sourcePath)
}

func ConvertParsed(parsed ParseResult, workspace *WorkspaceMetadata, sessionID, sourcePath string) (*minitrace.Session, error) {
	state := newConversionState(workspace, sessionID, sourcePath)
	for _, event := range parsed.Events {
		state.observeTimestamp(event.Timestamp)
		switch event.Type {
		case "session.start":
			state.applySessionStart(event)
		case "session.info":
			state.addGenericEvent(event, "session_info", "Copilot session info", stringValue(event.Data["message"]))
		case "session.model_change":
			state.applyModelChange(event)
		case "system.message":
			state.applySystemMessage(event)
		case "user.message":
			state.addUserMessage(event)
		case "assistant.turn_start":
			state.addTurnLifecycle(event, "assistant_turn_start", "Copilot assistant turn started")
		case "assistant.message":
			state.addAssistantMessage(event)
		case "assistant.turn_end":
			state.addTurnLifecycle(event, "assistant_turn_end", "Copilot assistant turn ended")
		case "tool.execution_start":
			state.startTool(event)
		case "tool.execution_complete":
			state.completeTool(event)
		case "permission.requested":
			state.permissionRequested(event)
		case "permission.completed":
			state.permissionCompleted(event)
		case "hook.start", "hook.end":
			state.addGenericEvent(event, strings.ReplaceAll(event.Type, ".", "_"), "Copilot "+event.Type, stringValue(event.Data["hookType"]))
		case "session.shutdown":
			state.applyShutdown(event)
		default:
			state.addGenericEvent(event, "copilot_unknown", "Copilot unknown event", event.Type)
		}
	}
	state.flushIncompleteTools()
	for _, badLine := range parsed.BadLines {
		state.annotations = append(state.annotations, minitrace.BuildAnnotation(
			fmt.Sprintf("ann-copilot-bad-json-%d", badLine.Line),
			"adapter",
			"session",
			state.sessionID,
			"data-quality",
			fmt.Sprintf("Skipped malformed Copilot JSONL line %d", badLine.Line),
			badLine.Error,
			[]string{"copilot", "jsonl", "malformed"},
			nil,
		))
	}
	return state.buildSession(), nil
}

func newConversionState(workspace *WorkspaceMetadata, sessionID, sourcePath string) *conversionState {
	if strings.TrimSpace(sessionID) == "" && workspace != nil {
		sessionID = workspace.ID
	}
	if strings.TrimSpace(sessionID) == "" && sourcePath != "" {
		sessionID = filepath.Base(filepath.Dir(sourcePath))
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "copilot-session"
	}
	state := &conversionState{
		sessionID:          sessionID,
		sourcePath:         sourcePath,
		workspace:          workspace,
		turns:              []minitrace.Turn{},
		toolCalls:          []minitrace.ToolCall{},
		events:             []minitrace.Event{},
		annotations:        []minitrace.Annotation{},
		timestamps:         []time.Time{},
		tokenTotals:        &minitrace.TokenTotals{},
		metadata:           map[string]any{},
		pendingTools:       map[string]*pendingTool{},
		toolIndexByID:      map[string]int{},
		turnIndexByTurnID:  map[string]int{},
		toolIDsByTurnID:    map[string][]string{},
		permissionByToolID: map[string]map[string]any{},
	}
	if workspace != nil {
		state.metadata["workspace"] = redactRaw(workspace.Raw)
	}
	return state
}

func (s *conversionState) observeTimestamp(timestamp string) {
	if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
		s.timestamps = append(s.timestamps, parsed)
	}
}

func (s *conversionState) applySessionStart(event EventEnvelope) {
	if id := stringValue(event.Data["sessionId"]); id != "" {
		s.sessionID = id
	}
	s.agentVersion = firstNonEmpty(stringValue(event.Data["copilotVersion"]), s.agentVersion)
	if producer := stringValue(event.Data["producer"]); producer != "" {
		s.metadata["producer"] = producer
	}
	if version, ok := event.Data["version"]; ok {
		s.metadata["version"] = version
	}
	if context := mapValue(event.Data["context"]); context != nil {
		s.metadata["context"] = redactRaw(context)
	}
	s.addGenericEvent(event, "session_start", "Copilot session started", s.sessionID)
}

func (s *conversionState) applyModelChange(event EventEnvelope) {
	if model := stringValue(event.Data["newModel"]); model != "" {
		s.latestModel = model
	}
	s.addGenericEvent(event, "model_change", "Copilot model changed", s.latestModel)
}

func (s *conversionState) applySystemMessage(event EventEnvelope) {
	content := stringValue(event.Data["content"])
	if s.systemPrompt == "" {
		s.systemPrompt = content
	}
	s.addGenericEvent(event, "system_message", "Copilot system message", truncateTitle(content, 120))
}

func (s *conversionState) addUserMessage(event EventEnvelope) {
	content := firstNonEmpty(stringValue(event.Data["transformedContent"]), stringValue(event.Data["content"]))
	turn := minitrace.BuildTurn(len(s.turns), optionalString(event.Timestamp), "user", ptr("human"), content)
	turn.InputChannel = ptr("user_input")
	turn.FrameworkMetadata = map[string]any{
		"copilot_event_id":     event.ID,
		"interaction_id":       stringValue(event.Data["interactionId"]),
		"parent_agent_task_id": stringValue(event.Data["parentAgentTaskId"]),
		"attachment_count":     len(listValue(event.Data["attachments"])),
	}
	s.turns = append(s.turns, turn)
}

func (s *conversionState) addAssistantMessage(event EventEnvelope) {
	turnID := stringValue(event.Data["turnId"])
	model := stringValue(event.Data["model"])
	if model != "" {
		s.latestModel = model
	}
	turnIndex := len(s.turns)
	turn := minitrace.BuildTurn(turnIndex, optionalString(event.Timestamp), "assistant", ptr("model"), stringValue(event.Data["content"]))
	turn.Model = optionalString(model)
	turn.FrameworkMetadata = map[string]any{
		"copilot_event_id":          event.ID,
		"turn_id":                   turnID,
		"interaction_id":            stringValue(event.Data["interactionId"]),
		"message_id":                stringValue(event.Data["messageId"]),
		"phase":                     stringValue(event.Data["phase"]),
		"request_id":                stringValue(event.Data["requestId"]),
		"service_request_id":        stringValue(event.Data["serviceRequestId"]),
		"reasoning_opaque_present":  stringValue(event.Data["reasoningOpaque"]) != "",
		"encrypted_content_present": stringValue(event.Data["encryptedContent"]) != "",
	}
	outputTokens := minitrace.SafeInt(event.Data["outputTokens"], 0)
	if outputTokens > 0 {
		turn.Usage = &minitrace.Usage{OutputTokens: &outputTokens}
	}
	if turnID != "" {
		s.turnIndexByTurnID[turnID] = turnIndex
		turn.ToolCallsInTurn = append(turn.ToolCallsInTurn, s.toolIDsByTurnID[turnID]...)
		for _, toolID := range s.toolIDsByTurnID[turnID] {
			if index, ok := s.toolIndexByID[toolID]; ok {
				indexCopy := turnIndex
				s.toolCalls[index].EmittingTurnIndex = &indexCopy
			}
		}
	}
	s.turns = append(s.turns, turn)
}

func (s *conversionState) addTurnLifecycle(event EventEnvelope, kind, title string) {
	turnID := stringValue(event.Data["turnId"])
	metadata := map[string]any{"copilot_event_id": event.ID, "turn_id": turnID}
	if interactionID := stringValue(event.Data["interactionId"]); interactionID != "" {
		metadata["interaction_id"] = interactionID
	}
	ev := minitrace.BuildEvent("copilot-"+kind+"-"+stableID(event), optionalString(event.Timestamp), kind, title, turnID, metadata)
	ev.Role = "assistant"
	ev.FrameworkMetadata = metadata
	s.events = append(s.events, ev)
}

func (s *conversionState) startTool(event EventEnvelope) {
	toolCallID := stringValue(event.Data["toolCallId"])
	if toolCallID == "" {
		toolCallID = "copilot-tool-" + stableID(event)
	}
	args := mapValue(event.Data["arguments"])
	if args == nil {
		args = map[string]any{}
	}
	turnID := stringValue(event.Data["turnId"])
	pending := &pendingTool{
		ID:          toolCallID,
		ToolName:    firstNonEmpty(stringValue(event.Data["toolName"]), "unknown"),
		TurnID:      turnID,
		Timestamp:   optionalString(event.Timestamp),
		Arguments:   args,
		Model:       stringValue(event.Data["model"]),
		StartRaw:    event.Raw,
		Description: stringValue(args["description"]),
	}
	s.pendingTools[toolCallID] = pending
	if turnID != "" {
		s.toolIDsByTurnID[turnID] = appendUnique(s.toolIDsByTurnID[turnID], toolCallID)
	}
}

func (s *conversionState) completeTool(event EventEnvelope) {
	toolCallID := stringValue(event.Data["toolCallId"])
	pending := s.pendingTools[toolCallID]
	if pending == nil {
		pending = &pendingTool{
			ID:        firstNonEmpty(toolCallID, "copilot-tool-"+stableID(event)),
			ToolName:  "unknown",
			TurnID:    stringValue(event.Data["turnId"]),
			Timestamp: optionalString(event.Timestamp),
			Arguments: map[string]any{},
		}
	}
	toolCall := s.buildToolCall(pending, event, true)
	s.appendToolCall(toolCall, pending.TurnID)
	delete(s.pendingTools, toolCallID)
}

func (s *conversionState) buildToolCall(pending *pendingTool, event EventEnvelope, completed bool) minitrace.ToolCall {
	result := mapValue(event.Data["result"])
	resultText := ""
	if result != nil {
		resultText = firstNonEmpty(stringValue(result["content"]), stringValue(result["detailedContent"]))
	}
	success := true
	if completed {
		success = boolValue(event.Data["success"], true)
	}
	var errorText *string
	if completed && !success {
		errorText = optionalString(resultText)
		s.containsErr = true
	}
	command := commandFromArgs(pending.Arguments)
	metadata := map[string]any{
		"copilot_event_id":         event.ID,
		"turn_id":                  firstNonEmpty(pending.TurnID, stringValue(event.Data["turnId"])),
		"model":                    firstNonEmpty(pending.Model, stringValue(event.Data["model"])),
		"start_event":              redactRaw(pending.StartRaw),
		"completion_event_present": completed,
	}
	if telemetry := mapValue(event.Data["toolTelemetry"]); telemetry != nil {
		metadata["tool_telemetry"] = redactRaw(telemetry)
	}
	if pending.Permission != nil {
		metadata["permission"] = redactRaw(pending.Permission)
	}
	toolCall := minitrace.BuildToolCall(
		pending.ID,
		s.turnIndexForTurnID(pending.TurnID),
		firstTimestamp(pending.Timestamp, optionalString(event.Timestamp)),
		pending.ToolName,
		classifyCopilotOperation(pending.ToolName, command, pending.Arguments, pending.Permission),
		optionalString(extractFilePath(command, pending.Permission)),
		optionalString(command),
		pending.Arguments,
		success,
		resultText,
		errorText,
		nil,
		metadata,
		nil,
		ptr("tool_output"),
		nil,
	)
	if pending.Description != "" {
		toolCall.Input.Justification = &pending.Description
	} else if pending.Permission != nil {
		if intention := stringValue(pending.Permission["intention"]); intention != "" {
			toolCall.Input.Justification = &intention
		}
	}
	return toolCall
}

func (s *conversionState) appendToolCall(toolCall minitrace.ToolCall, turnID string) {
	s.toolIndexByID[toolCall.ID] = len(s.toolCalls)
	s.toolCalls = append(s.toolCalls, toolCall)
	if turnID != "" {
		s.toolIDsByTurnID[turnID] = appendUnique(s.toolIDsByTurnID[turnID], toolCall.ID)
		if turnIndex, ok := s.turnIndexByTurnID[turnID]; ok && turnIndex >= 0 && turnIndex < len(s.turns) {
			s.turns[turnIndex].ToolCallsInTurn = appendUnique(s.turns[turnIndex].ToolCallsInTurn, toolCall.ID)
			turnIndexCopy := turnIndex
			s.toolCalls[len(s.toolCalls)-1].EmittingTurnIndex = &turnIndexCopy
		}
	}
}

func (s *conversionState) permissionRequested(event EventEnvelope) {
	permissionRequest := mapValue(event.Data["permissionRequest"])
	if permissionRequest == nil {
		permissionRequest = mapValue(event.Data["promptRequest"])
	}
	requestID := stringValue(event.Data["requestId"])
	toolCallID := stringValue(permissionRequest["toolCallId"])
	metadata := cloneMap(permissionRequest)
	metadata["request_id"] = requestID
	metadata["event_id"] = event.ID
	if toolCallID != "" {
		s.permissionByToolID[toolCallID] = metadata
		if pending := s.pendingTools[toolCallID]; pending != nil {
			pending.Permission = metadata
		}
	}
	ev := minitrace.BuildEvent("copilot-permission-requested-"+firstNonEmpty(requestID, stableID(event)), optionalString(event.Timestamp), "permission_request", "Copilot permission requested", stringValue(permissionRequest["intention"]), redactRaw(event.Raw))
	ev.Role = "system"
	ev.FrameworkMetadata = redactRaw(metadata)
	if toolCallID != "" {
		ev.ToolCallID = &toolCallID
	}
	s.events = append(s.events, ev)
}

func (s *conversionState) permissionCompleted(event EventEnvelope) {
	requestID := stringValue(event.Data["requestId"])
	toolCallID := stringValue(event.Data["toolCallId"])
	result := mapValue(event.Data["result"])
	resultKind := stringValue(result["kind"])
	metadata := map[string]any{"request_id": requestID, "tool_call_id": toolCallID, "result": resultKind, "event_id": event.ID}
	if toolCallID != "" {
		if existing := s.permissionByToolID[toolCallID]; existing != nil {
			existing["result"] = resultKind
		}
		if pending := s.pendingTools[toolCallID]; pending != nil {
			if pending.Permission == nil {
				pending.Permission = map[string]any{}
			}
			pending.Permission["result"] = resultKind
		}
	}
	ev := minitrace.BuildEvent("copilot-permission-completed-"+firstNonEmpty(requestID, stableID(event)), optionalString(event.Timestamp), "permission_decision", "Copilot permission completed", resultKind, redactRaw(event.Raw))
	ev.Role = "system"
	ev.FrameworkMetadata = metadata
	if toolCallID != "" {
		ev.ToolCallID = &toolCallID
	}
	s.events = append(s.events, ev)
}

func (s *conversionState) applyShutdown(event EventEnvelope) {
	if model := stringValue(event.Data["currentModel"]); model != "" {
		s.latestModel = model
	}
	if tokenDetails := mapValue(event.Data["tokenDetails"]); tokenDetails != nil {
		s.tokenTotals.Input = tokenCount(tokenDetails, "input")
		s.tokenTotals.Output = tokenCount(tokenDetails, "output")
		s.tokenTotals.CacheRead = tokenCount(tokenDetails, "cache_read")
	}
	if event.Data["modelMetrics"] != nil {
		s.metadata["model_metrics"] = redactRaw(event.Data["modelMetrics"])
	}
	if event.Data["codeChanges"] != nil {
		s.metadata["code_changes"] = redactRaw(event.Data["codeChanges"])
	}
	if duration := minitrace.SafeInt(event.Data["totalApiDurationMs"], 0); duration > 0 {
		s.metadata["total_api_duration_ms"] = duration
	}
	s.addGenericEvent(event, "session_shutdown", "Copilot session shutdown", stringValue(event.Data["shutdownType"]))
}

func (s *conversionState) flushIncompleteTools() {
	ids := make([]string, 0, len(s.pendingTools))
	for id := range s.pendingTools {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pending := s.pendingTools[id]
		toolCall := s.buildToolCall(pending, EventEnvelope{Type: "tool.incomplete", ID: id, Data: map[string]any{}}, false)
		toolCall.Output.Success = false
		msg := "tool execution start had no matching completion"
		toolCall.Output.Error = &msg
		s.appendToolCall(toolCall, pending.TurnID)
		s.annotations = append(s.annotations, minitrace.BuildAnnotation(
			"ann-copilot-incomplete-tool-"+truncateID(id),
			"adapter",
			"tool_call",
			id,
			"data-quality",
			"Incomplete Copilot tool call",
			msg,
			[]string{"copilot", "tool-call", "incomplete"},
			nil,
		))
	}
}

func (s *conversionState) addGenericEvent(event EventEnvelope, kind, title, summary string) {
	ev := minitrace.BuildEvent("copilot-"+kind+"-"+stableID(event), optionalString(event.Timestamp), kind, title, truncateTitle(summary, 200), redactRaw(event.Raw))
	ev.Role = "system"
	ev.FrameworkMetadata = map[string]any{"copilot_event_id": event.ID, "copilot_type": event.Type}
	s.events = append(s.events, ev)
}

func (s *conversionState) buildSession() *minitrace.Session {
	session := minitrace.BuildSessionSkeleton(s.sessionID, "copilot", SourceFormatEvents, AdapterVersion)
	session.Environment.PlatformType = ptr("agent")
	session.Environment.ProviderHint = ptr("github-copilot")
	session.Environment.AgentVersion = optionalString(s.agentVersion)
	session.Environment.Model = optionalString(s.latestModel)
	session.Environment.SystemPrompt = optionalString(s.systemPrompt)
	session.Environment.ToolsEnabled = uniqueToolNames(s.toolCalls)
	session.OperationalContext.FrameworkConfig = s.metadata
	if s.workspace != nil {
		session.OperationalContext.WorkingDirectory = optionalNormalizedPath(s.workspace.CWD)
		session.OperationalContext.GitBranch = optionalString(s.workspace.Branch)
		if s.workspace.GitRoot != "" {
			session.Coordination.ProjectID = optionalString(s.workspace.Repository)
		}
	}
	if context := mapValue(s.metadata["context"]); context != nil {
		if session.OperationalContext.WorkingDirectory == nil {
			session.OperationalContext.WorkingDirectory = optionalNormalizedPath(stringValue(context["cwd"]))
		}
		if session.OperationalContext.GitBranch == nil {
			session.OperationalContext.GitBranch = optionalString(stringValue(context["branch"]))
		}
		session.OperationalContext.GitRef = optionalString(firstNonEmpty(stringValue(context["headCommit"]), stringValue(context["baseCommit"])))
	}
	if s.sourcePath != "" {
		session.Provenance.SourcePath = ptr(s.sourcePath)
	}
	minitrace.ComputeToolCallContext(s.toolCalls)
	timing := minitrace.ComputeTiming(s.timestamps)
	quality := minitrace.AssignQualityTier(s.turns, s.toolCalls)
	session.Quality = &quality
	session.Title = titleFromWorkspaceOrTurns(s.workspace, s.turns)
	session.Timing = timing
	session.Turns = s.turns
	session.ToolCalls = s.toolCalls
	session.Events = s.events
	session.Annotations = s.annotations
	session.Metrics = minitrace.ComputeMetrics(s.turns, s.toolCalls, timing, 0, s.tokenTotals)
	session.Flags.ContainsError = s.containsErr
	session.Flags.ContainsPII = minitrace.DetectPIIInPaths(s.toolCalls)
	session.Flags.ForResearch = quality == "A" && !session.Flags.ContainsPII
	session.Flags.NeedsCleaning = quality != "A" || session.Flags.ContainsPII
	if session.Flags.ContainsPII {
		session.Classification = "confidential"
	}
	return &session
}

func parseJSONLFile(path string) (ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParseResult{}, errors.Wrap(err, "reading Copilot JSONL")
	}
	return parseJSONLBytes(data), nil
}

func parseJSONLBytes(data []byte) ParseResult {
	ret := ParseResult{Events: []EventEnvelope{}, BadLines: []BadJSONLine{}}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	buffer := make([]byte, 0, 1024*1024)
	scanner.Buffer(buffer, 10*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			ret.BadLines = append(ret.BadLines, BadJSONLine{Line: lineNumber, Error: err.Error()})
			continue
		}
		dataMap := mapValue(raw["data"])
		if dataMap == nil {
			dataMap = map[string]any{}
		}
		event := EventEnvelope{
			Type:      stringValue(raw["type"]),
			Data:      dataMap,
			ID:        stringValue(raw["id"]),
			Timestamp: stringValue(raw["timestamp"]),
			ParentID:  optionalString(stringValue(raw["parentId"])),
			Raw:       raw,
		}
		if ephemeral, ok := raw["ephemeral"].(bool); ok {
			event.Ephemeral = &ephemeral
		}
		ret.Events = append(ret.Events, event)
	}
	if err := scanner.Err(); err != nil {
		ret.BadLines = append(ret.BadLines, BadJSONLine{Line: lineNumber + 1, Error: err.Error()})
	}
	return ret
}

func (s *conversionState) turnIndexForTurnID(turnID string) *int {
	if turnID == "" {
		return nil
	}
	if index, ok := s.turnIndexByTurnID[turnID]; ok {
		indexCopy := index
		return &indexCopy
	}
	return nil
}

func commandFromArgs(args map[string]any) string {
	return firstNonEmpty(stringValue(args["command"]), stringValue(args["cmd"]), stringValue(args["script"]), stringValue(args["input"]))
}

func classifyCopilotOperation(toolName, command string, args map[string]any, permission map[string]any) string {
	lowerTool := strings.ToLower(toolName)
	if boolValue(permission["hasWriteFileRedirection"], false) {
		return "MODIFY"
	}
	if strings.Contains(lowerTool, "read") || strings.Contains(lowerTool, "list") || strings.Contains(lowerTool, "search") || strings.Contains(lowerTool, "grep") {
		return "READ"
	}
	if strings.Contains(lowerTool, "write") || strings.Contains(lowerTool, "edit") || strings.Contains(lowerTool, "patch") {
		if strings.Contains(lowerTool, "create") || stringValue(args["mode"]) == "create" {
			return "CREATE"
		}
		return "MODIFY"
	}
	lowerCommand := strings.TrimSpace(strings.ToLower(command))
	for _, pattern := range readCommandPatterns {
		if pattern.MatchString(lowerCommand) {
			return "READ"
		}
	}
	for _, pattern := range modifyCommandPatterns {
		if pattern.MatchString(lowerCommand) {
			return "MODIFY"
		}
	}
	return "EXECUTE"
}

func extractFilePath(command string, permission map[string]any) string {
	if permission != nil {
		paths := listValue(permission["possiblePaths"])
		if len(paths) > 0 {
			return stringValue(paths[0])
		}
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	for i := len(fields) - 1; i >= 1; i-- {
		candidate := strings.Trim(fields[i], "'\"")
		if candidate == "" || strings.HasPrefix(candidate, "-") {
			continue
		}
		if strings.Contains(candidate, "/") || strings.Contains(candidate, ".") {
			return candidate
		}
	}
	return ""
}

func tokenCount(details map[string]any, key string) int {
	entry := mapValue(details[key])
	if entry == nil {
		return 0
	}
	return minitrace.SafeInt(entry["tokenCount"], 0)
}

func uniqueToolNames(toolCalls []minitrace.ToolCall) []string {
	seen := map[string]struct{}{}
	ret := []string{}
	for _, toolCall := range toolCalls {
		if toolCall.ToolName == "" {
			continue
		}
		if _, ok := seen[toolCall.ToolName]; ok {
			continue
		}
		seen[toolCall.ToolName] = struct{}{}
		ret = append(ret, toolCall.ToolName)
	}
	sort.Strings(ret)
	return ret
}

func titleFromWorkspaceOrTurns(workspace *WorkspaceMetadata, turns []minitrace.Turn) *string {
	if workspace != nil && strings.TrimSpace(workspace.Name) != "" && !strings.EqualFold(workspace.Name, "Session Initialization") {
		return optionalString(workspace.Name)
	}
	return minitrace.ExtractTitle(turns, 80)
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if ret, ok := value.(map[string]any); ok {
		return ret
	}
	return nil
}

func listValue(value any) []any {
	if value == nil {
		return nil
	}
	if ret, ok := value.([]any); ok {
		return ret
	}
	return nil
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func boolValue(value any, fallback bool) bool {
	if value == nil {
		return fallback
	}
	if ret, ok := value.(bool); ok {
		return ret
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func optionalNormalizedPath(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	normalized := minitrace.NormalizePath(value)
	return &normalized
}

func ptr[T any](value T) *T { return &value }

func firstTimestamp(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return value
		}
	}
	return nil
}

func stableID(event EventEnvelope) string {
	if event.ID != "" {
		return truncateID(event.ID)
	}
	return truncateID(event.Type + "-" + event.Timestamp)
}

func truncateID(id string) string {
	id = strings.ReplaceAll(id, " ", "-")
	if len(id) > 24 {
		return id[:24]
	}
	if id == "" {
		return "unknown"
	}
	return id
}

func truncateTitle(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "…"
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func cloneMap(value map[string]any) map[string]any {
	ret := map[string]any{}
	for k, v := range value {
		ret[k] = v
	}
	return ret
}

func redactRaw(value any) any {
	sensitive := map[string]bool{
		"content":          true,
		"encryptedContent": true,
		"reasoningOpaque":  true,
		"detailedContent":  true,
		"sessionLog":       true,
		"textResultForLlm": true,
	}
	switch v := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, child := range v {
			if sensitive[k] {
				out[k] = "<redacted>"
				continue
			}
			out[k] = redactRaw(child)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = redactRaw(v[i])
		}
		return out
	default:
		return v
	}
}
