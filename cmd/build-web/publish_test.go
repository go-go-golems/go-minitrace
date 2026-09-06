package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishFrontendPreservesExistingAssetsOnFailure(t *testing.T) {
	for _, failedExport := range []bool{true, false} {
		t.Run(map[bool]string{true: "export error", false: "missing index"}[failedExport], func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "frontend")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{".gitkeep", "index.html", "old.js"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("old"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := publishFrontend(dir, func(staging string) error {
				if err := os.WriteFile(filepath.Join(staging, "partial.js"), nil, 0o644); err != nil {
					return err
				}
				if failedExport {
					return errors.New("build failed")
				}
				return nil
			})
			if err == nil {
				t.Fatal("expected failure")
			}
			for _, name := range []string{".gitkeep", "index.html", "old.js"} {
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil || string(data) != "old" {
					t.Fatalf("existing asset %s lost: %s, %v", name, data, err)
				}
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 1 {
				t.Fatalf("staging leaked: %v, %v", entries, err)
			}
		})
	}
}

func TestPublishFrontendReplacesBundleAndKeepsPlaceholder(t *testing.T) {
	for _, existing := range []bool{true, false} {
		t.Run(map[bool]string{true: "replace", false: "first build"}[existing], func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "frontend")
			if existing {
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "old.js"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := publishFrontend(dir, func(staging string) error {
				return os.WriteFile(filepath.Join(staging, "index.html"), []byte("new"), 0o644)
			}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(dir, "index.html"))
			if err != nil || string(data) != "new" {
				t.Fatalf("new bundle missing: %v", err)
			}
			if _, err := os.Stat(filepath.Join(dir, ".gitkeep")); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, "old.js")); !os.IsNotExist(err) {
				t.Fatal("stale asset retained")
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 1 {
				t.Fatalf("temporary directory leaked: %v, %v", entries, err)
			}
		})
	}
}
