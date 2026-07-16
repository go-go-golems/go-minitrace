package query

import (
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
