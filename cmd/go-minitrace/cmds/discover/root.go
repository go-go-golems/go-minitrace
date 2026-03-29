package discover

import "github.com/spf13/cobra"

func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "discover",
		Short: "Inspect native session stores without converting them",
		Long: `Inspect native Claude Code and Codex session stores and emit structured
rows describing the sessions that would be considered for conversion.`,
	}

	claudeCmd, err := NewClaudeCodeCommand()
	if err != nil {
		return nil, err
	}
	codexCmd, err := NewCodexCommand()
	if err != nil {
		return nil, err
	}
	piCmd, err := NewPiCommand()
	if err != nil {
		return nil, err
	}

	root.AddCommand(claudeCmd, codexCmd, piCmd)
	return root, nil
}
