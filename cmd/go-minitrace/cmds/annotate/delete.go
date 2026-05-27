package annotate

//glazedclilint:file-ignore legacy annotate command uses Cobra flags pending Glazed field migration

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an annotation",
		Long:  `Delete an annotation by its ID.`,
		RunE:  runDelete,
	}
	cmd.Flags().String("id", "", "Annotation ID to delete (required)")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func runDelete(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, _, err := openStore(cmd)
	if err != nil {
		return err
	}
	defer closeStore(store)

	id, _ := cmd.Flags().GetString("id")

	err = store.Delete(ctx, id)
	if err == annotate.ErrNotFound {
		return fmt.Errorf("annotation %q not found", id)
	}
	if err != nil {
		return fmt.Errorf("deleting annotation: %w", err)
	}

	fmt.Printf("Deleted annotation %s\n", id)
	return nil
}
