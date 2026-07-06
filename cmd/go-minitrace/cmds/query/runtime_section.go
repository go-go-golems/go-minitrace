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
			fields.New("db-path", fields.TypeString, fields.WithDefault(":memory:"), fields.WithHelp("Deprecated: ignored by SQL commands, still exposed to JS commands via mt.runtime")),
			fields.New("table-name", fields.TypeString, fields.WithDefault("sessions_base"), fields.WithHelp("Deprecated: ignored by SQL commands, still exposed to JS commands via mt.runtime")),
			fields.New("persist-loaded", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Deprecated: ignored by SQL commands, still exposed to JS commands via mt.runtime")),
		),
	)
}
