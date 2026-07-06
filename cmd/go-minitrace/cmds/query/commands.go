package query

//glazedclilint:file-ignore legacy query command uses Cobra flags pending Glazed field migration

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

JS files usually add one more group level based on the file stem, so a file like
pkg/minitracecmd/core/overview/session-tools.js with a verb named session-list becomes:
  go-minitrace query commands overview session-tools session-list

When a JS file defines exactly one verb and that verb has the same name as the file stem, go-minitrace collapses the redundant extra level. For example:
  query-commands/hardware-research/research-summary.js
with a single verb named research-summary becomes:
  go-minitrace query commands hardware-research research-summary

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
		parent, err := ensureCommandGroup(root, groups, command.Folder)
		if err != nil {
			return nil, err
		}
		wrapArgumentParseErrors(cobraCommand, command)
		if err := addCommandChild(parent, cobraCommand, command.Folder); err != nil {
			return nil, err
		}
	}

	return root, nil
}

// wrapArgumentParseErrors rejects surplus positional arguments on a leaf
// query command with an error that names the resolved command path and the
// verbs available next to it. The most common trigger is typing the
// uncollapsed path of a JS command file (e.g. `... usage command-freq` when
// the verb collapsed into `... usage`), which otherwise surfaces as a bare
// "Too many arguments" printed from deep inside argument parsing.
func wrapArgumentParseErrors(cmd *cobra.Command, command *minitracecmd.MinitraceCommand) {
	if cmd == nil {
		return
	}
	maxArgs, unlimited := maxPositionalArgs(command)
	originalRun := cmd.Run
	originalRunE := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if !unlimited && len(args) > maxArgs {
			var b strings.Builder
			fmt.Fprintf(&b, "too many arguments: %q does not accept", c.CommandPath())
			if maxArgs == 0 {
				b.WriteString(" positional arguments")
			} else {
				fmt.Fprintf(&b, " more than %d positional argument(s)", maxArgs)
			}
			fmt.Fprintf(&b, "\n  extra arguments: %s", strings.Join(args[maxArgs:], " "))
			if siblings := siblingVerbNames(c); len(siblings) > 0 {
				parentPath := c.CommandPath()
				if parent := c.Parent(); parent != nil {
					parentPath = parent.CommandPath()
				}
				fmt.Fprintf(&b, "\n  available verbs under %q: %s", parentPath, strings.Join(siblings, ", "))
			}
			return fmt.Errorf("%s", b.String())
		}
		if originalRunE != nil {
			return originalRunE(c, args)
		}
		if originalRun != nil {
			originalRun(c, args)
		}
		return nil
	}
}

// maxPositionalArgs derives how many positional arguments a catalog command
// accepts. A list-typed argument consumes the remainder, so the count is
// unlimited in that case.
func maxPositionalArgs(command *minitracecmd.MinitraceCommand) (int, bool) {
	if command == nil {
		return 0, false
	}
	count := 0
	for _, definition := range command.Arguments {
		if definition == nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(definition.Type)), "list") {
			return count, true
		}
		count++
	}
	return count, false
}

func siblingVerbNames(cmd *cobra.Command) []string {
	parent := cmd.Parent()
	if parent == nil {
		return nil
	}
	names := make([]string, 0, len(parent.Commands()))
	for _, sibling := range parent.Commands() {
		if sibling == nil || sibling.Hidden {
			continue
		}
		names = append(names, sibling.Name())
	}
	return names
}

func ensureCommandGroup(root *cobra.Command, groups map[string]*cobra.Command, folder string) (*cobra.Command, error) {
	if strings.TrimSpace(folder) == "" {
		return root, nil
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
		if existing := findCommandChild(parent, segment); existing != nil {
			return nil, fmt.Errorf("%w: folder group %q collides with existing command %q", minitracecmd.ErrCommandTreeCollision, currentPath, existing.CommandPath())
		}

		group := &cobra.Command{
			Use:   segment,
			Short: fmt.Sprintf("Query command group: %s", currentPath),
		}
		parent.AddCommand(group)
		groups[currentPath] = group
		parent = group
	}

	return parent, nil
}

func addCommandChild(parent *cobra.Command, child *cobra.Command, folder string) error {
	if parent == nil || child == nil {
		return nil
	}
	if existing := findCommandChild(parent, child.Name()); existing != nil {
		return fmt.Errorf("%w: command %q collides with existing command %q under %q", minitracecmd.ErrCommandTreeCollision, child.CommandPath(), existing.CommandPath(), folder)
	}
	parent.AddCommand(child)
	return nil
}

func findCommandChild(parent *cobra.Command, use string) *cobra.Command {
	if parent == nil {
		return nil
	}
	for _, existing := range parent.Commands() {
		if existing == nil {
			continue
		}
		if strings.TrimSpace(existing.Name()) == strings.TrimSpace(use) {
			return existing
		}
	}
	return nil
}
