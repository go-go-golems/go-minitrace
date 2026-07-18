package adapters

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/pkg/errors"
)

// ScanJSONLLastTimestamp streams a JSONL source and returns the latest valid
// timestamp supplied by timestampCandidates. Malformed JSON records are
// skipped so activity discovery has the same tolerance as native conversion.
func ScanJSONLLastTimestamp(path string, timestampCandidates func(map[string]any) []string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "opening JSONL activity source %s", path)
	}
	defer func() { _ = file.Close() }()

	var latest time.Time
	found := false
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		for _, candidate := range timestampCandidates(record) {
			timestamp, ok := minitrace.ParseTimestamp(candidate)
			if !ok || found && !timestamp.After(latest) {
				continue
			}
			latest = timestamp
			found = true
		}
	}
	if err := scanner.Err(); err != nil {
		return "", errors.Wrapf(err, "scanning JSONL activity source %s", path)
	}
	if !found {
		return "", nil
	}
	return latest.Format(time.RFC3339Nano), nil
}
