// Package queryplan describes immutable, local SQLite queries.
package queryplan

import (
	"fmt"
	"math"
	"strings"
)

type Kind string

const (
	String  Kind = "string"
	Boolean Kind = "boolean"
	Integer Kind = "integer"
	Number  Kind = "number"
)

type Type struct {
	Kind     Kind
	Nullable bool
}

type Column struct {
	Name string
	Type Type
}

// Owner identifies a local database handle independently of its schema.
type Owner struct{ marker byte }

func NewOwner() *Owner { return &Owner{} }

type Query struct {
	owner     *Owner
	table     string
	raw       *rawSource
	input     *Query
	columns   []Column
	predicate *Expr
	fields    []Field
	order     []Order
}

type Field struct {
	Name  string
	Value *Expr
}

type Order struct {
	Value      *Expr
	Descending bool
}

type Expr struct {
	typ       Type
	op        string
	row       *Query
	name      string
	value     any
	operands  []*Expr
	parameter Parameter
	count     *Query
}

func (e *Expr) Type() Type { return e.typ }

func Table(owner *Owner, name string, columns []Column) (*Query, error) {
	if owner == nil {
		return nil, fmt.Errorf("table requires a database owner")
	}
	return catalogQuery(owner, name, columns)
}

func catalogQuery(owner *Owner, name string, columns []Column) (*Query, error) {
	if !validName(name) || len(columns) == 0 || len(columns) > 128 {
		return nil, fmt.Errorf("invalid table description %q", name)
	}
	seen := make(map[string]bool, len(columns))
	for _, column := range columns {
		switch column.Type.Kind {
		case String, Boolean, Integer, Number:
		default:
			return nil, fmt.Errorf("unsupported type for %q", column.Name)
		}
		key := strings.ToLower(column.Name)
		if !validName(column.Name) || seen[key] {
			return nil, fmt.Errorf("invalid or duplicate column %q", column.Name)
		}
		seen[key] = true
	}
	return &Query{owner: owner, table: name, columns: append([]Column(nil), columns...)}, nil
}

func validName(name string) bool {
	return name != "" && !strings.ContainsRune(name, 0)
}

func (q *Query) Columns() []Column { return append([]Column(nil), q.columns...) }

func (q *Query) Column(name string) (*Expr, error) {
	for _, column := range q.columns {
		if column.Name == name {
			return &Expr{typ: column.Type, op: "column", row: q, name: name}, nil
		}
	}
	names := make([]string, len(q.columns))
	for i, c := range q.columns {
		names[i] = c.Name
	}
	return nil, fmt.Errorf("unknown field %q (available: %s)", name, strings.Join(names, ", "))
}

func (q *Query) Where(predicate *Expr) (*Query, error) {
	if predicate == nil || predicate.typ.Kind != Boolean {
		return nil, fmt.Errorf("where expects a boolean expression, use .eq() for a comparison")
	}
	return &Query{owner: q.owner, input: q, columns: q.columns, predicate: predicate, order: q.order}, nil
}

func (q *Query) Select(fields []Field) (*Query, error) {
	if len(fields) == 0 || len(fields) > 128 {
		return nil, fmt.Errorf("select expects between 1 and 128 fields")
	}
	result := &Query{owner: q.owner, input: q, fields: append([]Field(nil), fields...)}
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		key := strings.ToLower(field.Name)
		if !validName(field.Name) || seen[key] || field.Value == nil {
			return nil, fmt.Errorf("invalid or duplicate projection %q", field.Name)
		}
		seen[key] = true
		result.columns = append(result.columns, Column{field.Name, field.Value.typ})
	}
	for _, key := range q.order {
		index := -1
		for i, field := range fields {
			if sameOrderValue(field.Value, key.Value, q) {
				index = i
				break
			}
		}
		if index < 0 {
			return nil, fmt.Errorf("select must expose each ordering expression, or apply orderBy after select")
		}
		column := result.columns[index]
		result.order = append(result.order, Order{
			Value:      &Expr{typ: column.Type, op: "column", row: result, name: column.Name},
			Descending: key.Descending,
		})
	}
	return result, nil
}

func sameOrderValue(projected, key *Expr, input *Query) bool {
	// An inherited column sort is still visible under the input's current scope.
	return projected == key || (projected.op == "column" && key.op == "column" &&
		projected.row == input && projected.name == key.name && input.retains(key.row))
}

func (q *Query) retains(row *Query) bool {
	for current := q; current != nil; current = current.input {
		if current == row {
			return true
		}
		if current.fields != nil {
			break
		}
	}
	return false
}

func (q *Query) OrderBy(keys ...Order) (*Query, error) {
	if len(keys) == 0 || len(keys) > 128 {
		return nil, fmt.Errorf("orderBy expects between 1 and 128 keys")
	}
	c := compiler{owner: q.owner}
	for _, key := range keys {
		if key.Value == nil {
			return nil, fmt.Errorf("orderBy expects an expression")
		}
		if _, err := c.expr(key.Value, &scope{row: q, alias: "input"}, 0); err != nil {
			return nil, fmt.Errorf("orderBy: %w", err)
		}
	}
	return &Query{owner: q.owner, input: q, columns: q.columns, order: append([]Order(nil), keys...)}, nil
}

// DerivedFrom reports whether query composition retained the supplied input.
func (q *Query) DerivedFrom(input *Query) bool {
	for current := q; current != nil; current = current.input {
		if current == input {
			return true
		}
	}
	return false
}

func (q *Query) Count() *Expr {
	return &Expr{typ: Type{Kind: Integer}, op: "count", count: q}
}

func Literal(value any) (*Expr, error) {
	var typ Type
	switch v := value.(type) {
	case nil:
		typ.Nullable = true
	case string:
		if len(v) > 1<<20 {
			return nil, fmt.Errorf("string literal exceeds 1 MiB")
		}
		typ.Kind = String
	case bool:
		typ.Kind = Boolean
	case int:
		return Literal(int64(v))
	case int64:
		if v < -(1<<53-1) || v > 1<<53-1 {
			return nil, fmt.Errorf("integer literal must be a safe JavaScript integer")
		}
		typ.Kind = Integer
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("number literal must be finite")
		}
		typ.Kind = Number
	default:
		return nil, fmt.Errorf("unsupported literal type %T", v)
	}
	return &Expr{typ: typ, op: "literal", value: value}, nil
}

func Operation(op string, operands ...*Expr) (*Expr, error) {
	arity := 2
	switch op {
	case "not", "isNull", "isNotNull":
		arity = 1
	case "eq", "ne", "gt", "gte", "lt", "lte", "contains", "and", "or":
	default:
		return nil, fmt.Errorf("unknown expression operation %q", op)
	}
	if len(operands) != arity {
		return nil, fmt.Errorf("%s expects %d operands", op, arity)
	}
	var nullable bool
	for _, operand := range operands {
		if operand == nil {
			return nil, fmt.Errorf("%s expects an expression", op)
		}
		nullable = nullable || operand.typ.Nullable
	}
	left := operands[0].typ.Kind
	switch op {
	case "isNull", "isNotNull":
		nullable = false
	case "and", "or", "not":
		for _, operand := range operands {
			if operand.typ.Kind != Boolean {
				return nil, fmt.Errorf("%s expects boolean operands", op)
			}
		}
	case "contains":
		if left != String || operands[1].typ.Kind != String {
			return nil, fmt.Errorf("contains expects string operands")
		}
	default:
		right := operands[1].typ.Kind
		if left == "" || right == "" {
			return nil, fmt.Errorf("%s cannot compare with null, use isNull() or isNotNull()", op)
		}
		if left != right && !(numeric(left) && numeric(right)) {
			return nil, fmt.Errorf("%s cannot compare %s with %s", op, left, right)
		}
	}
	return &Expr{typ: Type{Boolean, nullable}, op: op, operands: append([]*Expr(nil), operands...)}, nil
}

func numeric(kind Kind) bool { return kind == Integer || kind == Number }
