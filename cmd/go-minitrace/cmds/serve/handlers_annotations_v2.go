package serve

import (
	"net/http"
	"strconv"
	"time"

	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/google/uuid"
)

var protoAnnotationScopeTypeToString = map[apiv1.AnnotationScopeType]string{
	apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_SESSION:   "session",
	apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TURN:      "turn",
	apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TOOL_CALL: "tool_call",
}

var stringToProtoAnnotationScopeType = map[string]apiv1.AnnotationScopeType{
	"session":   apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_SESSION,
	"turn":      apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TURN,
	"tool_call": apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_TOOL_CALL,
}

var protoAnnotationCategoryToString = map[apiv1.AnnotationCategory]string{
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_OBSERVATION:       "observation",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_AI_FAILURE:        "ai-failure",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_USER_ERROR:        "user-error",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_ENVIRONMENT_ISSUE: "environment-issue",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_SUCCESS:           "success",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_QUESTION:          "question",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_TO_DISCUSS:        "to-discuss",
	apiv1.AnnotationCategory_ANNOTATION_CATEGORY_TO_IMPROVE:        "to-improve",
}

var stringToProtoAnnotationCategory = map[string]apiv1.AnnotationCategory{
	"observation":       apiv1.AnnotationCategory_ANNOTATION_CATEGORY_OBSERVATION,
	"ai-failure":        apiv1.AnnotationCategory_ANNOTATION_CATEGORY_AI_FAILURE,
	"user-error":        apiv1.AnnotationCategory_ANNOTATION_CATEGORY_USER_ERROR,
	"environment-issue": apiv1.AnnotationCategory_ANNOTATION_CATEGORY_ENVIRONMENT_ISSUE,
	"success":           apiv1.AnnotationCategory_ANNOTATION_CATEGORY_SUCCESS,
	"question":          apiv1.AnnotationCategory_ANNOTATION_CATEGORY_QUESTION,
	"to-discuss":        apiv1.AnnotationCategory_ANNOTATION_CATEGORY_TO_DISCUSS,
	"to-improve":        apiv1.AnnotationCategory_ANNOTATION_CATEGORY_TO_IMPROVE,
}

func (s *Server) handleGetSessionAnnotationsV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := r.PathValue("id")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	anns, err := s.annoStore.GetAnnotationsForSession(ctx, sessionID)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("fetching annotations v2")
		writeError(w, http.StatusInternalServerError, "fetching annotations")
		return
	}

	payload := &apiv1.GetSessionAnnotationsResponse{
		Meta:        &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		SessionId:   sessionID,
		Count:       clampIntToUint32(len(anns)),
		Annotations: protoAnnotations(anns),
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleCreateAnnotationV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := r.PathValue("id")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	var req apiv1.CreateAnnotationRequest
	if err := decodeProtoJSONRequest(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.GetCategory() == apiv1.AnnotationCategory_ANNOTATION_CATEGORY_UNSPECIFIED {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}
	if req.GetTitle() == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	category, ok := protoAnnotationCategoryToString[req.GetCategory()]
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}

	scopeType := "session"
	if req.GetScopeType() != apiv1.AnnotationScopeType_ANNOTATION_SCOPE_TYPE_UNSPECIFIED {
		var ok bool
		scopeType, ok = protoAnnotationScopeTypeToString[req.GetScopeType()]
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid scope type")
			return
		}
	}

	targetID := req.GetTargetId()
	if targetID == "" {
		targetID = sessionID
	}
	annotator := req.GetAnnotator()
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
			Category: category,
			Title:    req.GetTitle(),
			Detail:   req.GetDetail(),
			Tags:     append([]string(nil), req.GetTags()...),
		},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: append([]string(nil), req.GetTaxonomyMinitrace()...),
			Mast:      append([]string(nil), req.GetTaxonomyMast()...),
			Toolemu:   append([]string(nil), req.GetTaxonomyToolemu()...),
		},
	}
	if req.Classification != nil {
		classification := req.GetClassification()
		ann.Classification = &classification
	}

	if err := s.annoStore.AddAnnotation(ctx, ann, sessionID); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("creating annotation v2")
		writeError(w, http.StatusInternalServerError, "creating annotation")
		return
	}

	writeProtoJSON(w, http.StatusCreated, protoAnnotation(ann))
}

func (s *Server) handleListAnnotationsV2(w http.ResponseWriter, r *http.Request) {
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
		log.Error().Err(err).Msg("listing annotations v2")
		writeError(w, http.StatusInternalServerError, "listing annotations")
		return
	}

	payload := &apiv1.ListAnnotationsResponse{
		Meta:        &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Annotations: protoAnnotationRows(rows),
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleUpdateAnnotationV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	annID := r.PathValue("annId")

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	var req apiv1.UpdateAnnotationRequest
	if err := decodeProtoJSONRequest(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	patch := annotate.AnnotationPatch{}
	if req.Title != nil {
		title := req.GetTitle()
		patch.Title = &title
	}
	if req.Detail != nil {
		detail := req.GetDetail()
		patch.Detail = &detail
	}
	if req.Category != nil {
		category, ok := protoAnnotationCategoryToString[req.GetCategory()]
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid category")
			return
		}
		patch.Category = &category
	}
	if req.GetTags() != nil {
		tags := append([]string(nil), req.GetTags().GetValues()...)
		patch.Tags = &tags
	}
	if req.GetTaxonomyMinitrace() != nil {
		taxonomy := append([]string(nil), req.GetTaxonomyMinitrace().GetValues()...)
		patch.TaxonomyM = &taxonomy
	}
	if req.GetTaxonomyMast() != nil {
		taxonomy := append([]string(nil), req.GetTaxonomyMast().GetValues()...)
		patch.TaxonomyMast = &taxonomy
	}
	if req.GetTaxonomyToolemu() != nil {
		taxonomy := append([]string(nil), req.GetTaxonomyToolemu().GetValues()...)
		patch.TaxonomyTm = &taxonomy
	}
	if req.Classification != nil {
		classification := req.GetClassification()
		patch.Classification = &classification
	}

	err := s.annoStore.Update(ctx, annID, patch)
	if err == annotate.ErrNotFound {
		writeError(w, http.StatusNotFound, "annotation not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("id", annID).Msg("updating annotation v2")
		writeError(w, http.StatusInternalServerError, "updating annotation")
		return
	}

	payload := &apiv1.UpdateAnnotationResponse{
		Meta:   &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Id:     annID,
		Status: "updated",
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleDeleteAnnotationV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	annID := r.PathValue("annId")

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
		log.Error().Err(err).Str("id", annID).Msg("deleting annotation v2")
		writeError(w, http.StatusInternalServerError, "deleting annotation")
		return
	}

	payload := &apiv1.DeleteAnnotationResponse{
		Meta:   &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Id:     annID,
		Status: "deleted",
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleSyncAnnotationsV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.annoStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	var req apiv1.SyncAnnotationsRequest
	if err := decodeProtoJSONRequest(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	report, err := s.annoStore.SyncAll(ctx, s.annoIndex, annotate.SyncOptions{
		DryRun:    req.GetDryRun(),
		SessionID: req.GetSessionId(),
	})
	if err != nil {
		log.Error().Err(err).Msg("sync annotations v2")
		writeError(w, http.StatusInternalServerError, "sync failed")
		return
	}

	status := http.StatusOK
	if len(report.Errors) > 0 {
		status = http.StatusPartialContent
	}
	writeProtoJSON(w, status, protoSyncAnnotationsResponse(report))
}

func protoAnnotations(annotations []minitrace.Annotation) []*apiv1.Annotation {
	ret := make([]*apiv1.Annotation, 0, len(annotations))
	for _, annotation := range annotations {
		ret = append(ret, protoAnnotation(annotation))
	}
	return ret
}

func protoAnnotation(annotation minitrace.Annotation) *apiv1.Annotation {
	return &apiv1.Annotation{
		Id:        annotation.ID,
		Timestamp: annotation.Timestamp,
		Annotator: annotation.Annotator,
		Scope: &apiv1.AnnotationScope{
			Type:     stringToProtoAnnotationScopeType[annotation.Scope.Type],
			TargetId: annotation.Scope.TargetID,
		},
		Content: &apiv1.AnnotationContent{
			Category: stringToProtoAnnotationCategory[annotation.Content.Category],
			Tags:     append([]string(nil), annotation.Content.Tags...),
			Title:    annotation.Content.Title,
			Detail:   annotation.Content.Detail,
		},
		TaxonomyMappings: &apiv1.TaxonomyMappings{
			Minitrace: append([]string(nil), annotation.TaxonomyMappings.Minitrace...),
			Mast:      append([]string(nil), annotation.TaxonomyMappings.Mast...),
			Toolemu:   append([]string(nil), annotation.TaxonomyMappings.Toolemu...),
		},
		Classification: annotation.Classification,
	}
}

func protoAnnotationRows(rows []annotate.AnnotationRow) []*apiv1.AnnotationListRow {
	ret := make([]*apiv1.AnnotationListRow, 0, len(rows))
	for _, row := range rows {
		ret = append(ret, &apiv1.AnnotationListRow{
			Id:                row.ID,
			SessionId:         row.SessionID,
			Annotator:         row.Annotator,
			ScopeType:         stringToProtoAnnotationScopeType[row.ScopeType],
			TargetId:          row.TargetID,
			Category:          stringToProtoAnnotationCategory[row.Category],
			Title:             row.Title,
			Detail:            row.Detail,
			Tags:              append([]string(nil), row.Tags...),
			TaxonomyMinitrace: append([]string(nil), row.TaxonomyM...),
			TaxonomyMast:      append([]string(nil), row.TaxonomyMast...),
			TaxonomyToolemu:   append([]string(nil), row.TaxonomyTm...),
			Classification:    row.Classification,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		})
	}
	return ret
}

func protoSyncAnnotationsResponse(report *annotate.SyncReport) *apiv1.SyncAnnotationsResponse {
	errors := make([]*apiv1.SyncError, 0, len(report.Errors))
	for _, syncErr := range report.Errors {
		errors = append(errors, &apiv1.SyncError{
			SessionId: syncErr.SessionID,
			Error:     syncErr.Error,
		})
	}
	return &apiv1.SyncAnnotationsResponse{
		Meta:    &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Synced:  append([]string(nil), report.Synced...),
		Skipped: append([]string(nil), report.Skipped...),
		Errors:  errors,
	}
}
