package serve

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	conn         *sql.Conn
	tableName    string
	presetDirs   []string
	queryDirs    []string
	sessionIndex map[string]string
	devMode      bool
	mux          *http.ServeMux
}

type QueryRequest struct {
	SQL string `json:"sql"`
}

type QueryError struct {
	Message string `json:"message"`
}

type QueryResponse struct {
	Columns    []string         `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	DurationMS int64            `json:"duration_ms"`
	RowCount   int              `json:"row_count"`
	Error      *QueryError      `json:"error,omitempty"`
}

func NewServer(conn *sql.Conn, settings *ServeSettings, sessionIndex map[string]string) *Server {
	s := &Server{
		conn:         conn,
		tableName:    settings.TableName,
		presetDirs:   normalizeDirList(settings.PresetDir),
		queryDirs:    normalizeDirList(settings.QueryDir),
		sessionIndex: sessionIndex,
		devMode:      settings.DevMode,
	}
	s.mux = http.NewServeMux()
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/sessions", s.handleGetSessions)
	s.mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
	s.mux.HandleFunc("GET /api/sessions/{id}/blocks", s.handleGetSessionBlocks)
	s.mux.HandleFunc("POST /api/query", s.handleExecuteQuery)
	s.mux.HandleFunc("GET /api/presets", s.handleGetPresets)
	s.mux.HandleFunc("GET /api/queries", s.handleGetQueries)
	s.mux.HandleFunc("POST /api/queries", s.handleSaveQuery)
	s.mux.HandleFunc("PUT /api/queries/{path...}", s.handleUpdateQuery)
	s.mux.HandleFunc("DELETE /api/queries/{path...}", s.handleDeleteQuery)
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

	rows, err := s.conn.QueryContext(r.Context(), req.SQL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: err.Error()},
		})
		return
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "reading query columns").Error()},
		})
		return
	}

	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			writeJSON(w, http.StatusInternalServerError, QueryResponse{
				Columns:    []string{},
				Rows:       []map[string]any{},
				DurationMS: time.Since(start).Milliseconds(),
				RowCount:   0,
				Error:      &QueryError{Message: errors.Wrap(err, "scanning query row").Error()},
			})
			return
		}

		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = queryengine.NormalizeValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, QueryResponse{
			Columns:    []string{},
			Rows:       []map[string]any{},
			DurationMS: time.Since(start).Milliseconds(),
			RowCount:   0,
			Error:      &QueryError{Message: errors.Wrap(err, "iterating query rows").Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, QueryResponse{
		Columns:    columns,
		Rows:       resultRows,
		DurationMS: time.Since(start).Milliseconds(),
		RowCount:   len(resultRows),
	})
}

func buildSessionIndex(archiveGlob string) (map[string]string, error) {
	if strings.TrimSpace(archiveGlob) == "" {
		return nil, errors.New("archive glob is required")
	}

	files, err := filepath.Glob(archiveGlob)
	if err != nil {
		return nil, errors.Wrap(err, "expanding archive glob")
	}
	if len(files) == 0 {
		return nil, errors.Errorf("archive glob matched no files: %s", archiveGlob)
	}

	index := make(map[string]string, len(files))
	for _, filePath := range files {
		base := filepath.Base(filePath)
		sessionID := strings.TrimSuffix(base, ".minitrace.json")
		if sessionID == base {
			continue
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving absolute path for %s", filePath)
		}
		if previous, ok := index[sessionID]; ok {
			return nil, errors.Errorf("duplicate session ID %q found in %s and %s", sessionID, previous, absPath)
		}
		index[sessionID] = absPath
	}
	if len(index) == 0 {
		return nil, errors.Errorf("archive glob matched no .minitrace.json files: %s", archiveGlob)
	}
	return index, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error().Err(err).Msg("writing JSON response")
	}
}

func decodeRequest(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return errors.Wrap(err, "decoding request body")
	}
	return nil
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
