package provider

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	_ "github.com/mattn/go-sqlite3"
)

type fakeHost struct {
	conn *sql.Conn
}

func (h fakeHost) Conn() *sql.Conn { return h.conn }
func (h fakeHost) RuntimeSettings() minitracejs.RuntimeSettings {
	return minitracejs.RuntimeSettings{TableName: "events", DBPath: ":memory:"}
}
func (h fakeHost) CommandName() string { return "test-command" }

func TestRegisterProvider(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, minitracejs.ModuleName)
	if !ok {
		t.Fatalf("missing module %s.%s", PackageID, minitracejs.ModuleName)
	}
	if mod.DefaultAs != minitracejs.ModuleName {
		t.Fatalf("default alias = %q, want %q", mod.DefaultAs, minitracejs.ModuleName)
	}
}

func TestProviderAllowsModuleWithoutHostServices(t *testing.T) {
	mod := resolveModule(t)
	loader, err := mod.New(providerapi.ModuleContext{})
	if err != nil {
		t.Fatalf("create loader without host services: %v", err)
	}
	if loader == nil {
		t.Fatalf("expected loader")
	}
}

func TestRegisterQueriesCommandProvider(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	provider, ok := registry.ResolveCommandSetProvider(PackageID, "queries")
	if !ok {
		t.Fatalf("missing command provider %s.queries", PackageID)
	}
	if provider.DefaultMount != "minitrace" {
		t.Fatalf("default mount = %q, want minitrace", provider.DefaultMount)
	}
	if !json.Valid(provider.ConfigSchema) {
		t.Fatalf("command provider config schema is not valid JSON: %s", string(provider.ConfigSchema))
	}
	if !containsJSONField(provider.ConfigSchema, "appName") || !containsJSONField(provider.ConfigSchema, "queryRepositories") {
		t.Fatalf("command provider config schema does not describe accepted fields: %s", string(provider.ConfigSchema))
	}
}

func TestQueriesCommandProviderBuildsCatalogCommands(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	provider, ok := registry.ResolveCommandSetProvider(PackageID, "queries")
	if !ok {
		t.Fatalf("missing command provider")
	}
	config, err := json.Marshal(map[string]any{"appName": "go-minitrace-test"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	set, err := provider.New(providerapi.CommandSetContext{
		Context:        context.Background(),
		PackageID:      PackageID,
		Name:           "queries",
		Mount:          "traces",
		RuntimeProfile: "main",
		Config:         config,
		Providers:      registry,
	})
	if err != nil {
		t.Fatalf("create command set: %v", err)
	}
	if set == nil || len(set.Commands) == 0 {
		t.Fatalf("expected embedded catalog commands")
	}
	foundNested := false
	for _, command := range set.Commands {
		if command == nil || command.Description() == nil {
			continue
		}
		if len(command.Description().Parents) > 0 {
			foundNested = true
			break
		}
	}
	if !foundNested {
		t.Fatalf("expected at least one catalog command to preserve folder parents")
	}
}

func TestModuleLoaderQueriesHostConnection(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	defer func() { _ = conn.Close() }()

	mod := resolveModule(t)
	loader, err := mod.New(providerapi.ModuleContext{Context: context.Background(), Host: fakeHost{conn: conn}})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}

	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set exports: %v", err)
	}
	loader(vm, moduleObj)
	queryOne, ok := goja.AssertFunction(exports.Get("queryOne"))
	if !ok {
		t.Fatalf("queryOne export is not a function")
	}
	ret, err := queryOne(goja.Undefined(), vm.ToValue("select 1 as ok"))
	if err != nil {
		t.Fatalf("queryOne: %v", err)
	}
	row := ret.ToObject(vm)
	if got := row.Get("ok").ToInteger(); got != 1 {
		t.Fatalf("ok = %d, want 1", got)
	}
}

func containsJSONField(raw json.RawMessage, field string) bool {
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return false
	}
	properties, _ := decoded["properties"].(map[string]any)
	_, ok := properties[field]
	return ok
}

func resolveModule(t *testing.T) providerapi.Module {
	t.Helper()
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, minitracejs.ModuleName)
	if !ok {
		t.Fatalf("missing module")
	}
	return mod
}
