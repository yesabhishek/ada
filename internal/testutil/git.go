package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func Run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}
	return string(output)
}

func InitGitRepo(t *testing.T, dir string) {
	t.Helper()
	Run(t, dir, "git", "init")
	Run(t, dir, "git", "config", "user.email", "ada@example.com")
	Run(t, dir, "git", "config", "user.name", "Ada Test")
}

func WriteAndCommit(t *testing.T, dir, relPath string, content []byte, message string) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
	Run(t, dir, "git", "add", relPath)
	Run(t, dir, "git", "commit", "-m", message)
}
