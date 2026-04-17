package indexer

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yesabhishek/ada/internal/language"
	"github.com/yesabhishek/ada/internal/model"
)

type Indexer struct {
	registry *language.Registry
}

func New(registry *language.Registry) *Indexer {
	return &Indexer{registry: registry}
}

type Options struct {
	Root             string
	RepoID           string
	GitCommit        string
	ParentSnapshotID *string
	Author           string
}

func (i *Indexer) IndexWorkspace(ctx context.Context, opts Options) (model.SnapshotBundle, error) {
	if opts.Root == "" {
		return model.SnapshotBundle{}, fmt.Errorf("index workspace: missing root")
	}
	snapshotID := uuid.NewString()
	files, err := i.collectFiles(ctx, opts.Root)
	if err != nil {
		return model.SnapshotBundle{}, err
	}
	snapshot := model.Snapshot{
		ID:               snapshotID,
		RepoID:           opts.RepoID,
		GitCommit:        opts.GitCommit,
		ParentSnapshotID: opts.ParentSnapshotID,
		Status:           model.SnapshotStatusIndexed,
		Reflection:       language.SnapshotReflection(len(files), 0),
		Author:           opts.Author,
		CreatedAt:        time.Now().UTC(),
	}
	bundle := model.SnapshotBundle{Snapshot: snapshot}
	symbolLookup := map[string]string{}

	for _, file := range files {
		fileVersionID := language.FileVersionID(snapshotID, file.Path)
		fileVersion := model.FileVersion{
			ID:            fileVersionID,
			SnapshotID:    snapshotID,
			Path:          file.Path,
			Language:      file.Parsed.Language,
			RawTextHash:   model.HashText(string(file.Raw)),
			RawText:       file.Raw,
			ParserVersion: file.Parsed.ParserVersion,
			FormatterID:   file.Parsed.FormatterID,
		}
		bundle.FileVersions = append(bundle.FileVersions, fileVersion)
		for _, symbol := range file.Parsed.Symbols {
			symbolID := language.SymbolID(opts.RepoID, file.Path, file.Parsed.Language, symbol.Kind, symbol.FQName)
			bundle.Symbols = append(bundle.Symbols, model.Symbol{
				ID:       symbolID,
				RepoID:   opts.RepoID,
				Path:     file.Path,
				FQName:   symbol.FQName,
				Language: file.Parsed.Language,
				Kind:     symbol.Kind,
			})
			bundle.SymbolVersions = append(bundle.SymbolVersions, model.SymbolVersion{
				SnapshotID:      snapshotID,
				SymbolID:        symbolID,
				FileVersionID:   fileVersionID,
				ContentHash:     model.HashText(symbol.NormalizedBody),
				SignatureHash:   model.HashText(symbol.Signature),
				StartByte:       symbol.StartByte,
				EndByte:         symbol.EndByte,
				AnchorStartByte: symbol.AnchorStartByte,
				AnchorEndByte:   symbol.AnchorEndByte,
				NodeType:        symbol.NodeType,
				NormalizedBody:  symbol.NormalizedBody,
				RawText:         symbol.RawText,
				LineStart:       symbol.LineStart,
				LineEnd:         symbol.LineEnd,
			})
			bundle.NodeBlobs = append(bundle.NodeBlobs, model.NodeBlob{
				ContentHash:    model.HashText(symbol.NormalizedBody),
				Language:       file.Parsed.Language,
				NodeType:       symbol.NodeType,
				NormalizedBody: symbol.NormalizedBody,
				SchemaVersion:  "v1",
			})
			symbolLookup[file.Path+"::"+symbol.FQName] = symbolID
		}
	}

	for _, file := range files {
		for _, edge := range file.Parsed.Edges {
			fromID, fromOK := symbolLookup[file.Path+"::"+edge.FromFQName]
			toID, toOK := symbolLookup[file.Path+"::"+edge.ToFQName]
			if !fromOK || !toOK {
				continue
			}
			bundle.Edges = append(bundle.Edges, model.Edge{
				SnapshotID:      snapshotID,
				FromSymbolID:    fromID,
				ToSymbolID:      toID,
				EdgeType:        edge.EdgeType,
				EdgeSource:      edge.EdgeSource,
				ConfidenceScore: edge.ConfidenceScore,
			})
		}
	}
	bundle.Snapshot.Reflection = language.SnapshotReflection(len(bundle.FileVersions), len(bundle.SymbolVersions))
	return bundle, nil
}

type indexedFile struct {
	Path   string
	Raw    []byte
	Parsed *language.ParsedFile
}

func (i *Indexer) collectFiles(ctx context.Context, root string) ([]indexedFile, error) {
	var files []indexedFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch rel {
			case ".ada", ".git", "node_modules", "dist", "build":
				if rel != "." {
					return filepath.SkipDir
				}
			}
			return nil
		}
		adapter := i.registry.ForPath(path)
		if adapter == nil {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", rel, err)
		}
		parsed, err := adapter.Parse(ctx, rel, raw)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		files = append(files, indexedFile{
			Path:   filepath.ToSlash(rel),
			Raw:    raw,
			Parsed: parsed,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.Compare(files[i].Path, files[j].Path) < 0
	})
	return files, nil
}
