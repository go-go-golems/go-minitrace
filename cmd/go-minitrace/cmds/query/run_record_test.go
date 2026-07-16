package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveArchiveInventoryIsIndependentOfGlobOrder(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.minitrace.json")
	second := filepath.Join(dir, "b.minitrace.json")
	if err := os.WriteFile(first, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	unmatched := filepath.Join(dir, "missing-*.json")
	left, err := resolveArchiveInventory([]string{second, unmatched, first})
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolveArchiveInventory([]string{first, second, unmatched})
	if err != nil {
		t.Fatal(err)
	}
	if left.InventorySHA != right.InventorySHA {
		t.Fatalf("inventory hash depends on glob order: %s != %s", left.InventorySHA, right.InventorySHA)
	}
	if len(left.Files) != 2 || len(left.Unmatched) != 1 || left.Unmatched[0] != unmatched {
		t.Fatalf("unexpected inventory: %+v", left)
	}
}

func TestQueryRunRecordRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "query.json")
	record := queryRunRecord{Schema: "go-minitrace-query-run-v1", Status: "success", Query: queryProvenance{Kind: "file", SHA256: hashText("select 1")}, Columns: []string{"value"}, RowCount: 1}
	if err := writeQueryRunRecord(path, record); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded queryRunRecord
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != record.Schema || decoded.Query.SHA256 != record.Query.SHA256 || decoded.FinishedAt == "" {
		t.Fatalf("unexpected receipt: %+v", decoded)
	}
}

func TestResolveArchiveInventoryDeduplicatesFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "one.minitrace.json")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	inventory, err := resolveArchiveInventory([]string{path, filepath.Join(dir, "*.minitrace.json")})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Files) != 1 {
		t.Fatalf("expected one deduplicated file, got %+v", inventory.Files)
	}
}
