package golang

import (
	"context"
	"testing"
)

func TestParseGoFileExtractsSymbolsAndEdges(t *testing.T) {
	source := []byte(`package main

type User struct{}

// helper performs the base addition.
func helper() int {
	return 41
}

// add keeps this doc comment attached to the function.
func add() int {
	return helper() + 1
}
`)

	parsed, err := New().Parse(context.Background(), "main.go", source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := len(parsed.Symbols); got != 3 {
		t.Fatalf("expected 3 symbols, got %d", got)
	}
	if parsed.Symbols[2].FQName != "add" {
		t.Fatalf("expected add symbol, got %s", parsed.Symbols[2].FQName)
	}
	if parsed.Symbols[2].AnchorStartByte >= parsed.Symbols[2].StartByte {
		t.Fatalf("expected anchor to expand to include the comment line")
	}
	if len(parsed.Edges) != 1 || parsed.Edges[0].ToFQName != "helper" {
		t.Fatalf("expected helper dependency edge, got %#v", parsed.Edges)
	}
}
