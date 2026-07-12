/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/source-quality-gates.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-017-open-source-publication.md
*/
package docsconsistency

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCheckCoverage(t *testing.T, input, floors string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	root := repoRoot(t)
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "coverage.txt")
	floorsPath := filepath.Join(dir, "floors.txt")
	if err := os.WriteFile(inputPath, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(floorsPath, []byte(floors), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", filepath.Join(root, "scripts", "check-coverage.sh"),
		"--input", inputPath, "--floors", floorsPath)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

const coverageFixture = `ok  	example.com/mod/internal/alpha	1.0s	coverage: 81.3% of statements
ok  	example.com/mod/internal/beta	(cached)	coverage: 43.7% of statements
?   	example.com/mod/internal/notests	[no test files]
ok  	example.com/mod/internal/nostatements	1.4s	coverage: [no statements]
	example.com/mod/internal/zero		coverage: 0.0% of statements
`

func TestCheckCoveragePassesWhenFloorsMet(t *testing.T) {
	floors := `# comment
example.com/mod/internal/alpha 70
example.com/mod/internal/beta 43
example.com/mod/internal/notests notest
example.com/mod/internal/nostatements nostmt
example.com/mod/internal/zero 0
`
	out, err := runCheckCoverage(t, coverageFixture, floors)
	if err != nil {
		t.Fatalf("expected pass, got error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "all packages meet their ratchet floors") {
		t.Fatalf("missing success message:\n%s", out)
	}
}

func TestCheckCoverageFailsOnRegressionBelowFloor(t *testing.T) {
	floors := `example.com/mod/internal/alpha 70
example.com/mod/internal/beta 45
example.com/mod/internal/notests notest
example.com/mod/internal/nostatements nostmt
example.com/mod/internal/zero 0
`
	out, err := runCheckCoverage(t, coverageFixture, floors)
	if err == nil {
		t.Fatalf("expected failure for beta below floor:\n%s", out)
	}
	if !strings.Contains(out, "internal/beta coverage 43.7% is below its ratchet floor 45%") {
		t.Fatalf("missing regression failure message:\n%s", out)
	}
}

func TestCheckCoverageFailsOnMissingFloorEntry(t *testing.T) {
	floors := `example.com/mod/internal/alpha 70
example.com/mod/internal/notests notest
example.com/mod/internal/nostatements nostmt
example.com/mod/internal/zero 0
`
	out, err := runCheckCoverage(t, coverageFixture, floors)
	if err == nil {
		t.Fatalf("expected failure for missing beta floor:\n%s", out)
	}
	if !strings.Contains(out, "internal/beta has no floor entry") {
		t.Fatalf("missing no-floor failure message:\n%s", out)
	}
}

func TestCheckCoverageFailsOnStaleFloorEntry(t *testing.T) {
	floors := `example.com/mod/internal/alpha 70
example.com/mod/internal/beta 43
example.com/mod/internal/notests notest
example.com/mod/internal/nostatements nostmt
example.com/mod/internal/zero 0
example.com/mod/internal/removed 50
`
	out, err := runCheckCoverage(t, coverageFixture, floors)
	if err == nil {
		t.Fatalf("expected failure for stale floor entry:\n%s", out)
	}
	if !strings.Contains(out, "internal/removed matches no package") {
		t.Fatalf("missing stale-entry failure message:\n%s", out)
	}
}

func TestCheckCoverageFailsWhenTestsDeleted(t *testing.T) {
	floors := `example.com/mod/internal/alpha 70
example.com/mod/internal/beta 43
example.com/mod/internal/notests 30
example.com/mod/internal/nostatements nostmt
example.com/mod/internal/zero 0
`
	out, err := runCheckCoverage(t, coverageFixture, floors)
	if err == nil {
		t.Fatalf("expected failure when a floored package has no test files:\n%s", out)
	}
	if !strings.Contains(out, "reports [no test files] but its floor is 30") {
		t.Fatalf("missing deleted-tests failure message:\n%s", out)
	}
}

func TestMakeVulnFailsClosedWhenScannerIsMissing(t *testing.T) {
	root := repoRoot(t)
	missing := filepath.Join(t.TempDir(), "missing-govulncheck")
	cmd := exec.Command("make", "vuln", "GOVULNCHECK="+missing)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing scanner to fail closed:\n%s", out)
	}
	text := string(out)
	if !strings.Contains(text, "vulnerability scanning is required") {
		t.Fatalf("missing fail-closed explanation:\n%s", text)
	}
	if !strings.Contains(text, "go install golang.org/x/vuln/cmd/govulncheck@v1.6.0") {
		t.Fatalf("missing pinned remediation command:\n%s", text)
	}
}

func TestMakeVulnPropagatesScannerFailure(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	scanner := filepath.Join(dir, "govulncheck")
	const script = "#!/bin/sh\necho scanner-database-failure >&2\nexit 23\n"
	if err := os.WriteFile(scanner, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "vuln", "GOVULNCHECK="+scanner)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected scanner failure to propagate:\n%s", out)
	}
	if !strings.Contains(string(out), "scanner-database-failure") {
		t.Fatalf("scanner failure output was not preserved:\n%s", out)
	}
}

func TestMakeVulnUsesConfiguredGoBinByDefault(t *testing.T) {
	root := repoRoot(t)
	goBin := t.TempDir()
	scanner := filepath.Join(goBin, "govulncheck")
	const script = "#!/bin/sh\necho configured-gobin-scanner\n"
	if err := os.WriteFile(scanner, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "vuln", "GOBIN="+goBin)
	cmd.Dir = root
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GOVULNCHECK=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected configured GOBIN scanner to run: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "configured-gobin-scanner") {
		t.Fatalf("make vuln did not select GOBIN/govulncheck:\n%s", out)
	}
}

// TestCoverageFloorsFileTracksRealPackages keeps scripts/coverage-floors.txt
// honest against the live module: every floor entry must name a package that
// exists, and every package in the module must have a floor entry.
func TestCoverageFloorsFileTracksRealPackages(t *testing.T) {
	root := repoRoot(t)
	floorsPath := filepath.Join(root, "scripts", "coverage-floors.txt")
	data, err := os.ReadFile(floorsPath)
	if err != nil {
		t.Fatalf("read floors file: %v", err)
	}
	floors := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Errorf("malformed floors line: %q", line)
			continue
		}
		floors[fields[0]] = true
	}

	cmd := exec.Command("go", "list", "./...")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list ./...: %v", err)
	}
	live := map[string]bool{}
	for _, pkg := range strings.Fields(string(out)) {
		live[pkg] = true
		if !floors[pkg] {
			t.Errorf("package %s has no floor entry in scripts/coverage-floors.txt; seed one from its current coverage", pkg)
		}
	}
	for pkg := range floors {
		if !live[pkg] {
			t.Errorf("floors entry %s matches no live package; remove the stale line", pkg)
		}
	}
}
