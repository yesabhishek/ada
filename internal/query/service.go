package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/store"
)

type Service struct {
	store *store.Store
}

func New(store *store.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Ask(ctx context.Context, repoID string, query string) (string, error) {
	symbols, err := s.store.SearchSymbols(ctx, repoID, query, 8)
	if err != nil {
		return "", err
	}
	snapshots, err := s.store.ListSnapshotCandidates(ctx, repoID, query, 5)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 && len(snapshots) == 0 {
		return "No strong matches yet. Try indexing the repo with `ada sync` or ask with a symbol/file name.", nil
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Query: %s", query))
	if len(symbols) > 0 {
		lines = append(lines, "Relevant symbols:")
		for _, symbol := range symbols {
			lines = append(lines, fmt.Sprintf("- %s [%s] in %s [symbol:%s]", symbol.FQName, symbol.Kind, symbol.Path, symbol.ID[:12]))
		}
	}
	if len(snapshots) > 0 {
		lines = append(lines, "Supporting snapshots:")
		for _, snapshot := range snapshots {
			lines = append(lines, fmt.Sprintf("- %s (%s) [snapshot:%s]", snapshot.Reflection, snapshot.GitCommit, snapshot.ID[:12]))
		}
	}
	if strings.Contains(strings.ToLower(query), "why") {
		lines = append(lines, "Confidence note: historical intent answers are evidence-backed summaries, not certainty, unless an exact task/proposal says so.")
	}
	return strings.Join(lines, "\n"), nil
}

func SnapshotCitations(candidates []model.SnapshotCandidate) []string {
	var out []string
	for _, candidate := range candidates {
		out = append(out, fmt.Sprintf("[snapshot:%s]", candidate.ID))
	}
	return out
}
