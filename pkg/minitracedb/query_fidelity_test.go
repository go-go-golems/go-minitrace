package minitracedb

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestQueryCellByteFidelity(t *testing.T) {
	tests := []struct {
		name, text       string
		limit            int
		want             string
		clipped, invalid bool
	}{
		{"ascii boundary", "abc", 3, "abc", false, false},
		{"ascii overflow", "abcd", 3, "abc", true, false},
		{"unicode exact", "abé", 4, "abé", false, false},
		{"unicode split", "abé", 3, "ab", true, false},
		{"supplementary split", "a😀z", 4, "a", true, false},
		{"supplementary exact", "a😀", 5, "a😀", false, false},
		{"no rune fits", "😀", 1, "", true, false},
		{"invalid utf8", "a\xffz", 10, "a�z", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := setupQueryRunner(t)
			runner.opts = WithDefaultQueryOptions(QueryOptions{MaxCellBytes: tt.limit})
			result, err := runner.QueryResult(context.Background(), "SELECT ? AS evidence", tt.text)
			if err != nil || result.Error != "" {
				t.Fatalf("query: %+v %v", result, err)
			}
			got, ok := result.Rows[0]["evidence"].(string)
			if !ok || got != tt.want || !utf8.ValidString(got) {
				t.Fatalf("got %q want %q", got, tt.want)
			}
			lossy := tt.clipped || tt.invalid
			if result.Complete == lossy || result.Truncated != lossy || result.RowsTruncated {
				t.Fatalf("bad completeness: %+v", result)
			}
			if lossy {
				if len(result.CellDiagnostics) != 1 {
					t.Fatalf("diagnostics: %+v", result)
				}
				d := result.CellDiagnostics[0]
				if d.RowIndex != 0 || d.ColumnIndex != 0 || d.Column != "evidence" || d.OriginalBytes != len(tt.text) || d.EmittedBytes != len(tt.want) || d.Truncated != tt.clipped || d.InvalidUTF8 != tt.invalid {
					t.Fatalf("diagnostic: %+v", d)
				}
			} else if len(result.CellDiagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", result)
			}
			raw, err := json.Marshal(result)
			if err != nil || !utf8.Valid(raw) {
				t.Fatalf("JSON serialization: %q %v", raw, err)
			}
			var decoded QueryResult
			if err := json.Unmarshal(raw, &decoded); err != nil || decoded.Complete != result.Complete {
				t.Fatalf("JSON completeness lost: %s %v", raw, err)
			}
		})
	}
}

func TestQueryStrictEvidenceIncludesRowsAndCells(t *testing.T) {
	for _, strict := range []bool{false, true} {
		runner := setupQueryRunner(t)
		runner.opts = WithDefaultQueryOptions(QueryOptions{MaxRows: 1, MaxCellBytes: 3, StrictEvidence: strict})
		result, err := runner.QueryResult(context.Background(), "SELECT 'abé' AS text UNION ALL SELECT 'second' ORDER BY text")
		if err != nil || result.Complete || !result.Truncated || !result.RowsTruncated || len(result.CellDiagnostics) != 1 {
			t.Fatalf("bad result %+v %v", result, err)
		}
		if strict {
			if result.Error == "" || result.Count != 0 || len(result.Rows) != 0 {
				t.Fatalf("strict leaked partial evidence: %+v", result)
			}
			if _, err := runner.Query(context.Background(), "SELECT 'abcd'"); err == nil {
				t.Fatal("Query should reject incomplete evidence")
			}
			if _, err := runner.QueryOne(context.Background(), "SELECT 'abcd'"); err == nil {
				t.Fatal("QueryOne should reject incomplete evidence")
			}
		} else if result.Error != "" || result.Count != 1 || result.Rows[0]["text"] != "ab" {
			t.Fatalf("permissive result: %+v", result)
		}
	}
	runner := setupQueryRunner(t)
	runner.opts = WithDefaultQueryOptions(QueryOptions{MaxRows: 1, MaxCellBytes: 3, StrictEvidence: true})
	exact, _ := runner.QueryResult(context.Background(), "SELECT 'abc' AS text")
	if !exact.Complete || exact.Error != "" || exact.Count != 1 {
		t.Fatalf("exact limit rejected: %+v", exact)
	}
	rowsOnly, _ := runner.QueryResult(context.Background(), "SELECT 1 UNION ALL SELECT 2")
	if rowsOnly.Complete || !rowsOnly.RowsTruncated || len(rowsOnly.CellDiagnostics) != 0 || rowsOnly.Error == "" {
		t.Fatalf("row overflow: %+v", rowsOnly)
	}
	errorResult, _ := runner.QueryResult(context.Background(), "DELETE FROM sessions")
	if errorResult.Complete || errorResult.Error == "" {
		t.Fatalf("error marked complete: %+v", errorResult)
	}
}

func TestQueryCellLimitAliasAndJSON(t *testing.T) {
	legacy := WithDefaultQueryOptions(QueryOptions{MaxCellChars: 3})
	if legacy.MaxCellBytes != 3 {
		t.Fatalf("legacy alias not resolved: %+v", legacy)
	}
	preferred := WithDefaultQueryOptions(QueryOptions{MaxCellChars: 2, MaxCellBytes: 4})
	if preferred.MaxCellBytes != 4 {
		t.Fatalf("explicit bytes should win: %+v", preferred)
	}
	runner := setupQueryRunner(t)
	runner.opts = WithDefaultQueryOptions(QueryOptions{MaxCellBytes: 8, StrictEvidence: true})
	result, _ := runner.QueryResult(context.Background(), "SELECT ? AS args", `{"query":"`+strings.Repeat("x", 17000)+`"}`)
	if result.Error == "" || result.Complete || len(result.Rows) != 0 || len(result.CellDiagnostics) != 1 {
		t.Fatalf("oversized JSON accepted: %+v", result)
	}
	runner.opts = WithDefaultQueryOptions(QueryOptions{MaxCellBytes: 3})
	blob, _ := runner.QueryResult(context.Background(), "SELECT cast(? AS blob) AS text", "abé")
	if blob.Rows[0]["text"] != "ab" || blob.Complete {
		t.Fatalf("blob not diagnosed: %+v", blob)
	}
}
