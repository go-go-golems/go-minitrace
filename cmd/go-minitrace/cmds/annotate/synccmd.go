package annotate

//glazedclilint:file-ignore legacy annotate command uses Cobra flags pending Glazed field migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/spf13/cobra"
)

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync annotations back to .minitrace.json files",
		Long: `Read annotations from the SQLite store and write them back to the
corresponding .minitrace.json files.

The session index is built by expanding --archive-glob, reading each JSON file,
and extracting the session ID. Use --dry-run to preview changes before writing.

If --session is provided, only that session is synced.
`,
		RunE: runSync,
	}
	cmd.Flags().String(
		"archive-glob",
		"",
		"Glob pattern for .minitrace.json files (default: <output-dir>/active/*/*.minitrace.json)",
	)
	cmd.Flags().String("session", "", "Sync only this session ID")
	cmd.Flags().Bool("dry-run", false, "Preview changes without writing files")
	return cmd
}

func runSync(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, outputDir, err := openStore(cmd)
	if err != nil {
		return err
	}
	defer closeStore(store)

	archiveGlob, _ := cmd.Flags().GetString("archive-glob")
	if archiveGlob == "" {
		// Infer from output-dir: <output-dir>/active/*/*.minitrace.json
		archiveGlob = filepath.Join(outputDir, "active", "*", "*.minitrace.json")
	}

	sessionID, _ := cmd.Flags().GetString("session")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Build session index: session ID → absolute file path.
	sessionIndex, err := buildSessionIndex(archiveGlob)
	if err != nil {
		return fmt.Errorf("building session index: %w", err)
	}
	if len(sessionIndex) == 0 {
		fmt.Println("No .minitrace.json files found.")
		return nil
	}

	opts := annotate.SyncOptions{
		DryRun:    dryRun,
		SessionID: sessionID,
	}

	report, err := store.SyncAll(ctx, sessionIndex, opts)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Report results.
	fmt.Printf("Synced: %d | Skipped: %d | Errors: %d\n",
		len(report.Synced), len(report.Skipped), len(report.Errors))

	if len(report.Synced) > 0 && !dryRun {
		fmt.Println("\nSynced sessions:")
		for _, id := range report.Synced {
			fmt.Printf("  %s\n", id)
		}
	}

	if len(report.Skipped) > 0 {
		fmt.Println("\nSkipped (no file found):")
		for _, id := range report.Skipped {
			fmt.Printf("  %s\n", id)
		}
	}

	if len(report.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range report.Errors {
			fmt.Printf("  %s: %s\n", e.SessionID, e.Error)
		}
		return fmt.Errorf("sync completed with %d error(s)", len(report.Errors))
	}

	return nil
}

// buildSessionIndex expands archiveGlob and reads each file to extract the session ID,
// returning a map from session ID to absolute file path.
func buildSessionIndex(archiveGlob string) (map[string]string, error) {
	files, err := query.ExpandArchiveGlobs([]string{archiveGlob})
	if err != nil {
		return nil, err
	}

	index := make(map[string]string, len(files))
	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filePath, err)
		}
		var session struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filePath, err)
		}
		if session.ID == "" {
			return nil, fmt.Errorf("session ID is empty in %s", filePath)
		}
		index[session.ID] = filePath
	}

	return index, nil
}

// GetSessionFromFile reads a .minitrace.json file and returns the session.
func GetSessionFromFile(filePath string) (*minitrace.Session, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var session minitrace.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}
