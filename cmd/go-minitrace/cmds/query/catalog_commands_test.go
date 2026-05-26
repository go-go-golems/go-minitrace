package query

import (
	"errors"
	"testing"

	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
)

func TestNewMinitraceCatalogCommandsRejectsLeafGroupCollision(t *testing.T) {
	catalog := &minitracecmd.Catalog{Commands: []*minitracecmd.MinitraceCommand{
		{Name: "session-tools", Folder: "overview", Runtime: minitracecmd.CommandRuntimeSQL, Query: "select 1"},
		{Name: "session-list", Folder: "overview/session-tools", Runtime: minitracecmd.CommandRuntimeSQL, Query: "select 1"},
	}}

	_, err := NewMinitraceCatalogCommands(catalog)
	if err == nil {
		t.Fatalf("NewMinitraceCatalogCommands returned nil error, want command tree collision")
	}
	if !errors.Is(err, minitracecmd.ErrCommandTreeCollision) {
		t.Fatalf("error = %v, want ErrCommandTreeCollision", err)
	}
}
