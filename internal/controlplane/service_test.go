package controlplane

import (
	"context"
	"testing"
	"time"
)

type fakeCommitSource struct {
	files map[string][]byte
}

func (f fakeCommitSource) FetchCommit(_ context.Context, _, _ string) (map[string][]byte, error) {
	return f.files, nil
}

type fakeIndexer struct{}

func (fakeIndexer) IndexCommit(_ context.Context, repoID, gitCommit string, files map[string][]byte) (Manifest, error) {
	manifest := Manifest{
		RepoID:     repoID,
		GitCommit:  gitCommit,
		SnapshotID: "reindexed-" + gitCommit,
		CreatedAt:  time.Now().UTC(),
	}
	for path, raw := range files {
		manifest.Files = append(manifest.Files, FileDigest{Path: path, RawTextHash: string(raw)})
	}
	return manifest, nil
}

func TestReconcileOnceHealsMissingCommit(t *testing.T) {
	store := NewMemoryStore()
	if err := store.EnqueueMissing(context.Background(), MissingCommit{RepoID: "ada", GitCommit: "abc123"}); err != nil {
		t.Fatalf("EnqueueMissing() error = %v", err)
	}
	service := NewService(store, fakeIndexer{}, fakeCommitSource{files: map[string][]byte{"main.go": []byte("package main")}})
	if err := service.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce() error = %v", err)
	}
	manifest, err := store.Manifest(context.Background(), "ada", "abc123")
	if err != nil {
		t.Fatalf("Manifest() error = %v", err)
	}
	if manifest == nil || manifest.SnapshotID == "" {
		t.Fatalf("expected reconciled manifest, got %#v", manifest)
	}
}
