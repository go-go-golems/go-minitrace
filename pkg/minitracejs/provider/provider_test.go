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

func TestModuleLoaderDBBuilderExposesCacheKeyAndCacheInfo(t *testing.T) {
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "cache-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
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
		const builder = mt.db().Content(jsonlContent, "cache-content.jsonl").AutoConvert(true);
		const builderKey = builder.CacheKey();
		const db = builder.Build();
		const handleKey = db.cacheInfo();
		const result = {
			builderKey: builderKey.key,
			handleKey: handleKey.key,
			schemaVersion: handleKey.options.schemaVersion,
			importerVersion: handleKey.options.importerVersion,
			converterVersion: handleKey.options.converterVersion,
			autoConvert: handleKey.options.autoConvert,
			sourceKind: handleKey.sources[0].kind,
			sourceName: handleKey.sources[0].name,
			sourceHash: handleKey.sources[0].sha256
		};
		db.close();
		JSON.stringify(result);
	`)
	if err != nil {
		t.Fatalf("run cache key script: %v", err)
	}
	var got struct {
		BuilderKey       string `json:"builderKey"`
		HandleKey        string `json:"handleKey"`
		SchemaVersion    string `json:"schemaVersion"`
		ImporterVersion  string `json:"importerVersion"`
		ConverterVersion string `json:"converterVersion"`
		AutoConvert      bool   `json:"autoConvert"`
		SourceKind       string `json:"sourceKind"`
		SourceName       string `json:"sourceName"`
		SourceHash       string `json:"sourceHash"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal cache key result: %v", err)
	}
	if got.BuilderKey == "" || got.BuilderKey != got.HandleKey {
		t.Fatalf("unexpected cache keys: %#v", got)
	}
	if got.SchemaVersion == "" || got.ImporterVersion == "" || got.ConverterVersion == "" || !got.AutoConvert {
		t.Fatalf("cache versions/options missing: %#v", got)
	}
	if got.SourceKind != "content" || got.SourceName != "cache-content.jsonl" || got.SourceHash == "" {
		t.Fatalf("source fingerprint missing: %#v", got)
	}
}

func TestModuleLoaderDBBuilderAutoCacheUsesMemoryDiskThenRebuild(t *testing.T) {
	cacheDir := t.TempDir()
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "auto-cache-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
		{"type": "message", "timestamp": "2026-03-29T12:00:01Z", "message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "hello"}}}},
	}))
	changedContent := content + "\n"
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
	_ = vm.Set("changedContent", changedContent)
	_ = vm.Set("cacheDir", cacheDir)
	value, err := vm.RunString(`
		const warmer = mt.db().Content(jsonlContent, "auto-cache-content.jsonl").AutoConvert(true).SQLiteDiskCache(cacheDir).Build();
		const warmInfo = warmer.cacheInfo();
		warmer.close();
		const fromDisk = mt.db().Content(jsonlContent, "auto-cache-content.jsonl").AutoConvert(true).Cache("auto").CacheDir(cacheDir).Build();
		const diskInfo = fromDisk.cacheInfo();
		const fromMemory = mt.db().Content(jsonlContent, "auto-cache-content.jsonl").AutoConvert(true).Cache("auto").CacheDir(cacheDir).Build();
		const memoryInfo = fromMemory.cacheInfo();
		fromMemory.close();
		const afterMemoryClose = fromDisk.cacheInfo();
		fromDisk.close();
		const rebuilt = mt.db().Content(changedContent, "auto-cache-content.jsonl").AutoConvert(true).Cache("auto").CacheDir(cacheDir).Build();
		const rebuiltInfo = rebuilt.cacheInfo();
		rebuilt.close();
		JSON.stringify({
			warmMode: warmInfo.mode,
			warmHit: warmInfo.hit,
			diskMode: diskInfo.mode,
			diskHit: diskInfo.hit,
			diskRefs: diskInfo.refCount,
			memoryHit: memoryInfo.hit,
			memoryRefs: memoryInfo.refCount,
			afterMemoryCloseRefs: afterMemoryClose.refCount,
			sameKey: diskInfo.key === memoryInfo.key,
			rebuiltHit: rebuiltInfo.hit,
			rebuiltKeyDifferent: rebuiltInfo.key !== diskInfo.key
		});
	`)
	if err != nil {
		t.Fatalf("run auto cache script: %v", err)
	}
	var got struct {
		WarmMode             string `json:"warmMode"`
		WarmHit              bool   `json:"warmHit"`
		DiskMode             string `json:"diskMode"`
		DiskHit              bool   `json:"diskHit"`
		DiskRefs             int    `json:"diskRefs"`
		MemoryHit            bool   `json:"memoryHit"`
		MemoryRefs           int    `json:"memoryRefs"`
		AfterMemoryCloseRefs int    `json:"afterMemoryCloseRefs"`
		SameKey              bool   `json:"sameKey"`
		RebuiltHit           bool   `json:"rebuiltHit"`
		RebuiltKeyDifferent  bool   `json:"rebuiltKeyDifferent"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal auto cache result: %v", err)
	}
	if got.WarmMode != "disk" || got.WarmHit {
		t.Fatalf("unexpected warm disk cache info: %#v", got)
	}
	if got.DiskMode != "auto" || !got.DiskHit || got.DiskRefs != 1 {
		t.Fatalf("expected auto disk hit with one memory ref: %#v", got)
	}
	if !got.MemoryHit || got.MemoryRefs != 2 || got.AfterMemoryCloseRefs != 1 || !got.SameKey {
		t.Fatalf("expected auto memory hit/ref-count behavior: %#v", got)
	}
	if got.RebuiltHit || !got.RebuiltKeyDifferent {
		t.Fatalf("expected changed source to rebuild with different key: %#v", got)
	}
}

func TestModuleLoaderDBBuilderDiskCacheBuildsAndReusesSQLiteFile(t *testing.T) {
	cacheDir := t.TempDir()
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "disk-cache-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
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
	_ = vm.Set("cacheDir", cacheDir)
	value, err := vm.RunString(`
		const first = mt.db().Content(jsonlContent, "disk-cache-content.jsonl").AutoConvert(true).SQLiteDiskCache(cacheDir).Build();
		const firstInfo = first.cacheInfo();
		const firstCount = first.queryOne("SELECT COUNT(*) AS n FROM sessions").n;
		const rejected = first.queryResult("INSERT INTO sessions(session_id) VALUES ('bad')").error;
		first.close();
		const second = mt.db().Content(jsonlContent, "disk-cache-content.jsonl").AutoConvert(true).SQLiteDiskCache(cacheDir).Build();
		const secondInfo = second.cacheInfo();
		second.close();
		const rebuilt = mt.db().Content(jsonlContent, "disk-cache-content.jsonl").AutoConvert(true).SQLiteDiskCache(cacheDir).ForceRebuild().Build();
		const rebuiltInfo = rebuilt.cacheInfo();
		rebuilt.close();
		JSON.stringify({
			firstHit: firstInfo.hit,
			secondHit: secondInfo.hit,
			rebuiltHit: rebuiltInfo.hit,
			mode: firstInfo.mode,
			path: firstInfo.path,
			samePath: firstInfo.path === secondInfo.path && secondInfo.path === rebuiltInfo.path,
			firstCount,
			rejected
		});
	`)
	if err != nil {
		t.Fatalf("run disk cache script: %v", err)
	}
	var got struct {
		FirstHit   bool   `json:"firstHit"`
		SecondHit  bool   `json:"secondHit"`
		RebuiltHit bool   `json:"rebuiltHit"`
		Mode       string `json:"mode"`
		Path       string `json:"path"`
		SamePath   bool   `json:"samePath"`
		FirstCount int    `json:"firstCount"`
		Rejected   string `json:"rejected"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal disk cache result: %v", err)
	}
	if got.FirstHit || !got.SecondHit || got.RebuiltHit || got.Mode != "disk" || !got.SamePath || got.FirstCount != 1 || got.Rejected == "" {
		t.Fatalf("unexpected disk cache result: %#v", got)
	}
	if got.Path == "" {
		t.Fatalf("expected disk cache path: %#v", got)
	}
	if _, err := os.Stat(got.Path); err != nil {
		t.Fatalf("expected disk cache file %s: %v", got.Path, err)
	}
}

func TestModuleLoaderDBBuilderMemoryCacheReusesByCacheKey(t *testing.T) {
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "memory-cache-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
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
		const first = mt.db().Content(jsonlContent, "memory-cache-content.jsonl").AutoConvert(true).Cache("memory").Build();
		const firstInfo = first.cacheInfo();
		const second = mt.db().Content(jsonlContent, "memory-cache-content.jsonl").AutoConvert(true).Cache("memory").Build();
		const secondInfo = second.cacheInfo();
		const secondCount = second.queryOne("SELECT COUNT(*) AS n FROM sessions").n;
		second.close();
		const afterSecondClose = first.cacheInfo();
		first.close();
		const changed = mt.db().Content(jsonlContent + "\n", "memory-cache-content.jsonl").AutoConvert(true).Cache("memory").Build();
		const changedInfo = changed.cacheInfo();
		changed.close();
		JSON.stringify({
			firstHit: firstInfo.hit,
			firstMode: firstInfo.mode,
			firstRefs: firstInfo.refCount,
			secondHit: secondInfo.hit,
			secondRefs: secondInfo.refCount,
			afterSecondCloseRefs: afterSecondClose.refCount,
			sameKey: firstInfo.key === secondInfo.key,
			changedKeyDifferent: changedInfo.key !== firstInfo.key,
			secondCount
		});
	`)
	if err != nil {
		t.Fatalf("run memory cache script: %v", err)
	}
	var got struct {
		FirstHit             bool   `json:"firstHit"`
		FirstMode            string `json:"firstMode"`
		FirstRefs            int    `json:"firstRefs"`
		SecondHit            bool   `json:"secondHit"`
		SecondRefs           int    `json:"secondRefs"`
		AfterSecondCloseRefs int    `json:"afterSecondCloseRefs"`
		SameKey              bool   `json:"sameKey"`
		ChangedKeyDifferent  bool   `json:"changedKeyDifferent"`
		SecondCount          int    `json:"secondCount"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal memory cache result: %v", err)
	}
	if got.FirstHit || got.FirstMode != "memory" || got.FirstRefs != 1 {
		t.Fatalf("unexpected first cache info: %#v", got)
	}
	if !got.SecondHit || got.SecondRefs != 2 || !got.SameKey {
		t.Fatalf("expected second build to hit memory cache: %#v", got)
	}
	if got.AfterSecondCloseRefs != 1 || !got.ChangedKeyDifferent || got.SecondCount != 1 {
		t.Fatalf("unexpected memory cache release/invalidation behavior: %#v", got)
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

func TestModuleLoaderComposesDBFromSubBuilders(t *testing.T) {
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "builder-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
		{"type": "message", "timestamp": "2026-03-29T12:00:01Z", "message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "hello from builder"}}}},
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
		const sources = mt.sources().Content(jsonlContent).Name("builder-content.jsonl").Build();
		const importPolicy = mt.importPolicy().AutoConvert().Strict().Build();
		const cache = mt.cache().None().Build();
		const limits = mt.limits().Rows(25).CellChars(2000).Build();
		const db = mt.db().Sources(sources).Import(importPolicy).Cache(cache).Limits(limits).Build();
		const row = db.queryOne("SELECT session_id, turn_count FROM sessions");
		const sourceSummary = sources.Summary();
		const cacheSummary = cache.Summary();
		db.close();
		JSON.stringify({ row, sourceSummary, cacheSummary });
	`)
	if err != nil {
		t.Fatalf("run composed builder script: %v", err)
	}
	var got struct {
		Row           map[string]any   `json:"row"`
		SourceSummary []map[string]any `json:"sourceSummary"`
		CacheSummary  map[string]any   `json:"cacheSummary"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal composed builder result: %v", err)
	}
	if got.Row["session_id"] != "builder-content" {
		t.Fatalf("unexpected session row: %#v", got.Row)
	}
	if len(got.SourceSummary) != 1 || got.SourceSummary[0]["name"] != "builder-content.jsonl" {
		t.Fatalf("unexpected source summary: %#v", got.SourceSummary)
	}
	if got.CacheSummary["mode"] != "none" {
		t.Fatalf("unexpected cache summary: %#v", got.CacheSummary)
	}
}

func TestModuleLoaderDBBuilderConveniencePresets(t *testing.T) {
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
		const db = mt.db().File(fixturePath).InteractiveDefaults().Build();
		const row = db.queryOne("SELECT COUNT(*) AS n FROM sessions");
		const cacheInfo = db.cacheInfo();
		db.close();
		JSON.stringify({ sessions: row.n, cacheMode: cacheInfo.mode });
	`)
	if err != nil {
		t.Fatalf("run convenience builder script: %v", err)
	}
	var got struct {
		Sessions  int    `json:"sessions"`
		CacheMode string `json:"cacheMode"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal convenience builder result: %v", err)
	}
	if got.Sessions != 1 || got.CacheMode != "auto" {
		t.Fatalf("unexpected convenience builder result: %#v", got)
	}
}

func TestModuleLoaderImporterSavesJSONLContent(t *testing.T) {
	content := string(writeJSONLFixture(t, []map[string]any{
		{"type": "session", "id": "importer-content", "version": 3, "timestamp": "2026-03-29T12:00:00Z"},
		{"type": "message", "timestamp": "2026-03-29T12:00:01Z", "message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "save me"}}}},
	}))
	root := t.TempDir()
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
	_ = vm.Set("rootDir", root)
	value, err := vm.RunString(`
		const importer = mt.importer()
		  .Content(jsonlContent)
		  .Name("importer-content.jsonl")
		  .Into(rootDir)
		  .SessionID("saved-session")
		  .AutoDetect()
		  .Strict()
		  .Convert();
		const converted = importer.Converted();
		const saved = importer.Save();
		JSON.stringify({ converted, saved, diagnostics: importer.Diagnostics() });
	`)
	if err != nil {
		t.Fatalf("run importer script: %v", err)
	}
	var got struct {
		Converted struct {
			SessionID string `json:"sessionId"`
			Format    string `json:"format"`
			TurnCount int    `json:"turnCount"`
		} `json:"converted"`
		Saved struct {
			SessionID    string `json:"sessionId"`
			Format       string `json:"format"`
			SessionPath  string `json:"sessionPath"`
			MetadataPath string `json:"metadataPath"`
		} `json:"saved"`
		Diagnostics []map[string]any `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal importer result: %v", err)
	}
	if got.Converted.SessionID != "importer-content" || got.Converted.Format != "pi-jsonl" || got.Converted.TurnCount == 0 {
		t.Fatalf("unexpected converted summary: %#v", got.Converted)
	}
	if got.Saved.SessionID != "saved-session" || got.Saved.Format != "pi-jsonl" {
		t.Fatalf("unexpected saved result: %#v", got.Saved)
	}
	if len(got.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
	if _, err := os.Stat(got.Saved.SessionPath); err != nil {
		t.Fatalf("expected saved archive: %v", err)
	}
	if _, err := os.Stat(got.Saved.MetadataPath); err != nil {
		t.Fatalf("expected saved metadata: %v", err)
	}
}

func TestModuleLoaderQueryAndViewBuilders(t *testing.T) {
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
		const db = mt.db().File(fixturePath).Build();
		const recipe = mt.query().TranscriptRows().IncludeTools().Build();
		const viaRecipe = db.query(recipe.sql(), ...recipe.args());
		const timeline = mt.view().DB(db).Timeline().Run();
		const usage = mt.view().DB(db).TokenUsage().ByTurn().Run();
		const frames = mt.view().DB(db).TurnFrames().Run();
		db.close();
		JSON.stringify({ recipe: recipe.toJSON(), transcriptRows: viaRecipe.length, timelineRows: timeline.length, usageRows: usage.length, frameCount: frames.length });
	`)
	if err != nil {
		t.Fatalf("run query/view builder script: %v", err)
	}
	var got struct {
		Recipe struct {
			Name string `json:"name"`
		} `json:"recipe"`
		TranscriptRows int `json:"transcriptRows"`
		TimelineRows   int `json:"timelineRows"`
		UsageRows      int `json:"usageRows"`
		FrameCount     int `json:"frameCount"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal query/view result: %v", err)
	}
	if got.Recipe.Name != "transcriptRows" || got.TranscriptRows == 0 || got.TimelineRows == 0 || got.UsageRows == 0 || got.FrameCount == 0 {
		t.Fatalf("unexpected query/view result: %#v", got)
	}
}

func TestModuleLoaderSessionBuilderViews(t *testing.T) {
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
		const session = mt.session().File(fixturePath).InteractiveCache().Open();
		const summary = session.summary();
		const transcript = session.view().Transcript().IncludeTools().Run();
		const timeline = session.view().Timeline().Run();
		const usage = session.view().TokenUsage().ByTurn().Run();
		const queried = session.query("SELECT COUNT(*) AS n FROM turns")[0].n;
		session.close();
		JSON.stringify({ id: session.id(), summaryId: summary.session_id, transcriptRows: transcript.length, timelineRows: timeline.length, usageRows: usage.length, queried });
	`)
	if err != nil {
		t.Fatalf("run session builder script: %v", err)
	}
	var got struct {
		ID             string `json:"id"`
		SummaryID      string `json:"summaryId"`
		TranscriptRows int    `json:"transcriptRows"`
		TimelineRows   int    `json:"timelineRows"`
		UsageRows      int    `json:"usageRows"`
		Queried        int    `json:"queried"`
	}
	if err := json.Unmarshal([]byte(value.String()), &got); err != nil {
		t.Fatalf("unmarshal session result: %v", err)
	}
	if got.ID == "" || got.ID != got.SummaryID || got.TranscriptRows == 0 || got.TimelineRows == 0 || got.UsageRows == 0 || got.Queried == 0 {
		t.Fatalf("unexpected session result: %#v", got)
	}
}
