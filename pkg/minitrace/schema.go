package minitrace

type Session struct {
	ID                 string             `json:"id"`
	SchemaVersion      string             `json:"schema_version"`
	Profile            string             `json:"profile"`
	ScenarioID         *string            `json:"scenario_id"`
	Quality            *string            `json:"quality"`
	Title              *string            `json:"title"`
	Summary            *string            `json:"summary"`
	Classification     string             `json:"classification"`
	Provenance         Provenance         `json:"provenance"`
	Flags              Flags              `json:"flags"`
	Environment        Environment        `json:"environment"`
	OperationalContext OperationalContext `json:"operational_context"`
	Timing             Timing             `json:"timing"`
	Condition          *Condition         `json:"condition"`
	Coordination       Coordination       `json:"coordination"`
	Handover           Handover           `json:"handover"`
	Turns              []Turn             `json:"turns"`
	ToolCalls          []ToolCall         `json:"tool_calls"`
	Events             []Event            `json:"events,omitempty"`
	Attachments        []Attachment       `json:"attachments,omitempty"`
	Outcome            *Outcome           `json:"outcome"`
	Annotations        []Annotation       `json:"annotations"`
	Metrics            Metrics            `json:"metrics"`
}

type Provenance struct {
	SourceFormat      string  `json:"source_format"`
	SourcePath        *string `json:"source_path"`
	ConvertedAt       string  `json:"converted_at"`
	ConverterVersion  string  `json:"converter_version"`
	OriginalSessionID *string `json:"original_session_id"`
	SourceFingerprint *string `json:"source_fingerprint,omitempty"`
	IdentityBasis     *string `json:"identity_basis,omitempty"`
}

type Flags struct {
	ForResearch   bool     `json:"for_research"`
	NeedsCleaning bool     `json:"needs_cleaning"`
	ContainsError bool     `json:"contains_error"`
	ContainsPII   bool     `json:"contains_pii"`
	Category      []string `json:"category"`
}

type Environment struct {
	Model          *string  `json:"model"`
	ModelVersion   *string  `json:"model_version"`
	Temperature    *float64 `json:"temperature"`
	ToolsEnabled   []string `json:"tools_enabled"`
	SystemPrompt   *string  `json:"system_prompt"`
	AgentFramework *string  `json:"agent_framework"`
	AgentVersion   *string  `json:"agent_version"`
	PlatformType   *string  `json:"platform_type"`
	ProviderHint   *string  `json:"provider_hint"`
}

type OperationalContext struct {
	WorkingDirectory *string `json:"working_directory"`
	GitBranch        *string `json:"git_branch"`
	GitRef           *string `json:"git_ref"`
	AutonomyLevel    *string `json:"autonomy_level"`
	Sandbox          *bool   `json:"sandbox"`
	FrameworkConfig  any     `json:"framework_config"`
}

type Timing struct {
	PrivacyLevel          string   `json:"privacy_level"`
	DurationSeconds       *float64 `json:"duration_seconds"`
	ActiveDurationSeconds *float64 `json:"active_duration_seconds"`
	StartedAt             *string  `json:"started_at"`
	EndedAt               *string  `json:"ended_at"`
	HourOfDay             *int     `json:"hour_of_day"`
	DayOfWeek             *int     `json:"day_of_week"`
}

type Condition struct {
	GuidanceVariant *string `json:"guidance_variant"`
	PermissionLevel *string `json:"permission_level"`
	Custom          any     `json:"custom"`
}

type Coordination struct {
	ProjectID          *string `json:"project_id"`
	PredecessorSession *string `json:"predecessor_session"`
	ConcurrentSessions *int    `json:"concurrent_sessions"`
	HumanAttention     string  `json:"human_attention"`
}

type Handover struct {
	Received *HandoverDocument `json:"received"`
	Produced *HandoverDocument `json:"produced"`
}

type HandoverDocument struct {
	FromSession      *string `json:"from_session"`
	ToSession        *string `json:"to_session"`
	Document         string  `json:"document"`
	StateDescription *string `json:"state_description"`
}

type Turn struct {
	Index             int            `json:"index"`
	Timestamp         *string        `json:"timestamp"`
	Role              string         `json:"role"`
	Source            *string        `json:"source"`
	Model             *string        `json:"model"`
	ContentType       *string        `json:"content_type"`
	InputChannel      *string        `json:"input_channel"`
	Content           string         `json:"content"`
	FrameworkMetadata any            `json:"framework_metadata"`
	ToolCallsInTurn   []string       `json:"tool_calls_in_turn"`
	Thinking          *string        `json:"thinking"`
	IntentMarkers     *IntentMarkers `json:"intent_markers"`
	Streaming         Streaming      `json:"streaming"`
	Usage             *Usage         `json:"usage"`
}

type IntentMarkers struct {
	Requested bool `json:"requested"`
	Inferred  bool `json:"inferred"`
	Proactive bool `json:"proactive"`
}

type Streaming struct {
	WasStreamed bool    `json:"was_streamed"`
	StreamLog   *string `json:"stream_log"`
}

type Usage struct {
	InputTokens         *int `json:"input_tokens"`
	OutputTokens        *int `json:"output_tokens"`
	CacheReadTokens     *int `json:"cache_read_tokens"`
	CacheCreationTokens *int `json:"cache_creation_tokens"`
	ReasoningTokens     *int `json:"reasoning_tokens"`
	ToolTokens          *int `json:"tool_tokens"`
}

type ToolCall struct {
	ID                string          `json:"id"`
	EmittingTurnIndex *int            `json:"emitting_turn_index"`
	Timestamp         *string         `json:"timestamp"`
	ToolName          string          `json:"tool_name"`
	OperationType     string          `json:"operation_type"`
	Input             ToolCallInput   `json:"input"`
	Output            ToolCallOutput  `json:"output"`
	Context           ToolCallContext `json:"context"`
	FrameworkMetadata any             `json:"framework_metadata"`
	SpawnedAgent      *SpawnedAgent   `json:"spawned_agent"`
}

type ToolCallInput struct {
	FilePath      *string `json:"file_path"`
	Command       *string `json:"command"`
	Justification *string `json:"justification"`
	Arguments     any     `json:"arguments"`
}

type ToolCallOutput struct {
	Success       bool    `json:"success"`
	Result        *string `json:"result"`
	Error         *string `json:"error"`
	ExitCode      *int    `json:"exit_code"`
	DurationMS    *int    `json:"duration_ms"`
	Truncated     bool    `json:"truncated"`
	FullBytes     *int    `json:"full_bytes"`
	FullHash      *string `json:"full_hash"`
	FullReference *string `json:"full_reference"`
	Redacted      *bool   `json:"redacted"`
	ContentOrigin *string `json:"content_origin"`
}

type ToolCallContext struct {
	PositionInSession *float64 `json:"position_in_session"`
	ToolsBefore       []string `json:"tools_before"`
	TimeSinceLastUser *float64 `json:"time_since_last_user"`
}

type SpawnedAgent struct {
	AgentType      string  `json:"agent_type"`
	TaskScope      string  `json:"task_scope"`
	SubSessionID   *string `json:"sub_session_id"`
	OutcomeSummary string  `json:"outcome_summary"`
}

type Outcome struct {
	Success      *bool    `json:"success"`
	Partial      bool     `json:"partial"`
	FailureCodes []string `json:"failure_codes"`
	OutcomeNotes *string  `json:"outcome_notes"`
}

type Event struct {
	ID                 string  `json:"id"`
	Timestamp          *string `json:"timestamp,omitempty"`
	TurnIndex          *int    `json:"turn_index,omitempty"`
	Ordinal            *int    `json:"ordinal,omitempty"`
	Kind               string  `json:"kind"`
	Role               string  `json:"role,omitempty"`
	ToolCallID         *string `json:"tool_call_id,omitempty"`
	AnnotationID       *string `json:"annotation_id,omitempty"`
	AttachmentID       *string `json:"attachment_id,omitempty"`
	Title              string  `json:"title,omitempty"`
	Summary            string  `json:"summary,omitempty"`
	Text               string  `json:"text,omitempty"`
	Severity           string  `json:"severity,omitempty"`
	CollapsedByDefault bool    `json:"collapsed_by_default,omitempty"`
	FrameworkMetadata  any     `json:"framework_metadata,omitempty"`
	RawJSON            any     `json:"raw_json,omitempty"`
}

type Attachment struct {
	ID                string  `json:"id"`
	Timestamp         *string `json:"timestamp,omitempty"`
	Kind              string  `json:"kind"`
	Name              string  `json:"name,omitempty"`
	MediaType         string  `json:"media_type,omitempty"`
	Path              string  `json:"path,omitempty"`
	URL               string  `json:"url,omitempty"`
	SizeBytes         *int    `json:"size_bytes,omitempty"`
	Hash              string  `json:"hash,omitempty"`
	ContentRef        string  `json:"content_ref,omitempty"`
	TextPreview       string  `json:"text_preview,omitempty"`
	TurnIndex         *int    `json:"turn_index,omitempty"`
	ToolCallID        *string `json:"tool_call_id,omitempty"`
	EventID           *string `json:"event_id,omitempty"`
	FrameworkMetadata any     `json:"framework_metadata,omitempty"`
	RawJSON           any     `json:"raw_json,omitempty"`
}

type Annotation struct {
	ID               string            `json:"id"`
	Timestamp        string            `json:"timestamp"`
	Annotator        string            `json:"annotator"`
	Scope            AnnotationScope   `json:"scope"`
	Content          AnnotationContent `json:"content"`
	TaxonomyMappings TaxonomyMappings  `json:"taxonomy_mappings"`
	Classification   *string           `json:"classification"`
}

type AnnotationScope struct {
	Type     string `json:"type"`
	TargetID string `json:"target_id"`
}

type AnnotationContent struct {
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
}

type TaxonomyMappings struct {
	Minitrace []string `json:"minitrace"`
	Mast      []string `json:"mast"`
	Toolemu   []string `json:"toolemu"`
}

type Metrics struct {
	TurnCount                int      `json:"turn_count"`
	ToolCallCount            int      `json:"tool_call_count"`
	ReadCount                int      `json:"read_count"`
	ModifyCount              int      `json:"modify_count"`
	CreateCount              int      `json:"create_count"`
	ExecuteCount             int      `json:"execute_count"`
	DelegateCount            int      `json:"delegate_count"`
	ReadRatio                *float64 `json:"read_ratio"`
	TimeToFirstAction        *float64 `json:"time_to_first_action"`
	IdleRatio                *float64 `json:"idle_ratio"`
	TotalInputTokens         *int     `json:"total_input_tokens"`
	TotalOutputTokens        *int     `json:"total_output_tokens"`
	TotalCacheReadTokens     *int     `json:"total_cache_read_tokens"`
	TotalCacheCreationTokens *int     `json:"total_cache_creation_tokens"`
	TotalReasoningTokens     *int     `json:"total_reasoning_tokens"`
	TotalToolTokens          *int     `json:"total_tool_tokens"`
	SessionCost              *float64 `json:"session_cost"`
	SubagentCount            int      `json:"subagent_count"`
	SubagentToolCalls        int      `json:"subagent_tool_calls"`
	ModelSwitches            *int     `json:"model_switches"`
	UniqueModels             *int     `json:"unique_models"`
	MedianResponseTokens     *int     `json:"median_response_tokens"`
	MaxResponseTokens        *int     `json:"max_response_tokens"`
}

type TokenTotals struct {
	Input         int
	Output        int
	CacheRead     int
	CacheCreation int
	Reasoning     int
	Tool          int
	Cost          *float64
}
