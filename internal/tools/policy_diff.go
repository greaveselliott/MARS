/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars/internal/safety"
)

func validateRepoDiff(ctx context.Context, root Root, session Session) error {
	if err := checkDiffForSecrets(ctx, root); err != nil {
		return err
	}
	limits := session.SafetyLimits
	if limits == (safety.Limits{}) {
		limits = safety.DefaultLimits()
	}
	stats, err := diffStats(ctx, root)
	if err != nil {
		return err
	}
	if err := safety.Check(stats, limits); err != nil {
		if hint := buildArtifactCleanupHint(ctx, root, stats, limits); hint != "" {
			return fmt.Errorf("%w. %s", err, hint)
		}
		return err
	}
	return nil
}

// ValidateRepoDiff checks the current repository diff against the same safety
// limits enforced after mutating tool calls.
func ValidateRepoDiff(ctx context.Context, root Root, session Session) error {
	return validateRepoDiff(ctx, root, session)
}

func buildArtifactCleanupHint(ctx context.Context, root Root, stats safety.DiffStats, limits safety.Limits) string {
	if limits.MaxLinesPerFile <= 0 {
		return ""
	}
	for rel, lines := range stats.LinesPerFile {
		if lines <= limits.MaxLinesPerFile {
			continue
		}
		generated, err := isUntrackedRootBuildArtifact(ctx, root, rel)
		if err != nil || !generated {
			continue
		}
		return fmt.Sprintf("Generated build artifact %q can be cleaned with `rm %s`, then rerun the blocked command", rel, rel)
	}
	return ""
}

func checkDiffForSecrets(ctx context.Context, root Root) error {
	files, err := changedFiles(ctx, root)
	if err != nil {
		return err
	}
	for _, rel := range files {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			return err
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if hits := safety.ScanForSecrets(rel, string(b)); len(hits) > 0 {
			return fmt.Errorf("policy: secret scanner blocked %s:%d (%s)", hits[0].File, hits[0].Line, hits[0].Pattern)
		}
	}
	return nil
}

func diffStats(ctx context.Context, root Root) (safety.DiffStats, error) {
	stats := safety.DiffStats{LinesPerFile: map[string]int{}}
	numstat, err := runGit(ctx, root, "diff", "--numstat", "HEAD", "--")
	if err != nil {
		return stats, err
	}
	if numstat.ExitCode != 0 {
		return stats, nil
	}
	for _, line := range strings.Split(numstat.Output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		added := atoiDiffField(fields[0])
		deleted := atoiDiffField(fields[1])
		path := strings.Join(fields[2:], " ")
		if IsGeneratedWorkspacePath(path) || IsGeneratedDependencyMetadataPath(path) {
			continue
		}
		lines := added + deleted
		stats.FilesChanged++
		stats.LinesPerFile[path] = lines
		stats.TotalLines += lines
	}
	untracked, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return stats, err
	}
	var untrackedPaths []string
	if untracked.ExitCode == 0 {
		for _, rel := range strings.Split(untracked.Output, "\n") {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}
			untrackedPaths = append(untrackedPaths, rel)
		}
	}
	status, err := runGit(ctx, root, "diff", "--name-status", "HEAD", "--")
	if err != nil {
		return stats, err
	}
	if status.ExitCode != 0 {
		return stats, nil
	}
	var lifecycleCounterpartPaths []string
	lifecycleCounterpartPaths = append(lifecycleCounterpartPaths, untrackedPaths...)
	for _, line := range strings.Split(status.Output, "\n") {
		trimmed := strings.TrimSpace(line)
		fields := strings.Fields(trimmed)
		code := ""
		if len(fields) > 0 {
			code = fields[0]
		}
		for _, path := range diffNameStatusAddedPaths(fields) {
			if _, _, ok := ticketLifecyclePathIdentity(path); ok {
				lifecycleCounterpartPaths = append(lifecycleCounterpartPaths, path)
			}
		}
		if !strings.HasPrefix(code, "D") || len(fields) < 2 {
			continue
		}
		path := fields[len(fields)-1]
		if IsGeneratedWorkspacePath(path) {
			continue
		}
		if isTicketLifecycleMoveDeletion(root, path, lifecycleCounterpartPaths) {
			continue
		}
		stats.Deletions++
	}
	if untracked.ExitCode == 0 {
		for _, rel := range untrackedPaths {
			rel = strings.TrimSpace(rel)
			if rel == "" || IsGeneratedWorkspacePath(rel) || IsGeneratedDependencyMetadataPath(rel) {
				continue
			}
			abs, err := root.ResolvePath(rel)
			if err != nil {
				return stats, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			lines := strings.Count(string(b), "\n")
			if len(b) > 0 && !strings.HasSuffix(string(b), "\n") {
				lines++
			}
			stats.FilesChanged++
			stats.LinesPerFile[rel] = lines
			stats.TotalLines += lines
		}
	}
	return stats, nil
}

func diffNameStatusAddedPaths(fields []string) []string {
	if len(fields) < 2 {
		return nil
	}
	code := fields[0]
	switch {
	case strings.HasPrefix(code, "A"):
		return []string{fields[len(fields)-1]}
	case strings.HasPrefix(code, "R"), strings.HasPrefix(code, "C"):
		if len(fields) >= 3 {
			return []string{fields[len(fields)-1]}
		}
	}
	return nil
}

func isTicketLifecycleMoveDeletion(root Root, deletedPath string, candidatePaths []string) bool {
	deletedID, deletedState, ok := ticketLifecyclePathIdentity(deletedPath)
	if !ok {
		return false
	}
	if ticketLifecycleCounterpartInCandidates(deletedPath, deletedID, deletedState, candidatePaths) {
		return true
	}
	return ticketLifecycleCounterpartExists(root, deletedPath, deletedID, deletedState)
}

func ticketLifecycleCounterpartInCandidates(deletedPath, deletedID, deletedState string, candidatePaths []string) bool {
	for _, rel := range candidatePaths {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" || rel == filepath.ToSlash(deletedPath) {
			continue
		}
		addedID, addedState, ok := ticketLifecyclePathIdentity(rel)
		if !ok {
			continue
		}
		if addedID == deletedID && addedState != deletedState {
			return true
		}
	}
	return false
}

func ticketLifecycleCounterpartExists(root Root, deletedPath, deletedID, deletedState string) bool {
	pattern := filepath.Join(root.Abs(), "docs", "tickets", "*", "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	deletedPath = filepath.ToSlash(deletedPath)
	for _, match := range matches {
		rel, err := filepath.Rel(root.Abs(), match)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == deletedPath {
			continue
		}
		id, state, ok := ticketLifecyclePathIdentity(rel)
		if ok && id == deletedID && state != deletedState {
			return true
		}
	}
	return false
}

func atoiDiffField(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
