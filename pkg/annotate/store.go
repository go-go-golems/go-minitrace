// Package annotate provides a SQLite-backed store for minitrace annotations.
// Annotations are stored in output/annotations.db alongside the session archive.
package annotate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	_ "github.com/mattn/go-sqlite3"
)

// ErrNotFound is returned when a requested annotation does not exist.
var ErrNotFound = errors.New("annotation not found")

// Store manages annotations in a SQLite database at outputDir/annotations.db.
type Store struct {
	db     *sql.DB
	dbPath string
}

// Open resolves outputDir to an absolute path, creates the directory and the
// annotations.db file if they don't exist, opens the SQLite connection with
// WAL mode and a busy timeout, and runs any pending migrations.
func Open(ctx context.Context, outputDir string) (*Store, error) {
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output dir: %w", err)
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	dbPath := filepath.Join(absDir, "annotations.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{db: db, dbPath: dbPath}, nil
}

// Close closes the underlying SQLite connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates the schema tables and indexes if they don't already exist.
func migrate(ctx context.Context, db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS annotations (
		id           TEXT PRIMARY KEY,
		session_id   TEXT NOT NULL,
		annotator    TEXT NOT NULL DEFAULT 'user',
		scope_type   TEXT NOT NULL DEFAULT 'session',
		target_id    TEXT NOT NULL DEFAULT '',
		category     TEXT NOT NULL DEFAULT '',
		title        TEXT NOT NULL DEFAULT '',
		detail       TEXT NOT NULL DEFAULT '',
		tags         TEXT NOT NULL DEFAULT '[]',
		taxonomy_m   TEXT NOT NULL DEFAULT '[]',
		taxonomy_mast TEXT NOT NULL DEFAULT '[]',
		taxonomy_tm  TEXT NOT NULL DEFAULT '[]',
		classification TEXT,
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_annotations_session_id ON annotations(session_id);
	CREATE INDEX IF NOT EXISTS idx_annotations_scope      ON annotations(scope_type, target_id);
	CREATE INDEX IF NOT EXISTS idx_annotations_category    ON annotations(category);
	CREATE INDEX IF NOT EXISTS idx_annotations_annotator  ON annotations(annotator);

	CREATE TABLE IF NOT EXISTS sync_state (
		session_id   TEXT PRIMARY KEY,
		synced_at    TEXT NOT NULL,
		change_count INTEGER NOT NULL DEFAULT 0
	);
	`
	_, err := db.ExecContext(ctx, schema)
	return err
}

// AddAnnotation inserts a new annotation into the store and marks its session
// as needing sync. The annotation's ID must already be set.
func (s *Store) AddAnnotation(ctx context.Context, ann minitrace.Annotation, sessionID string) error {
	tags, err := jsonMarshal(ann.Content.Tags)
	if err != nil {
		return fmt.Errorf("encoding tags: %w", err)
	}
	taxM, err := jsonMarshal(ann.TaxonomyMappings.Minitrace)
	if err != nil {
		return fmt.Errorf("encoding taxonomy_m: %w", err)
	}
	taxMast, err := jsonMarshal(ann.TaxonomyMappings.Mast)
	if err != nil {
		return fmt.Errorf("encoding taxonomy_mast: %w", err)
	}
	taxTm, err := jsonMarshal(ann.TaxonomyMappings.Toolemu)
	if err != nil {
		return fmt.Errorf("encoding taxonomy_tm: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO annotations
		  (id, session_id, annotator, scope_type, target_id,
		   category, title, detail, tags, taxonomy_m, taxonomy_mast, taxonomy_tm,
		   classification, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ann.ID, sessionID, ann.Annotator, ann.Scope.Type, ann.Scope.TargetID,
		ann.Content.Category, ann.Content.Title, ann.Content.Detail,
		tags, taxM, taxMast, taxTm,
		ann.Classification,
		ann.Timestamp, ann.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("inserting annotation: %w", err)
	}
	return s.markUnsynced(ctx, sessionID)
}

// GetAnnotationsForSession returns all annotations for a given session, sorted
// by creation time ascending.
func (s *Store) GetAnnotationsForSession(ctx context.Context, sessionID string) ([]minitrace.Annotation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, annotator, scope_type, target_id,
		       category, title, detail, tags, taxonomy_m, taxonomy_mast, taxonomy_tm,
		       classification, created_at, updated_at
		FROM annotations
		WHERE session_id = ?
		ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying annotations: %w", err)
	}
	defer closeRows(rows)

	anns, err := scanAnnotations(rows)
	if err != nil {
		return nil, fmt.Errorf("scanning annotations: %w", err)
	}
	return anns, nil
}

// ListOptions controls which annotations are returned by List.
type ListOptions struct {
	SessionID string
	ScopeType string
	Category  string
	Annotator string
	Taxonomy  string // matched as a substring in any taxonomy column
	Limit     int
	Offset    int
}

// AnnotationRow is the flattened form of an annotation as returned by List.
type AnnotationRow struct {
	ID             string
	SessionID      string
	Annotator      string
	ScopeType      string
	TargetID       string
	Category       string
	Title          string
	Detail         string
	Tags           []string
	TaxonomyM      []string
	TaxonomyMast   []string
	TaxonomyTm     []string
	Classification *string
	CreatedAt      string
	UpdatedAt      string
}

// List returns annotations matching the given filters. Results are sorted by
// created_at descending.
func (s *Store) List(ctx context.Context, opts ListOptions) ([]AnnotationRow, error) {
	query := `SELECT id, session_id, annotator, scope_type, target_id,
	                 category, title, detail, tags, taxonomy_m, taxonomy_mast, taxonomy_tm,
	                 classification, created_at, updated_at
	          FROM annotations WHERE 1=1`
	args := []any{}

	if opts.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, opts.SessionID)
	}
	if opts.ScopeType != "" {
		query += " AND scope_type = ?"
		args = append(args, opts.ScopeType)
	}
	if opts.Category != "" {
		query += " AND category = ?"
		args = append(args, opts.Category)
	}
	if opts.Annotator != "" {
		query += " AND annotator = ?"
		args = append(args, opts.Annotator)
	}
	if opts.Taxonomy != "" {
		query += " AND (taxonomy_m LIKE ? OR taxonomy_mast LIKE ? OR taxonomy_tm LIKE ?)"
		pat := "%" + opts.Taxonomy + "%"
		args = append(args, pat, pat, pat)
	}

	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	} else {
		query += " LIMIT 50"
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing annotations: %w", err)
	}
	defer closeRows(rows)

	var results []AnnotationRow
	for rows.Next() {
		var r AnnotationRow
		var tags, taxM, taxMast, taxTm string
		err := rows.Scan(
			&r.ID, &r.SessionID, &r.Annotator, &r.ScopeType, &r.TargetID,
			&r.Category, &r.Title, &r.Detail,
			&tags, &taxM, &taxMast, &taxTm,
			&r.Classification, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning annotation row: %w", err)
		}
		r.Tags = parseJSONArray(tags)
		r.TaxonomyM = parseJSONArray(taxM)
		r.TaxonomyMast = parseJSONArray(taxMast)
		r.TaxonomyTm = parseJSONArray(taxTm)
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return results, nil
}

// AnnotationPatch represents the fields that may be updated on an annotation.
// A nil or zero-value field means "do not update".
type AnnotationPatch struct {
	Title          *string
	Detail         *string
	Category       *string
	Tags           *[]string
	TaxonomyM      *[]string
	TaxonomyMast   *[]string
	TaxonomyTm     *[]string
	Classification *string
}

// Update applies patch to the annotation identified by id. It returns
// ErrNotFound if no annotation with that id exists.
func (s *Store) Update(ctx context.Context, id string, patch AnnotationPatch) error {
	// First verify the annotation exists and record its session_id.
	var sessionID string
	err := s.db.QueryRowContext(ctx, "SELECT session_id FROM annotations WHERE id = ?", id).Scan(&sessionID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("looking up annotation: %w", err)
	}

	// Build the SET clause dynamically from non-nil patch fields.
	set, args := buildPatchSET(patch)
	if set == "" {
		return nil // nothing to update
	}
	set += ", updated_at = ?"
	args = append(args, minitrace.FormatTimestamp(minitrace.NowUTC()))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE annotations SET %s WHERE id = ?", set)
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("updating annotation: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return s.markUnsynced(ctx, sessionID)
}

// Delete removes the annotation identified by id. It returns ErrNotFound if
// no such annotation exists.
func (s *Store) Delete(ctx context.Context, id string) error {
	// Look up session_id before deleting so we can mark the session unsynced.
	var sessionID string
	err := s.db.QueryRowContext(ctx, "SELECT session_id FROM annotations WHERE id = ?", id).Scan(&sessionID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("looking up annotation: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM annotations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting annotation: %w", err)
	}
	return s.markUnsynced(ctx, sessionID)
}

// SyncState records whether a session's annotations have been synced to its
// .minitrace.json file.
type SyncState struct {
	SessionID   string
	SyncedAt    string
	ChangeCount int
}

// GetUnsyncedSessions returns all sessions that have annotations in the store
// which have changed since the last successful sync.
func (s *Store) GetUnsyncedSessions(ctx context.Context) ([]SyncState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ss.session_id, ss.synced_at, ss.change_count
		FROM sync_state ss
		JOIN annotations a ON a.session_id = ss.session_id
		GROUP BY ss.session_id, ss.synced_at, ss.change_count
		HAVING COUNT(*) > 0
		ORDER BY MAX(a.created_at) ASC`)
	if err != nil {
		return nil, fmt.Errorf("querying unsynced sessions: %w", err)
	}
	defer closeRows(rows)

	var states []SyncState
	for rows.Next() {
		var ss SyncState
		if err := rows.Scan(&ss.SessionID, &ss.SyncedAt, &ss.ChangeCount); err != nil {
			return nil, fmt.Errorf("scanning sync_state: %w", err)
		}
		states = append(states, ss)
	}
	return states, rows.Err()
}

// markUnsynced records that sessionID has unsynced changes.
// If the session is not yet in sync_state, it inserts with change_count=1.
// If it is already present, it increments change_count.
func (s *Store) markUnsynced(ctx context.Context, sessionID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE sync_state SET change_count = change_count + 1 WHERE session_id = ?`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("incrementing change_count: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO sync_state (session_id, synced_at, change_count)
			VALUES (?, '', 1)`,
			sessionID,
		)
		if err != nil {
			return fmt.Errorf("inserting sync_state: %w", err)
		}
	}
	return nil
}

// markSynced records that sessionID has been successfully synced to its JSON file.
func (s *Store) markSynced(ctx context.Context, sessionID string, changeCount int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_state (session_id, synced_at, change_count)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET synced_at = excluded.synced_at, change_count = excluded.change_count`,
		sessionID, minitrace.FormatTimestamp(minitrace.NowUTC()), changeCount,
	)
	return err
}

// --- helpers ---

func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// closeRows silences the errcheck linter for rows.Close().
func closeRows(rows *sql.Rows) { _ = rows.Close() }

// parseJSONArray(s string) returns a non-nil empty []string for both null and [].
func parseJSONArray(s string) []string {
	if s == "" || s == "null" || s == "[]" {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

// scanAnnotations reads all rows from a *sql.Rows and returns them as a slice
// of minitrace.Annotation.
func scanAnnotations(rows *sql.Rows) ([]minitrace.Annotation, error) {
	var anns []minitrace.Annotation
	for rows.Next() {
		var (
			id, sessionID, annotator, scopeType, targetID string
			category, title, detail                       string
			tags, taxM, taxMast, taxTm                    string
			classification                                *string
			createdAt, updatedAt                          string
		)
		err := rows.Scan(
			&id, &sessionID, &annotator, &scopeType, &targetID,
			&category, &title, &detail,
			&tags, &taxM, &taxMast, &taxTm,
			&classification, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		anns = append(anns, minitrace.Annotation{
			ID:        id,
			Timestamp: createdAt,
			Annotator: annotator,
			Scope:     minitrace.AnnotationScope{Type: scopeType, TargetID: targetID},
			Content: minitrace.AnnotationContent{
				Category: category,
				Tags:     parseJSONArray(tags),
				Title:    title,
				Detail:   detail,
			},
			TaxonomyMappings: minitrace.TaxonomyMappings{
				Minitrace: parseJSONArray(taxM),
				Mast:      parseJSONArray(taxMast),
				Toolemu:   parseJSONArray(taxTm),
			},
			Classification: classification,
		})
	}
	return anns, rows.Err()
}

// buildPatchSET constructs a "col = ?, col = ?" clause from a non-nil AnnotationPatch.
// It returns the SET string and the args to bind to it.
func buildPatchSET(patch AnnotationPatch) (string, []any) {
	var set string
	var args []any
	appendSet := func(col string, val any) {
		if set != "" {
			set += ", "
		}
		set += col + " = ?"
		args = append(args, val)
	}

	if patch.Title != nil {
		appendSet("title", *patch.Title)
	}
	if patch.Detail != nil {
		appendSet("detail", *patch.Detail)
	}
	if patch.Category != nil {
		appendSet("category", *patch.Category)
	}
	if patch.Tags != nil {
		s, _ := jsonMarshal(*patch.Tags)
		appendSet("tags", s)
	}
	if patch.TaxonomyM != nil {
		s, _ := jsonMarshal(*patch.TaxonomyM)
		appendSet("taxonomy_m", s)
	}
	if patch.TaxonomyMast != nil {
		s, _ := jsonMarshal(*patch.TaxonomyMast)
		appendSet("taxonomy_mast", s)
	}
	if patch.TaxonomyTm != nil {
		s, _ := jsonMarshal(*patch.TaxonomyTm)
		appendSet("taxonomy_tm", s)
	}
	if patch.Classification != nil {
		appendSet("classification", *patch.Classification)
	}
	return set, args
}
