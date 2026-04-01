package serve

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
)

type SavedQuery struct {
	Name        string `json:"name"`
	Folder      string `json:"folder"`
	Path        string `json:"path"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
	Readonly    bool   `json:"readonly"`
}

type SaveQueryRequest struct {
	Name        string `json:"name"`
	Folder      string `json:"folder"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
}

type UpdateQueryRequest struct {
	Description string `json:"description"`
	SQL         string `json:"sql"`
}

func (s *Server) handleGetPresets(w http.ResponseWriter, _ *http.Request) {
	presets := make([]SavedQuery, 0)

	for _, name := range queryengine.ListPresets() {
		sqlText, err := queryengine.ResolvePresetSQL(name, s.tableName)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, QueryResponse{
				Columns:    []string{},
				Rows:       []map[string]any{},
				DurationMS: 0,
				RowCount:   0,
				Error:      &QueryError{Message: err.Error()},
			})
			return
		}
		presets = append(presets, SavedQuery{
			Name:        name,
			Folder:      "core",
			Path:        "core/" + name + ".sql",
			Description: extractSQLComment(sqlText),
			SQL:         sqlText,
			Readonly:    true,
		})
	}

	if len(s.presetDirs) > 0 {
		externalPresets, err := loadSQLDirs(s.presetDirs, true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, QueryResponse{
				Columns:    []string{},
				Rows:       []map[string]any{},
				DurationMS: 0,
				RowCount:   0,
				Error:      &QueryError{Message: err.Error()},
			})
			return
		}
		presets = append(presets, externalPresets...)
	}

	writeJSON(w, http.StatusOK, presets)
}

func (s *Server) handleGetQueries(w http.ResponseWriter, _ *http.Request) {
	queries, err := loadSQLDirs(s.queryDirs, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, queries)
}

func (s *Server) handleSaveQuery(w http.ResponseWriter, r *http.Request) {
	req := SaveQueryRequest{}
	if err := decodeRequest(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: "sql is required"},
		})
		return
	}

	baseDir, err := firstQueryDir(s.queryDirs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}

	relativePath, absolutePath, err := buildQueryCreatePath(baseDir, req.Folder, req.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "creating query directory").Error()},
		})
		return
	}

	content := buildSQLFileContent(req.Description, req.SQL)
	if err := os.WriteFile(absolutePath, []byte(content), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "writing query file").Error()},
		})
		return
	}

	query, err := loadSingleSQLFile(baseDir, relativePath, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusCreated, query)
}

func (s *Server) handleUpdateQuery(w http.ResponseWriter, r *http.Request) {
	relativePath := r.PathValue("path")
	req := UpdateQueryRequest{}
	if err := decodeRequest(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: "sql is required"},
		})
		return
	}

	baseDir, absolutePath, cleanRelativePath, err := findExistingQueryPath(s.queryDirs, relativePath)
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			message = "query not found"
		}
		writeJSON(w, status, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: message},
		})
		return
	}

	content := buildSQLFileContent(req.Description, req.SQL)
	if err := os.WriteFile(absolutePath, []byte(content), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "writing query file").Error()},
		})
		return
	}

	query, err := loadSingleSQLFile(baseDir, cleanRelativePath, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, query)
}

func (s *Server) handleDeleteQuery(w http.ResponseWriter, r *http.Request) {
	_, absolutePath, _, err := findExistingQueryPath(s.queryDirs, r.PathValue("path"))
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			message = "query not found"
		}
		writeJSON(w, status, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: message},
		})
		return
	}
	if err := os.Remove(absolutePath); err != nil {
		status := http.StatusInternalServerError
		message := errors.Wrap(err, "deleting query file").Error()
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			message = "query not found"
		}
		writeJSON(w, status, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: 0,
			RowCount:   0,
			Error:      &QueryError{Message: message},
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func firstQueryDir(queryDirs []string) (string, error) {
	if len(queryDirs) == 0 {
		return "", errors.New("query-dir is not configured")
	}
	return queryDirs[0], nil
}

func findExistingQueryPath(queryDirs []string, relativePath string) (string, string, string, error) {
	if len(queryDirs) == 0 {
		return "", "", "", errors.New("query-dir is not configured")
	}

	cleanPath, err := cleanRelativePath(relativePath)
	if err != nil {
		return "", "", "", err
	}

	for _, queryDir := range queryDirs {
		absolutePath, cleanRelativePath, err := safeQueryPath(queryDir, cleanPath)
		if err != nil {
			return "", "", "", err
		}
		if _, err := os.Stat(absolutePath); err == nil {
			return queryDir, absolutePath, cleanRelativePath, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", "", errors.Wrap(err, "stating query file")
		}
	}

	return "", "", cleanPath, os.ErrNotExist
}

func loadSQLDirs(dirs []string, readonly bool) ([]SavedQuery, error) {
	queries := make([]SavedQuery, 0)
	seen := make(map[string]struct{})

	for _, dir := range dirs {
		dirQueries, err := loadSQLDir(dir, readonly)
		if err != nil {
			return nil, err
		}
		for _, query := range dirQueries {
			if _, ok := seen[query.Path]; ok {
				continue
			}
			seen[query.Path] = struct{}{}
			queries = append(queries, query)
		}
	}

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Path < queries[j].Path
	})
	return queries, nil
}

func loadSQLDir(dir string, readonly bool) ([]SavedQuery, error) {
	if strings.TrimSpace(dir) == "" {
		return []SavedQuery{}, nil
	}
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SavedQuery{}, nil
		}
		return nil, errors.Wrap(err, "stating SQL directory")
	}

	queries := make([]SavedQuery, 0)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".sql") {
			return nil
		}
		query, err := loadSingleSQLFile(dir, mustRelative(dir, path), readonly)
		if err != nil {
			return err
		}
		queries = append(queries, query)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "walking SQL directory")
	}

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Path < queries[j].Path
	})
	return queries, nil
}

func loadSingleSQLFile(rootDir, relativePath string, readonly bool) (SavedQuery, error) {
	absolutePath, cleanRelativePath, err := safeQueryPath(rootDir, relativePath)
	if err != nil {
		return SavedQuery{}, err
	}
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return SavedQuery{}, errors.Wrap(err, "reading SQL file")
	}

	folder := filepath.Dir(cleanRelativePath)
	if folder == "." {
		folder = ""
	}
	name := strings.TrimSuffix(filepath.Base(cleanRelativePath), filepath.Ext(cleanRelativePath))
	sqlText := string(content)

	return SavedQuery{
		Name:        name,
		Folder:      filepath.ToSlash(folder),
		Path:        filepath.ToSlash(cleanRelativePath),
		Description: extractSQLComment(sqlText),
		SQL:         sqlText,
		Readonly:    readonly,
	}, nil
}

func extractSQLComment(sqlText string) string {
	for _, line := range strings.Split(sqlText, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-- ") {
			return strings.TrimPrefix(line, "-- ")
		}
		if line != "" && !strings.HasPrefix(line, "--") {
			break
		}
	}
	return ""
}

func buildSQLFileContent(description string, sqlText string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return sqlText
	}
	return "-- " + description + "\n" + sqlText
}

func buildQueryCreatePath(baseDir, folder, name string) (string, string, error) {
	safeName := sanitizeFilename(name)
	if safeName == "" {
		return "", "", errors.New("query name is required")
	}
	relativePath := safeName + ".sql"
	if strings.TrimSpace(folder) != "" {
		cleanFolder, err := cleanRelativePath(folder)
		if err != nil {
			return "", "", err
		}
		relativePath = filepath.ToSlash(filepath.Join(cleanFolder, relativePath))
	}
	absolutePath, cleanRelativePath, err := safeQueryPath(baseDir, relativePath)
	if err != nil {
		return "", "", err
	}
	return cleanRelativePath, absolutePath, nil
}

func safeQueryPath(baseDir, relativePath string) (string, string, error) {
	cleanRelativePath, err := cleanRelativePath(relativePath)
	if err != nil {
		return "", "", err
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", "", errors.Wrap(err, "resolving base directory")
	}
	absolutePath := filepath.Join(baseAbs, filepath.FromSlash(cleanRelativePath))
	absolutePath, err = filepath.Abs(absolutePath)
	if err != nil {
		return "", "", errors.Wrap(err, "resolving query path")
	}

	prefix := baseAbs + string(filepath.Separator)
	if absolutePath != baseAbs && !strings.HasPrefix(absolutePath, prefix) {
		return "", "", errors.New("query path escapes query directory")
	}

	return absolutePath, cleanRelativePath, nil
}

func cleanRelativePath(path string) (string, error) {
	path = strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	if path == "" {
		return "", errors.New("path is required")
	}
	if strings.HasPrefix(path, "/") {
		return "", errors.New("absolute paths are not allowed")
	}

	cleaned := pathpkgClean(path)
	if cleaned == "." || cleaned == "" {
		return "", errors.New("path is required")
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", errors.New("parent directory traversal is not allowed")
	}

	return cleaned, nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}

	safeName := strings.Trim(builder.String(), "-")
	return strings.TrimSpace(safeName)
}

func pathpkgClean(path string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
}

func mustRelative(rootDir, path string) string {
	relativePath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relativePath)
}
