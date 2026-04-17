package controlplane

import "time"

type Manifest struct {
	RepoID     string       `json:"repo_id"`
	GitCommit  string       `json:"git_commit"`
	SnapshotID string       `json:"snapshot_id"`
	CreatedAt  time.Time    `json:"created_at"`
	Files      []FileDigest `json:"files"`
}

type FileDigest struct {
	Path        string `json:"path"`
	Language    string `json:"language"`
	RawTextHash string `json:"raw_text_hash"`
}

type MissingCommit struct {
	RepoID    string
	GitCommit string
}
