package minitracejs

import (
	"context"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type module struct{}

var _ modules.NativeModule = (*module)(nil)
var _ modules.TypeScriptDeclarer = (*module)(nil)

func (*module) Name() string { return ModuleName }

func (*module) Doc() string {
	return `The minitrace module loads transcript/session archives and exposes query, view, session, import, cache, source, and SQL helper builders to JavaScript.

Use require("minitrace") from a go-go-goja runtime after linking this package into the host binary. Hand-built hosts typically use explicit sources such as mt.session().File(...), mt.db().File(...), mt.db().Content(...), or mt.sources().Glob(...). Generated xgoja hosts can still use the provider package when they need command-scoped runtime settings.`
}

func (*module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	NewLoader(context.Background(), nil, "", RuntimeSettings{})(vm, moduleObj)
}

func (*module) TypeScriptModule() *spec.Module {
	return TypeScriptModule()
}

func init() {
	modules.Register(&module{})
}
