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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillDef is a skill loaded from .harness/skills/<name>/SKILL.md.
type SkillDef struct {
	Name  string
	Scope string
	Body  string
}

// LoadSkills reads all skills from .harness/skills/ in a repo.
// Each subdirectory containing a SKILL.md is treated as a skill.
// Returns nil (not error) if the directory doesn't exist.
func LoadSkills(repoRoot, roleScope string) ([]SkillDef, error) {
	dir := filepath.Join(repoRoot, harnessDir, "skills")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bundle: read skills directory: %w", err)
	}

	roleScope = strings.TrimSpace(strings.ToLower(roleScope))
	var skills []SkillDef

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("bundle: read skill %q: %w", entry.Name(), err)
		}

		name, scope, body := parseSkillMD(string(data), entry.Name())

		sc := strings.TrimSpace(strings.ToLower(scope))
		if sc != "" && sc != "all" && sc != roleScope && roleScope != "" {
			continue
		}

		skills = append(skills, SkillDef{
			Name:  name,
			Scope: scope,
			Body:  body,
		})
	}

	return skills, nil
}

func parseSkillMD(content, fallbackName string) (name, scope, body string) {
	name = fallbackName
	lines := strings.SplitAfter(content, "\n")

	inFrontmatter := false
	frontmatterDone := false
	bodyStart := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && trimmed == "---" {
			frontmatterDone = true
			bodyStart = i + 1
			break
		}
		if inFrontmatter {
			kv := parseFrontmatterLine(trimmed)
			if kv[0] == "name" && kv[1] != "" {
				name = kv[1]
			}
			if kv[0] == "scope" && kv[1] != "" {
				scope = kv[1]
			}
		}
	}

	if !frontmatterDone {
		body = strings.TrimSpace(content)
		return
	}

	if bodyStart < len(lines) {
		body = strings.TrimSpace(strings.Join(lines[bodyStart:], ""))
	}

	return
}

func parseFrontmatterLine(line string) [2]string {
	s := bufio.NewScanner(strings.NewReader(line))
	s.Scan()
	text := s.Text()
	idx := strings.Index(text, ":")
	if idx < 0 {
		return [2]string{text, ""}
	}
	key := strings.TrimSpace(text[:idx])
	val := strings.TrimSpace(text[idx+1:])
	val = strings.Trim(val, `"'`)
	return [2]string{key, val}
}
