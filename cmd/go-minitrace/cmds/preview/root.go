package preview

import (
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/common"
	"github.com/spf13/cobra"
)

func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "preview",
		Short: "Preview normalized sessions without writing an archive",
		Long: `Preview supported native agent sessions after minitrace auto-detection and
conversion. Preview commands are intended for quick validation before importing,
saving, rendering, or querying sessions.`,
	}

	sessionCommand, err := NewSessionCommand()
	if err != nil {
		return nil, err
	}
	sessionCobra, err := common.BuildCobraCommand(sessionCommand)
	if err != nil {
		return nil, err
	}
	root.AddCommand(sessionCobra)
	return root, nil
}
