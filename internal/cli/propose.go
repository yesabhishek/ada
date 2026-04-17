package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newProposeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "propose [title]",
		Short: "Create a proposal for the latest snapshot.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()
			title := "Semantic proposal"
			if len(args) > 0 {
				title = strings.Join(args, " ")
			}
			proposal, err := f.tasks.CreateProposal(context.Background(), f.workspace, title)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created proposal %s with %d changed symbols\n", proposal.ID, len(proposal.ChangedSymbols))
			return nil
		},
	}
}
