package queryplan

type rawSource struct {
	sql  string
	args []any
}

// SQL builds an opaque source from normalized SQL and scalar arguments.
// Callers normalize and validate the statement using the SQLite query policy.
func SQL(sql string, args []any, columns []Column) (*Query, error) {
	q := &Query{}
	if columns != nil {
		var err error
		q, err = catalogQuery(nil, "SQL", columns)
		if err != nil {
			return nil, err
		}
		q.table = ""
	}
	q.raw = &rawSource{sql: sql, args: make([]any, len(args))}
	for i, arg := range args {
		literal, err := Literal(arg)
		if err != nil {
			return nil, err
		}
		q.raw.args[i] = literal.value
	}
	return q, nil
}
