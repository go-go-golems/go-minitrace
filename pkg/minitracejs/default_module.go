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
	return `The minitrace module opens local transcript archives and runs fluent query plans or tagged SQL.

Use require("minitrace") in a Goja host. Build a source selection with mt.sources(), open it with mt.db(), and execute a local plan with db.execute(plan). Use mt.importer() to convert raw transcripts and mt.native() to inspect native evidence.`
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
