package minitracecmd

import (
	"fmt"
	"strings"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

type ParsedCommandSpec struct {
	Spec *MinitraceCommandSpec
	Path string
}

func ParseJSCommandSpecs(path string, contents []byte) ([]ParsedCommandSpec, error) {
	registry, err := jsverbs.ScanSources([]jsverbs.SourceFile{{
		Path:   path,
		Source: contents,
	}})
	if err != nil {
		return nil, fmt.Errorf("scan js command %s: %w", path, err)
	}

	verbs := registry.Verbs()
	if len(verbs) == 0 {
		return nil, nil
	}

	ret := make([]ParsedCommandSpec, 0, len(verbs))
	for _, verb := range verbs {
		if verb == nil {
			continue
		}
		description, err := registry.CommandDescriptionForVerb(verb)
		if err != nil {
			return nil, fmt.Errorf("build js command description %s#%s: %w", path, verb.FunctionName, err)
		}

		flags, arguments := extractDefinitionsFromSchema(description.Schema)
		spec := &MinitraceCommandSpec{
			Name:      description.Name,
			Short:     description.Short,
			Long:      description.Long,
			Flags:     flags,
			Arguments: arguments,
			Tags:      append([]string(nil), description.Tags...),
			Metadata:  cloneMetadata(description.Metadata),
			Schema:    cloneSchema(description.Schema),
			Kind:      MinitraceCommandVerb,
			Runtime:   CommandRuntimeJS,
			JS: &JSCommandSpec{
				ModulePath:   verb.File.ModulePath,
				FunctionName: verb.FunctionName,
				OutputMode:   verb.OutputMode,
			},
		}
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("validate js command %s#%s: %w", path, verb.FunctionName, err)
		}

		ret = append(ret, ParsedCommandSpec{
			Spec: spec,
			Path: jsCommandPath(path, description.Name),
		})
	}

	return ret, nil
}

func jsCommandPath(sourcePath, commandName string) string {
	sourcePath = strings.TrimSpace(sourcePath)
	commandName = strings.TrimSpace(commandName)
	if sourcePath == "" {
		return commandName
	}
	if commandName == "" {
		return sourcePath
	}
	return sourcePath + ":" + commandName
}

func extractDefinitionsFromSchema(s *schema.Schema) ([]*fields.Definition, []*fields.Definition) {
	if s == nil {
		return nil, nil
	}

	flags := []*fields.Definition{}
	arguments := []*fields.Definition{}
	seen := map[string]struct{}{}

	s.ForEach(func(_ string, section schema.Section) {
		if section == nil {
			return
		}
		section.GetDefinitions().ForEach(func(def *fields.Definition) {
			if def == nil {
				return
			}
			if _, ok := seen[def.Name]; ok {
				return
			}
			seen[def.Name] = struct{}{}
			cloned := def.Clone()
			if cloned.IsArgument {
				arguments = append(arguments, cloned)
				return
			}
			flags = append(flags, cloned)
		})
	})

	return flags, arguments
}

func cloneSchema(s *schema.Schema) *schema.Schema {
	if s == nil {
		return nil
	}
	return s.Clone()
}

func cloneMetadata(m map[string]interface{}) map[string]any {
	if len(m) == 0 {
		return nil
	}
	ret := make(map[string]any, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}
