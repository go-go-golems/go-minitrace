package minitracecmd

import (
	"strings"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
)

type MinitraceCommandKind string

const (
	MinitraceCommandVerb  MinitraceCommandKind = "verb"
	MinitraceCommandAlias MinitraceCommandKind = "alias"
)

type MinitraceCommandSpec struct {
	Name      string               `yaml:"name"`
	Short     string               `yaml:"short"`
	Long      string               `yaml:"long,omitempty"`
	Layout    []*layout.Section    `yaml:"layout,omitempty"`
	Flags     []*fields.Definition `yaml:"flags,omitempty"`
	Arguments []*fields.Definition `yaml:"arguments,omitempty"`
	Tags      []string             `yaml:"tags,omitempty"`
	Metadata  map[string]any       `yaml:"metadata,omitempty"`

	Query      string               `yaml:"-"`
	AliasFor   string               `yaml:"aliasFor,omitempty"`
	AliasFlags map[string]any       `yaml:"-"`
	Kind       MinitraceCommandKind `yaml:"-"`
}

type MinitraceCommand struct {
	Name      string
	Folder    string
	Path      string
	Short     string
	Long      string
	Layout    []*layout.Section
	Flags     []*fields.Definition
	Arguments []*fields.Definition
	Tags      []string
	Metadata  map[string]any

	Query      string
	AliasFor   string
	AliasFlags map[string]any
	Kind       MinitraceCommandKind
	Readonly   bool
	SourceRoot string
	SourcePath string
}

func (s *MinitraceCommandSpec) Validate() error {
	if s == nil {
		return ErrNilSpec
	}

	name := strings.TrimSpace(s.Name)
	short := strings.TrimSpace(s.Short)
	query := strings.TrimSpace(s.Query)
	aliasFor := strings.TrimSpace(s.AliasFor)

	switch s.Kind {
	case MinitraceCommandVerb:
		if name == "" {
			return ErrMissingName
		}
		if short == "" {
			return ErrMissingShort
		}
		if query == "" {
			return ErrMissingQuery
		}
		if aliasFor != "" {
			return ErrVerbCannotSetAliasFor
		}
	case MinitraceCommandAlias:
		if name == "" {
			return ErrMissingName
		}
		if aliasFor == "" {
			return ErrMissingAliasTarget
		}
		if query != "" {
			return ErrAliasCannotSetQuery
		}
	default:
		return ErrUnknownCommandKind
	}

	return nil
}
