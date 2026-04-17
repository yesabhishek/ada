package semantic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/yesabhishek/ada/internal/indexer"
	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/store"
	"github.com/yesabhishek/ada/internal/workspace"
)

type DiffService struct {
	store   *store.Store
	indexer *indexer.Indexer
}

func NewDiffService(store *store.Store, idx *indexer.Indexer) *DiffService {
	return &DiffService{store: store, indexer: idx}
}

func (s *DiffService) SemanticDiff(ctx context.Context, ws *workspace.Workspace, gitCommit string, author string) ([]model.SemanticChange, error) {
	latest, err := s.store.LatestSnapshot(ctx, ws.Config.RepoID)
	if err != nil {
		return nil, err
	}
	var parentID *string
	if latest != nil {
		parentID = &latest.ID
	}
	current, err := s.indexer.IndexWorkspace(ctx, indexer.Options{
		Root:             ws.Root,
		RepoID:           ws.Config.RepoID,
		GitCommit:        gitCommit,
		ParentSnapshotID: parentID,
		Author:           author,
	})
	if err != nil {
		return nil, err
	}
	return diffBundles(ctx, s.store, ws.Config.RepoID, latest, current)
}

func (s *DiffService) TextDiff(ctx context.Context, ws *workspace.Workspace) (string, error) {
	latest, err := s.store.LatestSnapshot(ctx, ws.Config.RepoID)
	if err != nil {
		return "", err
	}
	if latest == nil {
		return "No stored snapshot yet.", nil
	}
	fileVersions, err := s.store.FileVersionsBySnapshot(ctx, latest.ID)
	if err != nil {
		return "", err
	}
	var output string
	for _, version := range fileVersions {
		currentPath := filepath.Join(ws.Root, version.Path)
		current, err := os.ReadFile(currentPath)
		if err != nil {
			continue
		}
		if string(current) == string(version.RawText) {
			continue
		}
		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(version.RawText)),
			B:        difflib.SplitLines(string(current)),
			FromFile: version.Path + " (snapshot)",
			ToFile:   version.Path + " (working)",
			Context:  3,
		})
		if err != nil {
			return "", fmt.Errorf("build text diff for %s: %w", version.Path, err)
		}
		output += diff
		if diff != "" && diff[len(diff)-1] != '\n' {
			output += "\n"
		}
	}
	if output == "" {
		return "No text changes from the latest snapshot.", nil
	}
	return output, nil
}

func diffBundles(ctx context.Context, db *store.Store, repoID string, latest *model.Snapshot, current model.SnapshotBundle) ([]model.SemanticChange, error) {
	catalog, err := db.SymbolCatalog(ctx, repoID)
	if err != nil {
		return nil, err
	}
	currentVersions := make(map[string]model.SymbolVersion, len(current.SymbolVersions))
	for _, version := range current.SymbolVersions {
		currentVersions[version.SymbolID] = version
	}
	if latest == nil {
		var changes []model.SemanticChange
		for _, symbol := range current.Symbols {
			version := currentVersions[symbol.ID]
			changes = append(changes, model.SemanticChange{
				Path:         symbol.Path,
				SymbolID:     symbol.ID,
				FQName:       symbol.FQName,
				Kind:         symbol.Kind,
				Language:     symbol.Language,
				ChangeKind:   model.SemanticChangeAdded,
				AfterHash:    version.ContentHash,
				AfterVersion: &version,
			})
		}
		sortChanges(changes)
		return changes, nil
	}
	priorVersionsSlice, err := db.SymbolVersionsBySnapshot(ctx, latest.ID)
	if err != nil {
		return nil, err
	}
	priorVersions := make(map[string]model.SymbolVersion, len(priorVersionsSlice))
	for _, version := range priorVersionsSlice {
		priorVersions[version.SymbolID] = version
	}
	seen := map[string]struct{}{}
	var changes []model.SemanticChange
	for symbolID, currentVersion := range currentVersions {
		seen[symbolID] = struct{}{}
		symbol := catalog[symbolID]
		priorVersion, exists := priorVersions[symbolID]
		if !exists {
			change := model.SemanticChange{
				Path:         symbol.Path,
				SymbolID:     symbolID,
				FQName:       symbol.FQName,
				Kind:         symbol.Kind,
				Language:     symbol.Language,
				ChangeKind:   model.SemanticChangeAdded,
				AfterHash:    currentVersion.ContentHash,
				AfterVersion: &currentVersion,
			}
			changes = append(changes, change)
			continue
		}
		if priorVersion.ContentHash == currentVersion.ContentHash {
			continue
		}
		before := priorVersion
		after := currentVersion
		changes = append(changes, model.SemanticChange{
			Path:          symbol.Path,
			SymbolID:      symbolID,
			FQName:        symbol.FQName,
			Kind:          symbol.Kind,
			Language:      symbol.Language,
			ChangeKind:    model.SemanticChangeModified,
			BeforeHash:    priorVersion.ContentHash,
			AfterHash:     currentVersion.ContentHash,
			BeforeVersion: &before,
			AfterVersion:  &after,
		})
	}
	for symbolID, priorVersion := range priorVersions {
		if _, ok := seen[symbolID]; ok {
			continue
		}
		symbol := catalog[symbolID]
		before := priorVersion
		changes = append(changes, model.SemanticChange{
			Path:          symbol.Path,
			SymbolID:      symbolID,
			FQName:        symbol.FQName,
			Kind:          symbol.Kind,
			Language:      symbol.Language,
			ChangeKind:    model.SemanticChangeDeleted,
			BeforeHash:    priorVersion.ContentHash,
			BeforeVersion: &before,
		})
	}
	sortChanges(changes)
	return changes, nil
}

func sortChanges(changes []model.SemanticChange) {
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Path == changes[j].Path {
			return changes[i].FQName < changes[j].FQName
		}
		return changes[i].Path < changes[j].Path
	})
}
