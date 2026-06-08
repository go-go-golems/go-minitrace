package minitracedb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/adapters/claudecode"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/codex"
	"github.com/go-go-golems/go-minitrace/pkg/adapters/pi"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type ConversionDiagnostic struct {
	Source     string `json:"source"`
	Severity   string `json:"severity"`
	Format     string `json:"format,omitempty"`
	Message    string `json:"message"`
	Adapter    string `json:"adapter,omitempty"`
	RecordRows int    `json:"recordRows,omitempty"`
}

type LoadOptions struct {
	SourceName  string
	SourcePath  string
	AutoConvert bool
}

type LoadedSession struct {
	Session     *minitrace.Session
	Format      string
	Adapter     string
	Diagnostics []ConversionDiagnostic
}

func LoadSessionFileAuto(path string, opts LoadOptions) (*LoadedSession, error) {
	payload, err := osReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source %s: %w", path, err)
	}
	if opts.SourcePath == "" {
		opts.SourcePath = path
	}
	if opts.SourceName == "" {
		opts.SourceName = filepath.Base(path)
	}
	return LoadSessionContentAuto(payload, opts)
}

func LoadSessionContentAuto(payload []byte, opts LoadOptions) (*LoadedSession, error) {
	source := firstNonEmpty(opts.SourceName, opts.SourcePath, "<content>")
	if session, err := decodeNativeSession(payload); err == nil {
		return &LoadedSession{Session: session, Format: "minitrace-json", Adapter: "native", Diagnostics: []ConversionDiagnostic{{Source: source, Severity: "info", Format: "minitrace-json", Adapter: "native", Message: "loaded native minitrace session"}}}, nil
	}
	if !opts.AutoConvert {
		return nil, fmt.Errorf("%s is not native minitrace JSON and AutoConvert is disabled", source)
	}
	records, err := parseJSONL(payload)
	if err != nil {
		return nil, fmt.Errorf("%s is not supported native JSON or JSONL: %w", source, err)
	}
	format := DetectJSONLFormat(records)
	fallbackID := strings.TrimSuffix(filepath.Base(firstNonEmpty(opts.SourcePath, opts.SourceName, "content")), filepath.Ext(firstNonEmpty(opts.SourcePath, opts.SourceName, "content")))
	if fallbackID == "" || fallbackID == "." {
		fallbackID = "content"
	}
	var (
		session *minitrace.Session
		adapter string
	)
	switch format {
	case "pi-jsonl":
		adapter = "pi"
		session, err = pi.ConvertRecords(records, fallbackID, opts.SourcePath)
	case "codex-jsonl":
		adapter = "codex"
		session, err = codex.ConvertRecords(records, fallbackID, opts.SourcePath, "unknown-jsonl")
	case "claude-code-jsonl":
		adapter = "claude-code"
		session, err = claudecode.ConvertRecords(records, fallbackID, opts.SourcePath)
	default:
		return nil, fmt.Errorf("%s JSONL format is unsupported", source)
	}
	if err != nil {
		return nil, fmt.Errorf("convert %s as %s: %w", source, format, err)
	}
	return &LoadedSession{Session: session, Format: format, Adapter: adapter, Diagnostics: []ConversionDiagnostic{{Source: source, Severity: "info", Format: format, Adapter: adapter, RecordRows: len(records), Message: "converted source into minitrace session"}}}, nil
}

func DetectJSONLFormat(records []map[string]any) string {
	for _, record := range records {
		t := stringValue(record["type"])
		if t == "session_meta" || t == "turn_context" || t == "response_item" || t == "event_msg" {
			return "codex-jsonl"
		}
		if t == "session" || t == "model_change" || t == "thinking_level_change" || t == "message" {
			return "pi-jsonl"
		}
		role := stringValue(record["role"])
		if role == "user" || role == "assistant" || role == "toolResult" {
			return "pi-jsonl"
		}
		if (t == "system" || t == "user" || t == "assistant") && record["message"] != nil {
			return "claude-code-jsonl"
		}
	}
	return "unknown-jsonl"
}

func decodeNativeSession(payload []byte) (*minitrace.Session, error) {
	var session minitrace.Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, err
	}
	if session.ID == "" {
		return nil, fmt.Errorf("missing session id")
	}
	if session.SchemaVersion == "" && session.Profile == "" && session.Provenance.SourceFormat == "" && len(session.Turns) == 0 && len(session.ToolCalls) == 0 {
		return nil, fmt.Errorf("not a native minitrace session document")
	}
	return &session, nil
}

func parseJSONL(payload []byte) ([]map[string]any, error) {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	records := []map[string]any{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty JSONL")
	}
	return records, nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var osReadFile = func(path string) ([]byte, error) { return os.ReadFile(path) }
