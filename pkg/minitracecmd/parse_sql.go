package minitracecmd

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func ParseSQLCommandSpec(path string, contents []byte) (*MinitraceCommandSpec, error) {
	metadataText, body, err := splitSqletonSQLPreamble(contents)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sqleton sql preamble: %s", path)
	}

	spec := &MinitraceCommandSpec{Kind: MinitraceCommandVerb}
	decoder := yaml.NewDecoder(strings.NewReader(metadataText))
	if err := decoder.Decode(spec); err != nil {
		return nil, errors.Wrapf(err, "decode sqleton sql metadata: %s", path)
	}

	spec.Query = strings.TrimSpace(body)
	if err := spec.Validate(); err != nil {
		return nil, errors.Wrapf(err, "validate sqleton sql command: %s", path)
	}

	return spec, nil
}

func ParseSQLCommandSpecFromReader(path string, r io.Reader) (*MinitraceCommandSpec, error) {
	contents, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrapf(err, "read sqleton sql command: %s", path)
	}
	return ParseSQLCommandSpec(path, contents)
}

func splitSqletonSQLPreamble(contents []byte) (string, string, error) {
	s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
	if !strings.HasPrefix(s, "/*") {
		return "", "", ErrMissingPreamble
	}

	end := strings.Index(s, "*/")
	if end == -1 {
		return "", "", ErrUnterminatedPreamble
	}

	raw := strings.TrimSpace(s[2:end])
	if !strings.HasPrefix(raw, "sqleton") {
		return "", "", ErrInvalidPreambleMarker
	}

	metadata := strings.TrimSpace(strings.TrimPrefix(raw, "sqleton"))
	body := strings.TrimSpace(s[end+2:])
	if metadata == "" {
		return "", "", ErrEmptyPreambleMetadata
	}
	if body == "" {
		return "", "", ErrMissingQuery
	}

	return metadata, body, nil
}

func LooksLikeSqletonSQLCommand(contents []byte) bool {
	s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
	if !strings.HasPrefix(s, "/*") {
		return false
	}

	end := strings.Index(s, "*/")
	if end == -1 {
		return false
	}

	raw := strings.TrimSpace(s[2:end])
	return strings.HasPrefix(raw, "sqleton")
}
