#!/usr/bin/env bash
set -euo pipefail

WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)"
cd "$WORKSPACE_ROOT"

probe_go="$(mktemp -t minitrace-issue20-probe-XXXXXX.go)"
trap 'rm -f "$probe_go"' EXIT

cat > "$probe_go" <<'GO'
package main

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-minitrace/pkg/minitracejs"
	_ "github.com/go-go-golems/goja-text/pkg/template"
)

func canRequire(factory *engine.RuntimeFactory) (string, error) {
	rt, err := factory.NewRuntime(
		engine.WithStartupContext(context.Background()),
		engine.WithLifetimeContext(context.Background()),
	)
	if err != nil {
		return "", err
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "probe", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, runErr := vm.RunString(`
			function canRequire(name) {
				try { require(name); return true; } catch (e) { return String(e); }
			}
			JSON.stringify({
				minitrace: canRequire("minitrace"),
				fs: canRequire("fs"),
				template: canRequire("template")
			});
		`)
		if runErr != nil {
			return nil, runErr
		}
		return v.String(), nil
	})
	if err != nil {
		return "", err
	}
	return ret.(string), nil
}

func main() {
	plain, err := engine.NewRuntimeFactoryBuilder().Build()
	if err != nil {
		panic(err)
	}
	explicit, err := engine.NewRuntimeFactoryBuilder().
		WithModules(engine.NativeModuleRegistrar{
			ModuleName: minitracejs.ModuleName,
			Loader:     minitracejs.NewLoader(context.Background(), nil, "probe", minitracejs.RuntimeSettings{}),
		}).
		Build()
	if err != nil {
		panic(err)
	}
	add, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareAdd("minitrace")).Build()
	if err != nil {
		fmt.Printf("middleware-add build error: %v\n", err)
	} else {
		s, runErr := canRequire(add)
		fmt.Printf("middleware-add=%s err=%v\n", s, runErr)
	}

	s, runErr := canRequire(plain)
	fmt.Printf("plain=%s err=%v\n", s, runErr)
	s, runErr = canRequire(explicit)
	fmt.Printf("explicit=%s err=%v\n", s, runErr)
}
GO

go run "$probe_go"
