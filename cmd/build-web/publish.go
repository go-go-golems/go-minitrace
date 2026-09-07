package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// Build/export away from the live embed directory. A failed or interrupted
// build must not remove the checked-in placeholder or a working UI bundle.
func publishFrontend(embedDir string, export func(string) error) error {
	parent := filepath.Dir(embedDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create frontend parent: %w", err)
	}
	staging, err := os.MkdirTemp(parent, ".frontend-stage-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := export(staging); err != nil {
		return fmt.Errorf("export staged frontend: %w", err)
	}
	info, err := os.Stat(filepath.Join(staging, "index.html"))
	if err != nil {
		return fmt.Errorf("validate staged frontend: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("staged frontend index.html is not a regular file")
	}
	if err := os.WriteFile(filepath.Join(staging, ".gitkeep"), nil, 0o644); err != nil {
		return fmt.Errorf("create frontend placeholder: %w", err)
	}

	backup, err := os.MkdirTemp(parent, ".frontend-backup-")
	if err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	hadPrevious := false
	if err := os.Rename(embedDir, backup); err == nil {
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("retain previous frontend: %w", err)
	}
	if err := os.Rename(staging, embedDir); err != nil {
		if hadPrevious {
			if restoreErr := os.Rename(backup, embedDir); restoreErr != nil {
				return fmt.Errorf("publish frontend: %w; rollback failed: %v; previous assets remain at %s", err, restoreErr, backup)
			}
		}
		return fmt.Errorf("publish frontend: %w", err)
	}
	if hadPrevious {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove previous frontend backup: %w", err)
		}
	}
	return nil
}
