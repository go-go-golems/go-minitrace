package minitracecmd

import (
	"errors"
	"testing"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
)

func TestParseSQLCommandSpec_ValidFile(t *testing.T) {
	contents := []byte(`/* sqleton
name: session-list
short: List sessions
flags:
  - name: framework
    type: string
    help: Filter by framework
  - name: verbose
    type: bool
    help: Include extra details
*/
SELECT *
FROM sessions_base
WHERE framework = {{ .framework | sqlString }}
`)

	spec, err := ParseSQLCommandSpec("queries/session-list.sql", contents)
	if err != nil {
		t.Fatalf("ParseSQLCommandSpec returned error: %v", err)
	}

	if spec.Kind != MinitraceCommandVerb {
		t.Fatalf("Kind = %q, want %q", spec.Kind, MinitraceCommandVerb)
	}
	if spec.Name != "session-list" {
		t.Fatalf("Name = %q, want session-list", spec.Name)
	}
	if spec.Short != "List sessions" {
		t.Fatalf("Short = %q, want List sessions", spec.Short)
	}
	if spec.Query == "" {
		t.Fatalf("Query should not be empty")
	}
	if len(spec.Flags) != 2 {
		t.Fatalf("len(Flags) = %d, want 2", len(spec.Flags))
	}
	if spec.Flags[0].Name != "framework" {
		t.Fatalf("Flags[0].Name = %q, want framework", spec.Flags[0].Name)
	}
	if spec.Flags[0].Type != fields.TypeString {
		t.Fatalf("Flags[0].Type = %q, want %q", spec.Flags[0].Type, fields.TypeString)
	}
	if spec.Flags[1].Type != fields.TypeBool {
		t.Fatalf("Flags[1].Type = %q, want %q", spec.Flags[1].Type, fields.TypeBool)
	}
	if !LooksLikeSqletonSQLCommand(contents) {
		t.Fatalf("LooksLikeSqletonSQLCommand returned false for a valid command")
	}
}

func TestParseSQLCommandSpec_MissingPreamble(t *testing.T) {
	_, err := ParseSQLCommandSpec("queries/session-list.sql", []byte("SELECT 1"))
	if !errors.Is(err, ErrMissingPreamble) {
		t.Fatalf("error = %v, want ErrMissingPreamble", err)
	}
}

func TestParseSQLCommandSpec_UnterminatedPreamble(t *testing.T) {
	contents := []byte("/* sqleton\nname: broken\nshort: Broken\nSELECT 1")
	_, err := ParseSQLCommandSpec("queries/broken.sql", contents)
	if !errors.Is(err, ErrUnterminatedPreamble) {
		t.Fatalf("error = %v, want ErrUnterminatedPreamble", err)
	}
}

func TestParseSQLCommandSpec_InvalidMarker(t *testing.T) {
	contents := []byte("/* not-sqleton\nname: broken\nshort: Broken\n*/\nSELECT 1")
	_, err := ParseSQLCommandSpec("queries/broken.sql", contents)
	if !errors.Is(err, ErrInvalidPreambleMarker) {
		t.Fatalf("error = %v, want ErrInvalidPreambleMarker", err)
	}
	if LooksLikeSqletonSQLCommand(contents) {
		t.Fatalf("LooksLikeSqletonSQLCommand returned true for invalid marker")
	}
}

func TestParseSQLCommandSpec_MissingShort(t *testing.T) {
	contents := []byte("/* sqleton\nname: session-list\n*/\nSELECT 1")
	_, err := ParseSQLCommandSpec("queries/session-list.sql", contents)
	if !errors.Is(err, ErrMissingShort) {
		t.Fatalf("error = %v, want ErrMissingShort", err)
	}
}

func TestParseSQLCommandSpec_MissingQueryBody(t *testing.T) {
	contents := []byte("/* sqleton\nname: session-list\nshort: List sessions\n*/")
	_, err := ParseSQLCommandSpec("queries/session-list.sql", contents)
	if !errors.Is(err, ErrMissingQuery) {
		t.Fatalf("error = %v, want ErrMissingQuery", err)
	}
	if !LooksLikeSqletonSQLCommand(contents) {
		t.Fatalf("LooksLikeSqletonSQLCommand should still detect the sqleton marker")
	}
}
