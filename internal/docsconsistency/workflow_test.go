/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-013-board-driven-integrations.md
*/
package docsconsistency

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

type forbiddenWorkflowPhrase struct {
	label string
	re    *regexp.Regexp
}

var forbiddenWorkflowPhrases = []forbiddenWorkflowPhrase{
	{label: "pull_request trigger", re: regexp.MustCompile(`(?i)\bpull_request\b`)},
	{label: "pull request workflow", re: regexp.MustCompile(`(?i)\bpull requests?\b`)},
	{label: "PR workflow", re: regexp.MustCompile(`\bPRs?\b`)},
	{label: "feature branch", re: regexp.MustCompile(`(?i)\bfeature branches?\b`)},
	{label: "open a PR", re: regexp.MustCompile(`(?i)\bopen(?:s|ed)? a PR\b`)},
	{label: "create PR", re: regexp.MustCompile(`(?i)\bcreate(?:s|d)? PR\b`)},
	{label: "merge PR", re: regexp.MustCompile(`(?i)\bmerge(?:s|d)? PR\b`)},
	{label: "auto-merge", re: regexp.MustCompile(`(?i)\bauto-merge\b`)},
	{label: "github PR tool", re: regexp.MustCompile(`(?i)\bgithub_pr\b`)},
	{label: "github check tool stub", re: regexp.MustCompile(`(?i)\bgithub_check\b`)},
	{label: "do not push main", re: regexp.MustCompile(`(?i)\bdo not push (?:to )?main\b`)},
	{label: "no direct push", re: regexp.MustCompile(`(?i)\bno direct push\b`)},
	{label: "default branch", re: regexp.MustCompile(`(?i)\bdefault branch\b`)},
	{label: "release branch", re: regexp.MustCompile(`(?i)\brelease branches?\b`)},
}

func TestStrictTrunkWorkflowDocs(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range workflowSurfaces() {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if info.IsDir() {
			err = filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if !isWorkflowFile(path) {
					return nil
				}
				checkStrictTrunkFile(t, root, path)
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", rel, err)
			}
			continue
		}
		checkStrictTrunkFile(t, root, path)
	}
}

func TestReadmePointsAtCurrentOperatingPlan(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "docs/exec-plans/active/current-operating-plan.md") {
		t.Fatalf("README.md must link to the current operating plan")
	}
	if strings.Contains(text, "docs/exec-plans/active/delivery-schedule.md") {
		t.Fatalf("README.md must not link to the removed active delivery schedule")
	}
}

func TestDogfoodMatrixNamesRequiredEvidence(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "design-docs", "dogfood-matrix.md"))
	if err != nil {
		t.Fatalf("read dogfood matrix: %v", err)
	}
	text := string(data)
	for _, needle := range []string{
		"mars start --repo <temp repo>",
		"fake-LLM dogfood path",
		"`../mars` is the supersession target",
		"Optional GitHub paths are skipped honestly",
		"docs/validation/profiles/mars-observer.md",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("dogfood matrix missing %q", needle)
		}
	}
}

func TestMarsObserverProfileStaysReadOnly(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "validation", "profiles", "mars-observer.md"))
	if err != nil {
		t.Fatalf("read Mars observer profile: %v", err)
	}
	text := string(data)
	for _, needle := range []string{
		"`../mars`",
		"Trust level stays `observer`",
		"must not call `file_write`",
		"No target harness files are overwritten",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("Mars observer profile missing %q", needle)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func workflowSurfaces() []string {
	return []string{
		"AGENTS.md",
		"README.md",
		"CONTRIBUTING.md",
		"ARCHITECTURE.md",
		".cursor/rules",
		"docs/tickets/README.md",
		"docs/product-specs",
		"docs/design-docs",
		"docs/roles",
		"docs/exec-plans/README.md",
		"docs/exec-plans/active/current-operating-plan.md",
		"docs/exec-plans/superseded/delivery-schedule.md",
		"docs/exec-plans/superseded/master-execution-plan.md",
		"internal/scanner/init.go",
		"examples/roles",
	}
}

func isWorkflowFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".md", ".mdc", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func checkStrictTrunkFile(t *testing.T, root, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("rel %s: %v", path, err)
	}
	text := string(data)
	for lineNo, line := range strings.Split(text, "\n") {
		if isHistoricalCompatibilityNote(line) || isOptionalBoardDrivenWorkflowNote(rel, text) {
			continue
		}
		for _, phrase := range forbiddenWorkflowPhrases {
			if phrase.re.MatchString(line) {
				t.Fatalf("%s:%d contains %s phrase %q; strict trunk defaults must say commit and push main directly",
					rel, lineNo+1, phrase.label, strings.TrimSpace(line))
			}
		}
	}
}

func isHistoricalCompatibilityNote(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "historical/compatibility") ||
		strings.Contains(lower, "legacy compatibility") ||
		strings.Contains(lower, "external research")
}

func isOptionalBoardDrivenWorkflowNote(rel, text string) bool {
	lowerText := strings.ToLower(text)
	if !strings.Contains(lowerText, "flow_profile") ||
		!strings.Contains(lowerText, "board-driven") ||
		!strings.Contains(lowerText, "ceo-led") ||
		!strings.Contains(lowerText, "strict-trunk") {
		return false
	}
	switch rel {
	case "docs/design-docs/board-driven-integrations.md",
		"docs/features/F-013-board-driven-integrations.md",
		"docs/exec-plans/active/current-operating-plan.md",
		"internal/scanner/init.go":
		return true
	default:
		return false
	}
}
