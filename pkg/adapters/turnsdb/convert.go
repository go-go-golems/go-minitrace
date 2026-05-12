package turnsdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

const (
	AdapterVersion = "go-minitrace-turnsdb-adapter-dev"
	SourceFormat   = "pinocchio-turns-sqlite-v1"
)

var phasePreference = []string{"final", "post_tools", "post_inference", "pre_inference"}

type SnapshotSummary struct {
	ConvID              string
	SessionID           string
	TurnID              string
	TurnCreatedAtMS     int64
	RuntimeKey          string
	InferenceID         string
	TurnMetadata        map[string]any
	TurnData            map[string]any
	Phase               string
	SnapshotCreatedAtMS int64
	BlockCount          int
}

type Block struct {
	ID       string
	Kind     string
	Role     string
	Payload  map[string]any
	Metadata map[string]any
}

type CanonicalTurnSnapshot struct {
	ConvID              string
	SessionID           string
	TurnID              string
	TurnCreatedAtMS     int64
	RuntimeKey          string
	InferenceID         string
	TurnMetadata        map[string]any
	TurnData            map[string]any
	Phase               string
	SnapshotCreatedAtMS int64
	Blocks              []Block
}

func ConvertDB(dbPath string, convID string) ([]*minitrace.Session, error) {
	root, err := expandHome(dbPath)
	if err != nil {
		return nil, err
	}
	db, err := connectReadOnly(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	turns, err := loadCanonicalTurns(db, convID)
	if err != nil {
		return nil, err
	}
	grouped := groupConversationSnapshots(turns)
	convIDs := make([]string, 0, len(grouped))
	for id := range grouped {
		convIDs = append(convIDs, id)
	}
	sort.Strings(convIDs)

	sessions := make([]*minitrace.Session, 0, len(convIDs))
	for _, id := range convIDs {
		session, err := convertConversationSnapshots(grouped[id], root)
		if err != nil {
			return nil, errors.Wrapf(err, "converting turns.db conversation %s", id)
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func connectReadOnly(dbPath string) (*sql.DB, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, errors.Wrap(err, "stat turns.db")
	}
	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "opening turns.db")
	}
	return db, nil
}

func loadCanonicalTurns(db *sql.DB, convID string) ([]CanonicalTurnSnapshot, error) {
	summaries, err := loadSnapshotSummaries(db, convID)
	if err != nil {
		return nil, err
	}
	grouped := map[string][]SnapshotSummary{}
	for _, summary := range summaries {
		key := snapshotKey(summary.ConvID, summary.SessionID, summary.TurnID)
		grouped[key] = append(grouped[key], summary)
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := grouped[keys[i]][0]
		right := grouped[keys[j]][0]
		if left.TurnCreatedAtMS != right.TurnCreatedAtMS {
			return left.TurnCreatedAtMS < right.TurnCreatedAtMS
		}
		if left.SessionID != right.SessionID {
			return left.SessionID < right.SessionID
		}
		return left.TurnID < right.TurnID
	})

	ret := make([]CanonicalTurnSnapshot, 0, len(keys))
	for _, key := range keys {
		summary := chooseCanonicalSummary(grouped[key])
		if summary == nil {
			continue
		}
		blocks, err := loadSnapshotBlocks(db, summary.ConvID, summary.SessionID, summary.TurnID, summary.Phase, summary.SnapshotCreatedAtMS)
		if err != nil {
			return nil, err
		}
		ret = append(ret, CanonicalTurnSnapshot{
			ConvID:              summary.ConvID,
			SessionID:           summary.SessionID,
			TurnID:              summary.TurnID,
			TurnCreatedAtMS:     summary.TurnCreatedAtMS,
			RuntimeKey:          summary.RuntimeKey,
			InferenceID:         summary.InferenceID,
			TurnMetadata:        summary.TurnMetadata,
			TurnData:            summary.TurnData,
			Phase:               summary.Phase,
			SnapshotCreatedAtMS: summary.SnapshotCreatedAtMS,
			Blocks:              blocks,
		})
	}

	return ret, nil
}

func loadSnapshotSummaries(db *sql.DB, convID string) ([]SnapshotSummary, error) {
	query := `
		SELECT
		  t.conv_id,
		  t.session_id,
		  t.turn_id,
		  t.turn_created_at_ms,
		  t.runtime_key,
		  t.inference_id,
		  COALESCE(t.turn_metadata_json, '{}') AS turn_metadata_json,
		  COALESCE(t.turn_data_json, '{}') AS turn_data_json,
		  m.phase,
		  m.snapshot_created_at_ms,
		  COUNT(*) AS block_count
		FROM turns t
		JOIN turn_block_membership m
		  ON m.conv_id = t.conv_id
		 AND m.session_id = t.session_id
		 AND m.turn_id = t.turn_id
	`

	args := []any{}
	if strings.TrimSpace(convID) != "" {
		query += " WHERE t.conv_id = ?"
		args = append(args, convID)
	}
	query += `
		GROUP BY
		  t.conv_id,
		  t.session_id,
		  t.turn_id,
		  t.turn_created_at_ms,
		  t.runtime_key,
		  t.inference_id,
		  t.turn_metadata_json,
		  t.turn_data_json,
		  m.phase,
		  m.snapshot_created_at_ms
		ORDER BY t.turn_created_at_ms ASC, t.session_id ASC, t.turn_id ASC, m.snapshot_created_at_ms ASC
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying turns.db snapshot summaries")
	}
	defer func() { _ = rows.Close() }()

	ret := []SnapshotSummary{}
	for rows.Next() {
		var raw SnapshotSummary
		var metadataJSON, dataJSON string
		if err := rows.Scan(
			&raw.ConvID,
			&raw.SessionID,
			&raw.TurnID,
			&raw.TurnCreatedAtMS,
			&raw.RuntimeKey,
			&raw.InferenceID,
			&metadataJSON,
			&dataJSON,
			&raw.Phase,
			&raw.SnapshotCreatedAtMS,
			&raw.BlockCount,
		); err != nil {
			return nil, errors.Wrap(err, "scanning turns.db snapshot summary")
		}
		raw.TurnMetadata = parseJSONObject(metadataJSON)
		raw.TurnData = parseJSONObject(dataJSON)
		ret = append(ret, raw)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating turns.db snapshot summaries")
	}
	return ret, nil
}

func chooseCanonicalSummary(summaries []SnapshotSummary) *SnapshotSummary {
	if len(summaries) == 0 {
		return nil
	}
	for _, phase := range phasePreference {
		var selected *SnapshotSummary
		for i := range summaries {
			if summaries[i].Phase != phase {
				continue
			}
			if selected == nil || summaries[i].SnapshotCreatedAtMS > selected.SnapshotCreatedAtMS {
				selected = &summaries[i]
			}
		}
		if selected != nil {
			return selected
		}
	}

	selected := &summaries[0]
	for i := range summaries {
		if summaries[i].SnapshotCreatedAtMS > selected.SnapshotCreatedAtMS {
			selected = &summaries[i]
		}
	}
	return selected
}

func loadSnapshotBlocks(db *sql.DB, convID, sessionID, turnID, phase string, snapshotCreatedAtMS int64) ([]Block, error) {
	rows, err := db.Query(`
		SELECT
		  m.ordinal,
		  b.block_id,
		  b.kind,
		  b.role,
		  COALESCE(b.payload_json, '{}') AS payload_json,
		  COALESCE(b.block_metadata_json, '{}') AS block_metadata_json
		FROM turn_block_membership m
		JOIN blocks b
		  ON b.block_id = m.block_id
		 AND b.content_hash = m.content_hash
		WHERE m.conv_id = ?
		  AND m.session_id = ?
		  AND m.turn_id = ?
		  AND m.phase = ?
		  AND m.snapshot_created_at_ms = ?
		ORDER BY m.ordinal ASC
	`, convID, sessionID, turnID, phase, snapshotCreatedAtMS)
	if err != nil {
		return nil, errors.Wrap(err, "querying turns.db snapshot blocks")
	}
	defer func() { _ = rows.Close() }()

	ret := []Block{}
	for rows.Next() {
		var ordinal int
		var blockID, kind, role, payloadJSON, metadataJSON string
		if err := rows.Scan(&ordinal, &blockID, &kind, &role, &payloadJSON, &metadataJSON); err != nil {
			return nil, errors.Wrap(err, "scanning turns.db block")
		}
		ret = append(ret, Block{
			ID:       blockID,
			Kind:     kind,
			Role:     role,
			Payload:  parseJSONObject(payloadJSON),
			Metadata: parseJSONObject(metadataJSON),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating turns.db blocks")
	}
	return ret, nil
}

func groupConversationSnapshots(turns []CanonicalTurnSnapshot) map[string][]CanonicalTurnSnapshot {
	grouped := map[string][]CanonicalTurnSnapshot{}
	for _, turn := range turns {
		grouped[turn.ConvID] = append(grouped[turn.ConvID], turn)
	}
	for convID := range grouped {
		sort.Slice(grouped[convID], func(i, j int) bool {
			left := grouped[convID][i]
			right := grouped[convID][j]
			if left.TurnCreatedAtMS != right.TurnCreatedAtMS {
				return left.TurnCreatedAtMS < right.TurnCreatedAtMS
			}
			if left.SessionID != right.SessionID {
				return left.SessionID < right.SessionID
			}
			return left.TurnID < right.TurnID
		})
	}
	return grouped
}

func convertConversationSnapshots(conversationTurns []CanonicalTurnSnapshot, sourcePath string) (*minitrace.Session, error) {
	if len(conversationTurns) == 0 {
		return nil, errors.New("conversation_turns is empty")
	}

	convID := conversationTurns[0].ConvID
	session := minitrace.BuildSessionSkeleton(convID, "pinocchio", SourceFormat, AdapterVersion)
	session.Provenance.SourcePath = minitracePtr(sourcePath)

	turns := []minitrace.Turn{}
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	allTimestamps := []time.Time{}
	modelsSeen := []string{}
	previousBlocks := []Block{}
	seenSessions := map[string]struct{}{}

	for _, snapshot := range conversationTurns {
		var timestampPtr *string
		if ts, ok := minitrace.SafeFromTimestamp(snapshot.TurnCreatedAtMS, true); ok {
			formatted := minitrace.FormatTimestamp(ts)
			timestampPtr = &formatted
			allTimestamps = append(allTimestamps, ts)
		} else {
			now := minitrace.FormatTimestamp(minitrace.NowUTC())
			timestampPtr = &now
		}
		if snapshot.RuntimeKey != "" {
			modelsSeen = append(modelsSeen, snapshot.RuntimeKey)
		}

		deltaBlocks := lcsDelta(previousBlocks, snapshot.Blocks)
		previousBlocks = snapshot.Blocks

		reasoningPending := []string{}
		startTurnIndex := len(turns)
		if _, ok := seenSessions[snapshot.SessionID]; !ok {
			seenSessions[snapshot.SessionID] = struct{}{}
			if len(seenSessions) > 1 {
				detail := mustJSON(map[string]any{
					"conv_id":    snapshot.ConvID,
					"session_id": snapshot.SessionID,
					"turn_id":    snapshot.TurnID,
				})
				annotations = append(annotations, minitrace.BuildAnnotation(
					fmt.Sprintf("ann-session-change-%d", len(annotations)),
					"adapter",
					"session",
					"session",
					"observation",
					"Conversation resumed in new runtime session",
					detail,
					[]string{"pinocchio", "session_id"},
					nil,
				))
			}
		}

		for _, block := range deltaBlocks {
			text := stringifyBlockPayload(block.Payload)
			if block.Kind == "reasoning" {
				if strings.TrimSpace(text) != "" {
					reasoningPending = append(reasoningPending, text)
				}
				continue
			}

			role, source, inputChannel := classifyTurnBlock(block)
			if role == "" {
				continue
			}

			turn := minitrace.BuildTurn(len(turns), timestampPtr, role, minitracePtr(source), text)
			if inputChannel != "" {
				turn.InputChannel = minitracePtr(inputChannel)
			}
			if snapshot.RuntimeKey != "" {
				turn.Model = &snapshot.RuntimeKey
			}
			turn.FrameworkMetadata = map[string]any{
				"pinocchio_kind": block.Kind,
				"original_role":  block.Role,
				"turn_id":        snapshot.TurnID,
				"phase":          snapshot.Phase,
				"session_id":     snapshot.SessionID,
			}
			if len(reasoningPending) > 0 && role == "assistant" {
				thinking := strings.Join(reasoningPending, "\n\n")
				turn.Thinking = &thinking
				reasoningPending = nil
			}
			turns = append(turns, turn)
		}

		emittingTurnIndex := -1
		if len(turns) > 0 && len(turns)-1 >= startTurnIndex {
			emittingTurnIndex = len(turns) - 1
		}
		deltaToolCalls, deltaAnnotations := buildToolCallsFromBlocks(deltaBlocks, timestampPtr, emittingTurnIndex)
		if emittingTurnIndex >= 0 && emittingTurnIndex < len(turns) {
			for _, toolCall := range deltaToolCalls {
				turns[emittingTurnIndex].ToolCallsInTurn = appendUniqueString(turns[emittingTurnIndex].ToolCallsInTurn, toolCall.ID)
			}
		}
		toolCalls = mergeToolCalls(toolCalls, deltaToolCalls)
		annotations = append(annotations, deltaAnnotations...)

		detail := mustJSON(map[string]any{
			"conv_id":                snapshot.ConvID,
			"session_id":             snapshot.SessionID,
			"turn_id":                snapshot.TurnID,
			"phase":                  snapshot.Phase,
			"snapshot_created_at_ms": snapshot.SnapshotCreatedAtMS,
			"delta_block_count":      len(deltaBlocks),
		})
		annotations = append(annotations, minitrace.BuildAnnotation(
			fmt.Sprintf("ann-snapshot-%d", len(annotations)),
			"adapter",
			"session",
			"session",
			"observation",
			"Canonical snapshot selected",
			detail,
			[]string{"pinocchio", "snapshot", snapshot.Phase},
			nil,
		))
	}

	minitrace.ComputeToolCallContext(toolCalls)
	timing := minitrace.ComputeTiming(allTimestamps)
	quality := minitrace.AssignQualityTier(turns, toolCalls)

	if len(modelsSeen) > 0 {
		model := modelsSeen[len(modelsSeen)-1]
		session.Environment.Model = &model
	}
	provider := providerHintFromRuntime(firstNonEmpty(sessionEnvironmentModel(session), ""))
	session.Environment.ProviderHint = &provider
	session.Environment.PlatformType = minitracePtr("agent")
	session.Environment.ToolsEnabled = uniqueToolNames(toolCalls)
	session.Timing = timing
	session.Turns = turns
	session.ToolCalls = toolCalls
	session.Annotations = annotations
	session.Title = minitrace.ExtractTitle(turns, 80)
	session.Quality = &quality
	session.Metrics = minitrace.ComputeMetrics(turns, toolCalls, timing, 0, nil)
	session.Classification = "internal"
	session.Flags.ForResearch = false
	session.Flags.NeedsCleaning = true
	session.Flags.ContainsPII = false

	return &session, nil
}

func blockFingerprint(block Block) string {
	payload, _ := json.Marshal(block.Payload)
	if block.Kind == "tool_call" || block.Kind == "tool_use" {
		return fmt.Sprintf("%s|%s|%s|%s", block.Kind, block.Role, block.ID, payload)
	}
	metadata, _ := json.Marshal(block.Metadata)
	return fmt.Sprintf("%s|%s|%s|%s|%s", block.Kind, block.Role, block.ID, payload, metadata)
}

func lcsDelta(previousBlocks, currentBlocks []Block) []Block {
	prevFP := make([]string, len(previousBlocks))
	currFP := make([]string, len(currentBlocks))
	for i, block := range previousBlocks {
		prevFP[i] = blockFingerprint(block)
	}
	for i, block := range currentBlocks {
		currFP[i] = blockFingerprint(block)
	}

	m := len(prevFP)
	n := len(currFP)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if prevFP[i] == currFP[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	matchedCurrent := map[int]struct{}{}
	i, j := 0, 0
	for i < m && j < n {
		if prevFP[i] == currFP[j] {
			matchedCurrent[j] = struct{}{}
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			i++
		} else {
			j++
		}
	}

	ret := []Block{}
	for idx, block := range currentBlocks {
		if _, ok := matchedCurrent[idx]; !ok {
			ret = append(ret, block)
		}
	}
	return ret
}

func classifyTurnBlock(block Block) (string, string, string) {
	text := stringifyBlockPayload(block.Payload)

	switch block.Kind {
	case "llm_text":
		return "assistant", "model", ""
	case "system":
		if text == "You are an assistant" {
			return "system", "system", "system_prompt"
		}
		return "system", "framework", "framework_control"
	case "user":
		if isFrameworkControlUserText(text) {
			return "system", "framework", "framework_control"
		}
		return "user", "human", "user_input"
	default:
		return "", "", ""
	}
}

func buildToolCallsFromBlocks(deltaBlocks []Block, timestamp *string, emittingTurnIndex int) ([]minitrace.ToolCall, []minitrace.Annotation) {
	toolCalls := []minitrace.ToolCall{}
	annotations := []minitrace.Annotation{}
	pending := map[string]int{}

	for _, block := range deltaBlocks {
		switch block.Kind {
		case "tool_call":
			toolCallID := firstNonEmpty(stringValue(block.Payload["id"]), fmt.Sprintf("tool-call-%d", len(toolCalls)))
			var turnIndexPtr *int
			if emittingTurnIndex >= 0 {
				turnIndexPtr = &emittingTurnIndex
			}
			toolCall := minitrace.BuildToolCall(
				toolCallID,
				turnIndexPtr,
				timestamp,
				firstNonEmpty(stringValue(block.Payload["name"]), "unknown"),
				"EXECUTE",
				nil,
				nil,
				block.Payload["args"],
				false,
				nil,
				nil,
				nil,
				map[string]any{
					"pinocchio_kind": block.Kind,
					"payload":        block.Payload,
				},
				nil,
				minitracePtr("local_exec"),
				nil,
			)
			pending[toolCallID] = len(toolCalls)
			toolCalls = append(toolCalls, toolCall)
		case "tool_use":
			toolCallID := stringValue(block.Payload["id"])
			if pendingIndex, ok := pending[toolCallID]; ok {
				applyToolResult(&toolCalls[pendingIndex], block.Payload["result"], stringValue(block.Payload["error"]), timestamp)
				delete(pending, toolCallID)
			} else {
				annotations = append(annotations, minitrace.BuildAnnotation(
					fmt.Sprintf("ann-tool-use-orphan-%d", len(annotations)),
					"adapter",
					"session",
					"session",
					"observation",
					"Orphan tool_use block",
					mustJSON(block.Payload),
					[]string{"pinocchio", "tool_use", "orphan"},
					nil,
				))
			}
		}
	}

	for toolCallID, idx := range pending {
		errorText := "no tool result received"
		toolCalls[idx].Output.Success = false
		toolCalls[idx].Output.Error = &errorText
		annotations = append(annotations, minitrace.BuildAnnotation(
			fmt.Sprintf("ann-tool-call-pending-%d", len(annotations)),
			"adapter",
			"tool_call",
			toolCallID,
			"observation",
			"Tool call never received result",
			toolCallID,
			[]string{"pinocchio", "tool_call", "orphan"},
			nil,
		))
	}

	return toolCalls, annotations
}

func applyToolResult(toolCall *minitrace.ToolCall, result any, errorText string, timestamp *string) {
	truncated, fullBytes, fullHash := minitrace.TruncateContent(stringifyAny(result), minitrace.TruncateLimit)
	toolCall.Output.Success = strings.TrimSpace(errorText) == ""
	toolCall.Output.Result = truncated
	toolCall.Output.Truncated = fullBytes != nil
	toolCall.Output.FullBytes = fullBytes
	toolCall.Output.FullHash = fullHash
	toolCall.Output.ContentOrigin = minitracePtr("local_exec")
	toolCall.Timestamp = timestamp
	if strings.TrimSpace(errorText) != "" {
		toolCall.Output.Error = &errorText
	}
}

func providerHintFromRuntime(runtimeKey string) string {
	key := strings.ToLower(strings.TrimSpace(runtimeKey))
	switch {
	case strings.HasPrefix(key, "gpt-"):
		return "openai"
	case strings.HasPrefix(key, "cerebras-"):
		return "cerebras"
	case strings.HasPrefix(key, "together-"):
		return "together"
	default:
		return "unknown"
	}
}

func isFrameworkControlUserText(text string) bool {
	value := strings.TrimSpace(text)
	if value == "" {
		return false
	}
	return strings.HasPrefix(value, "<currentMode>") ||
		strings.Contains(value, "<modeSwitchGuidelines>") ||
		strings.Contains(value, "<pinocchio:agent_mode_switch:v1>")
}

func stringifyBlockPayload(payload map[string]any) string {
	if payload == nil {
		return "{}"
	}
	if text, ok := payload["text"]; ok {
		return stringValue(text)
	}
	return stringifyAny(payload)
}

func stringifyAny(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(payload)
	}
}

func mergeToolCalls(existing []minitrace.ToolCall, incoming []minitrace.ToolCall) []minitrace.ToolCall {
	if len(incoming) == 0 {
		return existing
	}
	indexes := make(map[string]int, len(existing)+len(incoming))
	for i, toolCall := range existing {
		indexes[toolCall.ID] = i
	}
	for _, toolCall := range incoming {
		if idx, ok := indexes[toolCall.ID]; ok {
			existing[idx] = mergeToolCall(existing[idx], toolCall)
			continue
		}
		indexes[toolCall.ID] = len(existing)
		existing = append(existing, toolCall)
	}
	return existing
}

func mergeToolCall(existing, incoming minitrace.ToolCall) minitrace.ToolCall {
	if isPendingToolCall(existing) && !isPendingToolCall(incoming) {
		if incoming.EmittingTurnIndex == nil {
			incoming.EmittingTurnIndex = existing.EmittingTurnIndex
		}
		return incoming
	}
	if !isPendingToolCall(existing) && isPendingToolCall(incoming) {
		return existing
	}
	if existing.Output.Result == nil && incoming.Output.Result != nil {
		existing.Output = incoming.Output
		existing.Timestamp = incoming.Timestamp
	}
	if existing.EmittingTurnIndex == nil {
		existing.EmittingTurnIndex = incoming.EmittingTurnIndex
	}
	return existing
}

func isPendingToolCall(toolCall minitrace.ToolCall) bool {
	return !toolCall.Output.Success && toolCall.Output.Result == nil && toolCall.Output.Error != nil && *toolCall.Output.Error == "no tool result received"
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniqueToolNames(toolCalls []minitrace.ToolCall) []string {
	seen := map[string]struct{}{}
	ret := []string{}
	for _, toolCall := range toolCalls {
		if _, ok := seen[toolCall.ToolName]; ok {
			continue
		}
		seen[toolCall.ToolName] = struct{}{}
		ret = append(ret, toolCall.ToolName)
	}
	sort.Strings(ret)
	return ret
}

func snapshotKey(convID, sessionID, turnID string) string {
	return convID + "\x00" + sessionID + "\x00" + turnID
}

func parseJSONObject(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	ret := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return map[string]any{}
	}
	return ret
}

func mustJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func sessionEnvironmentModel(session minitrace.Session) string {
	if session.Environment.Model == nil {
		return ""
	}
	return *session.Environment.Model
}

func minitracePtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return filepath.Clean(path), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
