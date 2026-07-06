package serve

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	queryTarget        minitracedb.QueryTarget
	presetDirs         []string
	queryDirs          []string
	archiveGlobs       []string
	sessionIndex       map[string]string
	annoStore          *annotate.Store
	annoIndex          map[string]string
	devMode            bool
	commandSourceRoots []minitracecmd.SourceRoot
	mux                *http.ServeMux
}

type QueryRequest struct {
	SQL string `json:"sql"`
}

type QueryError struct {
	Message string `json:"message"`
}

// QueryResponse intentionally remains JSON-native in phase 1 of the protobuf API rollout.
// The execute-query surface returns arbitrary row shapes (`[]map[string]any`) whose schema
// depends on user SQL, so sessions, annotations, and saved-query metadata were migrated first.
// A future wrapper could use google.protobuf.Struct if a protobuf envelope becomes worthwhile.
type QueryResponse struct {
	Columns    []string         `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	DurationMS int64            `json:"duration_ms"`
	RowCount   int              `json:"row_count"`
	Error      *QueryError      `json:"error,omitempty"`
}

func NewServer(
	queryTarget minitracedb.QueryTarget,
	settings *ServeSettings,
	sessionIndex map[string]string,
	annoStore *annotate.Store,
	annoIndex map[string]string,
) *Server {
	s := &Server{
		queryTarget:  queryTarget,
		presetDirs:   normalizeDirList(settings.PresetDir),
		queryDirs:    normalizeDirList(settings.QueryDir),
		archiveGlobs: append([]string(nil), settings.ArchiveGlob...),
		sessionIndex: sessionIndex,
		annoStore:    annoStore,
		annoIndex:    annoIndex,
		devMode:      settings.DevMode,
	}
	s.mux = http.NewServeMux()
	s.routes()
	return s
}

func (s *Server) queryCommandCatalog() (*minitracecmd.Catalog, error) {
	if len(s.commandSourceRoots) == 0 {
		return minitracecmd.LoadEmbeddedCatalog()
	}
	return minitracecmd.LoadCatalog(s.commandSourceRoots)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v2/sessions", s.handleGetSessionsV2)
	s.mux.HandleFunc("GET /api/v2/sessions/{id}", s.handleGetSessionV2)
	s.mux.HandleFunc("GET /api/v2/sessions/{id}/summary", s.handleGetSessionSummaryV2)
	s.mux.HandleFunc("GET /api/v2/sessions/{id}/blocks", s.handleGetSessionBlocksV2)
	s.mux.HandleFunc("POST /api/query", s.handleExecuteQuery)
	s.mux.HandleFunc("GET /api/v2/presets", s.handleGetPresetsV2)
	s.mux.HandleFunc("GET /api/v2/queries", s.handleGetQueriesV2)
	s.mux.HandleFunc("POST /api/v2/queries", s.handleSaveQueryV2)
	s.mux.HandleFunc("PUT /api/v2/queries/{path...}", s.handleUpdateQueryV2)
	s.mux.HandleFunc("DELETE /api/v2/queries/{path...}", s.handleDeleteQueryV2)
	s.mux.HandleFunc("GET /api/v2/query-commands", s.handleGetQueryCommandsV2)
	s.mux.HandleFunc("POST /api/v2/query-commands/{path...}", s.handleExecuteQueryCommandV2)

	// Annotation routes.
	s.mux.HandleFunc("GET /api/v2/sessions/{id}/annotations", s.handleGetSessionAnnotationsV2)
	s.mux.HandleFunc("POST /api/v2/sessions/{id}/annotations", s.handleCreateAnnotationV2)
	s.mux.HandleFunc("GET /api/v2/annotations", s.handleListAnnotationsV2)
	s.mux.HandleFunc("PUT /api/v2/annotations/{annId}", s.handleUpdateAnnotationV2)
	s.mux.HandleFunc("DELETE /api/v2/annotations/{annId}", s.handleDeleteAnnotationV2)
	s.mux.HandleFunc("POST /api/v2/annotations/sync", s.handleSyncAnnotationsV2)

	if !s.devMode {
		s.mux.Handle("/", spaHandler(frontendFS))
	}
}

func (s *Server) ListenAndServe(ctx context.Context, port int) error {
	if port <= 0 {
		return errors.Errorf("port must be positive, got %d", port)
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		log.Info().Str("addr", httpServer.Addr).Bool("dev_mode", s.devMode).Msg("starting serve HTTP server")
		err := httpServer.ListenAndServe()
		if err == nil || stderrors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.Wrap(err, "running serve HTTP server")
	})

	group.Go(func() error {
		<-groupCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "shutting down serve HTTP server")
		}
		return nil
	})

	return group.Wait()
}

func (s *Server) handleExecuteQuery(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	req := QueryRequest{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "decoding query request").Error()},
		})
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: "sql is required"},
		})
		return
	}
	if s.queryTarget == nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: "query target is not initialized"},
		})
		return
	}

	// The sandboxed query runner validates read-only-ness, enforces the
	// lifted serve limits, and returns structured errors as result.Error.
	result, err := s.queryTarget.QueryResult(r.Context(), req.SQL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	if result.Error != "" {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: result.Error},
		})
		return
	}

	columns := result.Columns
	if columns == nil {
		columns = []string{}
	}
	rows := result.Rows
	if rows == nil {
		rows = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, QueryResponse{
		Columns:    columns,
		Rows:       rows,
		DurationMS: time.Since(start).Milliseconds(),
		RowCount:   result.Count,
	})
}

func buildSessionIndex(archiveGlobs []string) (map[string]string, error) {
	files, err := minitracedb.ExpandArchiveGlobs(archiveGlobs)
	if err != nil {
		return nil, err
	}

	index := make(map[string]string, len(files))
	for _, filePath := range files {
		sessionID, err := readSessionIDFromArchive(filePath)
		if err != nil {
			return nil, err
		}
		if previous, ok := index[sessionID]; ok {
			return nil, errors.Errorf("duplicate session ID %q found in %s and %s", sessionID, previous, filePath)
		}
		index[sessionID] = filePath
	}
	if len(index) == 0 {
		return nil, errors.Errorf("archive globs matched no .minitrace.json files")
	}
	return index, nil
}

func readSessionIDFromArchive(filePath string) (string, error) {
	payload, err := fs.ReadFile(os.DirFS(filepath.Dir(filePath)), filepath.Base(filePath))
	if err != nil {
		return "", errors.Wrap(err, "reading session archive for indexing")
	}

	var meta struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return "", errors.Wrap(err, "unmarshaling session archive for indexing")
	}
	if strings.TrimSpace(meta.ID) == "" {
		return "", errors.Errorf("session archive %q is missing id", filePath)
	}
	return meta.ID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error().Err(err).Msg("writing JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func normalizeDirList(dirs []string) []string {
	if len(dirs) == 0 {
		return []string{}
	}

	ret := make([]string, 0, len(dirs))
	seen := make(map[string]struct{}, len(dirs))
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		ret = append(ret, dir)
	}
	return ret
}

func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		file, err := fsys.Open(path)
		if err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		indexHTML, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.Error(w, "frontend index.html not found", http.StatusNotFound)
			return
		}

		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(indexHTML))
	})
}
