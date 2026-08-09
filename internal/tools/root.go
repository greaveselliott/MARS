/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/greaveselliott/mars/internal/repofs"
)

// Root is an absolute, cleaned workspace directory; all tool file paths must resolve under it.
type Root struct {
	abs    string
	dbPath string
	repo   *repofs.Root
}

// NewRoot resolves dir to an absolute path with symlinks evaluated where possible.
func NewRoot(dir string) (Root, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Root{}, fmt.Errorf("tools: resolve workdir: %w", err)
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = eval
	}
	repo, err := repofs.Open(abs)
	if err != nil {
		return Root{}, fmt.Errorf("tools: open workdir: %w", err)
	}
	return Root{abs: repo.Abs(), repo: repo}, nil
}

// Abs returns the absolute root path.
func (r Root) Abs() string { return r.abs }

// WithDBPath returns a root bound to the active Mars SQLite database for this
// repo job. File containment remains based only on Abs().
func (r Root) WithDBPath(dbPath string) Root {
	r.dbPath = strings.TrimSpace(dbPath)
	return r
}

// DBPath returns the active Mars SQLite database path, when the caller supplied
// one. Empty means tools should use their package default.
func (r Root) DBPath() string { return r.dbPath }

// RepoFS returns the descriptor-bound repository filesystem used by tool file
// operations.
func (r Root) RepoFS() *repofs.Root { return r.repo }

// ResolvePath joins root with rel and ensures the result stays within root.
// rel may use platform separators; leading slashes are stripped.
func (r Root) ResolvePath(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("tools: path is empty")
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return r.abs, nil
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("tools: path %q must be relative to the repository root", rel)
	}
	joined := filepath.Join(r.abs, rel)
	clean := filepath.Clean(joined)
	relOut, err := filepath.Rel(r.abs, clean)
	if err != nil {
		return "", fmt.Errorf("tools: path %q escapes repository root", rel)
	}
	if relOut == ".." || strings.HasPrefix(relOut, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("tools: path %q escapes repository root", rel)
	}
	return clean, nil
}

// OpenFile opens path under root with the given flags and perm.
func (r Root) OpenFile(rel string, flag int, perm fs.FileMode) (*os.File, string, error) {
	p, err := r.ResolvePath(rel)
	if err != nil {
		return nil, "", err
	}
	if flag != os.O_RDONLY {
		return nil, p, fmt.Errorf("tools: unsafe generic repository open flags are not supported")
	}
	f, err := r.repo.OpenFile(rel)
	if err != nil {
		return nil, p, err
	}
	return f, p, nil
}

func createExclusiveRepositoryFile(root Root, rel string, data []byte, perm fs.FileMode) error {
	repo := root.RepoFS()
	if repo == nil {
		return fmt.Errorf("tools: repository filesystem is unavailable")
	}
	if err := repo.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return err
	}
	file, err := repo.CreateExclusive(rel, perm)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = file.Close()
			_ = repo.Remove(rel)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func atomicWriteRepositoryFile(root Root, rel string, data []byte, perm fs.FileMode) error {
	repo := root.RepoFS()
	if repo == nil {
		return fmt.Errorf("tools: repository filesystem is unavailable")
	}
	if err := repo.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return err
	}
	mode := perm
	if info, err := repo.Stat(rel); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("tools: repository replacement target is not a regular file")
		}
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return repo.AtomicWrite(rel, data, mode)
}
