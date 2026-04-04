package annotate

import (
	"github.com/spf13/cobra"
)

const flagOutputDir = "output-dir"

// NewCommand returns the root annotate cobra command with all subcommands registered.
func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "annotate",
		Short: "Manage annotations on minitrace sessions",
		Long: `Annotate and classify minitrace sessions and their tool calls.

Annotations are stored in a SQLite database at outputDir/annotations.db.
The .minitrace.json source files are modified only via "annotate sync".

Storage model:
  - Annotations are written to outputDir/annotations.db (the working store).
  - The source .minitrace.json files are the canonical interchange format.
  - Use "annotate sync" to write annotations back to .minitrace.json files.
  - Use "annotate sync --dry-run" to preview changes before writing.

Profiles:
  - Controlled: requires scenario_id, condition, outcome on the session.
  - Organic: all annotation fields are optional.
`,
	}
	root.PersistentFlags().String(flagOutputDir, "", "Output directory containing .minitrace.json files (default: ./output)")

	root.AddCommand(newAddCommand())
	root.AddCommand(newListCommand())
	root.AddCommand(newEditCommand())
	root.AddCommand(newDeleteCommand())
	root.AddCommand(newSyncCommand())
	root.AddCommand(newImportCommand())

	return root, nil
}
