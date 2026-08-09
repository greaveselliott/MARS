/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-017-open-source-publication.md
*/
// Package repofs provides descriptor-bound operations inside one repository.
package repofs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const tempCreateAttempts = 10

// Root binds repository operations to the directory descriptor opened during
// admission so replacing the original pathname cannot redirect later access.
type Root struct {
	abs        string
	descriptor *os.Root
}

var _ fs.FS = (*Root)(nil)

// Open validates and canonicalizes a repository root.
func Open(path string) (*Root, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("repofs: resolve root: %w", err)
	}
	if evaluated, err := filepath.EvalSymlinks(abs); err == nil {
		abs = evaluated
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("repofs: inspect root: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("repofs: root is not a directory")
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("repofs: open root: %w", err)
	}
	return &Root{abs: filepath.Clean(abs), descriptor: root}, nil
}

// Abs returns the canonical absolute repository path for subprocess working
// directories and user-facing repository identity. File access must use the
// descriptor-relative methods below.
func (r *Root) Abs() string { return r.abs }

// Open implements fs.FS for descriptor-bound standard-library traversal.
func (r *Root) Open(name string) (fs.File, error) {
	return r.OpenFile(filepath.FromSlash(name))
}

// VerifyPath proves the pathname used by repository subprocesses still names
// the directory bound to this Root.
func (r *Root) VerifyPath() error {
	bound, err := r.descriptor.Stat(".")
	if err != nil {
		return errors.New("repofs: inspect bound repository identity")
	}
	current, err := os.Stat(r.abs)
	if err != nil || !os.SameFile(bound, current) {
		return errors.New("repofs: repository pathname identity changed")
	}
	return nil
}

// Resolve returns a lexically contained absolute path for legacy call sites.
// It does not grant safe file access; callers migrate to the methods below.
func (r *Root) Resolve(name string) (string, error) {
	clean, err := cleanName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.abs, clean), nil
}

// OpenFile opens an existing repository entry without accepting a symlink in
// any observed path component.
func (r *Root) OpenFile(name string) (*os.File, error) {
	clean, err := cleanName(name)
	if err != nil {
		return nil, err
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return nil, err
	}
	file, err := root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("repofs: open repository entry: %w", err)
	}
	return file, nil
}

// ReadFile reads an existing repository file.
func (r *Root) ReadFile(name string) ([]byte, error) {
	file, err := r.OpenFile(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("repofs: read repository entry: %w", err)
	}
	return data, nil
}

// Stat returns metadata without accepting observed symlink components.
func (r *Root) Stat(name string) (fs.FileInfo, error) {
	clean, err := cleanName(name)
	if err != nil {
		return nil, err
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return nil, err
	}
	info, err := root.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("repofs: stat repository entry: %w", err)
	}
	return info, nil
}

// Lstat returns metadata for an entry after rejecting symlink parents. The
// leaf may itself be a symlink so callers can classify it without following it.
func (r *Root) Lstat(name string) (fs.FileInfo, error) {
	clean, err := cleanName(name)
	if err != nil {
		return nil, err
	}
	root := r.descriptor
	if err := rejectObservedParentSymlinks(root, clean); err != nil {
		return nil, err
	}
	info, err := root.Lstat(clean)
	if err != nil {
		return nil, fmt.Errorf("repofs: lstat repository entry: %w", err)
	}
	return info, nil
}

// Glob returns sorted repository-relative matches. A literal prefix containing
// a symlink is rejected before traversal; every returned match is rechecked.
func (r *Root) Glob(pattern string) ([]string, error) {
	clean, err := cleanName(pattern)
	if err != nil {
		return nil, err
	}
	root := r.descriptor
	if prefix := literalGlobPrefix(clean); prefix != "." {
		if err := rejectObservedSymlinks(root, prefix); err != nil {
			return nil, err
		}
	}
	matches, err := fs.Glob(root.FS(), filepath.ToSlash(clean))
	if err != nil {
		return nil, fmt.Errorf("repofs: invalid glob: %w", err)
	}
	for _, match := range matches {
		if err := rejectObservedSymlinks(root, filepath.FromSlash(match)); err != nil {
			return nil, err
		}
	}
	sort.Strings(matches)
	return matches, nil
}

// MkdirAll creates repository directories without following an observed
// symlink parent or leaf.
func (r *Root) MkdirAll(name string, perm fs.FileMode) error {
	clean, err := cleanName(name)
	if err != nil {
		return err
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return err
	}
	if err := root.MkdirAll(clean, perm); err != nil {
		return fmt.Errorf("repofs: create repository directory: %w", err)
	}
	return rejectObservedSymlinks(root, clean)
}

// CreateExclusive creates a new owner-selected repository file.
func (r *Root) CreateExclusive(name string, perm fs.FileMode) (*os.File, error) {
	clean, err := cleanName(name)
	if err != nil {
		return nil, err
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return nil, err
	}
	file, err := root.OpenFile(clean, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return nil, fmt.Errorf("repofs: create repository file: %w", err)
	}
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		_ = root.Remove(clean)
		return nil, fmt.Errorf("repofs: set repository file mode: %w", err)
	}
	return file, nil
}

// AtomicWrite durably replaces a repository file through an exclusive
// same-directory temporary entry.
func (r *Root) AtomicWrite(name string, data []byte, perm fs.FileMode) error {
	clean, err := cleanName(name)
	if err != nil {
		return err
	}
	parent := filepath.Dir(clean)
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return err
	}

	var file *os.File
	var temp string
	for attempt := 0; attempt < tempCreateAttempts; attempt++ {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return errors.New("repofs: generate atomic-write nonce")
		}
		temp = filepath.Join(parent, "."+filepath.Base(clean)+".tmp-"+hex.EncodeToString(nonce[:]))
		file, err = root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("repofs: create atomic-write temporary file: %w", err)
		}
	}
	if file == nil {
		return errors.New("repofs: atomic-write temporary-name retries exhausted")
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = root.Remove(temp)
		}
	}()
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		return fmt.Errorf("repofs: set atomic-write mode: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("repofs: write atomic repository file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("repofs: sync atomic repository file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("repofs: close atomic repository file: %w", err)
	}
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return err
	}
	if err := root.Rename(temp, clean); err != nil {
		return fmt.Errorf("repofs: install atomic repository file: %w", err)
	}
	cleanup = false
	if err := syncDirectory(root, parent); err != nil {
		return err
	}
	return nil
}

// Chmod changes a repository entry mode after rejecting observed symlinks.
func (r *Root) Chmod(name string, mode fs.FileMode) error {
	clean, root, err := r.admitExisting(name)
	if err != nil {
		return err
	}
	if err := root.Chmod(clean, mode); err != nil {
		return fmt.Errorf("repofs: chmod repository entry: %w", err)
	}
	return nil
}

// Rename atomically renames one repository entry to another.
func (r *Root) Rename(oldName, newName string) error {
	oldClean, err := cleanName(oldName)
	if err != nil {
		return err
	}
	newClean, err := cleanName(newName)
	if err != nil {
		return err
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, oldClean); err != nil {
		return err
	}
	if err := rejectObservedSymlinks(root, newClean); err != nil {
		return err
	}
	if err := root.Rename(oldClean, newClean); err != nil {
		return fmt.Errorf("repofs: rename repository entry: %w", err)
	}
	if err := syncDirectory(root, filepath.Dir(oldClean)); err != nil {
		return err
	}
	if filepath.Dir(newClean) != filepath.Dir(oldClean) {
		return syncDirectory(root, filepath.Dir(newClean))
	}
	return nil
}

// Remove deletes one repository entry without accepting symlinks.
func (r *Root) Remove(name string) error {
	clean, root, err := r.admitExisting(name)
	if err != nil {
		return err
	}
	if err := root.Remove(clean); err != nil {
		return fmt.Errorf("repofs: remove repository entry: %w", err)
	}
	return syncDirectory(root, filepath.Dir(clean))
}

// RemoveAll recursively removes one repository subtree. The selected entry
// and every observed parent must be real directories or files, not symlinks.
func (r *Root) RemoveAll(name string) error {
	clean, err := cleanName(name)
	if err != nil {
		return err
	}
	if clean == "." {
		return errors.New("repofs: refusing to remove repository root")
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := root.RemoveAll(clean); err != nil {
		return fmt.Errorf("repofs: remove repository subtree: %w", err)
	}
	return syncDirectory(root, filepath.Dir(clean))
}

func (r *Root) admitExisting(name string) (string, *os.Root, error) {
	clean, err := cleanName(name)
	if err != nil {
		return "", nil, err
	}
	root := r.descriptor
	if err := rejectObservedSymlinks(root, clean); err != nil {
		return "", nil, err
	}
	return clean, root, nil
}

func cleanName(name string) (string, error) {
	if name == "" {
		return "", errors.New("repofs: path is empty")
	}
	if strings.IndexByte(name, 0) >= 0 {
		return "", errors.New("repofs: path contains NUL")
	}
	clean := filepath.Clean(name)
	if filepath.IsAbs(clean) {
		return "", errors.New("repofs: path must stay relative to the repository root")
	}
	if !filepath.IsLocal(clean) {
		return "", errors.New("repofs: path escapes repository root")
	}
	return clean, nil
}

func rejectObservedSymlinks(root *os.Root, name string) error {
	return rejectObservedComponents(root, name, true)
}

func rejectObservedParentSymlinks(root *os.Root, name string) error {
	return rejectObservedComponents(root, name, false)
}

func rejectObservedComponents(root *os.Root, name string, includeLeaf bool) error {
	if name == "." {
		return nil
	}
	parts := strings.Split(filepath.Clean(name), string(filepath.Separator))
	limit := len(parts)
	if !includeLeaf {
		limit--
	}
	current := "."
	for index := 0; index < limit; index++ {
		current = filepath.Join(current, parts[index])
		info, err := root.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("repofs: inspect repository path component: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("repofs: symbolic links are not allowed in repository operations")
		}
		if index < len(parts)-1 && !info.IsDir() {
			return errors.New("repofs: repository path parent is not a directory")
		}
	}
	return nil
}

func literalGlobPrefix(pattern string) string {
	parts := strings.Split(pattern, string(filepath.Separator))
	prefix := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.ContainsAny(part, "*?[") {
			break
		}
		prefix = append(prefix, part)
	}
	if len(prefix) == 0 {
		return "."
	}
	return filepath.Join(prefix...)
}

func syncDirectory(root *os.Root, name string) error {
	if name == "" {
		name = "."
	}
	directory, err := root.Open(name)
	if err != nil {
		return fmt.Errorf("repofs: open repository directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("repofs: sync repository directory: %w", err)
	}
	return nil
}
