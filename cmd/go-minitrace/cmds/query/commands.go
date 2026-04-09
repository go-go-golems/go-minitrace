package query

import (
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/spf13/cobra"
)

func NewCommandsCommand() (*cobra.Command, error) {
	catalog, err := minitracecmd.LoadEmbeddedCatalog()
	if err != nil {
		return nil, err
	}

	root := &cobra.Command{
		Use:   "commands",
		Short: "Run repository-backed MinitraceCommand queries",
		Long:  "Run repository-backed MinitraceCommand queries loaded from the embedded sqleton-style command catalog.",
	}

	for _, command := range catalog.Commands {
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			return nil, err
		}
		cobraCommand, err := common.BuildCobraCommand(glazeCommand)
		if err != nil {
			return nil, err
		}
		root.AddCommand(cobraCommand)
	}

	return root, nil
}
