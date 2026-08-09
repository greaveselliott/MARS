/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/design-docs/role-customization.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
*/
package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSkills_noDirectory(t *testing.T) {
	t.Parallel()
	skills, err := LoadSkills(t.TempDir(), "engineer")
	require.NoError(t, err)
	assert.Nil(t, skills)
}

func TestLoadSkills_happyPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".harness", "skills", "deploy")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))

	content := `---
name: deploy
description: How to deploy.
scope: all
---

# Deploy

Run: npm run deploy
`
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(content), 0o644))

	skills, err := LoadSkills(root, "engineer")
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "deploy", skills[0].Name)
	assert.Equal(t, "all", skills[0].Scope)
	assert.Contains(t, skills[0].Body, "npm run deploy")
}

func TestLoadSkills_scopeFilter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	qaDir := filepath.Join(root, ".harness", "skills", "test-setup")
	require.NoError(t, os.MkdirAll(qaDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(qaDir, "SKILL.md"), []byte(`---
name: test-setup
scope: qa
---
Run tests.
`), 0o644))

	allDir := filepath.Join(root, ".harness", "skills", "deploy")
	require.NoError(t, os.MkdirAll(allDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(allDir, "SKILL.md"), []byte(`---
name: deploy
scope: all
---
Deploy it.
`), 0o644))

	skills, err := LoadSkills(root, "engineer")
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "deploy", skills[0].Name)

	skills, err = LoadSkills(root, "qa")
	require.NoError(t, err)
	require.Len(t, skills, 2)
}

func TestLoadSkills_noFrontmatter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".harness", "skills", "simple")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("Just some instructions."), 0o644))

	skills, err := LoadSkills(root, "engineer")
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "simple", skills[0].Name)
	assert.Equal(t, "Just some instructions.", skills[0].Body)
}

func TestLoadSkills_emptyScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".harness", "skills", "generic")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(`---
name: generic
---
Available to all.
`), 0o644))

	skills, err := LoadSkills(root, "any-role")
	require.NoError(t, err)
	require.Len(t, skills, 1)
}

func TestLoadSkills_rejectsSymlinkLeaf(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".harness", "skills", "outside")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	outside := filepath.Join(t.TempDir(), "SKILL.md")
	require.NoError(t, os.WriteFile(outside, []byte("outside instructions"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(skillsDir, "SKILL.md")))

	skills, err := LoadSkills(root, "engineer")
	require.ErrorContains(t, err, "symbolic links are not allowed")
	require.Empty(t, skills)
	require.FileExists(t, outside)
}
