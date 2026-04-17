package model

import "time"

type SnapshotStatus string

const (
	SnapshotStatusIndexed          SnapshotStatus = "indexed"
	SnapshotStatusSynced           SnapshotStatus = "synced"
	SnapshotStatusPendingReconcile SnapshotStatus = "pending_reconcile"
	SnapshotStatusReindexing       SnapshotStatus = "reindexing"
	SnapshotStatusFailedLocalIndex SnapshotStatus = "failed_local_index"
	SnapshotStatusFailedGitPush    SnapshotStatus = "failed_git_push"
)

type Snapshot struct {
	ID               string
	RepoID           string
	GitCommit        string
	ParentSnapshotID *string
	Status           SnapshotStatus
	Reflection       string
	Author           string
	CreatedAt        time.Time
}

type FileVersion struct {
	ID            string
	SnapshotID    string
	Path          string
	Language      string
	RawTextHash   string
	RawText       []byte
	ParserVersion string
	FormatterID   string
}

type Symbol struct {
	ID       string
	RepoID   string
	Path     string
	FQName   string
	Language string
	Kind     string
}

type SymbolVersion struct {
	SnapshotID      string
	SymbolID        string
	FileVersionID   string
	ContentHash     string
	SignatureHash   string
	StartByte       int
	EndByte         int
	AnchorStartByte int
	AnchorEndByte   int
	NodeType        string
	NormalizedBody  string
	RawText         []byte
	LineStart       int
	LineEnd         int
}

type NodeBlob struct {
	ContentHash    string
	Language       string
	NodeType       string
	NormalizedBody string
	SchemaVersion  string
}

type Edge struct {
	SnapshotID      string
	FromSymbolID    string
	ToSymbolID      string
	EdgeType        string
	EdgeSource      string
	ConfidenceScore float64
}

type SyncRun struct {
	ID         string
	RepoID     string
	GitCommit  string
	SnapshotID string
	State      SnapshotStatus
	LastError  string
	RetryCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Task struct {
	ID         string
	SnapshotID string
	Title      string
	Status     string
	CreatedAt  time.Time
}

type Proposal struct {
	ID             string
	SnapshotID     string
	Title          string
	Status         string
	Summary        string
	ChangedSymbols []string
	CreatedAt      time.Time
}

type SnapshotBundle struct {
	Snapshot       Snapshot
	FileVersions   []FileVersion
	Symbols        []Symbol
	SymbolVersions []SymbolVersion
	NodeBlobs      []NodeBlob
	Edges          []Edge
}

type SemanticChangeKind string

const (
	SemanticChangeAdded    SemanticChangeKind = "added"
	SemanticChangeModified SemanticChangeKind = "modified"
	SemanticChangeDeleted  SemanticChangeKind = "deleted"
)

type SemanticChange struct {
	Path          string
	SymbolID      string
	FQName        string
	Kind          string
	Language      string
	ChangeKind    SemanticChangeKind
	BeforeHash    string
	AfterHash     string
	BeforeVersion *SymbolVersion
	AfterVersion  *SymbolVersion
}

type SnapshotCandidate struct {
	ID         string
	GitCommit  string
	Reflection string
	CreatedAt  time.Time
}
