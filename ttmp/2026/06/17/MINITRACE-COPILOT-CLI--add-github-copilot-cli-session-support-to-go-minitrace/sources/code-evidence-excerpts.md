# Code evidence excerpts
## convert/root.go
     1	package convert
     2	
     3	import "github.com/spf13/cobra"
     4	
     5	func NewCommand() (*cobra.Command, error) {
     6		root := &cobra.Command{
     7			Use:   "convert",
     8			Short: "Convert supported native session stores into minitrace output",
     9			Long: `Bootstrap command group for conversion.
    10	
    11	Claude Code and Codex are the first-class targets. The current skeleton exposes
    12	the intended Glazed command surface and discovery-backed planning output while
    13	the actual conversion engine is ported.`,
    14		}
    15	
    16		claudeCmd, err := NewClaudeCodeCommand()
    17		if err != nil {
    18			return nil, err
    19		}
    20		codexCmd, err := NewCodexCommand()
    21		if err != nil {
    22			return nil, err
    23		}
    24		piCmd, err := NewPiCommand()
    25		if err != nil {
    26			return nil, err
    27		}
    28		claudeAICmd, err := NewClaudeAICommand()
    29		if err != nil {
    30			return nil, err
    31		}
    32		chatGPTCmd, err := NewChatGPTCommand()
    33		if err != nil {
    34			return nil, err
    35		}
    36		chatGPTJSONCmd, err := NewChatGPTJSONCommand()
    37		if err != nil {
    38			return nil, err
    39		}
    40		turnsDBCmd, err := NewTurnsDBCommand()
    41		if err != nil {
    42			return nil, err
    43		}
    44	
    45		root.AddCommand(claudeCmd, codexCmd, piCmd, claudeAICmd, chatGPTCmd, chatGPTJSONCmd, turnsDBCmd)
    46		return root, nil
    47	}
## discover/root.go
     1	package discover
     2	
     3	import "github.com/spf13/cobra"
     4	
     5	func NewCommand() (*cobra.Command, error) {
     6		root := &cobra.Command{
     7			Use:   "discover",
     8			Short: "Inspect native session stores without converting them",
     9			Long: `Inspect native Claude Code and Codex session stores and emit structured
    10	rows describing the sessions that would be considered for conversion.`,
    11		}
    12	
    13		claudeCmd, err := NewClaudeCodeCommand()
    14		if err != nil {
    15			return nil, err
    16		}
    17		codexCmd, err := NewCodexCommand()
    18		if err != nil {
    19			return nil, err
    20		}
    21		piCmd, err := NewPiCommand()
    22		if err != nil {
    23			return nil, err
    24		}
    25	
    26		root.AddCommand(claudeCmd, codexCmd, piCmd)
    27		return root, nil
    28	}
## adapters/types.go
     1	package adapters
     2	
     3	type SessionLocator struct {
     4		ID         string
     5		FormatHint string
     6		SourcePath string
     7	}
## codex discover
     1	package codex
     2	
     3	import (
     4		"bufio"
     5		"encoding/json"
     6		"os"
     7		"path/filepath"
     8		"sort"
     9		"strings"
    10	
    11		"github.com/go-go-golems/go-minitrace/pkg/adapters"
    12	)
    13	
    14	func Discover(sourceDir string) ([]adapters.SessionLocator, error) {
    15		root, err := expandHome(sourceDir)
    16		if err != nil {
    17			return nil, err
    18		}
    19	
    20		searchRoot := filepath.Join(root, "sessions")
    21		if st, err := os.Stat(searchRoot); err != nil || !st.IsDir() {
    22			searchRoot = root
    23		}
    24	
    25		ret := []adapters.SessionLocator{}
    26		err = filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, walkErr error) error {
    27			if walkErr != nil {
    28				return walkErr
    29			}
    30			if d.IsDir() || filepath.Ext(path) != ".jsonl" {
    31				return nil
    32			}
    33	
    34			sid := strings.TrimSuffix(filepath.Base(path), ".jsonl")
    35			ret = append(ret, adapters.SessionLocator{
    36				ID:         sid,
    37				FormatHint: detectFormat(path),
    38				SourcePath: path,
    39			})
    40			return nil
    41		})
    42		if err != nil {
    43			return nil, err
    44		}
    45	
    46		sort.Slice(ret, func(i, j int) bool { return ret[i].SourcePath < ret[j].SourcePath })
    47		return ret, nil
    48	}
    49	
    50	func detectFormat(path string) string {
    51		f, err := os.Open(path)
    52		if err != nil {
    53			return "unknown-jsonl"
    54		}
    55		defer func() { _ = f.Close() }()
    56	
    57		scanner := bufio.NewScanner(f)
    58		for i := 0; i < 5 && scanner.Scan(); i++ {
    59			line := strings.TrimSpace(scanner.Text())
    60			if line == "" {
    61				continue
    62			}
    63			var payload map[string]any
    64			if err := json.Unmarshal([]byte(line), &payload); err != nil {
    65				continue
    66			}
    67			type_, _ := payload["type"].(string)
    68			switch type_ {
    69			case "session_meta", "response_item", "event_msg", "turn_context":
    70				return "session-jsonl-v1"
    71			case "thread.started", "turn.started", "turn.completed", "item.started", "item.completed":
    72				return "exec-jsonl-v1"
    73			}
    74		}
    75		return "unknown-jsonl"
    76	}
    77	
    78	func expandHome(path string) (string, error) {
    79		if path == "" || path[0] != '~' {
    80			return filepath.Clean(path), nil
    81		}
    82		home, err := os.UserHomeDir()
    83		if err != nil {
    84			return "", err
    85		}
    86		if path == "~" {
    87			return home, nil
    88		}
    89		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
    90	}
## codex convert skeleton
     1	package codex
     2	
     3	import (
     4		"bufio"
     5		"encoding/json"
     6		"fmt"
     7		"os"
     8		"regexp"
     9		"sort"
    10		"strconv"
    11		"strings"
    12		"time"
    13	
    14		"github.com/go-go-golems/go-minitrace/pkg/adapters"
    15		"github.com/go-go-golems/go-minitrace/pkg/minitrace"
    16		"github.com/pkg/errors"
    17	)
    18	
    19	const (
    20		AdapterVersion      = "go-minitrace-codex-adapter-dev"
    21		SourceFormatExec    = "codex-exec-jsonl-v1"
    22		SourceFormatSession = "codex-session-jsonl-v1"
    23	)
    24	
    25	var (
    26		readPatterns = []*regexp.Regexp{
    27			regexp.MustCompile(`^cat\s`),
    28			regexp.MustCompile(`^head\s`),
    29			regexp.MustCompile(`^tail\s`),
    30			regexp.MustCompile(`^less\s`),
    31			regexp.MustCompile(`^more\s`),
    32			regexp.MustCompile(`^find\s`),
    33			regexp.MustCompile(`^ls\s`),
    34			regexp.MustCompile(`^tree\s`),
    35			regexp.MustCompile(`^wc\s`),
    36			regexp.MustCompile(`^grep\s`),
    37			regexp.MustCompile(`^rg\s`),
    38			regexp.MustCompile(`^ag\s`),
    39			regexp.MustCompile(`^ack\s`),
    40			regexp.MustCompile(`^file\s`),
    41			regexp.MustCompile(`^stat\s`),
    42			regexp.MustCompile(`^du\s`),
    43			regexp.MustCompile(`^df\s`),
    44			regexp.MustCompile(`^git\s+(log|show|diff|status|branch|blame)\b`),
    45			regexp.MustCompile(`^python3?\s+-c\s+.*open.*read`),
    46		}
    47		modifyPatterns = []*regexp.Regexp{
    48			regexp.MustCompile(`^sed\s+-i`),
    49			regexp.MustCompile(`^perl\s+-i`),
    50			regexp.MustCompile(`^patch\s`),
    51			regexp.MustCompile(`^git\s+apply\b`),
    52			regexp.MustCompile(`^chmod\s`),
    53			regexp.MustCompile(`^chown\s`),
    54		}
    55	)
    56	
    57	func ConvertLocator(locator adapters.SessionLocator) (*minitrace.Session, error) {
    58		records, err := parseJSONLFile(locator.SourcePath)
    59		if err != nil {
    60			return nil, err
    61		}
    62		return ConvertRecords(records, locator.ID, locator.SourcePath, locator.FormatHint)
    63	}
    64	
    65	func ConvertRecords(records []map[string]any, sessionID, sourcePath, formatHint string) (*minitrace.Session, error) {
    66		actualFormat := formatHint
    67		if actualFormat == "" || actualFormat == "unknown-jsonl" {
    68			actualFormat = detectFormatRecords(records)
    69		}
    70	
    71		var (
    72			turns       []minitrace.Turn
    73			toolCalls   []minitrace.ToolCall
    74			annotations []minitrace.Annotation
    75			timestamps  []time.Time
    76			tokenTotals *minitrace.TokenTotals
    77			metadata    codexMetadata
    78		)
    79	
    80		switch actualFormat {
    81		case "session-jsonl-v1":
    82			turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseSessionJSONL(records)
    83		case "exec-jsonl-v1":
    84			turns, toolCalls, annotations, timestamps, tokenTotals, metadata = parseExecJSONL(records)
    85		default:
    86			return nil, errors.Errorf("unsupported Codex format hint: %s", actualFormat)
    87		}
    88	
    89		if metadata.SessionID != "" {
    90			sessionID = metadata.SessionID
    91		}
    92	
    93		session := minitrace.BuildSessionSkeleton(sessionID, "codex", sourceFormatName(actualFormat), AdapterVersion)
    94		session.Environment.PlatformType = ptr("agent")
    95		session.Environment.SystemPrompt = optionalString(metadata.SystemPrompt)
    96		session.Environment.Model = optionalString(metadata.Model)
    97		session.Environment.AgentVersion = optionalString(metadata.CLIVersion)
    98		session.Environment.ProviderHint = providerHint(metadata.ModelProvider)
    99		session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
   100		session.OperationalContext.WorkingDirectory = optionalNormalizedPath(metadata.CWD)
   101		session.OperationalContext.AutonomyLevel = optionalString(mapApprovalPolicy(metadata.ApprovalPolicy))
   102		session.OperationalContext.Sandbox = sandboxValue(metadata.SandboxPolicy)
   103		session.OperationalContext.FrameworkConfig = frameworkConfig(metadata)
   104		if sourcePath != "" {
   105			session.Provenance.SourcePath = ptr(sourcePath)
   106		}
   107	
   108		deduped, duplicateCount := minitrace.DeduplicateToolCalls(toolCalls, nil)
   109		toolCalls = deduped
   110		if duplicateCount > 0 {
   111			annotations = append(annotations, minitrace.BuildAnnotation(
   112				"ann-dedup-"+truncateID(sessionID),
   113				"adapter",
   114				"session",
   115				sessionID,
   116				"observation",
   117				fmt.Sprintf("Deduplicated %d duplicate tool calls", duplicateCount),
   118				fmt.Sprintf("Removed %d tool calls with duplicate IDs.", duplicateCount),
   119				[]string{"deduplication", "data-quality"},
   120				nil,
   121			))
   122		}
   123		minitrace.ComputeToolCallContext(toolCalls)
   124		events, attachments := buildCodexEventsAndAttachments(toolCalls, metadata)
   125	
   126		timing := minitrace.ComputeTiming(timestamps)
   127		quality := minitrace.AssignQualityTier(turns, toolCalls)
   128		containsPII := minitrace.DetectPIIInPaths(toolCalls)
   129	
   130		session.Quality = &quality
   131		session.Title = minitrace.ExtractTitle(turns, 80)
   132		session.Timing = timing
   133		session.Turns = turns
   134		session.ToolCalls = toolCalls
   135		session.Events = events
   136		session.Attachments = attachments
   137		session.Annotations = annotations
   138		session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, 0, tokenTotals)
   139		session.Flags.ContainsPII = containsPII
   140		session.Flags.ForResearch = quality == "A" && !containsPII
   141		session.Flags.NeedsCleaning = quality != "A" || containsPII
   142		if containsPII {
   143			session.Classification = "confidential"
   144		}
   145	
   146		return &session, nil
   147	}
   148	
   149	type codexMetadata struct {
   150		SessionID               string
   151		Model                   string
   152		ModelProvider           string
   153		CWD                     string
   154		CLIVersion              string
   155		Originator              string
   156		SessionSource           string
   157		SystemPrompt            string
   158		ApprovalPolicy          string
   159		SandboxPolicy           string
   160		SandboxPolicyDetail     any
   161		Personality             string
   162		CollaborationMode       string
   163		CollaborationModeDetail any
   164		ReasoningEffort         string
   165		Timezone                string
   166		ContextWindow           int
   167		TruncationPolicy        any
   168		LatestRateLimits        any
   169	}
   170	
   171	func buildCodexEventsAndAttachments(toolCalls []minitrace.ToolCall, metadata codexMetadata) ([]minitrace.Event, []minitrace.Attachment) {
   172		events := []minitrace.Event{}
   173		attachments := []minitrace.Attachment{}
   174		for _, toolCall := range toolCalls {
   175			switch toolCall.ToolName {
   176			case "view_image":
   177				attachment := buildCodexImageAttachment(toolCall)
   178				event := buildCodexToolEvent(toolCall, len(events), "image_view", "Codex image view", firstNonEmptyPointer(toolCall.Input.FilePath, toolCall.Output.Result))
   179				event.AttachmentID = &attachment.ID
   180				attachment.EventID = &event.ID
## minitrace schema
     1	package minitrace
     2	
     3	type Session struct {
     4		ID                 string             `json:"id"`
     5		SchemaVersion      string             `json:"schema_version"`
     6		Profile            string             `json:"profile"`
     7		ScenarioID         *string            `json:"scenario_id"`
     8		Quality            *string            `json:"quality"`
     9		Title              *string            `json:"title"`
    10		Summary            *string            `json:"summary"`
    11		Classification     string             `json:"classification"`
    12		Provenance         Provenance         `json:"provenance"`
    13		Flags              Flags              `json:"flags"`
    14		Environment        Environment        `json:"environment"`
    15		OperationalContext OperationalContext `json:"operational_context"`
    16		Timing             Timing             `json:"timing"`
    17		Condition          *Condition         `json:"condition"`
    18		Coordination       Coordination       `json:"coordination"`
    19		Handover           Handover           `json:"handover"`
    20		Turns              []Turn             `json:"turns"`
    21		ToolCalls          []ToolCall         `json:"tool_calls"`
    22		Events             []Event            `json:"events,omitempty"`
    23		Attachments        []Attachment       `json:"attachments,omitempty"`
    24		Outcome            *Outcome           `json:"outcome"`
    25		Annotations        []Annotation       `json:"annotations"`
    26		Metrics            Metrics            `json:"metrics"`
    27	}
    28	
    29	type Provenance struct {
    30		SourceFormat      string  `json:"source_format"`
    31		SourcePath        *string `json:"source_path"`
    32		ConvertedAt       string  `json:"converted_at"`
    33		ConverterVersion  string  `json:"converter_version"`
    34		OriginalSessionID *string `json:"original_session_id"`
    35	}
    36	
    37	type Flags struct {
    38		ForResearch   bool     `json:"for_research"`
    39		NeedsCleaning bool     `json:"needs_cleaning"`
    40		ContainsError bool     `json:"contains_error"`
    41		ContainsPII   bool     `json:"contains_pii"`
    42		Category      []string `json:"category"`
    43	}
    44	
    45	type Environment struct {
    46		Model          *string  `json:"model"`
    47		ModelVersion   *string  `json:"model_version"`
    48		Temperature    *float64 `json:"temperature"`
    49		ToolsEnabled   []string `json:"tools_enabled"`
    50		SystemPrompt   *string  `json:"system_prompt"`
    51		AgentFramework *string  `json:"agent_framework"`
    52		AgentVersion   *string  `json:"agent_version"`
    53		PlatformType   *string  `json:"platform_type"`
    54		ProviderHint   *string  `json:"provider_hint"`
    55	}
    56	
    57	type OperationalContext struct {
    58		WorkingDirectory *string `json:"working_directory"`
    59		GitBranch        *string `json:"git_branch"`
    60		GitRef           *string `json:"git_ref"`
    61		AutonomyLevel    *string `json:"autonomy_level"`
    62		Sandbox          *bool   `json:"sandbox"`
    63		FrameworkConfig  any     `json:"framework_config"`
    64	}
    65	
    66	type Timing struct {
    67		PrivacyLevel          string   `json:"privacy_level"`
    68		DurationSeconds       *float64 `json:"duration_seconds"`
    69		ActiveDurationSeconds *float64 `json:"active_duration_seconds"`
    70		StartedAt             *string  `json:"started_at"`
    71		EndedAt               *string  `json:"ended_at"`
    72		HourOfDay             *int     `json:"hour_of_day"`
    73		DayOfWeek             *int     `json:"day_of_week"`
    74	}
    75	
    76	type Condition struct {
    77		GuidanceVariant *string `json:"guidance_variant"`
    78		PermissionLevel *string `json:"permission_level"`
    79		Custom          any     `json:"custom"`
    80	}
    81	
    82	type Coordination struct {
    83		ProjectID          *string `json:"project_id"`
    84		PredecessorSession *string `json:"predecessor_session"`
    85		ConcurrentSessions *int    `json:"concurrent_sessions"`
    86		HumanAttention     string  `json:"human_attention"`
    87	}
    88	
    89	type Handover struct {
    90		Received *HandoverDocument `json:"received"`
    91		Produced *HandoverDocument `json:"produced"`
    92	}
    93	
    94	type HandoverDocument struct {
    95		FromSession      *string `json:"from_session"`
    96		ToSession        *string `json:"to_session"`
    97		Document         string  `json:"document"`
    98		StateDescription *string `json:"state_description"`
    99	}
   100	
   101	type Turn struct {
   102		Index             int            `json:"index"`
   103		Timestamp         *string        `json:"timestamp"`
   104		Role              string         `json:"role"`
   105		Source            *string        `json:"source"`
   106		Model             *string        `json:"model"`
   107		ContentType       *string        `json:"content_type"`
   108		InputChannel      *string        `json:"input_channel"`
   109		Content           string         `json:"content"`
   110		FrameworkMetadata any            `json:"framework_metadata"`
   111		ToolCallsInTurn   []string       `json:"tool_calls_in_turn"`
   112		Thinking          *string        `json:"thinking"`
   113		IntentMarkers     *IntentMarkers `json:"intent_markers"`
   114		Streaming         Streaming      `json:"streaming"`
   115		Usage             *Usage         `json:"usage"`
   116	}
   117	
   118	type IntentMarkers struct {
   119		Requested bool `json:"requested"`
   120		Inferred  bool `json:"inferred"`
   121		Proactive bool `json:"proactive"`
   122	}
   123	
   124	type Streaming struct {
   125		WasStreamed bool    `json:"was_streamed"`
   126		StreamLog   *string `json:"stream_log"`
   127	}
   128	
   129	type Usage struct {
   130		InputTokens         *int `json:"input_tokens"`
   131		OutputTokens        *int `json:"output_tokens"`
   132		CacheReadTokens     *int `json:"cache_read_tokens"`
   133		CacheCreationTokens *int `json:"cache_creation_tokens"`
   134		ReasoningTokens     *int `json:"reasoning_tokens"`
   135		ToolTokens          *int `json:"tool_tokens"`
   136	}
   137	
   138	type ToolCall struct {
   139		ID                string          `json:"id"`
   140		EmittingTurnIndex *int            `json:"emitting_turn_index"`
   141		Timestamp         *string         `json:"timestamp"`
   142		ToolName          string          `json:"tool_name"`
   143		OperationType     string          `json:"operation_type"`
   144		Input             ToolCallInput   `json:"input"`
   145		Output            ToolCallOutput  `json:"output"`
   146		Context           ToolCallContext `json:"context"`
   147		FrameworkMetadata any             `json:"framework_metadata"`
   148		SpawnedAgent      *SpawnedAgent   `json:"spawned_agent"`
   149	}
   150	
   151	type ToolCallInput struct {
   152		FilePath      *string `json:"file_path"`
   153		Command       *string `json:"command"`
   154		Justification *string `json:"justification"`
   155		Arguments     any     `json:"arguments"`
   156	}
   157	
   158	type ToolCallOutput struct {
   159		Success       bool    `json:"success"`
   160		Result        *string `json:"result"`
   161		Error         *string `json:"error"`
   162		ExitCode      *int    `json:"exit_code"`
   163		DurationMS    *int    `json:"duration_ms"`
   164		Truncated     bool    `json:"truncated"`
   165		FullBytes     *int    `json:"full_bytes"`
   166		FullHash      *string `json:"full_hash"`
   167		FullReference *string `json:"full_reference"`
   168		Redacted      *bool   `json:"redacted"`
   169		ContentOrigin *string `json:"content_origin"`
   170	}
   171	
   172	type ToolCallContext struct {
   173		PositionInSession *float64 `json:"position_in_session"`
   174		ToolsBefore       []string `json:"tools_before"`
   175		TimeSinceLastUser *float64 `json:"time_since_last_user"`
   176	}
   177	
   178	type SpawnedAgent struct {
   179		AgentType      string  `json:"agent_type"`
   180		TaskScope      string  `json:"task_scope"`
   181		SubSessionID   *string `json:"sub_session_id"`
   182		OutcomeSummary string  `json:"outcome_summary"`
   183	}
   184	
   185	type Outcome struct {
   186		Success      *bool    `json:"success"`
   187		Partial      bool     `json:"partial"`
   188		FailureCodes []string `json:"failure_codes"`
   189		OutcomeNotes *string  `json:"outcome_notes"`
   190	}
   191	
   192	type Event struct {
   193		ID                 string  `json:"id"`
   194		Timestamp          *string `json:"timestamp,omitempty"`
   195		TurnIndex          *int    `json:"turn_index,omitempty"`
   196		Ordinal            *int    `json:"ordinal,omitempty"`
   197		Kind               string  `json:"kind"`
   198		Role               string  `json:"role,omitempty"`
   199		ToolCallID         *string `json:"tool_call_id,omitempty"`
   200		AnnotationID       *string `json:"annotation_id,omitempty"`
   201		AttachmentID       *string `json:"attachment_id,omitempty"`
   202		Title              string  `json:"title,omitempty"`
   203		Summary            string  `json:"summary,omitempty"`
   204		Text               string  `json:"text,omitempty"`
   205		Severity           string  `json:"severity,omitempty"`
   206		CollapsedByDefault bool    `json:"collapsed_by_default,omitempty"`
   207		FrameworkMetadata  any     `json:"framework_metadata,omitempty"`
   208		RawJSON            any     `json:"raw_json,omitempty"`
   209	}
   210	
   211	type Attachment struct {
   212		ID                string  `json:"id"`
   213		Timestamp         *string `json:"timestamp,omitempty"`
   214		Kind              string  `json:"kind"`
   215		Name              string  `json:"name,omitempty"`
   216		MediaType         string  `json:"media_type,omitempty"`
   217		Path              string  `json:"path,omitempty"`
   218		URL               string  `json:"url,omitempty"`
   219		SizeBytes         *int    `json:"size_bytes,omitempty"`
   220		Hash              string  `json:"hash,omitempty"`
