package annotate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// closeStore silences the errcheck linter for Store.Close().
func closeStore(s *Store) { _ = s.Close() }

func TestOpenCreatesDB(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	// DB file should exist.
	dbPath := filepath.Join(dir, "annotations.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("annotations.db not created at %s", dbPath)
	}
}

func TestOpenMigrations(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	// Re-open should not fail (idempotent migration).
	store2, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	closeStore(store2)
}

func TestAddAndGet(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	ann := minitrace.Annotation{
		ID:               "ann-001",
		Timestamp:        minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator:        "tester",
		Scope:            minitrace.AnnotationScope{Type: "session", TargetID: "sess-001"},
		Content:          minitrace.AnnotationContent{Category: "observation", Title: "First", Detail: "Note"},
		TaxonomyMappings: minitrace.TaxonomyMappings{Minitrace: []string{"O-INT"}},
	}
	err = store.AddAnnotation(ctx, ann, "sess-001")
	if err != nil {
		t.Fatalf("AddAnnotation: %v", err)
	}

	got, err := store.GetAnnotationsForSession(ctx, "sess-001")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(got))
	}
	if got[0].ID != "ann-001" {
		t.Errorf("ID = %q, want %q", got[0].ID, "ann-001")
	}
	if got[0].Content.Category != "observation" {
		t.Errorf("Category = %q, want %q", got[0].Content.Category, "observation")
	}
	if len(got[0].TaxonomyMappings.Minitrace) != 1 {
		t.Errorf("TaxonomyMinitrace = %v, want [O-INT]", got[0].TaxonomyMappings.Minitrace)
	}
}

func TestAddAndGetUnknownSession(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	ann := minitrace.Annotation{
		ID:        "ann-002",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-unknown"},
		Content:   minitrace.AnnotationContent{Category: "ai-failure", Title: "Miss"},
	}
	if err := store.AddAnnotation(ctx, ann, "sess-unknown"); err != nil {
		t.Fatalf("AddAnnotation: %v", err)
	}

	got, err := store.GetAnnotationsForSession(ctx, "sess-unknown")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	for i := 0; i < 5; i++ {
		id := "ann-list-" + string(rune('a'+i))
		ann := minitrace.Annotation{
			ID:               id,
			Timestamp:        minitrace.FormatTimestamp(time.Now().UTC()),
			Annotator:        "tester",
			Scope:            minitrace.AnnotationScope{Type: "session", TargetID: "sess-001"},
			Content:          minitrace.AnnotationContent{Category: "observation", Title: id, Tags: []string{"tag1"}},
			TaxonomyMappings: minitrace.TaxonomyMappings{Minitrace: []string{"O-INT"}},
		}
		if err := store.AddAnnotation(ctx, ann, "sess-001"); err != nil {
			t.Fatalf("AddAnnotation %s: %v", id, err)
		}
	}

	// List all for session.
	all, err := store.List(ctx, ListOptions{SessionID: "sess-001", Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5, got %d", len(all))
	}

	// List with category filter.
	obs, err := store.List(ctx, ListOptions{SessionID: "sess-001", Category: "observation", Limit: 10})
	if err != nil {
		t.Fatalf("List with category: %v", err)
	}
	if len(obs) != 5 {
		t.Errorf("expected 5 observations, got %d", len(obs))
	}

	// Empty result.
	empty, err := store.List(ctx, ListOptions{SessionID: "sess-001", Category: "nonexistent"})
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}

	// Taxonomy filter.
	tax, err := store.List(ctx, ListOptions{SessionID: "sess-001", Taxonomy: "O-INT"})
	if err != nil {
		t.Fatalf("List with taxonomy: %v", err)
	}
	if len(tax) != 5 {
		t.Errorf("expected 5, got %d", len(tax))
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	ann := minitrace.Annotation{
		ID:        "ann-upd-001",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-001"},
		Content:   minitrace.AnnotationContent{Category: "ai-failure", Title: "Original"},
	}
	if err := store.AddAnnotation(ctx, ann, "sess-001"); err != nil {
		t.Fatalf("AddAnnotation: %v", err)
	}

	// Partial update: only title.
	newTitle := "Updated"
	patch := AnnotationPatch{Title: &newTitle}
	if err := store.Update(ctx, "ann-upd-001", patch); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.GetAnnotationsForSession(ctx, "sess-001")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Content.Title != "Updated" {
		t.Errorf("Title = %q, want %q", got[0].Content.Title, "Updated")
	}
	if got[0].Content.Category != "ai-failure" {
		t.Errorf("Category should be unchanged, got %q", got[0].Content.Category)
	}
}

func TestUpdateNotFound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	title := "x"
	err = store.Update(ctx, "does-not-exist", AnnotationPatch{Title: &title})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	ann := minitrace.Annotation{
		ID:        "ann-del-001",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-001"},
		Content:   minitrace.AnnotationContent{Category: "observation", Title: "ToDelete"},
	}
	if err := store.AddAnnotation(ctx, ann, "sess-001"); err != nil {
		t.Fatalf("AddAnnotation: %v", err)
	}

	if err := store.Delete(ctx, "ann-del-001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := store.GetAnnotationsForSession(ctx, "sess-001")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(got))
	}
}

func TestDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	err = store.Delete(ctx, "does-not-exist")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMarkSyncedAndUnsynced(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	// Add two annotations for the same session.
	ann1 := minitrace.Annotation{
		ID:        "ann-sync-1",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-sync"},
		Content:   minitrace.AnnotationContent{Category: "observation", Title: "A"},
	}
	ann2 := minitrace.Annotation{
		ID:        "ann-sync-2",
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: "tester",
		Scope:     minitrace.AnnotationScope{Type: "session", TargetID: "sess-sync"},
		Content:   minitrace.AnnotationContent{Category: "observation", Title: "B"},
	}
	if err := store.AddAnnotation(ctx, ann1, "sess-sync"); err != nil {
		t.Fatalf("AddAnnotation 1: %v", err)
	}
	if err := store.AddAnnotation(ctx, ann2, "sess-sync"); err != nil {
		t.Fatalf("AddAnnotation 2: %v", err)
	}

	// GetUnsyncedSessions should include sess-sync.
	unsynced, err := store.GetUnsyncedSessions(ctx)
	if err != nil {
		t.Fatalf("GetUnsyncedSessions: %v", err)
	}
	if len(unsynced) != 1 {
		t.Errorf("expected 1 unsynced session, got %d", len(unsynced))
	}

	// Mark synced.
	if err := store.markSynced(ctx, "sess-sync", 2); err != nil {
		t.Fatalf("markSynced: %v", err)
	}

	// After marking synced, GetUnsyncedSessions should still return the row
	// (it returns all sessions that have annotations, regardless of sync state).
	// The sync state is tracked in sync_state; the caller is responsible for
	// comparing synced_at against the file's timestamp.
	unsynced2, err := store.GetUnsyncedSessions(ctx)
	if err != nil {
		t.Fatalf("GetUnsyncedSessions after sync: %v", err)
	}
	if len(unsynced2) != 1 {
		t.Errorf("expected 1 session entry, got %d", len(unsynced2))
	}
}

func TestNilTagsAndTaxonomy(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeStore(store)

	// Annotation with nil/empty taxonomy and tags.
	ann := minitrace.Annotation{
		ID:               "ann-nil",
		Timestamp:        minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator:        "tester",
		Scope:            minitrace.AnnotationScope{Type: "session", TargetID: "sess-001"},
		Content:          minitrace.AnnotationContent{Category: "observation", Title: "NilTags"},
		TaxonomyMappings: minitrace.TaxonomyMappings{}, // empty, not nil
	}
	if err := store.AddAnnotation(ctx, ann, "sess-001"); err != nil {
		t.Fatalf("AddAnnotation: %v", err)
	}

	got, err := store.GetAnnotationsForSession(ctx, "sess-001")
	if err != nil {
		t.Fatalf("GetAnnotationsForSession: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// Tags and taxonomy are stored as '[]', so they round-trip as empty slices, not nil.
	if got[0].Content.Tags == nil {
		t.Errorf("Tags should be [] not nil after round-trip")
	}
	if len(got[0].Content.Tags) != 0 {
		t.Errorf("Tags = %v, want []", got[0].Content.Tags)
	}
	if got[0].TaxonomyMappings.Minitrace == nil {
		t.Errorf("TaxonomyMinitrace should not be nil after round-trip")
	}
	if len(got[0].TaxonomyMappings.Minitrace) != 0 {
		t.Errorf("TaxonomyMinitrace = %v, want []", got[0].TaxonomyMappings.Minitrace)
	}
}
