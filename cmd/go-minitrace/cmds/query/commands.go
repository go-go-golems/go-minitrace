package query

import (
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/spf13/cobra"
)

func NewCommandsCommand(flagPaths []string) (*cobra.Command, error) {
	catalog, err := minitracecmd.LoadConfiguredCatalog("go-minitrace", flagPaths)
	if err != nil {
		return nil, err
	}

	root := &cobra.Command{
		Use:   "commands",
		Short: "Run repository-backed MinitraceCommand queries",
		Long: `Run repository-backed MinitraceCommand queries loaded from the embedded sqleton-style catalog plus any configured external query repositories.

Additional repositories can be provided through:
  - app config: queryRepositories
  - environment: GO_MINITRACE_QUERY_REPOSITORIES
  - repeated CLI flags: --query-repository ./query-commands/team

Examples:
  go-minitrace query commands session-list --archive-glob './output/active/*/*.minitrace.json'
  go-minitrace query commands session-list --query-repository ./query-commands/team
  GO_MINITRACE_QUERY_REPOSITORIES=./query-commands/team go-minitrace query commands framework-summary`,
	}
	root.PersistentFlags().StringSlice(minitracecmd.QueryRepositoryFlagName, flagPaths, "Repeatable directory flag for additional structured query-command repository roots")

	for _, command := range catalog.Commands {
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			return nil, err
		}
		cobraCommand, err := common.BuildCobraCommandWithShortHelpSections(glazeCommand, "default", QueryRuntimeSectionSlug)
		if err != nil {
			return nil, err
		}
		root.AddCommand(cobraCommand)
	}

	return root, nil
}
