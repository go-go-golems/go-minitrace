package minitracejs_test

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	_ "github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

func TestDefaultBuilderCanRequireMinitrace(t *testing.T) {
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(
		engine.WithStartupContext(context.Background()),
		engine.WithLifetimeContext(context.Background()),
	)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "require-minitrace", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const mt = require("minitrace");
			const fs = require("fs");
			JSON.stringify({
				hasSession: typeof mt.session === "function",
				hasDB: typeof mt.db === "function",
				hasSources: typeof mt.sources === "function",
				hasRuntime: typeof mt.runtime === "object",
				hasFS: typeof fs === "object" && typeof fs.writeFileSync === "function"
			});
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run require smoke: %v", err)
	}

	const want = `{"hasSession":true,"hasDB":true,"hasSources":true,"hasRuntime":true,"hasFS":true}`
	if ret != want {
		t.Fatalf("module availability = %v, want %s", ret, want)
	}
}
