/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Root is an absolute, cleaned workspace directory; all tool file paths must resolve under it.
type Root struct {
	abs string
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
	fi, err := os.Stat(abs)
	if err != nil {
		return Root{}, fmt.Errorf("tools: stat workdir: %w", err)
	}
	if !fi.IsDir() {
		return Root{}, fmt.Errorf("tools: workdir %q is not a directory", abs)
	}
	return Root{abs: filepath.Clean(abs)}, nil
}

// Abs returns the absolute root path.
func (r Root) Abs() string { return r.abs }

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
	f, err := os.OpenFile(p, flag, perm)
	if err != nil {
		return nil, p, err
	}
	return f, p, nil
}
