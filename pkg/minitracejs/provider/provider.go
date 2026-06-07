package provider

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja_nodejs/require"
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	querycmd "github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/query"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

const PackageID = "go-minitrace"

type HostServices interface {
	Conn() *sql.Conn
	RuntimeSettings() minitracejs.RuntimeSettings
	CommandName() string
}

type queriesCommandProviderConfig struct {
	AppName           string   `json:"appName,omitempty"`
	QueryRepositories []string `json:"queryRepositories,omitempty"`
}

var moduleConfigSchema = json.RawMessage(`{
  "type": "object",
  "description": "Optional minitrace module settings. Query execution requires HostServices to provide a SQL connection.",
  "additionalProperties": false
}`)

var queriesConfigSchema = json.RawMessage(`{
  "type": "object",
  "description": "Configuration for the go-minitrace queries command provider.",
  "additionalProperties": false,
  "properties": {
    "appName": {
      "type": "string",
      "description": "Application config namespace used when loading configured query repositories."
    },
    "queryRepositories": {
      "type": "array",
      "description": "Additional query repository directories to load.",
      "items": { "type": "string" }
    }
  }
}`)

func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID,
		providerapi.Module{
			Name:         minitracejs.ModuleName,
			DefaultAs:    minitracejs.ModuleName,
			Description:  "Read-only minitrace query helpers exposed as require(\"minitrace\").",
			ConfigSchema: moduleConfigSchema,
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				var conn *sql.Conn
				var commandName string
				settings := minitracejs.RuntimeSettings{}
				if host, ok := ctx.Host.(HostServices); ok && host != nil {
					conn = host.Conn()
					commandName = host.CommandName()
					settings = host.RuntimeSettings()
				}
				baseCtx := ctx.Context
				if baseCtx == nil {
					baseCtx = context.Background()
				}
				return minitracejs.NewLoader(baseCtx, conn, commandName, settings), nil
			},
		},
		providerapi.CommandSetProvider{
			Name:          "queries",
			DefaultMount:  "minitrace",
			Description:   "Run repository-backed go-minitrace query commands",
			ConfigSchema:  queriesConfigSchema,
			NewCommandSet: newQueriesCommandSet,
		},
	)
}

func newQueriesCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	cfg := queriesCommandProviderConfig{AppName: "go-minitrace"}
	if len(ctx.Config) > 0 {
		if err := json.Unmarshal(ctx.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode go-minitrace queries command provider config: %w", err)
		}
	}
	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		appName = "go-minitrace"
	}
	catalog, err := minitracecmd.LoadConfiguredCatalog(appName, cfg.QueryRepositories)
	if err != nil {
		return nil, fmt.Errorf("load go-minitrace query catalog: %w", err)
	}
	commands, err := querycmd.NewMinitraceCatalogCommands(catalog)
	if err != nil {
		return nil, fmt.Errorf("build go-minitrace query commands: %w", err)
	}
	return &providerapi.CommandSet{
		Commands: commands,
		ParserConfig: &glazedcli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug, querycmd.QueryRuntimeSectionSlug, schema.GlobalDefaultSlug},
		},
	}, nil
}
