package minitracejs

import (
	"fmt"
	"slices"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

type fragmentMethod struct {
	name string
	call goja.Callable
}

type fragment struct {
	name     string
	requires []string
	methods  []fragmentMethod
}

func (b *relationalBinding) newFragment(config *goja.Object) *goja.Object {
	name := stringOption(b.vm, config, "name")
	if name == "" || stringOption(b.vm, config, "target") != "query" {
		panic(b.vm.NewTypeError("fragment requires a name and target: query"))
	}
	f := &fragment{name: name}
	if requires := config.Get("requires"); requires != nil && !goja.IsUndefined(requires) {
		array, ok := requires.(*goja.Object)
		if !ok || array.ClassName() != "Array" {
			panic(b.vm.NewTypeError("fragment %q requires must be an array of field names", name))
		}
		var required []goja.Value
		b.check(b.vm.ExportTo(array, &required))
		for _, value := range required {
			s, ok := value.(goja.String)
			if !ok || s.String() == "" {
				panic(b.vm.NewTypeError("fragment %q requires nonempty field names", name))
			}
			f.requires = append(f.requires, s.String())
		}
	}
	methods, ok := config.Get("methods").(*goja.Object)
	if !ok || methods.ClassName() != "Object" || len(methods.Keys()) == 0 {
		panic(b.vm.NewTypeError("fragment %q requires a methods object", name))
	}
	for _, method := range methods.Keys() {
		fn, ok := goja.AssertFunction(methods.Get(method))
		if !ok || method == "" || method == "then" || method == "mapRows" {
			panic(b.vm.NewTypeError("invalid fragment method %q in %q", method, name))
		}
		f.methods = append(f.methods, fragmentMethod{name: method, call: fn})
	}
	return b.remember(f)
}

func (b *relationalBinding) attachFragments(obj *goja.Object, q *queryplan.Query, fragments []*fragment) {
	_ = obj.Set("extend", func(value goja.Value) *goja.Object {
		f, ok := b.lookup(value).(*fragment)
		if !ok {
			panic(b.vm.NewTypeError("extend requires a value from mt.fragment()"))
		}
		for _, method := range f.methods {
			if value := obj.Get(method.name); value != nil {
				panic(b.vm.NewTypeError("fragment %q method %q collides with a query method", f.name, method.name))
			}
		}
		next := append(append([]*fragment(nil), fragments...), f)
		return b.queryObject(q, next...)
	})
	for _, f := range fragments {
		for _, method := range f.methods {
			_ = obj.Set(method.name, func(call goja.FunctionCall) goja.Value {
				for _, column := range f.requires {
					if _, err := q.Column(column); err != nil {
						b.check(fmt.Errorf("fragment %q method %q: %w", f.name, method.name, err))
					}
				}
				args := make([]goja.Value, 1, len(call.Arguments)+1)
				args[0] = obj
				result, err := method.call(goja.Undefined(), append(args, call.Arguments...)...)
				b.check(err)
				next, ok := b.lookup(result).(queryValue)
				if !ok || !next.query.DerivedFrom(q) ||
					len(next.fragments) < len(fragments) || !slices.Equal(next.fragments[:len(fragments)], fragments) {
					panic(b.vm.NewTypeError("fragment %q method %q must return a query derived from its input", f.name, method.name))
				}
				return result
			})
		}
	}
}
