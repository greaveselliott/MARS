package scanner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const harnessDir = ".harness"

// starterManifest is the minimal manifest template written during init.
type starterManifest struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Roles       map[string]string `yaml:"roles"`
}

// Init scaffolds the .harness/ directory for a repository.
// If .harness/ exists and force is false, returns an error.
// If the directory is not a git repo, returns an actionable error.
func Init(repoRoot string, force bool) error {
	if repoRoot == "" {
		return fmt.Errorf("init: repo root is empty — pass the path to the repository")
	}
	info, err := os.Stat(repoRoot)
	if err != nil {
		return fmt.Errorf("init: cannot access %s: %w — verify the path exists", repoRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("init: %s is not a directory — point to the repository root", repoRoot)
	}

	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("init: %s is not a git repository — run 'git init' first", repoRoot)
	}

	harnessPath := filepath.Join(repoRoot, harnessDir)
	if _, err := os.Stat(harnessPath); err == nil && !force {
		return fmt.Errorf("init: %s already exists — use --force to overwrite", harnessPath)
	}

	dirs := []string{
		harnessPath,
		filepath.Join(harnessPath, "roles"),
		filepath.Join(harnessPath, "guardrails"),
		filepath.Join(harnessPath, "knowledge"),
		filepath.Join(harnessPath, "tickets"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("init: create %s: %w — check directory permissions", d, err)
		}
		slog.Debug("created directory", "path", d)
	}

	manifest := starterManifest{
		Name:        filepath.Base(repoRoot),
		Description: "Mars Harness configuration",
		Roles:       map[string]string{},
	}
	manifestBytes, err := yaml.Marshal(&manifest)
	if err != nil {
		return fmt.Errorf("init: marshal manifest: %w", err)
	}
	manifestPath := filepath.Join(harnessPath, "manifest.yaml")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return fmt.Errorf("init: write %s: %w — check directory permissions", manifestPath, err)
	}

	slog.Info("initialized .harness/", "path", harnessPath)
	return nil
}
