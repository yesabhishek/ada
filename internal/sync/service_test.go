package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/yesabhishek/ada/internal/indexer"
	"github.com/yesabhishek/ada/internal/language"
	golanguage "github.com/yesabhishek/ada/internal/language/golang"
	tslanguage "github.com/yesabhishek/ada/internal/language/typescript"
	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/store"
	"github.com/yesabhishek/ada/internal/testutil"
	"github.com/yesabhishek/ada/internal/workspace"
)

func TestSyncMarksPendingReconcileWhenManifestPublishFails(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	testutil.InitGitRepo(t, root)
	testutil.Run(t, remote, "git", "init", "--bare")
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc add() int { return 1 }\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	testutil.Run(t, root, "git", "add", "main.go")
	testutil.Run(t, root, "git", "commit", "-m", "initial")
	testutil.Run(t, root, "git", "remote", "add", "origin", remote)
	testutil.Run(t, root, "git", "push", "-u", "origin", "HEAD")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	ws, err := workspace.Initialize(root, server.URL)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	db, err := store.Open(ws.DBPath())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()
	registry := language.NewRegistry(golanguage.New(), tslanguage.New())
	service := NewService(db, indexer.New(registry), &HTTPPublisher{
		BaseURL: server.URL,
		Client:  server.Client(),
	})

	result, err := service.Sync(context.Background(), ws, "Ada Test")
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.SyncRun.State != model.SnapshotStatusPendingReconcile {
		t.Fatalf("expected pending_reconcile, got %s", result.SyncRun.State)
	}
}

func TestSyncStoresPendingReconcileWhenPublisherFailsWithoutRemotePush(t *testing.T) {
	root := t.TempDir()
	testutil.InitGitRepo(t, root)
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc add() int { return 1 }\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	testutil.Run(t, root, "git", "add", "main.go")
	testutil.Run(t, root, "git", "commit", "-m", "initial")

	ws, err := workspace.Initialize(root, "")
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	db, err := store.Open(ws.DBPath())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	service := NewService(db, indexer.New(language.NewRegistry(golanguage.New(), tslanguage.New())), nil)
	result, err := service.Sync(context.Background(), ws, "Ada Test")
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.SyncRun.State != model.SnapshotStatusSynced {
		t.Fatalf("expected synced state, got %s", result.SyncRun.State)
	}
}
