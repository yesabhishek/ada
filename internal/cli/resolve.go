package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newResolveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <resolution>",
		Short: "Resolve the latest proposal with natural-language guidance.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			proposal, err := f.tasks.ResolveLatestProposal(context.Background(), f.workspace, strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Resolved proposal %s\n", proposal.ID)
			return nil
		},
	}
}
