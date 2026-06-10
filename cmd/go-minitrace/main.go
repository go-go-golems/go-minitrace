package main

import (
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	helpcmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/annotate"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/discover"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/preview"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/query"
	"github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/serve"
	validatecmd "github.com/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/validate"
	"github.com/go-go-golems/go-minitrace/pkg/doc"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "go-minitrace",
		Short:   "Glazed-based Go port of minitrace",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}

	cobra.CheckErr(logging.AddLoggingSectionToRootCommand(rootCmd, "go-minitrace"))

	helpSystem := help.NewHelpSystem()
	cobra.CheckErr(doc.AddDocToHelpSystem(helpSystem))
	helpcmd.SetupCobraRootCommand(helpSystem, rootCmd)

	discoverCmd, err := discover.NewCommand()
	cobra.CheckErr(err)
	convertCmd, err := convert.NewCommand()
	cobra.CheckErr(err)
	queryRepositoryFlags := minitracecmd.ExtractRepositoryFlagValuesFromArgs(os.Args[1:])
	queryCmd, err := query.NewCommand(queryRepositoryFlags)
	cobra.CheckErr(err)
	previewCmd, err := preview.NewCommand()
	cobra.CheckErr(err)
	serveCmd, err := serve.NewCommand()
	cobra.CheckErr(err)
	validateCommand, err := validatecmd.NewCommand()
	cobra.CheckErr(err)
	annotateCmd, err := annotate.NewCommand()
	cobra.CheckErr(err)

	rootCmd.AddCommand(discoverCmd, convertCmd, queryCmd, previewCmd, serveCmd, validateCommand, annotateCmd)

	cobra.CheckErr(rootCmd.Execute())
}
