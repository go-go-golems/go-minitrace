package minitracedb

import (
	"context"
	"fmt"
)

// MapResult applies a cardinality-preserving runtime stage, sharing the SQL
// runner's cell normalization and output limit policy.
func (r *QueryRunner) MapResult(ctx context.Context, input QueryResult, columns []string,
	mapper func(context.Context, map[string]any) ([]any, error),
) QueryResult {
	if input.Error != "" {
		return input
	}
	if !input.Complete {
		out := queryErrorResult("mapRows requires complete SQL input")
		out.Truncated, out.RowsTruncated = input.Truncated, input.RowsTruncated
		out.CellDiagnostics = input.CellDiagnostics
		return out
	}
	if len(columns) > r.opts.MaxColumns {
		return queryErrorResult(fmt.Sprintf("mapRows returns %d columns; max is %d", len(columns), r.opts.MaxColumns))
	}
	if len(input.Rows) > r.opts.MaxRows {
		return queryErrorResult("mapRows input exceeds the row limit")
	}
	ctx, cancel := context.WithTimeout(ctx, r.opts.Timeout)
	defer cancel()
	out := QueryResult{Columns: columns, Rows: make([]map[string]any, 0, len(input.Rows)), Complete: true}
	for i, row := range input.Rows {
		if err := ctx.Err(); err != nil {
			return queryErrorResult(fmt.Sprintf("mapRows row %d: %s", i, err))
		}
		values, err := mapper(ctx, row)
		if cancelErr := ctx.Err(); cancelErr != nil {
			err = cancelErr
		}
		if err != nil {
			return queryErrorResult(fmt.Sprintf("mapRows row %d: %s", i, err))
		}
		if len(values) != len(columns) {
			return queryErrorResult(fmt.Sprintf("mapRows row %d: output does not match declared columns", i))
		}
		r.appendResultRow(&out, values)
	}
	r.finishResult(&out, "mapRows")
	return out
}
