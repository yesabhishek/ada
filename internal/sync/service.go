package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yesabhishek/ada/internal/controlplane"
	"github.com/yesabhishek/ada/internal/gitutil"
	"github.com/yesabhishek/ada/internal/indexer"
	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/store"
	"github.com/yesabhishek/ada/internal/workspace"
)

type ManifestPublisher interface {
	PublishManifest(ctx context.Context, manifest controlplane.Manifest) error
}

type HTTPPublisher struct {
	BaseURL string
	Client  *http.Client
}

func (p *HTTPPublisher) PublishManifest(ctx context.Context, manifest controlplane.Manifest) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/api/v1/manifests", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build manifest request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("publish manifest: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("publish manifest: unexpected status %s", res.Status)
	}
	return nil
}

type Service struct {
	store     *store.Store
	indexer   *indexer.Indexer
	publisher ManifestPublisher
}

func NewService(store *store.Store, idx *indexer.Indexer, publisher ManifestPublisher) *Service {
	return &Service{store: store, indexer: idx, publisher: publisher}
}

type Result struct {
	Snapshot model.Snapshot
	SyncRun  model.SyncRun
}

func (s *Service) Sync(ctx context.Context, ws *workspace.Workspace, author string) (*Result, error) {
	repo, err := gitutil.Discover(ctx, ws.Root)
	if err != nil {
		return nil, fmt.Errorf("sync requires a git repository: %w", err)
	}
	clean, err := repo.IsClean(ctx)
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, fmt.Errorf("working tree is dirty; commit changes before `ada sync`")
	}
	headCommit, err := repo.HeadCommit(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := s.store.LatestSnapshot(ctx, ws.Config.RepoID)
	if err != nil {
		return nil, err
	}
	var parentID *string
	if latest != nil {
		parentID = &latest.ID
	}
	bundle, err := s.indexer.IndexWorkspace(ctx, indexer.Options{
		Root:             ws.Root,
		RepoID:           ws.Config.RepoID,
		GitCommit:        headCommit,
		ParentSnapshotID: parentID,
		Author:           author,
	})
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveSnapshotBundle(ctx, bundle); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	syncRun := model.SyncRun{
		ID:         uuid.NewString(),
		RepoID:     ws.Config.RepoID,
		GitCommit:  headCommit,
		SnapshotID: bundle.Snapshot.ID,
		State:      model.SnapshotStatusSynced,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if ws.Config.RemoteURL != "" {
		hasRemote, err := repo.HasRemote(ctx)
		if err != nil {
			return nil, err
		}
		if !hasRemote {
			syncRun.State = model.SnapshotStatusFailedGitPush
			syncRun.LastError = "workspace has an Ada remote URL but no git remote configured"
		} else if err := repo.Push(ctx); err != nil {
			syncRun.State = model.SnapshotStatusFailedGitPush
			syncRun.LastError = err.Error()
		} else if s.publisher != nil {
			manifest := manifestFromBundle(bundle)
			if err := s.publisher.PublishManifest(ctx, manifest); err != nil {
				syncRun.State = model.SnapshotStatusPendingReconcile
				syncRun.LastError = err.Error()
			}
		}
	}
	if err := s.store.UpdateSnapshotStatus(ctx, bundle.Snapshot.ID, syncRun.State, bundle.Snapshot.Reflection); err != nil {
		return nil, err
	}
	if err := s.store.CreateOrUpdateSyncRun(ctx, syncRun); err != nil {
		return nil, err
	}
	bundle.Snapshot.Status = syncRun.State
	return &Result{Snapshot: bundle.Snapshot, SyncRun: syncRun}, nil
}

func manifestFromBundle(bundle model.SnapshotBundle) controlplane.Manifest {
	manifest := controlplane.Manifest{
		RepoID:     bundle.Snapshot.RepoID,
		GitCommit:  bundle.Snapshot.GitCommit,
		SnapshotID: bundle.Snapshot.ID,
		CreatedAt:  bundle.Snapshot.CreatedAt,
	}
	for _, file := range bundle.FileVersions {
		manifest.Files = append(manifest.Files, controlplane.FileDigest{
			Path:        file.Path,
			Language:    file.Language,
			RawTextHash: file.RawTextHash,
		})
	}
	return manifest
}
