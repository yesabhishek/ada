package typescript

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"github.com/yesabhishek/ada/internal/language"
	"github.com/yesabhishek/ada/internal/model"
)

const parserVersion = "tree-sitter-typescript@0.23.2"

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Language() string {
	return "typescript"
}

func (a *Adapter) Supports(path string) bool {
	return language.SupportsTextFile(path, ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs")
}

func (a *Adapter) Parse(ctx context.Context, path string, source []byte) (*language.ParsedFile, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(languageForPath(path))); err != nil {
		return nil, fmt.Errorf("set typescript parser language: %w", err)
	}
	tree := parser.ParseCtx(ctx, source, nil)
	defer tree.Close()
	root := tree.RootNode()
	parsed := &language.ParsedFile{
		Path:          path,
		Language:      a.Language(),
		ParserVersion: parserVersion,
		FormatterID:   "prettier-compatible",
		RootKind:      root.Kind(),
	}
	cursor := root.Walk()
	defer cursor.Close()
	for _, child := range root.NamedChildren(cursor) {
		collectTSSymbols(source, child, "", &parsed.Symbols)
	}
	parsed.Edges = buildTSEdges(parsed.Symbols)
	return parsed, nil
}

func (a *Adapter) Format(ctx context.Context, path string, source []byte) ([]byte, error) {
	prettierPath, err := exec.LookPath("prettier")
	if err != nil {
		return source, nil
	}
	ext := filepath.Ext(path)
	parserName := "typescript"
	if ext == ".tsx" || ext == ".jsx" {
		parserName = "tsx"
	} else if ext == ".js" || ext == ".mjs" || ext == ".cjs" {
		parserName = "babel"
	}
	cmd := exec.CommandContext(ctx, prettierPath, "--stdin-filepath", path, "--parser", parserName)
	cmd.Stdin = strings.NewReader(string(source))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run prettier: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func collectTSSymbols(source []byte, node sitter.Node, parentClass string, out *[]language.ParsedSymbol) {
	switch node.Kind() {
	case "export_statement":
		if declaration := node.ChildByFieldName("declaration"); declaration != nil {
			collectTSSymbols(source, *declaration, parentClass, out)
		}
		return
	case "function_declaration":
		*out = append(*out, buildTSSymbol(source, node, "function", "", ""))
		return
	case "class_declaration":
		className := safeNodeText(source, node.ChildByFieldName("name"))
		*out = append(*out, buildTSSymbol(source, node, "class", className, ""))
		if body := node.ChildByFieldName("body"); body != nil {
			bodyCursor := body.Walk()
			for _, child := range body.NamedChildren(bodyCursor) {
				if child.Kind() == "method_definition" {
					*out = append(*out, buildTSSymbol(source, child, "method", safeNodeText(source, child.ChildByFieldName("name")), className))
				}
			}
			bodyCursor.Close()
		}
		return
	case "lexical_declaration", "variable_declaration":
		varCursor := node.Walk()
		for _, child := range node.NamedChildren(varCursor) {
			if child.Kind() == "variable_declarator" {
				*out = append(*out, buildTSSymbol(source, child, "variable", safeNodeText(source, child.ChildByFieldName("name")), ""))
			}
		}
		varCursor.Close()
		return
	}
}

func buildTSSymbol(source []byte, node sitter.Node, kind, name, parentClass string) language.ParsedSymbol {
	if name == "" {
		name = safeNodeText(source, node.ChildByFieldName("name"))
	}
	fqName := name
	if parentClass != "" {
		fqName = parentClass + "." + name
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
		RawText:         append([]byte(nil), source[anchorStart:anchorEnd]...),
		StartByte:       start,
		EndByte:         end,
		AnchorStartByte: anchorStart,
		AnchorEndByte:   anchorEnd,
		LineStart:       language.LineNumberAt(source, start),
		LineEnd:         language.LineNumberAt(source, end),
	}
}

func buildTSEdges(symbols []language.ParsedSymbol) []language.ParsedEdge {
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
					ConfidenceScore: 0.80,
				})
			}
		}
	}
	return edges
}

func safeNodeText(source []byte, node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return node.Utf8Text(source)
}

func languageForPath(path string) unsafe.Pointer {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".tsx" || ext == ".jsx" {
		return tstypescript.LanguageTSX()
	}
	return tstypescript.LanguageTypescript()
}
