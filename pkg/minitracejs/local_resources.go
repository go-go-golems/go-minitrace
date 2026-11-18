package minitracejs

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

func (binding *relationalBinding) sourcesBuilder(b *SourceSetBuilder) *goja.Object {
	vm := binding.vm
	obj := vm.NewObject()
	derive := func(apply func(*SourceSetBuilder)) *goja.Object {
		next := *b
		next.sources = append([]dbSource(nil), b.sources...)
		next.errors = append([]string(nil), b.errors...)
		apply(&next)
		return binding.sourcesBuilder(&next)
	}
	_ = obj.Set("files", func(paths []string) *goja.Object {
		return derive(func(next *SourceSetBuilder) {
			for _, path := range paths {
				next.AddFile(path)
			}
		})
	})
	_ = obj.Set("glob", func(pattern string) *goja.Object {
		return derive(func(next *SourceSetBuilder) {
			next.sources = append(next.sources, dbSource{Kind: "glob", Path: strings.TrimSpace(pattern)})
		})
	})
	_ = obj.Set("directory", func(path string) *goja.Object {
		return derive(func(next *SourceSetBuilder) {
			next.sources = append(next.sources, dbSource{Kind: "dir", Path: strings.TrimSpace(path)})
		})
	})
	_ = obj.Set("content", func(content string, options *goja.Object) *goja.Object {
		name := "content"
		if options != nil {
			name = firstNonEmpty(stringOption(vm, options, "name"), name)
		}
		return derive(func(next *SourceSetBuilder) { next.AddContent(content, name) })
	})
	_ = obj.Set("runtimeArchives", func() *goja.Object {
		return derive(func(next *SourceSetBuilder) {
			next.sources = append(next.sources, dbSource{Kind: "runtime"})
		})
	})
	_ = obj.Set("build", func() *goja.Object {
		sources, err := b.Build()
		binding.check(err)
		return binding.remember(sources)
	})
	return obj
}

func builderObject(vm *goja.Runtime, b *DBBuilder) *goja.Object {
	obj := vm.NewObject()
	derive := func(apply func(*DBBuilder)) *goja.Object {
		next := *b
		next.sources = append([]dbSource(nil), b.sources...)
		next.errors = append([]string(nil), b.errors...)
		apply(&next)
		return builderObject(vm, &next)
	}
	_ = obj.Set("sources", func(value goja.Value) *goja.Object {
		sources, ok := b.relation.lookup(value).(*SourceSet)
		if !ok {
			panic(vm.NewTypeError("sources requires a source selection from mt.sources().build()"))
		}
		return derive(func(next *DBBuilder) {
			next.applySourceSet(sources)
			if len(next.sources) == 0 && len(next.errors) == 0 {
				next.errors = append(next.errors, "source selection matched no archives")
			}
		})
	})
	_ = obj.Set("import", func(options *goja.Object) *goja.Object {
		return derive(func(next *DBBuilder) {
			if value := option(vm, options, "autoConvert"); value != nil && !goja.IsUndefined(value) {
				next.autoConvert = boolOption(vm, options, "autoConvert")
			}
			if value := option(vm, options, "strict"); value != nil && !goja.IsUndefined(value) {
				next.strictConversion = boolOption(vm, options, "strict")
			}
		})
	})
	_ = obj.Set("cache", func(options *goja.Object) *goja.Object {
		return derive(func(next *DBBuilder) {
			mode := stringOption(vm, options, "mode")
			next.applyCachePolicy(&CachePolicy{
				Mode: mode, Dir: stringOption(vm, options, "dir"),
				ForceRebuild: boolOption(vm, options, "forceRebuild"),
			})
			next.cacheMaxBytes = intOption(vm, options, "maxBytes")
		})
	})
	_ = obj.Set("limits", func(options *goja.Object) *goja.Object {
		return derive(func(next *DBBuilder) {
			timeout := intOption(vm, options, "timeoutMs")
			if timeout > int64((1<<63-1)/time.Millisecond) {
				panic(vm.NewTypeError("timeoutMs exceeds duration range"))
			}
			next.applyQueryLimits(&QueryLimits{
				MaxRows:        int(intOption(vm, options, "maxRows")),
				MaxColumns:     int(intOption(vm, options, "maxColumns")),
				MaxCellBytes:   int(intOption(vm, options, "maxCellBytes")),
				Timeout:        time.Duration(timeout) * time.Millisecond,
				StrictEvidence: boolOption(vm, options, "strict"),
				RequireOrderBy: boolOption(vm, options, "requireOrderBy"),
			})
		})
	})
	_ = obj.Set("cacheKey", func() (map[string]any, error) {
		key, err := b.CacheKey()
		if err != nil {
			return nil, err
		}
		return toPlainMap(key), nil
	})
	_ = obj.Set("build", func() (*goja.Object, error) {
		h, err := b.Build()
		if err != nil {
			return nil, err
		}
		h.owner, h.relation, h.ctx = queryplan.NewOwner(), b.relation, b.ctx
		return handleObject(vm, h), nil
	})
	return obj
}

func option(vm *goja.Runtime, obj *goja.Object, key string) goja.Value {
	if obj == nil || obj.ClassName() != "Object" {
		panic(vm.NewTypeError("expected an options object"))
	}
	return obj.Get(key)
}

func stringOption(vm *goja.Runtime, obj *goja.Object, key string) string {
	value := option(vm, obj, key)
	if value == nil || goja.IsUndefined(value) {
		return ""
	}
	if s, ok := value.(goja.String); ok {
		return s.String()
	}
	panic(vm.NewTypeError("%s must be a string", key))
}

func boolOption(vm *goja.Runtime, obj *goja.Object, key string) bool {
	value := option(vm, obj, key)
	if value == nil || goja.IsUndefined(value) {
		return false
	}
	if flag, ok := value.Export().(bool); ok {
		return flag
	}
	panic(vm.NewTypeError("%s must be a boolean", key))
}

func intOption(vm *goja.Runtime, obj *goja.Object, key string) int64 {
	value := option(vm, obj, key)
	if value == nil || goja.IsUndefined(value) {
		return 0
	}
	if n, ok := value.Export().(int64); ok && n > 0 && n <= 1<<53-1 {
		return n
	}
	panic(vm.NewTypeError(fmt.Sprintf("%s must be a positive safe integer", key)))
}
