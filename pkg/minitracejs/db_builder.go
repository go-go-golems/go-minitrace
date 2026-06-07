package minitracejs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

type DBBuilder struct {
	ctx                 context.Context
	backend             string
	storage             string
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
}

func NewDBBuilder(ctx context.Context) *DBBuilder {
	return NewDBBuilderWithRuntime(ctx, RuntimeSettings{})
}

func NewDBBuilderWithRuntime(ctx context.Context, runtimeSettings RuntimeSettings) *DBBuilder {
	if ctx == nil {
		ctx = context.Background()
	}
	return &DBBuilder{ctx: ctx, backend: "sqlite", storage: "memory", query: minitracedb.DefaultQueryOptions(), runtimeArchiveGlobs: append([]string(nil), runtimeSettings.ArchiveGlob...), strictConversion: true}
}

func builderObject(vm *goja.Runtime, b *DBBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("SQLiteMemory", func() *goja.Object {
		b.backend = "sqlite"
		b.storage = "memory"
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
	_ = obj.Set("sources", func() []map[string]any { return toPlainSlice(b.sources) })
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
	if b.storage != "memory" {
		errs = append(errs, "only memory storage is implemented in phase 1")
	}
	return ValidationResult{Valid: len(errs) == 0, Errors: errs}
}

func (b *DBBuilder) Build() (*DBHandle, error) {
	validation := b.Validate()
	if !validation.Valid {
		return nil, fmt.Errorf("minitrace.db: %v", validation.Errors)
	}
	db, err := minitracedb.OpenSQLiteMemory(b.ctx, "minitrace")
	if err != nil {
		return nil, err
	}
	if err := minitracedb.CreateSchema(b.ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	diagnostics := []minitracedb.ConversionDiagnostic{}
	for _, source := range b.sources {
		loaded, err := b.loadSource(source)
		if err != nil {
			diagnostic := minitracedb.ConversionDiagnostic{Source: firstNonEmpty(source.Name, source.Path, source.Kind), Severity: "error", Message: err.Error()}
			diagnostics = append(diagnostics, diagnostic)
			if b.strictConversion {
				_ = db.Close()
				return nil, err
			}
			continue
		}
		diagnostics = append(diagnostics, loaded.Diagnostics...)
		if err := minitracedb.MaterializeSession(b.ctx, db, loaded.Session, minitracedb.MaterializeOptions{SourcePath: source.Path}); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...), diagnostics: diagnostics}, nil
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

func handleObject(vm *goja.Runtime, h *DBHandle) *goja.Object {
	obj := vm.NewObject()
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
	_ = obj.Set("close", func() error {
		if h == nil || h.db == nil {
			return nil
		}
		return h.db.Close()
	})
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
