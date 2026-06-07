package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

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

func TestModuleLoaderDBBuilderLoadsNativeFile(t *testing.T) {
	path := writeNativeSessionFixture(t)
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
	_ = vm.Set("fixturePath", path)
	value, err := vm.RunString(`
		const db = mt.db().File(fixturePath).SQLiteMemory().Build();
		const summary = {
			sessions: db.queryOne("SELECT COUNT(*) AS n FROM sessions").n,
			turns: db.queryOne("SELECT COUNT(*) AS n FROM turns").n,
			tools: db.queryOne("SELECT COUNT(*) AS n FROM tool_calls").n,
			file: db.queryOne("SELECT path FROM files").path,
			sources: db.sources().length
		};
		db.close();
		JSON.stringify(summary);
	`)
	if err != nil {
		t.Fatalf("run file db builder script: %v", err)
	}
	var got struct {
		Sessions int    `json:"sessions"`
		Turns    int    `json:"turns"`
		Tools    int    `json:"tools"`
		File     string `json:"file"`
		Sources  int    `json:"sources"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal file result: %v", err)
	}
	if got.Sessions != 1 || got.Turns != 2 || got.Tools != 1 || got.File != "app.go" || got.Sources != 1 {
		t.Fatalf("unexpected file load result: %#v", got)
	}
}

func TestModuleLoaderDBBuilderLoadsNativeDirAndGlob(t *testing.T) {
	path := writeNativeSessionFixture(t)
	dir := filepath.Dir(path)
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
	_ = vm.Set("fixtureDir", dir)
	_ = vm.Set("fixtureGlob", filepath.Join(dir, "*.minitrace.json"))
	value, err := vm.RunString(`
		const byDir = mt.db().Dir(fixtureDir).Build();
		const byGlob = mt.db().Glob(fixtureGlob).Build();
		const result = {
			dirSessions: byDir.queryOne("SELECT COUNT(*) AS n FROM sessions").n,
			globSessions: byGlob.queryOne("SELECT COUNT(*) AS n FROM sessions").n,
			dirSources: byDir.sources().length,
			globSources: byGlob.sources().length
		};
		byDir.close();
		byGlob.close();
		JSON.stringify(result);
	`)
	if err != nil {
		t.Fatalf("run dir/glob db builder script: %v", err)
	}
	var got struct {
		DirSessions  int `json:"dirSessions"`
		GlobSessions int `json:"globSessions"`
		DirSources   int `json:"dirSources"`
		GlobSources  int `json:"globSources"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal dir/glob result: %v", err)
	}
	if got.DirSessions != 1 || got.GlobSessions != 1 || got.DirSources != 1 || got.GlobSources != 1 {
		t.Fatalf("unexpected dir/glob result: %#v", got)
	}
}

func TestModuleLoaderDBBuilderAutoConvertsJSONLContent(t *testing.T) {
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "pi-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
		{"type": "message", "timestamp": "2026-03-29T12:00:01Z", "message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "hello"}}}},
	}))
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
	_ = vm.Set("jsonlContent", content)
	value, err := vm.RunString(`
		const db = mt.db().Content(jsonlContent, {name: "pi-content.jsonl"}).AutoConvert(true).Build();
		const result = {
			id: db.queryOne("SELECT session_id FROM sessions").session_id,
			turns: db.queryOne("SELECT COUNT(*) AS n FROM turns").n,
			diagnosticFormat: db.diagnostics()[0].format
		};
		db.close();
		JSON.stringify(result);
	`)
	if err != nil {
		t.Fatalf("run content autoconvert script: %v", err)
	}
	var got struct {
		ID               string `json:"id"`
		Turns            int    `json:"turns"`
		DiagnosticFormat string `json:"diagnosticFormat"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal content result: %v", err)
	}
	if got.ID != "pi-content" || got.Turns != 1 || got.DiagnosticFormat != "pi-jsonl" {
		t.Fatalf("unexpected content autoconvert result: %#v", got)
	}
}

func TestModuleLoaderDBBuilderNonStrictConversionKeepsValidSources(t *testing.T) {
	path := writeNativeSessionFixture(t)
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
	_ = vm.Set("fixturePath", path)
	value, err := vm.RunString(`
		const db = mt.db().File(fixturePath).Content("not json", "bad.txt").StrictConversion(false).Build();
		const result = {
			sessions: db.queryOne("SELECT COUNT(*) AS n FROM sessions").n,
			diagnosticCount: db.diagnostics().length,
			errorSeverity: db.diagnostics().some(d => d.severity === "error")
		};
		db.close();
		JSON.stringify(result);
	`)
	if err != nil {
		t.Fatalf("run non-strict script: %v", err)
	}
	var got struct {
		Sessions        int  `json:"sessions"`
		DiagnosticCount int  `json:"diagnosticCount"`
		ErrorSeverity   bool `json:"errorSeverity"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal non-strict result: %v", err)
	}
	if got.Sessions != 1 || got.DiagnosticCount < 2 || !got.ErrorSeverity {
		t.Fatalf("unexpected non-strict result: %#v", got)
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

func writeJSONLFixture(t *testing.T, records []map[string]any) []byte {
	t.Helper()
	payloads := make([]byte, 0)
	for _, record := range records {
		payload, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal jsonl record: %v", err)
		}
		payloads = append(payloads, payload...)
		payloads = append(payloads, '\n')
	}
	return payloads
}

func writeNativeSessionFixture(t *testing.T) string {
	t.Helper()
	session := minitrace.BuildSessionSkeleton("session-file-1", "pi", "minitrace-json-v1", "test")
	session.Turns = []minitrace.Turn{{Index: 0, Role: "user", Content: "Read app.go"}, {Index: 1, Role: "assistant", Content: "Done", ToolCallsInTurn: []string{"tool-1"}}}
	path := "app.go"
	result := "package main"
	emittingTurn := 1
	session.ToolCalls = []minitrace.ToolCall{{ID: "tool-1", EmittingTurnIndex: &emittingTurn, ToolName: "Read", OperationType: "read", Input: minitrace.ToolCallInput{FilePath: &path}, Output: minitrace.ToolCallOutput{Success: true, Result: &result}}}
	session.Metrics = minitrace.ComputeMetrics(session.Turns, session.ToolCalls, session.Timing, len(session.Annotations), nil)
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	filePath := filepath.Join(t.TempDir(), "session.minitrace.json")
	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return filePath
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
