package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/store"
	"github.com/yesabhishek/ada/internal/workspace"
)

func newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <path-or-url>",
		Short: "Initialize a local Ada sidecar; URL mode is experimental.",
		Long: "Initialize Ada in the current repository or target path.\n\n" +
			"`ada start .` is the supported alpha workflow.\n" +
			"`ada start <url>` configures experimental remote control-plane mode.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			path, remoteURL := detectStartTarget(target)
			ws, err := workspace.Initialize(path, remoteURL)
			if err != nil {
				return runtimeError("initialize Ada workspace", err)
			}
			db, err := store.Open(ws.DBPath())
			if err != nil {
				return runtimeError("initialize local Ada store", err)
			}
			defer db.Close()
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized Ada workspace at %s\n", absoluteTarget(ws.Root))
			if remoteURL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Remote control plane: %s\n", remoteURL)
				fmt.Fprintln(cmd.OutOrStdout(), "Experimental: remote control-plane mode is not part of the supported public alpha workflow.")
			}
			return nil
		},
	}
}
