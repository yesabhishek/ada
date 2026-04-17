package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/yesabhishek/ada/internal/language"
	golanguage "github.com/yesabhishek/ada/internal/language/golang"
)

func TestApplyEditsPreservesUntouchedTriviaAndFormatsGo(t *testing.T) {
	source := []byte(`package main

// add keeps its doc comment.
func add(a int, b int) int {
    return a + b
}

func untouched() int {
	return 1
}
`)

	registry := language.NewRegistry(golanguage.New())
	materializer := NewMaterializer(registry)
	start := strings.Index(string(source), "func add")
	end := strings.Index(string(source), "\n\nfunc untouched")
	edited, err := materializer.ApplyEdits(context.Background(), "main.go", source, []Edit{{
		Path:        "main.go",
		StartByte:   start,
		EndByte:     end,
		Replacement: []byte("func add(a int, b int) int {\n\treturn a - b\n}\n"),
	}})
	if err != nil {
		t.Fatalf("ApplyEdits() error = %v", err)
	}
	got := string(edited)
	if !strings.Contains(got, "// add keeps its doc comment.") {
		t.Fatalf("expected comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "return a - b") {
		t.Fatalf("expected replacement body, got:\n%s", got)
	}
	if !strings.Contains(got, "func untouched() int {\n\treturn 1\n}") {
		t.Fatalf("expected untouched function to remain stable, got:\n%s", got)
	}
}
