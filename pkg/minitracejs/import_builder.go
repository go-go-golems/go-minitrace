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

func generateImportID(prefix string) string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(buf))
}
