package minitracejs

import (
	_ "embed"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

//go:embed minitrace.d.ts
var declarations string

func TypeScriptModule() *spec.Module {
	lines := strings.Split(strings.TrimSpace(declarations), "\n")
	for i, line := range lines {
		// xgoja wraps these lines in an ambient declare module block.
		lines[i] = strings.TrimPrefix(line, "declare ")
	}
	return &spec.Module{
		Name:        ModuleName,
		Description: "Local fluent and tagged SQL query plans for transcript archives.",
		RawDTS:      lines,
	}
}
