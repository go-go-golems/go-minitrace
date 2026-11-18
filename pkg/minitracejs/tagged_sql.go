package minitracejs

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

func (b *relationalBinding) sql(call goja.FunctionCall) goja.Value {
	template, ok := call.Argument(0).(*goja.Object)
	if !ok || template.ClassName() != "Array" || template.Get("raw") == nil {
		panic(b.vm.NewTypeError("sql requires a tagged template"))
	}
	var parts []string
	b.check(b.vm.ExportTo(template, &parts))
	if len(parts) != len(call.Arguments) {
		panic(b.vm.NewTypeError("invalid SQL template"))
	}
	statement, err := minitracedb.TaggedSQL(parts)
	b.check(err)
	args := make([]*queryplan.Expr, len(parts)-1)
	for i, value := range call.Arguments[1:] {
		args[i] = b.expression(value)
	}
	q, err := queryplan.SQL(statement, args, nil)
	b.check(err)
	obj := b.vm.NewObject()
	_ = obj.Set("build", func() *goja.Object { return b.planObject(q) })
	_ = obj.Set("columns", func(shape *goja.Object) *goja.Object {
		q, err := queryplan.SQL(statement, args, b.columnTypes(shape))
		b.check(err)
		return b.queryObject(q)
	})
	return obj
}

func (b *relationalBinding) columnTypes(shape *goja.Object) []queryplan.Column {
	if shape == nil || shape.ClassName() != "Object" {
		panic(b.vm.NewTypeError("columns requires an object mapping column names to types"))
	}
	names := shape.Keys()
	columns := make([]queryplan.Column, 0, len(names))
	for _, name := range names {
		typ, ok := b.lookup(shape.Get(name)).(queryplan.Type)
		if !ok {
			panic(b.vm.NewTypeError("invalid type for column %q, use mt.type", name))
		}
		columns = append(columns, queryplan.Column{Name: name, Type: typ})
	}
	return columns
}

func (b *relationalBinding) typeObject(typ queryplan.Type) *goja.Object {
	obj := b.remember(typ)
	_ = obj.Set("nullable", func() *goja.Object {
		copy := typ
		copy.Nullable = true
		return b.typeObject(copy)
	})
	return obj
}

func (b *relationalBinding) typeBuilders() *goja.Object {
	obj := b.vm.NewObject()
	for _, kind := range []queryplan.Kind{queryplan.String, queryplan.Integer, queryplan.Number, queryplan.Boolean} {
		_ = obj.Set(string(kind), func() *goja.Object {
			return b.typeObject(queryplan.Type{Kind: kind})
		})
	}
	return obj
}
