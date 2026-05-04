/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/harness-operating-model.md
- docs/features/F-001-delivery-operating-model.md
*/
package operatingmodel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckRepo_reportsMissingOperatingModelArtifacts(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()

	report, err := CheckRepo(repo)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}
	if report.OK() {
		t.Fatal("expected missing operating-model artifacts")
	}
	if len(report.Missing) == 0 {
		t.Fatalf("expected missing artifacts, got %+v", report)
	}
	if report.Summary() == "operating model is current" {
		t.Fatalf("expected drift summary, got %q", report.Summary())
	}
}

func TestCheckRepo_reportsStaleOperatingModelArtifact(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	for _, artifact := range requiredArtifacts {
		path := filepath.Join(repo, filepath.FromSlash(artifact.path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		content := stringsForNeedles(artifact.needles)
		if artifact.path == "docs/tickets/README.md" {
			content = "# Old Tickets\n"
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	report, err := CheckRepo(repo)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}
	if report.OK() {
		t.Fatal("expected stale operating-model artifact")
	}
	found := false
	for _, drift := range report.Stale {
		if drift.Path == "docs/tickets/README.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stale ticket README drift, got %+v", report.Stale)
	}
}

func TestCheckRepo_foundationRepoSkipsGeneratedHarnessContextGlossary(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	writeTestFile(t, repo, "go.mod", "module github.com/greaveselliott/mars-harness\n")
	writeTestFile(t, repo, "internal/scanner/init.go", "package scanner\n")
	for _, artifact := range requiredArtifacts {
		if artifact.path == ".harness/knowledge/context-glossary.yaml" {
			continue
		}
		writeTestFile(t, repo, artifact.path, stringsForNeedles(artifact.needles))
	}

	report, err := CheckRepo(repo)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected foundation repo to be current, got missing=%+v stale=%+v", report.Missing, report.Stale)
	}
}

func stringsForNeedles(needles []string) string {
	out := "# artifact\n"
	for _, needle := range needles {
		out += needle + "\n"
	}
	return out
}

func writeTestFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
