package minitracecmd

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"

	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	QueryRepositoriesEnvVar = "GO_MINITRACE_QUERY_REPOSITORIES"
	QueryRepositoryFlagName = "query-repository"

	LocalOverrideFileName        = ".go-minitrace.yml"
	LocalProjectOverrideFileName = ".go-minitrace.override.yml"
)

type AppConfig struct {
	QueryRepositories []string `yaml:"queryRepositories"`
}

func loadAppConfig(appName string) (*AppConfig, error) {
	configPaths, err := resolveAppConfigPaths(appName)
	if err != nil {
		return nil, err
	}

	return loadAppConfigFromPaths(configPaths)
}

func resolveAppConfigPaths(appName string) ([]string, error) {
	files, _, err := glazedconfig.NewPlan(
		glazedconfig.WithLayerOrder(
			glazedconfig.LayerSystem,
			glazedconfig.LayerUser,
			glazedconfig.LayerRepo,
			glazedconfig.LayerCWD,
		),
		glazedconfig.WithDedupePaths(),
	).Add(
		glazedconfig.SystemAppConfig(appName).Named("system-app-config").Kind("app-config"),
		glazedconfig.HomeAppConfig(appName).Named("home-app-config").Kind("app-config"),
		glazedconfig.XDGAppConfig(appName).Named("xdg-app-config").Kind("app-config"),
		glazedconfig.GitRootFile(LocalOverrideFileName).Named("git-root-local-go-minitrace").Kind("query-repository-overlay"),
		glazedconfig.GitRootFile(LocalProjectOverrideFileName).Named("git-root-local-go-minitrace-override").Kind("query-repository-overlay"),
		glazedconfig.WorkingDirFile(LocalOverrideFileName).Named("cwd-local-go-minitrace").Kind("query-repository-overlay"),
		glazedconfig.WorkingDirFile(LocalProjectOverrideFileName).Named("cwd-local-go-minitrace-override").Kind("query-repository-overlay"),
	).Resolve(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve app config files")
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		if strings.TrimSpace(file.Path) == "" {
			continue
		}
		paths = append(paths, file.Path)
	}
	return paths, nil
}

func loadAppConfigFromPaths(configPaths []string) (*AppConfig, error) {
	cfg := &AppConfig{}
	for _, configPath := range configPaths {
		partial, err := loadAppConfigFromPath(configPath)
		if err != nil {
			return nil, err
		}
		if partial.QueryRepositories != nil {
			cfg.QueryRepositories = append([]string(nil), partial.QueryRepositories...)
		}
	}
	cfg.QueryRepositories = normalizeRepositoryPaths(cfg.QueryRepositories)
	return cfg, nil
}

func loadAppConfigFromPath(configPath string) (*AppConfig, error) {
	if configPath == "" {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read app config")
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, errors.Wrap(err, "could not parse app config")
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "could not parse app config")
	}

	if _, ok := raw["queryRepositories"]; ok {
		cfg.QueryRepositories = normalizeRepositoryPathsRelativeTo(cfg.QueryRepositories, filepath.Dir(configPath))
	} else {
		cfg.QueryRepositories = nil
	}
	return &cfg, nil
}

func CollectRepositoryPaths(appName string, flagPaths []string) ([]string, error) {
	cfg, err := loadAppConfig(appName)
	if err != nil {
		return nil, err
	}

	return collectRepositoryPaths(cfg, flagPaths), nil
}

func collectRepositoryPaths(cfg *AppConfig, flagPaths []string) []string {
	repositoryPaths := append([]string{}, flagPaths...)
	repositoryPaths = append(repositoryPaths, repositoriesFromEnv()...)
	if cfg != nil {
		repositoryPaths = append(repositoryPaths, cfg.QueryRepositories...)
	}
	return normalizeRepositoryPaths(repositoryPaths)
}

func repositoriesFromEnv() []string {
	value, ok := os.LookupEnv(QueryRepositoriesEnvVar)
	if !ok || value == "" {
		return nil
	}

	return normalizeRepositoryPaths(filepath.SplitList(value))
}

func normalizeRepositoryPaths(paths []string) []string {
	return normalizeRepositoryPathsWithBase(paths, "")
}

func normalizeRepositoryPathsRelativeTo(paths []string, baseDir string) []string {
	return normalizeRepositoryPathsWithBase(paths, strings.TrimSpace(baseDir))
}

func normalizeRepositoryPathsWithBase(paths []string, baseDir string) []string {
	ret := make([]string, 0, len(paths))
	seen := map[string]struct{}{}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if baseDir != "" && shouldResolveRelativeToConfig(path) {
			path = filepath.Clean(filepath.Join(baseDir, path))
		} else if filepath.IsAbs(path) {
			path = filepath.Clean(path)
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		ret = append(ret, path)
	}

	return ret
}

func shouldResolveRelativeToConfig(path string) bool {
	if filepath.IsAbs(path) {
		return false
	}
	return !strings.HasPrefix(path, "$") && !strings.HasPrefix(path, "~")
}

func CommandSourceRoots(appName string, flagPaths []string) ([]SourceRoot, error) {
	paths, err := CollectRepositoryPaths(appName, flagPaths)
	if err != nil {
		return nil, err
	}
	return SourceRootsFromPaths(paths), nil
}

func SourceRootsFromPaths(paths []string) []SourceRoot {
	roots := make([]SourceRoot, 0, len(paths)+1)
	for _, path := range normalizeRepositoryPaths(paths) {
		dir := filepath.Clean(os.ExpandEnv(path))
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			continue
		}
		roots = append(roots, SourceRoot{
			Name:     dir,
			FS:       os.DirFS(dir),
			RootDir:  ".",
			Readonly: true,
		})
	}
	roots = append(roots, EmbeddedSourceRoot())
	return roots
}

func LoadConfiguredCatalog(appName string, flagPaths []string) (*Catalog, error) {
	roots, err := CommandSourceRoots(appName, flagPaths)
	if err != nil {
		return nil, err
	}
	return LoadCatalog(roots)
}

func ExtractRepositoryFlagValuesFromArgs(args []string) []string {
	values := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return normalizeRepositoryPaths(values)
		case arg == "--"+QueryRepositoryFlagName:
			if i+1 >= len(args) {
				continue
			}
			values = append(values, splitRepositoryFlagValue(args[i+1])...)
			i++
		case strings.HasPrefix(arg, "--"+QueryRepositoryFlagName+"="):
			values = append(values, splitRepositoryFlagValue(strings.TrimPrefix(arg, "--"+QueryRepositoryFlagName+"="))...)
		}
	}
	return normalizeRepositoryPaths(values)
}

func splitRepositoryFlagValue(value string) []string {
	if value == "" {
		return nil
	}

	reader := csv.NewReader(strings.NewReader(value))
	values, err := reader.Read()
	if err != nil {
		return []string{value}
	}
	return values
}
