package typescript

import (
	"context"
	"testing"
)

func TestParseTypeScriptFileExtractsClassAndMethodSymbols(t *testing.T) {
	source := []byte(`import { helper } from "./helper";

export function add(): number {
  return helper() + 1;
}

class User {
  getName() {
    return add();
  }
}
`)

	parsed, err := New().Parse(context.Background(), "main.ts", source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := len(parsed.Symbols); got != 3 {
		t.Fatalf("expected 3 symbols, got %d", got)
	}
	if parsed.Symbols[2].FQName != "User.getName" {
		t.Fatalf("expected class method fq name, got %s", parsed.Symbols[2].FQName)
	}
	if len(parsed.Edges) == 0 {
		t.Fatalf("expected at least one dependency edge")
	}
}
