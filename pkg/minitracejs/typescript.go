package minitracejs

import (
	_ "embed"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

//go:embed minitrace.d.ts
var declarations string

func TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name:        ModuleName,
		Description: "Local fluent and tagged SQL query plans for transcript archives.",
		RawDTS:      strings.Split(strings.TrimSpace(declarations), "\n"),
	}
}
