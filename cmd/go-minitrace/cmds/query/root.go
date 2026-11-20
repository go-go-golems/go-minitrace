package query

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
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
query catalog reports their registered paths and scanner diagnostics as JSON.
query schema lists normalized tables or the columns of one named table.

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
	catalog, err := minitracecmd.LoadConfiguredCatalog("go-minitrace", flagPaths)
	if err != nil {
		return nil, err
	}
	commandsCmd, err := newCommandsCommand(flagPaths, catalog)
	if err != nil {
		return nil, err
	}

	root.AddCommand(runCmd)
	catalogCmd := newCatalogCommand(catalog)
	// Catalog discovery and command execution use the same repository selection.
	catalogCmd.Flags().AddFlag(commandsCmd.PersistentFlags().Lookup(minitracecmd.QueryRepositoryFlagName))
	root.AddCommand(commandsCmd, catalogCmd, newSchemaCommand())
	return root, nil
}

func newCatalogCommand(catalog *minitracecmd.Catalog) *cobra.Command {
	type entry struct {
		Path       []string `json:"path"`
		SourceRoot string   `json:"sourceRoot"`
		SourcePath string   `json:"sourcePath"`
	}
	return &cobra.Command{
		Use: "catalog [source-file]", Short: "List registered command paths and scanner diagnostics as JSON",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := []entry{}
			diagnostics := []minitracecmd.CatalogDiagnostic{}
			for _, definition := range catalog.Commands {
				if len(args) != 0 && definition.SourcePath != args[0] {
					continue
				}
				path := []string{definition.Name}
				if definition.Folder != "" {
					path = append(strings.Split(definition.Folder, "/"), definition.Name)
				}
				entries = append(entries, entry{path, definition.SourceRoot, definition.SourcePath})
			}
			for _, diagnostic := range catalog.Diagnostics {
				if len(args) == 0 || diagnostic.SourcePath == args[0] {
					diagnostics = append(diagnostics, diagnostic)
				}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"commands": entries, "diagnostics": diagnostics})
		},
	}
}

func newSchemaCommand() *cobra.Command {
	return &cobra.Command{
		Use: "schema [table]", Short: "List normalized tables or one table's columns as JSON",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			if len(args) == 0 {
				return encoder.Encode(map[string]any{"tables": minitracedb.AllowedTableNames()})
			}
			for _, table := range minitracedb.Tables() {
				if table.Name == args[0] {
					return encoder.Encode(map[string]any{"table": table.Name, "columns": table.Columns})
				}
			}
			return fmt.Errorf("unknown normalized table %q; available tables: %s", args[0], strings.Join(minitracedb.AllowedTableNames(), ", "))
		},
	}
}
