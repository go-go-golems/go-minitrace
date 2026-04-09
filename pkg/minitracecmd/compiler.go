package minitracecmd

import (
	"path/filepath"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
)

type CompileOptions struct {
	Folder     string
	Path       string
	SourceRoot string
	SourcePath string
	Readonly   bool
}

type Compiler struct{}

func (c *Compiler) Compile(spec *MinitraceCommandSpec, opts CompileOptions) (*MinitraceCommand, error) {
	if spec == nil {
		return nil, ErrNilSpec
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	return &MinitraceCommand{
		Name:       spec.Name,
		Folder:     filepath.ToSlash(opts.Folder),
		Path:       filepath.ToSlash(opts.Path),
		Short:      spec.Short,
		Long:       spec.Long,
		Layout:     spec.Layout,
		Flags:      normalizeOptionalBoolFlags(spec.Flags),
		Arguments:  spec.Arguments,
		Tags:       spec.Tags,
		Metadata:   spec.Metadata,
		Query:      spec.Query,
		AliasFor:   spec.AliasFor,
		AliasFlags: spec.AliasFlags,
		Kind:       spec.Kind,
		Readonly:   opts.Readonly,
		SourceRoot: opts.SourceRoot,
		SourcePath: opts.SourcePath,
	}, nil
}

func normalizeOptionalBoolFlags(flags []*fields.Definition) []*fields.Definition {
	if len(flags) == 0 {
		return nil
	}

	ret := make([]*fields.Definition, 0, len(flags))
	for _, flag := range flags {
		if flag == nil {
			ret = append(ret, nil)
			continue
		}

		cloned := flag.Clone()
		if cloned.Type == fields.TypeBool && !cloned.Required && cloned.Default == nil {
			defaultValue := any(false)
			cloned.Default = &defaultValue
		}
		ret = append(ret, cloned)
	}
	return ret
}
