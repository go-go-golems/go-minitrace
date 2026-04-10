package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

const (
	defaultPNPMVersion  = "10.15.0"
	defaultBuilderImage = "node:22-bookworm"
)

func main() {
	if err := buildAndExportFrontend(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildAndExportFrontend(ctx context.Context) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	webDir := filepath.Join(repoRoot, "web")
	embedDir := filepath.Join(repoRoot, "cmd", "go-minitrace", "cmds", "serve", "frontend")
	if err := os.RemoveAll(embedDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old embed assets: %w", err)
	}

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("connect dagger: %w", err)
	}
	defer func() { _ = client.Close() }()

	pnpmVersion := strings.TrimSpace(os.Getenv("WEB_PNPM_VERSION"))
	if pnpmVersion == "" {
		pnpmVersion = defaultPNPMVersion
	}

	builderImage := strings.TrimSpace(os.Getenv("WEB_BUILDER_IMAGE"))
	if builderImage == "" {
		builderImage = defaultBuilderImage
	}

	source := client.Host().Directory(webDir, dagger.HostDirectoryOpts{
		Exclude: []string{
			"dist",
			"node_modules",
			"storybook-static",
			"tsconfig.tsbuildinfo",
			"tsconfig.app.tsbuildinfo",
			"tsconfig.node.tsbuildinfo",
		},
	})

	pnpmStore := client.CacheVolume("go-minitrace-web-pnpm-store")
	pathValue := "/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	container := client.Container().
		From(builderImage).
		WithEnvVariable("PNPM_HOME", "/pnpm").
		WithEnvVariable("PATH", pathValue).
		WithMountedCache("/pnpm/store", pnpmStore).
		WithDirectory("/src/web", source).
		WithWorkdir("/src/web").
		WithExec([]string{"sh", "-lc", "corepack enable && corepack prepare pnpm@" + pnpmVersion + " --activate"}).
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"pnpm", "run", "build"})

	if _, err := container.Directory("/src/web/dist").Export(ctx, embedDir); err != nil {
		return fmt.Errorf("export built frontend into serve/frontend: %w", err)
	}
	if err := os.WriteFile(filepath.Join(embedDir, ".gitkeep"), []byte{}, 0o644); err != nil {
		return fmt.Errorf("restore frontend .gitkeep: %w", err)
	}

	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
