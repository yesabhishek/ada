package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type ManifestStore interface {
	UpsertManifest(ctx context.Context, manifest Manifest) error
	Manifest(ctx context.Context, repoID, gitCommit string) (*Manifest, error)
	ListMissingCommits(ctx context.Context) ([]MissingCommit, error)
	MarkReindexing(ctx context.Context, repoID, gitCommit string) error
	EnqueueMissing(ctx context.Context, commit MissingCommit) error
}

type CommitSource interface {
	FetchCommit(ctx context.Context, repoID, gitCommit string) (map[string][]byte, error)
}

type ManifestIndexer interface {
	IndexCommit(ctx context.Context, repoID, gitCommit string, files map[string][]byte) (Manifest, error)
}

type Service struct {
	store   ManifestStore
	indexer ManifestIndexer
	source  CommitSource
}

func NewService(store ManifestStore, indexer ManifestIndexer, source CommitSource) *Service {
	return &Service{store: store, indexer: indexer, source: source}
}

func (s *Service) AcceptManifest(ctx context.Context, manifest Manifest) error {
	return s.store.UpsertManifest(ctx, manifest)
}

func (s *Service) ReconcileOnce(ctx context.Context) error {
	missing, err := s.store.ListMissingCommits(ctx)
	if err != nil {
		return err
	}
	for _, item := range missing {
		if err := s.store.MarkReindexing(ctx, item.RepoID, item.GitCommit); err != nil {
			return err
		}
		files, err := s.source.FetchCommit(ctx, item.RepoID, item.GitCommit)
		if err != nil {
			return fmt.Errorf("fetch commit %s@%s: %w", item.RepoID, item.GitCommit, err)
		}
		manifest, err := s.indexer.IndexCommit(ctx, item.RepoID, item.GitCommit, files)
		if err != nil {
			return fmt.Errorf("reindex commit %s@%s: %w", item.RepoID, item.GitCommit, err)
		}
		if err := s.store.UpsertManifest(ctx, manifest); err != nil {
			return fmt.Errorf("persist manifest %s@%s: %w", item.RepoID, item.GitCommit, err)
		}
	}
	return nil
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/manifests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var manifest Manifest
		if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.AcceptManifest(r.Context(), manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}

type MemoryStore struct {
	mu        sync.Mutex
	manifests map[string]Manifest
	missing   []MissingCommit
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{manifests: map[string]Manifest{}}
}

func (s *MemoryStore) UpsertManifest(_ context.Context, manifest Manifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manifests[key(manifest.RepoID, manifest.GitCommit)] = manifest
	return nil
}

func (s *MemoryStore) Manifest(_ context.Context, repoID, gitCommit string) (*Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, ok := s.manifests[key(repoID, gitCommit)]
	if !ok {
		return nil, nil
	}
	clone := manifest
	return &clone, nil
}

func (s *MemoryStore) ListMissingCommits(_ context.Context) ([]MissingCommit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]MissingCommit(nil), s.missing...)
	s.missing = nil
	return out, nil
}

func (s *MemoryStore) MarkReindexing(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *MemoryStore) EnqueueMissing(_ context.Context, commit MissingCommit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.missing = append(s.missing, commit)
	return nil
}

func key(repoID, gitCommit string) string {
	return repoID + "@" + gitCommit
}
