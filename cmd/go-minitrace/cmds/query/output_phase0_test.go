package query

import (
	"bytes"
	"context"
	"testing"

	jsonformatter "github.com/go-go-golems/glazed/pkg/formatters/json"
)

// TestStreamingJSONFormatterEmitsNoDocumentForZeroRows records the formatter
// behavior that caused downstream JSONDecodeError in the holdout evaluation.
// P3 must change this contract to emit exactly []\n in JSON-array mode.
func TestStreamingJSONFormatterEmitsNoDocumentForZeroRows(t *testing.T) {
	formatter := jsonformatter.NewOutputFormatter()
	var output bytes.Buffer
	if err := formatter.Close(context.Background(), &output); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if got := output.String(); got != "" {
		t.Fatalf("zero-row streaming JSON output = %q, want empty current behavior", got)
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
