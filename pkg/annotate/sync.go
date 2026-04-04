package annotate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

// SyncOptions controls the behavior of SyncAll.
type SyncOptions struct {
	// DryRun causes SyncSession to write to a temp file and return without
	// replacing the original. Useful for previewing changes.
	DryRun bool
	// SessionID restricts sync to a single session. If empty, all unsynced
	// sessions are processed.
	SessionID string
}

// SyncReport summarizes the outcome of a SyncAll call.
type SyncReport struct {
	Synced  []string    // session IDs that were successfully synced
	Skipped []string    // session IDs that had no file path in the index
	Errors  []SyncError // session IDs that failed
}

// SyncError records a sync failure for a specific session.
type SyncError struct {
	SessionID string
	Error     string
}

// SyncSession reads the .minitrace.json file at filePath, replaces its
// Annotations field with the provided slice, and writes the file back
// atomically (write-to-temp then rename on POSIX same-filesystem).
// If opts.DryRun is true, the file is not modified.
func SyncSession(
	ctx context.Context,
	filePath string,
	annotations []minitrace.Annotation,
	opts SyncOptions,
) error {
	// Read the existing session JSON.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filePath, err)
	}

	// Unmarshal into a generic map so we can replace the annotations field
	// without re-serializing the entire session.
	var session map[string]any
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("parsing %s: %w", filePath, err)
	}

	// Replace the annotations field. Use an empty slice, not nil,
	// so the JSON output is "annotations": [] rather than "annotations": null.
	if annotations == nil {
		session["annotations"] = []any{}
	} else {
		// Serialize each annotation individually so we get clean, annotated JSON.
		anns := make([]any, len(annotations))
		for i, a := range annotations {
			b, err := json.Marshal(a)
			if err != nil {
				return fmt.Errorf("marshaling annotation %d: %w", i, err)
			}
			var v any
			if err := json.Unmarshal(b, &v); err != nil {
				return fmt.Errorf("re-unmarshaling annotation %d: %w", i, err)
			}
			anns[i] = v
		}
		session["annotations"] = anns
	}

	// Re-marshal with 2-space indent.
	out, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("re-marshaling session: %w", err)
	}
	// Ensure file ends with newline.
	out = append(out, '\n')

	if opts.DryRun {
		// Dry-run: print to stdout instead of writing.
		fmt.Printf("[dry-run] %s would be updated with %d annotation(s)\n",
			filePath, len(annotations))
		return nil
	}

	// Atomic write: write to {file}.tmp then rename.
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0o644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath) // best-effort cleanup
		return fmt.Errorf("renaming temp file to %s: %w", filePath, err)
	}
	return nil
}

// SyncAll iterates over all sessions in sessionIndex (mapping session IDs to
// file paths) that have unsynced changes in the store, and writes their
// annotations back to the .minitrace.json files. On success, it marks each
// session as synced.
func (s *Store) SyncAll(
	ctx context.Context,
	sessionIndex map[string]string,
	opts SyncOptions,
) (*SyncReport, error) {
	var unsynced []SyncState
	var err error

	if opts.SessionID != "" {
		// Single-session sync requested.
		state := SyncState{SessionID: opts.SessionID}
		unsynced = []SyncState{state}
	} else {
		unsynced, err = s.GetUnsyncedSessions(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting unsynced sessions: %w", err)
		}
	}

	report := &SyncReport{}
	for _, state := range unsynced {
		sessionID := state.SessionID

		filePath, ok := sessionIndex[sessionID]
		if !ok {
			report.Skipped = append(report.Skipped, sessionID)
			continue
		}

		// Resolve to absolute path.
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			report.Errors = append(report.Errors, SyncError{
				SessionID: sessionID,
				Error:     fmt.Sprintf("resolving path: %v", err),
			})
			continue
		}

		// Fetch annotations for this session.
		annotations, err := s.GetAnnotationsForSession(ctx, sessionID)
		if err != nil {
			report.Errors = append(report.Errors, SyncError{
				SessionID: sessionID,
				Error:     fmt.Sprintf("fetching annotations: %v", err),
			})
			continue
		}

		if err := SyncSession(ctx, absPath, annotations, opts); err != nil {
			report.Errors = append(report.Errors, SyncError{
				SessionID: sessionID,
				Error:     err.Error(),
			})
			continue
		}

		// Mark synced after successful write.
		if !opts.DryRun {
			if err := s.markSynced(ctx, sessionID, state.ChangeCount); err != nil {
				// Non-fatal: log but don't fail the whole sync.
				report.Errors = append(report.Errors, SyncError{
					SessionID: sessionID,
					Error:     fmt.Sprintf("sync succeeded but markSynced failed: %v", err),
				})
			}
		}
		report.Synced = append(report.Synced, sessionID)
	}

	return report, nil
}
