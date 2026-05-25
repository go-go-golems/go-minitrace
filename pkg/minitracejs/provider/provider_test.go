package provider

import (
	"context"
	"database/sql"
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

func TestProviderRequiresHostServices(t *testing.T) {
	mod := resolveModule(t)
	if _, err := mod.New(providerapi.ModuleContext{}); err == nil {
		t.Fatalf("expected missing host services error")
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
