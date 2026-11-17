package query

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrictEvidenceCLIRejectsCellsBeforeEmission(t *testing.T) {
	archive := writeAdvancedFixtureArchive(t)
	receipt := filepath.Join(t.TempDir(), "receipt.json")
	command, values := mustRunQueryValues(t, archive, map[string]any{
		"sql": "SELECT 'abé' AS text", "max-cell-bytes": 3, "strict-evidence": true, "cache-mode": "memory", "run-record": receipt,
	})
	gp := &captureProcessor{}
	err := command.RunIntoGlazeProcessor(context.Background(), values, gp)
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete evidence error: %v", err)
	}
	if len(gp.rows) != 0 {
		t.Fatalf("strict mode emitted rows: %+v", gp.rows)
	}
	raw, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var record queryRunRecord
	if err = json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.Complete || !record.Truncated || len(record.CellDiagnostics) != 1 || record.Status != "failed" {
		t.Fatalf("receipt: %+v", record)
	}
}

func TestRootJSCommandShowsCustomRequiredFieldAndRuntimeFlags(t *testing.T) {
	setIsolatedConfigHome(t)
	dir := t.TempDir()
	source := `__section__("filters", {fields: {source: {type:"string",required:true}}});
 function probe(filters){return {source:filters.source}};
 __verb__("probe", {name:"probe",short:"Probe",sections:["filters"]});`
	if err := os.WriteFile(filepath.Join(dir, "probe.js"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	cmd, err := NewCommandsCommand([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"probe", "--help"})
	if err = cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--source", "--archive-glob"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("missing %s in help: %s", want, buf.String())
		}
	}
}

func TestValidCommandExecutesDespiteInvalidSibling(t *testing.T) {
	setIsolatedConfigHome(t)
	dir := t.TempDir()
	for name, source := range map[string]string{
		"good.js": `function good(){return {answer:42}}; __verb__("good",{name:"good",short:"Good"});`,
		"bad.js":  `function bad(){}; __verb__("bad",{name:"bad",short:"Bad",fields:{limit:{type:"int",default:-1}}});`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd, err := NewCommandsCommand([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"good", "--output", "json"})
	if err = cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestResultEnvelopePreservesMetadataAndEmptyResults(t *testing.T) {
	archive := writeAdvancedFixtureArchive(t)
	for _, tc := range []struct {
		name, sql string
		strict    bool
		wantCount int
		complete  bool
	}{
		{"clipped", "SELECT 'abé' AS text", false, 1, false},
		{"empty", "SELECT 1 AS n WHERE 0", false, 0, true},
		{"strict", "SELECT 'abé' AS text", true, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command, values := mustRunQueryValues(t, archive, map[string]any{
				"sql": tc.sql, "max-cell-bytes": 3, "strict-evidence": tc.strict, "cache-mode": "memory", "result-envelope": true,
			})
			gp := &captureProcessor{}
			err := command.RunIntoGlazeProcessor(context.Background(), values, gp)
			if tc.strict {
				if err == nil || len(gp.rows) != 0 {
					t.Fatalf("strict emission: %v %+v", err, gp.rows)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			rows := capturedRowMaps(gp)
			if len(rows) != 1 {
				t.Fatalf("envelope count: %d", len(rows))
			}
			payload, err := json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
			var envelopes []struct {
				Count           int              `json:"count"`
				Complete        bool             `json:"complete"`
				Rows            []map[string]any `json:"rows"`
				CellDiagnostics []any            `json:"cellDiagnostics"`
			}
			if err = json.Unmarshal(payload, &envelopes); err != nil {
				t.Fatal(err)
			}
			got := envelopes[0]
			if got.Count != tc.wantCount || got.Complete != tc.complete || got.Rows == nil {
				t.Fatalf("envelope: %s", payload)
			}
			if !tc.complete && len(got.CellDiagnostics) != 1 {
				t.Fatalf("missing diagnostics: %s", payload)
			}
		})
	}
}

func TestEvidenceWorkflowNestedExample(t *testing.T) {
	setIsolatedConfigHome(t)
	archive := writeAdvancedFixtureArchive(t)
	repo, err := filepath.Abs("../../../../testdata/query-repositories/evidence-workflows")
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := NewCommandsCommand([]string{repo})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"analysis", "session-tools", "counts", "--archive-glob", archive, "--output", "json"})
	if err = cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
