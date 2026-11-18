package queryplan_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
	_ "github.com/mattn/go-sqlite3"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

var stringType = queryplan.Type{Kind: queryplan.String}

func TestCompiledQueryAgainstSQLite(t *testing.T) {
	ctx := context.Background()
	db := must(sql.Open("sqlite3", ":memory:"))
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE sessions (id TEXT, title TEXT, stored_count INTEGER);
		CREATE TABLE tools (id TEXT);
		INSERT INTO sessions VALUES ('a', 'plain', 123), ('b', NULL, 999), ('c', 'plain', 456);
		INSERT INTO tools VALUES ('a'), ('a'), ('c');
	`)
	if err != nil {
		t.Fatal(err)
	}
	runner := must(minitracedb.NewQueryRunner(db, []string{"sessions", "tools"}, minitracedb.QueryOptions{}))
	owner := queryplan.NewOwner()
	sessions := must(queryplan.Table(owner, "sessions", []queryplan.Column{
		{Name: "id", Type: stringType}, {Name: "title", Type: queryplan.Type{Kind: queryplan.String, Nullable: true}},
	}))
	tools := must(queryplan.Table(owner, "tools", []queryplan.Column{{Name: "id", Type: stringType}}))
	linked := must(tools.Where(must(queryplan.Operation("eq", must(tools.Column("id")), must(sessions.Column("id"))))))
	ordered := must(sessions.OrderBy(queryplan.Order{Value: must(sessions.Column("id")), Descending: true}))
	projection := must(ordered.Select([]queryplan.Field{
		{Name: "session", Value: must(ordered.Column("id"))},
		{Name: "calls", Value: linkedFor(ordered, tools).Count()},
		{Name: "label", Value: must(queryplan.Literal("a'b?"))},
	}))
	filtered := must(projection.Where(must(queryplan.Operation("gte", must(projection.Column("calls")), must(queryplan.Literal(0))))))
	plan := must(filtered.Build())
	result := must(runner.QueryResult(ctx, plan.SQL(), plan.Args()...))
	if result.Error != "" || !result.Complete || len(result.Rows) != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	for i, want := range []struct {
		id    string
		count int64
	}{{"c", 1}, {"b", 0}, {"a", 2}} {
		row := result.Rows[i]
		if row["session"] != want.id || row["calls"] != want.count || row["label"] != "a'b?" {
			t.Fatalf("row %d = %#v, expected %v", i, row, want)
		}
	}
	if _, err := linked.Build(); err == nil || !strings.Contains(err.Error(), "outside its row scope") {
		t.Fatalf("escaped correlated query: %v", err)
	}
	if plan.Owner() != owner || queryplan.NewOwner() == owner {
		t.Fatal("invalid database ownership")
	}
}

func linkedFor(sessions, tools *queryplan.Query) *queryplan.Query {
	return must(tools.Where(must(queryplan.Operation("eq", must(tools.Column("id")), must(sessions.Column("id"))))))
}

func TestBranchAndOrderValidation(t *testing.T) {
	owner := queryplan.NewOwner()
	columns := []queryplan.Column{{Name: "id", Type: stringType}}
	base := must(queryplan.Table(owner, "sessions", columns))
	columns[0].Name = "mutated"
	id := must(base.Column("id"))
	ordered := must(base.OrderBy(queryplan.Order{Value: id}))
	if _, err := ordered.Select([]queryplan.Field{{Name: "label", Value: must(queryplan.Literal("x"))}}); err == nil {
		t.Fatal("projection removed ordering key")
	}
	replaced := must(ordered.OrderBy(queryplan.Order{Value: must(ordered.Column("id")), Descending: true}))
	if strings.Contains(must(base.Build()).SQL(), "ORDER BY") || !strings.HasSuffix(must(replaced.Build()).SQL(), " DESC") {
		t.Fatal("order replacement changed base or failed to replace the sort")
	}
	if _, err := ordered.OrderBy(queryplan.Order{Value: id}); err == nil {
		t.Fatal("order accepted an escaped scope")
	}
	other := must(queryplan.Table(queryplan.NewOwner(), "tools", columns))
	combined := must(base.Select([]queryplan.Field{{Name: "calls", Value: other.Count()}}))
	if _, err := combined.Build(); err == nil || !strings.Contains(err.Error(), "different database") {
		t.Fatalf("cross-database count: %v", err)
	}
}
