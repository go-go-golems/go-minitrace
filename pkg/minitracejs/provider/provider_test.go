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
func (h fakeHost) CommandName() string                      { return "test-command" }
func (h fakeHost) AssetResolver() providerapi.AssetResolver { return nil }

func TestRegisterProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
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
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{})
	if err != nil {
		t.Fatalf("create loader without host services: %v", err)
	}
	if loader == nil {
		t.Fatalf("expected loader")
	}
}

func TestRegisterQueriesCommandProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
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
	registry := providerapi.NewProviderRegistry()
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
	set, err := provider.NewCommandSet(providerapi.CommandSetContext{
		Context:   context.Background(),
		PackageID: PackageID,
		Name:      "queries",
		Mount:     "traces",
		Config:    config,
		Providers: registry,
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

func TestModuleLoaderProvidesDBBuilder(t *testing.T) {
	mod := resolveModule(t)
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background()})
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
	if err := vm.Set("mt", exports); err != nil {
		t.Fatalf("set mt: %v", err)
	}

	value, err := vm.RunString(`
		const db = mt.db().SQLiteMemory().MaxRows(5).Build();
		const stats = db.stats();
		const tables = db.tables().map(t => t.name).sort();
		const row = db.queryOne("SELECT COUNT(*) AS n FROM sessions");
		const rejected = db.queryResult("INSERT INTO sessions(session_id) VALUES ('bad')");
		db.close();
		JSON.stringify({ stats, tables, row, rejectedError: rejected.error });
	`)
	if err != nil {
		t.Fatalf("run db builder script: %v", err)
	}
	var got struct {
		Stats struct {
			SchemaVersion string `json:"schemaVersion"`
			Dialect       string `json:"dialect"`
			Tables        int    `json:"tables"`
		} `json:"stats"`
		Tables        []string       `json:"tables"`
		Row           map[string]any `json:"row"`
		RejectedError string         `json:"rejectedError"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got.Stats.Dialect != "sqlite" || got.Stats.Tables == 0 {
		t.Fatalf("unexpected stats: %#v", got.Stats)
	}
	if len(got.Tables) == 0 || got.Tables[0] == "" {
		t.Fatalf("expected table names, got %#v", got.Tables)
	}
	if got.Row["n"] == nil {
		t.Fatalf("expected count row, got %#v", got.Row)
	}
	if got.RejectedError == "" {
		t.Fatalf("expected rejected write error")
	}
}

func TestModuleLoaderDBBuilderValidation(t *testing.T) {
	mod := resolveModule(t)
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background()})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}
	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	_ = moduleObj.Set("exports", exports)
	loader(vm, moduleObj)
	_ = vm.Set("mt", exports)
	value, err := vm.RunString(`JSON.stringify(mt.db().MaxRows(-1).Validate())`)
	if err != nil {
		t.Fatalf("run validation script: %v", err)
	}
	var got struct {
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal validation: %v", err)
	}
	if got.Valid || len(got.Errors) == 0 {
		t.Fatalf("expected validation errors, got %#v", got)
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
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Host: fakeHost{conn: conn}})
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
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, minitracejs.ModuleName)
	if !ok {
		t.Fatalf("missing module")
	}
	return mod
}
