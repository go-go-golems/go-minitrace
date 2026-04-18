package minitracecmd

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	claysql "github.com/go-go-golems/clay/pkg/sql"
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
	tmpl := claysql.CreateTemplate(context.Background(), nil, copyValues(ctx.Values), nil).
		Option("missingkey=zero").
		Funcs(template.FuncMap{
			"sqlString":   sqlString,
			"sqlStringIn": sqlStringIn,
			"sqlIntIn":    sqlIntIn,
			"sqlLike":     sqlLike,
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
