package query

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand(flagPaths []string) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "query",
		Short: "Query converted minitrace archives",
		Long: `Query converted minitrace archives through the normalized SQLite engine.

query run builds a cached, sandboxed database from minitrace JSON archives and
runs either named presets or ad hoc SQL against the normalized schema
(sessions, turns, tool_calls, ... plus the sessions_base compatibility view).
query commands exposes .sql and .js query-command files as typed CLI verbs.

Note: the legacy DuckDB backend (query duckdb) has been removed; use
query run instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if args[0] == "duckdb" {
				return fmt.Errorf("unknown command %q for %q: the legacy DuckDB backend was removed; use %q instead",
					args[0], cmd.CommandPath(), cmd.CommandPath()+" run")
			}
			return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
		},
	}

	runCmd, err := NewRunCommand()
	if err != nil {
		return nil, err
	}
	commandsCmd, err := NewCommandsCommand(flagPaths)
	if err != nil {
		return nil, err
	}

	root.AddCommand(runCmd)
	root.AddCommand(commandsCmd)
	return root, nil
}
