package query

import (
	"bytes"
	"context"
	"testing"

	jsonformatter "github.com/go-go-golems/glazed/pkg/formatters/json"
)

// TestStreamingJSONFormatterEmitsEmptyArrayForZeroRows locks the valid JSON
// contract fixed upstream in Glazed: array mode is total even with no rows.
func TestStreamingJSONFormatterEmitsEmptyArrayForZeroRows(t *testing.T) {
	formatter := jsonformatter.NewOutputFormatter()
	var output bytes.Buffer
	if err := formatter.Close(context.Background(), &output); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if got := output.String(); got != "[]\n" {
		t.Fatalf("zero-row streaming JSON output = %q, want exact empty array", got)
	}
}

func TestEmitJSResultWithEmptyArrayEmitsNoRows(t *testing.T) {
	processor := &captureProcessor{}
	if err := emitJSResult(context.Background(), processor, []any{}); err != nil {
		t.Fatalf("emitJSResult returned error: %v", err)
	}
	if len(processor.rows) != 0 {
		t.Fatalf("row count = %d, want 0", len(processor.rows))
	}
}
