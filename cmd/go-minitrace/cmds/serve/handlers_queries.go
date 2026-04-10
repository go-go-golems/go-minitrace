package serve

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

type validatedQueryPath struct {
	rootDir      string
	relativePath string
	absolutePath string
}

func firstQueryDir(queryDirs []string) (string, error) {
	if len(queryDirs) == 0 {
		return "", errors.New("query-dir is not configured")
	}
	return queryDirs[0], nil
}

func findExistingQueryPath(queryDirs []string, relativePath string) (validatedQueryPath, error) {
	if len(queryDirs) == 0 {
		return validatedQueryPath{}, errors.New("query-dir is not configured")
	}

	cleanPath, err := cleanRelativePath(relativePath)
	if err != nil {
		return validatedQueryPath{}, err
	}

	for _, queryDir := range queryDirs {
		queryPath, err := newValidatedQueryPath(queryDir, cleanPath)
		if err != nil {
			return validatedQueryPath{}, err
		}
		if _, err := queryPath.Stat(); err == nil {
			return queryPath, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return validatedQueryPath{}, errors.Wrap(err, "stating query file")
		}
	}

	return validatedQueryPath{relativePath: cleanPath}, os.ErrNotExist
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
	queryPath, err := newValidatedQueryPath(rootDir, relativePath)
	if err != nil {
		return SavedQuery{}, err
	}
	content, err := queryPath.ReadFile()
	if err != nil {
		return SavedQuery{}, errors.Wrap(err, "reading SQL file")
	}

	folder := filepath.Dir(queryPath.relativePath)
	if folder == "." {
		folder = ""
	}
	name := strings.TrimSuffix(filepath.Base(queryPath.relativePath), filepath.Ext(queryPath.relativePath))
	sqlText := string(content)

	return SavedQuery{
		Name:        name,
		Folder:      filepath.ToSlash(folder),
		Path:        filepath.ToSlash(queryPath.relativePath),
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

func buildQueryCreatePath(baseDir, folder, name string) (validatedQueryPath, error) {
	safeName := sanitizeFilename(name)
	if safeName == "" {
		return validatedQueryPath{}, errors.New("query name is required")
	}
	relativePath := safeName + ".sql"
	if strings.TrimSpace(folder) != "" {
		cleanFolder, err := cleanRelativePath(folder)
		if err != nil {
			return validatedQueryPath{}, err
		}
		relativePath = filepath.ToSlash(filepath.Join(cleanFolder, relativePath))
	}
	return newValidatedQueryPath(baseDir, relativePath)
}

func newValidatedQueryPath(baseDir, relativePath string) (validatedQueryPath, error) {
	cleanPath, err := cleanRelativePath(relativePath)
	if err != nil {
		return validatedQueryPath{}, err
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return validatedQueryPath{}, errors.Wrap(err, "resolving base directory")
	}
	absolutePath := filepath.Join(baseAbs, filepath.FromSlash(cleanPath))
	absolutePath, err = filepath.Abs(absolutePath)
	if err != nil {
		return validatedQueryPath{}, errors.Wrap(err, "resolving query path")
	}

	relativeToBase, err := filepath.Rel(baseAbs, absolutePath)
	if err != nil {
		return validatedQueryPath{}, errors.Wrap(err, "resolving query path relative to base directory")
	}
	relativeToBase = filepath.ToSlash(relativeToBase)
	if relativeToBase == "." || relativeToBase == "" {
		return validatedQueryPath{}, errors.New("query path must point to a file")
	}
	if strings.HasPrefix(relativeToBase, "../") || relativeToBase == ".." || filepath.IsAbs(relativeToBase) {
		return validatedQueryPath{}, errors.New("query path escapes query directory")
	}

	return validatedQueryPath{
		rootDir:      baseAbs,
		relativePath: relativeToBase,
		absolutePath: absolutePath,
	}, nil
}

func (p validatedQueryPath) ensureWithinRoot() error {
	if strings.TrimSpace(p.rootDir) == "" || strings.TrimSpace(p.relativePath) == "" || strings.TrimSpace(p.absolutePath) == "" {
		return errors.New("query path is not initialized")
	}
	relativeToRoot, err := filepath.Rel(p.rootDir, p.absolutePath)
	if err != nil {
		return errors.Wrap(err, "resolving query path relative to root")
	}
	relativeToRoot = filepath.ToSlash(relativeToRoot)
	if relativeToRoot == "." || relativeToRoot == "" {
		return errors.New("query path must point to a file")
	}
	if strings.HasPrefix(relativeToRoot, "../") || relativeToRoot == ".." || filepath.IsAbs(relativeToRoot) {
		return errors.New("query path escapes query directory")
	}
	return nil
}

func (p validatedQueryPath) MkdirAllParent() error {
	if err := p.ensureWithinRoot(); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(p.absolutePath), 0o755)
}

func (p validatedQueryPath) WriteFile(content []byte) error {
	if err := p.ensureWithinRoot(); err != nil {
		return err
	}
	return os.WriteFile(p.absolutePath, content, 0o644)
}

func (p validatedQueryPath) ReadFile() ([]byte, error) {
	if err := p.ensureWithinRoot(); err != nil {
		return nil, err
	}
	return os.ReadFile(p.absolutePath)
}

func (p validatedQueryPath) Remove() error {
	if err := p.ensureWithinRoot(); err != nil {
		return err
	}
	return os.Remove(p.absolutePath)
}

func (p validatedQueryPath) Stat() (os.FileInfo, error) {
	if err := p.ensureWithinRoot(); err != nil {
		return nil, err
	}
	return os.Stat(p.absolutePath)
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
