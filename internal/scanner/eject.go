/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-004-target-harness-lifecycle.md
*/
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// EjectOptions controls removal of a deployed Mars Harness from a target repo.
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
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return EjectResult{}, fmt.Errorf("eject: resolve repo root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return EjectResult{}, fmt.Errorf("eject: cannot access %s: %w — verify the path exists", absRoot, err)
	}
	if !info.IsDir() {
		return EjectResult{}, fmt.Errorf("eject: %s is not a directory — point to the repository root", absRoot)
	}

	result := EjectResult{RepoRoot: absRoot}
	for _, rel := range ejectHarnessPaths {
		rel = filepath.Clean(rel)
		abs := filepath.Join(absRoot, rel)
		if _, err := os.Lstat(abs); os.IsNotExist(err) {
			result.Missing = append(result.Missing, filepath.ToSlash(rel))
			continue
		} else if err != nil {
			return result, fmt.Errorf("eject: inspect %s: %w", abs, err)
		}
		result.Removed = append(result.Removed, filepath.ToSlash(rel))
		if opts.Apply {
			if err := os.RemoveAll(abs); err != nil {
				return result, fmt.Errorf("eject: remove %s: %w", abs, err)
			}
		}
	}

	if opts.Apply {
		for _, rel := range ejectPruneDirs {
			rel = filepath.Clean(rel)
			abs := filepath.Join(absRoot, rel)
			pruned, err := removeIfEmpty(abs)
			if err != nil {
				return result, fmt.Errorf("eject: prune %s: %w", abs, err)
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

func removeIfEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(entries) > 0 {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}
