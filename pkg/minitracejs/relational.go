package minitracejs

import (
	"context"
	"reflect"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

// Relational values are ordinary JS objects. The private WeakMap gives them
// identity without exposing Go pointers or retaining discarded query branches.
type relationalBinding struct {
	vm          *goja.Runtime
	state       *goja.Object
	get, set    goja.Callable
	mappingRows bool
}

type relationalValue struct{ value any }

func newRelationalBinding(vm *goja.Runtime) *relationalBinding {
	state, err := vm.New(vm.Get("WeakMap"))
	if err != nil {
		panic(vm.NewGoError(err))
	}
	get, _ := goja.AssertFunction(state.Get("get"))
	set, _ := goja.AssertFunction(state.Get("set"))
	return &relationalBinding{vm: vm, state: state, get: get, set: set}
}

func (b *relationalBinding) remember(value any) *goja.Object {
	obj := b.vm.NewObject()
	_, err := b.set(b.state, obj, b.vm.ToValue(&relationalValue{value}))
	b.check(err)
	return obj
}

func (b *relationalBinding) lookup(value goja.Value) any {
	if _, ok := value.(*goja.Object); !ok {
		return nil
	}
	entry, err := b.get(b.state, value)
	b.check(err)
	if entry != nil && !goja.IsUndefined(entry) {
		if wrapped, ok := entry.Export().(*relationalValue); ok {
			return wrapped.value
		}
	}
	return nil
}

func (b *relationalBinding) check(err error) {
	if err != nil {
		panic(b.vm.NewGoError(err))
	}
}

func (b *relationalBinding) expression(value goja.Value) *queryplan.Expr {
	if expr, ok := b.lookup(value).(*queryplan.Expr); ok {
		return expr
	}
	raw, err := scalarValue(value)
	b.checkBuild(err, "E_EXPRESSION", "literal")
	expr, err := queryplan.Literal(raw)
	b.checkBuild(err, "E_EXPRESSION", "literal")
	return expr
}

func (b *relationalBinding) exprObject(expr *queryplan.Expr) *goja.Object {
	obj := b.remember(expr)
	for _, op := range []string{"eq", "ne", "gt", "gte", "lt", "lte", "and", "or", "contains"} {
		_ = obj.Set(op, func(value goja.Value) *goja.Object {
			next, err := queryplan.Operation(op, expr, b.expression(value))
			b.checkBuild(err, "E_EXPRESSION", op)
			return b.exprObject(next)
		})
	}
	for _, op := range []string{"not", "isNull", "isNotNull"} {
		_ = obj.Set(op, func() *goja.Object {
			next, err := queryplan.Operation(op, expr)
			b.checkBuild(err, "E_EXPRESSION", op)
			return b.exprObject(next)
		})
	}
	_ = obj.Set("asc", func() *goja.Object { return b.remember(queryplan.Order{Value: expr}) })
	_ = obj.Set("desc", func() *goja.Object { return b.remember(queryplan.Order{Value: expr, Descending: true}) })
	return obj
}

func (b *relationalBinding) row(q *queryplan.Query, method string) goja.Value {
	target := b.vm.NewObject()
	_ = target.SetPrototype(nil)
	proxy := b.vm.NewProxy(target, &goja.ProxyTrapConfig{
		Get: func(_ *goja.Object, name string, _ goja.Value) goja.Value {
			column, err := q.Column(name)
			if err != nil && name == "then" {
				return goja.Undefined()
			}
			b.checkBuild(err, "E_UNKNOWN_COLUMN", method+"."+name)
			return b.exprObject(column)
		},
	})
	return b.vm.ToValue(proxy)
}

func (b *relationalBinding) callback(method string, value, arg goja.Value) goja.Value {
	fn, ok := goja.AssertFunction(value)
	if !ok {
		b.buildError("E_CALLBACK", method, method+" requires a callback")
	}
	result, err := fn(goja.Undefined(), arg)
	if exception, ok := err.(*goja.Exception); ok {
		panic(exception)
	}
	b.check(err)
	if obj, ok := result.(*goja.Object); ok {
		if obj.ExportType() == promiseType {
			b.buildError("E_CALLBACK", method, method+" callback must return synchronously")
		}
	}
	return result
}

var promiseType = reflect.TypeFor[*goja.Promise]()

type tableValue struct{ query *queryplan.Query }

func (b *relationalBinding) from(value goja.Value) *goja.Object {
	switch source := b.lookup(value).(type) {
	case tableValue:
		return b.queryObject(source.query)
	case executablePlan:
		b.buildError("E_QUERY_SOURCE", "from", "from expects db.table(), and build() ends query construction. For SQL composition, call .columns(...) on the tagged SQL source before build()")
	case queryValue:
		b.buildError("E_QUERY_SOURCE", "from", "this value is already a query builder. Continue composition directly before calling build()")
	default:
		b.buildError("E_QUERY_SOURCE", "from", "from expects a table from db.table()")
	}
	return nil
}

type queryValue struct {
	query     *queryplan.Query
	fragments []*fragment
}

func (b *relationalBinding) queryObject(q *queryplan.Query, fragments ...*fragment) *goja.Object {
	obj := b.remember(queryValue{q, fragments})
	_ = obj.Set("where", func(fn goja.Value) *goja.Object {
		input := q.RowScope()
		value := b.callback("where", fn, b.row(input, "where"))
		predicate, ok := b.lookup(value).(*queryplan.Expr)
		if !ok {
			b.buildError("E_EXPECTED_PREDICATE", "where", "where expects a symbolic predicate, use .eq() instead of ===")
		}
		next, err := input.Where(predicate)
		b.checkBuild(err, "E_EXPECTED_PREDICATE", "where")
		return b.queryObject(next, fragments...)
	})
	_ = obj.Set("select", func(fn goja.Value) *goja.Object {
		input := q.RowScope()
		value, ok := b.callback("select", fn, b.row(input, "select")).(*goja.Object)
		if !ok || value.ClassName() != "Object" || b.lookup(value) != nil {
			b.buildError("E_PROJECTION", "select", "select expects an object mapping output names to expressions")
		}
		fields := make([]queryplan.Field, 0, len(value.Keys()))
		for _, name := range value.Keys() {
			fields = append(fields, queryplan.Field{Name: name, Value: b.expression(value.Get(name))})
		}
		next, err := input.Select(fields)
		b.checkBuild(err, "E_PROJECTION", "select")
		return b.queryObject(next, fragments...)
	})
	_ = obj.Set("orderBy", func(fn goja.Value) *goja.Object {
		input := q.RowScope()
		value := b.callback("orderBy", fn, b.row(input, "orderBy"))
		values := []goja.Value{value}
		if array, ok := value.(*goja.Object); ok && array.ClassName() == "Array" {
			b.check(b.vm.ExportTo(array, &values))
		}
		keys := make([]queryplan.Order, len(values))
		for i, item := range values {
			key, ok := b.lookup(item).(queryplan.Order)
			if !ok {
				b.buildError("E_ORDER", "orderBy", "orderBy expects .asc() or .desc() keys")
			}
			keys[i] = key
		}
		next, err := input.OrderBy(keys...)
		b.checkBuild(err, "E_ORDER", "orderBy")
		return b.queryObject(next, fragments...)
	})
	_ = obj.Set("use", func(fn goja.Value) goja.Value {
		value := b.callback("use", fn, obj)
		if _, ok := b.lookup(value).(queryValue); !ok {
			panic(b.vm.NewTypeError("use expects a query builder result"))
		}
		return value
	})
	_ = obj.Set("count", func() *goja.Object { return b.exprObject(q.Count()) })
	_ = obj.Set("build", func() *goja.Object { return b.planObject(q, nil) })
	_ = obj.Set("mapRows", func(fn goja.Value, options *goja.Object) *goja.Object {
		return b.mappedQuery(q, fn, options)
	})
	b.attachFragments(obj, q, fragments)
	return obj
}

func (b *relationalBinding) planObject(q *queryplan.Query, mapper *runtimeMapper) *goja.Object {
	plan, err := q.Build()
	b.checkBuild(err, "E_QUERY_BUILD", "build")
	obj := b.remember(executablePlan{plan, mapper})
	_ = obj.Set("explain", func() map[string]any { return map[string]any{"sql": plan.SQL(), "mapRows": mapper != nil} })
	_ = obj.Set("parameters", func() []map[string]any { return toPlainSlice(plan.Parameters()) })
	return obj
}

func (b *relationalBinding) table(h *DBHandle, name string) *goja.Object {
	if h.closed {
		panic(b.vm.NewTypeError("table requires an open database"))
	}
	for _, table := range h.schema.Tables {
		if table.Name != name {
			continue
		}
		columns := make([]queryplan.Column, len(table.Columns))
		for i, column := range table.Columns {
			var kind queryplan.Kind
			switch strings.ToUpper(column.Type) {
			case "INTEGER":
				kind = queryplan.Integer
			case "REAL":
				kind = queryplan.Number
			case "TEXT":
				kind = queryplan.String
			default:
				panic(b.vm.NewTypeError("unsupported catalog type %q for %s.%s", column.Type, name, column.Name))
			}
			columns[i] = queryplan.Column{Name: column.Name, Type: queryplan.Type{Kind: kind, Nullable: column.Nullable}}
		}
		q, err := queryplan.Table(h.owner, name, columns)
		b.check(err)
		return b.remember(tableValue{q})
	}
	panic(b.vm.NewTypeError("unknown table %q, see db.tables()", name))
}

func (b *relationalBinding) execute(ctx context.Context, h *DBHandle, executable executablePlan, options goja.Value) goja.Value {
	plan := executable.query
	if executable.mapper != nil && b.mappingRows {
		return b.executionError("mapRows cannot execute another mapping stage from a callback")
	}
	if h.closed || (plan.Owner() != nil && plan.Owner() != h.owner) {
		return b.executionError("execute requires the open database that owns this plan")
	}
	params, err := b.boundParameters(options)
	if err != nil {
		return b.executionError(err.Error())
	}
	result, err := h.runner.QueryPlanResult(ctx, plan, params)
	if err != nil {
		return b.executionError(err.Error())
	}
	if executable.mapper != nil {
		result = b.mapResult(ctx, h, result, executable.mapper)
	}
	envelope := toPlainMap(result)
	// Preserve the SQL runner’s integer and real cell types for Go hosts.
	envelope["rows"] = result.Rows
	return b.vm.ToValue(envelope)
}

func (b *relationalBinding) executionError(message string) goja.Value {
	return b.vm.ToValue(map[string]any{
		"error": message, "complete": false, "count": int64(0),
		"rows": []map[string]any{}, "columns": []string{},
	})
}
