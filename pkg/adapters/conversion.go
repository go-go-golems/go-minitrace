package adapters

import "github.com/go-go-golems/go-minitrace/pkg/minitrace"

type ConversionStatus string

const (
	ConversionCreated   ConversionStatus = "created"
	ConversionUnchanged ConversionStatus = "unchanged"
	ConversionReplaced  ConversionStatus = "replaced"
	ConversionFailed    ConversionStatus = "failed"
	ConversionSkipped   ConversionStatus = "skipped"
)

// ConversionResult is the adapter-neutral per-source result consumed by the
// future shared batch publisher and conversion receipts. Session is present
// only for successfully converted sources; Error is a stable human-readable
// diagnostic until a dedicated error-code taxonomy is introduced.
type ConversionResult struct {
	Source   SourceIdentity      `json:"source"`
	Session  *minitrace.Session  `json:"-"`
	Status   ConversionStatus    `json:"status"`
	Warnings []ConversionWarning `json:"warnings,omitempty"`
	Error    string              `json:"error,omitempty"`
}
