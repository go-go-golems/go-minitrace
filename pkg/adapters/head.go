package adapters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

const (
	// HeadMaxLines bounds how many leading JSONL records are inspected when
	// extracting cheap session header metadata during discovery.
	HeadMaxLines = 50
	// HeadMaxBytes bounds how much of a session file is read when extracting
	// cheap session header metadata during discovery.
	HeadMaxBytes = 256 * 1024
)

// ScanJSONLHead reads at most maxLines lines from the first maxBytes of the
// file at path, decodes each non-empty line as a JSON object, and calls fn for
// every successfully decoded record. Scanning stops early when fn returns
// true. Lines that fail to decode (including a line truncated by the byte
// cap) are skipped silently so discovery stays cheap and robust.
func ScanJSONLHead(path string, maxLines int, maxBytes int64, fn func(record map[string]any) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(io.LimitReader(f, maxBytes))
	scanner.Buffer(make([]byte, 0, 64*1024), int(maxBytes))
	for i := 0; i < maxLines && scanner.Scan(); i++ {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		if fn(record) {
			return nil
		}
	}
	return scanner.Err()
}
