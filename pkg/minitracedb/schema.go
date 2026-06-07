package minitracedb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const SchemaVersion = "normalized-sqlite-v1"

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
	return []TableDescriptor{sessionsTable(), turnsTable(), toolCallsTable(), turnToolCallsTable(), filesTable(), metricsTable(), eventsTable()}
}

func AllowedTableNames() []string {
	tables := Tables()
	ret := make([]string, 0, len(tables))
	for _, table := range tables {
		ret = append(ret, table.Name)
	}
	return ret
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
	return nil
}

func sessionsTable() TableDescriptor {
	return TableDescriptor{
		Name:        "sessions",
		Description: "One row per minitrace session.",
		Columns: []ColumnDescriptor{
			{Name: "session_id", Type: "TEXT", PrimaryKey: true, Description: "Stable minitrace session id."},
			{Name: "schema_version", Type: "TEXT", Nullable: true},
			{Name: "profile", Type: "TEXT", Nullable: true},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "summary", Type: "TEXT", Nullable: true},
			{Name: "classification", Type: "TEXT", Nullable: true},
			{Name: "source_format", Type: "TEXT", Nullable: true},
			{Name: "source_path", Type: "TEXT", Nullable: true},
			{Name: "converted_at", Type: "TEXT", Nullable: true},
			{Name: "model", Type: "TEXT", Nullable: true},
			{Name: "agent_framework", Type: "TEXT", Nullable: true},
			{Name: "working_directory", Type: "TEXT", Nullable: true},
			{Name: "started_at", Type: "TEXT", Nullable: true},
			{Name: "ended_at", Type: "TEXT", Nullable: true},
			{Name: "turn_count", Type: "INTEGER", Nullable: true},
			{Name: "tool_call_count", Type: "INTEGER", Nullable: true},
			{Name: "read_count", Type: "INTEGER", Nullable: true},
			{Name: "modify_count", Type: "INTEGER", Nullable: true},
			{Name: "create_count", Type: "INTEGER", Nullable: true},
			{Name: "execute_count", Type: "INTEGER", Nullable: true},
			{Name: "contains_error", Type: "INTEGER", Nullable: true},
			{Name: "raw_json", Type: "TEXT", Nullable: true},
		},
		CreateSQL: `CREATE TABLE IF NOT EXISTS sessions (
	session_id TEXT PRIMARY KEY,
	schema_version TEXT,
	profile TEXT,
	title TEXT,
	summary TEXT,
	classification TEXT,
	source_format TEXT,
	source_path TEXT,
	converted_at TEXT,
	model TEXT,
	agent_framework TEXT,
	working_directory TEXT,
	started_at TEXT,
	ended_at TEXT,
	turn_count INTEGER,
	tool_call_count INTEGER,
	read_count INTEGER,
	modify_count INTEGER,
	create_count INTEGER,
	execute_count INTEGER,
	contains_error INTEGER,
	raw_json TEXT
);`,
	}
}

func turnsTable() TableDescriptor {
	return TableDescriptor{Name: "turns", Description: "One row per conversational turn.", Columns: []ColumnDescriptor{{Name: "session_id", Type: "TEXT", PrimaryKey: true}, {Name: "turn_index", Type: "INTEGER", PrimaryKey: true}, {Name: "timestamp", Type: "TEXT", Nullable: true}, {Name: "role", Type: "TEXT", Nullable: true}, {Name: "source", Type: "TEXT", Nullable: true}, {Name: "model", Type: "TEXT", Nullable: true}, {Name: "content_type", Type: "TEXT", Nullable: true}, {Name: "input_channel", Type: "TEXT", Nullable: true}, {Name: "content", Type: "TEXT", Nullable: true}, {Name: "thinking", Type: "TEXT", Nullable: true}, {Name: "was_streamed", Type: "INTEGER", Nullable: true}, {Name: "input_tokens", Type: "INTEGER", Nullable: true}, {Name: "output_tokens", Type: "INTEGER", Nullable: true}, {Name: "reasoning_tokens", Type: "INTEGER", Nullable: true}, {Name: "raw_json", Type: "TEXT", Nullable: true}}, CreateSQL: `CREATE TABLE IF NOT EXISTS turns (
	session_id TEXT NOT NULL,
	turn_index INTEGER NOT NULL,
	timestamp TEXT,
	role TEXT,
	source TEXT,
	model TEXT,
	content_type TEXT,
	input_channel TEXT,
	content TEXT,
	thinking TEXT,
	was_streamed INTEGER,
	input_tokens INTEGER,
	output_tokens INTEGER,
	reasoning_tokens INTEGER,
	raw_json TEXT,
	PRIMARY KEY (session_id, turn_index)
);`}
}

func toolCallsTable() TableDescriptor {
	return TableDescriptor{Name: "tool_calls", Description: "One row per tool call.", Columns: []ColumnDescriptor{{Name: "session_id", Type: "TEXT", PrimaryKey: true}, {Name: "tool_call_id", Type: "TEXT", PrimaryKey: true}, {Name: "emitting_turn_index", Type: "INTEGER", Nullable: true}, {Name: "timestamp", Type: "TEXT", Nullable: true}, {Name: "tool_name", Type: "TEXT", Nullable: true}, {Name: "operation_type", Type: "TEXT", Nullable: true}, {Name: "file_path", Type: "TEXT", Nullable: true}, {Name: "command", Type: "TEXT", Nullable: true}, {Name: "justification", Type: "TEXT", Nullable: true}, {Name: "arguments_json", Type: "TEXT", Nullable: true}, {Name: "success", Type: "INTEGER", Nullable: true}, {Name: "result", Type: "TEXT", Nullable: true}, {Name: "error", Type: "TEXT", Nullable: true}, {Name: "exit_code", Type: "INTEGER", Nullable: true}, {Name: "duration_ms", Type: "INTEGER", Nullable: true}, {Name: "truncated", Type: "INTEGER", Nullable: true}, {Name: "raw_json", Type: "TEXT", Nullable: true}}, CreateSQL: `CREATE TABLE IF NOT EXISTS tool_calls (
	session_id TEXT NOT NULL,
	tool_call_id TEXT NOT NULL,
	emitting_turn_index INTEGER,
	timestamp TEXT,
	tool_name TEXT,
	operation_type TEXT,
	file_path TEXT,
	command TEXT,
	justification TEXT,
	arguments_json TEXT,
	success INTEGER,
	result TEXT,
	error TEXT,
	exit_code INTEGER,
	duration_ms INTEGER,
	truncated INTEGER,
	raw_json TEXT,
	PRIMARY KEY (session_id, tool_call_id)
);`}
}

func turnToolCallsTable() TableDescriptor {
	return TableDescriptor{Name: "turn_tool_calls", Description: "Join table preserving tool call membership and ordinal per turn.", Columns: []ColumnDescriptor{{Name: "session_id", Type: "TEXT", PrimaryKey: true}, {Name: "turn_index", Type: "INTEGER", PrimaryKey: true}, {Name: "tool_call_id", Type: "TEXT", PrimaryKey: true}, {Name: "ordinal", Type: "INTEGER", PrimaryKey: true}}, CreateSQL: `CREATE TABLE IF NOT EXISTS turn_tool_calls (
	session_id TEXT NOT NULL,
	turn_index INTEGER NOT NULL,
	tool_call_id TEXT NOT NULL,
	ordinal INTEGER NOT NULL,
	PRIMARY KEY (session_id, turn_index, ordinal)
);`}
}

func filesTable() TableDescriptor {
	return TableDescriptor{Name: "files", Description: "One row per file path touched by a tool call.", Columns: []ColumnDescriptor{{Name: "session_id", Type: "TEXT"}, {Name: "tool_call_id", Type: "TEXT"}, {Name: "path", Type: "TEXT"}, {Name: "operation_type", Type: "TEXT", Nullable: true}, {Name: "tool_name", Type: "TEXT", Nullable: true}, {Name: "success", Type: "INTEGER", Nullable: true}, {Name: "turn_index", Type: "INTEGER", Nullable: true}}, CreateSQL: `CREATE TABLE IF NOT EXISTS files (
	session_id TEXT NOT NULL,
	tool_call_id TEXT NOT NULL,
	path TEXT NOT NULL,
	operation_type TEXT,
	tool_name TEXT,
	success INTEGER,
	turn_index INTEGER
);`}
}

func metricsTable() TableDescriptor {
	return TableDescriptor{Name: "metrics", Description: "Wide metrics table with one row per session.", Columns: []ColumnDescriptor{{Name: "session_id", Type: "TEXT", PrimaryKey: true}, {Name: "turn_count", Type: "INTEGER", Nullable: true}, {Name: "tool_call_count", Type: "INTEGER", Nullable: true}, {Name: "read_count", Type: "INTEGER", Nullable: true}, {Name: "modify_count", Type: "INTEGER", Nullable: true}, {Name: "create_count", Type: "INTEGER", Nullable: true}, {Name: "execute_count", Type: "INTEGER", Nullable: true}, {Name: "delegate_count", Type: "INTEGER", Nullable: true}, {Name: "read_ratio", Type: "REAL", Nullable: true}, {Name: "time_to_first_action", Type: "REAL", Nullable: true}, {Name: "total_input_tokens", Type: "INTEGER", Nullable: true}, {Name: "total_output_tokens", Type: "INTEGER", Nullable: true}, {Name: "session_cost", Type: "REAL", Nullable: true}, {Name: "raw_json", Type: "TEXT", Nullable: true}}, CreateSQL: `CREATE TABLE IF NOT EXISTS metrics (
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
	total_input_tokens INTEGER,
	total_output_tokens INTEGER,
	session_cost REAL,
	raw_json TEXT
);`}
}

func eventsTable() TableDescriptor {
	return TableDescriptor{Name: "events", Description: "One row per timeline/renderable event derived from turns and tool calls.", Columns: []ColumnDescriptor{{Name: "session_id", Type: "TEXT", PrimaryKey: true}, {Name: "event_id", Type: "TEXT", PrimaryKey: true}, {Name: "turn_index", Type: "INTEGER", Nullable: true}, {Name: "ordinal", Type: "INTEGER", Nullable: true}, {Name: "kind", Type: "TEXT", Nullable: true}, {Name: "role", Type: "TEXT", Nullable: true}, {Name: "tool_call_id", Type: "TEXT", Nullable: true}, {Name: "title", Type: "TEXT", Nullable: true}, {Name: "summary", Type: "TEXT", Nullable: true}, {Name: "text", Type: "TEXT", Nullable: true}, {Name: "severity", Type: "TEXT", Nullable: true}, {Name: "collapsed_by_default", Type: "INTEGER", Nullable: true}, {Name: "raw_json", Type: "TEXT", Nullable: true}}, CreateSQL: `CREATE TABLE IF NOT EXISTS events (
	session_id TEXT NOT NULL,
	event_id TEXT NOT NULL,
	turn_index INTEGER,
	ordinal INTEGER,
	kind TEXT,
	role TEXT,
	tool_call_id TEXT,
	title TEXT,
	summary TEXT,
	text TEXT,
	severity TEXT,
	collapsed_by_default INTEGER,
	raw_json TEXT,
	PRIMARY KEY (session_id, event_id)
);`}
}

func indexStatements() []string {
	return []string{
		`CREATE INDEX IF NOT EXISTS idx_turns_session_role ON turns(session_id, role);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_session_turn ON tool_calls(session_id, emitting_turn_index);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_tool_operation ON tool_calls(tool_name, operation_type);`,
		`CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);`,
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
