/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/features/F-005-agent-execution-runtime.md
*/
package context

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	defaultMaxFileSize = 512 * 1024 // 512 KB
	binarySampleSize   = 8192
	nullThreshold      = 0.10 // >10% null bytes ⇒ binary
)

// FileFilter controls which files in a repo are eligible for context inclusion (M1.4.2).
type FileFilter struct {
	MaxFileSize    int      // bytes; 0 = defaultMaxFileSize
	IgnorePatterns []string // gitignore-style globs from .harnessignore
}

// LoadHarnessIgnore reads a .harnessignore file and returns patterns.
// Missing file is not an error — returns nil patterns.
func LoadHarnessIgnore(repoRoot string) ([]string, error) {
	path := filepath.Join(repoRoot, ".harnessignore")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, sc.Err()
}

// ShouldInclude returns true if the file at relPath (relative to repo root) passes all filters.
func (ff *FileFilter) ShouldInclude(repoRoot, relPath string) bool {
	if ff.matchesIgnorePattern(relPath) {
		return false
	}
	abs := filepath.Join(repoRoot, relPath)
	info, err := os.Stat(abs)
	if err != nil {
		return false
	}
	maxSize := ff.maxSize()
	if info.Size() > int64(maxSize) {
		return false
	}
	if isBinary(abs) {
		return false
	}
	return true
}

func (ff *FileFilter) matchesIgnorePattern(relPath string) bool {
	for _, pat := range ff.IgnorePatterns {
		if matched, _ := filepath.Match(pat, filepath.Base(relPath)); matched {
			return true
		}
		if matched, _ := filepath.Match(pat, relPath); matched {
			return true
		}
	}
	return false
}

func (ff *FileFilter) maxSize() int {
	if ff.MaxFileSize > 0 {
		return ff.MaxFileSize
	}
	return defaultMaxFileSize
}

func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, binarySampleSize)
	n, _ := f.Read(buf)
	if n == 0 {
		return false
	}
	buf = buf[:n]
	if !utf8.Valid(buf) {
		return true
	}
	nullCount := 0
	for _, b := range buf {
		if b == 0 {
			nullCount++
		}
	}
	return float64(nullCount)/float64(n) > nullThreshold
}
