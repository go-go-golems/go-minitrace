package convert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

type conversionRunRecord struct {
	Schema          string                    `json:"schema"`
	Adapter         string                    `json:"adapter"`
	StartedAt       string                    `json:"started_at"`
	FinishedAt      string                    `json:"finished_at"`
	OutputDir       string                    `json:"output_dir"`
	CollisionPolicy minitrace.CollisionPolicy `json:"collision_policy"`
	Inputs          []adapters.SourceIdentity `json:"inputs"`
	Outputs         []conversionRunOutput     `json:"outputs"`
	Complete        bool                      `json:"complete"`
}

type conversionRunOutput struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
}

func writeConversionRunRecord(path string, record conversionRunRecord) error {
	if path == "" {
		return nil
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshaling conversion run record")
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err, "creating conversion run record directory")
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".conversion-run-*.tmp")
	if err != nil {
		return errors.Wrap(err, "creating conversion run record")
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err := temporary.Write(payload); err != nil {
		_ = temporary.Close()
		return errors.Wrap(err, "writing conversion run record")
	}
	if err := temporary.Close(); err != nil {
		return errors.Wrap(err, "closing conversion run record")
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return errors.Wrap(err, "publishing conversion run record")
	}
	return nil
}

func newConversionRunRecord(adapter, outputDir string, policy minitrace.CollisionPolicy, locators []adapters.SessionLocator) conversionRunRecord {
	inputs := make([]adapters.SourceIdentity, 0, len(locators))
	for _, locator := range locators {
		if locator.Identity != nil {
			inputs = append(inputs, *locator.Identity)
		}
	}
	return conversionRunRecord{Schema: "go-minitrace-conversion-run-v1", Adapter: adapter, StartedAt: minitrace.FormatTimestamp(time.Now().UTC()), OutputDir: outputDir, CollisionPolicy: policy, Inputs: inputs, Outputs: []conversionRunOutput{}}
}
