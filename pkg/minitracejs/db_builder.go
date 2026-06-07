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
	ctx     context.Context
	backend string
	storage string
	query   minitracedb.QueryOptions
	sources []dbSource
	errors  []string
}

type dbSource struct {
	Kind string `json:"kind"`
	Path string `json:"path,omitempty"`
}

type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type DBHandle struct {
	db      *sql.DB
	runner  *minitracedb.QueryRunner
	schema  minitracedb.SchemaDescriptor
	sources []dbSource
}

func NewDBBuilder(ctx context.Context) *DBBuilder {
	if ctx == nil {
		ctx = context.Background()
	}
	return &DBBuilder{ctx: ctx, backend: "sqlite", storage: "memory", query: minitracedb.DefaultQueryOptions()}
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
	_ = obj.Set("Glob", func(pattern string) *goja.Object {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			b.errors = append(b.errors, fmt.Sprintf("invalid glob %q: %v", pattern, err))
		} else {
			sort.Strings(paths)
			b.addFiles(paths)
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
	b.sources = append(b.sources, dbSource{Kind: "file", Path: path})
}

func (b *DBBuilder) addFiles(paths []string) {
	for _, path := range paths {
		b.addFile(path)
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
		if strings.HasSuffix(path, ".minitrace.json") {
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
	for _, source := range b.sources {
		switch source.Kind {
		case "file":
			session, err := minitracedb.LoadSessionFile(source.Path)
			if err != nil {
				_ = db.Close()
				return nil, err
			}
			if err := minitracedb.MaterializeSession(b.ctx, db, session, minitracedb.MaterializeOptions{SourcePath: source.Path}); err != nil {
				_ = db.Close()
				return nil, err
			}
		default:
			_ = db.Close()
			return nil, fmt.Errorf("unsupported source kind %q", source.Kind)
		}
	}
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), b.query)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBHandle{db: db, runner: runner, schema: minitracedb.Schema(), sources: append([]dbSource(nil), b.sources...)}, nil
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
	_ = obj.Set("diagnostics", func() []map[string]any { return []map[string]any{} })
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
