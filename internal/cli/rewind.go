package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/gitutil"
)

func newRewindCommand() *cobra.Command {
	rewindCmd := &cobra.Command{
		Use:   "rewind",
		Short: "Resolve and apply historical semantic snapshots.",
	}
	rewindCmd.AddCommand(&cobra.Command{
		Use:   "search <intent>",
		Short: "Find candidate snapshots that match the provided intent.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			candidates, err := f.store.ListSnapshotCandidates(context.Background(), f.workspace.Config.RepoID, args[0], 10)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No matching snapshots found.")
				return nil
			}
			for _, candidate := range candidates {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", candidate.ID, candidate.GitCommit, candidate.Reflection)
			}
			return nil
		},
	})
	var dryRun bool
	rewindCmd.AddCommand(&cobra.Command{
		Use:   "apply <snapshot-id>",
		Short: "Apply a chosen snapshot back into the working tree.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			snapshot, err := f.store.SnapshotByID(context.Background(), args[0])
			if err != nil {
				return err
			}
			if snapshot == nil {
				return fmt.Errorf("snapshot %s not found", args[0])
			}
			if repo, err := gitutil.Discover(context.Background(), f.workspace.Root); err == nil {
				clean, cleanErr := repo.IsClean(context.Background())
				if cleanErr == nil && !clean && !dryRun {
					return fmt.Errorf("working tree is dirty; clean it or use `ada rewind apply --dry-run` first")
				}
			}
			files, err := f.store.FileVersionsBySnapshot(context.Background(), snapshot.ID)
			if err != nil {
				return err
			}
			for _, file := range files {
				target := filepath.Join(f.workspace.Root, file.Path)
				if dryRun {
					fmt.Fprintf(cmd.OutOrStdout(), "Would write %s\n", target)
					continue
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(target, file.RawText, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", target)
			}
			return nil
		},
	})
	rewindCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show the files that would be restored without writing them")
	return rewindCmd
}
