package minitracejs

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

type runtimeMapper struct {
	fn      goja.Callable
	columns []queryplan.Column
}

type executablePlan struct {
	query  *queryplan.Plan
	mapper *runtimeMapper
}

func (b *relationalBinding) mappedQuery(q *queryplan.Query, fn goja.Value, options *goja.Object) *goja.Object {
	mapper, ok := goja.AssertFunction(fn)
	if !ok {
		panic(b.vm.NewTypeError("mapRows requires a callback"))
	}
	shape, _ := option(b.vm, options, "columns").(*goja.Object)
	columns := b.columnTypes(shape)
	seen := make(map[string]bool, len(columns))
	for _, col := range columns {
		key := strings.ToLower(col.Name)
		if col.Name == "" || strings.ContainsRune(col.Name, 0) || seen[key] {
			panic(b.vm.NewTypeError("invalid or duplicate mapRows output column %q", col.Name))
		}
		seen[key] = true
	}
	if len(columns) == 0 || len(columns) > 128 {
		panic(b.vm.NewTypeError("mapRows requires between 1 and 128 output columns"))
	}
	stage := &runtimeMapper{fn: mapper, columns: columns}
	obj := b.vm.NewObject()
	_ = obj.Set("build", func() *goja.Object { return b.planObject(q, stage) })
	return obj
}

func (b *relationalBinding) mapResult(ctx context.Context, h *DBHandle, result minitracedb.QueryResult, mapper *runtimeMapper) minitracedb.QueryResult {
	columns := make([]string, len(mapper.columns))
	for i, column := range mapper.columns {
		columns[i] = column.Name
	}
	b.mappingRows = true
	defer func() { b.mappingRows = false }()
	return h.runner.MapResult(ctx, result, columns, func(ctx context.Context, row map[string]any) ([]any, error) {
		return b.mapRow(ctx, row, mapper)
	})
}

func (b *relationalBinding) mapRow(ctx context.Context, row map[string]any, mapper *runtimeMapper) ([]any, error) {
	fired := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		b.vm.Interrupt(ctx.Err())
		close(fired)
	})
	defer func() {
		if !stop() {
			<-fired
			b.vm.ClearInterrupt()
		}
	}()

	// Keep invocation and property reads inside one Goja call boundary so an
	// exception or interruption from either can be turned into a row error.
	var values []any
	var schemaErr error
	call, _ := goja.AssertFunction(b.vm.ToValue(func(goja.FunctionCall) goja.Value {
		value, err := mapper.fn(goja.Undefined(), b.vm.ToValue(row))
		if err != nil {
			schemaErr = fmt.Errorf("callback failed")
			return goja.Undefined()
		}
		values, schemaErr = mappedValues(value, mapper.columns)
		return goja.Undefined()
	}))
	if _, err := call(goja.Undefined()); err != nil {
		return nil, fmt.Errorf("callback failed while reading the output row")
	}
	return values, schemaErr
}

func mappedValues(value goja.Value, columns []queryplan.Column) ([]any, error) {
	obj, ok := value.(*goja.Object)
	if !ok || obj.ClassName() != "Object" || obj.ExportType() == promiseType {
		return nil, fmt.Errorf("callback must return a synchronous object")
	}
	keys := obj.Keys()
	if len(keys) != len(columns) {
		return nil, fmt.Errorf("output fields do not match declared columns")
	}
	own := make(map[string]bool, len(keys))
	for _, name := range keys {
		own[name] = true
	}
	values := make([]any, len(columns))
	for i, column := range columns {
		if !own[column.Name] {
			return nil, fmt.Errorf("missing output column %q", column.Name)
		}
		value, err := scalarValue(obj.Get(column.Name))
		if err != nil || !matchesMappedType(value, column.Type) {
			return nil, fmt.Errorf("output column %q expects %s", column.Name, column.Type.Kind)
		}
		values[i] = value
	}
	return values, nil
}

func matchesMappedType(value any, typ queryplan.Type) bool {
	switch v := value.(type) {
	case nil:
		return typ.Nullable
	case string:
		return typ.Kind == queryplan.String
	case bool:
		return typ.Kind == queryplan.Boolean
	case int64:
		return typ.Kind == queryplan.Number || (typ.Kind == queryplan.Integer &&
			v >= -(1<<53-1) && v <= 1<<53-1)
	case float64:
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return false
		}
		return typ.Kind == queryplan.Number || (typ.Kind == queryplan.Integer &&
			math.Trunc(v) == v && v >= -(1<<53-1) && v <= 1<<53-1)
	default:
		return false
	}
}
