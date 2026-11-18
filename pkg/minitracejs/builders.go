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
	errors  []string
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

}

func (b *SourceSetBuilder) Validate() ValidationResult {
	errs := append([]string(nil), b.errors...)
	if len(b.sources) == 0 {
		errs = append(errs, "source selection is empty")
	}
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

type CachePolicy struct {
	Mode         string `json:"mode"`
	Dir          string `json:"dir,omitempty"`
	ForceRebuild bool   `json:"forceRebuild,omitempty"`
}

type QueryLimits struct {
	MaxRows        int           `json:"maxRows,omitempty"`
	MaxColumns     int           `json:"maxColumns,omitempty"`
	MaxCellChars   int           `json:"maxCellChars,omitempty"`
	MaxCellBytes   int           `json:"maxCellBytes,omitempty"`
	StrictEvidence bool          `json:"strictEvidence,omitempty"`
	Timeout        time.Duration `json:"-"`
	TimeoutMs      int           `json:"timeoutMs,omitempty"`
	RequireOrderBy bool          `json:"requireOrderBy,omitempty"`
}

func optionalBool(call goja.FunctionCall, defaultValue bool) bool {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) || goja.IsNull(call.Argument(0)) {
		return defaultValue
	}
	return call.Argument(0).ToBoolean()
}

func baseName(path string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}
