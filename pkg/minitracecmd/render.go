package minitracecmd

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
)

type RenderContext struct {
	TableName string
	Values    map[string]any
}

var renderIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func RenderCommand(cmd *MinitraceCommand, ctx RenderContext) (string, error) {
	if cmd == nil {
		return "", ErrNilCommand
	}
	if cmd.Kind == MinitraceCommandAlias {
		return "", ErrCannotRenderAlias
	}
	if strings.TrimSpace(cmd.Query) == "" {
		return "", ErrMissingQuery
	}
	if !renderIdentifierPattern.MatchString(ctx.TableName) {
		return "", fmt.Errorf("%w: %q", ErrInvalidTableName, ctx.TableName)
	}

	queryText := strings.ReplaceAll(cmd.Query, "{{TABLE_NAME}}", ctx.TableName)
	tmpl := templating.CreateTemplate("query").
		Funcs(templating.TemplateFuncs).
		Option("missingkey=zero").
		Funcs(template.FuncMap{
			"sqlString":   sqlString,
			"sqlStringIn": sqlStringIn,
			"sqlIntIn":    sqlIntIn,
			"sqlLike":     sqlLike,
			"sqlDate":     sqlDate,
			"sqlDateTime": sqlDateTime,
			"sqlEscape":   sqlEscape,
		})
	parsed, err := tmpl.Parse(queryText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, copyValues(ctx.Values)); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// sqlDate formats a date value for SQL queries as 'YYYY-MM-DD' (or RFC3339
// for non-local timezones). Ported from clay/pkg/sql so the SQL command
// renderer does not pull in clay's DuckDB-backed sql package.
func sqlDate(date any) (string, error) {
	return sqlFormatDate(date, time.RFC3339, "2006-01-02")
}

// sqlDateTime formats a datetime value for SQL queries as
// 'YYYY-MM-DDTHH:MM:SS' (or RFC3339 for non-local timezones).
func sqlDateTime(date any) (string, error) {
	return sqlFormatDate(date, time.RFC3339, "2006-01-02T15:04:05")
}

func sqlFormatDate(date any, fullFormat string, defaultFormat string) (string, error) {
	switch v := date.(type) {
	case string:
		parsedDate, err := fields.ParseDate(v)
		if err != nil {
			return "", err
		}
		if parsedDate.Location() == time.Local {
			return "'" + parsedDate.Format(defaultFormat) + "'", nil
		}
		return "'" + parsedDate.Format(fullFormat) + "'", nil
	case time.Time:
		if v.Location() == time.Local {
			return "'" + v.Format(defaultFormat) + "'", nil
		}
		return "'" + v.Format(fullFormat) + "'", nil
	default:
		return "", fmt.Errorf("could not parse date %v", date)
	}
}

func ResolveAliasCommand(catalog *Catalog, cmd *MinitraceCommand, values map[string]any) (*MinitraceCommand, map[string]any, error) {
	if cmd == nil {
		return nil, nil, ErrNilCommand
	}
	if cmd.Kind != MinitraceCommandAlias {
		return cmd, copyValues(values), nil
	}
	if catalog == nil {
		return nil, nil, fmt.Errorf("%w: alias %q targets %q", ErrAliasTargetNotFound, cmd.Name, cmd.AliasFor)
	}

	target, ok := catalog.ByName[cmd.AliasFor]
	if !ok || target == nil {
		return nil, nil, fmt.Errorf("%w: alias %q targets %q", ErrAliasTargetNotFound, cmd.Name, cmd.AliasFor)
	}

	return target, mergeValues(cmd.AliasFlags, values), nil
}
