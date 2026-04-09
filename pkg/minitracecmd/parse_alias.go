package minitracecmd

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type aliasYAML struct {
	Name     string         `yaml:"name"`
	Short    string         `yaml:"short,omitempty"`
	AliasFor string         `yaml:"aliasFor"`
	Flags    map[string]any `yaml:"flags,omitempty"`
}

func ParseAliasSpec(path string, contents []byte) (*MinitraceCommandSpec, error) {
	payload := &aliasYAML{}
	if err := yaml.Unmarshal(contents, payload); err != nil {
		return nil, errors.Wrapf(err, "decode alias yaml: %s", path)
	}

	spec := &MinitraceCommandSpec{
		Name:       strings.TrimSpace(payload.Name),
		Short:      strings.TrimSpace(payload.Short),
		AliasFor:   strings.TrimSpace(payload.AliasFor),
		AliasFlags: payload.Flags,
		Kind:       MinitraceCommandAlias,
	}
	if err := spec.Validate(); err != nil {
		return nil, errors.Wrapf(err, "validate alias yaml: %s", path)
	}

	return spec, nil
}

func ParseAliasSpecFromReader(path string, r io.Reader) (*MinitraceCommandSpec, error) {
	contents, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrapf(err, "read alias yaml: %s", path)
	}
	return ParseAliasSpec(path, contents)
}
