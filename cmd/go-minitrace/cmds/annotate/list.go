package annotate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/go-go-golems/go-minitrace/pkg/annotate"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List annotations",
		Long:  `List annotations from the annotation store. Use --session to filter by session.`,
		RunE:  runList,
	}
	cmd.Flags().String("session", "", "Filter by session ID")
	cmd.Flags().String("scope", "", "Filter by scope type (session, turn, tool_call)")
	cmd.Flags().String("category", "", "Filter by category")
	cmd.Flags().String("annotator", "", "Filter by annotator")
	cmd.Flags().String("taxonomy", "", "Match taxonomy code in any taxonomy column")
	cmd.Flags().Int("limit", 50, "Maximum number of results")
	cmd.Flags().String("format", "table", "Output format: table, json")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	store, _, err := openStore(cmd)
	if err != nil {
		return err
	}
	defer closeStore(store)

	sessionID, _ := cmd.Flags().GetString("session")
	scopeType, _ := cmd.Flags().GetString("scope")
	category, _ := cmd.Flags().GetString("category")
	annotator, _ := cmd.Flags().GetString("annotator")
	taxonomy, _ := cmd.Flags().GetString("taxonomy")
	limit, _ := cmd.Flags().GetInt("limit")
	format, _ := cmd.Flags().GetString("format")

	opts := annotate.ListOptions{
		SessionID: sessionID,
		ScopeType: scopeType,
		Category:  category,
		Annotator: annotator,
		Taxonomy:  taxonomy,
		Limit:     limit,
	}

	rows, err := store.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("listing annotations: %w", err)
	}

	if format == "json" {
		return printJSON(rows)
	}
	return printTable(rows)
}

func printTable(rows []annotate.AnnotationRow) error {
	if len(rows) == 0 {
		fmt.Println("No annotations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tSESSION\tSCOPE\tCATEGORY\tTITLE\tANNOTATOR\tCREATED\n")
	for _, r := range rows {
		scope := r.ScopeType
		if r.TargetID != "" && r.TargetID != r.SessionID {
			scope = scope + "/" + r.TargetID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.SessionID, scope, r.Category, r.Title, r.Annotator, r.CreatedAt)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flushing table: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\n%d annotation(s)\n", len(rows))
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}
