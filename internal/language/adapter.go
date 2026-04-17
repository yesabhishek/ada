package language

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yesabhishek/ada/internal/model"
)

type Adapter interface {
	Language() string
	Supports(path string) bool
	Parse(ctx context.Context, path string, source []byte) (*ParsedFile, error)
	Format(ctx context.Context, path string, source []byte) ([]byte, error)
}

type ParsedFile struct {
	Path          string
	Language      string
	ParserVersion string
	FormatterID   string
	RootKind      string
	Symbols       []ParsedSymbol
	Edges         []ParsedEdge
}

type ParsedSymbol struct {
	Name            string
	FQName          string
	Kind            string
	NodeType        string
	Signature       string
	NormalizedBody  string
	RawText         []byte
	StartByte       int
	EndByte         int
	AnchorStartByte int
	AnchorEndByte   int
	LineStart       int
	LineEnd         int
}

type ParsedEdge struct {
	FromFQName      string
	ToFQName        string
	EdgeType        string
	EdgeSource      string
	ConfidenceScore float64
}

type Registry struct {
	adapters []Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	return &Registry{adapters: adapters}
}

func (r *Registry) All() []Adapter {
	return append([]Adapter(nil), r.adapters...)
}

func (r *Registry) ForPath(path string) Adapter {
	for _, adapter := range r.adapters {
		if adapter.Supports(path) {
			return adapter
		}
	}
	return nil
}

func SupportsTextFile(path string, extensions ...string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, candidate := range extensions {
		if ext == candidate {
			return true
		}
	}
	return false
}

func ExpandAnchor(source []byte, startByte, endByte int) (int, int) {
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(source) {
		endByte = len(source)
	}
	start := lineStart(source, startByte)
	end := lineEnd(source, endByte)

	commentStart := start
	for commentStart > 0 {
		prevLineEnd := commentStart - 1
		prevLineStart := lineStart(source, prevLineEnd)
		line := strings.TrimSpace(string(source[prevLineStart:prevLineEnd]))
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "*/") {
			commentStart = prevLineStart
			continue
		}
		break
	}
	return commentStart, end
}

func lineStart(source []byte, index int) int {
	if index > len(source) {
		index = len(source)
	}
	for index > 0 && source[index-1] != '\n' {
		index--
	}
	return index
}

func lineEnd(source []byte, index int) int {
	if index > len(source) {
		index = len(source)
	}
	for index < len(source) && source[index] != '\n' {
		index++
	}
	if index < len(source) {
		index++
	}
	return index
}

func LineNumberAt(source []byte, index int) int {
	if index <= 0 {
		return 1
	}
	if index > len(source) {
		index = len(source)
	}
	return bytes.Count(source[:index], []byte("\n")) + 1
}

func SymbolID(repoID, path, languageName, kind, fqName string) string {
	return model.HashText(repoID, path, languageName, kind, fqName)
}

func FileVersionID(snapshotID, path string) string {
	return model.HashText(snapshotID, path)
}

func SnapshotReflection(fileCount, symbolCount int) string {
	return fmt.Sprintf("Indexed %d supported files and %d semantic symbols.", fileCount, symbolCount)
}

func ContainsIdentifier(body string, identifier string) bool {
	pattern := `(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(identifier) + `([^A-Za-z0-9_]|$)`
	return regexp.MustCompile(pattern).FindStringIndex(body) != nil
}
