package ui

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/yesabhishek/ada/internal/gitutil"
	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/semantic"
	"github.com/yesabhishek/ada/internal/store"
	"github.com/yesabhishek/ada/internal/workspace"
)

type Server struct {
	workspace *workspace.Workspace
	store     *store.Store
	diff      *semantic.DiffService
	author    string
}

func New(workspace *workspace.Workspace, store *store.Store, diff *semantic.DiffService, author string) *Server {
	return &Server{
		workspace: workspace,
		store:     store,
		diff:      diff,
		author:    author,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/dashboard", s.handleDashboard)
	return mux
}

type dashboardPayload struct {
	GeneratedAt     time.Time            `json:"generated_at"`
	Workspace       workspaceView        `json:"workspace"`
	Git             gitView              `json:"git"`
	LatestSnapshot  *snapshotView        `json:"latest_snapshot,omitempty"`
	RecentSnapshots []snapshotView       `json:"recent_snapshots"`
	SyncRuns        []syncRunView        `json:"sync_runs"`
	Tasks           []taskView           `json:"tasks"`
	Proposals       []proposalView       `json:"proposals"`
	SemanticChanges []semanticChangeView `json:"semantic_changes"`
	SemanticCounts  semanticSummary      `json:"semantic_counts"`
	TextDiff        string               `json:"text_diff"`
	Warnings        []string             `json:"warnings"`
}

type workspaceView struct {
	Root      string `json:"root"`
	RepoID    string `json:"repo_id"`
	RemoteURL string `json:"remote_url,omitempty"`
}

type gitView struct {
	Present       bool     `json:"present"`
	Branch        string   `json:"branch,omitempty"`
	Head          string   `json:"head,omitempty"`
	Clean         bool     `json:"clean"`
	StatusEntries []string `json:"status_entries,omitempty"`
	Error         string   `json:"error,omitempty"`
}

type semanticChangeView struct {
	Path       string `json:"path"`
	FQName     string `json:"fq_name"`
	Kind       string `json:"kind"`
	Language   string `json:"language"`
	ChangeKind string `json:"change_kind"`
}

type semanticSummary struct {
	Added    int `json:"added"`
	Modified int `json:"modified"`
	Deleted  int `json:"deleted"`
}

type snapshotView struct {
	ID               string    `json:"id"`
	RepoID           string    `json:"repo_id"`
	GitCommit        string    `json:"git_commit"`
	ParentSnapshotID *string   `json:"parent_snapshot_id,omitempty"`
	Status           string    `json:"status"`
	Reflection       string    `json:"reflection"`
	Author           string    `json:"author"`
	CreatedAt        time.Time `json:"created_at"`
}

type syncRunView struct {
	ID         string    `json:"id"`
	RepoID     string    `json:"repo_id"`
	GitCommit  string    `json:"git_commit"`
	SnapshotID string    `json:"snapshot_id"`
	State      string    `json:"state"`
	LastError  string    `json:"last_error"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type taskView struct {
	ID         string    `json:"id"`
	SnapshotID string    `json:"snapshot_id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type proposalView struct {
	ID             string    `json:"id"`
	SnapshotID     string    `json:"snapshot_id"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	Summary        string    `json:"summary"`
	ChangedSymbols []string  `json:"changed_symbols"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = dashboardPage.Execute(w, map[string]any{
		"Title": "Ada UI",
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	payload := s.loadDashboard(r.Context())
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func (s *Server) loadDashboard(ctx context.Context) dashboardPayload {
	payload := dashboardPayload{
		GeneratedAt: time.Now().UTC(),
		Workspace: workspaceView{
			Root:      s.workspace.Root,
			RepoID:    s.workspace.Config.RepoID,
			RemoteURL: s.workspace.Config.RemoteURL,
		},
	}
	if repo, err := gitutil.Discover(ctx, s.workspace.Root); err == nil {
		payload.Git.Present = true
		payload.Git.Branch, _ = repo.CurrentBranch(ctx)
		payload.Git.Head, _ = repo.HeadCommit(ctx)
		payload.Git.StatusEntries, _ = repo.StatusEntries(ctx)
		payload.Git.Clean = len(payload.Git.StatusEntries) == 0
	} else {
		payload.Git.Error = err.Error()
	}

	latestSnapshot, err := s.store.LatestSnapshot(ctx, s.workspace.Config.RepoID)
	if err != nil {
		payload.Warnings = append(payload.Warnings, err.Error())
	} else {
		payload.LatestSnapshot = toSnapshotView(latestSnapshot)
	}
	if snapshots, err := s.store.ListSnapshots(ctx, s.workspace.Config.RepoID, 8); err == nil {
		payload.RecentSnapshots = make([]snapshotView, 0, len(snapshots))
		for _, snapshot := range snapshots {
			payload.RecentSnapshots = append(payload.RecentSnapshots, *toSnapshotView(&snapshot))
		}
	} else {
		payload.Warnings = append(payload.Warnings, err.Error())
	}
	if syncRuns, err := s.store.RecentSyncRuns(ctx, s.workspace.Config.RepoID, 8); err == nil {
		payload.SyncRuns = make([]syncRunView, 0, len(syncRuns))
		for _, run := range syncRuns {
			payload.SyncRuns = append(payload.SyncRuns, syncRunView{
				ID:         run.ID,
				RepoID:     run.RepoID,
				GitCommit:  run.GitCommit,
				SnapshotID: run.SnapshotID,
				State:      string(run.State),
				LastError:  run.LastError,
				RetryCount: run.RetryCount,
				CreatedAt:  run.CreatedAt,
				UpdatedAt:  run.UpdatedAt,
			})
		}
	} else {
		payload.Warnings = append(payload.Warnings, err.Error())
	}
	if tasks, err := s.store.ListTasks(ctx, s.workspace.Config.RepoID, 8); err == nil {
		payload.Tasks = make([]taskView, 0, len(tasks))
		for _, task := range tasks {
			payload.Tasks = append(payload.Tasks, taskView{
				ID:         task.ID,
				SnapshotID: task.SnapshotID,
				Title:      task.Title,
				Status:     task.Status,
				CreatedAt:  task.CreatedAt,
			})
		}
	} else {
		payload.Warnings = append(payload.Warnings, err.Error())
	}
	if proposals, err := s.store.ListProposals(ctx, s.workspace.Config.RepoID, 8); err == nil {
		payload.Proposals = make([]proposalView, 0, len(proposals))
		for _, proposal := range proposals {
			payload.Proposals = append(payload.Proposals, proposalView{
				ID:             proposal.ID,
				SnapshotID:     proposal.SnapshotID,
				Title:          proposal.Title,
				Status:         proposal.Status,
				Summary:        proposal.Summary,
				ChangedSymbols: proposal.ChangedSymbols,
				CreatedAt:      proposal.CreatedAt,
			})
		}
	} else {
		payload.Warnings = append(payload.Warnings, err.Error())
	}

	gitCommit := "working-tree"
	if payload.Git.Head != "" {
		gitCommit = payload.Git.Head
	}
	if changes, err := s.diff.SemanticDiff(ctx, s.workspace, gitCommit, s.author); err == nil {
		payload.SemanticChanges = make([]semanticChangeView, 0, min(25, len(changes)))
		for idx, change := range changes {
			if idx == 25 {
				break
			}
			switch change.ChangeKind {
			case model.SemanticChangeAdded:
				payload.SemanticCounts.Added++
			case model.SemanticChangeModified:
				payload.SemanticCounts.Modified++
			case model.SemanticChangeDeleted:
				payload.SemanticCounts.Deleted++
			}
			payload.SemanticChanges = append(payload.SemanticChanges, semanticChangeView{
				Path:       change.Path,
				FQName:     change.FQName,
				Kind:       change.Kind,
				Language:   change.Language,
				ChangeKind: string(change.ChangeKind),
			})
		}
		for _, change := range changes[len(payload.SemanticChanges):] {
			switch change.ChangeKind {
			case model.SemanticChangeAdded:
				payload.SemanticCounts.Added++
			case model.SemanticChangeModified:
				payload.SemanticCounts.Modified++
			case model.SemanticChangeDeleted:
				payload.SemanticCounts.Deleted++
			}
		}
	} else {
		payload.Warnings = append(payload.Warnings, "semantic diff: "+err.Error())
	}
	if diffText, err := s.diff.TextDiff(ctx, s.workspace); err == nil {
		payload.TextDiff = trimTextDiff(diffText, 120)
	} else {
		payload.Warnings = append(payload.Warnings, "text diff: "+err.Error())
	}
	return payload
}

func trimTextDiff(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n") + "\n... output truncated ..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func toSnapshotView(snapshot *model.Snapshot) *snapshotView {
	if snapshot == nil {
		return nil
	}
	return &snapshotView{
		ID:               snapshot.ID,
		RepoID:           snapshot.RepoID,
		GitCommit:        snapshot.GitCommit,
		ParentSnapshotID: snapshot.ParentSnapshotID,
		Status:           string(snapshot.Status),
		Reflection:       snapshot.Reflection,
		Author:           snapshot.Author,
		CreatedAt:        snapshot.CreatedAt,
	}
}

var dashboardPage = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f4efe7;
      --bg-panel: rgba(255, 252, 247, 0.82);
      --ink: #1f2a2c;
      --muted: #667678;
      --line: rgba(31, 42, 44, 0.1);
      --accent: #b4542d;
      --accent-soft: rgba(180, 84, 45, 0.12);
      --good: #2b6a4b;
      --warn: #9b5d17;
      --bad: #a13434;
      --shadow: 0 18px 50px rgba(58, 44, 27, 0.12);
      --radius: 22px;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Avenir Next", "Segoe UI", "Helvetica Neue", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(180, 84, 45, 0.18), transparent 35%),
        radial-gradient(circle at top right, rgba(68, 109, 118, 0.16), transparent 28%),
        linear-gradient(180deg, #f7f1e8 0%, #efe8de 100%);
      min-height: 100vh;
    }
    .shell {
      max-width: 1320px;
      margin: 0 auto;
      padding: 28px 22px 48px;
    }
    .hero {
      display: grid;
      gap: 14px;
      margin-bottom: 22px;
    }
    .eyebrow {
      display: inline-flex;
      width: fit-content;
      padding: 8px 12px;
      border-radius: 999px;
      letter-spacing: 0.08em;
      font-size: 12px;
      text-transform: uppercase;
      background: rgba(31, 42, 44, 0.06);
      color: var(--muted);
    }
    h1 {
      margin: 0;
      font-size: clamp(32px, 5vw, 58px);
      line-height: 0.94;
      letter-spacing: -0.04em;
      max-width: 12ch;
    }
    .hero p {
      margin: 0;
      max-width: 60ch;
      color: var(--muted);
      font-size: 16px;
      line-height: 1.6;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      color: var(--muted);
      font-size: 14px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 16px;
    }
    .card {
      background: var(--bg-panel);
      backdrop-filter: blur(14px);
      border: 1px solid var(--line);
      border-radius: var(--radius);
      box-shadow: var(--shadow);
      padding: 18px 18px 16px;
    }
    .card h2 {
      margin: 0 0 14px;
      font-size: 15px;
      letter-spacing: 0.06em;
      text-transform: uppercase;
      color: var(--muted);
    }
    .span-4 { grid-column: span 4; }
    .span-5 { grid-column: span 5; }
    .span-6 { grid-column: span 6; }
    .span-7 { grid-column: span 7; }
    .span-8 { grid-column: span 8; }
    .span-12 { grid-column: span 12; }
    .stat {
      display: grid;
      gap: 4px;
    }
    .stat strong {
      font-size: 24px;
      letter-spacing: -0.03em;
    }
    .muted { color: var(--muted); }
    .status-row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-bottom: 14px;
    }
    .pill {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      border-radius: 999px;
      font-size: 13px;
      background: rgba(31, 42, 44, 0.06);
      color: var(--ink);
    }
    .pill.good { background: rgba(43, 106, 75, 0.1); color: var(--good); }
    .pill.warn { background: rgba(155, 93, 23, 0.12); color: var(--warn); }
    .pill.bad { background: rgba(161, 52, 52, 0.12); color: var(--bad); }
    .list {
      display: grid;
      gap: 10px;
    }
    .item {
      padding: 12px 14px;
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.64);
      border: 1px solid rgba(31, 42, 44, 0.08);
    }
    .item .title {
      font-weight: 600;
      margin-bottom: 4px;
      word-break: break-word;
    }
    .item .sub {
      color: var(--muted);
      font-size: 13px;
      word-break: break-word;
    }
    .split {
      display: grid;
      gap: 12px;
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    pre {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      font-family: "IBM Plex Mono", "SFMono-Regular", "Menlo", monospace;
      font-size: 12px;
      line-height: 1.5;
      color: #243033;
      background: rgba(255, 255, 255, 0.72);
      border-radius: 18px;
      border: 1px solid rgba(31, 42, 44, 0.08);
      padding: 14px;
      max-height: 420px;
      overflow: auto;
    }
    .warning {
      padding: 10px 12px;
      border-radius: 14px;
      background: rgba(155, 93, 23, 0.12);
      color: var(--warn);
      font-size: 13px;
    }
    @media (max-width: 980px) {
      .span-4, .span-5, .span-6, .span-7, .span-8, .span-12 { grid-column: span 12; }
      .split { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <span class="eyebrow">Ada Observer</span>
      <h1>Simple live view into the semantic sidecar.</h1>
      <p>Watch Git state, snapshots, semantic drift, sync health, tasks, and proposals in one place. This page refreshes itself every few seconds, so you can leave it open while you work.</p>
      <div class="meta">
        <span id="workspaceRoot">Loading workspace…</span>
        <span id="updatedAt"></span>
      </div>
    </section>

    <div class="grid">
      <section class="card span-4">
        <h2>Workspace</h2>
        <div class="list" id="workspaceInfo"></div>
      </section>

      <section class="card span-4">
        <h2>Git</h2>
        <div class="status-row" id="gitPills"></div>
        <div class="list" id="gitDetails"></div>
      </section>

      <section class="card span-4">
        <h2>Semantic Drift</h2>
        <div class="split">
          <div class="stat"><span class="muted">Added</span><strong id="addedCount">0</strong></div>
          <div class="stat"><span class="muted">Modified</span><strong id="modifiedCount">0</strong></div>
          <div class="stat"><span class="muted">Deleted</span><strong id="deletedCount">0</strong></div>
          <div class="stat"><span class="muted">Visible</span><strong id="visibleChanges">0</strong></div>
        </div>
      </section>

      <section class="card span-6">
        <h2>Latest Snapshot</h2>
        <div class="list" id="latestSnapshot"></div>
      </section>

      <section class="card span-6">
        <h2>Recent Sync Runs</h2>
        <div class="list" id="syncRuns"></div>
      </section>

      <section class="card span-7">
        <h2>Semantic Changes</h2>
        <div class="list" id="semanticChanges"></div>
      </section>

      <section class="card span-5">
        <h2>Snapshot Timeline</h2>
        <div class="list" id="snapshots"></div>
      </section>

      <section class="card span-6">
        <h2>Tasks</h2>
        <div class="list" id="tasks"></div>
      </section>

      <section class="card span-6">
        <h2>Proposals</h2>
        <div class="list" id="proposals"></div>
      </section>

      <section class="card span-12">
        <h2>Text Diff</h2>
        <pre id="textDiff">Loading…</pre>
      </section>

      <section class="card span-12">
        <h2>Warnings</h2>
        <div class="list" id="warnings"></div>
      </section>
    </div>
  </div>
  <script>
    const byId = (id) => document.getElementById(id);
    const bullet = " • ";

    function escapeHtml(text) {
      return String(text ?? "")
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;");
    }

    function listHtml(items, render, empty) {
      if (!items || items.length === 0) {
        return '<div class="item"><div class="sub">' + escapeHtml(empty) + '</div></div>';
      }
      return items.map(render).join("");
    }

    function pill(label, kind) {
      return '<span class="pill ' + (kind || "") + '">' + escapeHtml(label) + '</span>';
    }

    function timeAgo(value) {
      if (!value) return "";
      const then = new Date(value);
      const seconds = Math.round((Date.now() - then.getTime()) / 1000);
      if (seconds < 60) return String(seconds) + "s ago";
      const minutes = Math.round(seconds / 60);
      if (minutes < 60) return String(minutes) + "m ago";
      const hours = Math.round(minutes / 60);
      if (hours < 24) return String(hours) + "h ago";
      const days = Math.round(hours / 24);
      return String(days) + "d ago";
    }

    async function refresh() {
      const res = await fetch("/api/dashboard", { cache: "no-store" });
      const data = await res.json();

      byId("workspaceRoot").textContent = data.workspace.root;
      byId("updatedAt").textContent = "Updated " + timeAgo(data.generated_at) + bullet + "repo " + data.workspace.repo_id;

      byId("workspaceInfo").innerHTML = listHtml([
        { title: "Root", sub: data.workspace.root },
        { title: "Repo ID", sub: data.workspace.repo_id },
        { title: "Remote", sub: data.workspace.remote_url || "No remote configured" },
      ], (item) => '<div class="item"><div class="title">' + escapeHtml(item.title) + '</div><div class="sub">' + escapeHtml(item.sub) + '</div></div>', "No workspace info.");

      const gitPills = [];
      if (data.git.present) {
        gitPills.push(pill(data.git.clean ? "Working tree clean" : "Working tree dirty", data.git.clean ? "good" : "warn"));
        if (data.latest_snapshot && data.latest_snapshot.status) {
          const statusKind = data.latest_snapshot.status === "synced" ? "good" : (data.latest_snapshot.status === "pending_reconcile" ? "warn" : "bad");
          gitPills.push(pill("Snapshot " + data.latest_snapshot.status, statusKind));
        }
      } else {
        gitPills.push(pill("Git repo not detected", "bad"));
      }
      byId("gitPills").innerHTML = gitPills.join("");

      const gitDetails = [];
      if (data.git.present) {
        gitDetails.push({ title: "Branch", sub: data.git.branch || "detached" });
        gitDetails.push({ title: "HEAD", sub: data.git.head || "unknown" });
        gitDetails.push({ title: "Pending file changes", sub: (data.git.status_entries || []).join(" | ") || "none" });
      } else {
        gitDetails.push({ title: "Status", sub: data.git.error || "Git metadata unavailable" });
      }
      byId("gitDetails").innerHTML = listHtml(gitDetails, (item) => '<div class="item"><div class="title">' + escapeHtml(item.title) + '</div><div class="sub">' + escapeHtml(item.sub) + '</div></div>', "No git details.");

      byId("addedCount").textContent = data.semantic_counts.added;
      byId("modifiedCount").textContent = data.semantic_counts.modified;
      byId("deletedCount").textContent = data.semantic_counts.deleted;
      byId("visibleChanges").textContent = data.semantic_changes.length;

      const latest = data.latest_snapshot ? [{
        title: data.latest_snapshot.id + bullet + data.latest_snapshot.status,
        sub: data.latest_snapshot.git_commit + bullet + timeAgo(data.latest_snapshot.created_at) + bullet + data.latest_snapshot.reflection
      }] : [];
      byId("latestSnapshot").innerHTML = listHtml(latest, (item) => '<div class="item"><div class="title">' + escapeHtml(item.title) + '</div><div class="sub">' + escapeHtml(item.sub) + '</div></div>', "No snapshots yet. Run ada sync.");

      byId("syncRuns").innerHTML = listHtml(data.sync_runs, (run) => (
        '<div class="item">' +
          '<div class="title">' + escapeHtml(run.state) + bullet + escapeHtml(run.git_commit) + '</div>' +
          '<div class="sub">' + escapeHtml(run.snapshot_id) + bullet + timeAgo(run.updated_at) + (run.last_error ? bullet + escapeHtml(run.last_error) : "") + '</div>' +
        '</div>'
      ), "No sync runs yet.");

      byId("semanticChanges").innerHTML = listHtml(data.semantic_changes, (change) => (
        '<div class="item">' +
          '<div class="title">' + escapeHtml(change.change_kind) + bullet + escapeHtml(change.fq_name) + '</div>' +
          '<div class="sub">' + escapeHtml(change.language) + bullet + escapeHtml(change.kind) + bullet + escapeHtml(change.path) + '</div>' +
        '</div>'
      ), "No semantic changes from the latest snapshot.");

      byId("snapshots").innerHTML = listHtml(data.recent_snapshots, (snapshot) => (
        '<div class="item">' +
          '<div class="title">' + escapeHtml(snapshot.status) + bullet + escapeHtml(snapshot.git_commit) + '</div>' +
          '<div class="sub">' + escapeHtml(snapshot.reflection) + bullet + timeAgo(snapshot.created_at) + '</div>' +
        '</div>'
      ), "No snapshots yet.");

      byId("tasks").innerHTML = listHtml(data.tasks, (task) => (
        '<div class="item">' +
          '<div class="title">' + escapeHtml(task.status) + bullet + escapeHtml(task.title) + '</div>' +
          '<div class="sub">' + escapeHtml(task.snapshot_id) + bullet + timeAgo(task.created_at) + '</div>' +
        '</div>'
      ), "No tasks recorded.");

      byId("proposals").innerHTML = listHtml(data.proposals, (proposal) => (
        '<div class="item">' +
          '<div class="title">' + escapeHtml(proposal.status) + bullet + escapeHtml(proposal.title) + '</div>' +
          '<div class="sub">' + escapeHtml(proposal.summary) + bullet + timeAgo(proposal.created_at) + '</div>' +
        '</div>'
      ), "No proposals recorded.");

      byId("textDiff").textContent = data.text_diff || "No text diff from the latest snapshot.";
      byId("warnings").innerHTML = listHtml(data.warnings, (warning) => '<div class="warning">' + escapeHtml(warning) + '</div>', "No warnings.");
    }

    refresh();
    setInterval(refresh, 4000);
  </script>
</body>
</html>`))
