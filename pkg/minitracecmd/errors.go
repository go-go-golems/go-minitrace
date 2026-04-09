package minitracecmd

import "errors"

var (
	ErrUnknownCommandKind    = errors.New("unknown minitrace command kind")
	ErrMissingName           = errors.New("minitrace command spec is missing name")
	ErrMissingShort          = errors.New("minitrace command spec is missing short description")
	ErrMissingQuery          = errors.New("minitrace command spec is missing query body")
	ErrMissingAliasTarget    = errors.New("minitrace command alias is missing alias target")
	ErrVerbCannotSetAliasFor = errors.New("minitrace command verb cannot set aliasFor")
	ErrAliasCannotSetQuery   = errors.New("minitrace command alias cannot set query")
	ErrNilSpec               = errors.New("minitrace command spec is nil")
	ErrAliasTargetNotFound   = errors.New("minitrace alias target not found")

	ErrMissingPreamble       = errors.New("missing sqleton sql preamble")
	ErrUnterminatedPreamble  = errors.New("unterminated sqleton sql preamble")
	ErrInvalidPreambleMarker = errors.New("invalid sqleton sql preamble marker")
	ErrEmptyPreambleMetadata = errors.New("empty sqleton sql preamble metadata")
)
