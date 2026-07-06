package query

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	glazedvalues "github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestRunQueryCommand_ExecutesInlineSQL(t *testing.T) {
	archiveGlob := writeAdvancedFixtureArchive(t)
	rows := runSQLiteQueryRows(t, archiveGlob, map[string]any{
		"sql": "SELECT COUNT(*) AS sessions FROM sessions",
	})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0]["sessions"] != int64(3) {
		t.Fatalf("sessions = %#v, want 3", rows[0]["sessions"])
	}
}

func TestRunQueryCommand_ExecutesPreset(t *testing.T) {
	archiveGlob := writeAdvancedFixtureArchive(t)
	rows := runSQLiteQueryRows(t, archiveGlob, map[string]any{
		"preset": "overview/session-list",
	})
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	if rows[0]["id"] != "fixture-alpha-1" {
		t.Fatalf("first id = %#v, want fixture-alpha-1", rows[0]["id"])
	}
	if rows[0]["framework"] != "pi" {
		t.Fatalf("framework = %#v, want pi", rows[0]["framework"])
	}
}

func TestRunQueryCommand_RejectsWriteSQL(t *testing.T) {
	archiveGlob := writeAdvancedFixtureArchive(t)
	cmd, parsedValues := mustRunQueryValues(t, archiveGlob, map[string]any{
		"sql": "DELETE FROM sessions",
	})
	gp := &captureProcessor{}
	err := cmd.RunIntoGlazeProcessor(context.Background(), parsedValues, gp)
	if err == nil || !strings.Contains(err.Error(), "only SELECT and WITH queries are allowed") {
		t.Fatalf("err = %v, want read-only rejection", err)
	}
}

func TestRunQueryCommand_RequiresExactlyOneQuerySource(t *testing.T) {
	archiveGlob := writeAdvancedFixtureArchive(t)
	cmd, parsedValues := mustRunQueryValues(t, archiveGlob, map[string]any{})
	gp := &captureProcessor{}
	if err := cmd.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err == nil {
		t.Fatalf("expected error when no query source is given")
	}

	cmd, parsedValues = mustRunQueryValues(t, archiveGlob, map[string]any{
		"sql":    "SELECT 1 AS one FROM sessions",
		"preset": "session-list",
	})
	if err := cmd.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err == nil {
		t.Fatalf("expected error when preset and sql are combined")
	}
}

// TestQueryRunPresetGoldenRows pins the rewritten presets to golden rows over
// the fixture archive. These expectations were validated against the legacy
// DuckDB presets by the (now-retired) old-vs-new parity harness before the
// DuckDB engine was removed.
func TestQueryRunPresetGoldenRows(t *testing.T) {
	archiveGlob := writeAdvancedFixtureArchive(t)

	t.Run("overview/session-list", func(t *testing.T) {
		rows := runSQLiteQueryRows(t, archiveGlob, map[string]any{"preset": "overview/session-list"})
		if len(rows) != 3 {
			t.Fatalf("len(rows) = %d, want 3", len(rows))
		}
		sortRowsByColumns(rows, []string{"id"})
		wantIDs := []string{"fixture-alpha-1", "fixture-alpha-2", "fixture-beta-1"}
		wantTools := []int64{6, 4, 4}
		for i := range rows {
			if rows[i]["id"] != wantIDs[i] {
				t.Fatalf("row %d id = %#v, want %s", i, rows[i]["id"], wantIDs[i])
			}
			if rows[i]["framework"] != "pi" {
				t.Fatalf("row %d framework = %#v, want pi", i, rows[i]["framework"])
			}
			if !parityValuesAgree(rows[i]["tools"], wantTools[i]) {
				t.Fatalf("row %d tools = %#v, want %d", i, rows[i]["tools"], wantTools[i])
			}
		}
	})

	t.Run("overview/framework-summary", func(t *testing.T) {
		rows := runSQLiteQueryRows(t, archiveGlob, map[string]any{"preset": "overview/framework-summary"})
		if len(rows) != 1 {
			t.Fatalf("len(rows) = %d, want 1", len(rows))
		}
		if rows[0]["framework"] != "pi" {
			t.Fatalf("framework = %#v, want pi", rows[0]["framework"])
		}
		if !parityValuesAgree(rows[0]["sessions"], int64(3)) {
			t.Fatalf("sessions = %#v, want 3", rows[0]["sessions"])
		}
	})

	t.Run("tools/tool-operation-breakdown", func(t *testing.T) {
		rows := runSQLiteQueryRows(t, archiveGlob, map[string]any{"preset": "tools/tool-operation-breakdown"})
		counts := map[string]any{}
		for _, row := range rows {
			counts[fmt.Sprint(row["framework"], "/", row["operation"])] = row["count"]
		}
		want := map[string]int64{"pi/EXECUTE": 6, "pi/read": 5, "pi/modify": 3}
		if len(counts) != len(want) {
			t.Fatalf("got %d operation rows %#v, want %d", len(counts), counts, len(want))
		}
		for key, wantCount := range want {
			if !parityValuesAgree(counts[key], wantCount) {
				t.Fatalf("count[%s] = %#v, want %d", key, counts[key], wantCount)
			}
		}
	})

	t.Run("tools/tool-failures", func(t *testing.T) {
		rows := runSQLiteQueryRows(t, archiveGlob, map[string]any{"preset": "tools/tool-failures"})
		if len(rows) != 0 {
			t.Fatalf("len(rows) = %d, want 0 (fixture has no failed tool calls)", len(rows))
		}
	})
}

func runSQLiteQueryRows(t *testing.T, archiveGlob string, values map[string]any) []map[string]any {
	t.Helper()
	cmd, parsedValues := mustRunQueryValues(t, archiveGlob, values)
	gp := &captureProcessor{}
	if err := cmd.RunIntoGlazeProcessor(context.Background(), parsedValues, gp); err != nil {
		t.Fatalf("RunIntoGlazeProcessor(%v): %v", values, err)
	}
	return capturedRowMaps(gp)
}

func mustRunQueryValues(t *testing.T, archiveGlob string, values map[string]any) (*RunQueryCommand, *glazedvalues.Values) {
	t.Helper()
	cmd, err := NewRunQueryGlazeCommand()
	if err != nil {
		t.Fatalf("NewRunQueryGlazeCommand: %v", err)
	}
	sectionValues := map[string]any{"archive-glob": []string{archiveGlob}}
	for key, value := range values {
		sectionValues[key] = value
	}
	parsedValues, err := runner.ParseCommandValues(cmd, runner.WithValuesForSections(map[string]map[string]any{
		schema.DefaultSlug: sectionValues,
	}))
	if err != nil {
		t.Fatalf("ParseCommandValues: %v", err)
	}
	return cmd, parsedValues
}

func capturedRowMaps(gp *captureProcessor) []map[string]any {
	ret := make([]map[string]any, 0, len(gp.rows))
	for _, row := range gp.rows {
		ret = append(ret, rowToMap(row))
	}
	return ret
}

func sortRowsByColumns(rows []map[string]any, columns []string) {
	sort.Slice(rows, func(i, j int) bool {
		return rowSortKey(rows[i], columns) < rowSortKey(rows[j], columns)
	})
}

func rowSortKey(row map[string]any, columns []string) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, fmt.Sprint(row[column]))
	}
	return strings.Join(parts, "\x00")
}

func parityValuesAgree(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	af, aok := parityFloat(a)
	bf, bok := parityFloat(b)
	if aok && bok {
		return math.Abs(af-bf) < 1e-6
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func parityFloat(v any) (float64, bool) {
	switch typed := v.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case string:
		f, err := strconv.ParseFloat(typed, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
