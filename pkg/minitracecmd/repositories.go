package minitracecmd

import (
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
)

type AppConfig struct {
	QueryRepositories []string `yaml:"queryRepositories"`
}

func loadAppConfig(appName string) (*AppConfig, error) {
	configPath, err := glazedconfig.ResolveAppConfigPath(appName, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve app config path")
	}

	return loadAppConfigFromPath(configPath)
}

func loadAppConfigFromPath(configPath string) (*AppConfig, error) {
	if configPath == "" {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read app config")
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "could not parse app config")
	}

	cfg.QueryRepositories = normalizeRepositoryPaths(cfg.QueryRepositories)
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
	ret := make([]string, 0, len(paths))
	seen := map[string]struct{}{}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		ret = append(ret, path)
	}

	return ret
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
			values = append(values, args[i+1])
			i++
		case strings.HasPrefix(arg, "--"+QueryRepositoryFlagName+"="):
			values = append(values, strings.TrimPrefix(arg, "--"+QueryRepositoryFlagName+"="))
		}
	}
	return normalizeRepositoryPaths(values)
}
