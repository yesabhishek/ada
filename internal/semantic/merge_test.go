package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/yesabhishek/ada/internal/language"
	golanguage "github.com/yesabhishek/ada/internal/language/golang"
	"github.com/yesabhishek/ada/internal/model"
)

func TestThreeWayMergeMergesDisjointFunctions(t *testing.T) {
	base := []byte(`package main

func alpha() int {
	return 1
}

func beta() int {
	return 2
}
`)
	ours := []byte(`package main

func alpha() int {
	return 10
}

func beta() int {
	return 2
}
`)
	theirs := []byte(`package main

func alpha() int {
	return 1
}

func beta() int {
	return 20
}
`)

	registry := language.NewRegistry(golanguage.New())
	result, err := ThreeWayMerge(context.Background(), registry, "main.go", base, parseSymbolVersions(t, "main.go", base), parseSymbolVersions(t, "main.go", ours), parseSymbolVersions(t, "main.go", theirs))
	if err != nil {
		t.Fatalf("ThreeWayMerge() error = %v", err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %#v", result.Conflicts)
	}
	merged := string(result.MergedSource)
	if !containsAll(merged, "return 10", "return 20") {
		t.Fatalf("expected both edits in merge result, got:\n%s", merged)
	}
}

func TestThreeWayMergeDetectsSameSymbolConflict(t *testing.T) {
	base := []byte(`package main

func alpha() int {
	return 1
}
`)
	ours := []byte(`package main

func alpha() int {
	return 10
}
`)
	theirs := []byte(`package main

func alpha() int {
	return 20
}
`)

	registry := language.NewRegistry(golanguage.New())
	result, err := ThreeWayMerge(context.Background(), registry, "main.go", base, parseSymbolVersions(t, "main.go", base), parseSymbolVersions(t, "main.go", ours), parseSymbolVersions(t, "main.go", theirs))
	if err != nil {
		t.Fatalf("ThreeWayMerge() error = %v", err)
	}
	if len(result.Conflicts) != 1 {
		t.Fatalf("expected one conflict, got %#v", result.Conflicts)
	}
}

func parseSymbolVersions(t *testing.T, path string, source []byte) []model.SymbolVersion {
	t.Helper()
	parsed, err := golanguage.New().Parse(context.Background(), path, source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var versions []model.SymbolVersion
	for _, symbol := range parsed.Symbols {
		versions = append(versions, model.SymbolVersion{
			SymbolID:        symbol.FQName,
			ContentHash:     model.HashText(symbol.NormalizedBody),
			AnchorStartByte: symbol.AnchorStartByte,
			AnchorEndByte:   symbol.AnchorEndByte,
			RawText:         symbol.RawText,
		})
	}
	return versions
}

func containsAll(text string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(text, sub) {
			return false
		}
	}
	return true
}
