package minitracedb

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	_ "github.com/mattn/go-sqlite3"
)

func TestMaterializeSessionPopulatesCoreTables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	session := fixtureSession()
	if err := MaterializeSession(ctx, db, session, MaterializeOptions{SourcePath: "/tmp/session.minitrace.json"}); err != nil {
		t.Fatalf("MaterializeSession: %v", err)
	}
	assertCount(t, db, "sessions", 1)
	assertCount(t, db, "turns", 2)
	assertCount(t, db, "tool_calls", 1)
	assertCount(t, db, "turn_tool_calls", 1)
	assertCount(t, db, "files", 1)
	assertCount(t, db, "metrics", 1)
	assertCount(t, db, "events", 3)

	runner, err := NewQueryRunner(db, AllowedTableNames(), QueryOptions{})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}
	row, err := runner.QueryOne(ctx, `SELECT tool_name, operation_type, file_path FROM tool_calls`)
	if err != nil {
		t.Fatalf("query tool calls: %v", err)
	}
	if row["tool_name"] != "Read" || row["operation_type"] != "read" || row["file_path"] != "app.go" {
		t.Fatalf("unexpected tool call row: %#v", row)
	}
}

func TestMaterializeSessionPopulatesPhaseThreeFields(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	session := richFixtureSession()
	if err := MaterializeSession(ctx, db, session, MaterializeOptions{SourcePath: "/tmp/rich.minitrace.json"}); err != nil {
		t.Fatalf("MaterializeSession: %v", err)
	}
	assertCount(t, db, "annotations", 1)
	assertCount(t, db, "handovers", 2)
	assertCount(t, db, "events", 4)

	runner, err := NewQueryRunner(db, AllowedTableNames(), QueryOptions{})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}

	sessionRow, err := runner.QueryOne(ctx, `SELECT scenario_id, quality, git_branch, sandbox, duration_seconds, project_id, outcome_success, category_json, tools_enabled_json, framework_config_json FROM sessions`)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	assertValue(t, sessionRow, "scenario_id", "scenario-1")
	assertValue(t, sessionRow, "quality", "gold")
	assertValue(t, sessionRow, "git_branch", "main")
	assertValue(t, sessionRow, "sandbox", int64(1))
	assertValue(t, sessionRow, "duration_seconds", 42.5)
	assertValue(t, sessionRow, "project_id", "project-1")
	assertValue(t, sessionRow, "outcome_success", int64(1))
	assertContains(t, sessionRow, "category_json", "research")
	assertContains(t, sessionRow, "tools_enabled_json", "Read")
	assertContains(t, sessionRow, "framework_config_json", "approval")

	turnRow, err := runner.QueryOne(ctx, `SELECT intent_requested, intent_inferred, stream_log, cache_read_tokens, cache_creation_tokens, tool_tokens, framework_metadata_json FROM turns WHERE turn_index = 1`)
	if err != nil {
		t.Fatalf("query turns: %v", err)
	}
	assertValue(t, turnRow, "intent_requested", int64(1))
	assertValue(t, turnRow, "intent_inferred", int64(1))
	assertValue(t, turnRow, "stream_log", "delta-stream")
	assertValue(t, turnRow, "cache_read_tokens", int64(3))
	assertValue(t, turnRow, "cache_creation_tokens", int64(4))
	assertValue(t, turnRow, "tool_tokens", int64(6))
	assertContains(t, turnRow, "framework_metadata_json", "turn-meta")

	toolRow, err := runner.QueryOne(ctx, `SELECT full_hash, redacted, content_origin, position_in_session, tools_before_json, spawned_agent_type, spawned_agent_sub_session_id, spawned_agent_outcome_summary, framework_metadata_json FROM tool_calls`)
	if err != nil {
		t.Fatalf("query tool_calls: %v", err)
	}
	assertValue(t, toolRow, "full_hash", "sha256:abc")
	assertValue(t, toolRow, "redacted", int64(0))
	assertValue(t, toolRow, "content_origin", "stdout")
	assertValue(t, toolRow, "position_in_session", 0.5)
	assertContains(t, toolRow, "tools_before_json", "List")
	assertValue(t, toolRow, "spawned_agent_type", "codex")
	assertValue(t, toolRow, "spawned_agent_sub_session_id", "sub-1")
	assertValue(t, toolRow, "spawned_agent_outcome_summary", "completed audit")
	assertContains(t, toolRow, "framework_metadata_json", "tool-meta")

	annotationRow, err := runner.QueryOne(ctx, `SELECT scope_type, target_id, category, classification, tags_json, minitrace_taxonomy_json FROM annotations`)
	if err != nil {
		t.Fatalf("query annotations: %v", err)
	}
	assertValue(t, annotationRow, "scope_type", "tool_call")
	assertValue(t, annotationRow, "target_id", "tool-1")
	assertValue(t, annotationRow, "category", "risk")
	assertValue(t, annotationRow, "classification", "high")
	assertContains(t, annotationRow, "tags_json", "file-read")
	assertContains(t, annotationRow, "minitrace_taxonomy_json", "taxonomy/risk")

	handoverRow, err := runner.QueryOne(ctx, `SELECT document, state_description FROM handovers WHERE direction = 'produced'`)
	if err != nil {
		t.Fatalf("query handovers: %v", err)
	}
	assertValue(t, handoverRow, "document", "next steps")
	assertValue(t, handoverRow, "state_description", "ready")

	metricsRow, err := runner.QueryOne(ctx, `SELECT total_cache_read_tokens, total_cache_creation_tokens, total_reasoning_tokens, total_tool_tokens, idle_ratio, subagent_count, model_switches FROM metrics`)
	if err != nil {
		t.Fatalf("query metrics: %v", err)
	}
	assertValue(t, metricsRow, "total_cache_read_tokens", int64(3))
	assertValue(t, metricsRow, "total_cache_creation_tokens", int64(4))
	assertValue(t, metricsRow, "total_reasoning_tokens", int64(5))
	assertValue(t, metricsRow, "total_tool_tokens", int64(6))
	assertValue(t, metricsRow, "idle_ratio", 0.25)
	assertValue(t, metricsRow, "subagent_count", int64(1))
	assertValue(t, metricsRow, "model_switches", int64(1))
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}

func assertValue(t *testing.T, row map[string]any, key string, want any) {
	t.Helper()
	if row[key] != want {
		t.Fatalf("%s = %#v (%T), want %#v (%T); row=%#v", key, row[key], row[key], want, want, row)
	}
}

func assertContains(t *testing.T, row map[string]any, key, substr string) {
	t.Helper()
	value, ok := row[key].(string)
	if !ok {
		t.Fatalf("%s = %#v (%T), want string containing %q", key, row[key], row[key], substr)
	}
	if !strings.Contains(value, substr) {
		t.Fatalf("%s = %q, want substring %q", key, value, substr)
	}
}

func fixtureSession() *minitrace.Session {
	session := minitrace.BuildSessionSkeleton("session-1", "pi", "minitrace-json-v1", "test")
	title := "Fixture session"
	session.Title = &title
	session.Turns = []minitrace.Turn{
		{Index: 0, Role: "user", Content: "Read app.go"},
		{Index: 1, Role: "assistant", Content: "Reading", ToolCallsInTurn: []string{"tool-1"}},
	}
	path := "app.go"
	result := "package main"
	emittingTurn := 1
	session.ToolCalls = []minitrace.ToolCall{{
		ID:                "tool-1",
		EmittingTurnIndex: &emittingTurn,
		ToolName:          "Read",
		OperationType:     "read",
		Input:             minitrace.ToolCallInput{FilePath: &path},
		Output:            minitrace.ToolCallOutput{Success: true, Result: &result},
	}}
	metrics := minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, len(session.Annotations), nil)
	session.Metrics = metrics
	return &session
}

func richFixtureSession() *minitrace.Session {
	session := fixtureSession()
	scenarioID := "scenario-1"
	quality := "gold"
	session.ScenarioID = &scenarioID
	session.Quality = &quality
	session.Flags.ForResearch = true
	session.Flags.NeedsCleaning = true
	session.Flags.ContainsPII = true
	session.Flags.Category = []string{"research", "audit"}
	modelVersion := "2026-06"
	temperature := 0.2
	systemPrompt := "be concise"
	agentVersion := "v1"
	platformType := "local"
	providerHint := "goja"
	session.Environment.ModelVersion = &modelVersion
	session.Environment.Temperature = &temperature
	session.Environment.ToolsEnabled = []string{"Read", "Bash"}
	session.Environment.SystemPrompt = &systemPrompt
	session.Environment.AgentVersion = &agentVersion
	session.Environment.PlatformType = &platformType
	session.Environment.ProviderHint = &providerHint
	gitBranch := "main"
	gitRef := "abc123"
	autonomy := "supervised"
	sandbox := true
	session.OperationalContext.GitBranch = &gitBranch
	session.OperationalContext.GitRef = &gitRef
	session.OperationalContext.AutonomyLevel = &autonomy
	session.OperationalContext.Sandbox = &sandbox
	session.OperationalContext.FrameworkConfig = map[string]any{"approval": "on-request"}
	duration := 42.5
	activeDuration := 40.0
	hour := 14
	day := 2
	session.Timing.PrivacyLevel = "internal"
	session.Timing.DurationSeconds = &duration
	session.Timing.ActiveDurationSeconds = &activeDuration
	session.Timing.HourOfDay = &hour
	session.Timing.DayOfWeek = &day
	guidanceVariant := "baseline"
	permissionLevel := "workspace-write"
	session.Condition = &minitrace.Condition{GuidanceVariant: &guidanceVariant, PermissionLevel: &permissionLevel, Custom: map[string]any{"bucket": "A"}}
	projectID := "project-1"
	predecessor := "session-0"
	concurrent := 2
	session.Coordination.ProjectID = &projectID
	session.Coordination.PredecessorSession = &predecessor
	session.Coordination.ConcurrentSessions = &concurrent
	session.Coordination.HumanAttention = "medium"
	fromSession := "session-0"
	toSession := "session-2"
	receivedState := "warm"
	producedState := "ready"
	session.Handover.Received = &minitrace.HandoverDocument{FromSession: &fromSession, Document: "prior state", StateDescription: &receivedState}
	session.Handover.Produced = &minitrace.HandoverDocument{ToSession: &toSession, Document: "next steps", StateDescription: &producedState}
	outcomeSuccess := true
	outcomeNotes := "finished"
	session.Outcome = &minitrace.Outcome{Success: &outcomeSuccess, Partial: false, FailureCodes: []string{"none"}, OutcomeNotes: &outcomeNotes}
	streamLog := "delta-stream"
	inputTokens := 1
	outputTokens := 2
	cacheReadTokens := 3
	cacheCreationTokens := 4
	reasoningTokens := 5
	toolTokens := 6
	session.Turns[1].Streaming.StreamLog = &streamLog
	session.Turns[1].IntentMarkers = &minitrace.IntentMarkers{Requested: true, Inferred: true, Proactive: false}
	session.Turns[1].FrameworkMetadata = map[string]any{"turn-meta": true}
	session.Turns[1].Usage = &minitrace.Usage{InputTokens: &inputTokens, OutputTokens: &outputTokens, CacheReadTokens: &cacheReadTokens, CacheCreationTokens: &cacheCreationTokens, ReasoningTokens: &reasoningTokens, ToolTokens: &toolTokens}
	fullBytes := 123
	fullHash := "sha256:abc"
	fullReference := "artifact://tool-1"
	redacted := false
	contentOrigin := "stdout"
	position := 0.5
	timeSinceUser := 1.25
	subSessionID := "sub-1"
	session.ToolCalls[0].Output.FullBytes = &fullBytes
	session.ToolCalls[0].Output.FullHash = &fullHash
	session.ToolCalls[0].Output.FullReference = &fullReference
	session.ToolCalls[0].Output.Redacted = &redacted
	session.ToolCalls[0].Output.ContentOrigin = &contentOrigin
	session.ToolCalls[0].Context.PositionInSession = &position
	session.ToolCalls[0].Context.ToolsBefore = []string{"List"}
	session.ToolCalls[0].Context.TimeSinceLastUser = &timeSinceUser
	session.ToolCalls[0].FrameworkMetadata = map[string]any{"tool-meta": true}
	session.ToolCalls[0].SpawnedAgent = &minitrace.SpawnedAgent{AgentType: "codex", TaskScope: "audit", SubSessionID: &subSessionID, OutcomeSummary: "completed audit"}
	classification := "high"
	session.Annotations = []minitrace.Annotation{{
		ID:        "ann-1",
		Timestamp: "2026-06-07T20:00:00Z",
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "tool_call", TargetID: "tool-1"},
		Content:   minitrace.AnnotationContent{Category: "risk", Tags: []string{"file-read"}, Title: "Risk", Detail: "Read requires review"},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: []string{"taxonomy/risk"},
			Mast:      []string{"mast/tool"},
			Toolemu:   []string{"toolemu/read"},
		},
		Classification: &classification,
	}}
	readRatio := 1.0
	timeToFirstAction := 0.5
	idleRatio := 0.25
	sessionCost := 0.01
	modelSwitches := 1
	uniqueModels := 2
	medianResponseTokens := 2
	maxResponseTokens := 2
	session.Metrics = minitrace.Metrics{
		TurnCount:                len(session.Turns),
		ToolCallCount:            len(session.ToolCalls),
		ReadCount:                1,
		DelegateCount:            1,
		ReadRatio:                &readRatio,
		TimeToFirstAction:        &timeToFirstAction,
		IdleRatio:                &idleRatio,
		TotalInputTokens:         &inputTokens,
		TotalOutputTokens:        &outputTokens,
		TotalCacheReadTokens:     &cacheReadTokens,
		TotalCacheCreationTokens: &cacheCreationTokens,
		TotalReasoningTokens:     &reasoningTokens,
		TotalToolTokens:          &toolTokens,
		SessionCost:              &sessionCost,
		SubagentCount:            1,
		SubagentToolCalls:        1,
		ModelSwitches:            &modelSwitches,
		UniqueModels:             &uniqueModels,
		MedianResponseTokens:     &medianResponseTokens,
		MaxResponseTokens:        &maxResponseTokens,
	}
	return session
}
