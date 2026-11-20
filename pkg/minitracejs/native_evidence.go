package minitracejs

import (
	"context"
	"encoding/json"
	"math"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/nativeevidence"
)

// nativeEvidenceBuilderObject is opt-in: it does not modify normalized archives
// or use DB caches. Each operation opens and closes the selected native file.
func nativeEvidenceBuilderObject(ctx context.Context, vm *goja.Runtime) *goja.Object {
	return nativeEvidenceObject(ctx, vm, "", nativeevidence.Options{})
}

func nativeEvidenceObject(ctx context.Context, vm *goja.Runtime, path string, opts nativeevidence.Options) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("file", func(value string) *goja.Object { return nativeEvidenceObject(ctx, vm, value, opts) })
	positive := func(call goja.FunctionCall) int64 {
		v := call.Argument(0).ToFloat()
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || math.Trunc(v) != v || v > float64(1<<53-1) {
			panic(vm.NewTypeError("native limit must be a positive safe integer"))
		}
		return int64(v)
	}
	_ = obj.Set("maxRecordBytes", func(call goja.FunctionCall) goja.Value {
		n := positive(call)
		if uint64(n) > uint64(^uint(0)>>1) {
			panic(vm.NewTypeError("record limit exceeds platform integer range"))
		}
		next := opts
		next.MaxRecordBytes = int(n)
		return nativeEvidenceObject(ctx, vm, path, next)
	})
	_ = obj.Set("maxIndexBytes", func(call goja.FunctionCall) goja.Value {
		next := opts
		next.MaxIndexBytes = positive(call)
		return nativeEvidenceObject(ctx, vm, path, next)
	})
	checkPath := func() {
		if path == "" {
			panic(vm.NewTypeError("native file(path) is required"))
		}
	}
	_ = obj.Set("forEach", func(call goja.FunctionCall) goja.Value {
		checkPath()
		callback, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.NewTypeError("forEach requires a callback; return false to stop"))
		}
		stats, err := nativeevidence.ScanFile(ctx, path, opts, func(record nativeevidence.Record) (bool, error) {
			value, err := callback(goja.Undefined(), nativeEvidenceValue(vm, record))
			if err != nil {
				return false, err
			}
			return value.Export() != false, nil
		})
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return nativeEvidenceValue(vm, stats)
	})
	_ = obj.Set("inspect", func() goja.Value {
		checkPath()
		report, err := nativeevidence.InspectFile(ctx, path, opts)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return nativeEvidenceValue(vm, report)
	})
	return obj
}

// JSON conversion preserves the public lower-camel field names independently of
// runtime struct mapper settings. No transcript content is evaluated as JS.
func nativeEvidenceValue(vm *goja.Runtime, value any) goja.Value {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(vm.NewGoError(err))
	}
	var plain any
	if err = json.Unmarshal(raw, &plain); err != nil {
		panic(vm.NewGoError(err))
	}
	return vm.ToValue(plain)
}
