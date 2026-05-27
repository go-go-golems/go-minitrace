package annotate

//glazedclilint:file-ignore legacy annotate command uses Cobra flags pending Glazed field migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an annotation to a session",
		Long:  `Add an annotation to a minitrace session. Annotations are stored in outputDir/annotations.db.`,
		RunE:  runAdd,
	}
	cmd.Flags().String("session", "", "Session ID to annotate (required)")
	cmd.Flags().String("scope", "session", "Scope: session, turn, or tool_call")
	cmd.Flags().String("target-id", "", "Target ID for turn/tool_call scope (default: session ID)")
	cmd.Flags().String("annotator", "user", "Annotator identifier")
	cmd.Flags().String("category", "", "Annotation category (required)")
	cmd.Flags().String("title", "", "Annotation title (required)")
	cmd.Flags().String("detail", "", "Annotation detail/description")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().String("taxonomy-minitrace", "", "Comma-separated minitrace taxonomy codes (e.g. F-AUT,O-INT)")
	cmd.Flags().String("taxonomy-mast", "", "Comma-separated MAST taxonomy codes")
	cmd.Flags().String("taxonomy-toolemu", "", "Comma-separated Toolemu taxonomy codes")
	cmd.Flags().String("classification", "", "Classification level (public, internal, confidential, customer-confidential)")
	_ = cmd.MarkFlagRequired("session")
	_ = cmd.MarkFlagRequired("category")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func runAdd(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, outputDir, err := openStore(cmd)
	if err != nil {
		return err
	}
	defer closeStore(store)

	sessionID, _ := cmd.Flags().GetString("session")
	scopeType, _ := cmd.Flags().GetString("scope")
	targetID, _ := cmd.Flags().GetString("target-id")
	annotator, _ := cmd.Flags().GetString("annotator")
	category, _ := cmd.Flags().GetString("category")
	title, _ := cmd.Flags().GetString("title")
	detail, _ := cmd.Flags().GetString("detail")
	tagsStr, _ := cmd.Flags().GetString("tags")
	taxMStr, _ := cmd.Flags().GetString("taxonomy-minitrace")
	taxMastStr, _ := cmd.Flags().GetString("taxonomy-mast")
	taxTmStr, _ := cmd.Flags().GetString("taxonomy-toolemu")
	classification, _ := cmd.Flags().GetString("classification")

	if targetID == "" {
		targetID = sessionID
	}

	// Validate scope type.
	switch scopeType {
	case "session", "turn", "tool_call":
	default:
		return fmt.Errorf("--scope must be one of: session, turn, tool_call")
	}

	// Validate category against known values.
	validCategories := map[string]bool{
		"observation":       true,
		"ai-failure":        true,
		"user-error":        true,
		"environment-issue": true,
		"success":           true,
		"question":          true,
		"to-discuss":        true,
		"to-improve":        true,
	}
	if !validCategories[category] {
		return fmt.Errorf("--category %q is not a recognized category (see annotate --help)", category)
	}

	tags := parseCommaList(tagsStr)
	taxM := parseCommaList(taxMStr)
	taxMast := parseCommaList(taxMastStr)
	taxTm := parseCommaList(taxTmStr)

	ann := minitrace.Annotation{
		ID:               uuid.New().String(),
		Timestamp:        minitrace.FormatTimestamp(time.Now().UTC()),
		Annotator:        annotator,
		Scope:            minitrace.AnnotationScope{Type: scopeType, TargetID: targetID},
		Content:          minitrace.AnnotationContent{Category: category, Tags: tags, Title: title, Detail: detail},
		TaxonomyMappings: minitrace.TaxonomyMappings{Minitrace: taxM, Mast: taxMast, Toolemu: taxTm},
	}
	if classification != "" {
		ann.Classification = &classification
	}

	if err := store.AddAnnotation(ctx, ann, sessionID); err != nil {
		return fmt.Errorf("adding annotation: %w", err)
	}

	fmt.Printf("Added annotation %s to session %s\n", ann.ID, sessionID)
	fmt.Printf("  scope: %s / %s\n", scopeType, targetID)
	fmt.Printf("  category: %s | title: %s\n", category, title)
	if len(tags) > 0 {
		fmt.Printf("  tags: %s\n", tagsStr)
	}
	if len(taxM) > 0 {
		fmt.Printf("  taxonomy(minitrace): %s\n", taxMStr)
	}
	fmt.Printf("  stored in: %s\n", outputDir)

	return nil
}

func openStore(cmd *cobra.Command) (*annotate.Store, string, error) {
	outputDir, err := cmd.Flags().GetString(flagOutputDir)
	if err != nil {
		return nil, "", err
	}
	if outputDir == "" {
		outputDir = "./output"
	}
	store, err := annotate.Open(context.Background(), outputDir)
	if err != nil {
		return nil, outputDir, fmt.Errorf("opening annotation store: %w", err)
	}
	return store, outputDir, nil
}

func closeStore(store *annotate.Store) { _ = store.Close() }

func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	// Remove empty parts.
	result := parts[:0]
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func flagIsSet(cmd *cobra.Command, name string) bool {
	fs := cmd.Flags()
	found := false
	fs.VisitAll(func(flag *pflag.Flag) {
		if flag.Name == name {
			found = flag.Changed
		}
	})
	return found
}
