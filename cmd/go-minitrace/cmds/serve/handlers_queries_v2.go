package serve

import (
	"net/http"
	"os"
	"strings"

	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
)

func (s *Server) handleGetPresetsV2(w http.ResponseWriter, _ *http.Request) {
	presets := make([]SavedQuery, 0)

	for _, preset := range queryengine.ListPresetEntries() {
		presetID := strings.TrimSuffix(preset.Path, ".sql")
		sqlText, err := queryengine.ResolvePresetSQL(presetID, s.tableName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		folder := "core"
		if preset.Folder != "" {
			folder = "core/" + preset.Folder
		}
		presets = append(presets, SavedQuery{
			Name:        preset.Name,
			Folder:      folder,
			Path:        "core/" + preset.Path,
			Description: extractSQLComment(sqlText),
			SQL:         sqlText,
			Readonly:    true,
		})
	}

	if len(s.presetDirs) > 0 {
		externalPresets, err := loadSQLDirs(s.presetDirs, true)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		presets = append(presets, externalPresets...)
	}

	payload := &apiv1.ListPresetsResponse{
		Meta:    &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Presets: protoSavedQueries(presets),
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleGetQueriesV2(w http.ResponseWriter, _ *http.Request) {
	queries, err := loadSQLDirs(s.queryDirs, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := &apiv1.ListQueriesResponse{
		Meta:    &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Queries: protoSavedQueries(queries),
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleSaveQueryV2(w http.ResponseWriter, r *http.Request) {
	var req apiv1.SaveQueryRequest
	if err := decodeProtoJSONRequest(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.GetSql()) == "" {
		writeError(w, http.StatusBadRequest, "sql is required")
		return
	}

	baseDir, err := firstQueryDir(s.queryDirs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	queryPath, err := buildQueryCreatePath(baseDir, req.GetFolder(), req.GetName())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := queryPath.MkdirAllParent(); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Wrap(err, "creating query directory").Error())
		return
	}

	content := buildSQLFileContent(req.GetDescription(), req.GetSql())
	if err := queryPath.WriteFile([]byte(content)); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Wrap(err, "writing query file").Error())
		return
	}

	query, err := loadSingleSQLFile(baseDir, queryPath.relativePath, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeProtoJSON(w, http.StatusCreated, protoSavedQuery(query))
}

func (s *Server) handleUpdateQueryV2(w http.ResponseWriter, r *http.Request) {
	relativePath := r.PathValue("path")

	var req apiv1.UpdateQueryRequest
	if err := decodeProtoJSONRequest(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.GetSql()) == "" {
		writeError(w, http.StatusBadRequest, "sql is required")
		return
	}

	queryPath, err := findExistingQueryPath(s.queryDirs, relativePath)
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			message = "query not found"
		}
		writeError(w, status, message)
		return
	}

	content := buildSQLFileContent(req.GetDescription(), req.GetSql())
	if err := queryPath.WriteFile([]byte(content)); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Wrap(err, "writing query file").Error())
		return
	}

	query, err := loadSingleSQLFile(queryPath.rootDir, queryPath.relativePath, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeProtoJSON(w, http.StatusOK, protoSavedQuery(query))
}

func (s *Server) handleDeleteQueryV2(w http.ResponseWriter, r *http.Request) {
	relativePath := r.PathValue("path")

	queryPath, err := findExistingQueryPath(s.queryDirs, relativePath)
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			message = "query not found"
		}
		writeError(w, status, message)
		return
	}
	if err := queryPath.Remove(); err != nil {
		status := http.StatusInternalServerError
		message := errors.Wrap(err, "deleting query file").Error()
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			message = "query not found"
		}
		writeError(w, status, message)
		return
	}

	payload := &apiv1.DeleteQueryResponse{
		Meta:   &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Path:   queryPath.relativePath,
		Status: "deleted",
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func protoSavedQueries(queries []SavedQuery) []*apiv1.SavedQuery {
	ret := make([]*apiv1.SavedQuery, 0, len(queries))
	for _, query := range queries {
		ret = append(ret, protoSavedQuery(query))
	}
	return ret
}

func protoSavedQuery(query SavedQuery) *apiv1.SavedQuery {
	return &apiv1.SavedQuery{
		Name:        query.Name,
		Folder:      query.Folder,
		Path:        query.Path,
		Description: query.Description,
		Sql:         query.SQL,
		Readonly:    query.Readonly,
	}
}
