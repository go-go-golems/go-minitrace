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
