package serve

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// CreateAnnotationRequest is the body of POST /api/sessions/{id}/annotations.
type CreateAnnotationRequest struct {
	Annotator         string   `json:"annotator"`
	ScopeType         string   `json:"scope_type"`
	TargetID          string   `json:"target_id"`
	Category          string   `json:"category"`
	Title             string   `json:"title"`
	Detail            string   `json:"detail"`
	Tags              []string `json:"tags"`
	TaxonomyMinitrace []string `json:"taxonomy_minitrace"`
	TaxonomyMast      []string `json:"taxonomy_mast"`
	TaxonomyToolemu   []string `json:"taxonomy_toolemu"`
	Classification    string   `json:"classification"`
}

// handleGetSessionAnnotations returns all annotations for a session.
func (s *Server) handleGetSessionAnnotations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := extractPathParam(r, "id")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	anns, err := s.annoStore.GetAnnotationsForSession(ctx, sessionID)
	if err != nil {
		log.Err(err).Str("session_id", sessionID).Msg("fetching annotations")
		writeError(w, http.StatusInternalServerError, "fetching annotations")
		return
	}

	// Ensure nil annotations become empty slices for JSON array output.
	if anns == nil {
		anns = []minitrace.Annotation{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":  sessionID,
		"count":       len(anns),
		"annotations": anns,
	})
}

// handleCreateAnnotation creates a new annotation on a session.
func (s *Server) handleCreateAnnotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := extractPathParam(r, "id")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	var req CreateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate required fields.
	if req.Category == "" {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	// Validate category.
	validCategories := map[string]bool{
		"observation":       true,
		"ai-failure":        true,
		"user-error":        true,
		"environment-issue": true,
		"success":           true,
		"question":          true,
		"to-discuss":        true,
		"to-improve":        true,
	}
	if !validCategories[req.Category] {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}

	scopeType := req.ScopeType
	if scopeType == "" {
		scopeType = "session"
	}
	targetID := req.TargetID
	if targetID == "" {
		targetID = sessionID
	}
	annotator := req.Annotator
	if annotator == "" {
		annotator = "user"
	}

	ann := minitrace.Annotation{
		ID:        uuid.New().String(),
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: annotator,
		Scope: minitrace.AnnotationScope{
			Type:     scopeType,
			TargetID: targetID,
		},
		Content: minitrace.AnnotationContent{
			Category: req.Category,
			Title:    req.Title,
			Detail:   req.Detail,
			Tags:     req.Tags,
		},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: req.TaxonomyMinitrace,
			Mast:      req.TaxonomyMast,
			Toolemu:   req.TaxonomyToolemu,
		},
	}
	if req.Classification != "" {
		ann.Classification = &req.Classification
	}

	if err := s.annoStore.AddAnnotation(ctx, ann, sessionID); err != nil {
		log.Err(err).Str("session_id", sessionID).Msg("creating annotation")
		writeError(w, http.StatusInternalServerError, "creating annotation")
		return
	}

	// Annotations are live in DuckDB via sqlite_scanner — no refresh needed.

	writeJSON(w, http.StatusCreated, ann)
}

// handleListAnnotations returns all annotations with optional filters.
func (s *Server) handleListAnnotations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	q := r.URL.Query()
	opts := annotate.ListOptions{
		SessionID: q.Get("session"),
		ScopeType: q.Get("scope"),
		Category:  q.Get("category"),
		Annotator: q.Get("annotator"),
		Taxonomy:  q.Get("taxonomy"),
	}

	// Parse limit.
	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			opts.Limit = limit
		}
	}
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	rows, err := s.annoStore.List(ctx, opts)
	if err != nil {
		log.Err(err).Msg("listing annotations")
		writeError(w, http.StatusInternalServerError, "listing annotations")
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// handleUpdateAnnotation patches an existing annotation.
func (s *Server) handleUpdateAnnotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	annID := extractPathParam(r, "annId")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	patch := annotate.AnnotationPatch{}
	if v, ok := req["title"]; ok {
		if s, ok := v.(string); ok {
			patch.Title = &s
		}
	}
	if v, ok := req["detail"]; ok {
		if s, ok := v.(string); ok {
			patch.Detail = &s
		}
	}
	if v, ok := req["category"]; ok {
		if s, ok := v.(string); ok {
			patch.Category = &s
		}
	}
	if v, ok := req["classification"]; ok {
		if s, ok := v.(string); ok {
			patch.Classification = &s
		}
	}
	if v, ok := req["tags"]; ok {
		if arr, ok := v.([]any); ok {
			tags := make([]string, 0, len(arr))
			for _, e := range arr {
				if s, ok := e.(string); ok {
					tags = append(tags, s)
				}
			}
			patch.Tags = &tags
		}
	}
	if v, ok := req["taxonomy_minitrace"]; ok {
		if arr, ok := v.([]any); ok {
			tax := make([]string, 0, len(arr))
			for _, e := range arr {
				if s, ok := e.(string); ok {
					tax = append(tax, s)
				}
			}
			patch.TaxonomyM = &tax
		}
	}

	err := s.annoStore.Update(ctx, annID, patch)
	if err == annotate.ErrNotFound {
		writeError(w, http.StatusNotFound, "annotation not found")
		return
	}
	if err != nil {
		log.Err(err).Str("id", annID).Msg("updating annotation")
		writeError(w, http.StatusInternalServerError, "updating annotation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": annID, "status": "updated"})
}

// handleDeleteAnnotation deletes an annotation.
func (s *Server) handleDeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	annID := extractPathParam(r, "annId")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	err := s.annoStore.Delete(ctx, annID)
	if err == annotate.ErrNotFound {
		writeError(w, http.StatusNotFound, "annotation not found")
		return
	}
	if err != nil {
		log.Err(err).Str("id", annID).Msg("deleting annotation")
		writeError(w, http.StatusInternalServerError, "deleting annotation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSyncAnnotations syncs annotations from SQLite back to .minitrace.json files.
func (s *Server) handleSyncAnnotations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	syncReq := struct {
		SessionID string `json:"session_id"`
		DryRun    bool   `json:"dry_run"`
	}{}
	// Best-effort decode; zero values mean sync all sessions.
	_ = json.NewDecoder(r.Body).Decode(&syncReq)

	opts := annotate.SyncOptions{
		DryRun:    syncReq.DryRun,
		SessionID: syncReq.SessionID,
	}

	report, err := s.annoStore.SyncAll(ctx, s.annoIndex, opts)
	if err != nil {
		log.Err(err).Msg("sync annotations")
		writeError(w, http.StatusInternalServerError, "sync failed")
		return
	}

	status := http.StatusOK
	if len(report.Errors) > 0 {
		status = http.StatusPartialContent
	}

	writeJSON(w, status, report)
}

// extractPathParam extracts a path segment from a mux pattern like "/api/sessions/{id}/annotations".
// For patterns with named params, we rely on the request's URL path.
func extractPathParam(r *http.Request, name string) string {
	// Simple approach: trim the pattern prefix.
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(path, "/")
	switch name {
	case "id":
		if len(parts) >= 2 {
			return parts[1]
		}
	case "annId":
		// /api/annotations/{annId} or /api/annotations/{annId}/...
		if len(parts) >= 2 {
			return parts[2]
		}
	}
	return ""
}
