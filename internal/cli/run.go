package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <prompt>",
		Short: "Record an agent workflow prompt against the latest snapshot.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			task, err := f.tasks.AddTask(context.Background(), f.workspace, "[run] "+strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Recorded agent run as task %s on snapshot %s\n", task.ID, task.SnapshotID)
			return nil
		},
	}
}
