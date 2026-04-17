package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yesabhishek/ada/internal/model"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			snapshot_id TEXT PRIMARY KEY,
			repo_id TEXT NOT NULL,
			git_commit TEXT NOT NULL,
			parent_snapshot_id TEXT,
			status TEXT NOT NULL,
			reflection TEXT NOT NULL,
			author TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_repo_created ON snapshots(repo_id, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS file_versions (
			file_version_id TEXT PRIMARY KEY,
			snapshot_id TEXT NOT NULL REFERENCES snapshots(snapshot_id) ON DELETE CASCADE,
			path TEXT NOT NULL,
			language TEXT NOT NULL,
			raw_text_hash TEXT NOT NULL,
			raw_text BLOB NOT NULL,
			parser_version TEXT NOT NULL,
			formatter_id TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_file_versions_snapshot ON file_versions(snapshot_id, path);`,
		`CREATE TABLE IF NOT EXISTS symbols (
			symbol_id TEXT PRIMARY KEY,
			repo_id TEXT NOT NULL,
			path TEXT NOT NULL,
			fq_name TEXT NOT NULL,
			language TEXT NOT NULL,
			kind TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_query ON symbols(repo_id, fq_name, path);`,
		`CREATE TABLE IF NOT EXISTS symbol_versions (
			snapshot_id TEXT NOT NULL REFERENCES snapshots(snapshot_id) ON DELETE CASCADE,
			symbol_id TEXT NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
			file_version_id TEXT NOT NULL REFERENCES file_versions(file_version_id) ON DELETE CASCADE,
			content_hash TEXT NOT NULL,
			signature_hash TEXT NOT NULL,
			start_byte INTEGER NOT NULL,
			end_byte INTEGER NOT NULL,
			anchor_start_byte INTEGER NOT NULL,
			anchor_end_byte INTEGER NOT NULL,
			node_type TEXT NOT NULL,
			normalized_body TEXT NOT NULL,
			raw_text BLOB NOT NULL,
			line_start INTEGER NOT NULL,
			line_end INTEGER NOT NULL,
			PRIMARY KEY(snapshot_id, symbol_id)
		);`,
		`CREATE TABLE IF NOT EXISTS node_blobs (
			content_hash TEXT PRIMARY KEY,
			language TEXT NOT NULL,
			node_type TEXT NOT NULL,
			normalized_body TEXT NOT NULL,
			schema_version TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS edges (
			snapshot_id TEXT NOT NULL REFERENCES snapshots(snapshot_id) ON DELETE CASCADE,
			from_symbol_id TEXT NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
			to_symbol_id TEXT NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
			edge_type TEXT NOT NULL,
			edge_source TEXT NOT NULL,
			confidence_score REAL NOT NULL,
			PRIMARY KEY(snapshot_id, from_symbol_id, to_symbol_id, edge_type)
		);`,
		`CREATE TABLE IF NOT EXISTS sync_runs (
			sync_run_id TEXT PRIMARY KEY,
			repo_id TEXT NOT NULL,
			git_commit TEXT NOT NULL,
			snapshot_id TEXT NOT NULL REFERENCES snapshots(snapshot_id) ON DELETE CASCADE,
			state TEXT NOT NULL,
			last_error TEXT NOT NULL,
			retry_count INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sync_runs_repo_updated ON sync_runs(repo_id, updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			task_id TEXT PRIMARY KEY,
			snapshot_id TEXT NOT NULL REFERENCES snapshots(snapshot_id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS proposals (
			proposal_id TEXT PRIMARY KEY,
			snapshot_id TEXT NOT NULL REFERENCES snapshots(snapshot_id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL,
			changed_symbols_json TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("run migration: %w", err)
		}
	}
	return nil
}

func (s *Store) SaveSnapshotBundle(ctx context.Context, bundle model.SnapshotBundle) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO snapshots(snapshot_id, repo_id, git_commit, parent_snapshot_id, status, reflection, author, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		bundle.Snapshot.ID,
		bundle.Snapshot.RepoID,
		bundle.Snapshot.GitCommit,
		bundle.Snapshot.ParentSnapshotID,
		string(bundle.Snapshot.Status),
		bundle.Snapshot.Reflection,
		bundle.Snapshot.Author,
		bundle.Snapshot.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}

	for _, fileVersion := range bundle.FileVersions {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO file_versions(file_version_id, snapshot_id, path, language, raw_text_hash, raw_text, parser_version, formatter_id)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			fileVersion.ID,
			fileVersion.SnapshotID,
			fileVersion.Path,
			fileVersion.Language,
			fileVersion.RawTextHash,
			fileVersion.RawText,
			fileVersion.ParserVersion,
			fileVersion.FormatterID,
		); err != nil {
			return fmt.Errorf("insert file version %s: %w", fileVersion.Path, err)
		}
	}

	for _, symbol := range bundle.Symbols {
		if _, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO symbols(symbol_id, repo_id, path, fq_name, language, kind)
			VALUES(?, ?, ?, ?, ?, ?)`,
			symbol.ID,
			symbol.RepoID,
			symbol.Path,
			symbol.FQName,
			symbol.Language,
			symbol.Kind,
		); err != nil {
			return fmt.Errorf("insert symbol %s: %w", symbol.FQName, err)
		}
	}

	for _, nodeBlob := range bundle.NodeBlobs {
		if _, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO node_blobs(content_hash, language, node_type, normalized_body, schema_version)
			VALUES(?, ?, ?, ?, ?)`,
			nodeBlob.ContentHash,
			nodeBlob.Language,
			nodeBlob.NodeType,
			nodeBlob.NormalizedBody,
			nodeBlob.SchemaVersion,
		); err != nil {
			return fmt.Errorf("insert node blob %s: %w", nodeBlob.ContentHash, err)
		}
	}

	for _, version := range bundle.SymbolVersions {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO symbol_versions(snapshot_id, symbol_id, file_version_id, content_hash, signature_hash, start_byte, end_byte, anchor_start_byte, anchor_end_byte, node_type, normalized_body, raw_text, line_start, line_end)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			version.SnapshotID,
			version.SymbolID,
			version.FileVersionID,
			version.ContentHash,
			version.SignatureHash,
			version.StartByte,
			version.EndByte,
			version.AnchorStartByte,
			version.AnchorEndByte,
			version.NodeType,
			version.NormalizedBody,
			version.RawText,
			version.LineStart,
			version.LineEnd,
		); err != nil {
			return fmt.Errorf("insert symbol version %s: %w", version.SymbolID, err)
		}
	}

	for _, edge := range bundle.Edges {
		if _, err = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO edges(snapshot_id, from_symbol_id, to_symbol_id, edge_type, edge_source, confidence_score)
			VALUES(?, ?, ?, ?, ?, ?)`,
			edge.SnapshotID,
			edge.FromSymbolID,
			edge.ToSymbolID,
			edge.EdgeType,
			edge.EdgeSource,
			edge.ConfidenceScore,
		); err != nil {
			return fmt.Errorf("insert edge %s->%s: %w", edge.FromSymbolID, edge.ToSymbolID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit snapshot bundle: %w", err)
	}
	return nil
}

func (s *Store) LatestSnapshot(ctx context.Context, repoID string) (*model.Snapshot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT snapshot_id, repo_id, git_commit, parent_snapshot_id, status, reflection, author, created_at
		FROM snapshots
		WHERE repo_id = ?
		ORDER BY created_at DESC
		LIMIT 1`, repoID)
	return scanSnapshot(row)
}

func (s *Store) SnapshotByID(ctx context.Context, snapshotID string) (*model.Snapshot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT snapshot_id, repo_id, git_commit, parent_snapshot_id, status, reflection, author, created_at
		FROM snapshots
		WHERE snapshot_id = ?`, snapshotID)
	return scanSnapshot(row)
}

func (s *Store) ListSnapshots(ctx context.Context, repoID string, limit int) ([]model.Snapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT snapshot_id, repo_id, git_commit, parent_snapshot_id, status, reflection, author, created_at
		FROM snapshots
		WHERE repo_id = ?
		ORDER BY created_at DESC
		LIMIT ?`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	var snapshots []model.Snapshot
	for rows.Next() {
		snapshot, err := scanSnapshotFromRows(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *Store) ListSnapshotCandidates(ctx context.Context, repoID string, query string, limit int) ([]model.SnapshotCandidate, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT snapshot_id, git_commit, reflection, created_at
		FROM snapshots
		WHERE repo_id = ?
		  AND (LOWER(reflection) LIKE ? OR LOWER(git_commit) LIKE ?)
		ORDER BY created_at DESC
		LIMIT ?`, repoID, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("list snapshot candidates: %w", err)
	}
	defer rows.Close()
	var candidates []model.SnapshotCandidate
	for rows.Next() {
		var candidate model.SnapshotCandidate
		if err := rows.Scan(&candidate.ID, &candidate.GitCommit, &candidate.Reflection, &candidate.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (s *Store) FileVersionsBySnapshot(ctx context.Context, snapshotID string) ([]model.FileVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT file_version_id, snapshot_id, path, language, raw_text_hash, raw_text, parser_version, formatter_id
		FROM file_versions
		WHERE snapshot_id = ?
		ORDER BY path`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("list file versions: %w", err)
	}
	defer rows.Close()
	var versions []model.FileVersion
	for rows.Next() {
		var version model.FileVersion
		if err := rows.Scan(
			&version.ID,
			&version.SnapshotID,
			&version.Path,
			&version.Language,
			&version.RawTextHash,
			&version.RawText,
			&version.ParserVersion,
			&version.FormatterID,
		); err != nil {
			return nil, fmt.Errorf("scan file version: %w", err)
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) SymbolCatalog(ctx context.Context, repoID string) (map[string]model.Symbol, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT symbol_id, repo_id, path, fq_name, language, kind
		FROM symbols
		WHERE repo_id = ?`, repoID)
	if err != nil {
		return nil, fmt.Errorf("query symbol catalog: %w", err)
	}
	defer rows.Close()
	catalog := make(map[string]model.Symbol)
	for rows.Next() {
		var symbol model.Symbol
		if err := rows.Scan(&symbol.ID, &symbol.RepoID, &symbol.Path, &symbol.FQName, &symbol.Language, &symbol.Kind); err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		catalog[symbol.ID] = symbol
	}
	return catalog, rows.Err()
}

func (s *Store) SymbolVersionsBySnapshot(ctx context.Context, snapshotID string) ([]model.SymbolVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT snapshot_id, symbol_id, file_version_id, content_hash, signature_hash, start_byte, end_byte, anchor_start_byte, anchor_end_byte, node_type, normalized_body, raw_text, line_start, line_end
		FROM symbol_versions
		WHERE snapshot_id = ?`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("query symbol versions: %w", err)
	}
	defer rows.Close()
	var versions []model.SymbolVersion
	for rows.Next() {
		var version model.SymbolVersion
		if err := rows.Scan(
			&version.SnapshotID,
			&version.SymbolID,
			&version.FileVersionID,
			&version.ContentHash,
			&version.SignatureHash,
			&version.StartByte,
			&version.EndByte,
			&version.AnchorStartByte,
			&version.AnchorEndByte,
			&version.NodeType,
			&version.NormalizedBody,
			&version.RawText,
			&version.LineStart,
			&version.LineEnd,
		); err != nil {
			return nil, fmt.Errorf("scan symbol version: %w", err)
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) CreateOrUpdateSyncRun(ctx context.Context, syncRun model.SyncRun) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_runs(sync_run_id, repo_id, git_commit, snapshot_id, state, last_error, retry_count, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(sync_run_id) DO UPDATE SET
			state = excluded.state,
			last_error = excluded.last_error,
			retry_count = excluded.retry_count,
			updated_at = excluded.updated_at`,
		syncRun.ID,
		syncRun.RepoID,
		syncRun.GitCommit,
		syncRun.SnapshotID,
		string(syncRun.State),
		syncRun.LastError,
		syncRun.RetryCount,
		syncRun.CreatedAt,
		syncRun.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert sync run: %w", err)
	}
	return nil
}

func (s *Store) UpdateSnapshotStatus(ctx context.Context, snapshotID string, status model.SnapshotStatus, reflection string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE snapshots
		SET status = ?, reflection = ?
		WHERE snapshot_id = ?`, string(status), reflection, snapshotID)
	if err != nil {
		return fmt.Errorf("update snapshot status: %w", err)
	}
	return nil
}

func (s *Store) RecentSyncRuns(ctx context.Context, repoID string, limit int) ([]model.SyncRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sync_run_id, repo_id, git_commit, snapshot_id, state, last_error, retry_count, created_at, updated_at
		FROM sync_runs
		WHERE repo_id = ?
		ORDER BY updated_at DESC
		LIMIT ?`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("query sync runs: %w", err)
	}
	defer rows.Close()
	var runs []model.SyncRun
	for rows.Next() {
		var run model.SyncRun
		var state string
		if err := rows.Scan(&run.ID, &run.RepoID, &run.GitCommit, &run.SnapshotID, &state, &run.LastError, &run.RetryCount, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sync run: %w", err)
		}
		run.State = model.SnapshotStatus(state)
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, task model.Task) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks(task_id, snapshot_id, title, status, created_at)
		VALUES(?, ?, ?, ?, ?)`,
		task.ID, task.SnapshotID, task.Title, task.Status, task.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (s *Store) CreateProposal(ctx context.Context, proposal model.Proposal) error {
	changedSymbols, err := json.Marshal(proposal.ChangedSymbols)
	if err != nil {
		return fmt.Errorf("marshal proposal symbols: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO proposals(proposal_id, snapshot_id, title, status, summary, changed_symbols_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		proposal.ID, proposal.SnapshotID, proposal.Title, proposal.Status, proposal.Summary, string(changedSymbols), proposal.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create proposal: %w", err)
	}
	return nil
}

func (s *Store) LatestProposal(ctx context.Context, snapshotID string) (*model.Proposal, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT proposal_id, snapshot_id, title, status, summary, changed_symbols_json, created_at
		FROM proposals
		WHERE snapshot_id = ?
		ORDER BY created_at DESC
		LIMIT 1`, snapshotID)
	var proposal model.Proposal
	var changedSymbols string
	if err := row.Scan(&proposal.ID, &proposal.SnapshotID, &proposal.Title, &proposal.Status, &proposal.Summary, &changedSymbols, &proposal.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest proposal: %w", err)
	}
	if err := json.Unmarshal([]byte(changedSymbols), &proposal.ChangedSymbols); err != nil {
		return nil, fmt.Errorf("unmarshal proposal symbols: %w", err)
	}
	return &proposal, nil
}

func (s *Store) UpdateProposalStatus(ctx context.Context, proposalID string, status string, summary string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE proposals
		SET status = ?, summary = ?
		WHERE proposal_id = ?`, status, summary, proposalID)
	if err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}
	return nil
}

func (s *Store) SearchSymbols(ctx context.Context, repoID string, query string, limit int) ([]model.Symbol, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT symbol_id, repo_id, path, fq_name, language, kind
		FROM symbols
		WHERE repo_id = ?
		  AND (LOWER(fq_name) LIKE ? OR LOWER(path) LIKE ? OR LOWER(kind) LIKE ?)
		ORDER BY fq_name
		LIMIT ?`, repoID, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search symbols: %w", err)
	}
	defer rows.Close()
	var symbols []model.Symbol
	for rows.Next() {
		var symbol model.Symbol
		if err := rows.Scan(&symbol.ID, &symbol.RepoID, &symbol.Path, &symbol.FQName, &symbol.Language, &symbol.Kind); err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}
	return symbols, rows.Err()
}

func (s *Store) ListTasks(ctx context.Context, repoID string, limit int) ([]model.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.task_id, t.snapshot_id, t.title, t.status, t.created_at
		FROM tasks t
		INNER JOIN snapshots s ON s.snapshot_id = t.snapshot_id
		WHERE s.repo_id = ?
		ORDER BY t.created_at DESC
		LIMIT ?`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	var tasks []model.Task
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(&task.ID, &task.SnapshotID, &task.Title, &task.Status, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) ListProposals(ctx context.Context, repoID string, limit int) ([]model.Proposal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.proposal_id, p.snapshot_id, p.title, p.status, p.summary, p.changed_symbols_json, p.created_at
		FROM proposals p
		INNER JOIN snapshots s ON s.snapshot_id = p.snapshot_id
		WHERE s.repo_id = ?
		ORDER BY p.created_at DESC
		LIMIT ?`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	defer rows.Close()
	var proposals []model.Proposal
	for rows.Next() {
		var proposal model.Proposal
		var changedSymbols string
		if err := rows.Scan(&proposal.ID, &proposal.SnapshotID, &proposal.Title, &proposal.Status, &proposal.Summary, &changedSymbols, &proposal.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		if err := json.Unmarshal([]byte(changedSymbols), &proposal.ChangedSymbols); err != nil {
			return nil, fmt.Errorf("unmarshal proposal symbols: %w", err)
		}
		proposals = append(proposals, proposal)
	}
	return proposals, rows.Err()
}

func scanSnapshot(row interface{ Scan(dest ...any) error }) (*model.Snapshot, error) {
	var snapshot model.Snapshot
	var parent sql.NullString
	var status string
	if err := row.Scan(&snapshot.ID, &snapshot.RepoID, &snapshot.GitCommit, &parent, &status, &snapshot.Reflection, &snapshot.Author, &snapshot.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan snapshot: %w", err)
	}
	if parent.Valid {
		snapshot.ParentSnapshotID = &parent.String
	}
	snapshot.Status = model.SnapshotStatus(status)
	return &snapshot, nil
}

func scanSnapshotFromRows(rows *sql.Rows) (model.Snapshot, error) {
	var snapshot model.Snapshot
	var parent sql.NullString
	var status string
	if err := rows.Scan(&snapshot.ID, &snapshot.RepoID, &snapshot.GitCommit, &parent, &status, &snapshot.Reflection, &snapshot.Author, &snapshot.CreatedAt); err != nil {
		return model.Snapshot{}, fmt.Errorf("scan snapshot row: %w", err)
	}
	if parent.Valid {
		snapshot.ParentSnapshotID = &parent.String
	}
	snapshot.Status = model.SnapshotStatus(status)
	return snapshot, nil
}
