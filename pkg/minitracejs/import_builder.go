package minitracejs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

type ImportBuilder struct {
	content     string
	path        string
	name        string
	sourcePath  string
	rootDir     string
	sessionID   string
	strict      bool
	autoConvert bool
	format      string
	overwrite   bool
	converted   *minitracedb.LoadedSession
}

type SavedSession struct {
	SessionID    string                             `json:"sessionId"`
	Format       string                             `json:"format"`
	Adapter      string                             `json:"adapter,omitempty"`
	Original     string                             `json:"original,omitempty"`
	Title        string                             `json:"title,omitempty"`
	UploadedAt   time.Time                          `json:"uploadedAt"`
	SessionDir   string                             `json:"sessionDir"`
	SessionPath  string                             `json:"sessionPath"`
	MetadataPath string                             `json:"metadataPath"`
	Diagnostics  []minitracedb.ConversionDiagnostic `json:"diagnostics,omitempty"`
}

type ConvertedSession struct {
	SessionID   string                             `json:"sessionId"`
	Format      string                             `json:"format"`
	Adapter     string                             `json:"adapter,omitempty"`
	Title       string                             `json:"title,omitempty"`
	TurnCount   int                                `json:"turnCount"`
	ToolCount   int                                `json:"toolCallCount"`
	Diagnostics []minitracedb.ConversionDiagnostic `json:"diagnostics,omitempty"`
}

type SessionPreview struct {
	SessionID       string                             `json:"sessionId"`
	Format          string                             `json:"format"`
	Adapter         string                             `json:"adapter,omitempty"`
	Title           string                             `json:"title,omitempty"`
	AgentFramework  string                             `json:"agentFramework,omitempty"`
	Model           string                             `json:"model,omitempty"`
	WorkingDir      string                             `json:"workingDirectory,omitempty"`
	HasSystemPrompt bool                               `json:"hasSystemPrompt"`
	HasThinking     bool                               `json:"hasThinking"`
	HasImageSignals bool                               `json:"hasImageSignals"`
	TurnCount       int                                `json:"turnCount"`
	ToolCallCount   int                                `json:"toolCallCount"`
	SubagentCount   int                                `json:"subagentCount"`
	RoleCounts      map[string]int                     `json:"roleCounts"`
	ToolCounts      map[string]int                     `json:"toolCounts"`
	SampleTurns     []PreviewTurn                      `json:"sampleTurns,omitempty"`
	SampleTools     []PreviewToolCall                  `json:"sampleTools,omitempty"`
	Diagnostics     []minitracedb.ConversionDiagnostic `json:"diagnostics,omitempty"`
}

type PreviewTurn struct {
	Index       int      `json:"index"`
	Role        string   `json:"role"`
	Source      string   `json:"source,omitempty"`
	Model       string   `json:"model,omitempty"`
	ContentType string   `json:"contentType,omitempty"`
	HasContent  bool     `json:"hasContent"`
	HasThinking bool     `json:"hasThinking"`
	ToolCalls   []string `json:"toolCalls,omitempty"`
	Preview     string   `json:"preview,omitempty"`
}

type PreviewToolCall struct {
	ID                 string `json:"id"`
	TurnIndex          *int   `json:"turnIndex,omitempty"`
	ToolName           string `json:"toolName"`
	OperationType      string `json:"operationType"`
	FilePath           string `json:"filePath,omitempty"`
	Command            string `json:"command,omitempty"`
	Success            bool   `json:"success"`
	HasResult          bool   `json:"hasResult"`
	HasError           bool   `json:"hasError"`
	Truncated          bool   `json:"truncated"`
	SpawnedAgentType   string `json:"spawnedAgentType,omitempty"`
	SpawnedSubSession  string `json:"spawnedSubSession,omitempty"`
	SpawnedAgentScope  string `json:"spawnedAgentScope,omitempty"`
	OutputContentBytes int    `json:"outputContentBytes,omitempty"`
}

func NewImportBuilder() *ImportBuilder {
	return &ImportBuilder{strict: true, autoConvert: true}
}

func importBuilderObject(vm *goja.Runtime, b *ImportBuilder) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("Content", func(content string) *goja.Object {
		b.content = content
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("File", func(path string) *goja.Object {
		b.path = strings.TrimSpace(path)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("Name", func(name string) *goja.Object {
		b.name = strings.TrimSpace(name)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("SourcePath", func(path string) *goja.Object {
		b.sourcePath = strings.TrimSpace(path)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("AutoDetect", func() *goja.Object {
		b.autoConvert = true
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("Format", func(format string) *goja.Object {
		b.format = strings.TrimSpace(format)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("Strict", func(call goja.FunctionCall) goja.Value {
		b.strict = optionalBool(call, true)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("Into", func(rootDir string) *goja.Object {
		b.rootDir = strings.TrimSpace(rootDir)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("SessionID", func(id string) *goja.Object {
		b.sessionID = strings.TrimSpace(id)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("Overwrite", func(call goja.FunctionCall) goja.Value {
		b.overwrite = optionalBool(call, true)
		return importBuilderObject(vm, b)
	})
	_ = obj.Set("Detect", func() (map[string]any, error) {
		loaded, err := b.load()
		if err != nil {
			return nil, err
		}
		b.converted = loaded
		return map[string]any{"format": loaded.Format, "adapter": loaded.Adapter, "diagnostics": toPlainSlice(loaded.Diagnostics)}, nil
	})
	_ = obj.Set("Convert", func() (*goja.Object, error) {
		if _, err := b.Convert(); err != nil {
			return importBuilderObject(vm, b), err
		}
		return importBuilderObject(vm, b), nil
	})
	_ = obj.Set("Converted", func() (map[string]any, error) {
		converted, err := b.Converted()
		return toPlainMap(converted), err
	})
	_ = obj.Set("Preview", func() (map[string]any, error) {
		preview, err := b.Preview()
		return toPlainMap(preview), err
	})
	_ = obj.Set("Diagnostics", func() []map[string]any {
		if b.converted == nil {
			return nil
		}
		return toPlainSlice(b.converted.Diagnostics)
	})
	_ = obj.Set("Save", func() (map[string]any, error) {
		saved, err := b.Save()
		return toPlainMap(saved), err
	})
	return obj
}

func (b *ImportBuilder) Convert() (*ImportBuilder, error) {
	loaded, err := b.load()
	if err != nil {
		return b, err
	}
	b.converted = loaded
	return b, nil
}

func (b *ImportBuilder) Converted() (ConvertedSession, error) {
	if b.converted == nil {
		if _, err := b.Convert(); err != nil {
			return ConvertedSession{}, err
		}
	}
	session := b.converted.Session
	return ConvertedSession{SessionID: session.ID, Format: b.converted.Format, Adapter: b.converted.Adapter, Title: stringPtr(session.Title), TurnCount: len(session.Turns), ToolCount: len(session.ToolCalls), Diagnostics: b.converted.Diagnostics}, nil
}

func (b *ImportBuilder) Preview() (SessionPreview, error) {
	if b.converted == nil {
		if _, err := b.Convert(); err != nil {
			return SessionPreview{}, err
		}
	}
	session := b.converted.Session
	preview := SessionPreview{
		SessionID:       session.ID,
		Format:          b.converted.Format,
		Adapter:         b.converted.Adapter,
		Title:           stringPtr(session.Title),
		AgentFramework:  stringPtr(session.Environment.AgentFramework),
		Model:           stringPtr(session.Environment.Model),
		WorkingDir:      stringPtr(session.OperationalContext.WorkingDirectory),
		HasSystemPrompt: strings.TrimSpace(stringPtr(session.Environment.SystemPrompt)) != "",
		TurnCount:       len(session.Turns),
		ToolCallCount:   len(session.ToolCalls),
		SubagentCount:   session.Metrics.SubagentCount,
		RoleCounts:      map[string]int{},
		ToolCounts:      map[string]int{},
		Diagnostics:     b.converted.Diagnostics,
	}
	for _, turn := range session.Turns {
		preview.RoleCounts[turn.Role]++
		if strings.TrimSpace(stringPtr(turn.Thinking)) != "" {
			preview.HasThinking = true
		}
		if hasImageSignal(turn.ContentType, turn.Content, turn.FrameworkMetadata) {
			preview.HasImageSignals = true
		}
		if len(preview.SampleTurns) < 12 {
			preview.SampleTurns = append(preview.SampleTurns, PreviewTurn{
				Index:       turn.Index,
				Role:        turn.Role,
				Source:      stringPtr(turn.Source),
				Model:       stringPtr(turn.Model),
				ContentType: stringPtr(turn.ContentType),
				HasContent:  strings.TrimSpace(turn.Content) != "",
				HasThinking: strings.TrimSpace(stringPtr(turn.Thinking)) != "",
				ToolCalls:   append([]string(nil), turn.ToolCallsInTurn...),
				Preview:     truncatePreview(turn.Content, 240),
			})
		}
	}
	for _, toolCall := range session.ToolCalls {
		preview.ToolCounts[toolCall.ToolName]++
		if hasImageSignal(nil, toolCall.ToolName, toolCall.FrameworkMetadata) || hasImageSignal(toolCall.Output.ContentOrigin, stringPtr(toolCall.Output.Result), toolCall.Input.Arguments) {
			preview.HasImageSignals = true
		}
		if len(preview.SampleTools) < 12 {
			sample := PreviewToolCall{
				ID:                 toolCall.ID,
				TurnIndex:          toolCall.EmittingTurnIndex,
				ToolName:           toolCall.ToolName,
				OperationType:      toolCall.OperationType,
				FilePath:           stringPtr(toolCall.Input.FilePath),
				Command:            truncatePreview(stringPtr(toolCall.Input.Command), 240),
				Success:            toolCall.Output.Success,
				HasResult:          strings.TrimSpace(stringPtr(toolCall.Output.Result)) != "",
				HasError:           strings.TrimSpace(stringPtr(toolCall.Output.Error)) != "",
				Truncated:          toolCall.Output.Truncated,
				OutputContentBytes: outputContentBytes(toolCall.Output.Result, toolCall.Output.Error),
			}
			if toolCall.SpawnedAgent != nil {
				sample.SpawnedAgentType = toolCall.SpawnedAgent.AgentType
				sample.SpawnedAgentScope = toolCall.SpawnedAgent.TaskScope
				sample.SpawnedSubSession = stringPtr(toolCall.SpawnedAgent.SubSessionID)
			}
			preview.SampleTools = append(preview.SampleTools, sample)
		}
	}
	return preview, nil
}

func (b *ImportBuilder) Save() (SavedSession, error) {
	if b.converted == nil {
		if _, err := b.Convert(); err != nil {
			return SavedSession{}, err
		}
	}
	if strings.TrimSpace(b.rootDir) == "" {
		return SavedSession{}, fmt.Errorf("Into(rootDir) is required")
	}
	session := b.converted.Session
	sessionID := firstNonEmpty(b.sessionID, session.ID, generateImportID("sess"))
	session.ID = sessionID
	sessionDir := filepath.Join(b.rootDir, sessionID)
	if !b.overwrite {
		if _, err := os.Stat(sessionDir); err == nil {
			return SavedSession{}, fmt.Errorf("session directory %s already exists", sessionDir)
		} else if !os.IsNotExist(err) {
			return SavedSession{}, fmt.Errorf("stat session directory: %w", err)
		}
	}
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return SavedSession{}, fmt.Errorf("create session dir: %w", err)
	}
	sessionPath := filepath.Join(sessionDir, "session.minitrace.json")
	if err := writePrettyJSON(sessionPath, session); err != nil {
		return SavedSession{}, fmt.Errorf("write archive: %w", err)
	}
	saved := SavedSession{SessionID: sessionID, Format: b.converted.Format, Adapter: b.converted.Adapter, Original: b.sourceName(), Title: stringPtr(session.Title), UploadedAt: time.Now().UTC(), SessionDir: sessionDir, SessionPath: sessionPath, MetadataPath: filepath.Join(sessionDir, "metadata.json"), Diagnostics: b.converted.Diagnostics}
	if err := writePrettyJSON(saved.MetadataPath, saved); err != nil {
		return SavedSession{}, fmt.Errorf("write metadata: %w", err)
	}
	return saved, nil
}

func (b *ImportBuilder) load() (*minitracedb.LoadedSession, error) {
	if strings.TrimSpace(b.path) != "" {
		return minitracedb.LoadSessionFileAuto(b.path, minitracedb.LoadOptions{SourcePath: firstNonEmpty(b.sourcePath, b.path), SourceName: firstNonEmpty(b.name, filepath.Base(b.path)), AutoConvert: b.autoConvert})
	}
	if strings.TrimSpace(b.content) == "" {
		return nil, fmt.Errorf("Content(...) or File(...) is required")
	}
	return minitracedb.LoadSessionContentAuto([]byte(b.content), minitracedb.LoadOptions{SourcePath: b.sourcePath, SourceName: b.sourceName(), AutoConvert: b.autoConvert})
}

func (b *ImportBuilder) sourceName() string {
	return firstNonEmpty(b.name, filepath.Base(b.path), b.sourcePath, "content")
}

func writePrettyJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func stringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func truncatePreview(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

func outputContentBytes(result, errorText *string) int {
	if result != nil {
		return len(*result)
	}
	if errorText != nil {
		return len(*errorText)
	}
	return 0
}

func hasImageSignal(contentType *string, text string, metadata any) bool {
	joined := strings.ToLower(strings.Join([]string{stringPtr(contentType), text, compactJSON(metadata)}, " "))
	return strings.Contains(joined, "image") || strings.Contains(joined, "mime") || strings.Contains(joined, "base64") || strings.Contains(joined, "blob")
}

func compactJSON(value any) string {
	if value == nil {
		return ""
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(payload)
}

func generateImportID(prefix string) string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(buf))
}
