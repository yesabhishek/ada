package golang

import (
	"context"
	"fmt"
	"go/format"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	treego "github.com/tree-sitter/tree-sitter-go/bindings/go"
	"github.com/yesabhishek/ada/internal/language"
	"github.com/yesabhishek/ada/internal/model"
)

const parserVersion = "tree-sitter-go@0.25.0"

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Language() string {
	return "go"
}

func (a *Adapter) Supports(path string) bool {
	return language.SupportsTextFile(path, ".go")
}

func (a *Adapter) Parse(ctx context.Context, path string, source []byte) (*language.ParsedFile, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(treego.Language())); err != nil {
		return nil, fmt.Errorf("set go parser language: %w", err)
	}
	tree := parser.ParseCtx(ctx, source, nil)
	defer tree.Close()
	root := tree.RootNode()
	parsed := &language.ParsedFile{
		Path:          path,
		Language:      a.Language(),
		ParserVersion: parserVersion,
		FormatterID:   "gofmt",
		RootKind:      root.Kind(),
	}

	cursor := root.Walk()
	defer cursor.Close()
	children := root.NamedChildren(cursor)
	for _, child := range children {
		switch child.Kind() {
		case "function_declaration":
			parsed.Symbols = append(parsed.Symbols, buildGoSymbol(path, source, child, "function", ""))
		case "method_declaration":
			receiver := ""
			if recv := child.ChildByFieldName("receiver"); recv != nil {
				receiver = extractReceiverName(recv.Utf8Text(source))
			}
			parsed.Symbols = append(parsed.Symbols, buildGoSymbol(path, source, child, "method", receiver))
		case "type_declaration":
			typeCursor := child.Walk()
			for _, spec := range child.NamedChildren(typeCursor) {
				if spec.Kind() != "type_spec" {
					continue
				}
				parsed.Symbols = append(parsed.Symbols, buildGoTypeSpec(path, source, &spec))
			}
			typeCursor.Close()
		}
	}

	parsed.Edges = buildGoEdges(parsed.Symbols)
	return parsed, nil
}

func (a *Adapter) Format(_ context.Context, _ string, source []byte) ([]byte, error) {
	formatted, err := format.Source(source)
	if err != nil {
		return nil, fmt.Errorf("gofmt source: %w", err)
	}
	return formatted, nil
}

func buildGoSymbol(path string, source []byte, node sitter.Node, kind string, receiver string) language.ParsedSymbol {
	nameNode := node.ChildByFieldName("name")
	name := nameNode.Utf8Text(source)
	fqName := name
	if receiver != "" {
		fqName = receiver + "." + name
	}
	start := int(node.StartByte())
	end := int(node.EndByte())
	anchorStart, anchorEnd := language.ExpandAnchor(source, start, end)
	signature := node.Utf8Text(source)
	if body := node.ChildByFieldName("body"); body != nil {
		signature = strings.TrimSpace(string(source[start:int(body.StartByte())]))
	}
	return language.ParsedSymbol{
		Name:            name,
		FQName:          fqName,
		Kind:            kind,
		NodeType:        node.Kind(),
		Signature:       strings.TrimSpace(signature),
		NormalizedBody:  model.NormalizeWhitespace(node.Utf8Text(source)),
		RawText:         cloneBytes(source[anchorStart:anchorEnd]),
		StartByte:       start,
		EndByte:         end,
		AnchorStartByte: anchorStart,
		AnchorEndByte:   anchorEnd,
		LineStart:       language.LineNumberAt(source, start),
		LineEnd:         language.LineNumberAt(source, end),
	}
}

func buildGoTypeSpec(path string, source []byte, node *sitter.Node) language.ParsedSymbol {
	nameNode := node.ChildByFieldName("name")
	name := nameNode.Utf8Text(source)
	start := int(node.StartByte())
	end := int(node.EndByte())
	anchorStart, anchorEnd := language.ExpandAnchor(source, start, end)
	return language.ParsedSymbol{
		Name:            name,
		FQName:          name,
		Kind:            "type",
		NodeType:        node.Kind(),
		Signature:       strings.TrimSpace(node.Utf8Text(source)),
		NormalizedBody:  model.NormalizeWhitespace(node.Utf8Text(source)),
		RawText:         cloneBytes(source[anchorStart:anchorEnd]),
		StartByte:       start,
		EndByte:         end,
		AnchorStartByte: anchorStart,
		AnchorEndByte:   anchorEnd,
		LineStart:       language.LineNumberAt(source, start),
		LineEnd:         language.LineNumberAt(source, end),
	}
}

func buildGoEdges(symbols []language.ParsedSymbol) []language.ParsedEdge {
	lookup := make(map[string]string, len(symbols))
	for _, symbol := range symbols {
		lookup[symbol.Name] = symbol.FQName
	}
	seen := make(map[string]struct{})
	var edges []language.ParsedEdge
	for _, symbol := range symbols {
		body := string(symbol.RawText)
		for name, target := range lookup {
			if target == symbol.FQName {
				continue
			}
			if language.ContainsIdentifier(body, name) {
				key := symbol.FQName + "->" + target
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				edges = append(edges, language.ParsedEdge{
					FromFQName:      symbol.FQName,
					ToFQName:        target,
					EdgeType:        "reference",
					EdgeSource:      "parser",
					ConfidenceScore: 0.85,
				})
			}
		}
	}
	return edges
}

func extractReceiverName(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "(")
	raw = strings.TrimSuffix(raw, ")")
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "receiver"
	}
	return strings.TrimPrefix(fields[len(fields)-1], "*")
}

func cloneBytes(in []byte) []byte {
	return append([]byte(nil), in...)
}
