package minitracejs

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

type QueryRecipe struct {
	Name        string `json:"name"`
	SQL         string `json:"sql"`
	Args        []any  `json:"args"`
	Description string `json:"description,omitempty"`
	Output      string `json:"output,omitempty"`
}

type QueryRecipeBuilder struct {
	kind         string
	sessionID    string
	includeTools bool
	groupBy      string
}

func NewQueryRecipeBuilder() *QueryRecipeBuilder { return &QueryRecipeBuilder{groupBy: "turn"} }

func queryRecipeBuilderObject(vm *goja.Runtime, b *QueryRecipeBuilder) *goja.Object {
	obj := vm.NewObject()
	setKind := func(kind string) *goja.Object { b.kind = kind; return queryRecipeBuilderObject(vm, b) }
	_ = obj.Set("SessionSummary", func() *goja.Object { return setKind("sessionSummary") })
	_ = obj.Set("TurnRows", func() *goja.Object { return setKind("turnRows") })
	_ = obj.Set("ToolRows", func() *goja.Object { return setKind("toolRows") })
	_ = obj.Set("EventRows", func() *goja.Object { return setKind("eventRows") })
	_ = obj.Set("TurnBlockRows", func() *goja.Object { return setKind("turnBlockRows") })
	_ = obj.Set("TokenUsageRows", func() *goja.Object { return setKind("tokenUsageRows") })
	_ = obj.Set("TranscriptRows", func() *goja.Object { return setKind("transcriptRows") })
	_ = obj.Set("TimelineRows", func() *goja.Object { return setKind("timelineRows") })
	_ = obj.Set("SessionID", func(id string) *goja.Object {
		b.sessionID = strings.TrimSpace(id)
		return queryRecipeBuilderObject(vm, b)
	})
	_ = obj.Set("IncludeTools", func(call goja.FunctionCall) goja.Value {
		b.includeTools = optionalBool(call, true)
		return queryRecipeBuilderObject(vm, b)
	})
	_ = obj.Set("BySession", func() *goja.Object { b.groupBy = "session"; return queryRecipeBuilderObject(vm, b) })
	_ = obj.Set("ByTurn", func() *goja.Object { b.groupBy = "turn"; return queryRecipeBuilderObject(vm, b) })
	_ = obj.Set("ByRole", func() *goja.Object { b.groupBy = "role"; return queryRecipeBuilderObject(vm, b) })
	_ = obj.Set("ByTool", func() *goja.Object { b.groupBy = "tool"; return queryRecipeBuilderObject(vm, b) })
	_ = obj.Set("Build", func() (*goja.Object, error) {
		recipe, err := b.Build()
		if err != nil {
			return nil, err
		}
		return queryRecipeObject(vm, recipe), nil
	})
	return obj
}

func (b *QueryRecipeBuilder) Build() (*QueryRecipe, error) {
	sessionID := b.sessionID
	args := []any{sessionID, sessionID}
	sessionFilter := "(? = '' OR session_id = ?)"
	switch b.kind {
	case "sessionSummary", "":
		return &QueryRecipe{Name: "sessionSummary", SQL: `SELECT session_id, title, summary, agent_framework, model, working_directory, started_at, ended_at, turn_count, tool_call_count FROM sessions WHERE ` + sessionFilter + ` ORDER BY started_at DESC, session_id LIMIT 1`, Args: args, Description: "Session-level summary row.", Output: "SessionSummary"}, nil
	case "turnRows":
		return &QueryRecipe{Name: "turnRows", SQL: `SELECT session_id, turn_index, timestamp, role, source, model, content_type, content, thinking, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, reasoning_tokens, tool_tokens, raw_json FROM turns WHERE ` + sessionFilter + ` ORDER BY turn_index`, Args: args, Description: "Ordered conversational turns.", Output: "TurnRow[]"}, nil
	case "toolRows":
		return &QueryRecipe{Name: "toolRows", SQL: `SELECT session_id, tool_call_id, emitting_turn_index AS turn_index, timestamp, tool_name, operation_type, file_path, command, justification, success, result, error, exit_code, duration_ms, truncated, full_bytes, raw_json FROM tool_calls WHERE ` + sessionFilter + ` ORDER BY COALESCE(emitting_turn_index, 999999), timestamp, tool_call_id`, Args: args, Description: "Ordered tool call rows.", Output: "ToolRow[]"}, nil
	case "eventRows", "turnBlockRows":
		return &QueryRecipe{Name: b.kind, SQL: `SELECT session_id, event_id, turn_index, ordinal, kind, role, tool_call_id, annotation_id, title, summary, text, severity, collapsed_by_default, raw_json FROM events WHERE ` + sessionFilter + ` ORDER BY COALESCE(turn_index, 999999), COALESCE(ordinal, 999999), event_id`, Args: args, Description: "Renderable event/block rows.", Output: "TurnBlockRow[]"}, nil
	case "transcriptRows":
		return transcriptRowsRecipe(sessionID, b.includeTools), nil
	case "timelineRows":
		return timelineRowsRecipe(sessionID), nil
	case "tokenUsageRows":
		return tokenUsageRecipe(sessionID, b.groupBy), nil
	default:
		return nil, fmt.Errorf("unknown query recipe %q", b.kind)
	}
}

func queryRecipeObject(vm *goja.Runtime, recipe *QueryRecipe) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("name", func() string { return recipe.Name })
	_ = obj.Set("sql", func() string { return recipe.SQL })
	_ = obj.Set("args", func() []any { return append([]any(nil), recipe.Args...) })
	_ = obj.Set("description", func() string { return recipe.Description })
	_ = obj.Set("output", func() string { return recipe.Output })
	_ = obj.Set("toJSON", func() map[string]any { return toPlainMap(recipe) })
	return obj
}

func transcriptRowsRecipe(sessionID string, includeTools bool) *QueryRecipe {
	args := []any{sessionID, sessionID, sessionID, sessionID}
	sql := `SELECT 'turn-' || turn_index AS id, session_id, turn_index, 0 AS ordinal, role, 'message' AS kind, model AS name, role || ' message' AS title, content AS text, timestamp,
    COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0) + COALESCE(reasoning_tokens, 0) AS tokens,
    NULL AS tool_call_id, 'info' AS severity, 0 AS collapsed_by_default, raw_json AS metadata
  FROM turns WHERE (? = '' OR session_id = ?) AND COALESCE(content, '') != ''
UNION ALL
SELECT 'turn-' || turn_index || '-thinking' AS id, session_id, turn_index, 1 AS ordinal, role, 'thinking' AS kind, model AS name, 'Thinking' AS title, thinking AS text, timestamp,
    COALESCE(reasoning_tokens, 0) AS tokens, NULL AS tool_call_id, 'info' AS severity, 1 AS collapsed_by_default, raw_json AS metadata
  FROM turns WHERE (? = '' OR session_id = ?) AND COALESCE(thinking, '') != ''`
	if includeTools {
		sql += `
UNION ALL
SELECT 'tool-' || tc.tool_call_id AS id, tc.session_id, COALESCE(ttc.turn_index, tc.emitting_turn_index, 0) AS turn_index, 10 + COALESCE(ttc.ordinal, 0) AS ordinal, 'tool' AS role,
    CASE WHEN COALESCE(tc.success, 1) = 0 THEN 'tool_error' ELSE 'tool_result' END AS kind,
    tc.tool_name AS name, tc.tool_name || ' / ' || COALESCE(tc.operation_type, 'operation') AS title,
    COALESCE(tc.error, tc.result, tc.command, tc.file_path, '') AS text, tc.timestamp,
    CAST(ROUND(COALESCE(tc.full_bytes, LENGTH(COALESCE(tc.result, tc.error, ''))) / 4.0) AS INTEGER) AS tokens,
    tc.tool_call_id, CASE WHEN COALESCE(tc.success, 1) = 0 THEN 'error' ELSE 'info' END AS severity, 1 AS collapsed_by_default, tc.raw_json AS metadata
  FROM tool_calls tc LEFT JOIN turn_tool_calls ttc ON ttc.session_id = tc.session_id AND ttc.tool_call_id = tc.tool_call_id
  WHERE (? = '' OR tc.session_id = ?)`
		args = append(args, sessionID, sessionID)
	}
	sql += ` ORDER BY turn_index, ordinal, id`
	return &QueryRecipe{Name: "transcriptRows", SQL: sql, Args: args, Description: "Transcript rows for messages, thinking, and optional tools.", Output: "TranscriptRow[]"}
}

func timelineRowsRecipe(sessionID string) *QueryRecipe {
	return &QueryRecipe{Name: "timelineRows", SQL: `SELECT t.session_id, t.turn_index, t.role, t.timestamp, SUBSTR(COALESCE(t.content, t.thinking, ''), 1, 600) AS preview,
  COALESCE(t.input_tokens, 0) + COALESCE(t.output_tokens, 0) + COALESCE(t.cache_read_tokens, 0) + COALESCE(t.cache_creation_tokens, 0) + COALESCE(t.reasoning_tokens, 0) + COALESCE(t.tool_tokens, 0) AS total_tokens,
  COUNT(tc.tool_call_id) AS tool_call_count, SUM(CASE WHEN COALESCE(tc.success, 1) = 0 THEN 1 ELSE 0 END) AS failed_tool_count, COUNT(DISTINCT f.path) AS file_count,
  CASE WHEN COALESCE(t.thinking, '') != '' THEN 1 ELSE 0 END AS has_thinking, CASE WHEN COUNT(a.annotation_id) > 0 THEN 1 ELSE 0 END AS has_annotations
FROM turns t
LEFT JOIN turn_tool_calls ttc ON ttc.session_id = t.session_id AND ttc.turn_index = t.turn_index
LEFT JOIN tool_calls tc ON tc.session_id = t.session_id AND tc.tool_call_id = ttc.tool_call_id
LEFT JOIN files f ON f.session_id = t.session_id AND f.tool_call_id = tc.tool_call_id
LEFT JOIN annotations a ON a.session_id = t.session_id AND a.target_id = CAST(t.turn_index AS TEXT)
WHERE (? = '' OR t.session_id = ?)
GROUP BY t.session_id, t.turn_index
ORDER BY t.turn_index`, Args: []any{sessionID, sessionID}, Description: "Condensed timeline rows.", Output: "TimelineRow[]"}
}

func tokenUsageRecipe(sessionID, groupBy string) *QueryRecipe {
	base := `COALESCE(input_tokens, 0) AS input_tokens, COALESCE(output_tokens, 0) AS output_tokens, COALESCE(cache_read_tokens, 0) AS cache_read_tokens, COALESCE(cache_creation_tokens, 0) AS cache_creation_tokens, COALESCE(reasoning_tokens, 0) AS reasoning_tokens, COALESCE(tool_tokens, 0) AS tool_tokens, COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0) + COALESCE(cache_read_tokens, 0) + COALESCE(cache_creation_tokens, 0) + COALESCE(reasoning_tokens, 0) + COALESCE(tool_tokens, 0) AS total_tokens`
	switch groupBy {
	case "session":
		return &QueryRecipe{Name: "tokenUsageRows", SQL: `SELECT 'session' AS scope, session_id, NULL AS turn_index, NULL AS role, NULL AS tool_call_id, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, SUM(cache_read_tokens) AS cache_read_tokens, SUM(cache_creation_tokens) AS cache_creation_tokens, SUM(reasoning_tokens) AS reasoning_tokens, SUM(tool_tokens) AS tool_tokens, SUM(total_tokens) AS total_tokens FROM (SELECT session_id, ` + base + ` FROM turns WHERE (? = '' OR session_id = ?)) GROUP BY session_id ORDER BY session_id`, Args: []any{sessionID, sessionID}, Description: "Session-level token usage.", Output: "TokenUsageRow[]"}
	case "role":
		return &QueryRecipe{Name: "tokenUsageRows", SQL: `SELECT 'role' AS scope, session_id, NULL AS turn_index, role, NULL AS tool_call_id, SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens, SUM(cache_read_tokens) AS cache_read_tokens, SUM(cache_creation_tokens) AS cache_creation_tokens, SUM(reasoning_tokens) AS reasoning_tokens, SUM(tool_tokens) AS tool_tokens, SUM(total_tokens) AS total_tokens FROM (SELECT session_id, role, ` + base + ` FROM turns WHERE (? = '' OR session_id = ?)) GROUP BY session_id, role ORDER BY session_id, total_tokens DESC`, Args: []any{sessionID, sessionID}, Description: "Role-level token usage.", Output: "TokenUsageRow[]"}
	case "tool":
		return &QueryRecipe{Name: "tokenUsageRows", SQL: `SELECT 'tool' AS scope, session_id, emitting_turn_index AS turn_index, 'tool' AS role, tool_call_id, 0 AS input_tokens, 0 AS output_tokens, 0 AS cache_read_tokens, 0 AS cache_creation_tokens, 0 AS reasoning_tokens, 0 AS tool_tokens, CAST(ROUND(COALESCE(full_bytes, LENGTH(COALESCE(result, error, ''))) / 4.0) AS INTEGER) AS total_tokens FROM tool_calls WHERE (? = '' OR session_id = ?) ORDER BY session_id, turn_index, tool_call_id`, Args: []any{sessionID, sessionID}, Description: "Estimated tool-output token usage.", Output: "TokenUsageRow[]"}
	default:
		return &QueryRecipe{Name: "tokenUsageRows", SQL: `SELECT 'turn' AS scope, session_id, turn_index, role, NULL AS tool_call_id, ` + base + ` FROM turns WHERE (? = '' OR session_id = ?) ORDER BY session_id, turn_index`, Args: []any{sessionID, sessionID}, Description: "Turn-level token usage.", Output: "TokenUsageRow[]"}
	}
}

type ViewPlanBuilder struct {
	db                 *DBHandle
	sessionID          string
	kind               string
	includeTools       bool
	includeThinking    bool
	includeToolResults bool
	collapseLongTextAt int
	groupBy            string
}

func NewViewPlanBuilder() *ViewPlanBuilder {
	return &ViewPlanBuilder{includeTools: true, includeThinking: true, includeToolResults: true, groupBy: "turn"}
}

func viewPlanBuilderObject(vm *goja.Runtime, b *ViewPlanBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("DB", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			if h, ok := call.Argument(0).ToObject(vm).Get("_handle").Export().(*DBHandle); ok {
				b.db = h
			}
		}
		return viewPlanBuilderObject(vm, b)
	})
	_ = obj.Set("SessionID", func(id string) *goja.Object { b.sessionID = strings.TrimSpace(id); return viewPlanBuilderObject(vm, b) })
	setKind := func(kind string) *goja.Object { b.kind = kind; return viewPlanBuilderObject(vm, b) }
	_ = obj.Set("Transcript", func() *goja.Object { return setKind("transcript") })
	_ = obj.Set("TurnFrames", func() *goja.Object { return setKind("turnFrames") })
	_ = obj.Set("Timeline", func() *goja.Object { return setKind("timeline") })
	_ = obj.Set("TokenUsage", func() *goja.Object { return setKind("tokenUsage") })
	_ = obj.Set("SessionSummary", func() *goja.Object { return setKind("sessionSummary") })
	_ = obj.Set("IncludeTools", func(call goja.FunctionCall) goja.Value {
		b.includeTools = optionalBool(call, true)
		return viewPlanBuilderObject(vm, b)
	})
	_ = obj.Set("IncludeThinking", func(call goja.FunctionCall) goja.Value {
		b.includeThinking = optionalBool(call, true)
		return viewPlanBuilderObject(vm, b)
	})
	_ = obj.Set("IncludeToolResults", func(call goja.FunctionCall) goja.Value {
		b.includeToolResults = optionalBool(call, true)
		return viewPlanBuilderObject(vm, b)
	})
	_ = obj.Set("CollapseLongTextAt", func(chars int) *goja.Object { b.collapseLongTextAt = chars; return viewPlanBuilderObject(vm, b) })
	_ = obj.Set("BySession", func() *goja.Object { b.groupBy = "session"; return viewPlanBuilderObject(vm, b) })
	_ = obj.Set("ByTurn", func() *goja.Object { b.groupBy = "turn"; return viewPlanBuilderObject(vm, b) })
	_ = obj.Set("ByRole", func() *goja.Object { b.groupBy = "role"; return viewPlanBuilderObject(vm, b) })
	_ = obj.Set("ByTool", func() *goja.Object { b.groupBy = "tool"; return viewPlanBuilderObject(vm, b) })
	_ = obj.Set("Plan", func() (*goja.Object, error) {
		recipe, err := b.recipe()
		if err != nil {
			return nil, err
		}
		return queryRecipeObject(vm, recipe), nil
	})
	_ = obj.Set("Run", func() (any, error) { return b.Run() })
	return obj
}

func (b *ViewPlanBuilder) recipe() (*QueryRecipe, error) {
	qb := &QueryRecipeBuilder{sessionID: b.sessionID, includeTools: b.includeTools, groupBy: b.groupBy}
	switch b.kind {
	case "", "sessionSummary":
		qb.kind = "sessionSummary"
	case "transcript":
		qb.kind = "transcriptRows"
	case "timeline":
		qb.kind = "timelineRows"
	case "tokenUsage":
		qb.kind = "tokenUsageRows"
	case "turnFrames":
		qb.kind = "turnBlockRows"
	default:
		return nil, fmt.Errorf("unknown view kind %q", b.kind)
	}
	return qb.Build()
}

func (b *ViewPlanBuilder) Run() (any, error) {
	if b.db == nil {
		return nil, fmt.Errorf("view requires DB(...) or session-bound db")
	}
	if b.kind == "turnFrames" {
		return b.runTurnFrames()
	}
	recipe, err := b.recipe()
	if err != nil {
		return nil, err
	}
	if b.kind == "" || b.kind == "sessionSummary" {
		return b.db.runner.QueryOne(context.Background(), recipe.SQL, recipe.Args...)
	}
	return b.db.runner.Query(context.Background(), recipe.SQL, recipe.Args...)
}

func (b *ViewPlanBuilder) runTurnFrames() ([]map[string]any, error) {
	blocksRecipe, err := (&QueryRecipeBuilder{kind: "turnBlockRows", sessionID: b.sessionID}).Build()
	if err != nil {
		return nil, err
	}
	toolRecipe, err := (&QueryRecipeBuilder{kind: "toolRows", sessionID: b.sessionID}).Build()
	if err != nil {
		return nil, err
	}
	blocks, err := b.db.runner.Query(context.Background(), blocksRecipe.SQL, blocksRecipe.Args...)
	if err != nil {
		return nil, err
	}
	tools, err := b.db.runner.Query(context.Background(), toolRecipe.SQL, toolRecipe.Args...)
	if err != nil {
		return nil, err
	}
	frames := map[int]map[string]any{}
	for _, block := range blocks {
		turnIndex := intFromAny(block["turn_index"])
		frame := frameFor(frames, turnIndex)
		frame["blocks"] = append(frame["blocks"].([]map[string]any), block)
	}
	for _, tool := range tools {
		turnIndex := intFromAny(tool["turn_index"])
		frame := frameFor(frames, turnIndex)
		frame["toolCalls"] = append(frame["toolCalls"].([]map[string]any), tool)
		stats := frame["stats"].(map[string]any)
		stats["toolCalls"] = intFromAny(stats["toolCalls"]) + 1
		if intFromAny(tool["success"]) == 0 {
			stats["failedToolCalls"] = intFromAny(stats["failedToolCalls"]) + 1
		}
	}
	out := make([]map[string]any, 0, len(frames))
	for i := 0; ; i++ {
		frame, ok := frames[i]
		if !ok {
			if len(out) == len(frames) {
				break
			}
			continue
		}
		out = append(out, frame)
	}
	return out, nil
}

func frameFor(frames map[int]map[string]any, turnIndex int) map[string]any {
	frame := frames[turnIndex]
	if frame == nil {
		frame = map[string]any{"turnIndex": turnIndex, "blocks": []map[string]any{}, "toolCalls": []map[string]any{}, "stats": map[string]any{"chars": 0, "toolCalls": 0, "failedToolCalls": 0, "estimatedTokens": 0}}
		frames[turnIndex] = frame
	}
	return frame
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case nil:
		return 0
	default:
		return 0
	}
}

type SessionBuilder struct {
	ctx       context.Context
	runtime   RuntimeSettings
	sources   *SourceSetBuilder
	policy    *ImportPolicyBuilder
	cache     *CachePolicyBuilder
	limits    *QueryLimitsBuilder
	sessionID string
}

type SessionHandle struct {
	db        *DBHandle
	sessionID string
	summary   map[string]any
}

func NewSessionBuilder(ctx context.Context, runtime RuntimeSettings) *SessionBuilder {
	return &SessionBuilder{ctx: ctx, runtime: runtime, sources: NewSourceSetBuilder(), policy: NewImportPolicyBuilder(), cache: NewCachePolicyBuilder(), limits: NewQueryLimitsBuilder()}
}

func sessionBuilderObject(vm *goja.Runtime, b *SessionBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("Sources", func(sources *SourceSet) *goja.Object {
		b.sources = &SourceSetBuilder{sources: append([]dbSource(nil), sources.sources...), last: len(sources.sources) - 1}
		return sessionBuilderObject(vm, b)
	})
	_ = obj.Set("Source", func(sources *SourceSet) *goja.Object {
		b.sources = &SourceSetBuilder{sources: append([]dbSource(nil), sources.sources...), last: len(sources.sources) - 1}
		return sessionBuilderObject(vm, b)
	})
	_ = obj.Set("Import", func(policy *ImportPolicy) *goja.Object { b.policy.policy = *policy; return sessionBuilderObject(vm, b) })
	_ = obj.Set("Cache", func(policy *CachePolicy) *goja.Object { b.cache.policy = *policy; return sessionBuilderObject(vm, b) })
	_ = obj.Set("Limits", func(limits *QueryLimits) *goja.Object { b.limits.limits = *limits; return sessionBuilderObject(vm, b) })
	_ = obj.Set("SessionID", func(id string) *goja.Object { b.sessionID = strings.TrimSpace(id); return sessionBuilderObject(vm, b) })
	_ = obj.Set("File", func(path string) *goja.Object { b.sources.AddFile(path); return sessionBuilderObject(vm, b) })
	_ = obj.Set("Content", func(content string) *goja.Object {
		b.sources.AddContent(content, "content")
		return sessionBuilderObject(vm, b)
	})
	_ = obj.Set("Name", func(name string) *goja.Object { b.sources.NameMostRecent(name); return sessionBuilderObject(vm, b) })
	_ = obj.Set("InteractiveCache", func(call goja.FunctionCall) goja.Value {
		b.cache.policy.Mode = "auto"
		b.cache.policy.Dir = optionalString(call)
		return sessionBuilderObject(vm, b)
	})
	_ = obj.Set("Strict", func(call goja.FunctionCall) goja.Value {
		b.policy.policy.Strict = optionalBool(call, true)
		return sessionBuilderObject(vm, b)
	})
	_ = obj.Set("Open", func() (*goja.Object, error) {
		handle, err := b.Open()
		if err != nil {
			return nil, err
		}
		return sessionHandleObject(vm, handle), nil
	})
	return obj
}

func (b *SessionBuilder) Open() (*SessionHandle, error) {
	sources, err := b.sources.Build()
	if err != nil {
		return nil, err
	}
	importPolicy := b.policy.policy
	cachePolicy := b.cache.policy
	limits := b.limits.limits
	db, err := NewDBBuilderWithRuntime(b.ctx, b.runtime).withSourceSet(sources).withImportPolicy(&importPolicy).withCachePolicy(&cachePolicy).withQueryLimits(&limits).Build()
	if err != nil {
		return nil, err
	}
	view := &ViewPlanBuilder{db: db, sessionID: b.sessionID, kind: "sessionSummary", includeTools: true, groupBy: "turn"}
	summaryAny, err := view.Run()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	summary, _ := summaryAny.(map[string]any)
	sessionID := b.sessionID
	if sessionID == "" && summary != nil {
		sessionID, _ = summary["session_id"].(string)
	}
	return &SessionHandle{db: db, sessionID: sessionID, summary: summary}, nil
}

func sessionHandleObject(vm *goja.Runtime, h *SessionHandle) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("id", func() string { return h.sessionID })
	_ = obj.Set("summary", func() map[string]any { return h.summary })
	_ = obj.Set("diagnostics", func() []map[string]any { return toPlainSlice(h.db.diagnostics) })
	_ = obj.Set("cacheInfo", func() map[string]any { return toPlainMap(h.db.CacheInfo()) })
	_ = obj.Set("db", func() *goja.Object { return handleObject(vm, h.db) })
	_ = obj.Set("query", func(sqlText string, args ...any) ([]map[string]any, error) {
		return h.db.runner.Query(context.Background(), sqlText, args...)
	})
	_ = obj.Set("view", func() *goja.Object {
		return viewPlanBuilderObject(vm, &ViewPlanBuilder{db: h.db, sessionID: h.sessionID, includeTools: true, includeThinking: true, includeToolResults: true, groupBy: "turn"})
	})
	_ = obj.Set("close", func() error { return h.db.Close() })
	return obj
}

func (b *DBBuilder) withSourceSet(sources *SourceSet) *DBBuilder { b.applySourceSet(sources); return b }
func (b *DBBuilder) withImportPolicy(policy *ImportPolicy) *DBBuilder {
	b.applyImportPolicy(policy)
	return b
}
func (b *DBBuilder) withCachePolicy(policy *CachePolicy) *DBBuilder {
	b.applyCachePolicy(policy)
	return b
}
func (b *DBBuilder) withQueryLimits(limits *QueryLimits) *DBBuilder {
	b.applyQueryLimits(limits)
	return b
}
