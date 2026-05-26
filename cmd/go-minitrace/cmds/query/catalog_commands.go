package query

import (
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
)

// NewMinitraceCatalogCommands converts a compiled minitrace command catalog into
// Glazed commands. The returned descriptions preserve catalog folders as Glazed
// parents so callers that mount the commands into another CLI (for example an
// xgoja command provider) retain the same nested command shape as the native
// go-minitrace CLI.
func NewMinitraceCatalogCommands(catalog *minitracecmd.Catalog) ([]cmds.Command, error) {
	if catalog == nil {
		return nil, nil
	}
	commands := make([]cmds.Command, 0, len(catalog.Commands))
	for _, command := range catalog.Commands {
		glazeCommand, err := NewMinitraceCatalogGlazeCommand(command, catalog)
		if err != nil {
			return nil, err
		}
		if desc := glazeCommand.Description(); desc != nil {
			desc.Parents = commandParents(command.Folder)
		}
		commands = append(commands, glazeCommand)
	}
	return commands, nil
}

func commandParents(folder string) []string {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return nil
	}
	parts := strings.Split(folder, "/")
	parents := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			parents = append(parents, part)
		}
	}
	return parents
}
