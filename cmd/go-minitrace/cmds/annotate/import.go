package annotate

//glazedclilint:file-ignore legacy annotate command uses Cobra flags pending Glazed field migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import annotations from a JSON file",
		Long: `Import annotations from a JSON file into the annotation store.

The file must contain a JSON array of annotation objects. Each object is
inserted as-is into the store. The session_id is required in each object.

Example input:
[
  {
    "session_id": "sess-001",
    "annotator": "user",
    "scope_type": "session",
    "category": "observation",
    "title": "Imported annotation"
  }
]
`,
		RunE: runImport,
	}
	cmd.Flags().String("file", "", "JSON file to import (use - for stdin)")
	return cmd
}

func runImport(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, _, err := openStore(cmd)
	if err != nil {
		return err
	}
	defer closeStore(store)

	filePath, _ := cmd.Flags().GetString("file")

	var data []byte
	if filePath == "" || filePath == "-" {
		data, err = os.ReadFile("/dev/stdin")
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	imported := 0
	for i, obj := range raw {
		sessionID, ok := obj["session_id"].(string)
		if !ok || sessionID == "" {
			fmt.Fprintf(os.Stderr, "skipping item %d: missing or invalid session_id\n", i)
			continue
		}

		ann, err := rawToAnnotation(obj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping item %d: %v\n", i, err)
			continue
		}

		if err := store.AddAnnotation(ctx, *ann, sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "error importing item %d: %v\n", i, err)
			continue
		}
		imported++
	}

	fmt.Printf("Imported %d annotation(s)\n", imported)
	return nil
}

func rawToAnnotation(obj map[string]any) (*minitrace.Annotation, error) {
	scopeType := getString(obj, "scope_type", "session")
	targetID := getString(obj, "target_id", getString(obj, "session_id", ""))

	ann := &minitrace.Annotation{
		ID:        getString(obj, "id", uuid.New().String()),
		Timestamp: minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator: getString(obj, "annotator", "import"),
		Scope: minitrace.AnnotationScope{
			Type:     scopeType,
			TargetID: targetID,
		},
		Content: minitrace.AnnotationContent{
			Category: getString(obj, "category", ""),
			Title:    getString(obj, "title", ""),
			Detail:   getString(obj, "detail", ""),
			Tags:     getStringArray(obj, "tags"),
		},
		TaxonomyMappings: minitrace.TaxonomyMappings{
			Minitrace: getStringArray(obj, "taxonomy_minitrace"),
			Mast:      getStringArray(obj, "taxonomy_mast"),
			Toolemu:   getStringArray(obj, "taxonomy_toolemu"),
		},
	}
	if classif, ok := obj["classification"].(string); ok && classif != "" {
		ann.Classification = &classif
	}
	return ann, nil
}

func getString(obj map[string]any, key, fallback string) string {
	if v, ok := obj[key].(string); ok {
		return v
	}
	return fallback
}

func getStringArray(obj map[string]any, key string) []string {
	v, ok := obj[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case []any:
		out := make([]string, 0, len(arr))
		for _, e := range arr {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return arr
	case string:
		return parseCommaList(arr)
	}
	return nil
}
