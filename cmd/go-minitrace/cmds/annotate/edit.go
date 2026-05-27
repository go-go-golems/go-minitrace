package annotate

//glazedclilint:file-ignore legacy annotate command uses Cobra flags pending Glazed field migration

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/spf13/cobra"
)

func newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit an existing annotation",
		Long:  `Edit the fields of an existing annotation. Only non-empty flags are applied as patches.`,
		RunE:  runEdit,
	}
	cmd.Flags().String("id", "", "Annotation ID to edit (required)")
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("detail", "", "New detail")
	cmd.Flags().String("category", "", "New category")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().String("taxonomy-minitrace", "", "Comma-separated minitrace taxonomy codes")
	cmd.Flags().String("classification", "", "New classification level")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func runEdit(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, _, err := openStore(cmd)
	if err != nil {
		return err
	}
	defer closeStore(store)

	id, _ := cmd.Flags().GetString("id")
	title, _ := cmd.Flags().GetString("title")
	detail, _ := cmd.Flags().GetString("detail")
	category, _ := cmd.Flags().GetString("category")
	tagsStr, _ := cmd.Flags().GetString("tags")
	taxMStr, _ := cmd.Flags().GetString("taxonomy-minitrace")
	classification, _ := cmd.Flags().GetString("classification")

	patch := annotate.AnnotationPatch{}
	if flagIsSet(cmd, "title") {
		patch.Title = &title
	}
	if flagIsSet(cmd, "detail") {
		patch.Detail = &detail
	}
	if flagIsSet(cmd, "category") {
		patch.Category = &category
	}
	if flagIsSet(cmd, "tags") {
		tags := parseCommaList(tagsStr)
		patch.Tags = &tags
	}
	if flagIsSet(cmd, "taxonomy-minitrace") {
		taxM := parseCommaList(taxMStr)
		patch.TaxonomyM = &taxM
	}
	if flagIsSet(cmd, "classification") {
		patch.Classification = &classification
	}

	err = store.Update(ctx, id, patch)
	if err == annotate.ErrNotFound {
		return fmt.Errorf("annotation %q not found", id)
	}
	if err != nil {
		return fmt.Errorf("updating annotation: %w", err)
	}

	fmt.Printf("Updated annotation %s\n", id)
	return nil
}
