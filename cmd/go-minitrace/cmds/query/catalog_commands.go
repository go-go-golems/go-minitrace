package query

import (
	"fmt"
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
	if err := validateCatalogCommandTree(catalog.Commands); err != nil {
		return nil, err
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

func validateCatalogCommandTree(commands []*minitracecmd.MinitraceCommand) error {
	paths := map[string]string{}
	for _, command := range commands {
		if command == nil {
			continue
		}
		parents := commandParents(command.Folder)
		current := ""
		for _, parent := range parents {
			if current == "" {
				current = parent
			} else {
				current += "/" + parent
			}
			if existing := paths[current]; existing == "leaf" {
				return fmt.Errorf("%w: folder group %q collides with existing command", minitracecmd.ErrCommandTreeCollision, current)
			}
			paths[current] = "group"
		}

		name := strings.TrimSpace(command.Name)
		if name == "" {
			continue
		}
		leafPath := name
		if current != "" {
			leafPath = current + "/" + name
		}
		if existing := paths[leafPath]; existing != "" {
			return fmt.Errorf("%w: command %q collides with existing %s", minitracecmd.ErrCommandTreeCollision, leafPath, existing)
		}
		paths[leafPath] = "leaf"
	}
	return nil
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
