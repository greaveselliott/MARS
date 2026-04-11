package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFixture(t *testing.T, root, relPath, content string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
}

func TestLoad_ValidManifest(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: test-project
description: A test bundle
roles:
  fixer:
    prompt: roles/fixer.md
    tools:
      - file_read
      - shell_exec
    triggers:
      - workflow_run.conclusion == "failure"
  reviewer:
    prompt: roles/reviewer.md
    model: qwen3-coder
    tools:
      - file_read
      - grep
`)

	m, err := Load(root)
	require.NoError(t, err)
	assert.Equal(t, "test-project", m.Name)
	assert.Equal(t, "A test bundle", m.Description)
	assert.Len(t, m.Roles, 2)

	fixer := m.Roles["fixer"]
	assert.Equal(t, "roles/fixer.md", fixer.Prompt)
	assert.Equal(t, []string{"file_read", "shell_exec"}, fixer.Tools)
	assert.Equal(t, []string{`workflow_run.conclusion == "failure"`}, fixer.Triggers)

	reviewer := m.Roles["reviewer"]
	assert.Equal(t, "qwen3-coder", reviewer.Model)
}

func TestLoad_MissingHarnessDir(t *testing.T) {
	root := t.TempDir()

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing .harness/ directory")
	assert.Contains(t, err.Error(), "mars-harness init")
}

func TestLoad_MissingManifestFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o755))

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing manifest.yaml")
}

func TestLoad_InvalidYAML(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", "{{invalid yaml")

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check YAML syntax")
}

func TestLoad_MissingName(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
description: no name field
roles:
  fixer:
    prompt: roles/fixer.md
`)

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field 'name'")
}

func TestLoad_NoRoles(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: empty-roles
description: No roles defined
roles: {}
`)

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "defines no roles")
}

func TestLoad_RoleMissingPrompt(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: bad-role
roles:
  fixer:
    tools:
      - file_read
`)

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no prompt path")
}

func TestLoad_EmptyRepoRoot(t *testing.T) {
	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo root path is empty")
}

func TestRolePrompt_ReadsFile(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: test
roles:
  fixer:
    prompt: roles/fixer.md
`)
	writeFixture(t, root, ".harness/roles/fixer.md", "You are a CI fixer agent.")

	m, err := Load(root)
	require.NoError(t, err)

	prompt, err := m.RolePrompt(root, "fixer")
	require.NoError(t, err)
	assert.Equal(t, "You are a CI fixer agent.", prompt)
}

func TestRolePrompt_UnknownRole(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: test
roles:
  fixer:
    prompt: roles/fixer.md
`)

	m, err := Load(root)
	require.NoError(t, err)

	_, err = m.RolePrompt(root, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in manifest")
	assert.Contains(t, err.Error(), "fixer")
}

func TestRolePrompt_MissingFile(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: test
roles:
  fixer:
    prompt: roles/fixer.md
`)

	m, err := Load(root)
	require.NoError(t, err)

	_, err = m.RolePrompt(root, "fixer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRolePrompt_EmptyFile(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: test
roles:
  fixer:
    prompt: roles/fixer.md
`)
	writeFixture(t, root, ".harness/roles/fixer.md", "   \n  ")

	m, err := Load(root)
	require.NoError(t, err)

	_, err = m.RolePrompt(root, "fixer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}

func TestLoad_HarnessDirIsFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness"), []byte("oops"), 0o644))

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}
