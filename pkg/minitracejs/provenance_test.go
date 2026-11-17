package minitracejs

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

func TestRecipeTokenOriginsAndSourceProjection(t *testing.T) {
	ctx := context.Background()
	db, err := minitracedb.OpenSQLiteMemory(ctx, "recipe-provenance")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = minitracedb.CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	n := 0
	zero := 0
	body := "éééé"
	yes := true
	s := &minitrace.Session{ID: "s", Turns: []minitrace.Turn{{Index: 0, Role: "assistant", Content: "known zero", Usage: &minitrace.Usage{InputTokens: &zero}}, {Index: 1, Role: "user", Content: "unknown"}}, ToolCalls: []minitrace.ToolCall{{ID: "z-first", EmittingTurnIndex: &n, Input: minitrace.ToolCallInput{Arguments: map[string]any{"q": 1}}, Output: minitrace.ToolCallOutput{Result: &body}}, {ID: "a-second", Output: minitrace.ToolCallOutput{Success: &yes, Status: minitrace.ToolOutcomeStatus("succeeded")}}}}
	if err = minitracedb.MaterializeSession(ctx, db, s, minitracedb.MaterializeOptions{}); err != nil {
		t.Fatal(err)
	}
	runner, err := minitracedb.NewQueryRunner(db, minitracedb.AllowedTableNames(), minitracedb.QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	query := func(recipe *QueryRecipe) []map[string]any {
		t.Helper()
		r, e := runner.QueryResult(ctx, recipe.SQL, recipe.Args...)
		if e != nil || r.Error != "" {
			t.Fatalf("query: %v %s", e, r.Error)
		}
		return r.Rows
	}
	turns := query(tokenUsageRecipe("s", "turn"))
	if turns[0]["token_origin"] != "recorded_usage" || turns[1]["token_origin"] != "unknown" {
		t.Fatalf("known zero vs missing: %v", turns)
	}
	for _, group := range []string{"session", "role"} {
		rows := query(tokenUsageRecipe("s", group))
		if len(rows) == 0 || rows[0]["token_unit"] != "tokens" {
			t.Fatalf("aggregate provenance: %v", rows)
		}
	}
	tools := query(tokenUsageRecipe("s", "tool"))
	var found bool
	for _, r := range tools {
		if r["tool_call_id"] == "z-first" {
			found = true
			if r["total_tokens"] != int64(2) || r["token_origin"] != "estimated" || r["token_method"] != "round_utf8_bytes_div_4" {
				t.Fatalf("UTF8 estimation: %v", r)
			}
		}
	}
	if !found {
		t.Fatal("missing tool estimate")
	}
	recipe, e := (&QueryRecipeBuilder{kind: "toolRows", sessionID: "s"}).Build()
	if e != nil {
		t.Fatal(e)
	}
	rows := query(recipe)
	if rows[0]["source_ordinal"] != int64(0) || rows[0]["arguments_json"] != "{\"q\":1}" || rows[0]["success"] != nil || rows[0]["output_status"] != nil {
		t.Fatalf("tool provenance: %v", rows[0])
	}
	for _, r := range query(transcriptRowsRecipe("s", true)) {
		if r["role"] == "tool" && r["token_unit"] != "estimated_tokens" {
			t.Fatalf("unlabeled transcript estimate: %v", r)
		}
	}
	query(timelineRowsRecipe("s"))
}
