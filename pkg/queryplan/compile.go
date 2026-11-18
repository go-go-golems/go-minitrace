package queryplan

import (
	"fmt"
	"strings"
)

type Plan struct {
	owner   *Owner
	sql     string
	args    []any
	columns []Column
}

func (p *Plan) Owner() *Owner     { return p.owner }
func (p *Plan) SQL() string       { return p.sql }
func (p *Plan) Args() []any       { return append([]any(nil), p.args...) }
func (p *Plan) Columns() []Column { return append([]Column(nil), p.columns...) }

func (q *Query) Build() (*Plan, error) {
	c := compiler{owner: q.owner}
	sql, err := c.query(q, nil, 0)
	if err != nil {
		return nil, err
	}
	if len(q.order) != 0 {
		alias := c.alias()
		scope := &scope{row: q, alias: alias, inherited: true}
		keys := make([]string, len(q.order))
		for i, order := range q.order {
			key, err := c.expr(order.Value, scope, 0)
			if err != nil {
				return nil, fmt.Errorf("orderBy: %w", err)
			}
			if order.Descending {
				key += " DESC"
			} else {
				key += " ASC"
			}
			keys[i] = key
		}
		sql = "SELECT * FROM (" + sql + ") AS " + alias + " ORDER BY " + strings.Join(keys, ", ")
	}
	if len(sql) > 1<<20 {
		return nil, fmt.Errorf("compiled SQL exceeds 1 MiB")
	}
	return &Plan{owner: q.owner, sql: sql, args: c.args, columns: q.Columns()}, nil
}

type compiler struct {
	owner   *Owner
	aliases int
	nodes   int
	args    []any
}

type scope struct {
	row       *Query
	alias     string
	parent    *scope
	inherited bool
}

func (c *compiler) alias() string {
	c.aliases++
	return fmt.Sprintf("q%d", c.aliases)
}

func (c *compiler) visit(depth int) error {
	c.nodes++
	if depth > 32 || c.nodes > 4096 {
		return fmt.Errorf("query exceeds the nesting or expression limit")
	}
	return nil
}

func quote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func (c *compiler) query(q *Query, outer *scope, depth int) (string, error) {
	if err := c.visit(depth); err != nil {
		return "", err
	}
	if q.owner != c.owner {
		return "", fmt.Errorf("query references a different database")
	}
	if q.input == nil {
		names := make([]string, len(q.columns))
		for i, column := range q.columns {
			names[i] = quote(column.Name)
		}
		return "SELECT " + strings.Join(names, ", ") + " FROM " + quote(q.table), nil
	}
	// With no row-selection limit, sort nodes can be omitted from subqueries.
	// Build emits the effective order once, at the outermost level.
	if q.fields == nil && q.predicate == nil {
		return c.query(q.input, outer, depth+1)
	}
	alias := c.alias()
	visible := &scope{row: q.input, alias: alias, parent: outer}
	projection := "*"
	if q.fields != nil {
		fields := make([]string, len(q.fields))
		for i, field := range q.fields {
			value, err := c.expr(field.Value, visible, depth+1)
			if err != nil {
				return "", fmt.Errorf("select %q: %w", field.Name, err)
			}
			fields[i] = value + " AS " + quote(field.Name)
		}
		projection = strings.Join(fields, ", ")
	}
	input, err := c.query(q.input, outer, depth+1)
	if err != nil {
		return "", err
	}
	sql := "SELECT " + projection + " FROM (" + input + ") AS " + alias
	if q.predicate != nil {
		predicate, err := c.expr(q.predicate, visible, depth+1)
		if err != nil {
			return "", fmt.Errorf("where: %w", err)
		}
		sql += " WHERE " + predicate
	}
	return sql, nil
}

func (c *compiler) expr(e *Expr, visible *scope, depth int) (string, error) {
	if err := c.visit(depth); err != nil {
		return "", err
	}
	switch e.op {
	case "column":
		for s := visible; s != nil; s = s.parent {
			if s.row == e.row || (s.inherited && s.row.retains(e.row)) {
				return s.alias + "." + quote(e.name), nil
			}
		}
		return "", fmt.Errorf("column %q is outside its row scope", e.name)
	case "literal":
		c.args = append(c.args, e.value)
		return "?", nil
	case "count":
		sql, err := c.query(e.count, visible, depth+1)
		if err != nil {
			return "", err
		}
		return "(SELECT COUNT(*) FROM (" + sql + ") AS " + c.alias() + ")", nil
	}
	lhs, err := c.expr(e.operands[0], visible, depth+1)
	if err != nil {
		return "", err
	}
	switch e.op {
	case "not":
		return "(NOT " + lhs + ")", nil
	case "isNull":
		return "(" + lhs + " IS NULL)", nil
	case "isNotNull":
		return "(" + lhs + " IS NOT NULL)", nil
	}
	rhs, err := c.expr(e.operands[1], visible, depth+1)
	if err != nil {
		return "", err
	}
	if e.op == "contains" {
		return "(instr(" + lhs + ", " + rhs + ") > 0)", nil
	}
	operator := map[string]string{
		"eq": "=", "ne": "<>", "gt": ">", "gte": ">=", "lt": "<", "lte": "<=",
		"and": "AND", "or": "OR",
	}[e.op]
	return "(" + lhs + " " + operator + " " + rhs + ")", nil
}
