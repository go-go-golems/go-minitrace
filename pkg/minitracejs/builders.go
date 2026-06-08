package minitracejs

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type SourceSet struct {
	sources []dbSource
}

type SourceSetBuilder struct {
	sources []dbSource
	last    int
	errors  []string
}

func NewSourceSetBuilder() *SourceSetBuilder {
	return &SourceSetBuilder{last: -1}
}

func sourcesBuilderObject(vm *goja.Runtime, b *SourceSetBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("File", func(path string) *goja.Object {
		b.AddFile(path)
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Archive", func(path string) *goja.Object {
		b.AddFile(path)
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Files", func(paths []string) *goja.Object {
		for _, path := range paths {
			b.AddFile(path)
		}
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Dir", func(path string) *goja.Object {
		b.sources = append(b.sources, dbSource{Kind: "dir", Path: strings.TrimSpace(path), Name: strings.TrimSpace(path)})
		b.last = len(b.sources) - 1
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Glob", func(pattern string) *goja.Object {
		b.sources = append(b.sources, dbSource{Kind: "glob", Path: strings.TrimSpace(pattern), Name: strings.TrimSpace(pattern)})
		b.last = len(b.sources) - 1
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Content", func(content string) *goja.Object {
		b.AddContent(content, "content")
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Name", func(name string) *goja.Object {
		b.NameMostRecent(name)
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("RuntimeArchives", func() *goja.Object {
		b.sources = append(b.sources, dbSource{Kind: "runtime"})
		b.last = len(b.sources) - 1
		return sourcesBuilderObject(vm, b)
	})
	_ = obj.Set("Validate", func() ValidationResult { return b.Validate() })
	_ = obj.Set("Build", func() (*SourceSet, error) { return b.Build() })
	return obj
}

func (b *SourceSetBuilder) AddFile(path string) {
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
	b.sources = append(b.sources, dbSource{Kind: "file", Path: path, Name: baseName(path)})
	b.last = len(b.sources) - 1
}

func (b *SourceSetBuilder) AddContent(content, name string) {
	if strings.TrimSpace(content) == "" {
		b.errors = append(b.errors, "content source must not be empty")
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "content"
	}
	b.sources = append(b.sources, dbSource{Kind: "content", Name: name, Content: content})
	b.last = len(b.sources) - 1
}

func (b *SourceSetBuilder) NameMostRecent(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		b.errors = append(b.errors, "source name must not be empty")
		return
	}
	if b.last < 0 || b.last >= len(b.sources) {
		b.errors = append(b.errors, "Name() requires a preceding source")
		return
	}
	b.sources[b.last].Name = name
}

func (b *SourceSetBuilder) Validate() ValidationResult {
	errs := append([]string(nil), b.errors...)
	for _, source := range b.sources {
		switch source.Kind {
		case "file", "dir", "glob":
			if strings.TrimSpace(source.Path) == "" {
				errs = append(errs, fmt.Sprintf("%s source path must not be empty", source.Kind))
			}
		case "content":
			if strings.TrimSpace(source.Content) == "" {
				errs = append(errs, "content source must not be empty")
			}
		case "runtime":
		default:
			errs = append(errs, fmt.Sprintf("unsupported source kind %q", source.Kind))
		}
	}
	return ValidationResult{Valid: len(errs) == 0, Errors: errs}
}

func (b *SourceSetBuilder) Build() (*SourceSet, error) {
	validation := b.Validate()
	if !validation.Valid {
		return nil, fmt.Errorf("minitrace.sources: %v", validation.Errors)
	}
	return &SourceSet{sources: append([]dbSource(nil), b.sources...)}, nil
}

func (s *SourceSet) Summary() []map[string]any { return toPlainSlice(s.sources) }
func (s *SourceSet) toJSON() []map[string]any  { return s.Summary() }

type ImportPolicy struct {
	AutoConvert  bool   `json:"autoConvert"`
	Strict       bool   `json:"strict"`
	ForcedFormat string `json:"forcedFormat,omitempty"`
}

type ImportPolicyBuilder struct {
	policy ImportPolicy
}

func NewImportPolicyBuilder() *ImportPolicyBuilder {
	return &ImportPolicyBuilder{policy: ImportPolicy{AutoConvert: true, Strict: true}}
}

func importPolicyBuilderObject(vm *goja.Runtime, b *ImportPolicyBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("AutoConvert", func(call goja.FunctionCall) goja.Value {
		b.policy.AutoConvert = optionalBool(call, true)
		return importPolicyBuilderObject(vm, b)
	})
	_ = obj.Set("NativeOnly", func() *goja.Object {
		b.policy.AutoConvert = false
		return importPolicyBuilderObject(vm, b)
	})
	_ = obj.Set("Strict", func(call goja.FunctionCall) goja.Value {
		b.policy.Strict = optionalBool(call, true)
		return importPolicyBuilderObject(vm, b)
	})
	_ = obj.Set("Lenient", func() *goja.Object {
		b.policy.Strict = false
		return importPolicyBuilderObject(vm, b)
	})
	_ = obj.Set("Format", func(format string) *goja.Object {
		b.policy.ForcedFormat = strings.TrimSpace(format)
		return importPolicyBuilderObject(vm, b)
	})
	_ = obj.Set("Build", func() *ImportPolicy {
		policy := b.policy
		return &policy
	})
	return obj
}

func (p *ImportPolicy) Summary() map[string]any { return toPlainMap(p) }
func (p *ImportPolicy) toJSON() map[string]any  { return p.Summary() }

type CachePolicy struct {
	Mode         string `json:"mode"`
	Dir          string `json:"dir,omitempty"`
	ForceRebuild bool   `json:"forceRebuild,omitempty"`
}

type CachePolicyBuilder struct {
	policy CachePolicy
}

func NewCachePolicyBuilder() *CachePolicyBuilder {
	return &CachePolicyBuilder{policy: CachePolicy{Mode: "none"}}
}

func cachePolicyBuilderObject(vm *goja.Runtime, b *CachePolicyBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("None", func() *goja.Object { b.policy.Mode = "none"; return cachePolicyBuilderObject(vm, b) })
	_ = obj.Set("Memory", func() *goja.Object { b.policy.Mode = "memory"; return cachePolicyBuilderObject(vm, b) })
	_ = obj.Set("Disk", func() *goja.Object { b.policy.Mode = "disk"; return cachePolicyBuilderObject(vm, b) })
	_ = obj.Set("Auto", func() *goja.Object { b.policy.Mode = "auto"; return cachePolicyBuilderObject(vm, b) })
	_ = obj.Set("Dir", func(path string) *goja.Object {
		b.policy.Dir = strings.TrimSpace(path)
		return cachePolicyBuilderObject(vm, b)
	})
	_ = obj.Set("ForceRebuild", func(call goja.FunctionCall) goja.Value {
		b.policy.ForceRebuild = optionalBool(call, true)
		return cachePolicyBuilderObject(vm, b)
	})
	_ = obj.Set("Build", func() (*CachePolicy, error) {
		policy := b.policy
		if policy.Mode == "" {
			policy.Mode = "none"
		}
		switch policy.Mode {
		case "none", "memory", "disk", "auto":
			return &policy, nil
		default:
			return nil, fmt.Errorf("unsupported cache mode %q", policy.Mode)
		}
	})
	return obj
}

func (p *CachePolicy) Summary() map[string]any { return toPlainMap(p) }
func (p *CachePolicy) toJSON() map[string]any  { return p.Summary() }

type QueryLimits struct {
	MaxRows        int           `json:"maxRows,omitempty"`
	MaxColumns     int           `json:"maxColumns,omitempty"`
	MaxCellChars   int           `json:"maxCellChars,omitempty"`
	Timeout        time.Duration `json:"-"`
	TimeoutMs      int           `json:"timeoutMs,omitempty"`
	RequireOrderBy bool          `json:"requireOrderBy,omitempty"`
}

type QueryLimitsBuilder struct {
	limits QueryLimits
}

func NewQueryLimitsBuilder() *QueryLimitsBuilder { return &QueryLimitsBuilder{} }

func queryLimitsBuilderObject(vm *goja.Runtime, b *QueryLimitsBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("Rows", func(n int) *goja.Object { b.limits.MaxRows = n; return queryLimitsBuilderObject(vm, b) })
	_ = obj.Set("Columns", func(n int) *goja.Object { b.limits.MaxColumns = n; return queryLimitsBuilderObject(vm, b) })
	_ = obj.Set("CellChars", func(n int) *goja.Object { b.limits.MaxCellChars = n; return queryLimitsBuilderObject(vm, b) })
	_ = obj.Set("TimeoutMs", func(n int) *goja.Object {
		b.limits.TimeoutMs = n
		b.limits.Timeout = time.Duration(n) * time.Millisecond
		return queryLimitsBuilderObject(vm, b)
	})
	_ = obj.Set("RequireOrderBy", func(call goja.FunctionCall) goja.Value {
		b.limits.RequireOrderBy = optionalBool(call, true)
		return queryLimitsBuilderObject(vm, b)
	})
	_ = obj.Set("Build", func() (*QueryLimits, error) {
		limits := b.limits
		if limits.MaxRows < 0 || limits.MaxColumns < 0 || limits.MaxCellChars < 0 || limits.TimeoutMs < 0 {
			return nil, fmt.Errorf("query limits must be non-negative")
		}
		return &limits, nil
	})
	return obj
}

func (l *QueryLimits) Summary() map[string]any { return toPlainMap(l) }
func (l *QueryLimits) toJSON() map[string]any  { return l.Summary() }

func optionalBool(call goja.FunctionCall, defaultValue bool) bool {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) || goja.IsNull(call.Argument(0)) {
		return defaultValue
	}
	return call.Argument(0).ToBoolean()
}

func optionalString(call goja.FunctionCall) string {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) || goja.IsNull(call.Argument(0)) {
		return ""
	}
	return strings.TrimSpace(call.Argument(0).String())
}

func baseName(path string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}
