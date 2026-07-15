package convert

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-minitrace/pkg/adapters"
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

func phase0RowToMap(row types.Row) map[string]any {
	ret := map[string]any{}
	for pair := row.Oldest(); pair != nil; pair = pair.Next() {
		ret[pair.Key] = pair.Value
	}
	return ret
}

// TestConvertCodexPublishesSuccessfulSourcesWhenAnotherSourceFails captures
// the current partial-batch behavior for P0.7. P1 must replace this with an
// explicit strict/allow-partial contract and a durable batch receipt.
func TestPreflightCodexLocatorsRejectsConflictingNativeIDsBeforePublication(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.jsonl")
	second := filepath.Join(dir, "second.jsonl")
	if err := os.WriteFile(first, []byte(`{"type":"session_meta","payload":{"id":"same-native-id"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("writing first source: %v", err)
	}
	if err := os.WriteFile(second, []byte(`{"type":"session_meta","payload":{"id":"same-native-id","cwd":"/different"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("writing second source: %v", err)
	}
	if _, err := preflightCodexLocators([]adapters.SessionLocator{{SourcePath: first}, {SourcePath: second}}); err == nil {
		t.Fatalf("expected conflicting native-ID fingerprints to fail preflight")
	}
}

func TestPreflightCodexLocatorsCollapsesIdenticalNativeSources(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.jsonl")
	second := filepath.Join(dir, "second.jsonl")
	payload := []byte(`{"type":"session_meta","payload":{"id":"same-native-id"}}` + "\n")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			t.Fatalf("writing source %s: %v", path, err)
		}
	}
	locators, err := preflightCodexLocators([]adapters.SessionLocator{{SourcePath: second}, {SourcePath: first}})
	if err != nil {
		t.Fatalf("preflightCodexLocators returned error: %v", err)
	}
	if len(locators) != 1 || locators[0].Identity == nil || locators[0].Identity.NativeSessionID != "same-native-id" {
		t.Fatalf("unexpected preflight result: %+v", locators)
	}
}

func TestConvertCodexEmitsPreflightProvenance(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.jsonl")
	if err := os.WriteFile(source, []byte(`{"type":"session_meta","payload":{"id":"child-id","parent_thread_id":"parent-id"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}
	command, err := NewConvertCodexGlazeCommand()
	if err != nil {
		t.Fatalf("NewConvertCodexGlazeCommand returned error: %v", err)
	}
	runRecordPath := filepath.Join(dir, "runs", "conversion.json")
	values, err := runner.ParseCommandValues(command, runner.WithValuesForSections(map[string]map[string]any{
		schema.DefaultSlug: {"source-session": []string{source}, "output-dir": filepath.Join(dir, "output"), "run-record": runRecordPath},
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues returned error: %v", err)
	}
	processor := &phase0CaptureProcessor{}
	if err := command.RunIntoGlazeProcessor(context.Background(), values, processor); err != nil {
		t.Fatalf("RunIntoGlazeProcessor returned error: %v", err)
	}
	if len(processor.rows) < 1 {
		t.Fatalf("expected conversion row")
	}
	row := phase0RowToMap(processor.rows[0])
	if row["identity_basis"] != "first-session-meta" || row["parent_session_id"] != "parent-id" || row["source_fingerprint"] == "" {
		t.Fatalf("missing provenance columns: %#v", row)
	}
	payload, err := os.ReadFile(runRecordPath)
	if err != nil {
		t.Fatalf("reading run record: %v", err)
	}
	var record conversionRunRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("decoding run record: %v", err)
	}
	if record.Schema != "go-minitrace-conversion-run-v1" || !record.Complete || len(record.Inputs) != 1 || len(record.Outputs) != 1 {
		t.Fatalf("unexpected run record: %+v", record)
	}
}

func TestConvertCodexPreflightRejectsInvalidSourceBeforePublication(t *testing.T) {
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
	if err := command.RunIntoGlazeProcessor(context.Background(), values, processor); err == nil {
		t.Fatalf("expected invalid source to fail preflight")
	}

	archive := filepath.Join(outputDir, "active", "unknown", "valid-session.minitrace.json")
	if _, err := os.Stat(archive); !os.IsNotExist(err) {
		t.Fatalf("preflight published archive before rejecting batch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("preflight published manifest before rejecting batch: %v", err)
	}
	if len(processor.rows) != 0 {
		t.Fatalf("row count = %d, want 0 before preflight failure", len(processor.rows))
	}
}
