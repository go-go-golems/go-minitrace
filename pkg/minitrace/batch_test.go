package minitrace

import (
	"os"
	"path/filepath"
	"testing"
)

func batchTestSession(id, fingerprint, startedAt string) *Session {
	session := BuildSessionSkeleton(id, "test", "test-v1", "go-minitrace/test")
	session.Provenance.SourceFingerprint = ptr(fingerprint)
	session.Timing.StartedAt = ptr(startedAt)
	return &session
}

func TestPublishSessionBatchRejectsAllCollisionsBeforePublishing(t *testing.T) {
	outputDir := t.TempDir()
	existing := batchTestSession("collision", "old", "2026-01-01T00:00:00Z")
	if _, err := WriteSession(existing, outputDir); err != nil {
		t.Fatalf("writing existing session: %v", err)
	}

	created := batchTestSession("would-be-created", "new", "2026-02-01T00:00:00Z")
	conflict := batchTestSession("collision", "different", "2026-01-01T00:00:00Z")
	if _, err := PublishSessionBatch([]*Session{created, conflict}, outputDir, CollisionError); err == nil {
		t.Fatal("expected collision error")
	}
	createdPath := filepath.Join(outputDir, "active", "2026-02", "would-be-created.minitrace.json")
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("strict batch published an earlier session before collision: %v", err)
	}
}

func TestPublishSessionBatchReportsCreatedUnchangedAndReplaced(t *testing.T) {
	outputDir := t.TempDir()
	unchanged := batchTestSession("unchanged", "same", "2026-01-01T00:00:00Z")
	replacedOld := batchTestSession("replaced", "old", "2026-01-01T00:00:00Z")
	if _, err := PublishSessionBatch([]*Session{unchanged, replacedOld}, outputDir, CollisionError); err != nil {
		t.Fatalf("initial batch: %v", err)
	}

	created := batchTestSession("created", "created", "2026-01-01T00:00:00Z")
	replacedNew := batchTestSession("replaced", "new", "2026-01-01T00:00:00Z")
	results, err := PublishSessionBatch([]*Session{unchanged, replacedNew, created}, outputDir, CollisionReplace)
	if err != nil {
		t.Fatalf("second batch: %v", err)
	}
	statuses := map[string]PublicationStatus{}
	for _, result := range results {
		statuses[result.SessionID] = result.Status
	}
	if statuses["created"] != PublicationCreated || statuses["unchanged"] != PublicationUnchanged || statuses["replaced"] != PublicationReplaced {
		t.Fatalf("unexpected statuses: %+v", statuses)
	}
}

func TestPublishSessionBatchRejectsConflictingDestinationsWithinBatch(t *testing.T) {
	outputDir := t.TempDir()
	first := batchTestSession("same-id", "first", "2026-01-01T00:00:00Z")
	second := batchTestSession("same-id", "second", "2026-01-01T00:00:00Z")
	if _, err := PublishSessionBatch([]*Session{first, second}, outputDir, CollisionError); err == nil {
		t.Fatal("expected conflicting batch destination error")
	}
	path := filepath.Join(outputDir, "active", "2026-01", "same-id.minitrace.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("conflicting batch published an archive: %v", err)
	}
}
