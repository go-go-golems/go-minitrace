package minitracedb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const SchemaVersion = "normalized-sqlite-v5"

// SessionsBaseCompatView is a compatibility VIEW that reconstructs the legacy
// DuckDB `sessions_base` shape (5 scalar columns plus JSON blob columns) from
// the normalized sessions table, so session-level saved SQL written for the
// old engine keeps working on SQLite >= 3.38 (`->`/`->>` on JSON text).
const SessionsBaseCompatView = "sessions_base"

type ColumnDescriptor struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	PrimaryKey  bool   `json:"primaryKey,omitempty"`
	Description string `json:"description,omitempty"`
}

type TableDescriptor struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Columns     []ColumnDescriptor `json:"columns"`
	CreateSQL   string             `json:"createSql"`
}

type SchemaDescriptor struct {
	Version string            `json:"version"`
	Dialect string            `json:"dialect"`
	Tables  []TableDescriptor `json:"tables"`
}

func Schema() SchemaDescriptor {
	return SchemaDescriptor{Version: SchemaVersion, Dialect: "sqlite", Tables: Tables()}
}

func Tables() []TableDescriptor {
	return []TableDescriptor{
		sessionsTable(),
		turnsTable(),
		toolCallsTable(),
		turnToolCallsTable(),
		filesTable(),
		annotationsTable(),
		handoversTable(),
		metricsTable(),
		attachmentsTable(),
		eventsTable(),
	}
}

func AllowedTableNames() []string {
	tables := Tables()
	ret := make([]string, 0, len(tables))
	for _, table := range tables {
		ret = append(ret, table.Name)
	}
	return ret
}

// AllowedObjectNames returns every main-schema table and compatibility view a
// sandboxed query is allowed to read. Bare names are interpreted by
// NewQueryRunner as main.<name>.
func AllowedObjectNames() []string {
	return append(AllowedTableNames(), SessionsBaseCompatView)
}

// AnnotationsAttachSchema is the schema name under which serve ATTACHes the
// live annotation store (annotations.db) next to the normalized database, so
// sandboxed SQL can read live annotations as anno.annotations.
const AnnotationsAttachSchema = "anno"

// AllowedObjectNamesWithLiveAnnotations extends the sandbox allowlist with the
// exact tables of the attached live annotation store. Attached objects must be
// schema-qualified so a crafted annotations.db cannot expose a colliding table
// name such as anno.sessions through the main-schema allowlist.
func AllowedObjectNamesWithLiveAnnotations() []string {
	return append(AllowedObjectNames(), AnnotationsAttachSchema+".annotations", AnnotationsAttachSchema+".sync_state")
}

func CreateSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for _, table := range Tables() {
		if _, err := db.ExecContext(ctx, table.CreateSQL); err != nil {
			return fmt.Errorf("create table %s: %w", table.Name, err)
		}
	}
	for _, stmt := range indexStatements() {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, sessionsBaseCompatViewSQL); err != nil {
		return fmt.Errorf("create view %s: %w", SessionsBaseCompatView, err)
	}
	return nil
}

const sessionsBaseCompatViewSQL = `CREATE VIEW IF NOT EXISTS sessions_base AS
SELECT
	session_id AS id,
	title,
	summary,
	classification,
	profile,
	json_extract(raw_json, '$.provenance') AS provenance,
	json_extract(raw_json, '$.flags') AS flags,
	json_extract(raw_json, '$.environment') AS environment,
	json_extract(raw_json, '$.operational_context') AS operational_context,
	json_extract(raw_json, '$.timing') AS timing,
	json_extract(raw_json, '$.turns') AS turns,
	json_extract(raw_json, '$.tool_calls') AS tool_calls,
	json_extract(raw_json, '$.annotations') AS annotations,
	json_extract(raw_json, '$.metrics') AS metrics
FROM sessions;`

func sessionsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "sessions",
		Description: "One row per minitrace session with top-level provenance, environment, timing, operational context, coordination, and outcome fields.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true, Description: "Stable minitrace session id."},
			{Name: "schema_version", Type: "TEXT", Nullable: true},
			{Name: "profile", Type: "TEXT", Nullable: true},
			{Name: "scenario_id", Type: "TEXT", Nullable: true},
			{Name: "quality", Type: "TEXT", Nullable: true},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "summary", Type: "TEXT", Nullable: true},
			{Name: "classification", Type: "TEXT", Nullable: true},
			{Name: "source_format", Type: "TEXT", Nullable: true},
			{Name: "source_path", Type: "TEXT", Nullable: true},
			{Name: "converted_at", Type: "TEXT", Nullable: true},
			{Name: "converter_version", Type: "TEXT", Nullable: true},
			{Name: "original_session_id", Type: "TEXT", Nullable: true},
			{Name: "for_research", Type: "INTEGER", Nullable: true},
			{Name: "needs_cleaning", Type: "INTEGER", Nullable: true},
			{Name: "contains_error", Type: "INTEGER", Nullable: true},
			{Name: "contains_pii", Type: "INTEGER", Nullable: true},
			{Name: "category_json", Type: "TEXT", Nullable: true},
			{Name: "model", Type: "TEXT", Nullable: true},
			{Name: "model_version", Type: "TEXT", Nullable: true},
			{Name: "temperature", Type: "REAL", Nullable: true},
			{Name: "tools_enabled_json", Type: "TEXT", Nullable: true},
			{Name: "system_prompt", Type: "TEXT", Nullable: true},
			{Name: "agent_framework", Type: "TEXT", Nullable: true},
			{Name: "agent_version", Type: "TEXT", Nullable: true},
			{Name: "platform_type", Type: "TEXT", Nullable: true},
			{Name: "provider_hint", Type: "TEXT", Nullable: true},
			{Name: "working_directory", Type: "TEXT", Nullable: true},
			{Name: "git_branch", Type: "TEXT", Nullable: true},
			{Name: "git_ref", Type: "TEXT", Nullable: true},
			{Name: "autonomy_level", Type: "TEXT", Nullable: true},
			{Name: "sandbox", Type: "INTEGER", Nullable: true},
			{Name: "framework_config_json", Type: "TEXT", Nullable: true},
			{Name: "privacy_level", Type: "TEXT", Nullable: true},
			{Name: "duration_seconds", Type: "REAL", Nullable: true},
			{Name: "active_duration_seconds", Type: "REAL", Nullable: true},
			{Name: "started_at", Type: "TEXT", Nullable: true},
			{Name: "ended_at", Type: "TEXT", Nullable: true},
			{Name: "hour_of_day", Type: "INTEGER", Nullable: true},
			{Name: "day_of_week", Type: "INTEGER", Nullable: true},
			{Name: "guidance_variant", Type: "TEXT", Nullable: true},
			{Name: "permission_level", Type: "TEXT", Nullable: true},
			{Name: "condition_custom_json", Type: "TEXT", Nullable: true},
			{Name: "project_id", Type: "TEXT", Nullable: true},
			{Name: "predecessor_session", Type: "TEXT", Nullable: true},
			{Name: "concurrent_sessions", Type: "INTEGER", Nullable: true},
			{Name: "human_attention", Type: "TEXT", Nullable: true},
			{Name: "outcome_success", Type: "INTEGER", Nullable: true},
			{Name: "outcome_partial", Type: "INTEGER", Nullable: true},
			{Name: "failure_codes_json", Type: "TEXT", Nullable: true},
			{Name: "outcome_notes", Type: "TEXT", Nullable: true},
			{Name: "turn_count", Type: "INTEGER", Nullable: true},
			{Name: "tool_call_count", Type: "INTEGER", Nullable: true},
			{Name: "read_count", Type: "INTEGER", Nullable: true},
			{Name: "modify_count", Type: "INTEGER", Nullable: true},
			{Name: "create_count", Type: "INTEGER", Nullable: true},
			{Name: "execute_count", Type: "INTEGER", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS sessions (
	session_id TEXT PRIMARY KEY,
	schema_version TEXT,
	profile TEXT,
	scenario_id TEXT,
	quality TEXT,
	title TEXT,
	summary TEXT,
	classification TEXT,
	source_format TEXT,
	source_path TEXT,
	converted_at TEXT,
	converter_version TEXT,
	original_session_id TEXT,
	for_research INTEGER,
	needs_cleaning INTEGER,
	contains_error INTEGER,
	contains_pii INTEGER,
	category_json TEXT,
	model TEXT,
	model_version TEXT,
	temperature REAL,
	tools_enabled_json TEXT,
	system_prompt TEXT,
	agent_framework TEXT,
	agent_version TEXT,
	platform_type TEXT,
	provider_hint TEXT,
	working_directory TEXT,
	git_branch TEXT,
	git_ref TEXT,
	autonomy_level TEXT,
	sandbox INTEGER,
	framework_config_json TEXT,
	privacy_level TEXT,
	duration_seconds REAL,
	active_duration_seconds REAL,
	started_at TEXT,
	ended_at TEXT,
	hour_of_day INTEGER,
	day_of_week INTEGER,
	guidance_variant TEXT,
	permission_level TEXT,
	condition_custom_json TEXT,
	project_id TEXT,
	predecessor_session TEXT,
	concurrent_sessions INTEGER,
	human_attention TEXT,
	outcome_success INTEGER,
	outcome_partial INTEGER,
	failure_codes_json TEXT,
	outcome_notes TEXT,
	turn_count INTEGER,
	tool_call_count INTEGER,
	read_count INTEGER,
	modify_count INTEGER,
	create_count INTEGER,
	execute_count INTEGER,
	raw_json TEXT
);`,
	}
}

func turnsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "turns",
		Description: "One row per conversational turn with role/content, streaming, intent, framework metadata, and usage token fields.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "turn_index", Type: "INTEGER", PrimaryKey: true},
			{Name: "timestamp", Type: "TEXT", Nullable: true},
			{Name: "role", Type: "TEXT", Nullable: true},
			{Name: "source", Type: "TEXT", Nullable: true},
			{Name: "model", Type: "TEXT", Nullable: true},
			{Name: "content_type", Type: "TEXT", Nullable: true},
			{Name: "input_channel", Type: "TEXT", Nullable: true},
			{Name: "content", Type: "TEXT", Nullable: true},
			{Name: "framework_metadata_json", Type: "TEXT", Nullable: true},
			{Name: "thinking", Type: "TEXT", Nullable: true},
			{Name: "intent_requested", Type: "INTEGER", Nullable: true},
			{Name: "intent_inferred", Type: "INTEGER", Nullable: true},
			{Name: "intent_proactive", Type: "INTEGER", Nullable: true},
			{Name: "was_streamed", Type: "INTEGER", Nullable: true},
			{Name: "stream_log", Type: "TEXT", Nullable: true},
			{Name: "input_tokens", Type: "INTEGER", Nullable: true},
			{Name: "output_tokens", Type: "INTEGER", Nullable: true},
			{Name: "cache_read_tokens", Type: "INTEGER", Nullable: true},
			{Name: "cache_creation_tokens", Type: "INTEGER", Nullable: true},
			{Name: "reasoning_tokens", Type: "INTEGER", Nullable: true},
			{Name: "tool_tokens", Type: "INTEGER", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS turns (
	session_id TEXT NOT NULL,
	turn_index INTEGER NOT NULL,
	timestamp TEXT,
	role TEXT,
	source TEXT,
	model TEXT,
	content_type TEXT,
	input_channel TEXT,
	content TEXT,
	framework_metadata_json TEXT,
	thinking TEXT,
	intent_requested INTEGER,
	intent_inferred INTEGER,
	intent_proactive INTEGER,
	was_streamed INTEGER,
	stream_log TEXT,
	input_tokens INTEGER,
	output_tokens INTEGER,
	cache_read_tokens INTEGER,
	cache_creation_tokens INTEGER,
	reasoning_tokens INTEGER,
	tool_tokens INTEGER,
	raw_json TEXT,
	PRIMARY KEY (session_id, turn_index)
);`,
	}
}

func toolCallsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "tool_calls",
		Description: "One row per tool call with input/output fields, context, framework metadata, and spawned-agent details.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "tool_call_id", Type: "TEXT", PrimaryKey: true},
			{Name: "emitting_turn_index", Type: "INTEGER", Nullable: true},
			{Name: "timestamp", Type: "TEXT", Nullable: true},
			{Name: "tool_name", Type: "TEXT", Nullable: true},
			{Name: "operation_type", Type: "TEXT", Nullable: true},
			{Name: "file_path", Type: "TEXT", Nullable: true},
			{Name: "record_kind", Type: "TEXT"},
			{Name: "command", Type: "TEXT", Nullable: true},
			{Name: "justification", Type: "TEXT", Nullable: true},
			{Name: "arguments_json", Type: "TEXT", Nullable: true},
			{Name: "success", Type: "INTEGER", Nullable: true},
			{Name: "outcome_status", Type: "TEXT", Nullable: true},
			{Name: "result", Type: "TEXT", Nullable: true},
			{Name: "error", Type: "TEXT", Nullable: true},
			{Name: "exit_code", Type: "INTEGER", Nullable: true},
			{Name: "duration_ms", Type: "INTEGER", Nullable: true},
			{Name: "truncated", Type: "INTEGER", Nullable: true},
			{Name: "full_bytes", Type: "INTEGER", Nullable: true},
			{Name: "full_hash", Type: "TEXT", Nullable: true},
			{Name: "full_reference", Type: "TEXT", Nullable: true},
			{Name: "redacted", Type: "INTEGER", Nullable: true},
			{Name: "content_origin", Type: "TEXT", Nullable: true},
			{Name: "position_in_session", Type: "REAL", Nullable: true},
			{Name: "tools_before_json", Type: "TEXT", Nullable: true},
			{Name: "time_since_last_user", Type: "REAL", Nullable: true},
			{Name: "framework_metadata_json", Type: "TEXT", Nullable: true},
			{Name: "spawned_agent_type", Type: "TEXT", Nullable: true},
			{Name: "spawned_agent_task_scope", Type: "TEXT", Nullable: true},
			{Name: "spawned_agent_sub_session_id", Type: "TEXT", Nullable: true},
			{Name: "spawned_agent_outcome_summary", Type: "TEXT", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS tool_calls (
	session_id TEXT NOT NULL,
	tool_call_id TEXT NOT NULL,
	emitting_turn_index INTEGER,
	timestamp TEXT,
	tool_name TEXT,
	operation_type TEXT,
	file_path TEXT,
	record_kind TEXT NOT NULL,
	command TEXT,
	justification TEXT,
	arguments_json TEXT,
	success INTEGER,
	outcome_status TEXT,
	result TEXT,
	error TEXT,
	exit_code INTEGER,
	duration_ms INTEGER,
	truncated INTEGER,
	full_bytes INTEGER,
	full_hash TEXT,
	full_reference TEXT,
	redacted INTEGER,
	content_origin TEXT,
	position_in_session REAL,
	tools_before_json TEXT,
	time_since_last_user REAL,
	framework_metadata_json TEXT,
	spawned_agent_type TEXT,
	spawned_agent_task_scope TEXT,
	spawned_agent_sub_session_id TEXT,
	spawned_agent_outcome_summary TEXT,
	raw_json TEXT,
	PRIMARY KEY (session_id, tool_call_id)
);`,
	}
}

func turnToolCallsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "turn_tool_calls",
		Description: "Join table preserving tool call membership and ordinal per turn.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "turn_index", Type: "INTEGER", PrimaryKey: true},
			{Name: "tool_call_id", Type: "TEXT", PrimaryKey: true},
			{Name: "ordinal", Type: "INTEGER", PrimaryKey: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS turn_tool_calls (
	session_id TEXT NOT NULL,
	turn_index INTEGER NOT NULL,
	tool_call_id TEXT NOT NULL,
	ordinal INTEGER NOT NULL,
	PRIMARY KEY (session_id, turn_index, ordinal)
);`,
	}
}

func filesTable() TableDescriptor {
	return TableDescriptor{
		Name:        "files",
		Description: "One row per structural file target or legacy scalar report; evidence_status distinguishes attempts from confirmed effects.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT"},
			{Name: "tool_call_id", Type: "TEXT"},
			{Name: "path", Type: "TEXT"},
			{Name: "operation_type", Type: "TEXT", Nullable: true},
			{Name: "tool_name", Type: "TEXT", Nullable: true},
			{Name: "success", Type: "INTEGER", Nullable: true},
			{Name: "turn_index", Type: "INTEGER", Nullable: true},
			{Name: "target_ordinal", Type: "INTEGER"},
			{Name: "evidence_kind", Type: "TEXT"},
			{Name: "evidence_status", Type: "TEXT"},
			{Name: "cwd", Type: "TEXT", Nullable: true},
			{Name: "resolved", Type: "INTEGER"},
			{Name: "native_path", Type: "TEXT", Nullable: true},
			{Name: "source_reference", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS files (
	session_id TEXT NOT NULL,
	tool_call_id TEXT NOT NULL,
	path TEXT NOT NULL,
	operation_type TEXT,
	tool_name TEXT,
	success INTEGER,
	turn_index INTEGER,
	target_ordinal INTEGER NOT NULL,
	evidence_kind TEXT NOT NULL,
	evidence_status TEXT NOT NULL,
	cwd TEXT,
	resolved INTEGER NOT NULL,
	native_path TEXT,
	source_reference TEXT
);`,
	}
}

func annotationsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "annotations",
		Description: "One row per annotation, including scope, content, taxonomy mappings, classification, and raw JSON.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "annotation_id", Type: "TEXT", PrimaryKey: true},
			{Name: "timestamp", Type: "TEXT", Nullable: true},
			{Name: "annotator", Type: "TEXT", Nullable: true},
			{Name: "scope_type", Type: "TEXT", Nullable: true},
			{Name: "target_id", Type: "TEXT", Nullable: true},
			{Name: "category", Type: "TEXT", Nullable: true},
			{Name: "tags_json", Type: "TEXT", Nullable: true},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "detail", Type: "TEXT", Nullable: true},
			{Name: "minitrace_taxonomy_json", Type: "TEXT", Nullable: true},
			{Name: "mast_taxonomy_json", Type: "TEXT", Nullable: true},
			{Name: "toolemu_taxonomy_json", Type: "TEXT", Nullable: true},
			{Name: "classification", Type: "TEXT", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS annotations (
	session_id TEXT NOT NULL,
	annotation_id TEXT NOT NULL,
	timestamp TEXT,
	annotator TEXT,
	scope_type TEXT,
	target_id TEXT,
	category TEXT,
	tags_json TEXT,
	title TEXT,
	detail TEXT,
	minitrace_taxonomy_json TEXT,
	mast_taxonomy_json TEXT,
	toolemu_taxonomy_json TEXT,
	classification TEXT,
	raw_json TEXT,
	PRIMARY KEY (session_id, annotation_id)
);`,
	}
}

func handoversTable() TableDescriptor {
	return TableDescriptor{
		Name:        "handovers",
		Description: "One row for each received or produced handover document attached to a session.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "direction", Type: "TEXT", PrimaryKey: true},
			{Name: "from_session", Type: "TEXT", Nullable: true},
			{Name: "to_session", Type: "TEXT", Nullable: true},
			{Name: "document", Type: "TEXT", Nullable: true},
			{Name: "state_description", Type: "TEXT", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS handovers (
	session_id TEXT NOT NULL,
	direction TEXT NOT NULL,
	from_session TEXT,
	to_session TEXT,
	document TEXT,
	state_description TEXT,
	raw_json TEXT,
	PRIMARY KEY (session_id, direction)
);`,
	}
}

func metricsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "metrics",
		Description: "Wide metrics table with one row per session, including token, timing, subagent, and model-switch metrics.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "turn_count", Type: "INTEGER", Nullable: true},
			{Name: "tool_call_count", Type: "INTEGER", Nullable: true},
			{Name: "read_count", Type: "INTEGER", Nullable: true},
			{Name: "modify_count", Type: "INTEGER", Nullable: true},
			{Name: "create_count", Type: "INTEGER", Nullable: true},
			{Name: "execute_count", Type: "INTEGER", Nullable: true},
			{Name: "delegate_count", Type: "INTEGER", Nullable: true},
			{Name: "read_ratio", Type: "REAL", Nullable: true},
			{Name: "time_to_first_action", Type: "REAL", Nullable: true},
			{Name: "idle_ratio", Type: "REAL", Nullable: true},
			{Name: "total_input_tokens", Type: "INTEGER", Nullable: true},
			{Name: "total_output_tokens", Type: "INTEGER", Nullable: true},
			{Name: "total_cache_read_tokens", Type: "INTEGER", Nullable: true},
			{Name: "total_cache_creation_tokens", Type: "INTEGER", Nullable: true},
			{Name: "total_reasoning_tokens", Type: "INTEGER", Nullable: true},
			{Name: "total_tool_tokens", Type: "INTEGER", Nullable: true},
			{Name: "session_cost", Type: "REAL", Nullable: true},
			{Name: "subagent_count", Type: "INTEGER", Nullable: true},
			{Name: "subagent_tool_calls", Type: "INTEGER", Nullable: true},
			{Name: "model_switches", Type: "INTEGER", Nullable: true},
			{Name: "unique_models", Type: "INTEGER", Nullable: true},
			{Name: "median_response_tokens", Type: "INTEGER", Nullable: true},
			{Name: "max_response_tokens", Type: "INTEGER", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS metrics (
	session_id TEXT PRIMARY KEY,
	turn_count INTEGER,
	tool_call_count INTEGER,
	read_count INTEGER,
	modify_count INTEGER,
	create_count INTEGER,
	execute_count INTEGER,
	delegate_count INTEGER,
	read_ratio REAL,
	time_to_first_action REAL,
	idle_ratio REAL,
	total_input_tokens INTEGER,
	total_output_tokens INTEGER,
	total_cache_read_tokens INTEGER,
	total_cache_creation_tokens INTEGER,
	total_reasoning_tokens INTEGER,
	total_tool_tokens INTEGER,
	session_cost REAL,
	subagent_count INTEGER,
	subagent_tool_calls INTEGER,
	model_switches INTEGER,
	unique_models INTEGER,
	median_response_tokens INTEGER,
	max_response_tokens INTEGER,
	raw_json TEXT
);`,
	}
}

func attachmentsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "attachments",
		Description: "One row per source artifact or attachment reference associated with a session, turn, tool call, or event.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "attachment_id", Type: "TEXT", PrimaryKey: true},
			{Name: "timestamp", Type: "TEXT", Nullable: true},
			{Name: "kind", Type: "TEXT", Nullable: true},
			{Name: "name", Type: "TEXT", Nullable: true},
			{Name: "media_type", Type: "TEXT", Nullable: true},
			{Name: "path", Type: "TEXT", Nullable: true},
			{Name: "url", Type: "TEXT", Nullable: true},
			{Name: "size_bytes", Type: "INTEGER", Nullable: true},
			{Name: "hash", Type: "TEXT", Nullable: true},
			{Name: "content_ref", Type: "TEXT", Nullable: true},
			{Name: "text_preview", Type: "TEXT", Nullable: true},
			{Name: "turn_index", Type: "INTEGER", Nullable: true},
			{Name: "tool_call_id", Type: "TEXT", Nullable: true},
			{Name: "event_id", Type: "TEXT", Nullable: true},
			{Name: "framework_metadata_json", Type: "TEXT", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS attachments (
	session_id TEXT NOT NULL,
	attachment_id TEXT NOT NULL,
	timestamp TEXT,
	kind TEXT,
	name TEXT,
	media_type TEXT,
	path TEXT,
	url TEXT,
	size_bytes INTEGER,
	hash TEXT,
	content_ref TEXT,
	text_preview TEXT,
	turn_index INTEGER,
	tool_call_id TEXT,
	event_id TEXT,
	framework_metadata_json TEXT,
	raw_json TEXT,
	PRIMARY KEY (session_id, attachment_id)
);`,
	}
}

func eventsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "events",
		Description: "One row per timeline/renderable event derived from turns, tool calls, annotations, or explicit source event records.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true},
			{Name: "event_id", Type: "TEXT", PrimaryKey: true},
			{Name: "timestamp", Type: "TEXT", Nullable: true},
			{Name: "turn_index", Type: "INTEGER", Nullable: true},
			{Name: "ordinal", Type: "INTEGER", Nullable: true},
			{Name: "kind", Type: "TEXT", Nullable: true},
			{Name: "role", Type: "TEXT", Nullable: true},
			{Name: "tool_call_id", Type: "TEXT", Nullable: true},
			{Name: "annotation_id", Type: "TEXT", Nullable: true},
			{Name: "attachment_id", Type: "TEXT", Nullable: true},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "summary", Type: "TEXT", Nullable: true},
			{Name: "text", Type: "TEXT", Nullable: true},
			{Name: "severity", Type: "TEXT", Nullable: true},
			{Name: "collapsed_by_default", Type: "INTEGER", Nullable: true},
			{Name: "framework_metadata_json", Type: "TEXT", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS events (
	session_id TEXT NOT NULL,
	event_id TEXT NOT NULL,
	timestamp TEXT,
	turn_index INTEGER,
	ordinal INTEGER,
	kind TEXT,
	role TEXT,
	tool_call_id TEXT,
	annotation_id TEXT,
	attachment_id TEXT,
	title TEXT,
	summary TEXT,
	text TEXT,
	severity TEXT,
	collapsed_by_default INTEGER,
	framework_metadata_json TEXT,
	raw_json TEXT,
	PRIMARY KEY (session_id, event_id)
);`,
	}
}

func indexStatements() []string {
	return []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent_framework ON sessions(agent_framework);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_working_directory ON sessions(working_directory);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_turns_session_role ON turns(session_id, role);`,
		`CREATE INDEX IF NOT EXISTS idx_turns_session_model ON turns(session_id, model);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_session_turn ON tool_calls(session_id, emitting_turn_index);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_tool_operation ON tool_calls(tool_name, operation_type);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_spawned_agent ON tool_calls(spawned_agent_type, spawned_agent_task_scope);`,
		`CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);`,
		`CREATE INDEX IF NOT EXISTS idx_annotations_scope ON annotations(scope_type, target_id);`,
		`CREATE INDEX IF NOT EXISTS idx_annotations_category ON annotations(category);`,
		`CREATE INDEX IF NOT EXISTS idx_handovers_direction ON handovers(direction);`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_kind ON attachments(kind);`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_tool_call ON attachments(session_id, tool_call_id);`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_event ON attachments(session_id, event_id);`,
		`CREATE INDEX IF NOT EXISTS idx_events_session_turn ON events(session_id, turn_index, ordinal);`,
		`CREATE INDEX IF NOT EXISTS idx_events_kind ON events(kind);`,
	}
}

func ValidateAllowedTable(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, allowed := range AllowedTableNames() {
		if name == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}
