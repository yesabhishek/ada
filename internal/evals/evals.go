package evals

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/yesabhishek/ada/internal/language"
	"github.com/yesabhishek/ada/internal/model"
	"github.com/yesabhishek/ada/internal/semantic"
)

type Scenario struct {
	Name          string
	Description   string
	Language      string
	Path          string
	Tags          []string
	Base          string
	Ours          string
	Theirs        string
	ExpectVerdict string
}

type EngineResult struct {
	Status          string `json:"status"`
	Parseable       bool   `json:"parseable"`
	ConflictMarkers int    `json:"conflict_markers,omitempty"`
	ConflictCount   int    `json:"conflict_count,omitempty"`
	Error           string `json:"error,omitempty"`
}

type ScenarioResult struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Language    string       `json:"language"`
	Tags        []string     `json:"tags"`
	Verdict     string       `json:"verdict"`
	Note        string       `json:"note"`
	Git         EngineResult `json:"git"`
	Ada         EngineResult `json:"ada"`
}

type Summary struct {
	Total        int `json:"total"`
	AdaAdvantage int `json:"ada_advantage"`
	GitAdvantage int `json:"git_advantage"`
	Tie          int `json:"tie"`
	BothClean    int `json:"both_clean"`
	BothConflict int `json:"both_conflict"`
}

type Report struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Results     []ScenarioResult `json:"results"`
	Summary     Summary          `json:"summary"`
}

func Run(ctx context.Context, registry *language.Registry, scenarios []Scenario) (Report, error) {
	results := make([]ScenarioResult, 0, len(scenarios))
	for _, scenario := range scenarios {
		result, err := runScenario(ctx, registry, scenario)
		if err != nil {
			return Report{}, fmt.Errorf("run scenario %s: %w", scenario.Name, err)
		}
		results = append(results, result)
	}
	report := Report{
		GeneratedAt: time.Now().UTC(),
		Results:     results,
		Summary:     summarize(results),
	}
	return report, nil
}

func RenderText(report Report) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Ada vs Git merge evals\nGenerated: %s\n\n", report.GeneratedAt.Format(time.RFC3339))
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SCENARIO\tLANG\tGIT\tADA\tVERDICT\tNOTE")
	for _, result := range report.Results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			result.Name,
			result.Language,
			renderEngineShort(result.Git),
			renderEngineShort(result.Ada),
			result.Verdict,
			result.Note,
		)
	}
	_ = w.Flush()
	fmt.Fprintf(&buf, "\nSummary\n")
	fmt.Fprintf(&buf, "- total: %d\n", report.Summary.Total)
	fmt.Fprintf(&buf, "- ada_advantage: %d\n", report.Summary.AdaAdvantage)
	fmt.Fprintf(&buf, "- git_advantage: %d\n", report.Summary.GitAdvantage)
	fmt.Fprintf(&buf, "- tie: %d\n", report.Summary.Tie)
	fmt.Fprintf(&buf, "- both_clean: %d\n", report.Summary.BothClean)
	fmt.Fprintf(&buf, "- both_conflict: %d\n", report.Summary.BothConflict)
	return buf.String()
}

func RenderMarkdown(report Report) string {
	var buf strings.Builder
	buf.WriteString("# Ada vs Git merge evals\n\n")
	buf.WriteString("| Scenario | Lang | Git | Ada | Verdict | Note |\n")
	buf.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, result := range report.Results {
		fmt.Fprintf(&buf, "| %s | %s | %s | %s | %s | %s |\n",
			result.Name,
			result.Language,
			renderEngineShort(result.Git),
			renderEngineShort(result.Ada),
			result.Verdict,
			escapePipe(result.Note),
		)
	}
	buf.WriteString("\n")
	fmt.Fprintf(&buf, "- total: %d\n", report.Summary.Total)
	fmt.Fprintf(&buf, "- ada_advantage: %d\n", report.Summary.AdaAdvantage)
	fmt.Fprintf(&buf, "- git_advantage: %d\n", report.Summary.GitAdvantage)
	fmt.Fprintf(&buf, "- tie: %d\n", report.Summary.Tie)
	return buf.String()
}

func runScenario(ctx context.Context, registry *language.Registry, scenario Scenario) (ScenarioResult, error) {
	gitResult, err := runGitMerge(ctx, registry, scenario)
	if err != nil {
		return ScenarioResult{}, err
	}
	adaResult, err := runAdaMerge(ctx, registry, scenario)
	if err != nil {
		return ScenarioResult{}, err
	}
	verdict, note := compare(gitResult, adaResult)
	return ScenarioResult{
		Name:        scenario.Name,
		Description: scenario.Description,
		Language:    scenario.Language,
		Tags:        append([]string(nil), scenario.Tags...),
		Verdict:     verdict,
		Note:        note,
		Git:         gitResult,
		Ada:         adaResult,
	}, nil
}

func runGitMerge(ctx context.Context, registry *language.Registry, scenario Scenario) (EngineResult, error) {
	tmpDir, err := os.MkdirTemp("", "ada-git-eval-*")
	if err != nil {
		return EngineResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if _, err := git(ctx, tmpDir, "init"); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "config", "user.name", "Ada Eval"); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "config", "user.email", "ada-eval@example.com"); err != nil {
		return EngineResult{}, err
	}
	if err := writeFile(tmpDir, scenario.Path, scenario.Base); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "add", scenario.Path); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "commit", "-m", "base"); err != nil {
		return EngineResult{}, err
	}
	baseCommit, err := git(ctx, tmpDir, "rev-parse", "HEAD")
	if err != nil {
		return EngineResult{}, err
	}
	mainBranch, err := git(ctx, tmpDir, "branch", "--show-current")
	if err != nil {
		return EngineResult{}, err
	}
	baseCommit = strings.TrimSpace(baseCommit)
	mainBranch = strings.TrimSpace(mainBranch)

	if _, err := git(ctx, tmpDir, "checkout", "-b", "ours"); err != nil {
		return EngineResult{}, err
	}
	if err := writeFile(tmpDir, scenario.Path, scenario.Ours); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "add", scenario.Path); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "commit", "-m", "ours"); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "checkout", "-b", "theirs", baseCommit); err != nil {
		return EngineResult{}, err
	}
	if err := writeFile(tmpDir, scenario.Path, scenario.Theirs); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "add", scenario.Path); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "commit", "-m", "theirs"); err != nil {
		return EngineResult{}, err
	}
	if _, err := git(ctx, tmpDir, "checkout", "ours"); err != nil {
		return EngineResult{}, err
	}
	_, mergeErr := git(ctx, tmpDir, "merge", "--no-commit", "--no-ff", "theirs")

	mergedPath := filepath.Join(tmpDir, filepath.FromSlash(scenario.Path))
	merged, readErr := os.ReadFile(mergedPath)
	if readErr != nil {
		merged = nil
	}
	parseable := parseable(registry, scenario.Path, merged)
	conflictMarkers := countConflictMarkers(string(merged))
	if mergeErr == nil {
		return EngineResult{
			Status:    "clean",
			Parseable: parseable,
		}, nil
	}
	if conflictMarkers > 0 {
		return EngineResult{
			Status:          "conflict",
			Parseable:       parseable,
			ConflictMarkers: conflictMarkers,
			Error:           mergeErr.Error(),
		}, nil
	}
	return EngineResult{
		Status:    "error",
		Parseable: parseable,
		Error:     mergeErr.Error(),
	}, nil
}

func runAdaMerge(ctx context.Context, registry *language.Registry, scenario Scenario) (EngineResult, error) {
	adapter := registry.ForPath(scenario.Path)
	if adapter == nil {
		return EngineResult{}, fmt.Errorf("no language adapter for %s", scenario.Path)
	}
	baseVersions, err := symbolVersionsFromSource(ctx, adapter, scenario.Path, scenario.Base)
	if err != nil {
		return EngineResult{Status: "error", Error: err.Error()}, nil
	}
	oursVersions, err := symbolVersionsFromSource(ctx, adapter, scenario.Path, scenario.Ours)
	if err != nil {
		return EngineResult{Status: "error", Error: err.Error()}, nil
	}
	theirsVersions, err := symbolVersionsFromSource(ctx, adapter, scenario.Path, scenario.Theirs)
	if err != nil {
		return EngineResult{Status: "error", Error: err.Error()}, nil
	}
	result, err := semantic.ThreeWayMerge(ctx, registry, scenario.Path, []byte(scenario.Base), baseVersions, oursVersions, theirsVersions)
	if err != nil {
		return EngineResult{Status: "error", Error: err.Error()}, nil
	}
	if len(result.Conflicts) > 0 {
		return EngineResult{
			Status:        "conflict",
			ConflictCount: len(result.Conflicts),
		}, nil
	}
	return EngineResult{
		Status:    "clean",
		Parseable: parseable(registry, scenario.Path, result.MergedSource),
	}, nil
}

func symbolVersionsFromSource(ctx context.Context, adapter language.Adapter, path string, source string) ([]model.SymbolVersion, error) {
	parsed, err := adapter.Parse(ctx, path, []byte(source))
	if err != nil {
		return nil, err
	}
	versions := make([]model.SymbolVersion, 0, len(parsed.Symbols))
	for _, symbol := range parsed.Symbols {
		versions = append(versions, model.SymbolVersion{
			SymbolID:        symbol.FQName,
			ContentHash:     model.HashText(symbol.NormalizedBody),
			AnchorStartByte: symbol.AnchorStartByte,
			AnchorEndByte:   symbol.AnchorEndByte,
			RawText:         symbol.RawText,
		})
	}
	return versions, nil
}

func parseable(registry *language.Registry, path string, source []byte) bool {
	if len(source) == 0 {
		return false
	}
	adapter := registry.ForPath(path)
	if adapter == nil {
		return false
	}
	_, err := adapter.Parse(context.Background(), path, source)
	return err == nil
}

func compare(gitResult, adaResult EngineResult) (string, string) {
	switch {
	case gitResult.Status == "conflict" && adaResult.Status == "clean" && adaResult.Parseable:
		return "ada_advantage", "Ada merged while Git produced a textual conflict."
	case gitResult.Status == "clean" && gitResult.Parseable && adaResult.Status == "conflict":
		return "git_advantage", "Git merged the case, while Ada conservatively flagged a semantic conflict."
	case gitResult.Status == "error" && adaResult.Status != "error":
		return "ada_advantage", "Git errored while Ada still produced a usable result."
	case adaResult.Status == "error" && gitResult.Status != "error":
		return "git_advantage", "Ada errored while Git still produced a usable result."
	case gitResult.Status == "clean" && adaResult.Status == "clean":
		switch {
		case gitResult.Parseable && !adaResult.Parseable:
			return "git_advantage", "Both merged, but only Git produced a parseable file."
		case !gitResult.Parseable && adaResult.Parseable:
			return "ada_advantage", "Both merged, but only Ada produced a parseable file."
		default:
			return "tie", "Both engines merged cleanly."
		}
	case gitResult.Status == "conflict" && adaResult.Status == "conflict":
		return "tie", "Both engines correctly blocked a conflicting change."
	default:
		return "tie", "Neither engine showed a decisive advantage on this scenario."
	}
}

func summarize(results []ScenarioResult) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		switch result.Verdict {
		case "ada_advantage":
			summary.AdaAdvantage++
		case "git_advantage":
			summary.GitAdvantage++
		default:
			summary.Tie++
		}
		if result.Git.Status == "clean" && result.Ada.Status == "clean" {
			summary.BothClean++
		}
		if result.Git.Status == "conflict" && result.Ada.Status == "conflict" {
			summary.BothConflict++
		}
	}
	return summary
}

func renderEngineShort(result EngineResult) string {
	if result.Status == "conflict" {
		if result.ConflictMarkers > 0 {
			return fmt.Sprintf("conflict(%d)", result.ConflictMarkers)
		}
		if result.ConflictCount > 0 {
			return fmt.Sprintf("conflict(%d)", result.ConflictCount)
		}
	}
	if result.Status == "clean" && !result.Parseable {
		return "clean(!parse)"
	}
	return result.Status
}

func git(ctx context.Context, cwd string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func writeFile(root, rel, content string) error {
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", rel, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

func countConflictMarkers(text string) int {
	return strings.Count(text, "<<<<<<<")
}

func escapePipe(text string) string {
	return strings.ReplaceAll(text, "|", "\\|")
}
