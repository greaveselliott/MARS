/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/role-customization.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-019-typescript-monorepo-docsync.md
*/
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
docsync:
  include_roots: [apps, packages]
  include_extensions: [.ts, .tsx]
  exclude_globs: ["**/dist/**"]
roles:
  fixer:
    prompt: roles/fixer.md
    domain: engineer
    mode: pipeline-repair
    trust_level: contributor
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
	assert.Equal(t, []string{"apps", "packages"}, m.DocSync.IncludeRoots)
	assert.Equal(t, []string{".ts", ".tsx"}, m.DocSync.IncludeExtensions)
	assert.Equal(t, []string{"**/dist/**"}, m.DocSync.ExcludeGlobs)

	fixer := m.Roles["fixer"]
	assert.Equal(t, "roles/fixer.md", fixer.Prompt)
	assert.Equal(t, "engineer", fixer.Domain)
	assert.Equal(t, "pipeline-repair", fixer.Mode)
	assert.Equal(t, "contributor", fixer.TrustLevel)
	assert.Equal(t, []string{"file_read", "shell_exec"}, fixer.Tools)
	assert.Equal(t, []string{`workflow_run.conclusion == "failure"`}, fixer.Triggers)

	reviewer := m.Roles["reviewer"]
	assert.Equal(t, "qwen3-coder", reviewer.Model)
}

func TestLoad_DispatchOrchestrationMode(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: dispatch-project
orchestration_mode: dispatch
roles:
  engineer:
    prompt: roles/engineer.md
`)

	m, err := Load(root)
	require.NoError(t, err)
	require.True(t, m.DispatchMode())
}

func TestLoadDocSyncConfigDoesNotRequireRoleValidation(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `docsync:
  include_roots: [apps]
  include_extensions: [.ts, .tsx]
  exclude_globs: ["**/dist/**"]
`)

	config, found, err := LoadDocSyncConfig(root)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"apps"}, config.IncludeRoots)
	require.Equal(t, []string{".ts", ".tsx"}, config.IncludeExtensions)
	require.Equal(t, []string{"**/dist/**"}, config.ExcludeGlobs)
}

func TestLoadDocSyncConfigMissingManifestUsesDefaultsAtCaller(t *testing.T) {
	config, found, err := LoadDocSyncConfig(t.TempDir())
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, config.IncludeRoots)
}

func TestLoad_InvalidOrchestrationMode(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: bad-dispatch-project
orchestration_mode: freestyle
roles:
  engineer:
    prompt: roles/engineer.md
`)

	_, err := Load(root)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid orchestration_mode")
}

func TestLoad_InvalidTrustLevel(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: bad-trust-project
roles:
  engineer:
    prompt: roles/engineer.md
    trust_level: root
`)

	_, err := Load(root)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid trust_level")
}

func TestLoad_MissingHarnessDir(t *testing.T) {
	root := t.TempDir()

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing .harness/ directory")
	assert.Contains(t, err.Error(), "mars init")
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

func TestLoad_ThenField(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: chain-test
roles:
  fixer:
    prompt: roles/fixer.md
    triggers:
      - workflow_run.conclusion == "failure"
    then: [qa]
  qa:
    prompt: roles/qa.md
    triggers:
      - pull_request.opened
`)

	m, err := Load(root)
	require.NoError(t, err)
	assert.Equal(t, []string{"qa"}, m.Roles["fixer"].Then)
	assert.Empty(t, m.Roles["qa"].Then)
}

func TestLoad_ThenBadReference(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: bad-chain
roles:
  fixer:
    prompt: roles/fixer.md
    then: [nonexistent]
`)

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chains to \"nonexistent\"")
	assert.Contains(t, err.Error(), "not defined in the manifest")
}

func TestLoad_SchedulePreset(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: sched-test
roles:
  daily-check:
    prompt: roles/check.md
    schedule: daily
`)

	m, err := Load(root)
	require.NoError(t, err)
	assert.Equal(t, "daily", m.Roles["daily-check"].Schedule)
}

func TestLoad_ScheduleCustomCron(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: cron-test
roles:
  engineer:
    prompt: roles/engineer.md
    schedule: "0 0,6,12,18 * * 1-5"
`)

	m, err := Load(root)
	require.NoError(t, err)
	assert.Equal(t, "0 0,6,12,18 * * 1-5", m.Roles["engineer"].Schedule)
}

func TestLoad_ScheduleInvalid(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: bad-cron
roles:
  broken:
    prompt: roles/broken.md
    schedule: "not a cron"
`)

	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule")
}

func TestLoad_DualModeRoles(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: dual-mode
roles:
  cto-pr-merge:
    prompt: roles/cto.md
    model: coding
    triggers:
      - pull_request.merged
  cto-weekly:
    prompt: roles/cto.md
    model: reasoning
    schedule: "0 21 * * 0"
`)

	m, err := Load(root)
	require.NoError(t, err)
	assert.Len(t, m.Roles, 2)
	assert.Equal(t, "coding", m.Roles["cto-pr-merge"].Model)
	assert.Equal(t, "reasoning", m.Roles["cto-weekly"].Model)
	assert.Equal(t, "0 21 * * 0", m.Roles["cto-weekly"].Schedule)
}

func TestLoad_AllMarsSchedules(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".harness/manifest.yaml", `
name: mars-schedules
roles:
  ceo:
    prompt: roles/ceo.md
    schedule: "0 20 * * 0"
  cto-weekly:
    prompt: roles/cto.md
    schedule: "0 21 * * 0"
  engineer:
    prompt: roles/eng.md
    schedule: "0 0,6,12,18 * * 1-5"
  security-weekly:
    prompt: roles/sec.md
    schedule: "0 22 * * 0"
  release-weekly:
    prompt: roles/rel.md
    schedule: "0 8 * * 1"
  dogfood:
    prompt: roles/dog.md
    schedule: "0 10 * * 1-5"
  preset-test:
    prompt: roles/preset.md
    schedule: weekly
`)

	m, err := Load(root)
	require.NoError(t, err)
	assert.Len(t, m.Roles, 7)
}
