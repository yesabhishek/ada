package semantic

import (
	"context"
	"sort"

	"github.com/yesabhishek/ada/internal/language"
	"github.com/yesabhishek/ada/internal/model"
)

type Conflict struct {
	Path   string
	Symbol string
	Reason string
}

type MergeResult struct {
	MergedSource []byte
	Conflicts    []Conflict
}

func ThreeWayMerge(ctx context.Context, registry *language.Registry, path string, base []byte, baseSymbols, oursSymbols, theirsSymbols []model.SymbolVersion) (MergeResult, error) {
	baseMap := mapBySymbol(baseSymbols)
	oursMap := mapBySymbol(oursSymbols)
	theirsMap := mapBySymbol(theirsSymbols)
	keys := unionKeys(baseMap, oursMap, theirsMap)
	sort.Strings(keys)

	var edits []Edit
	var conflicts []Conflict
	for _, key := range keys {
		baseVersion, baseOK := baseMap[key]
		oursVersion, oursOK := oursMap[key]
		theirsVersion, theirsOK := theirsMap[key]

		switch {
		case !baseOK && oursOK && theirsOK:
			if oursVersion.ContentHash != theirsVersion.ContentHash {
				conflicts = append(conflicts, Conflict{Path: path, Symbol: key, Reason: "symbol added differently on both sides"})
				continue
			}
			edits = append(edits, Edit{Path: path, StartByte: 0, EndByte: 0, Replacement: oursVersion.RawText})
		case baseOK && oursOK && theirsOK:
			oursChanged := baseVersion.ContentHash != oursVersion.ContentHash
			theirsChanged := baseVersion.ContentHash != theirsVersion.ContentHash
			if oursChanged && theirsChanged && oursVersion.ContentHash != theirsVersion.ContentHash {
				conflicts = append(conflicts, Conflict{Path: path, Symbol: key, Reason: "same symbol changed differently"})
				continue
			}
			if oursChanged {
				edits = append(edits, Edit{
					Path:        path,
					StartByte:   baseVersion.AnchorStartByte,
					EndByte:     baseVersion.AnchorEndByte,
					Replacement: oursVersion.RawText,
				})
			} else if theirsChanged {
				edits = append(edits, Edit{
					Path:        path,
					StartByte:   baseVersion.AnchorStartByte,
					EndByte:     baseVersion.AnchorEndByte,
					Replacement: theirsVersion.RawText,
				})
			}
		case baseOK && !oursOK && !theirsOK:
			edits = append(edits, Edit{
				Path:        path,
				StartByte:   baseVersion.AnchorStartByte,
				EndByte:     baseVersion.AnchorEndByte,
				Replacement: nil,
			})
		case baseOK && oursOK && !theirsOK:
			if baseVersion.ContentHash != oursVersion.ContentHash {
				conflicts = append(conflicts, Conflict{Path: path, Symbol: key, Reason: "deleted in theirs but changed in ours"})
				continue
			}
			edits = append(edits, Edit{Path: path, StartByte: baseVersion.AnchorStartByte, EndByte: baseVersion.AnchorEndByte, Replacement: nil})
		case baseOK && !oursOK && theirsOK:
			if baseVersion.ContentHash != theirsVersion.ContentHash {
				conflicts = append(conflicts, Conflict{Path: path, Symbol: key, Reason: "deleted in ours but changed in theirs"})
				continue
			}
			edits = append(edits, Edit{Path: path, StartByte: baseVersion.AnchorStartByte, EndByte: baseVersion.AnchorEndByte, Replacement: nil})
		}
	}
	if len(conflicts) > 0 {
		return MergeResult{Conflicts: conflicts}, nil
	}
	materializer := NewMaterializer(registry)
	merged, err := materializer.ApplyEdits(ctx, path, base, edits)
	if err != nil {
		return MergeResult{}, err
	}
	return MergeResult{MergedSource: merged}, nil
}

func mapBySymbol(versions []model.SymbolVersion) map[string]model.SymbolVersion {
	out := make(map[string]model.SymbolVersion, len(versions))
	for _, version := range versions {
		out[version.SymbolID] = version
	}
	return out
}

func unionKeys(maps ...map[string]model.SymbolVersion) []string {
	seen := map[string]struct{}{}
	var keys []string
	for _, m := range maps {
		for key := range m {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	return keys
}
