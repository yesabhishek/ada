package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yesabhishek/ada/internal/model"
)

const (
	DirName    = ".ada"
	ConfigName = "config.json"
	DBName     = "ada.db"
)

type Config struct {
	Version   int       `json:"version"`
	RepoID    string    `json:"repo_id"`
	RemoteURL string    `json:"remote_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Workspace struct {
	Root   string
	AdaDir string
	Config Config
}

func New(root string, cfg Config) *Workspace {
	return &Workspace{
		Root:   root,
		AdaDir: filepath.Join(root, DirName),
		Config: cfg,
	}
}

func (w *Workspace) DBPath() string {
	return filepath.Join(w.AdaDir, DBName)
}

func (w *Workspace) ConfigPath() string {
	return filepath.Join(w.AdaDir, ConfigName)
}

func Initialize(root string, remoteURL string) (*Workspace, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	adaDir := filepath.Join(absRoot, DirName)
	if err := os.MkdirAll(adaDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .ada directory: %w", err)
	}
	cfg := Config{
		Version:   1,
		RepoID:    deriveRepoID(absRoot),
		RemoteURL: remoteURL,
		CreatedAt: time.Now().UTC(),
	}
	ws := New(absRoot, cfg)
	if err := ws.SaveConfig(); err != nil {
		return nil, err
	}
	return ws, nil
}

func Find(start string) (*Workspace, error) {
	absStart, err := filepath.Abs(start)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace search root: %w", err)
	}
	current := absStart
	for {
		adaDir := filepath.Join(current, DirName)
		configPath := filepath.Join(adaDir, ConfigName)
		if _, err := os.Stat(configPath); err == nil {
			cfg, err := loadConfig(configPath)
			if err != nil {
				return nil, err
			}
			return &Workspace{Root: current, AdaDir: adaDir, Config: cfg}, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil, errors.New("no Ada workspace found; run `ada start <path>` first")
		}
		current = parent
	}
}

func (w *Workspace) SaveConfig() error {
	if err := os.MkdirAll(w.AdaDir, 0o755); err != nil {
		return fmt.Errorf("create .ada directory: %w", err)
	}
	body, err := json.MarshalIndent(w.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(w.ConfigPath(), append(body, '\n'), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (w *Workspace) SnapshotSummary(snapshot model.Snapshot) string {
	return fmt.Sprintf("%s %s %s", snapshot.ID, snapshot.GitCommit, snapshot.Status)
}

func loadConfig(path string) (Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func deriveRepoID(root string) string {
	name := filepath.Base(root)
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "ada-repo"
	}
	return name
}
