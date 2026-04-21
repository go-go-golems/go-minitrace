package minitracecmd

import "errors"

var (
	ErrUnknownCommandKind      = errors.New("unknown minitrace command kind")
	ErrMissingName             = errors.New("minitrace command spec is missing name")
	ErrMissingShort            = errors.New("minitrace command spec is missing short description")
	ErrMissingQuery            = errors.New("minitrace command spec is missing query body")
	ErrMissingAliasTarget      = errors.New("minitrace command alias is missing alias target")
	ErrMissingRuntime          = errors.New("minitrace command verb is missing runtime configuration")
	ErrMultipleRuntimes        = errors.New("minitrace command verb cannot set multiple runtime payloads")
	ErrVerbCannotSetAliasFor   = errors.New("minitrace command verb cannot set aliasFor")
	ErrAliasCannotSetQuery     = errors.New("minitrace command alias cannot set query")
	ErrAliasCannotSetJS        = errors.New("minitrace command alias cannot set js runtime")
	ErrDuplicateCommandPath    = errors.New("duplicate minitrace command path")
	ErrMissingJSModulePath     = errors.New("minitrace js command spec is missing module path")
	ErrMissingJSFunctionName   = errors.New("minitrace js command spec is missing function name")
	ErrUnsupportedJSOutputMode = errors.New("unsupported minitrace js command output mode")
	ErrNilSpec                 = errors.New("minitrace command spec is nil")
	ErrNilCommand              = errors.New("minitrace command is nil")
	ErrAliasTargetNotFound     = errors.New("minitrace alias target not found")
	ErrCannotRenderAlias       = errors.New("minitrace alias command cannot be rendered directly")
	ErrInvalidTableName        = errors.New("invalid minitrace render table name")
	ErrCommandTreeCollision    = errors.New("minitrace query command tree collision")

	ErrMissingPreamble       = errors.New("missing sqleton sql preamble")
	ErrUnterminatedPreamble  = errors.New("unterminated sqleton sql preamble")
	ErrInvalidPreambleMarker = errors.New("invalid sqleton sql preamble marker")
	ErrEmptyPreambleMetadata = errors.New("empty sqleton sql preamble metadata")
)
