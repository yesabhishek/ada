package gitutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotGitRepo = errors.New("not a git repository")

type Repo struct {
	Root string
}

func Discover(ctx context.Context, root string) (*Repo, error) {
	output, err := run(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, ErrNotGitRepo
	}
	return &Repo{Root: strings.TrimSpace(output)}, nil
}

func (r *Repo) HeadCommit(ctx context.Context) (string, error) {
	out, err := run(ctx, r.Root, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve HEAD: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (r *Repo) CurrentBranch(ctx context.Context) (string, error) {
	out, err := run(ctx, r.Root, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("read current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (r *Repo) IsClean(ctx context.Context) (bool, error) {
	entries, err := r.StatusEntries(ctx)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func (r *Repo) HasRemote(ctx context.Context) (bool, error) {
	out, err := run(ctx, r.Root, "remote")
	if err != nil {
		return false, fmt.Errorf("list remotes: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *Repo) Push(ctx context.Context) error {
	if _, err := run(ctx, r.Root, "push"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

func (r *Repo) StatusEntries(ctx context.Context) ([]string, error) {
	out, err := run(ctx, r.Root, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("read git status: %w", err)
	}
	return filterStatusEntries(out), nil
}

func run(ctx context.Context, cwd string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", cwd}, args...)...)
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
		return "", errors.New(msg)
	}
	return stdout.String(), nil
}

func filterStatusEntries(out string) []string {
	var entries []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, filepath.ToSlash(".ada/")) || strings.HasSuffix(line, ".ada") {
			continue
		}
		entries = append(entries, line)
	}
	return entries
}
