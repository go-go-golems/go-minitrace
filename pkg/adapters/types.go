package adapters

type SessionLocator struct {
	ID             string
	FormatHint     string
	SourcePath     string
	Cwd            string          `json:"cwd,omitempty"`
	StartedAt      string          `json:"started_at,omitempty"`
	LastActivityAt string          `json:"last_activity_at,omitempty"`
	Identity       *SourceIdentity `json:"identity,omitempty"`
}
