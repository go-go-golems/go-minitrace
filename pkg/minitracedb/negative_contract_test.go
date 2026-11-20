package minitracedb

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestQueryErrorEnvelopesHaveEmptyArrays(t *testing.T) {
	cases := []struct {
		name, sql string
		configure func(*QueryRunner)
	}{
		{"empty", "", nil},
		{"write", "DELETE FROM sessions", nil},
		{"multiple", "SELECT 1; SELECT 2", nil},
		{"denied", "SELECT name FROM sqlite_master", nil},
		{"invalid-column", "SELECT nonexistent FROM sessions", nil},
		{"binding", "SELECT ?", nil},
		{"column-limit", "SELECT 1 AS a, 2 AS b", func(r *QueryRunner) { r.opts.MaxColumns = 1 }},
		{"closed-db", "SELECT 1", func(r *QueryRunner) { _ = r.db.Close() }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := setupQueryRunner(t)
			if tt.configure != nil {
				tt.configure(r)
			}
			got, err := r.QueryResult(context.Background(), tt.sql)
			assertErrorArrays(t, got, err)
		})
	}
	var r *QueryRunner
	got, err := r.QueryResult(context.Background(), "SELECT 1")
	assertErrorArrays(t, got, err)
}

func assertErrorArrays(t *testing.T, got QueryResult, err error) {
	t.Helper()
	if err != nil || got.Error == "" || got.Complete || got.Count != 0 || got.Rows == nil || got.Columns == nil || len(got.Rows) != 0 || len(got.Columns) != 0 {
		t.Fatalf("invalid error envelope: %#v, %v", got, err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var plain map[string]any
	if err = json.Unmarshal(data, &plain); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"rows", "columns"} {
		if a, ok := plain[key].([]any); !ok || len(a) != 0 {
			t.Fatalf("%s is not [] in %s", key, data)
		}
	}
}

func TestMalformedNormalizedSessionPreservesTypedError(t *testing.T) {
	for _, auto := range []bool{false, true} {
		_, err := LoadSessionContentAuto([]byte(`{"id":"broken","turns":[],"metrics":[]}`), LoadOptions{SourceName: "broken.json", AutoConvert: auto})
		var typed *json.UnmarshalTypeError
		if !errors.As(err, &typed) || typed.Field != "metrics" {
			t.Fatalf("auto=%v: expected typed metrics error, got %v", auto, err)
		}
		if !strings.Contains(err.Error(), "decode normalized minitrace session broken.json") || strings.Contains(err.Error(), "AutoConvert is disabled") {
			t.Fatalf("unhelpful error: %v", err)
		}
	}
}

func TestNormalizedErrorDetectionDoesNotCaptureNativeRecords(t *testing.T) {
	records := []map[string]any{
		{"type": "session", "id": "pi", "version": 3},
		{"role": "user", "id": "pi-role", "content": "hello"},
		{"type": "session_meta", "id": "cx", "payload": map[string]any{"id": "cx"}},
		{"type": "user", "id": "cc", "message": map[string]any{"content": "hello"}},
		{"type": "session.start", "id": "cp", "data": map[string]any{"sessionId": "cp"}},
	}
	for _, record := range records {
		// These extra keys intentionally resemble an invalid archive. Native
		// discriminators must still take priority when deciding the diagnostic.
		record["turns"] = []any{}
		record["metrics"] = []any{}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if looksLikeNormalizedSession(data) {
			t.Fatalf("native record classified as archive: %s", data)
		}
		loaded, err := LoadSessionContentAuto(data, LoadOptions{SourceName: "native.jsonl", AutoConvert: true})
		if err != nil || loaded.Adapter == "native" {
			t.Fatalf("native conversion changed: %s: %#v %v", data, loaded, err)
		}
	}
	for _, payload := range []string{`{"id":"arbitrary","turns":[]}`, `{"id":"session","type":"session"}`, "{\"id\":\"x\"}\n{\"type\":\"message\"}"} {
		if looksLikeNormalizedSession([]byte(payload)) {
			t.Fatalf("overbroad archive recognition: %s", payload)
		}
	}
}
