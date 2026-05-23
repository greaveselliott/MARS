/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/harness-glossary.md
- docs/design-docs/harness-operating-model.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
- docs/roles/ROLES.md
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFoundationVendorAdaptersStayThin(t *testing.T) {
	root := repoRoot(t)
	adapters := []string{
		"CLAUDE.md",
		"GEMINI.md",
		".github/copilot-instructions.md",
	}
	cursorRules, err := filepath.Glob(filepath.Join(root, ".cursor", "rules", "*.mdc"))
	if err != nil {
		t.Fatalf("glob cursor rules: %v", err)
	}
	if len(cursorRules) == 0 {
		t.Fatal("expected Cursor adapter rules")
	}
	for _, path := range cursorRules {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("rel cursor rule: %v", err)
		}
		adapters = append(adapters, filepath.ToSlash(rel))
	}

	for _, rel := range adapters {
		text := readRepoText(t, root, rel)
		if !strings.Contains(text, "AGENTS.md") {
			t.Fatalf("%s must point at AGENTS.md", rel)
		}
		if !strings.Contains(text, "docs/roles/personas/foundation-maintainer.md") {
			t.Fatalf("%s must point at the foundation maintainer role packet", rel)
		}
		if nonBlankLineCount(text) > 12 {
			t.Fatalf("%s must stay a thin adapter; got %d non-blank lines", rel, nonBlankLineCount(text))
		}
		for _, forbidden := range []string{"## Working Discipline", "## The Nine Tenets", "## Operations", "## Directory Structure"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s duplicates canonical doctrine section %q", rel, forbidden)
			}
		}
	}
}

func TestFoundationSupportedClientMatrixIsComplete(t *testing.T) {
	root := repoRoot(t)
	agents := readRepoText(t, root, "AGENTS.md")
	if !strings.Contains(agents, "Foundation Mode For AI Clients") {
		t.Fatal("AGENTS.md must document foundation mode for AI clients")
	}
	for _, client := range []string{
		"Claude Code (recommended)",
		"Cursor",
		"Gemini CLI",
		"Windsurf",
		"OpenCode",
		"GitHub Copilot",
		"Kiro IDE & CLI",
		"Codex / Other Agents",
	} {
		if !strings.Contains(agents, client) {
			t.Fatalf("foundation client matrix missing %q", client)
		}
	}
}

func TestValidationSubjectVocabularyStaysOutOfGenericFoundationSurfaces(t *testing.T) {
	root := repoRoot(t)
	var surfaces []string
	surfaces = append(surfaces,
		"AGENTS.md",
		"README.md",
		"CLAUDE.md",
		"GEMINI.md",
		".github/copilot-instructions.md",
		"internal/scanner/init.go",
		"internal/personas/personas.go",
		"internal/tools/policy.go",
	)
	surfaces = append(surfaces, markdownFilesUnder(t, root, ".cursor/rules")...)
	surfaces = append(surfaces, markdownFilesUnder(t, root, "docs/features")...)
	surfaces = append(surfaces, markdownFilesUnder(t, root, "docs/roles")...)

	blocked := []string{
		"tetris",
		"tetromino",
		"space invaders",
		"demo-123",
		"demo-tetris",
		"phaser-tetris",
		"phaser tetris",
	}
	for _, rel := range surfaces {
		text := strings.ToLower(readRepoText(t, root, rel))
		for _, term := range blocked {
			if strings.Contains(text, term) {
				t.Fatalf("%s contains validation-subject vocabulary %q; keep demo subjects in evidence surfaces, not generic doctrine", rel, term)
			}
		}
	}
}

func readRepoText(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func nonBlankLineCount(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func markdownFilesUnder(t *testing.T, root, relDir string) []string {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	var out []string
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".md" && ext != ".mdc" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", relDir, err)
	}
	return out
}
