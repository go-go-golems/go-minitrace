package validate

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

type captureProcessor struct{ rows []types.Row }

var _ middlewares.Processor = (*captureProcessor)(nil)

func (p *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	p.rows = append(p.rows, row)
	return nil
}
func (p *captureProcessor) Close(context.Context) error { return nil }

func TestArchiveValidationReturnsErrorAfterEmittingFindings(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatal(err)
	}
	command, err := NewGlazeCommand()
	if err != nil {
		t.Fatal(err)
	}
	values, err := runner.ParseCommandValues(command, runner.WithValuesForSections(map[string]map[string]any{
		schema.DefaultSlug: {"path": root, "archive": true},
	}))
	if err != nil {
		t.Fatal(err)
	}
	processor := &captureProcessor{}
	if err := command.RunIntoGlazeProcessor(context.Background(), values, processor); err == nil {
		t.Fatal("expected non-zero archive validation result")
	}
	if len(processor.rows) == 0 {
		t.Fatal("expected structured findings before the command error")
	}
}

func TestArchiveValidationCanCollectErrorsWithoutFailing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatal(err)
	}
	command, err := NewGlazeCommand()
	if err != nil {
		t.Fatal(err)
	}
	values, err := runner.ParseCommandValues(command, runner.WithValuesForSections(map[string]map[string]any{
		schema.DefaultSlug: {"path": root, "archive": true, "fail-on-error": false},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := command.RunIntoGlazeProcessor(context.Background(), values, &captureProcessor{}); err != nil {
		t.Fatalf("fail-on-error=false returned error: %v", err)
	}
}
