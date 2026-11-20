package minitracejs

import (
	"fmt"
	"math"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

func scalarValue(value goja.Value) (any, error) {
	switch value.(type) {
	case *goja.Object, *goja.Symbol, nil:
		return nil, fmt.Errorf("expected a scalar value")
	}
	if goja.IsUndefined(value) {
		return nil, fmt.Errorf("expected a scalar value, received undefined")
	}
	return value.Export(), nil
}

func (b *relationalBinding) parameterBuilders() *goja.Object {
	obj := b.vm.NewObject()
	for _, kind := range []queryplan.Kind{queryplan.String, queryplan.Boolean, queryplan.Integer, queryplan.Number} {
		_ = obj.Set(string(kind), func(name goja.Value) *goja.Object {
			s, ok := name.(goja.String)
			if !ok {
				panic(b.vm.NewTypeError("parameter name must be a string"))
			}
			return b.parameterObject(queryplan.Parameter{Name: s.String(), Kind: kind})
		})
	}
	return obj
}

func (b *relationalBinding) parameterObject(descriptor queryplan.Parameter) *goja.Object {
	expr, err := queryplan.ParameterExpression(descriptor)
	b.checkBuild(err, "E_PARAMETER", descriptor.Name)
	obj := b.exprObject(expr)
	_ = obj.Set("default", func(value goja.Value) *goja.Object {
		next := descriptor
		scalar, err := scalarValue(value)
		b.check(err)
		next.Default = scalar
		next.HasDefault = true
		return b.parameterObject(next)
	})
	if descriptor.Kind == queryplan.Integer {
		_ = obj.Set("range", func(minValue, maxValue goja.Value) *goja.Object {
			next := descriptor
			next.Min = b.safeInteger(minValue)
			next.Max = b.safeInteger(maxValue)
			next.HasRange = true
			return b.parameterObject(next)
		})
	}
	_ = obj.Set("ui", func(options *goja.Object) *goja.Object {
		next := descriptor
		next.UI = queryplan.ParameterUI{
			Widget: stringOption(b.vm, options, "widget"),
			Label:  stringOption(b.vm, options, "label"),
			Help:   stringOption(b.vm, options, "help"),
		}
		return b.parameterObject(next)
	})
	return obj
}

func (b *relationalBinding) safeInteger(value goja.Value) int64 {
	raw, err := scalarValue(value)
	b.check(err)
	switch v := raw.(type) {
	case int64:
		if v >= -(1<<53-1) && v <= 1<<53-1 {
			return v
		}
	case float64:
		if math.Trunc(v) == v && v >= -(1<<53-1) && v <= 1<<53-1 {
			return int64(v)
		}
	}
	panic(b.vm.NewTypeError("range bounds must be safe integers"))
}

func (b *relationalBinding) boundParameters(options goja.Value) (map[string]any, error) {
	if goja.IsUndefined(options) || options == nil {
		return nil, nil
	}
	obj, ok := options.(*goja.Object)
	if !ok || obj.ClassName() != "Object" {
		return nil, fmt.Errorf("execute options must be an object")
	}
	value := obj.Get("params")
	if goja.IsUndefined(value) || value == nil {
		return nil, nil
	}
	params, ok := value.(*goja.Object)
	if !ok || params.ClassName() != "Object" {
		return nil, fmt.Errorf("execute params must be an object")
	}
	out := make(map[string]any)
	for _, name := range params.Keys() {
		value, err := scalarValue(params.Get(name))
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", name, err)
		}
		out[name] = value
	}
	return out, nil
}
