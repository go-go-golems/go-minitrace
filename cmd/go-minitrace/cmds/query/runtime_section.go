package query

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

const QueryRuntimeSectionSlug = "query-runtime"

func NewQueryRuntimeSection() (schema.Section, error) {
	return schema.NewSection(
		QueryRuntimeSectionSlug,
		"Query execution settings",
		schema.WithFields(
			fields.New("archive-glob", fields.TypeStringList, fields.WithDefault([]string{"./output/active/*/*.minitrace.json"}), fields.WithHelp("Repeatable glob flag for minitrace session JSON files to load")),
			fields.New("db-path", fields.TypeString, fields.WithDefault(":memory:"), fields.WithHelp("DuckDB database path to use; :memory: keeps the query session ephemeral")),
			fields.New("table-name", fields.TypeString, fields.WithDefault("sessions_base"), fields.WithHelp("Table name to create from the loaded archive")),
			fields.New("persist-loaded", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Request a persistent loaded table rather than a temp table")),
		),
	)
}
