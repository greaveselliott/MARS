/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/harness-operating-model.md
- docs/product-specs/product-surface.md
- docs/roles/ROLES.md
*/
package personas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultPersonasValidate(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, p := range DefaultPersonas() {
		require.NoError(t, Validate(p), p.RoleKey)
		require.False(t, seen[p.RoleKey], "duplicate role key %s", p.RoleKey)
		seen[p.RoleKey] = true
	}
	require.ElementsMatch(t, []string{
		"ceo", "head-of-strategy", "coo", "cto-weekly", "engineer", "qa",
		"security", "dependency-manager", "release-manager", "dogfood",
		"pipeline-fixer", "orchestrator", "janitor",
	}, DefaultRoleKeys())
}

func TestRenderedManualsIncludeRequiredSections(t *testing.T) {
	t.Parallel()

	requiredManualSections := []string{
		"## Modus Operandi",
		"## Priorities",
		"## Owns",
		"## Does Not Own",
		"## Best Feedback Format",
		"## Feedback I Need",
		"## Feedback I Give",
		"## Stop Conditions",
		"## Orchestrator Handoff",
	}
	requiredPromptSections := []string{
		"## Personal Guide",
		"### Modus Operandi",
		"### Priorities",
		"### Owns",
		"### Does Not Own",
		"### Best Feedback Format",
		"### How I Like To Receive Feedback",
		"### Feedback I Give",
		"### Stop Conditions",
		"### Orchestrator Handoff",
	}

	for _, p := range DefaultPersonas() {
		manual := RenderManual(p)
		prompt := RenderPromptManual(p)
		for _, section := range requiredManualSections {
			require.Contains(t, manual, section, p.RoleKey)
		}
		for _, section := range requiredPromptSections {
			require.Contains(t, prompt, section, p.RoleKey)
		}
	}
}

func TestSourcePersonaDocsMatchCanonicalDefinitions(t *testing.T) {
	repoRoot := findRepoRoot(t)
	for path, expected := range DefaultManualDocs() {
		abs := filepath.Join(repoRoot, filepath.FromSlash(path))
		if os.Getenv("UPDATE_PERSONA_DOCS") == "1" {
			require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755), path)
			require.NoError(t, os.WriteFile(abs, []byte(expected), 0o644), path)
		}
		actual, err := os.ReadFile(abs)
		require.NoError(t, err, path)
		require.Equal(t, expected, string(actual), path)
	}
}

func TestValidateRejectsMissingRequiredManualSections(t *testing.T) {
	t.Parallel()

	p := MustDefault("engineer")
	p.Owns = nil
	require.ErrorContains(t, Validate(p), "owns is required")

	p = MustDefault("qa")
	p.FeedbackINeed = nil
	require.ErrorContains(t, Validate(p), "feedback_i_need is required")

	p = MustDefault("ceo")
	p.StopConditions = nil
	require.ErrorContains(t, Validate(p), "stop_conditions is required")
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir || strings.TrimSpace(parent) == "" {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
