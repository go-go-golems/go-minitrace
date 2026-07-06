package minitracedb

import (
	"context"
	"testing"
)

func TestListPresetsMatchesLegacyPresetSet(t *testing.T) {
	want := []string{
		"files/file-operations",
		"files/file-timeline",
		"overview/annotations",
		"overview/framework-summary",
		"overview/session-list",
		"timing/timing-analysis",
		"tools/read-ratio-distribution",
		"tools/tool-failures",
		"tools/tool-operation-breakdown",
	}
	got := ListPresets()
	if len(got) != len(want) {
		t.Fatalf("len(presets) = %d, want %d: %v", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i] != id {
			t.Fatalf("presets[%d] = %q, want %q", i, got[i], id)
		}
	}
}

func TestPresetsExecuteOnEmptyNormalizedDB(t *testing.T) {
	ctx := context.Background()
	db, err := OpenSQLiteMemory(ctx, "presets_test")
	if err != nil {
		t.Fatalf("OpenSQLiteMemory: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	runner, err := NewQueryRunner(db, AllowedObjectNames(), QueryOptions{})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}

	for _, id := range ListPresets() {
		sqlText, err := ResolvePresetSQL(id)
		if err != nil {
			t.Fatalf("ResolvePresetSQL(%s): %v", id, err)
		}
		result, err := runner.QueryResult(ctx, sqlText)
		if err != nil {
			t.Fatalf("QueryResult(%s): %v", id, err)
		}
		if result.Error != "" {
			t.Fatalf("preset %s failed: %s", id, result.Error)
		}
	}
}

func TestResolvePresetSQLResolvesBareNamesAndRejectsUnknown(t *testing.T) {
	if _, err := ResolvePresetSQL("session-list"); err != nil {
		t.Fatalf("ResolvePresetSQL(session-list): %v", err)
	}
	if _, err := ResolvePresetSQL("no-such-preset"); err == nil {
		t.Fatalf("expected error for unknown preset")
	}
}

func TestSessionsBaseCompatViewExposesLegacyColumns(t *testing.T) {
	ctx := context.Background()
	db, err := OpenSQLiteMemory(ctx, "compat_view_test")
	if err != nil {
		t.Fatalf("OpenSQLiteMemory: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions(session_id, title, raw_json) VALUES ('s1', 'Fixture', '{"environment":{"agent_framework":"pi"},"metrics":{"tool_call_count":3}}')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	runner, err := NewQueryRunner(db, AllowedObjectNames(), QueryOptions{})
	if err != nil {
		t.Fatalf("NewQueryRunner: %v", err)
	}
	row, err := runner.QueryOne(ctx, `SELECT id, title, environment->>'agent_framework' AS framework, CAST(metrics->>'tool_call_count' AS INT) AS tools FROM sessions_base`)
	if err != nil {
		t.Fatalf("QueryOne on sessions_base: %v", err)
	}
	if row["id"] != "s1" || row["title"] != "Fixture" || row["framework"] != "pi" {
		t.Fatalf("unexpected compat view row: %#v", row)
	}
	if row["tools"] != int64(3) {
		t.Fatalf("tools = %#v, want 3", row["tools"])
	}
}
