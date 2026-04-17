package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newTaskCommand() *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage Ada tasks.",
	}
	taskCmd.AddCommand(&cobra.Command{
		Use:   "add <title>",
		Short: "Add a task tied to the latest snapshot.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			task, err := f.tasks.AddTask(context.Background(), f.workspace, strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created task %s (%s)\n", task.ID, task.Status)
			return nil
		},
	})
	return taskCmd
}
