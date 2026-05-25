package provider

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

const PackageID = "go-minitrace"

type HostServices interface {
	Conn() *sql.Conn
	RuntimeSettings() minitracejs.RuntimeSettings
	CommandName() string
}

var configSchema = json.RawMessage(`{
  "type": "object",
  "description": "The first provider version is host-services only. Static config is reserved for a future read-only DB opening mode.",
  "additionalProperties": false
}`)

func Register(registry *providerapi.Registry) error {
	return registry.Package(PackageID, providerapi.Module{
		Name:         minitracejs.ModuleName,
		DefaultAs:    minitracejs.ModuleName,
		Description:  "Read-only minitrace query helpers exposed as require(\"minitrace\").",
		ConfigSchema: configSchema,
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			host, ok := ctx.Host.(HostServices)
			if !ok || host == nil {
				return nil, fmt.Errorf("go-minitrace provider requires minitrace HostServices")
			}
			conn := host.Conn()
			if conn == nil {
				return nil, fmt.Errorf("go-minitrace provider host returned nil SQL connection")
			}
			baseCtx := ctx.Context
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			return minitracejs.NewLoader(baseCtx, conn, host.CommandName(), host.RuntimeSettings()), nil
		},
	})
}
