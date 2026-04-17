package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/buildinfo"
)

func newVersionCommand() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show Ada version and build information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := buildinfo.Current()
			if jsonOutput {
				return writeJSON(cmd.OutOrStdout(), info)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), buildinfo.HumanString())
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output version information as JSON")
	return cmd
}
