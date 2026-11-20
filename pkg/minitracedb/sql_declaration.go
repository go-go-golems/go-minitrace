package minitracedb

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

func checkSQLDeclaration(ctx context.Context, conn *sql.Conn, source queryplan.DeclaredSource, maxColumns int) error {
	if len(source.Columns) > maxColumns {
		return fmt.Errorf("declared SQL source returns %d columns; max is %d", len(source.Columns), maxColumns)
	}
	statement, err := validateReadOnlySQLiteQuery(source.SQL, QueryOptions{})
	if err != nil {
		return err
	}
	if err := ensureReadonlyPreparedQuery(conn, statement); err != nil {
		return err
	}
	rows, err := conn.QueryContext(ctx, statement, source.Args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	if err := checkDeclaredNames(names, source.Columns); err != nil {
		return err
	}
	values := make([]any, len(names))
	ptrs := make([]any, len(names))
	for i := range values {
		ptrs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		if err := checkDeclaredValues(values, source.Columns); err != nil {
			return err
		}
	}
	return rows.Err()
}

func checkDeclaredNames(names []string, columns []queryplan.Column) error {
	if len(names) != len(columns) {
		return fmt.Errorf("declared SQL source expects %d columns, SQLite returned %d", len(columns), len(names))
	}
	for i, name := range names {
		if name != columns[i].Name {
			return fmt.Errorf("declared SQL column %d expects %q, SQLite returned %q", i, columns[i].Name, name)
		}
	}
	return nil
}

func checkDeclaredValues(values []any, columns []queryplan.Column) error {
	for i, value := range values {
		column := columns[i]
		if !declaredValueMatches(value, column.Type) {
			return fmt.Errorf("declared SQL column %q expects %s (nullable: %t), SQLite returned %T",
				column.Name, column.Type.Kind, column.Type.Nullable, value)
		}
	}
	return nil
}

func declaredValueMatches(value any, typ queryplan.Type) bool {
	switch v := value.(type) {
	case nil:
		return typ.Nullable
	case string:
		return typ.Kind == queryplan.String
	case int64:
		return typ.Kind == queryplan.Integer || typ.Kind == queryplan.Number ||
			(typ.Kind == queryplan.Boolean && (v == 0 || v == 1))
	case bool:
		return typ.Kind == queryplan.Boolean
	case float64:
		return typ.Kind == queryplan.Number && !math.IsNaN(v) && !math.IsInf(v, 0)
	default:
		return false
	}
}
