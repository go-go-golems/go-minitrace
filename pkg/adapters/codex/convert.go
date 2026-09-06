package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion      = "go-minitrace-codex-adapter-dev"
	SourceFormatExec    = "codex-exec-jsonl-v1"
	SourceFormatSession = "codex-session-jsonl-v1"
	SourceFormatLegacy  = "codex-legacy-rollout-jsonl-v0"
)

var (
	readPatterns = []*regexp.Regexp{
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
		regexp.MustCompile(`^ag\s`),
		regexp.MustCompile(`^ack\s`),
		regexp.MustCompile(`^file\s`),
		regexp.MustCompile(`^stat\s`),
		regexp.MustCompile(`^du\s`),
		regexp.MustCompile(`^df\s`),
		regexp.MustCompile(`^git\s+(log|show|diff|status|branch|blame)\b`),
		regexp.MustCompile(`^python3?\s+-c\s+.*open.*read`),
	}
	modifyPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^sed\s+-i`),
		regexp.MustCompile(`^perl\s+-i`),
		regexp.MustCompile(`^patch\s`),
		regexp.MustCompile(`^git\s+apply\b`),
		regexp.MustCompile(`^chmod\s`),
		regexp.MustCompile(`^chown\s`),
	}
)

func ConvertLocator(locator adapters.SessionLocator) (*minitrace.Session, error) {
	records, err := parseJSONLFile(locator.SourcePath)
	if err != nil {
		return nil, err
	}
	var identity adapters.SourceIdentity
	if locator.Identity != nil {
		identity = *locator.Identity
	} else {
		identity, err = inspectSourceRecords(records, locator.SourcePath, locator.FormatHint)
		if err != nil {
			return nil, err
		}
	}
	session, err := ConvertRecords(records, locator.ID, locator.SourcePath, locator.FormatHint)
	if err != nil {
		return nil, err
	}
	applySourceIdentity(session, identity)
	return session, nil
}

// InspectSource returns native identity and byte-level evidence for a Codex
// source file. Conversion uses the same inspection result so discovery and
// conversion do not need incompatible identity rules.
func InspectSource(path string) (adapters.SourceIdentity, error) {
	records, err := parseJSONLFile(path)
	if err != nil {
		return adapters.SourceIdentity{}, err
	}
	return inspectSourceRecords(records, path, detectFormatRecords(records))
}

func inspectSourceRecords(records []map[string]any, sourcePath, formatHint string) (adapters.SourceIdentity, error) {
	sha256Hex, sizeBytes, normalizedPath, err := adapters.FingerprintSource(sourcePath)
	if err != nil {
		return adapters.SourceIdentity{}, err
	}
	identity := adapters.SourceIdentity{
		SourcePath:   normalizedPath,
		SourceFormat: sourceFormatName(formatHint),
		SHA256:       sha256Hex,
		SizeBytes:    sizeBytes,
		Role:         "unknown",
	}
	for recordIndex, record := range records {
		if stringValue(record["type"]) != "session_meta" {
			continue
		}
		payload := mapValue(record["payload"])
		if payload == nil {
			continue
		}
		recordID := stringValue(payload["id"])
		if identity.NativeSessionID == "" {
			identity.NativeSessionID = recordID
			if identity.NativeSessionID != "" {
				identity.IdentityBasis = "first-session-meta"
			}
		} else if recordID != "" && recordID != identity.NativeSessionID {
			identity.Warnings = append(identity.Warnings, adapters.ConversionWarning{
				Code:        "codex-replayed-session-meta",
				Message:     "later session_meta ID differs from the source header and was retained as replay metadata",
				RecordIndex: recordIndex,
			})
		}
		identity.WorkingDirectory = firstNonEmpty(identity.WorkingDirectory, stringValue(payload["cwd"]))
		identity.ParentSessionID = firstNonEmpty(identity.ParentSessionID, stringValue(payload["parent_thread_id"]))
		if source := mapValue(payload["source"]); source != nil {
			if subagent := mapValue(source["subagent"]); subagent != nil {
				identity.Role = "subagent"
				if spawn := mapValue(subagent["thread_spawn"]); spawn != nil {
					identity.ParentSessionID = firstNonEmpty(identity.ParentSessionID, stringValue(spawn["parent_thread_id"]))
				}
			}
		}
	}
	if identity.ParentSessionID != "" && identity.Role == "unknown" {
		identity.Role = "subagent"
	}
	return identity, nil
}

func applySourceIdentity(session *minitrace.Session, identity adapters.SourceIdentity) {
	if session == nil {
		return
	}
	if identity.NativeSessionID != "" {
		session.ID = identity.NativeSessionID
		session.Provenance.OriginalSessionID = ptr(identity.NativeSessionID)
	}
	if identity.SourcePath != "" {
		session.Provenance.SourcePath = ptr(identity.SourcePath)
	}
	if identity.SHA256 != "" {
		session.Provenance.SourceFingerprint = ptr(identity.SHA256)
	}
	if identity.IdentityBasis != "" {
		session.Provenance.IdentityBasis = ptr(identity.IdentityBasis)
	}
	if identity.ParentSessionID != "" {
		session.Coordination.PredecessorSession = ptr(identity.ParentSessionID)
	}
	if len(identity.Warnings) > 0 {
		config, _ := session.OperationalContext.FrameworkConfig.(map[string]any)
		if config == nil {
			config = map[string]any{}
		}
		config["conversion_warnings"] = identity.Warnings
		session.OperationalContext.FrameworkConfig = config
	}
}

func ConvertRecords(records []map[string]any, sessionID, sourcePath, formatHint string) (*minitrace.Session, error) {
	actualFormat := formatHint
	if actualFormat == "" || actualFormat == "unknown-jsonl" {
		actualFormat = detectFormatRecords(records)
	}

	var (
		turns       []minitrace.Turn
		toolCalls   []minitrace.ToolCall
		annotations []minitrace.Annotation
		timestamps  []time.Time
		tokenTotals *minitrace.TokenTotals
		metadata    codexMetadata
	)

	switch actualFormat {
	case "session-jsonl-v1":
		turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseSessionJSONL(records)
	case "exec-jsonl-v1":
		turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseExecJSONL(records)
	case "legacy-rollout-jsonl-v0":
		turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseLegacyRolloutJSONL(records)
	default:
		return nil, errors.Errorf("unsupported Codex format hint: %s", actualFormat)
	}

	if metadata.SessionID != "" {
		sessionID = metadata.SessionID
	}
	for index := range toolCalls {
		if ref := toolCalls[index].Output.FullReference; ref != nil && sourcePath != "" {
			if line, ok := strings.CutPrefix(*ref, "line:"); ok {
				toolCalls[index].Output.FullReference = ptr(sourcePath + "#L" + line)
			}
		}
	}

	session := minitrace.BuildSessionSkeleton(sessionID, "codex", sourceFormatName(actualFormat), AdapterVersion)
	session.Environment.PlatformType = ptr("agent")
	session.Environment.SystemPrompt = optionalString(metadata.SystemPrompt)
	session.Environment.Model = optionalString(metadata.Model)
	session.Environment.AgentVersion = optionalString(metadata.CLIVersion)
	session.Environment.ProviderHint = providerHint(metadata.ModelProvider)
	session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
	session.OperationalContext.WorkingDirectory = optionalNormalizedPath(metadata.CWD)
	session.OperationalContext.GitBranch = optionalString(metadata.GitBranch)
	session.OperationalContext.GitRef = optionalString(metadata.GitCommit)
	session.OperationalContext.AutonomyLevel = optionalString(mapApprovalPolicy(metadata.ApprovalPolicy))
	session.OperationalContext.Sandbox = sandboxValue(metadata.SandboxPolicy)
	session.OperationalContext.FrameworkConfig = frameworkConfig(metadata)
	if metadata.ParentThreadID != "" {
		session.Coordination.PredecessorSession = &metadata.ParentThreadID
	}
	if sourcePath != "" {
		session.Provenance.SourcePath = ptr(sourcePath)
	}

	deduped, duplicateCount := minitrace.DeduplicateToolCalls(toolCalls, nil)
	toolCalls = deduped
	if duplicateCount > 0 {
		annotations = append(annotations, minitrace.BuildAnnotation(
			"ann-dedup-"+truncateID(sessionID),
			"adapter",
			"session",
			sessionID,
			"observation",
			fmt.Sprintf("Deduplicated %d duplicate tool calls", duplicateCount),
			fmt.Sprintf("Removed %d tool calls with duplicate IDs.", duplicateCount),
			[]string{"deduplication", "data-quality"},
			nil,
		))
	}
	minitrace.ComputeToolCallContext(toolCalls)
	events, attachments := buildCodexEventsAndAttachments(toolCalls, metadata)

	timing := minitrace.ComputeTiming(timestamps)
	quality := minitrace.AssignQualityTier(turns, toolCalls)
	containsPII := minitrace.DetectPIIInPaths(toolCalls)

	session.Quality = &quality
	session.Title = minitrace.ExtractTitle(turns, 80)
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Events = events
	session.Attachments = attachments
	session.Annotations = annotations
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, countSubagents(toolCalls), tokenTotals)
	session.Flags.ContainsPII = containsPII
	session.Flags.ForResearch = quality == "A" && !containsPII
	session.Flags.NeedsCleaning = quality != "A" || containsPII
	if containsPII {
		session.Classification = "confidential"
	}
	applyCodexFidelityReport(&session, codexFidelity(records, turns, toolCalls))

	return &session, nil
}

type codexMetadata struct {
	SessionID               string
	ParentThreadID          string
	AgentNickname           string
	AgentRole               string
	Model                   string
	ModelProvider           string
	CWD                     string
	CLIVersion              string
	Originator              string
	SessionSource           string
	SystemPrompt            string
	ApprovalPolicy          string
	SandboxPolicy           string
	SandboxPolicyDetail     any
	Personality             string
	CollaborationMode       string
	CollaborationModeDetail any
	ReasoningEffort         string
	Timezone                string
	ContextWindow           int
	TruncationPolicy        any
	LatestRateLimits        any
	GitBranch               string
	GitCommit               string
	RepositoryURL           string
}

func buildCodexEventsAndAttachments(toolCalls []minitrace.ToolCall, metadata codexMetadata) ([]minitrace.Event, []minitrace.Attachment) {
	events := []minitrace.Event{}
	attachments := []minitrace.Attachment{}
	for _, toolCall := range toolCalls {
		switch toolCall.ToolName {
		case "view_image":
			attachment := buildCodexImageAttachment(toolCall)
			event := buildCodexToolEvent(toolCall, len(events), "image_view", "Codex image view", firstNonEmptyPointer(toolCall.Input.FilePath, toolCall.Output.Result))
			event.AttachmentID = &attachment.ID
			attachment.EventID = &event.ID
			attachments = append(attachments, attachment)
			events = append(events, event)
		case "spawn_agent":
			events = append(events, buildCodexToolEvent(toolCall, len(events), "subagent_spawn", "Codex subagent spawn", summarizeCodexSpawnedAgent(toolCall.SpawnedAgent)))
		case "wait_agent":
			events = append(events, buildCodexToolEvent(toolCall, len(events), "subagent_wait", "Codex subagent wait", summarizeCodexSpawnedAgent(toolCall.SpawnedAgent)))
		}
	}
	if metadata.LatestRateLimits != nil {
		event := minitrace.BuildEvent("codex-rate-limits", nil, "rate_limits", "Codex rate limits", "Latest Codex rate-limit snapshot", map[string]any{"rate_limits": metadata.LatestRateLimits})
		event.Role = "system"
		event.FrameworkMetadata = map[string]any{"rate_limits": metadata.LatestRateLimits}
		events = append(events, event)
	}
	return events, attachments
}

func buildCodexImageAttachment(toolCall minitrace.ToolCall) minitrace.Attachment {
	attachmentID := "codex-image-" + truncateID(toolCall.ID)
	path := firstNonEmptyPointer(toolCall.Input.FilePath)
	attachment := minitrace.BuildAttachment(attachmentID, toolCall.Timestamp, "image", baseName(path), mediaTypeFromPath(path), map[string]any{"tool_call_id": toolCall.ID, "tool_name": toolCall.ToolName})
	attachment.Path = path
	attachment.ToolCallID = &toolCall.ID
	attachment.FrameworkMetadata = map[string]any{"tool_call_id": toolCall.ID, "has_image_signal": true}
	return attachment
}

func buildCodexToolEvent(toolCall minitrace.ToolCall, ordinal int, kind, title, summary string) minitrace.Event {
	event := minitrace.BuildEvent(fmt.Sprintf("codex-%s-%s", strings.ReplaceAll(kind, "_", "-"), truncateID(toolCall.ID)), toolCall.Timestamp, kind, title, summary, map[string]any{"tool_call_id": toolCall.ID, "tool_name": toolCall.ToolName})
	event.Role = "tool"
	event.ToolCallID = &toolCall.ID
	if toolCall.EmittingTurnIndex != nil {
		event.TurnIndex = toolCall.EmittingTurnIndex
	}
	event.Ordinal = &ordinal
	event.FrameworkMetadata = toolCall.FrameworkMetadata
	return event
}

func summarizeCodexSpawnedAgent(agent *minitrace.SpawnedAgent) string {
	if agent == nil {
		return "Codex subagent operation."
	}
	parts := []string{}
	if agent.AgentType != "" {
		parts = append(parts, "agent_type="+agent.AgentType)
	}
	if agent.TaskScope != "" {
		parts = append(parts, "task="+truncateTitle(agent.TaskScope, 120))
	}
	if agent.SubSessionID != nil && *agent.SubSessionID != "" {
		parts = append(parts, "sub_session="+*agent.SubSessionID)
	}
	if agent.OutcomeSummary != "" {
		parts = append(parts, "outcome="+truncateTitle(agent.OutcomeSummary, 120))
	}
	if len(parts) == 0 {
		return "Codex subagent operation."
	}
	return strings.Join(parts, "; ")
}

func mediaTypeFromPath(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	default:
		return "image"
	}
}

func firstNonEmptyPointer(values ...*string) string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return *value
		}
	}
	return ""
}

func baseName(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		return "image"
	}
	if index := strings.LastIndex(path, "/"); index >= 0 && index < len(path)-1 {
		return path[index+1:]
	}
	return path
}

func parseSessionJSONL(records []map[string]any) ([]minitrace.Turn, []minitrace.ToolCall, []minitrace.Annotation, []time.Time, *minitrace.TokenTotals, codexMetadata) {
	turns := []minitrace.Turn{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	timestamps := []time.Time{}
	tokenTotals := &minitrace.TokenTotals{}
	metadata := codexMetadata{}
	messages := collectCodexMessages(records)
	pendingOutputs := map[string][]codexNativeOutput{}

	pendingFunctionCalls := map[string]int{}
	currentThinking := []string{}
	currentTurnID := ""
	toolCounter := 0

	for recordIndex, record := range records {
		recordType := stringValue(record["type"])
		timestamp := stringValue(record["timestamp"])
		if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
			timestamps = append(timestamps, parsed)
		}
		timestampPtr := optionalString(timestamp)
		payload := mapValue(record["payload"])
		if payload == nil {
			continue
		}
		if message := messages[recordIndex]; message != nil {
			turns = append(turns, message.buildTurn(len(turns), metadata.Model, currentThinking))
			if message.role == "assistant" {
				currentThinking = nil
			}
		}

		switch recordType {
		case "session_meta":
			// A Codex subagent transcript can replay its parent's session_meta
			// record after its own header. The first native session header owns
			// this source file's identity; later metadata must not turn a child
			// archive into its parent and overwrite another archive on disk.
			if metadata.SessionID == "" {
				metadata.SessionID = stringValue(payload["id"])
			}
			metadata.ParentThreadID = firstNonEmpty(stringValue(payload["parent_thread_id"]), metadata.ParentThreadID)
			metadata.AgentNickname = firstNonEmpty(stringValue(payload["agent_nickname"]), metadata.AgentNickname)
			metadata.AgentRole = firstNonEmpty(stringValue(payload["agent_role"]), metadata.AgentRole)
			metadata.CWD = firstNonEmpty(stringValue(payload["cwd"]), metadata.CWD)
			metadata.CLIVersion = firstNonEmpty(stringValue(payload["cli_version"]), metadata.CLIVersion)
			metadata.Originator = firstNonEmpty(stringValue(payload["originator"]), metadata.Originator)
			metadata.SessionSource = firstNonEmpty(stringValue(payload["source"]), metadata.SessionSource)
			metadata.ModelProvider = firstNonEmpty(stringValue(payload["model_provider"]), metadata.ModelProvider)
			baseInstructions := mapValue(payload["base_instructions"])
			if baseInstructions != nil {
				metadata.SystemPrompt = firstNonEmpty(stringValue(baseInstructions["text"]), metadata.SystemPrompt)
			}
		case "turn_context":
			metadata.CWD = firstNonEmpty(stringValue(payload["cwd"]), metadata.CWD)
			metadata.Model = firstNonEmpty(stringValue(payload["model"]), metadata.Model)
			metadata.ApprovalPolicy = firstNonEmpty(stringValue(payload["approval_policy"]), metadata.ApprovalPolicy)
			metadata.SandboxPolicy = firstNonEmpty(extractSandboxPolicy(payload["sandbox_policy"]), metadata.SandboxPolicy)
			if payload["sandbox_policy"] != nil {
				metadata.SandboxPolicyDetail = payload["sandbox_policy"]
			}
			metadata.Personality = firstNonEmpty(stringValue(payload["personality"]), metadata.Personality)
			metadata.Timezone = firstNonEmpty(stringValue(payload["timezone"]), metadata.Timezone)
			metadata.ReasoningEffort = firstNonEmpty(stringValue(payload["effort"]), metadata.ReasoningEffort)
			currentTurnID = firstNonEmpty(stringValue(payload["turn_id"]), currentTurnID)
			if collaboration := mapValue(payload["collaboration_mode"]); collaboration != nil {
				metadata.CollaborationMode = firstNonEmpty(stringValue(collaboration["mode"]), metadata.CollaborationMode)
				metadata.CollaborationModeDetail = payload["collaboration_mode"]
				settings := mapValue(collaboration["settings"])
				if settings != nil {
					metadata.ReasoningEffort = firstNonEmpty(stringValue(settings["reasoning_effort"]), metadata.ReasoningEffort)
				}
			}
			if payload["truncation_policy"] != nil {
				metadata.TruncationPolicy = payload["truncation_policy"]
			}
		case "event_msg":
			eventType := stringValue(payload["type"])
			switch eventType {
			case "task_started":
				metadata.ContextWindow = firstNonZero(minitrace.SafeInt(payload["model_context_window"], 0), metadata.ContextWindow)
				currentTurnID = firstNonEmpty(stringValue(payload["turn_id"]), currentTurnID)
			case "agent_reasoning":
				if text := stringValue(payload["text"]); text != "" {
					currentThinking = append(currentThinking, text)
				}
			case "token_count":
				if payload["rate_limits"] != nil {
					metadata.LatestRateLimits = payload["rate_limits"]
				}
				info := mapValue(payload["info"])
				metadata.ContextWindow = firstNonZero(minitrace.SafeInt(info["model_context_window"], 0), metadata.ContextWindow)
				lastUsage := mapValue(info["last_token_usage"])
				if lastUsage == nil {
					continue
				}
				inputTokens := minitrace.SafeInt(lastUsage["input_tokens"], 0)
				outputTokens := minitrace.SafeInt(lastUsage["output_tokens"], 0)
				cacheReadTokens := minitrace.SafeInt(lastUsage["cached_input_tokens"], 0)
				reasoningTokens := minitrace.SafeInt(lastUsage["reasoning_output_tokens"], 0)
				tokenTotals.Input += inputTokens
				tokenTotals.Output += outputTokens
				tokenTotals.CacheRead += cacheReadTokens
				tokenTotals.Reasoning += reasoningTokens
				if len(turns) > 0 && turns[len(turns)-1].Role == "assistant" {
					usage := &minitrace.Usage{}
					if inputTokens != 0 {
						usage.InputTokens = &inputTokens
					}
					if outputTokens != 0 {
						usage.OutputTokens = &outputTokens
					}
					if cacheReadTokens != 0 {
						usage.CacheReadTokens = &cacheReadTokens
					}
					if reasoningTokens != 0 {
						usage.ReasoningTokens = &reasoningTokens
					}
					turns[len(turns)-1].Usage = usage
				}
			case "exec_command_end":
				callID := stringValue(payload["call_id"])
				if callID == "" {
					continue
				}
				if turnID := stringValue(payload["turn_id"]); turnID != "" {
					currentTurnID = turnID
				}
				// Reconcile terminal evidence after all response calls/results exist.
				// Native end notifications may arrive before the invocation.
			}
		case "response_item":
			payloadType := stringValue(payload["type"])
			switch payloadType {
			case "reasoning":
				for _, item := range listValue(payload["summary"]) {
					summary := mapValue(item)
					if summary == nil {
						continue
					}
					if text := stringValue(summary["text"]); text != "" {
						currentThinking = append(currentThinking, text)
					}
				}
			case "function_call", "custom_tool_call":
				callID := stringValue(payload["call_id"])
				if callID == "" {
					callID = fmt.Sprintf("tc-codex-%04d", toolCounter)
				}
				toolCounter++
				toolCall := buildCodexResponseToolCall(callID, timestampPtr, codexNativeTurnID(payload, currentTurnID), payload)
				toolCall.FrameworkMetadata = mergeMetadataMap(toolCall.FrameworkMetadata, codexCallSourceMetadata(payload, recordIndex))
				toolCalls = append(toolCalls, toolCall)
				pendingFunctionCalls[callID] = len(toolCalls) - 1
				if outputs, ok := pendingOutputs[callID]; ok {
					for _, output := range outputs {
						output.apply(&toolCalls[len(toolCalls)-1])
					}
					delete(pendingOutputs, callID)
				}
			case "function_call_output", "custom_tool_call_output":
				callID := stringValue(payload["call_id"])
				output := codexNativeOutput{value: payload["output"], line: recordIndex + 1}
				if index, ok := pendingFunctionCalls[callID]; ok {
					output.apply(&toolCalls[index])
				} else {
					pendingOutputs[callID] = append(pendingOutputs[callID], output)
				}
			}
		}
	}

	flushCodexThinkingToLastAssistant(turns, currentThinking)
	reconcileCodexLegacyEnds(records, toolCalls)
	toolCalls = appendCodexExecutions(records, toolCalls)
	linkCodexMessageCalls(turns, toolCalls)

	return turns, toolCalls, annotations, timestamps, tokenTotals, metadata
}

func parseLegacyRolloutJSONL(records []map[string]any) ([]minitrace.Turn, []minitrace.ToolCall, []minitrace.Annotation, []time.Time, *minitrace.TokenTotals, codexMetadata) {
	turns := []minitrace.Turn{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	timestamps := []time.Time{}
	tokenTotals := &minitrace.TokenTotals{}
	metadata := codexMetadata{}
	pendingOutputs := map[string][]codexNativeOutput{}

	pendingFunctionCalls := map[string]int{}
	currentThinking := []string{}
	turnIndex := 0
	toolCounter := 0

	for recordIndex, record := range records {
		timestamp := stringValue(record["timestamp"])
		if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
			timestamps = append(timestamps, parsed)
		}
		timestampPtr := optionalString(timestamp)

		recordType := firstNonEmpty(stringValue(record["type"]), stringValue(record["record_type"]))
		if recordType == "" && record["id"] != nil {
			metadata.SessionID = firstNonEmpty(stringValue(record["id"]), metadata.SessionID)
			metadata.SystemPrompt = firstNonEmpty(stringValue(record["instructions"]), metadata.SystemPrompt)
			if git := mapValue(record["git"]); git != nil {
				metadata.GitBranch = firstNonEmpty(stringValue(git["branch"]), metadata.GitBranch)
				metadata.GitCommit = firstNonEmpty(stringValue(git["commit_hash"]), metadata.GitCommit)
				metadata.RepositoryURL = firstNonEmpty(stringValue(git["repository_url"]), metadata.RepositoryURL)
			}
			continue
		}

		switch recordType {
		case "state":
			continue
		case "reasoning":
			for _, item := range listValue(record["summary"]) {
				summary := mapValue(item)
				if summary == nil {
					continue
				}
				if text := stringValue(summary["text"]); text != "" {
					currentThinking = append(currentThinking, text)
				}
			}
		case "message":
			role := firstNonEmpty(stringValue(record["role"]), "assistant")
			source := ptr("model")
			switch role {
			case "user":
				source = ptr("human")
			case "system":
				source = ptr("system")
			}
			turn := minitrace.BuildTurn(turnIndex, timestampPtr, role, source, flattenCodexLegacyContent(record["content"]))
			thinkingMetadata := map[string]any(nil)
			if role == "assistant" {
				thinkingMetadata = attachCodexThinking(&turn, currentThinking)
				currentThinking = nil
			}
			turn.FrameworkMetadata = codexTurnMetadata("", record, thinkingMetadata)
			turns = append(turns, turn)
			turnIndex++
		case "function_call":
			callID := stringValue(record["call_id"])
			if callID == "" {
				callID = firstNonEmpty(stringValue(record["id"]), fmt.Sprintf("tc-codex-legacy-%04d", toolCounter))
			}
			toolCounter++
			toolCall := buildCodexResponseToolCall(callID, timestampPtr, "", normalizeLegacyCodexFunctionCall(record))
			toolCalls = append(toolCalls, toolCall)
			pendingFunctionCalls[callID] = len(toolCalls) - 1
			if outputs, ok := pendingOutputs[callID]; ok {
				for _, output := range outputs {
					output.apply(&toolCalls[len(toolCalls)-1])
				}
				delete(pendingOutputs, callID)
			}
		case "function_call_output":
			callID := stringValue(record["call_id"])
			output := codexNativeOutput{value: record["output"], line: recordIndex + 1}
			if index, ok := pendingFunctionCalls[callID]; ok {
				output.apply(&toolCalls[index])
			} else {
				pendingOutputs[callID] = append(pendingOutputs[callID], output)
			}
		}
	}

	flushCodexThinkingToLastAssistant(turns, currentThinking)
	for i := range toolCalls {
		toolCalls[i].FrameworkMetadata = mergeMetadataMap(toolCalls[i].FrameworkMetadata, map[string]any{"turn_association": "unknown"})
	}

	return turns, toolCalls, annotations, timestamps, tokenTotals, metadata
}

func parseExecJSONL(records []map[string]any) ([]minitrace.Turn, []minitrace.ToolCall, []minitrace.Annotation, []time.Time, *minitrace.TokenTotals, codexMetadata) {
	turns := []minitrace.Turn{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	timestamps := []time.Time{}
	tokenTotals := &minitrace.TokenTotals{}
	metadata := codexMetadata{}

	currentThinking := []string{}
	turnIndex := 0

	for _, record := range records {
		recordType := stringValue(record["type"])
		timestamp := stringValue(record["timestamp"])
		if parsed, ok := minitrace.ParseTimestamp(timestamp); ok {
			timestamps = append(timestamps, parsed)
		}
		timestampPtr := optionalString(timestamp)

		switch recordType {
		case "thread.started":
			metadata.SessionID = stringValue(record["thread_id"])
		case "item.completed":
			item := mapValue(record["item"])
			if item == nil {
				continue
			}
			itemType := stringValue(item["type"])
			itemID := firstNonEmpty(stringValue(item["id"]), fmt.Sprintf("item-%d", len(toolCalls)))
			switch itemType {
			case "error":
				message := stringValue(item["message"])
				annotations = append(annotations, minitrace.BuildAnnotation(
					"ann-error-"+truncateID(itemID),
					"adapter",
					"session",
					firstNonEmpty(metadata.SessionID, "unknown"),
					"observation",
					"Codex error: "+truncateTitle(message, 60),
					message,
					[]string{"codex-error"},
					nil,
				))
			case "reasoning":
				if text := stringValue(item["text"]); text != "" {
					currentThinking = append(currentThinking, text)
				}
			case "command_execution":
				command := stringValue(item["command"])
				output := stringValue(item["aggregated_output"])
				var success bool
				var exitCode *int
				if parsedExitCode, valid := codexInteger(item["exit_code"]); valid {
					exitCode = &parsedExitCode
					success = parsedExitCode == 0
				}
				// Copy the loop variable so each tool call keeps the turn index
				// it was emitted at instead of aliasing one shared int.
				turnIndexCopy := turnIndex
				toolCall := minitrace.BuildToolCall(
					itemID,
					&turnIndexCopy,
					timestampPtr,
					"exec_command",
					classifyOperationFromCommand(command),
					optionalString(extractFilePathFromCommand(command)),
					optionalString(command),
					map[string]any{"cmd": command},
					success,
					output,
					nil,
					nil,
					map[string]any{
						"codex_function": "exec_command",
						"exit_code":      item["exit_code"],
						"status":         item["status"],
					},
					nil,
					classifyContentOrigin("exec_command"),
					nil,
				)
				toolCall.Output.ExitCode = exitCode
				if exitCode == nil {
					toolCall.Output.Success = nil
					toolCall.Output.Status = minitrace.ToolOutcomeUnknown
					if status := stringValue(item["status"]); status == "cancelled" || status == "canceled" {
						toolCall.Output.Status = minitrace.ToolOutcomeCancelled
					}
				}
				toolCall.FrameworkMetadata = mergeMetadataMap(toolCall.FrameworkMetadata, codexToolExecutionMetadata(item))
				toolCalls = append(toolCalls, toolCall)
			case "agent_message":
				source := ptr("model")
				turn := minitrace.BuildTurn(turnIndex, timestampPtr, "assistant", source, stringValue(item["text"]))
				thinkingMetadata := attachCodexThinking(&turn, currentThinking)
				turn.FrameworkMetadata = codexTurnMetadata(stringValue(item["turn_id"]), item, thinkingMetadata)
				toolIDs := []string{}
				for _, toolCall := range toolCalls {
					if toolCall.EmittingTurnIndex != nil && *toolCall.EmittingTurnIndex == turnIndex {
						toolIDs = append(toolIDs, toolCall.ID)
					}
				}
				turn.ToolCallsInTurn = toolIDs
				turns = append(turns, turn)
				turnIndex++
				currentThinking = nil
			}
		}
	}

	flushCodexThinkingToLastAssistant(turns, currentThinking)

	return turns, toolCalls, annotations, timestamps, tokenTotals, metadata
}

func parseJSONLFile(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening Codex JSONL")
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 1024*1024)
	scanner.Buffer(buffer, 10*1024*1024)

	records := []map[string]any{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanning Codex JSONL")
	}
	return records, nil
}

func detectFormatRecords(records []map[string]any) string {
	for i, record := range records {
		if i >= 5 {
			break
		}
		switch stringValue(record["type"]) {
		case "session_meta", "response_item", "event_msg", "turn_context":
			return "session-jsonl-v1"
		case "thread.started", "turn.started", "turn.completed", "item.started", "item.completed":
			return "exec-jsonl-v1"
		case "message", "reasoning", "function_call", "function_call_output":
			return "legacy-rollout-jsonl-v0"
		}
		if stringValue(record["record_type"]) == "state" {
			return "legacy-rollout-jsonl-v0"
		}
		if record["id"] != nil && record["timestamp"] != nil && record["type"] == nil {
			return "legacy-rollout-jsonl-v0"
		}
	}
	return "unknown-jsonl"
}

func sourceFormatName(formatHint string) string {
	switch formatHint {
	case "session-jsonl-v1":
		return SourceFormatSession
	case "exec-jsonl-v1":
		return SourceFormatExec
	case "legacy-rollout-jsonl-v0":
		return SourceFormatLegacy
	default:
		return formatHint
	}
}

func parseArguments(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		return value
	case string:
		if value == "" {
			return map[string]any{}
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(value), &parsed); err == nil && parsed != nil {
			return parsed
		}
		return map[string]any{"raw": value}
	default:
		return map[string]any{}
	}
}

func normalizeLegacyCodexFunctionCall(record map[string]any) map[string]any {
	payload := map[string]any{}
	for key, value := range record {
		payload[key] = value
	}
	if stringValue(payload["name"]) == "shell" {
		payload["name"] = "exec_command"
		args := parseArguments(payload["arguments"])
		if commandList := listValue(args["command"]); len(commandList) > 0 {
			parts := make([]string, 0, len(commandList))
			for _, part := range commandList {
				parts = append(parts, stringValue(part))
			}
			args["cmd"] = strings.Join(parts, " ")
		} else if command := stringValue(args["command"]); command != "" {
			args["cmd"] = command
		}
		payload["arguments"] = args
	}
	return payload
}

func flattenCodexLegacyContent(value any) string {
	parts := []string{}
	for _, item := range listValue(value) {
		block := mapValue(item)
		if block == nil {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				parts = append(parts, text)
			}
			continue
		}
		if text := stringValue(block["text"]); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return stringValue(value)
	}
	return strings.Join(parts, "\n")
}

func buildCodexResponseToolCall(callID string, timestamp *string, currentTurnID string, payload map[string]any) minitrace.ToolCall {
	funcName := firstNonEmpty(stringValue(payload["name"]), "unknown")
	args := parseArguments(payload["arguments"])
	if stringValue(payload["type"]) == "custom_tool_call" {
		args = map[string]any{"input": stringValue(payload["input"])}
	}
	command := commandForFunction(funcName, args)
	filePath := filePathForFunction(funcName, args, command)
	metadata := codexFrameworkMetadata(funcName, args)
	metadata["record_kind"] = "tool_call"
	if funcName == "exec" {
		metadata["record_kind"] = "orchestration"
	}
	if namespace := stringValue(payload["namespace"]); namespace != "" {
		metadata["namespace"] = namespace
	}
	if status, ok := payload["status"]; ok {
		metadata["status"] = status
	}
	if stringValue(payload["type"]) == "custom_tool_call" {
		metadata["custom_tool"] = true
	}
	if funcName == "view_image" {
		metadata["has_image_signal"] = true
		if detail := stringValue(args["detail"]); detail != "" {
			metadata["image_detail"] = detail
		}
	}
	typeOrigin := classifyContentOrigin(funcName)
	toolCall := minitrace.BuildToolCall(
		callID,
		nil,
		timestamp,
		funcName,
		classifyFunction(funcName, args),
		optionalNormalizedPath(filePath),
		optionalString(command),
		args,
		true,
		nil,
		nil,
		nil,
		metadata,
		buildCodexSpawnedAgent(funcName, args),
		typeOrigin,
		nil,
	)
	if justification := stringValue(args["justification"]); justification != "" {
		toolCall.Input.Justification = &justification
	}
	toolCall.FrameworkMetadata = mergeMetadataMap(toolCall.FrameworkMetadata, codexTurnMetadata(currentTurnID, nil, nil))
	toolCall.Output.Success = nil
	toolCall.Output.Status = minitrace.ToolOutcomePending
	return toolCall
}

func commandForFunction(functionName string, args map[string]any) string {
	switch functionName {
	case "exec_command":
		return stringValue(args["cmd"])
	case "write_stdin":
		return "write_stdin"
	case "apply_patch":
		return "apply_patch"
	default:
		return ""
	}
}

func filePathForFunction(functionName string, args map[string]any, command string) string {
	switch functionName {
	case "view_image":
		return firstNonEmpty(stringValue(args["path"]), stringValue(args["file_path"]), stringValue(args["image_path"]))
	case "read_file", "write_file", "edit_file", "apply_patch", "apply_diff":
		return firstNonEmpty(stringValue(args["path"]), stringValue(args["file_path"]), extractFilePathFromPatch(stringValue(args["input"])))
	case "exec_command":
		return extractFilePathFromCommand(command)
	default:
		return ""
	}
}

func buildCodexSpawnedAgent(functionName string, args map[string]any) *minitrace.SpawnedAgent {
	if functionName != "spawn_agent" && functionName != "wait_agent" {
		return nil
	}
	agentType := firstNonEmpty(stringValue(args["agent_type"]), stringValue(args["type"]), "codex")
	taskScope := firstNonEmpty(stringValue(args["message"]), stringValue(args["task"]), stringValue(args["prompt"]), firstStringFromAnyList(args["targets"]))
	return &minitrace.SpawnedAgent{AgentType: agentType, TaskScope: taskScope}
}

func promoteCodexOutputMetadata(toolCall *minitrace.ToolCall, output string) {
	if toolCall == nil {
		return
	}
	switch toolCall.ToolName {
	case "spawn_agent":
		if toolCall.SpawnedAgent == nil {
			return
		}
		if id := extractCodexSessionID(output); id != "" {
			toolCall.SpawnedAgent.SubSessionID = &id
		}
		if strings.TrimSpace(output) != "" && toolCall.SpawnedAgent.OutcomeSummary == "" {
			toolCall.SpawnedAgent.OutcomeSummary = truncateTitle(strings.TrimSpace(output), 240)
		}
	case "wait_agent":
		metadata := map[string]any{}
		args := mapValue(toolCall.Input.Arguments)
		if args != nil && args["targets"] != nil {
			metadata["targets"] = args["targets"]
		}
		if summary, timedOut := summarizeWaitAgentOutput(output); summary != "" || timedOut != nil {
			if summary != "" && toolCall.SpawnedAgent != nil {
				toolCall.SpawnedAgent.OutcomeSummary = summary
			}
			if timedOut != nil {
				metadata["timed_out"] = *timedOut
			}
		}
		toolCall.FrameworkMetadata = mergeMetadataMap(toolCall.FrameworkMetadata, metadata)
	}
}

func summarizeWaitAgentOutput(output string) (string, *bool) {
	var parsed map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed) != nil {
		return truncateTitle(strings.TrimSpace(output), 240), nil
	}
	var timedOut *bool
	if value, ok := parsed["timed_out"].(bool); ok {
		timedOut = &value
	}
	status := mapValue(parsed["status"])
	if status == nil {
		return "", timedOut
	}
	for _, raw := range status {
		entry := mapValue(raw)
		if entry == nil {
			continue
		}
		if completed := stringValue(entry["completed"]); completed != "" {
			return truncateTitle(completed, 240), timedOut
		}
		if failed := stringValue(entry["failed"]); failed != "" {
			return truncateTitle(failed, 240), timedOut
		}
	}
	return "", timedOut
}

func extractCodexSessionID(output string) string {
	output = strings.TrimSpace(output)
	if regexp.MustCompile(`^[0-9a-f]{8,}-[0-9a-f-]{8,}$`).MatchString(output) {
		return output
	}
	return ""
}

func firstStringFromAnyList(value any) string {
	items := listValue(value)
	for _, item := range items {
		if s := stringValue(item); s != "" {
			return s
		}
	}
	return ""
}

func extractFilePathFromPatch(patch string) string {
	for _, line := range strings.Split(patch, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"*** Update File:", "*** Add File:", "*** Delete File:"} {
			if strings.HasPrefix(line, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	return ""
}

func codexFrameworkMetadata(functionName string, args map[string]any) map[string]any {
	metadata := map[string]any{
		"codex_function": functionName,
	}
	if justification := stringValue(args["justification"]); justification != "" {
		metadata["justification"] = justification
	}
	return metadata
}

func classifyOperationFromCommand(command string) string {
	if command == "" {
		return "EXECUTE"
	}
	commandLower := strings.ToLower(strings.TrimSpace(command))
	if len(commandLower) > 1024 {
		commandLower = commandLower[:1024]
	}
	switch {
	case strings.Contains(command, ">>"):
		return "MODIFY"
	case strings.Contains(command, ">"):
		return "NEW"
	case regexp.MustCompile(`^tee\s`).MatchString(commandLower):
		return "NEW"
	case regexp.MustCompile(`^(touch|mkdir|cp)\s`).MatchString(commandLower):
		return "NEW"
	}
	for _, pattern := range modifyPatterns {
		if pattern.MatchString(commandLower) {
			return "MODIFY"
		}
	}
	for _, pattern := range readPatterns {
		if pattern.MatchString(commandLower) {
			return "READ"
		}
	}
	return "EXECUTE"
}

func classifyFunction(functionName string, args map[string]any) string {
	if functionName == "exec_command" {
		return classifyOperationFromCommand(stringValue(args["cmd"]))
	}
	switch functionName {
	case "read_file", "view_image":
		return "READ"
	case "write_file":
		return "NEW"
	case "edit_file", "apply_patch", "apply_diff":
		return "MODIFY"
	case "write_stdin":
		return "EXECUTE"
	case "spawn_agent", "wait_agent":
		return "DELEGATE"
	default:
		return "OTHER"
	}
}

func extractFilePathFromCommand(command string) string {
	if command == "" {
		return ""
	}
	command = strings.TrimSpace(command)
	if match := regexp.MustCompile(`>{1,2}\s*(\S+)\s*$`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`\|\s*tee\s+(?:-a\s+)?(\S+)`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:cat|head|tail|less|more|wc|file|stat)\s+(?:-\S+\s+(?:\d+\s+)*)*(\S+)`).FindStringSubmatch(command); len(match) == 2 && !strings.HasPrefix(match[1], "-") {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:touch|mkdir)\s+(?:-\S+\s+)*(\S+)`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:cp|mv)\s+(?:-\S+\s+)*(\S+)\s+(\S+)`).FindStringSubmatch(command); len(match) == 3 {
		return match[2]
	}
	if match := regexp.MustCompile(`^sed\s+-i[^\s]*\s+.*\s+(\S+)\s*$`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`^(?:chmod|chown)\s+(?:-\S+\s+)*\S+\s+(\S+)`).FindStringSubmatch(command); len(match) == 2 {
		return match[1]
	}
	return ""
}

func classifyContentOrigin(functionName string) *string {
	switch functionName {
	case "exec_command", "write_stdin":
		return ptr("local_exec")
	case "read_file":
		return ptr("local_file")
	case "view_image":
		return ptr("image")
	case "write_file", "edit_file", "apply_patch", "apply_diff":
		return ptr("model_echo")
	default:
		return nil
	}
}

func providerHint(modelProvider string) *string {
	switch modelProvider {
	case "ollama", "openai", "":
		return ptr("openai-compatible")
	default:
		return ptr(modelProvider)
	}
}

func mapApprovalPolicy(policy string) string {
	switch policy {
	case "never":
		return "full-auto"
	case "always":
		return "suggest"
	default:
		return policy
	}
}

func sandboxValue(policy string) *bool {
	switch {
	case policy == "danger-full-access":
		return ptr(false)
	case strings.Contains(strings.ToLower(policy), "sandbox"), strings.Contains(strings.ToLower(policy), "read-only"), strings.Contains(strings.ToLower(policy), "workspace-write"):
		return ptr(true)
	default:
		return nil
	}
}

func frameworkConfig(metadata codexMetadata) any {
	config := map[string]any{}
	if metadata.ParentThreadID != "" {
		config["parent_thread_id"] = metadata.ParentThreadID
	}
	if metadata.AgentNickname != "" {
		config["agent_nickname"] = metadata.AgentNickname
	}
	if metadata.AgentRole != "" {
		config["agent_role"] = metadata.AgentRole
	}
	if metadata.Personality != "" {
		config["personality"] = metadata.Personality
	}
	if metadata.CollaborationMode != "" {
		config["collaboration_mode"] = metadata.CollaborationMode
	}
	if metadata.CollaborationModeDetail != nil {
		config["collaboration_mode_detail"] = metadata.CollaborationModeDetail
	}
	if metadata.ReasoningEffort != "" {
		config["reasoning_effort"] = metadata.ReasoningEffort
	}
	if metadata.Originator != "" {
		config["originator"] = metadata.Originator
	}
	if metadata.SessionSource != "" {
		config["session_source"] = metadata.SessionSource
	}
	if metadata.ApprovalPolicy != "" {
		config["approval_policy"] = metadata.ApprovalPolicy
	}
	if metadata.SandboxPolicyDetail != nil {
		config["sandbox_policy"] = metadata.SandboxPolicyDetail
	}
	if metadata.TruncationPolicy != nil {
		config["truncation_policy"] = metadata.TruncationPolicy
	}
	if metadata.LatestRateLimits != nil {
		config["rate_limits"] = metadata.LatestRateLimits
	}
	if metadata.GitBranch != "" || metadata.GitCommit != "" || metadata.RepositoryURL != "" {
		config["git"] = map[string]any{
			"branch":         metadata.GitBranch,
			"commit_hash":    metadata.GitCommit,
			"repository_url": metadata.RepositoryURL,
		}
	}
	if metadata.ContextWindow != 0 {
		config["model_context_window"] = metadata.ContextWindow
	}
	if metadata.Timezone != "" {
		config["timezone"] = metadata.Timezone
	}
	if len(config) == 0 {
		return nil
	}
	return config
}

func attachCodexThinking(turn *minitrace.Turn, thinkingBlocks []string) map[string]any {
	return appendCodexThinking(turn, thinkingBlocks, nil)
}

func flushCodexThinkingToLastAssistant(turns []minitrace.Turn, thinkingBlocks []string) {
	if len(thinkingBlocks) == 0 {
		return
	}
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Role != "assistant" {
			continue
		}
		appendCodexThinking(&turns[i], thinkingBlocks, map[string]any{
			"reasoning_flushed_without_following_message": true,
		})
		return
	}
}

func appendCodexThinking(turn *minitrace.Turn, thinkingBlocks []string, extra map[string]any) map[string]any {
	if len(thinkingBlocks) == 0 {
		return nil
	}
	thinking := strings.Join(thinkingBlocks, "\n")
	if turn.Thinking != nil && *turn.Thinking != "" {
		thinking = *turn.Thinking + "\n" + thinking
	}
	turn.Thinking = &thinking

	metadata := map[string]any{}
	if current, ok := turn.FrameworkMetadata.(map[string]any); ok {
		for key, value := range current {
			metadata[key] = value
		}
	}
	metadata["reasoning_block_count"] = metadataInt(metadata["reasoning_block_count"]) + len(thinkingBlocks)
	for key, value := range extra {
		metadata[key] = value
	}
	turn.FrameworkMetadata = metadata
	return map[string]any{
		"reasoning_block_count": len(thinkingBlocks),
	}
}

func metadataInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func codexTurnMetadata(turnID string, payload map[string]any, extra map[string]any) map[string]any {
	metadata := map[string]any{}
	if turnID != "" {
		metadata["turn_id"] = turnID
	}
	if payload != nil {
		if phase, ok := payload["phase"]; ok {
			metadata["phase"] = phase
		}
		if memoryCitation, ok := payload["memory_citation"]; ok {
			metadata["memory_citation"] = memoryCitation
		}
	}
	for key, value := range extra {
		metadata[key] = value
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func codexToolExecutionMetadata(payload map[string]any) map[string]any {
	metadata := map[string]any{}
	if source := stringValue(payload["source"]); source != "" {
		metadata["source"] = source
	}
	if parsedCmd := payload["parsed_cmd"]; parsedCmd != nil {
		metadata["parsed_cmd"] = parsedCmd
	}
	if stdout, ok := payload["stdout"]; ok {
		metadata["stdout"] = stdout
	}
	if stderr, ok := payload["stderr"]; ok {
		metadata["stderr"] = stderr
	}
	if status, ok := payload["status"]; ok {
		metadata["status"] = status
	}
	if turnID := stringValue(payload["turn_id"]); turnID != "" {
		metadata["turn_id"] = turnID
	}
	return metadata
}

func mergeMetadataMap(existing any, fields map[string]any) any {
	if len(fields) == 0 {
		return existing
	}
	metadata := map[string]any{}
	if current, ok := existing.(map[string]any); ok {
		for key, value := range current {
			metadata[key] = value
		}
	}
	for key, value := range fields {
		metadata[key] = value
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func extractSandboxPolicy(value any) string {
	if value == nil {
		return ""
	}
	if raw, ok := value.(string); ok {
		return raw
	}
	sandbox := mapValue(value)
	if sandbox == nil {
		return ""
	}
	return firstNonEmpty(stringValue(sandbox["type"]), stringValue(sandbox["mode"]))
}

// countSubagents mirrors the claude-code adapter's subagent counting: each
// spawn_agent tool call that produced a SpawnedAgent counts as one subagent.
// wait_agent calls also carry a SpawnedAgent but do not spawn anything new.
func countSubagents(toolCalls []minitrace.ToolCall) int {
	count := 0
	for _, toolCall := range toolCalls {
		if toolCall.ToolName == "spawn_agent" && toolCall.SpawnedAgent != nil {
			count++
		}
	}
	return count
}

func uniqueToolNames(toolCalls []minitrace.ToolCall) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		if toolCall.ToolName == "" {
			continue
		}
		if _, ok := seen[toolCall.ToolName]; ok {
			continue
		}
		seen[toolCall.ToolName] = struct{}{}
		names = append(names, toolCall.ToolName)
	}
	sort.Strings(names)
	return names
}

func optionalNormalizedPath(path string) *string {
	if path == "" {
		return nil
	}
	normalized := minitrace.NormalizePath(path)
	return &normalized
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func truncateID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func truncateTitle(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if v, ok := value.(map[string]any); ok {
		return v
	}
	return nil
}

func listValue(value any) []any {
	if value == nil {
		return nil
	}
	if v, ok := value.([]any); ok {
		return v
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return fmt.Sprint(value)
}

func ptr[T any](value T) *T {
	return &value
}
