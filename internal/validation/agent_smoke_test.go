/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/validation-matrix-gating.md
- docs/validation/README.md
- docs/product-specs/product-surface.md
- docs/features/F-012-self-improvement-loop.md
*/
package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestLoadAgentSmokeMatrixCoversDefaultRoles(t *testing.T) {
	matrix, err := LoadAgentSmokeMatrix(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load matrix: %v", err)
	}
	roles := map[string]bool{}
	for _, c := range matrix.Cases {
		roles[c.Role] = true
	}
	for _, role := range []string{
		"ceo",
		"head-of-strategy",
		"coo",
		"cto-weekly",
		"engineer",
		"qa",
		"security",
		"dependency-manager",
		"release-manager",
		"dogfood",
		"pipeline-fixer",
		"orchestrator",
		"janitor",
		"foundation-maintainer",
	} {
		if !roles[role] {
			t.Fatalf("matrix missing role %s", role)
		}
	}
}

func TestSelectAgentSmokeFastRotatesOneCasePerRole(t *testing.T) {
	matrix, err := LoadAgentSmokeMatrix(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load matrix: %v", err)
	}
	selected, err := SelectAgentSmokeCases(matrix, AgentSmokeOptions{Suite: AgentSmokeSuiteFast, Cycle: "0"})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	seen := map[string]bool{}
	for _, c := range selected {
		if seen[c.Role] {
			t.Fatalf("fast suite selected role %s more than once", c.Role)
		}
		seen[c.Role] = true
	}
	if len(seen) < 10 {
		t.Fatalf("expected broad role coverage, got %d roles", len(seen))
	}
}

func TestRunAgentSmokeGeneratesAndDiscardsEphemeralTarget(t *testing.T) {
	root := t.TempDir()
	report, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot: repoRootForTest(t),
		Root:        root,
		Suite:       AgentSmokeSuiteFast,
		Role:        "ceo",
		ProjectType: "static-web",
	})
	if err != nil {
		t.Fatalf("run smoke: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected report ok: %+v", report)
	}
	if report.Selected != 1 || report.Passed != 1 {
		t.Fatalf("expected one passing case, got selected=%d passed=%d failed=%d", report.Selected, report.Passed, report.Failed)
	}
	if !report.Results[0].Discarded {
		t.Fatalf("successful run should be discarded by default: %+v", report.Results[0])
	}
	if _, err := os.Stat(report.Results[0].RunPath); !os.IsNotExist(err) {
		t.Fatalf("expected run path to be removed, stat err=%v", err)
	}
}

func TestRunAgentSmokeKeepRunsAndCleanupOnly(t *testing.T) {
	root := t.TempDir()
	report, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot: repoRootForTest(t),
		Root:        root,
		Suite:       AgentSmokeSuiteDefault,
		Role:        "engineer",
		ProjectType: "go-api",
		KeepRuns:    true,
	})
	if err != nil {
		t.Fatalf("run smoke: %v", err)
	}
	if !report.OK() {
		t.Fatalf("expected report ok: %+v", report)
	}
	runPath := report.Results[0].RunPath
	if _, err := os.Stat(filepath.Join(runPath, "target", ".harness", "manifest.yaml")); err != nil {
		t.Fatalf("expected retained harness manifest: %v", err)
	}
	cleanup, err := RunAgentSmoke(context.Background(), AgentSmokeOptions{
		HarnessRoot: repoRootForTest(t),
		Root:        root,
		CleanupOnly: true,
	})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if cleanup.Cleaned != 1 {
		t.Fatalf("expected one cleaned run, got %d", cleanup.Cleaned)
	}
	if _, err := os.Stat(runPath); !os.IsNotExist(err) {
		t.Fatalf("expected cleanup to remove run path, stat err=%v", err)
	}
}
