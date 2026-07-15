package convert

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

type phase0CaptureProcessor struct {
	rows []types.Row
}

var _ middlewares.Processor = (*phase0CaptureProcessor)(nil)

func (p *phase0CaptureProcessor) AddRow(_ context.Context, row types.Row) error {
	p.rows = append(p.rows, row)
	return nil
}

func (p *phase0CaptureProcessor) Close(context.Context) error { return nil }

// TestConvertCodexPublishesSuccessfulSourcesWhenAnotherSourceFails captures
// the current partial-batch behavior for P0.7. P1 must replace this with an
// explicit strict/allow-partial contract and a durable batch receipt.
func TestConvertCodexPublishesSuccessfulSourcesWhenAnotherSourceFails(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.jsonl")
	invalid := filepath.Join(dir, "invalid.jsonl")
	if err := os.WriteFile(valid, []byte(`{"type":"session_meta","payload":{"id":"valid-session"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("writing valid source: %v", err)
	}
	if err := os.WriteFile(invalid, []byte(`{"type":"unrecognized","payload":{}}`+"\n"), 0o644); err != nil {
		t.Fatalf("writing invalid source: %v", err)
	}

	command, err := NewConvertCodexGlazeCommand()
	if err != nil {
		t.Fatalf("NewConvertCodexGlazeCommand returned error: %v", err)
	}
	outputDir := filepath.Join(dir, "output")
	values, err := runner.ParseCommandValues(command, runner.WithValuesForSections(map[string]map[string]any{
		schema.DefaultSlug: {
			"source-session": []string{valid, invalid},
			"output-dir":     outputDir,
		},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}

	processor := &phase0CaptureProcessor{}
	if err := command.RunIntoGlazeProcessor(context.Background(), values, processor); err != nil {
		t.Fatalf("RunIntoGlazeProcessor returned error: %v; current behavior returns success when at least one source converts", err)
	}

	archive := filepath.Join(outputDir, "active", "unknown", "valid-session.minitrace.json")
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("expected successful source archive after partial conversion: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "manifest.json")); err != nil {
		t.Fatalf("expected manifest after partial conversion: %v", err)
	}
	if len(processor.rows) != 3 {
		t.Fatalf("row count = %d, want 3 (success, failed source, manifest)", len(processor.rows))
	}
}
