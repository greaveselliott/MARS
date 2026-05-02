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
	{label: "skip github flag", re: regexp.MustCompile(`--skip-github`)},
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
	for lineNo, line := range strings.Split(string(data), "\n") {
		if isHistoricalCompatibilityNote(line) {
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
