package query

import (
	"fmt"
	"strings"

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

Subdirectories in the repository are exposed as nested CLI groups. SQL files map directly to leaf commands, so a file like
pkg/minitracecmd/core/overview/session-list.sql becomes:
  go-minitrace query commands overview session-list

JS files add one more group level based on the file stem, so a file like
pkg/minitracecmd/core/overview/session-tools.js with a verb named session-list becomes:
  go-minitrace query commands overview session-tools session-list

The nightly review commands live under the nightly subverb, so a file like
pkg/minitracecmd/core/nightly/session-inventory.sql becomes:
  go-minitrace query commands nightly session-inventory

Additional repositories can be provided through:
  - app config: queryRepositories
  - environment: GO_MINITRACE_QUERY_REPOSITORIES
  - repeated CLI flags: --query-repository ./query-commands/team

Examples:
  go-minitrace query commands overview session-list --archive-glob './output/active/*/*.minitrace.json'
  go-minitrace query commands overview session-tools session-list --query-repository ./query-commands/team
  GO_MINITRACE_QUERY_REPOSITORIES=./query-commands/team go-minitrace query commands overview framework-summary`,
	}
	groups := map[string]*cobra.Command{}
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
		parent := ensureCommandGroup(root, groups, command.Folder)
		parent.AddCommand(cobraCommand)
	}

	return root, nil
}

func ensureCommandGroup(root *cobra.Command, groups map[string]*cobra.Command, folder string) *cobra.Command {
	if strings.TrimSpace(folder) == "" {
		return root
	}

	parent := root
	currentPath := ""
	for _, segment := range strings.Split(folder, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if currentPath == "" {
			currentPath = segment
		} else {
			currentPath += "/" + segment
		}
		if existing, ok := groups[currentPath]; ok {
			parent = existing
			continue
		}

		group := &cobra.Command{
			Use:   segment,
			Short: fmt.Sprintf("Query command group: %s", currentPath),
		}
		parent.AddCommand(group)
		groups[currentPath] = group
		parent = group
	}

	return parent
}
