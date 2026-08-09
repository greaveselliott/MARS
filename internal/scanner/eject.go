/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-004-target-harness-lifecycle.md
*/
package scanner

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/greaveselliott/mars/internal/repofs"
)

// EjectOptions controls removal of a deployed MARS from a target repo.
type EjectOptions struct {
	Apply bool
}

// EjectResult describes the repo-local artifacts found by Eject.
type EjectResult struct {
	RepoRoot string
	Removed  []string
	Missing  []string
	Pruned   []string
}

var ejectHarnessPaths = []string{
	harnessDir,
	"AGENTS.md",
	"VERSION",
	"CHANGELOG.md",
	"docs/QUALITY_SCORE.md",
	"docs/tickets",
	"docs/exec-plans",
	"docs/features",
	"docs/goals",
	"docs/roles",
	"docs/references",
	"docs/reports/qa",
	"docs/reports/security",
	"docs/reports/dependencies",
	"docs/reports/dogfood",
	"docs/reports/strategy",
	"docs/design-docs",
}

var ejectPruneDirs = []string{
	filepath.Join("docs", "reports"),
	"docs",
}

// Eject removes the deployed harness surface from repoRoot when Apply is true.
// Without Apply it returns the same plan without mutating the filesystem.
func Eject(repoRoot string, opts EjectOptions) (EjectResult, error) {
	if repoRoot == "" {
		return EjectResult{}, fmt.Errorf("eject: repo root is empty — pass the path to the repository")
	}
	root, err := repofs.Open(repoRoot)
	if err != nil {
		return EjectResult{}, fmt.Errorf("eject: cannot access repository: %w — verify the path exists", err)
	}

	result := EjectResult{RepoRoot: root.Abs()}
	for _, rel := range ejectHarnessPaths {
		rel = filepath.Clean(rel)
		if _, err := root.Stat(rel); errors.Is(err, fs.ErrNotExist) {
			result.Missing = append(result.Missing, filepath.ToSlash(rel))
			continue
		} else if err != nil {
			return result, fmt.Errorf("eject: inspect %s: %w", rel, err)
		}
		result.Removed = append(result.Removed, filepath.ToSlash(rel))
	}

	if opts.Apply {
		for _, rel := range result.Removed {
			if err := root.RemoveAll(filepath.FromSlash(rel)); err != nil {
				return result, fmt.Errorf("eject: remove %s: %w", rel, err)
			}
		}
		for _, rel := range ejectPruneDirs {
			rel = filepath.Clean(rel)
			pruned, err := removeIfEmpty(root, rel)
			if err != nil {
				return result, fmt.Errorf("eject: prune %s: %w", rel, err)
			}
			if pruned {
				result.Pruned = append(result.Pruned, filepath.ToSlash(rel))
			}
		}
	}

	sort.Strings(result.Removed)
	sort.Strings(result.Missing)
	sort.Strings(result.Pruned)
	return result, nil
}

func removeIfEmpty(root *repofs.Root, rel string) (bool, error) {
	directory, err := root.OpenFile(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	entries, readErr := directory.ReadDir(1)
	closeErr := directory.Close()
	if closeErr != nil {
		return false, closeErr
	}
	if len(entries) > 0 {
		return false, nil
	}
	if !errors.Is(readErr, io.EOF) {
		return false, readErr
	}
	if err := root.Remove(rel); err != nil {
		return false, err
	}
	return true, nil
}
