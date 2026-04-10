package common

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/spf13/cobra"
)

func BuildCobraCommand(command cmds.Command) (*cobra.Command, error) {
	return BuildCobraCommandWithShortHelpSections(command, schema.DefaultSlug)
}

func BuildCobraCommandWithShortHelpSections(command cmds.Command, shortHelpSections ...string) (*cobra.Command, error) {
	return cli.BuildCobraCommandFromCommand(command,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: shortHelpSections,
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
}
