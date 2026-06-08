package minitracejs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

type DBBuilder struct {
	ctx                 context.Context
	backend             string
	storage             string
	cacheMode           string
	cacheDir            string
	forceRebuild        bool
	query               minitracedb.QueryOptions
	sources             []dbSource
	runtimeArchiveGlobs []string
	autoConvert         bool
	strictConversion    bool
	errors              []string
}

type dbSource struct {
	Kind    string `json:"kind"`
	Path    string `json:"path,omitempty"`
	Name    string `json:"name,omitempty"`
	Content string `json:"-"`
}

type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type DBHandle struct {
	db          *sql.DB
	runner      *minitracedb.QueryRunner
	schema      minitracedb.SchemaDescriptor
	sources     []dbSource
	diagnostics []minitracedb.ConversionDiagnostic
	cacheKey    minitracedb.CacheKey
	cacheMode   string
	cacheHit    bool
	cachePath   string
	closed      bool
}

type DBCacheInfo struct {
	Key       string                          `json:"key"`
	Mode      string                          `json:"mode"`
	Hit       bool                            `json:"hit"`
	RefCount  int                             `json:"refCount"`
	CreatedAt string                          `json:"createdAt,omitempty"`
	LastUsed  string                          `json:"lastUsed,omitempty"`
	Path      string                          `json:"path,omitempty"`
	Options   minitracedb.CacheKeyOptions     `json:"options"`
	Sources   []minitracedb.SourceFingerprint `json:"sources"`
}

var globalMemoryCache = &memoryDBCache{entries: map[string]*memoryDBCacheEntry{}}

type memoryDBCache struct {
	mu      sync.Mutex
	entries map[string]*memoryDBCacheEntry
}

type memoryDBCacheEntry struct {
	db        *sql.DB
	key       minitracedb.CacheKey
	refs      int
	createdAt time.Time
	lastUsed  time.Time
}

func NewDBBuilder(ctx context.Context) *DBBuilder {
	return NewDBBuilderWithRuntime(ctx, RuntimeSettings{})
}

func NewDBBuilderWithRuntime(ctx context.Context, runtimeSettings RuntimeSettings) *DBBuilder {
	if ctx == nil {
		ctx = context.Background()
	}
	return &DBBuilder{ctx: ctx, backend: "sqlite", storage: "memory", cacheMode: "none", query: minitracedb.DefaultQueryOptions(), runtimeArchiveGlobs: append([]string(nil), runtimeSettings.ArchiveGlob...), strictConversion: true}
}

func builderObject(vm *goja.Runtime, b *DBBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("SQLiteMemory", func() *goja.Object {
		b.backend = "sqlite"
		b.storage = "memory"
		return builderObject(vm, b)
	})
	_ = obj.Set("SQLiteDiskCache", func(call goja.FunctionCall) goja.Value {
		b.backend = "sqlite"
		b.storage = "disk-cache"
		b.cacheMode = "disk"
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
			b.cacheDir = strings.TrimSpace(call.Argument(0).String())
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("File", func(path string) *goja.Object {
		b.addFile(path)
		return builderObject(vm, b)
	})
	_ = obj.Set("Files", func(paths []string) *goja.Object {
		b.addFiles(paths)
		return builderObject(vm, b)
	})
	_ = obj.Set("Archive", func(path string) *goja.Object {
		b.addFile(path)
		return builderObject(vm, b)
	})
	_ = obj.Set("Content", func(call goja.FunctionCall) goja.Value {
		content := call.Argument(0).String()
		name := "content"
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) && !goja.IsNull(call.Argument(1)) {
			arg := call.Argument(1)
			if exported := arg.Export(); exported != nil {
				switch typed := exported.(type) {
				case string:
					name = typed
				case map[string]any:
					if value, ok := typed["name"].(string); ok && value != "" {
						name = value
					}
				}
			}
		}
		b.addContent(content, name)
		return builderObject(vm, b)
	})
	_ = obj.Set("Glob", func(pattern string) *goja.Object {
		b.addGlob(pattern)
		return builderObject(vm, b)
	})
	_ = obj.Set("RuntimeArchives", func() *goja.Object {
		if len(b.runtimeArchiveGlobs) == 0 {
			b.errors = append(b.errors, "runtime archive glob is not configured")
		} else {
			for _, pattern := range b.runtimeArchiveGlobs {
				b.addGlob(pattern)
			}
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("Dir", func(path string) *goja.Object {
		paths, err := collectMinitraceFiles(path)
		if err != nil {
			b.errors = append(b.errors, err.Error())
		} else {
			b.addFiles(paths)
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("MaxRows", func(n int) *goja.Object {
		if n <= 0 {
			b.errors = append(b.errors, "maxRows must be > 0")
		} else {
			b.query.MaxRows = n
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("MaxColumns", func(n int) *goja.Object {
		if n <= 0 {
			b.errors = append(b.errors, "maxColumns must be > 0")
		} else {
			b.query.MaxColumns = n
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("MaxCellChars", func(n int) *goja.Object {
		if n <= 0 {
			b.errors = append(b.errors, "maxCellChars must be > 0")
		} else {
			b.query.MaxCellChars = n
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("Timeout", func(ms int) *goja.Object {
		if ms <= 0 {
			b.errors = append(b.errors, "timeout must be > 0")
		} else {
			b.query.Timeout = time.Duration(ms) * time.Millisecond
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("RequireOrderBy", func(enabled bool) *goja.Object {
		b.query.RequireOrderBy = enabled
		return builderObject(vm, b)
	})
	_ = obj.Set("AutoConvert", func(enabled bool) *goja.Object {
		b.autoConvert = enabled
		return builderObject(vm, b)
	})
	_ = obj.Set("StrictConversion", func(enabled bool) *goja.Object {
		b.strictConversion = enabled
		return builderObject(vm, b)
	})
	_ = obj.Set("Sources", func(sources *SourceSet) *goja.Object {
		b.applySourceSet(sources)
		return builderObject(vm, b)
	})
	_ = obj.Set("Import", func(policy *ImportPolicy) *goja.Object {
		b.applyImportPolicy(policy)
		return builderObject(vm, b)
	})
	_ = obj.Set("Limits", func(limits *QueryLimits) *goja.Object {
		b.applyQueryLimits(limits)
		return builderObject(vm, b)
	})
	_ = obj.Set("Cache", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) || goja.IsNull(call.Argument(0)) {
			b.applyCachePolicy(&CachePolicy{Mode: "none"})
			return builderObject(vm, b)
		}
		if policy, ok := call.Argument(0).Export().(*CachePolicy); ok {
			b.applyCachePolicy(policy)
			return builderObject(vm, b)
		}
		b.applyCacheMode(call.Argument(0).String())
		return builderObject(vm, b)
	})
	_ = obj.Set("CacheAuto", func(call goja.FunctionCall) goja.Value {
		b.applyCachePolicy(&CachePolicy{Mode: "auto", Dir: optionalString(call)})
		return builderObject(vm, b)
	})
	_ = obj.Set("QueryCommandDefaults", func() *goja.Object {
		b.autoConvert = true
		b.strictConversion = true
		return builderObject(vm, b)
	})
	_ = obj.Set("InteractiveDefaults", func(call goja.FunctionCall) goja.Value {
		b.autoConvert = true
		b.strictConversion = true
		b.applyCachePolicy(&CachePolicy{Mode: "auto", Dir: optionalString(call)})
		return builderObject(vm, b)
	})
	_ = obj.Set("CacheDir", func(path string) *goja.Object {
		b.cacheDir = strings.TrimSpace(path)
		return builderObject(vm, b)
	})
	_ = obj.Set("ForceRebuild", func(call goja.FunctionCall) goja.Value {
		b.forceRebuild = true
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
			b.forceRebuild = call.Argument(0).ToBoolean()
		}
		return builderObject(vm, b)
	})
	_ = obj.Set("sources", func() []map[string]any { return toPlainSlice(b.sources) })
	_ = obj.Set("CacheKey", func() (map[string]any, error) {
		key, err := b.CacheKey()
		if err != nil {
			return nil, err
		}
		return toPlainMap(key), nil
	})
	_ = obj.Set("Validate", func() ValidationResult { return b.Validate() })
	_ = obj.Set("Build", func() (*goja.Object, error) {
		h, err := b.Build()
		if err != nil {
			return nil, err
		}
		return handleObject(vm, h), nil
	})
	return obj
}

func (b *DBBuilder) addFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		b.errors = append(b.errors, "file path must not be empty")
		return
	}
	for _, source := range b.sources {
		if source.Kind == "file" && source.Path == path {
			return
		}
	}
	b.sources = append(b.sources, dbSource{Kind: "file", Path: path, Name: filepath.Base(path)})
}

func (b *DBBuilder) addFiles(paths []string) {
	for _, path := range paths {
		b.addFile(path)
	}
}

func (b *DBBuilder) addGlob(pattern string) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		b.errors = append(b.errors, fmt.Sprintf("invalid glob %q: %v", pattern, err))
		return
	}
	sort.Strings(paths)
	b.addFiles(paths)
}

func (b *DBBuilder) addContent(content, name string) {
	if strings.TrimSpace(content) == "" {
		b.errors = append(b.errors, "content source must not be empty")
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "content"
	}
	b.sources = append(b.sources, dbSource{Kind: "content", Name: name, Content: content})
}
func (b *DBBuilder) applySourceSet(set *SourceSet) {
	if set == nil {
		b.errors = append(b.errors, "source set must not be nil")
		return
	}
	for _, source := range set.sources {
		switch source.Kind {
		case "file":
			b.addFile(source.Path)
		case "content":
			b.addContent(source.Content, source.Name)
		case "dir":
			paths, err := collectMinitraceFiles(source.Path)
			if err != nil {
				b.errors = append(b.errors, err.Error())
			} else {
				b.addFiles(paths)
			}
		case "glob":
			b.addGlob(source.Path)
		case "runtime":
			if len(b.runtimeArchiveGlobs) == 0 {
				b.errors = append(b.errors, "runtime archive glob is not configured")
			} else {
				for _, pattern := range b.runtimeArchiveGlobs {
					b.addGlob(pattern)
				}
			}
		default:
			b.errors = append(b.errors, fmt.Sprintf("unsupported source kind %q", source.Kind))
		}
	}
}

func (b *DBBuilder) applyImportPolicy(policy *ImportPolicy) {
	if policy == nil {
		b.errors = append(b.errors, "import policy must not be nil")
		return
	}
	b.autoConvert = policy.AutoConvert
	b.strictConversion = policy.Strict
}

func (b *DBBuilder) applyQueryLimits(limits *QueryLimits) {
	if limits == nil {
		b.errors = append(b.errors, "query limits must not be nil")
		return
	}
	if limits.MaxRows > 0 {
		b.query.MaxRows = limits.MaxRows
	}
	if limits.MaxColumns > 0 {
		b.query.MaxColumns = limits.MaxColumns
	}
	if limits.MaxCellChars > 0 {
		b.query.MaxCellChars = limits.MaxCellChars
	}
	if limits.Timeout > 0 {
		b.query.Timeout = limits.Timeout
	}
	b.query.RequireOrderBy = limits.RequireOrderBy
}

func (b *DBBuilder) applyCachePolicy(policy *CachePolicy) {
	if policy == nil {
		b.errors = append(b.errors, "cache policy must not be nil")
		return
	}
	b.applyCacheMode(policy.Mode)
	if strings.TrimSpace(policy.Dir) != "" {
		b.cacheDir = strings.TrimSpace(policy.Dir)
	}
	b.forceRebuild = policy.ForceRebuild
}

func (b *DBBuilder) applyCacheMode(mode string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none":
		b.cacheMode = "none"
		if b.storage == "disk-cache" {
			b.storage = "memory"
		}
	case "memory":
		b.cacheMode = "memory"
		b.storage = "memory"
	case "disk":
		b.cacheMode = "disk"
		b.storage = "disk-cache"
	case "auto":
		b.cacheMode = "auto"
		b.storage = "disk-cache"
	default:
		b.errors = append(b.errors, fmt.Sprintf("unsupported cache mode %q", mode))
	}
}

func collectMinitraceFiles(root string) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("directory path must not be empty")
	}
	var paths []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".minitrace.json") || strings.HasSuffix(path, ".jsonl") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan directory %s: %w", root, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func (b *DBBuilder) Validate() ValidationResult {
	errs := append([]string(nil), b.errors...)
	if b.backend != "sqlite" {
		errs = append(errs, "only sqlite backend is implemented in phase 1")
	}
	if b.storage != "memory" && b.storage != "disk-cache" {
		errs = append(errs, "only memory and disk-cache storage are implemented")
	}
	if b.cacheMode != "none" && b.cacheMode != "memory" && b.cacheMode != "disk" && b.cacheMode != "auto" {
		errs = append(errs, fmt.Sprintf("unsupported cache mode %q", b.cacheMode))
	}
	return ValidationResult{Valid: len(errs) == 0, Errors: errs}
}

func (b *DBBuilder) CacheKey() (minitracedb.CacheKey, error) {
	validation := b.Validate()
	if !validation.Valid {
		return minitracedb.CacheKey{}, fmt.Errorf("minitrace.db: %v", validation.Errors)
	}
	fingerprints := make([]minitracedb.SourceFingerprint, 0, len(b.sources))
	for _, source := range b.sources {
		switch source.Kind {
		case "file":
			fp, err := minitracedb.FingerprintFile(source.Path)
			if err != nil {
				return minitracedb.CacheKey{}, err
			}
			fingerprints = append(fingerprints, fp)
		case "content":
			fingerprints = append(fingerprints, minitracedb.FingerprintContent(source.Name, []byte(source.Content)))
		default:
			return minitracedb.CacheKey{}, fmt.Errorf("unsupported source kind %q", source.Kind)
		}
	}
	opts := minitracedb.DefaultCacheKeyOptions()
	opts.Backend = b.backend
	opts.Storage = b.storage
	opts.AutoConvert = b.autoConvert
	return minitracedb.ComputeCacheKey(fingerprints, opts)
}

func (b *DBBuilder) Build() (*DBHandle, error) {
	validation := b.Validate()
	if !validation.Valid {
		return nil, fmt.Errorf("minitrace.db: %v", validation.Errors)
	}
	cacheKey, err := b.CacheKey()
	if err != nil {
		return nil, err
	}
	if b.cacheMode == "memory" {
		return b.buildMemoryCached(cacheKey)
	}
	if b.cacheMode == "disk" {
		return b.buildDiskCached(cacheKey)
	}
	if b.cacheMode == "auto" {
		return b.buildAutoCached(cacheKey)
	}
	db, diagnostics, err := b.materializeFreshDB()
	if err != nil {
		return nil, err
	}
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), diagnostics: diagnostics, cacheKey: cacheKey, cacheMode: "none"}, nil
}

func (b *DBBuilder) buildMemoryCached(cacheKey minitracedb.CacheKey) (*DBHandle, error) {
	now := time.Now().UTC()
	globalMemoryCache.mu.Lock()
	entry := globalMemoryCache.entries[cacheKey.Key]
	if entry != nil {
		entry.refs++
		entry.lastUsed = now
		db := entry.db
		globalMemoryCache.mu.Unlock()
		runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
		if err != nil {
			releaseMemoryCache(cacheKey.Key)
			return nil, err
		}
		return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), cacheKey: cacheKey, cacheMode: "memory", cacheHit: true}, nil
	}
	globalMemoryCache.mu.Unlock()

	db, diagnostics, err := b.materializeFreshDB()
	if err != nil {
		return nil, err
	}
	globalMemoryCache.mu.Lock()
	if existing := globalMemoryCache.entries[cacheKey.Key]; existing != nil {
		existing.refs++
		existing.lastUsed = now
		dbToClose := db
		db = existing.db
		globalMemoryCache.mu.Unlock()
		_ = dbToClose.Close()
		runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
		if err != nil {
			releaseMemoryCache(cacheKey.Key)
			return nil, err
		}
		return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), diagnostics: diagnostics, cacheKey: cacheKey, cacheMode: "memory", cacheHit: true}, nil
	}
	globalMemoryCache.entries[cacheKey.Key] = &memoryDBCacheEntry{db: db, key: cacheKey, refs: 1, createdAt: now, lastUsed: now}
	globalMemoryCache.mu.Unlock()
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
	if err != nil {
		releaseMemoryCache(cacheKey.Key)
		return nil, err
	}
	return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), diagnostics: diagnostics, cacheKey: cacheKey, cacheMode: "memory", cacheHit: false}, nil
}

func (b *DBBuilder) buildDiskCached(cacheKey minitracedb.CacheKey) (*DBHandle, error) {
	cacheDir, err := b.resolvedCacheDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(cacheDir, cacheKey.Key+".sqlite")
	cacheHit := false
	diagnostics := []minitracedb.ConversionDiagnostic(nil)
	if !b.forceRebuild {
		if _, err := os.Stat(path); err == nil {
			cacheHit = true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat disk cache %s: %w", path, err)
		}
	}
	if !cacheHit {
		var err error
		diagnostics, err = b.buildDiskCacheFile(cacheKey, path)
		if err != nil {
			return nil, err
		}
	}
	db, err := minitracedb.OpenSQLiteReadOnly(b.ctx, path)
	if err != nil {
		return nil, err
	}
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), diagnostics: diagnostics, cacheKey: cacheKey, cacheMode: "disk", cacheHit: cacheHit, cachePath: path}, nil
}

func (b *DBBuilder) buildAutoCached(cacheKey minitracedb.CacheKey) (*DBHandle, error) {
	cacheDir, err := b.resolvedCacheDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(cacheDir, cacheKey.Key+".sqlite")
	if !b.forceRebuild {
		if db, ok := acquireMemoryCache(cacheKey.Key); ok {
			runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
			if err != nil {
				releaseMemoryCache(cacheKey.Key)
				return nil, err
			}
			return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), cacheKey: cacheKey, cacheMode: "auto", cacheHit: true, cachePath: path}, nil
		}
	}
	cacheHit := false
	diagnostics := []minitracedb.ConversionDiagnostic(nil)
	if !b.forceRebuild {
		if _, err := os.Stat(path); err == nil {
			cacheHit = true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat disk cache %s: %w", path, err)
		}
	}
	if !cacheHit {
		var err error
		diagnostics, err = b.buildDiskCacheFile(cacheKey, path)
		if err != nil {
			return nil, err
		}
	}
	db, err := minitracedb.OpenSQLiteReadOnly(b.ctx, path)
	if err != nil {
		return nil, err
	}
	db, cacheHit = installMemoryCache(cacheKey, db, cacheHit)
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
	if err != nil {
		releaseMemoryCache(cacheKey.Key)
		return nil, err
	}
	return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), diagnostics: diagnostics, cacheKey: cacheKey, cacheMode: "auto", cacheHit: cacheHit, cachePath: path}, nil
}

func (b *DBBuilder) buildDiskCacheFile(cacheKey minitracedb.CacheKey, path string) ([]minitracedb.ConversionDiagnostic, error) {
	cacheDir := filepath.Dir(path)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create disk cache directory: %w", err)
	}
	tmp, err := os.CreateTemp(cacheDir, cacheKey.Key+"-*.tmp.sqlite")
	if err != nil {
		return nil, fmt.Errorf("create disk cache temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)
	builtDB, diagnostics, err := b.materializeFreshDBAt(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	if err := builtDB.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("close built disk cache db: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("install disk cache db: %w", err)
	}
	return diagnostics, nil
}

func (b *DBBuilder) resolvedCacheDir() (string, error) {
	if strings.TrimSpace(b.cacheDir) != "" {
		return b.cacheDir, nil
	}
	root, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(root) == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "go-minitrace", "minitracedb"), nil
}

func (b *DBBuilder) materializeFreshDB() (*sql.DB, []minitracedb.ConversionDiagnostic, error) {
	db, err := minitracedb.OpenSQLiteMemory(b.ctx, "minitrace")
	if err != nil {
		return nil, nil, err
	}
	diagnostics, err := b.materializeInto(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return db, diagnostics, nil
}

func (b *DBBuilder) materializeFreshDBAt(path string) (*sql.DB, []minitracedb.ConversionDiagnostic, error) {
	db, err := minitracedb.OpenSQLiteFile(b.ctx, path)
	if err != nil {
		return nil, nil, err
	}
	diagnostics, err := b.materializeInto(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return db, diagnostics, nil
}

func (b *DBBuilder) materializeInto(db *sql.DB) ([]minitracedb.ConversionDiagnostic, error) {
	if err := minitracedb.CreateSchema(b.ctx, db); err != nil {
		return nil, err
	}
	diagnostics := []minitracedb.ConversionDiagnostic{}
	for _, source := range b.sources {
		loaded, err := b.loadSource(source)
		if err != nil {
			diagnostic := minitracedb.ConversionDiagnostic{Source: firstNonEmpty(source.Name, source.Path, source.Kind), Severity: "error", Message: err.Error()}
			diagnostics = append(diagnostics, diagnostic)
			if b.strictConversion {
				return nil, err
			}
			continue
		}
		diagnostics = append(diagnostics, loaded.Diagnostics...)
		if err := minitracedb.MaterializeSession(b.ctx, db, loaded.Session, minitracedb.MaterializeOptions{SourcePath: source.Path}); err != nil {
			return nil, err
		}
	}
	return diagnostics, nil
}

func (b *DBBuilder) loadSource(source dbSource) (*minitracedb.LoadedSession, error) {
	switch source.Kind {
	case "file":
		return minitracedb.LoadSessionFileAuto(source.Path, minitracedb.LoadOptions{SourcePath: source.Path, SourceName: filepath.Base(source.Path), AutoConvert: b.autoConvert})
	case "content":
		return minitracedb.LoadSessionContentAuto([]byte(source.Content), minitracedb.LoadOptions{SourceName: source.Name, AutoConvert: b.autoConvert})
	default:
		return nil, fmt.Errorf("unsupported source kind %q", source.Kind)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func acquireMemoryCache(key string) (*sql.DB, bool) {
	globalMemoryCache.mu.Lock()
	defer globalMemoryCache.mu.Unlock()
	entry := globalMemoryCache.entries[key]
	if entry == nil {
		return nil, false
	}
	entry.refs++
	entry.lastUsed = time.Now().UTC()
	return entry.db, true
}

func installMemoryCache(cacheKey minitracedb.CacheKey, db *sql.DB, hit bool) (*sql.DB, bool) {
	now := time.Now().UTC()
	globalMemoryCache.mu.Lock()
	defer globalMemoryCache.mu.Unlock()
	if existing := globalMemoryCache.entries[cacheKey.Key]; existing != nil {
		existing.refs++
		existing.lastUsed = now
		_ = db.Close()
		return existing.db, true
	}
	globalMemoryCache.entries[cacheKey.Key] = &memoryDBCacheEntry{db: db, key: cacheKey, refs: 1, createdAt: now, lastUsed: now}
	return db, hit
}

func releaseMemoryCache(key string) {
	globalMemoryCache.mu.Lock()
	defer globalMemoryCache.mu.Unlock()
	if entry := globalMemoryCache.entries[key]; entry != nil && entry.refs > 0 {
		entry.refs--
		entry.lastUsed = time.Now().UTC()
	}
}

func (h *DBHandle) CacheInfo() DBCacheInfo {
	info := DBCacheInfo{Key: h.cacheKey.Key, Mode: firstNonEmpty(h.cacheMode, "none"), Hit: h.cacheHit, Path: h.cachePath, Options: h.cacheKey.Options, Sources: append([]minitracedb.SourceFingerprint(nil), h.cacheKey.Sources...)}
	if (h.cacheMode == "disk" || h.cacheMode == "auto") && h.cachePath != "" {
		if stat, err := os.Stat(h.cachePath); err == nil {
			info.CreatedAt = stat.ModTime().UTC().Format(time.RFC3339Nano)
			info.LastUsed = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if h.cacheMode == "disk" {
			return info
		}
	}
	if h.cacheMode != "memory" && h.cacheMode != "auto" {
		return info
	}
	globalMemoryCache.mu.Lock()
	defer globalMemoryCache.mu.Unlock()
	if entry := globalMemoryCache.entries[h.cacheKey.Key]; entry != nil {
		info.RefCount = entry.refs
		info.CreatedAt = entry.createdAt.UTC().Format(time.RFC3339Nano)
		info.LastUsed = entry.lastUsed.UTC().Format(time.RFC3339Nano)
	}
	return info
}

func (h *DBHandle) Close() error {
	if h == nil || h.db == nil || h.closed {
		return nil
	}
	h.closed = true
	if h.cacheMode == "memory" || h.cacheMode == "auto" {
		releaseMemoryCache(h.cacheKey.Key)
		return nil
	}
	return h.db.Close()
}

func handleObject(vm *goja.Runtime, h *DBHandle) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("_handle", h)
	_ = obj.Set("query", func(sqlText string, args ...any) ([]map[string]any, error) {
		return h.runner.Query(context.Background(), sqlText, args...)
	})
	_ = obj.Set("queryOne", func(sqlText string, args ...any) (map[string]any, error) {
		return h.runner.QueryOne(context.Background(), sqlText, args...)
	})
	_ = obj.Set("queryResult", func(sqlText string, args ...any) (map[string]any, error) {
		result, err := h.runner.QueryResult(context.Background(), sqlText, args...)
		return toPlainMap(result), err
	})
	_ = obj.Set("schema", func() map[string]any { return toPlainMap(h.schema) })
	_ = obj.Set("tables", func() []map[string]any { return toPlainSlice(h.schema.Tables) })
	_ = obj.Set("stats", func() map[string]any {
		return map[string]any{"schemaVersion": h.schema.Version, "dialect": h.schema.Dialect, "tables": len(h.schema.Tables)}
	})
	_ = obj.Set("sources", func() []map[string]any { return toPlainSlice(h.sources) })
	_ = obj.Set("diagnostics", func() []map[string]any { return toPlainSlice(h.diagnostics) })
	_ = obj.Set("cacheInfo", func() map[string]any { return toPlainMap(h.CacheInfo()) })
	_ = obj.Set("CacheInfo", func() map[string]any { return toPlainMap(h.CacheInfo()) })
	_ = obj.Set("close", func() error { return h.Close() })
	return obj
}

func toPlainMap(v any) map[string]any {
	payload, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	out := map[string]any{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return out
}

func toPlainSlice[T any](values []T) []map[string]any {
	ret := make([]map[string]any, 0, len(values))
	for _, value := range values {
		ret = append(ret, toPlainMap(value))
	}
	return ret
}
