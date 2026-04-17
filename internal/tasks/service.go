package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/semantic"
	"github.com/yesabhishek/ada/internal/store"
	"github.com/yesabhishek/ada/internal/workspace"
)

type Service struct {
	store *store.Store
	diff  *semantic.DiffService
}

func New(store *store.Store, diff *semantic.DiffService) *Service {
	return &Service{store: store, diff: diff}
}

func (s *Service) AddTask(ctx context.Context, ws *workspace.Workspace, title string) (*model.Task, error) {
	snapshot, err := s.store.LatestSnapshot(ctx, ws.Config.RepoID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, fmt.Errorf("no indexed snapshot yet; run `ada sync` first")
	}
	task := &model.Task{
		ID:         uuid.NewString(),
		SnapshotID: snapshot.ID,
		Title:      title,
		Status:     "todo",
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.store.CreateTask(ctx, *task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) CreateProposal(ctx context.Context, ws *workspace.Workspace, title string) (*model.Proposal, error) {
	snapshot, err := s.store.LatestSnapshot(ctx, ws.Config.RepoID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, fmt.Errorf("no snapshot available; run `ada sync` first")
	}
	changes, err := s.diff.SemanticDiff(ctx, ws, snapshot.GitCommit, "proposal")
	if err != nil {
		return nil, err
	}
	changedSymbols := make([]string, 0, len(changes))
	for _, change := range changes {
		changedSymbols = append(changedSymbols, change.FQName)
	}
	summary := snapshot.Reflection
	if len(changedSymbols) > 0 {
		summary = summary + " Proposal captures semantic drift in working tree: " + strings.Join(changedSymbols, ", ")
	}
	proposal := &model.Proposal{
		ID:             uuid.NewString(),
		SnapshotID:     snapshot.ID,
		Title:          title,
		Status:         "qa",
		Summary:        summary,
		ChangedSymbols: changedSymbols,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.store.CreateProposal(ctx, *proposal); err != nil {
		return nil, err
	}
	return proposal, nil
}

func (s *Service) ResolveLatestProposal(ctx context.Context, ws *workspace.Workspace, resolution string) (*model.Proposal, error) {
	snapshot, err := s.store.LatestSnapshot(ctx, ws.Config.RepoID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, fmt.Errorf("no snapshot available")
	}
	proposal, err := s.store.LatestProposal(ctx, snapshot.ID)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, fmt.Errorf("no proposal found for latest snapshot")
	}
	if err := s.store.UpdateProposalStatus(ctx, proposal.ID, "resolved", resolution); err != nil {
		return nil, err
	}
	proposal.Status = "resolved"
	proposal.Summary = resolution
	return proposal, nil
}
