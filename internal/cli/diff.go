package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/gitutil"
)

func newDiffCommand() *cobra.Command {
	var (
		mode         string
		semanticOnly bool
		textOnly     bool
		jsonOutput   bool
	)
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show semantic or text diff against the latest snapshot.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()

			if semanticOnly && textOnly {
				return usageErrorf("choose either --semantic or --text, not both")
			}
			if semanticOnly {
				mode = "semantic"
			}
			if textOnly {
				mode = "text"
			}

			switch mode {
			case "semantic":
				commit := "working-tree"
				if repo, err := gitutil.Discover(context.Background(), f.workspace.Root); err == nil {
					if head, headErr := repo.HeadCommit(context.Background()); headErr == nil {
						commit = head
					}
				}
				changes, err := f.diff.SemanticDiff(context.Background(), f.workspace, commit, currentAuthor())
				if err != nil {
					return runtimeError("compute semantic diff", err)
				}
				if jsonOutput {
					items := make([]map[string]any, 0, len(changes))
					for _, change := range changes {
						items = append(items, map[string]any{
							"path":        change.Path,
							"symbol_id":   change.SymbolID,
							"fq_name":     change.FQName,
							"kind":        change.Kind,
							"language":    change.Language,
							"change_kind": change.ChangeKind,
							"before_hash": change.BeforeHash,
							"after_hash":  change.AfterHash,
						})
					}
					return writeJSON(cmd.OutOrStdout(), struct {
						Mode    string           `json:"mode"`
						Count   int              `json:"count"`
						Changes []map[string]any `json:"changes"`
					}{
						Mode:    "semantic",
						Count:   len(changes),
						Changes: items,
					})
				}
				if len(changes) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No semantic changes from the latest snapshot.")
					return nil
				}
				for _, change := range changes {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s [%s] %s\n", change.ChangeKind, change.FQName, change.Kind, change.Path)
				}
			case "text":
				diff, err := f.diff.TextDiff(context.Background(), f.workspace)
				if err != nil {
					return runtimeError("compute text diff", err)
				}
				if jsonOutput {
					return writeJSON(cmd.OutOrStdout(), struct {
						Mode string `json:"mode"`
						Diff string `json:"diff"`
					}{
						Mode: "text",
						Diff: diff,
					})
				}
				fmt.Fprint(cmd.OutOrStdout(), diff)
			default:
				return usageErrorf("unsupported diff mode %q", mode)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "semantic", "diff mode: semantic or text")
	cmd.Flags().BoolVar(&semanticOnly, "semantic", false, "show semantic diff")
	cmd.Flags().BoolVar(&textOnly, "text", false, "show text diff")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output diff as JSON")
	return cmd
}
