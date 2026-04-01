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
	piCmd, err := NewPiCommand()
	if err != nil {
		return nil, err
	}
	claudeAICmd, err := NewClaudeAICommand()
	if err != nil {
		return nil, err
	}
	chatGPTCmd, err := NewChatGPTCommand()
	if err != nil {
		return nil, err
	}
	chatGPTJSONCmd, err := NewChatGPTJSONCommand()
	if err != nil {
		return nil, err
	}
	turnsDBCmd, err := NewTurnsDBCommand()
	if err != nil {
		return nil, err
	}

	root.AddCommand(claudeCmd, codexCmd, piCmd, claudeAICmd, chatGPTCmd, chatGPTJSONCmd, turnsDBCmd)
	return root, nil
}
