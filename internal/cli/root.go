package cli

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/indexer"
	"github.com/yesabhishek/ada/internal/language"
	golanguage "github.com/yesabhishek/ada/internal/language/golang"
	tslanguage "github.com/yesabhishek/ada/internal/language/typescript"
	"github.com/yesabhishek/ada/internal/query"
	"github.com/yesabhishek/ada/internal/semantic"
	"github.com/yesabhishek/ada/internal/store"
	syncsvc "github.com/yesabhishek/ada/internal/sync"
	"github.com/yesabhishek/ada/internal/tasks"
	"github.com/yesabhishek/ada/internal/workspace"
)

type factory struct {
	workspace *workspace.Workspace
	store     *store.Store
	registry  *language.Registry
	indexer   *indexer.Indexer
	diff      *semantic.DiffService
	query     *query.Service
	sync      *syncsvc.Service
	tasks     *tasks.Service
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ada",
		Short:         "Ada is a semantic sidecar for Git repositories.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newStartCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newDiffCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newAskCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newTaskCommand())
	cmd.AddCommand(newProposeCommand())
	cmd.AddCommand(newResolveCommand())
	cmd.AddCommand(newRewindCommand())
	cmd.AddCommand(newUICommand())
	cmd.AddCommand(newEvalCommand())
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newDoctorCommand())
	return cmd
}

func newFactory(ctx context.Context) (*factory, error) {
	ws, err := workspace.Find(".")
	if err != nil {
		return nil, environmentErrorf("%v", err)
	}
	db, err := store.Open(ws.DBPath())
	if err != nil {
		return nil, runtimeError("open local Ada store", err)
	}
	registry := language.NewRegistry(golanguage.New(), tslanguage.New())
	idx := indexer.New(registry)
	diffService := semantic.NewDiffService(db, idx)
	queryService := query.New(db)
	var publisher syncsvc.ManifestPublisher
	if ws.Config.RemoteURL != "" {
		publisher = &syncsvc.HTTPPublisher{BaseURL: ws.Config.RemoteURL}
	}
	return &factory{
		workspace: ws,
		store:     db,
		registry:  registry,
		indexer:   idx,
		diff:      diffService,
		query:     queryService,
		sync:      syncsvc.NewService(db, idx, publisher),
		tasks:     tasks.New(db, diffService),
	}, nil
}

func (f *factory) Close() error {
	if f == nil {
		return nil
	}
	return f.store.Close()
}

func currentAuthor() string {
	u, err := user.Current()
	if err != nil {
		return "ada"
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

func detectStartTarget(raw string) (path string, remoteURL string) {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return ".", raw
	}
	if raw == "" {
		return ".", ""
	}
	return raw, ""
}

func absoluteTarget(path string) string {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func printErr(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}
