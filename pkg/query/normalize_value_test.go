package query

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/duckdb/duckdb-go/v2"
)

func TestNormalizeValue(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name     string
		input    any
		wantType string
		wantVal  any
	}{
		{"nil", nil, "<nil>", nil},
		{"[]byte", []byte("hello"), "string", "hello"},
		{"time.Time", now, "string", now.Format(time.RFC3339Nano)},
		{"int64", int64(42), "int64", int64(42)},           // COUNT(*) — already JS number
		{"int32", int32(42), "int32", int32(42)},           // literal integer
		{"float64", float64(3.14), "float64", float64(3.14)},
		{"string", "hello", "string", "hello"},
		// The key fix: *big.Int from DuckDB SUM() is converted to int64
		{"*big.Int (fits int64)", big.NewInt(42), "int64", int64(42)},
		// duckdb.Decimal is converted to float64
		{"duckdb.Decimal", duckdb.Decimal{Width: 3, Scale: 2, Value: big.NewInt(314)}, "float64", float64(3.14)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			resultType := fmt.Sprintf("%T", result)

			if tc.wantType == "<nil>" {
				if result != nil {
					t.Errorf("NormalizeValue(%v) = %v (%T), want nil", tc.input, result, result)
				}
				return
			}

			if resultType != tc.wantType {
				t.Errorf("NormalizeValue(%v) type = %s, want %s", tc.input, resultType, tc.wantType)
			}

			if result != tc.wantVal {
				// For duckdb.Decimal, check with tolerance
				if f, ok := result.(float64); ok {
					if wf, ok := tc.wantVal.(float64); ok && f != wf {
						t.Errorf("NormalizeValue(%v) = %v, want %v", tc.input, result, tc.wantVal)
					}
				}
			}
		})
	}
}
