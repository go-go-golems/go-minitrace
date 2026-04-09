package query

import "github.com/spf13/cobra"

func NewCommand(flagPaths []string) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "query",
		Short: "Query converted minitrace archives",
		Long: `Query converted minitrace archives through built-in analysis backends.

The first backend is DuckDB, which loads minitrace JSON archives and runs
either named presets or ad hoc SQL.`,
	}

	duckDBCmd, err := NewDuckDBCommand()
	if err != nil {
		return nil, err
	}
	commandsCmd, err := NewCommandsCommand(flagPaths)
	if err != nil {
		return nil, err
	}

	root.AddCommand(duckDBCmd)
	root.AddCommand(commandsCmd)
	return root, nil
}
