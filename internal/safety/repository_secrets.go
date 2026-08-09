/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/features/F-007-guardrails-and-safety.md
- docs/features/F-017-open-source-publication.md
*/
package safety

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars/internal/repofs"
)

// RepositorySecretScanMode selects the repository view scanned for secrets.
type RepositorySecretScanMode uint8

const (
	// RepositorySecretScanFull scans the complete index plus dirty tracked and
	// ordinary non-ignored untracked worktree files.
	RepositorySecretScanFull RepositorySecretScanMode = iota
	// RepositorySecretScanStaged scans only resulting staged blobs, plus the
	// tracked local credential file when it exists in the index.
	RepositorySecretScanStaged
)

// RepositorySecretFinding intentionally omits the candidate value and object
// identifier so callers cannot accidentally render either one.
type RepositorySecretFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Pattern string `json:"pattern"`
}

type repositoryIndexEntry struct {
	mode string
	oid  string
}

// ScanRepositoryForSecrets scans Git-authoritative bytes without copying
// candidate values or object identifiers into results or errors.
func ScanRepositoryForSecrets(ctx context.Context, root *repofs.Root, mode RepositorySecretScanMode) ([]RepositorySecretFinding, error) {
	if mode != RepositorySecretScanFull && mode != RepositorySecretScanStaged {
		return nil, errors.New("repository secret scan: unsupported scan mode")
	}
	if root == nil || root.VerifyPath() != nil {
		return nil, errors.New("repository secret scan: repository admission failed")
	}
	admission, err := repositoryGitOutput(ctx, root, "rev-parse", "--is-inside-work-tree", "--show-prefix")
	if err != nil || string(admission) != "true\n\n" {
		return nil, errors.New("repository secret scan: Git worktree-root admission failed")
	}

	index, err := repositoryIndex(ctx, root)
	if err != nil {
		return nil, err
	}
	scanner := repositoryBlobScanner{
		ctx:   ctx,
		root:  root,
		cache: make(map[string][]byte),
	}

	if mode == RepositorySecretScanStaged {
		changes, err := repositoryChanges(ctx, root, true)
		if err != nil {
			return nil, err
		}
		paths := make(map[string]bool)
		for _, change := range changes {
			if change.deleted {
				if _, exists := index[change.path]; exists {
					return nil, errors.New("repository secret scan: staged deletion reconciliation failed")
				}
				continue
			}
			paths[change.path] = true
		}
		if _, tracked := index[localCredentialPath]; tracked {
			paths[localCredentialPath] = true
		}
		for _, path := range sortedRepositoryPaths(paths) {
			entry, ok := index[path]
			if !ok {
				return nil, errors.New("repository secret scan: staged entry reconciliation failed")
			}
			if err := scanner.scanIndex(path, entry); err != nil {
				return nil, err
			}
		}
		return scanner.sortedFindings(), nil
	}

	indexPaths := make([]string, 0, len(index))
	for path := range index {
		indexPaths = append(indexPaths, path)
	}
	sort.Strings(indexPaths)
	for _, path := range indexPaths {
		if err := scanner.scanIndex(path, index[path]); err != nil {
			return nil, err
		}
	}

	dirty, err := repositoryChanges(ctx, root, false)
	if err != nil {
		return nil, err
	}
	for _, change := range dirty {
		if change.deleted {
			continue
		}
		if err := scanner.scanWorktree(root, change.path); err != nil {
			return nil, err
		}
	}
	hidden, err := repositoryHiddenWorktreePaths(ctx, root)
	if err != nil {
		return nil, err
	}
	for _, path := range hidden {
		if _, err := root.Stat(filepath.FromSlash(path)); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, errors.New("repository secret scan: hidden worktree entry inspection failed")
		}
		if err := scanner.scanWorktree(root, path); err != nil {
			return nil, err
		}
	}
	untracked, err := repositoryGitOutput(ctx, root, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		return nil, errors.New("repository secret scan: untracked-file inventory failed")
	}
	for _, path := range splitRepositoryNUL(untracked) {
		if err := scanner.scanWorktree(root, path); err != nil {
			return nil, err
		}
	}
	return scanner.sortedFindings(), nil
}

func repositoryHiddenWorktreePaths(ctx context.Context, root *repofs.Root) ([]string, error) {
	raw, err := repositoryGitOutput(ctx, root, "ls-files", "-v", "-z", "--")
	if err != nil {
		return nil, errors.New("repository secret scan: tracked-file flag inventory failed")
	}
	seen := make(map[string]bool)
	for _, record := range splitRepositoryNUL(raw) {
		if len(record) < 3 || record[1] != ' ' {
			return nil, errors.New("repository secret scan: malformed tracked-file flag inventory")
		}
		tag := record[0]
		if tag != 'S' && (tag < 'a' || tag > 'z') {
			continue
		}
		path := filepath.ToSlash(record[2:])
		if path == "" {
			return nil, errors.New("repository secret scan: malformed tracked-file flag inventory")
		}
		seen[path] = true
	}
	return sortedRepositoryPaths(seen), nil
}

const localCredentialPath = ".harness/.env.local"

type repositoryChange struct {
	path    string
	deleted bool
}

func repositoryChanges(ctx context.Context, root *repofs.Root, staged bool) ([]repositoryChange, error) {
	args := []string{"diff", "--name-status", "-z", "--find-renames", "--find-copies"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--")
	raw, err := repositoryGitOutput(ctx, root, args...)
	if err != nil {
		return nil, errors.New("repository secret scan: Git change inventory failed")
	}
	fields := splitRepositoryNUL(raw)
	changes := make([]repositoryChange, 0, len(fields)/2)
	for index := 0; index < len(fields); {
		status := fields[index]
		index++
		if status == "" {
			return nil, errors.New("repository secret scan: malformed Git change inventory")
		}
		kind := status[0]
		switch kind {
		case 'A', 'M', 'D', 'T':
			if len(status) != 1 || index >= len(fields) {
				return nil, errors.New("repository secret scan: malformed Git change inventory")
			}
			path := fields[index]
			index++
			if path == "" {
				return nil, errors.New("repository secret scan: malformed Git change inventory")
			}
			changes = append(changes, repositoryChange{path: filepath.ToSlash(path), deleted: kind == 'D'})
		case 'C', 'R':
			if !validSimilarityStatus(status) || index+1 >= len(fields) {
				return nil, errors.New("repository secret scan: malformed Git change inventory")
			}
			index++ // The source is not part of the resulting staged/worktree view.
			destination := fields[index]
			index++
			if destination == "" {
				return nil, errors.New("repository secret scan: malformed Git change inventory")
			}
			changes = append(changes, repositoryChange{path: filepath.ToSlash(destination)})
		default:
			return nil, errors.New("repository secret scan: unsupported Git change state")
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].path < changes[j].path })
	return changes, nil
}

func validSimilarityStatus(status string) bool {
	if len(status) < 2 {
		return false
	}
	percent, err := strconv.Atoi(status[1:])
	return err == nil && percent >= 0 && percent <= 100
}

func repositoryIndex(ctx context.Context, root *repofs.Root) (map[string]repositoryIndexEntry, error) {
	raw, err := repositoryGitOutput(ctx, root, "ls-files", "--stage", "-z", "--")
	if err != nil {
		return nil, errors.New("repository secret scan: Git index inventory failed")
	}
	entries := make(map[string]repositoryIndexEntry)
	for _, record := range splitRepositoryNUL(raw) {
		tab := strings.IndexByte(record, '\t')
		if tab <= 0 || tab == len(record)-1 {
			return nil, errors.New("repository secret scan: malformed Git index inventory")
		}
		metadata := strings.Fields(record[:tab])
		if len(metadata) != 3 || metadata[2] != "0" || !validGitOID(metadata[1]) {
			return nil, errors.New("repository secret scan: unsupported Git index state")
		}
		path := filepath.ToSlash(record[tab+1:])
		if path == "" {
			return nil, errors.New("repository secret scan: malformed Git index inventory")
		}
		if _, duplicate := entries[path]; duplicate {
			return nil, errors.New("repository secret scan: duplicate Git index entry")
		}
		entries[path] = repositoryIndexEntry{mode: metadata[0], oid: metadata[1]}
	}
	return entries, nil
}

func validGitOID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type repositoryBlobScanner struct {
	ctx      context.Context
	root     *repofs.Root
	cache    map[string][]byte
	findings []RepositorySecretFinding
}

func (s *repositoryBlobScanner) scanIndex(path string, entry repositoryIndexEntry) error {
	if entry.mode != "100644" && entry.mode != "100755" {
		return errors.New("repository secret scan: unsupported Git index entry type")
	}
	data, ok := s.cache[entry.oid]
	if !ok {
		var err error
		data, err = repositoryGitOutput(s.ctx, s.root, "cat-file", "blob", entry.oid)
		if err != nil {
			return errors.New("repository secret scan: indexed blob read failed")
		}
		s.cache[entry.oid] = data
	}
	s.scan(path, data)
	return nil
}

func (s *repositoryBlobScanner) scanWorktree(root *repofs.Root, path string) error {
	info, err := root.Stat(filepath.FromSlash(path))
	if err != nil {
		return errors.New("repository secret scan: worktree entry inspection failed")
	}
	if !info.Mode().IsRegular() {
		return errors.New("repository secret scan: unsupported worktree entry type")
	}
	data, err := root.ReadFile(filepath.FromSlash(path))
	if err != nil {
		return errors.New("repository secret scan: worktree entry read failed")
	}
	s.scan(path, data)
	return nil
}

func (s *repositoryBlobScanner) scan(path string, data []byte) {
	for _, finding := range ScanForSecrets(path, string(data)) {
		s.findings = append(s.findings, RepositorySecretFinding{
			File:    finding.File,
			Line:    finding.Line,
			Pattern: finding.Pattern,
		})
	}
}

func (s *repositoryBlobScanner) sortedFindings() []RepositorySecretFinding {
	sort.Slice(s.findings, func(i, j int) bool {
		if s.findings[i].File != s.findings[j].File {
			return s.findings[i].File < s.findings[j].File
		}
		if s.findings[i].Line != s.findings[j].Line {
			return s.findings[i].Line < s.findings[j].Line
		}
		return s.findings[i].Pattern < s.findings[j].Pattern
	})
	return s.findings
}

func repositoryGitOutput(ctx context.Context, root *repofs.Root, args ...string) ([]byte, error) {
	if root.VerifyPath() != nil {
		return nil, errors.New("repository identity verification failed")
	}
	command := exec.CommandContext(ctx, "git", append([]string{"--no-replace-objects", "-C", root.Abs()}, args...)...)
	command.Stderr = io.Discard
	output, err := command.Output()
	if root.VerifyPath() != nil {
		return nil, errors.New("repository identity verification failed")
	}
	if err != nil {
		return nil, errors.New("Git command failed")
	}
	return output, nil
}

func splitRepositoryNUL(raw []byte) []string {
	parts := strings.Split(string(raw), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func sortedRepositoryPaths(set map[string]bool) []string {
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
