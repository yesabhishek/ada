package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <query>",
		Short: "Query the local semantic memory for symbols and snapshots.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			answer, err := f.query.Ask(context.Background(), f.workspace.Config.RepoID, strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), answer)
			return nil
		},
	}
}
