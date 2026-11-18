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
	vm       *goja.Runtime
	state    *goja.Object
	get, set goja.Callable
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
	if _, ok := value.(*goja.Object); ok || value == nil || goja.IsUndefined(value) {
		panic(b.vm.NewTypeError("expected a symbolic expression or scalar literal"))
	}
	expr, err := queryplan.Literal(value.Export())
	b.check(err)
	return expr
}

func (b *relationalBinding) exprObject(expr *queryplan.Expr) *goja.Object {
	obj := b.remember(expr)
	for _, op := range []string{"eq", "ne", "gt", "gte", "lt", "lte", "and", "or", "contains"} {
		_ = obj.Set(op, func(value goja.Value) *goja.Object {
			next, err := queryplan.Operation(op, expr, b.expression(value))
			b.check(err)
			return b.exprObject(next)
		})
	}
	for _, op := range []string{"not", "isNull", "isNotNull"} {
		_ = obj.Set(op, func() *goja.Object {
			next, err := queryplan.Operation(op, expr)
			b.check(err)
			return b.exprObject(next)
		})
	}
	_ = obj.Set("asc", func() *goja.Object { return b.remember(queryplan.Order{Value: expr}) })
	_ = obj.Set("desc", func() *goja.Object { return b.remember(queryplan.Order{Value: expr, Descending: true}) })
	return obj
}

func (b *relationalBinding) row(q *queryplan.Query) goja.Value {
	target := b.vm.NewObject()
	_ = target.SetPrototype(nil)
	proxy := b.vm.NewProxy(target, &goja.ProxyTrapConfig{
		Get: func(_ *goja.Object, name string, _ goja.Value) goja.Value {
			column, err := q.Column(name)
			if err != nil && name == "then" {
				return goja.Undefined()
			}
			b.check(err)
			return b.exprObject(column)
		},
	})
	return b.vm.ToValue(proxy)
}

func (b *relationalBinding) callback(method string, value, arg goja.Value) goja.Value {
	fn, ok := goja.AssertFunction(value)
	if !ok {
		panic(b.vm.NewTypeError("%s requires a callback", method))
	}
	result, err := fn(goja.Undefined(), arg)
	b.check(err)
	if obj, ok := result.(*goja.Object); ok {
		if obj.ExportType() == promiseType {
			panic(b.vm.NewTypeError("%s callback must return synchronously", method))
		}
	}
	return result
}

var promiseType = reflect.TypeFor[*goja.Promise]()

type tableValue struct{ query *queryplan.Query }

func (b *relationalBinding) from(value goja.Value) *goja.Object {
	table, ok := b.lookup(value).(tableValue)
	if !ok {
		panic(b.vm.NewTypeError("from requires a table from db.table()"))
	}
	return b.queryObject(table.query)
}

func (b *relationalBinding) queryObject(q *queryplan.Query) *goja.Object {
	obj := b.remember(q)
	_ = obj.Set("where", func(fn goja.Value) *goja.Object {
		value := b.callback("where", fn, b.row(q))
		predicate, ok := b.lookup(value).(*queryplan.Expr)
		if !ok {
			panic(b.vm.NewTypeError("where expects a symbolic predicate, use .eq() instead of ==="))
		}
		next, err := q.Where(predicate)
		b.check(err)
		return b.queryObject(next)
	})
	_ = obj.Set("select", func(fn goja.Value) *goja.Object {
		value, ok := b.callback("select", fn, b.row(q)).(*goja.Object)
		if !ok || value.ClassName() != "Object" || b.lookup(value) != nil {
			panic(b.vm.NewTypeError("select expects an object mapping output names to expressions"))
		}
		fields := make([]queryplan.Field, 0, len(value.Keys()))
		for _, name := range value.Keys() {
			fields = append(fields, queryplan.Field{Name: name, Value: b.expression(value.Get(name))})
		}
		next, err := q.Select(fields)
		b.check(err)
		return b.queryObject(next)
	})
	_ = obj.Set("orderBy", func(fn goja.Value) *goja.Object {
		value := b.callback("orderBy", fn, b.row(q))
		values := []goja.Value{value}
		if array, ok := value.(*goja.Object); ok && array.ClassName() == "Array" {
			b.check(b.vm.ExportTo(array, &values))
		}
		keys := make([]queryplan.Order, len(values))
		for i, item := range values {
			key, ok := b.lookup(item).(queryplan.Order)
			if !ok {
				panic(b.vm.NewTypeError("orderBy expects .asc() or .desc() keys"))
			}
			keys[i] = key
		}
		next, err := q.OrderBy(keys...)
		b.check(err)
		return b.queryObject(next)
	})
	_ = obj.Set("use", func(fn goja.Value) goja.Value {
		value := b.callback("use", fn, obj)
		if _, ok := b.lookup(value).(*queryplan.Query); !ok {
			panic(b.vm.NewTypeError("use expects a query builder result"))
		}
		return value
	})
	_ = obj.Set("count", func() *goja.Object { return b.exprObject(q.Count()) })
	_ = obj.Set("build", func() *goja.Object { return b.planObject(q) })
	return obj
}

func (b *relationalBinding) planObject(q *queryplan.Query) *goja.Object {
	plan, err := q.Build()
	b.check(err)
	obj := b.remember(plan)
	_ = obj.Set("explain", func() map[string]any { return map[string]any{"sql": plan.SQL()} })
	_ = obj.Set("parameters", func() []any { return []any{} })
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

func (b *relationalBinding) execute(ctx context.Context, h *DBHandle, plan *queryplan.Plan) goja.Value {
	if h.closed || (plan.Owner() != nil && plan.Owner() != h.owner) {
		panic(b.vm.NewTypeError("execute requires the open database that owns this plan"))
	}
	result, err := h.QueryResult(ctx, plan.SQL(), plan.Args()...)
	b.check(err)
	envelope := toPlainMap(result)
	// Preserve the SQL runner’s integer and real cell types for Go hosts.
	envelope["rows"] = result.Rows
	return b.vm.ToValue(envelope)
}
