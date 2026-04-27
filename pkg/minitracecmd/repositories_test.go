package minitracecmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadAppConfigFromPath_LoadsQueryRepositories(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("queryRepositories:\n  - ./queries/team\n  - ./queries/shared\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := loadAppConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("loadAppConfigFromPath returned error: %v", err)
	}

	want := []string{filepath.Join(dir, "queries/team"), filepath.Join(dir, "queries/shared")}
	if len(cfg.QueryRepositories) != len(want) {
		t.Fatalf("QueryRepositories length = %d, want %d (%#v)", len(cfg.QueryRepositories), len(want), cfg.QueryRepositories)
	}
	for i, path := range want {
		if cfg.QueryRepositories[i] != path {
			t.Fatalf("QueryRepositories[%d] = %q, want %q", i, cfg.QueryRepositories[i], path)
		}
	}
}

func TestCollectRepositoryPaths_PrioritizesFlagsThenEnvThenConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("queryRepositories:\n  - config-a\n  - shared\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv(QueryRepositoriesEnvVar, "env-a:shared")
	flagPaths := []string{"flag-a", "shared"}

	cfg, err := loadAppConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("loadAppConfigFromPath returned error: %v", err)
	}

	got := collectRepositoryPaths(cfg, flagPaths)
	want := []string{"flag-a", "shared", "env-a", filepath.Join(dir, "config-a"), filepath.Join(dir, "shared")}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i, path := range want {
		if got[i] != path {
			t.Fatalf("got[%d] = %q, want %q (all=%#v)", i, got[i], path, got)
		}
	}
}

func TestLoadAppConfigFromPaths_OverlaysLaterFiles(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.yaml")
	userPath := filepath.Join(dir, "user.yaml")
	if err := os.WriteFile(basePath, []byte("queryRepositories:\n  - base-a\n  - shared\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(base) returned error: %v", err)
	}
	if err := os.WriteFile(userPath, []byte("queryRepositories:\n  - user-a\n  - shared\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(user) returned error: %v", err)
	}

	cfg, err := loadAppConfigFromPaths([]string{basePath, userPath})
	if err != nil {
		t.Fatalf("loadAppConfigFromPaths returned error: %v", err)
	}

	want := []string{filepath.Join(dir, "user-a"), filepath.Join(dir, "shared")}
	if len(cfg.QueryRepositories) != len(want) {
		t.Fatalf("len(cfg.QueryRepositories) = %d, want %d (%#v)", len(cfg.QueryRepositories), len(want), cfg.QueryRepositories)
	}
	for i, path := range want {
		if cfg.QueryRepositories[i] != path {
			t.Fatalf("cfg.QueryRepositories[%d] = %q, want %q", i, cfg.QueryRepositories[i], path)
		}
	}
}

func TestLoadAppConfigFromPaths_KeepsEarlierValuesWhenLaterFileOmitsField(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.yaml")
	userPath := filepath.Join(dir, "user.yaml")
	if err := os.WriteFile(basePath, []byte("queryRepositories:\n  - base-a\n  - shared\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(base) returned error: %v", err)
	}
	if err := os.WriteFile(userPath, []byte("otherField: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(user) returned error: %v", err)
	}

	cfg, err := loadAppConfigFromPaths([]string{basePath, userPath})
	if err != nil {
		t.Fatalf("loadAppConfigFromPaths returned error: %v", err)
	}

	want := []string{filepath.Join(dir, "base-a"), filepath.Join(dir, "shared")}
	if len(cfg.QueryRepositories) != len(want) {
		t.Fatalf("len(cfg.QueryRepositories) = %d, want %d (%#v)", len(cfg.QueryRepositories), len(want), cfg.QueryRepositories)
	}
	for i, path := range want {
		if cfg.QueryRepositories[i] != path {
			t.Fatalf("cfg.QueryRepositories[%d] = %q, want %q", i, cfg.QueryRepositories[i], path)
		}
	}
}

func TestLoadAppConfigFromPath_KeepsEnvAndHomeRelativePathsUnresolved(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("queryRepositories:\n  - $HOME/queries\n  - ~/queries\n  - /absolute/queries\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := loadAppConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("loadAppConfigFromPath returned error: %v", err)
	}

	want := []string{"$HOME/queries", "~/queries", "/absolute/queries"}
	if len(cfg.QueryRepositories) != len(want) {
		t.Fatalf("QueryRepositories length = %d, want %d (%#v)", len(cfg.QueryRepositories), len(want), cfg.QueryRepositories)
	}
	for i, path := range want {
		if cfg.QueryRepositories[i] != path {
			t.Fatalf("QueryRepositories[%d] = %q, want %q", i, cfg.QueryRepositories[i], path)
		}
	}
}

func TestResolveAppConfigPaths_IncludesGitRootAndWorkingDirLocalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	repoRoot := t.TempDir()
	if err := exec.Command("git", "-C", repoRoot, "init").Run(); err != nil {
		t.Fatalf("git init returned error: %v", err)
	}
	subdir := filepath.Join(repoRoot, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll(subdir) returned error: %v", err)
	}
	gitRootConfig := filepath.Join(repoRoot, LocalOverrideFileName)
	cwdOverrideConfig := filepath.Join(subdir, LocalProjectOverrideFileName)
	if err := os.WriteFile(gitRootConfig, []byte("queryRepositories:\n  - ./repo-queries\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(gitRootConfig) returned error: %v", err)
	}
	if err := os.WriteFile(cwdOverrideConfig, []byte("queryRepositories:\n  - ./cwd-private\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cwdOverrideConfig) returned error: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	paths, err := resolveAppConfigPaths("go-minitrace")
	if err != nil {
		t.Fatalf("resolveAppConfigPaths returned error: %v", err)
	}
	want := []string{gitRootConfig, cwdOverrideConfig}
	if len(paths) != len(want) {
		t.Fatalf("len(paths) = %d, want %d (%#v)", len(paths), len(want), paths)
	}
	for i, path := range want {
		if paths[i] != filepath.Clean(path) {
			t.Fatalf("paths[%d] = %q, want %q (all=%#v)", i, paths[i], filepath.Clean(path), paths)
		}
	}
}

func TestLoadAppConfigFromPaths_LocalOverrideReplacesGitRootRepositories(t *testing.T) {
	repoRoot := t.TempDir()
	subdir := filepath.Join(repoRoot, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll(subdir) returned error: %v", err)
	}
	gitRootConfig := filepath.Join(repoRoot, LocalOverrideFileName)
	cwdOverrideConfig := filepath.Join(subdir, LocalProjectOverrideFileName)
	if err := os.WriteFile(gitRootConfig, []byte("queryRepositories:\n  - ./repo-queries\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(gitRootConfig) returned error: %v", err)
	}
	if err := os.WriteFile(cwdOverrideConfig, []byte("queryRepositories:\n  - ./cwd-private\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cwdOverrideConfig) returned error: %v", err)
	}

	cfg, err := loadAppConfigFromPaths([]string{gitRootConfig, cwdOverrideConfig})
	if err != nil {
		t.Fatalf("loadAppConfigFromPaths returned error: %v", err)
	}

	want := []string{filepath.Join(subdir, "cwd-private")}
	if len(cfg.QueryRepositories) != len(want) {
		t.Fatalf("QueryRepositories length = %d, want %d (%#v)", len(cfg.QueryRepositories), len(want), cfg.QueryRepositories)
	}
	for i, path := range want {
		if cfg.QueryRepositories[i] != path {
			t.Fatalf("QueryRepositories[%d] = %q, want %q", i, cfg.QueryRepositories[i], path)
		}
	}
}

func TestSourceRootsFromPaths_PutsEmbeddedLastAndSkipsMissingDirs(t *testing.T) {
	repo := t.TempDir()
	roots := SourceRootsFromPaths([]string{"/missing/repo", repo})
	if len(roots) != 2 {
		t.Fatalf("len(roots) = %d, want 2", len(roots))
	}
	if roots[0].Name != repo {
		t.Fatalf("roots[0].Name = %q, want %q", roots[0].Name, repo)
	}
	if roots[1].Name != "embedded" {
		t.Fatalf("roots[1].Name = %q, want embedded", roots[1].Name)
	}
}

func TestExtractRepositoryFlagValuesFromArgs_SupportsSplitAndEqualsForms(t *testing.T) {
	args := []string{
		"query",
		"commands",
		"--query-repository", "./queries/team",
		"--query-repository=./queries/shared",
		"session-list",
	}

	got := ExtractRepositoryFlagValuesFromArgs(args)
	want := []string{"./queries/team", "./queries/shared"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i, path := range want {
		if got[i] != path {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], path)
		}
	}
}

func TestExtractRepositoryFlagValuesFromArgs_SplitsCommaSeparatedValues(t *testing.T) {
	args := []string{
		"query",
		"commands",
		"--query-repository", "./queries/team,./queries/shared",
		"--query-repository=./queries/extra,./queries/override",
	}

	got := ExtractRepositoryFlagValuesFromArgs(args)
	want := []string{"./queries/team", "./queries/shared", "./queries/extra", "./queries/override"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i, path := range want {
		if got[i] != path {
			t.Fatalf("got[%d] = %q, want %q (all=%#v)", i, got[i], path, got)
		}
	}
}
