package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/gitutil"
	"github.com/yesabhishek/ada/internal/model"
)

type statusPayload struct {
	Workspace struct {
		Root      string `json:"root"`
		RepoID    string `json:"repo_id"`
		RemoteURL string `json:"remote_url,omitempty"`
	} `json:"workspace"`
	Git struct {
		Present       bool     `json:"present"`
		Branch        string   `json:"branch,omitempty"`
		Head          string   `json:"head,omitempty"`
		Clean         bool     `json:"clean"`
		StatusEntries []string `json:"status_entries,omitempty"`
	} `json:"git"`
	LatestSnapshot *model.Snapshot `json:"latest_snapshot,omitempty"`
	SyncRuns       []model.SyncRun `json:"sync_runs,omitempty"`
}

func newStatusCommand() *cobra.Command {
	var (
		includeSync bool
		jsonOutput  bool
	)
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show workspace and sync status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()

			payload := statusPayload{}
			payload.Workspace.Root = f.workspace.Root
			payload.Workspace.RepoID = f.workspace.Config.RepoID
			payload.Workspace.RemoteURL = f.workspace.Config.RemoteURL

			if repo, err := gitutil.Discover(context.Background(), f.workspace.Root); err == nil {
				payload.Git.Present = true
				payload.Git.Head, _ = repo.HeadCommit(context.Background())
				payload.Git.Branch, _ = repo.CurrentBranch(context.Background())
				payload.Git.Clean, _ = repo.IsClean(context.Background())
				payload.Git.StatusEntries, _ = repo.StatusEntries(context.Background())
			}

			latest, err := f.store.LatestSnapshot(context.Background(), f.workspace.Config.RepoID)
			if err != nil {
				return runtimeError("read latest snapshot", err)
			}
			payload.LatestSnapshot = latest

			if includeSync {
				runs, err := f.store.RecentSyncRuns(context.Background(), f.workspace.Config.RepoID, 5)
				if err != nil {
					return runtimeError("read sync runs", err)
				}
				payload.SyncRuns = runs
			}

			if jsonOutput {
				return writeJSON(cmd.OutOrStdout(), payload)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Workspace: %s\n", payload.Workspace.Root)
			fmt.Fprintf(cmd.OutOrStdout(), "Repo ID: %s\n", payload.Workspace.RepoID)
			if payload.Workspace.RemoteURL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Remote URL: %s\n", payload.Workspace.RemoteURL)
				fmt.Fprintln(cmd.OutOrStdout(), "Remote mode: experimental")
			}
			if payload.Git.Present {
				fmt.Fprintf(cmd.OutOrStdout(), "Git: %s @ %s (clean=%t)\n", payload.Git.Branch, payload.Git.Head, payload.Git.Clean)
			}
			if latest == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Latest snapshot: none")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Latest snapshot: %s %s %s\n", latest.ID, latest.GitCommit, latest.Status)
				fmt.Fprintf(cmd.OutOrStdout(), "Reflection: %s\n", latest.Reflection)
			}
			if includeSync {
				if len(payload.SyncRuns) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Sync runs: none")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Recent sync runs:")
					for _, run := range payload.SyncRuns {
						line := fmt.Sprintf("- %s %s %s", run.ID, run.GitCommit, run.State)
						if run.LastError != "" {
							line += " error=" + run.LastError
						}
						fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(line))
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&includeSync, "sync", false, "include recent sync run history")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output status as JSON")
	return cmd
}
