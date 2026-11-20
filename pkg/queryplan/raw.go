package queryplan

import "fmt"

type rawSource struct {
	sql  string
	args []*Expr
}

// SQL builds an opaque source from normalized SQL and scalar arguments.
// Callers normalize and validate the statement using the SQLite query policy.
func SQL(sql string, args []*Expr, columns []Column) (*Query, error) {
	q := &Query{}
	if columns != nil {
		var err error
		q, err = catalogQuery(nil, "SQL", columns)
		if err != nil {
			return nil, err
		}
		q.table = ""
	}
	for _, arg := range args {
		if arg == nil || (arg.op != "literal" && arg.op != "parameter") {
			return nil, fmt.Errorf("SQL interpolation requires a scalar literal or parameter")
		}
	}
	q.raw = &rawSource{sql: sql, args: append([]*Expr(nil), args...)}
	return q, nil
}

type declaredSource struct {
	sql        string
	columns    []Column
	start, end int
}

// DeclaredSource describes an opaque input's contract and its local bindings.
type DeclaredSource struct {
	SQL     string
	Columns []Column
	Args    []any
}

// SQLDeclarations returns the direct output declaration or the declarations
// of composed inputs that must be checked before running the outer statement.
// Call with the positional values returned by Bind.
func (p *Plan) SQLDeclarations(args []any) ([]Column, []DeclaredSource) {
	if p.directSQL {
		return p.Columns(), nil
	}
	sources := make([]DeclaredSource, len(p.sources))
	for i, source := range p.sources {
		sources[i] = DeclaredSource{
			SQL:     source.sql,
			Columns: append([]Column(nil), source.columns...),
			Args:    append([]any(nil), args[source.start:source.end]...),
		}
	}
	return nil, sources
}
