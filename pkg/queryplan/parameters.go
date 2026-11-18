package queryplan

import (
	"fmt"
	"math"
	"sort"
)

type ParameterUI struct {
	Widget string `json:"widget,omitempty"`
	Label  string `json:"label,omitempty"`
	Help   string `json:"help,omitempty"`
}

// Parameter describes a value binding. UI metadata does not affect validation.
type Parameter struct {
	Name       string      `json:"name"`
	Kind       Kind        `json:"type"`
	HasDefault bool        `json:"hasDefault"`
	Default    any         `json:"default,omitempty"`
	HasRange   bool        `json:"hasRange"`
	Min        int64       `json:"min"`
	Max        int64       `json:"max"`
	UI         ParameterUI `json:"ui"`
}

func ParameterExpression(descriptor Parameter) (*Expr, error) {
	if !validName(descriptor.Name) {
		return nil, fmt.Errorf("parameter name must be nonempty")
	}
	switch descriptor.Kind {
	case Integer, String, Boolean, Number:
	default:
		return nil, fmt.Errorf("parameter %q has unsupported type %q", descriptor.Name, descriptor.Kind)
	}
	if descriptor.HasRange && (descriptor.Kind != Integer || descriptor.Min > descriptor.Max ||
		descriptor.Min < -(1<<53-1) || descriptor.Max > 1<<53-1) {
		return nil, fmt.Errorf("parameter %q needs an ordered range of safe integers", descriptor.Name)
	}
	if descriptor.HasDefault {
		value, err := descriptor.validate(descriptor.Default)
		if err != nil {
			return nil, err
		}
		descriptor.Default = value
	} else {
		descriptor.Default = nil
	}
	return &Expr{op: "parameter", typ: Type{Kind: descriptor.Kind}, parameter: descriptor}, nil
}

func (p Parameter) validate(value any) (any, error) {
	if v, ok := value.(float64); ok && p.Kind == Integer && math.Trunc(v) == v && v >= -(1<<53-1) && v <= 1<<53-1 {
		value = int64(v)
	}
	lit, err := Literal(value)
	if err != nil {
		return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
	}
	if lit.typ.Kind != p.Kind && !(p.Kind == Number && lit.typ.Kind == Integer) {
		return nil, fmt.Errorf("parameter %q expects %s", p.Name, p.Kind)
	}
	if p.HasRange {
		n := lit.value.(int64)
		if n < p.Min || n > p.Max {
			return nil, fmt.Errorf("parameter %q expects integer in [%d, %d]", p.Name, p.Min, p.Max)
		}
	}
	if p.Kind == Number && lit.typ.Kind == Integer {
		return float64(lit.value.(int64)), nil
	}
	return lit.value, nil
}

type binding struct {
	name  string
	value any
}

func (c *compiler) bind(expr *Expr) error {
	if expr.op == "literal" {
		c.bindings = append(c.bindings, binding{value: expr.value})
		return nil
	}
	param := expr.parameter
	if c.parameters == nil {
		c.parameters = make(map[string]Parameter)
	}
	if first, exists := c.parameters[param.Name]; exists {
		current := param
		current.UI = first.UI
		if first != current {
			return fmt.Errorf("conflicting declarations for parameter %q", param.Name)
		}
	} else {
		c.parameters[param.Name] = param
		c.paramOrder = append(c.paramOrder, param.Name)
	}
	c.bindings = append(c.bindings, binding{name: param.Name})
	return nil
}

func (p *Plan) Parameters() []Parameter {
	out := make([]Parameter, 0, len(p.paramOrder))
	for _, name := range p.paramOrder {
		out = append(out, p.parameters[name])
	}
	return out
}

// Bind resolves named values into the positional arguments for the compiled SQL.
func (p *Plan) Bind(supplied map[string]any) ([]any, error) {
	for _, name := range sortedKeys(supplied) {
		if _, exists := p.parameters[name]; !exists {
			return nil, fmt.Errorf("unknown parameter %q", name)
		}
	}
	resolved := make(map[string]any, len(p.parameters))
	for _, name := range p.paramOrder {
		param := p.parameters[name]
		value, ok := supplied[name]
		if !ok {
			if !param.HasDefault {
				return nil, fmt.Errorf("missing parameter %q", name)
			}
			value = param.Default
		}
		value, err := param.validate(value)
		if err != nil {
			return nil, err
		}
		resolved[name] = value
	}
	args := make([]any, len(p.bindings))
	for i, slot := range p.bindings {
		if slot.name != "" {
			args[i] = resolved[slot.name]
		} else {
			args[i] = slot.value
		}
	}
	return args, nil
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}
