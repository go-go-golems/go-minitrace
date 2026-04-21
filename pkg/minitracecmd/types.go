package minitracecmd

import (
	"strings"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

type MinitraceCommandKind string

const (
	MinitraceCommandVerb  MinitraceCommandKind = "verb"
	MinitraceCommandAlias MinitraceCommandKind = "alias"
)

type CommandRuntimeKind string

const (
	CommandRuntimeUnknown CommandRuntimeKind = ""
	CommandRuntimeSQL     CommandRuntimeKind = "sql"
	CommandRuntimeJS      CommandRuntimeKind = "js"
)

type JSCommandSpec struct {
	ModulePath   string `yaml:"-"`
	FunctionName string `yaml:"-"`
	OutputMode   string `yaml:"-"`
}

type MinitraceCommandSpec struct {
	Name      string               `yaml:"name"`
	Short     string               `yaml:"short"`
	Long      string               `yaml:"long,omitempty"`
	Layout    []*layout.Section    `yaml:"layout,omitempty"`
	Flags     []*fields.Definition `yaml:"flags,omitempty"`
	Arguments []*fields.Definition `yaml:"arguments,omitempty"`
	Tags      []string             `yaml:"tags,omitempty"`
	Metadata  map[string]any       `yaml:"metadata,omitempty"`
	Schema    *schema.Schema       `yaml:"-"`

	Runtime    CommandRuntimeKind   `yaml:"-"`
	Query      string               `yaml:"-"`
	JS         *JSCommandSpec       `yaml:"-"`
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
	Schema    *schema.Schema

	Runtime    CommandRuntimeKind
	Query      string
	JS         *JSCommandSpec
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
	jsConfigured := s.JS != nil
	queryConfigured := query != ""

	switch s.Kind {
	case MinitraceCommandVerb:
		if name == "" {
			return ErrMissingName
		}
		if short == "" {
			return ErrMissingShort
		}
		if aliasFor != "" {
			return ErrVerbCannotSetAliasFor
		}
		if !queryConfigured && !jsConfigured {
			return ErrMissingRuntime
		}
		if queryConfigured && jsConfigured {
			return ErrMultipleRuntimes
		}
		switch s.Runtime {
		case CommandRuntimeUnknown:
			if queryConfigured {
				s.Runtime = CommandRuntimeSQL
			} else {
				s.Runtime = CommandRuntimeJS
			}
		case CommandRuntimeSQL:
			if !queryConfigured {
				return ErrMissingQuery
			}
		case CommandRuntimeJS:
			if !jsConfigured {
				return ErrMissingRuntime
			}
			if strings.TrimSpace(s.JS.ModulePath) == "" {
				return ErrMissingJSModulePath
			}
			if strings.TrimSpace(s.JS.FunctionName) == "" {
				return ErrMissingJSFunctionName
			}
			switch strings.TrimSpace(s.JS.OutputMode) {
			case "", jsverbs.OutputModeGlaze, jsverbs.OutputModeText:
			default:
				return ErrUnsupportedJSOutputMode
			}
		default:
			return ErrMissingRuntime
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
		if jsConfigured {
			return ErrAliasCannotSetJS
		}
	default:
		return ErrUnknownCommandKind
	}

	return nil
}
