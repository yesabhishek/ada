package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Index the current Git commit and sync its semantic snapshot.",
		Long: "Sync the local semantic snapshot for the current Git commit.\n\n" +
			"Local snapshotting is part of the supported alpha workflow.\n" +
			"Remote publish behavior remains experimental when a remote URL is configured.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			result, err := f.sync.Sync(context.Background(), f.workspace, currentAuthor())
			if err != nil {
				return runtimeError("sync Ada snapshot", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Snapshot: %s\n", result.Snapshot.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Git commit: %s\n", result.Snapshot.GitCommit)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.SyncRun.State)
			if result.SyncRun.LastError != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Note: %s\n", result.SyncRun.LastError)
			}
			if f.workspace.Config.RemoteURL != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Experimental: remote publish is not part of the supported public alpha workflow.")
			}
			return nil
		},
	}
}
