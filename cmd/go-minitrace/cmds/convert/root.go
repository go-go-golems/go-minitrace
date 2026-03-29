package convert

import "github.com/spf13/cobra"

func NewCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "convert",
		Short: "Convert supported native session stores into minitrace output",
		Long: `Bootstrap command group for conversion.

Claude Code and Codex are the first-class targets. The current skeleton exposes
the intended Glazed command surface and discovery-backed planning output while
the actual conversion engine is ported.`,
	}

	claudeCmd, err := NewClaudeCodeCommand()
	if err != nil {
		return nil, err
	}
	codexCmd, err := NewCodexCommand()
	if err != nil {
		return nil, err
	}

	root.AddCommand(claudeCmd, codexCmd)
	return root, nil
}
