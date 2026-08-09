/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/features/F-007-guardrails-and-safety.md
*/
package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars/internal/repofs"
)

func TestCheck_withinLimits(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 3,
		LinesPerFile: map[string]int{"a.go": 50, "b.go": 30, "c.go": 10},
		TotalLines:   90,
		Deletions:    0,
	}
	if err := Check(stats, DefaultLimits()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheck_defaultAllowsManySmallFiles(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 100,
		LinesPerFile: map[string]int{},
		TotalLines:   100,
	}
	if err := Check(stats, DefaultLimits()); err != nil {
		t.Fatalf("expected default file-count cap to be disabled, got %v", err)
	}
}

func TestCheck_exceedsFileCountWhenConfigured(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 20,
		LinesPerFile: map[string]int{},
		TotalLines:   100,
	}
	limits := DefaultLimits()
	limits.MaxFilesPerJob = 10
	err := Check(stats, limits)
	if err == nil {
		t.Fatal("expected error for exceeding file count")
	}
	if !strings.Contains(err.Error(), "files changed") {
		t.Errorf("expected file count error, got %v", err)
	}
}

func TestCheck_exceedsLinesPerFile(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 1,
		LinesPerFile: map[string]int{"huge.go": 1000},
		TotalLines:   1000,
	}
	err := Check(stats, DefaultLimits())
	if err == nil {
		t.Fatal("expected error for exceeding per-file line count")
	}
	if !strings.Contains(err.Error(), "lines changed in huge.go") {
		t.Errorf("expected per-file error, got %v", err)
	}
}

func TestCheck_exceedsTotalLines(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 5,
		LinesPerFile: map[string]int{
			"a.go": 490, "b.go": 490, "c.go": 490,
			"d.go": 490, "e.go": 490,
		},
		TotalLines: 5000,
	}
	err := Check(stats, DefaultLimits())
	if err == nil {
		t.Fatal("expected error for exceeding total lines")
	}
	if !strings.Contains(err.Error(), "total lines") {
		t.Errorf("expected total lines error, got %v", err)
	}
}

func TestCheck_forbidDelete(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 1,
		LinesPerFile: map[string]int{"a.go": 10},
		TotalLines:   10,
		Deletions:    2,
	}
	limits := DefaultLimits()
	err := Check(stats, limits)
	if err == nil {
		t.Fatal("expected error for file deletions")
	}
	if !strings.Contains(err.Error(), "deletions") {
		t.Errorf("expected deletion error, got %v", err)
	}
}

func TestCheck_allowsDeleteWhenPermitted(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 1,
		LinesPerFile: map[string]int{"a.go": 10},
		TotalLines:   10,
		Deletions:    2,
	}
	limits := DefaultLimits()
	limits.ForbidDelete = false
	if err := Check(stats, limits); err != nil {
		t.Errorf("expected no error with ForbidDelete=false, got %v", err)
	}
}

func TestScanForSecrets_detectsAWSKeys(t *testing.T) {
	content := `aws_access_key_id = ` + "AKIA" + "IOSFODNN7EXAMPLE"
	results := ScanForSecrets("config.yaml", content)
	if len(results) == 0 {
		t.Fatal("expected to detect AWS key")
	}
	if results[0].Pattern != "AWS Access Key" {
		t.Errorf("expected AWS Access Key pattern, got %q", results[0].Pattern)
	}
}

func TestScanForSecrets_detectsGitHubTokens(t *testing.T) {
	tests := []string{
		"token: " + "ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh1234",
		"secret: " + "gho_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh1234",
		"pat: " + "ghs_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh1234",
	}
	for _, content := range tests {
		results := ScanForSecrets("env.sh", content)
		if len(results) == 0 {
			t.Errorf("expected to detect GitHub token in %q", content)
			continue
		}
		if !strings.Contains(results[0].Pattern, "GitHub Token") {
			t.Errorf("expected GitHub Token pattern, got %q", results[0].Pattern)
		}
	}
}

func TestScanForSecrets_detectsPrivateKeys(t *testing.T) {
	content := `-----BEGIN ` + `RSA PRIVATE KEY-----
MIIEowIBAAKCAQ...
-----END RSA PRIVATE KEY-----`
	results := ScanForSecrets("id_rsa", content)
	if len(results) == 0 {
		t.Fatal("expected to detect private key")
	}
	if !strings.Contains(results[0].Pattern, "Private Key") {
		t.Errorf("expected Private Key pattern, got %q", results[0].Pattern)
	}
}

func TestScanForSecrets_detectsPasswordInURL(t *testing.T) {
	content := `DATABASE_URL=postgres://admin:` + `s3cret` + `@db.example.com/mydb`
	results := ScanForSecrets(".env", content)
	if len(results) == 0 {
		t.Fatal("expected to detect password in URL")
	}
	found := false
	for _, r := range results {
		if r.Pattern == "Password in URL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Password in URL pattern among results")
	}
}

func TestScanForSecrets_detectsGenericAPIKey(t *testing.T) {
	content := `api_key = "` + `sk_` + `live_abcdefghijklmnop"`
	results := ScanForSecrets("config.toml", content)
	if len(results) == 0 {
		t.Fatal("expected to detect generic API key")
	}
	found := false
	for _, r := range results {
		if r.Pattern == "Generic API Key Assignment" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Generic API Key Assignment pattern among results")
	}
}

func TestScanForSecrets_cleanFileReturnsNone(t *testing.T) {
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`
	results := ScanForSecrets("main.go", content)
	if len(results) != 0 {
		t.Errorf("expected no secrets in clean file, got %d: %+v", len(results), results)
	}
}

func TestScanForSecrets_reportsCorrectLineNumber(t *testing.T) {
	content := "line1\nline2\n" + "AKIA" + "IOSFODNN7EXAMPLE" + "\nline4"
	results := ScanForSecrets("test.txt", content)
	if len(results) == 0 {
		t.Fatal("expected to detect AWS key")
	}
	if results[0].Line != 3 {
		t.Errorf("expected line 3, got %d", results[0].Line)
	}
}

func TestRepositorySecretScanStagedUsesIndexBlobs(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	root := openSecretScanRoot(t, repo)
	secret := testRepositorySecret()
	const path = "  staged.txt  "
	writeSecretScanFile(t, repo, path, "token = \""+secret+"\"\n")
	runSecretScanGit(t, repo, "add", path)
	writeSecretScanFile(t, repo, path, "clean worktree\n")

	findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
	if err != nil {
		t.Fatalf("scan staged index blob: %v", err)
	}
	assertRepositoryFinding(t, findings, path)
	assertRepositoryScanRedacted(t, findings, secret)

	if err := os.Remove(filepath.Join(repo, path)); err != nil {
		t.Fatalf("remove worktree entry: %v", err)
	}
	findings, err = ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
	if err != nil {
		t.Fatalf("scan index-only blob: %v", err)
	}
	assertRepositoryFinding(t, findings, path)
}

func TestRepositorySecretScanStagedIgnoresGitReplaceObjects(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	root := openSecretScanRoot(t, repo)
	secret := testRepositorySecret()
	writeSecretScanFile(t, repo, "staged.txt", "token = \""+secret+"\"\n")
	runSecretScanGit(t, repo, "add", "staged.txt")
	original := strings.TrimSpace(runSecretScanGitOutput(t, repo, nil, "rev-parse", ":staged.txt"))
	clean := strings.TrimSpace(runSecretScanGitOutput(t, repo, strings.NewReader("clean\n"), "hash-object", "-w", "--stdin"))
	runSecretScanGit(t, repo, "replace", original, clean)
	if replaced := runSecretScanGitOutput(t, repo, nil, "cat-file", "blob", original); replaced != "clean\n" {
		t.Fatalf("replace-object fixture was not active: %q", replaced)
	}

	findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
	if err != nil {
		t.Fatalf("scan staged blob with replacement ref: %v", err)
	}
	assertRepositoryFinding(t, findings, "staged.txt")
	assertRepositoryScanRedacted(t, findings, secret)
}

func TestRepositorySecretScanIncludesTrackedLocalCredentialFile(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	root := openSecretScanRoot(t, repo)
	secret := testRepositorySecret()
	writeSecretScanFile(t, repo, ".gitignore", ".harness/.env.local\n")
	runSecretScanGit(t, repo, "add", ".gitignore")
	runSecretScanGit(t, repo, "commit", "-m", "ignore local credentials")
	writeSecretScanFile(t, repo, localCredentialPath, "ACCESS_TOKEN="+secret+"\n")

	findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanFull)
	if err != nil {
		t.Fatalf("scan ignored untracked credential file: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("ignored untracked credential file should be omitted, got %+v", findings)
	}

	runSecretScanGit(t, repo, "add", "-f", localCredentialPath)
	findings, err = ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
	if err != nil {
		t.Fatalf("scan force-added credential file: %v", err)
	}
	assertRepositoryFinding(t, findings, localCredentialPath)
	runSecretScanGit(t, repo, "commit", "-m", "track credential fixture")
	writeSecretScanFile(t, repo, "note.txt", "clean\n")
	runSecretScanGit(t, repo, "add", "note.txt")
	findings, err = ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
	if err != nil {
		t.Fatalf("scan tracked credential file: %v", err)
	}
	assertRepositoryFinding(t, findings, localCredentialPath)
}

func TestRepositorySecretScanReconcilesRenameAndDeletion(t *testing.T) {
	t.Run("rename destination", func(t *testing.T) {
		repo := newSecretScanGitRepo(t)
		root := openSecretScanRoot(t, repo)
		secret := testRepositorySecret()
		writeSecretScanFile(t, repo, "old.txt", "token = \""+secret+"\"\n")
		runSecretScanGit(t, repo, "add", "old.txt")
		runSecretScanGit(t, repo, "commit", "-m", "add fixture")
		runSecretScanGit(t, repo, "mv", "old.txt", "new.txt")

		findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
		if err != nil {
			t.Fatalf("scan renamed blob: %v", err)
		}
		assertRepositoryFinding(t, findings, "new.txt")
		for _, finding := range findings {
			if finding.File == "old.txt" {
				t.Fatalf("rename source must not be scanned: %+v", findings)
			}
		}
	})

	t.Run("deletion is tombstone", func(t *testing.T) {
		repo := newSecretScanGitRepo(t)
		root := openSecretScanRoot(t, repo)
		secret := testRepositorySecret()
		writeSecretScanFile(t, repo, "deleted.txt", "token = \""+secret+"\"\n")
		runSecretScanGit(t, repo, "add", "deleted.txt")
		runSecretScanGit(t, repo, "commit", "-m", "add fixture")
		runSecretScanGit(t, repo, "rm", "deleted.txt")
		writeSecretScanFile(t, repo, "deleted.txt", "token = \""+secret+"\"\n")

		findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
		if err != nil {
			t.Fatalf("scan staged deletion: %v", err)
		}
		if len(findings) != 0 {
			t.Fatalf("staged deletion must not fall back to worktree bytes: %+v", findings)
		}
	})
}

func TestRepositorySecretScanFullIncludesDirtyAndUntrackedFiles(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	root := openSecretScanRoot(t, repo)
	secret := testRepositorySecret()
	writeSecretScanFile(t, repo, "README.md", "ACCESS_TOKEN="+secret+"\n")
	writeSecretScanFile(t, repo, "untracked.txt", "ACCESS_TOKEN="+secret+"\n")

	findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanFull)
	if err != nil {
		t.Fatalf("scan full repository view: %v", err)
	}
	assertRepositoryFinding(t, findings, "README.md")
	assertRepositoryFinding(t, findings, "untracked.txt")
}

func TestRepositorySecretScanRequiresWorktreeTopLevel(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	nested := filepath.Join(repo, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	root := openSecretScanRoot(t, nested)

	_, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanFull)
	if err == nil || err.Error() != "repository secret scan: Git worktree-root admission failed" {
		t.Fatalf("expected fixed worktree-root rejection, got %v", err)
	}
	if strings.Contains(err.Error(), repo) || strings.Contains(err.Error(), nested) {
		t.Fatalf("worktree-root error exposed a path: %v", err)
	}
}

func TestRepositorySecretScanFullReadsAssumeUnchangedWorktreeEntry(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	root := openSecretScanRoot(t, repo)
	secret := testRepositorySecret()
	runSecretScanGit(t, repo, "update-index", "--assume-unchanged", "README.md")
	writeSecretScanFile(t, repo, "README.md", "ACCESS_TOKEN="+secret+"\n")
	if output := runSecretScanGitOutput(t, repo, nil, "diff", "--name-only", "--"); output != "" {
		t.Fatalf("assume-unchanged fixture remained visible to ordinary diff: %q", output)
	}

	findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanFull)
	if err != nil {
		t.Fatalf("scan assume-unchanged worktree entry: %v", err)
	}
	assertRepositoryFinding(t, findings, "README.md")
	assertRepositoryScanRedacted(t, findings, secret)
}

func TestRepositorySecretScanFailsClosedWithoutLeakingCandidates(t *testing.T) {
	repo := newSecretScanGitRepo(t)
	root := openSecretScanRoot(t, repo)
	secret := testRepositorySecret()
	if err := os.Symlink(secret, filepath.Join(repo, "credential-link")); err != nil {
		t.Fatalf("create credential symlink fixture: %v", err)
	}
	runSecretScanGit(t, repo, "add", "credential-link")

	findings, err := ScanRepositoryForSecrets(context.Background(), root, RepositorySecretScanStaged)
	if err == nil {
		t.Fatal("expected unsupported staged symlink to fail closed")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error exposed candidate value: %v", err)
	}
	assertRepositoryScanRedacted(t, findings, secret)
}

func newSecretScanGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}
	repo := t.TempDir()
	runSecretScanGit(t, repo, "init", "-q")
	runSecretScanGit(t, repo, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, repo, "config", "user.name", "MARS Tests")
	runSecretScanGit(t, repo, "config", "commit.gpgsign", "false")
	writeSecretScanFile(t, repo, "README.md", "clean\n")
	runSecretScanGit(t, repo, "add", "README.md")
	runSecretScanGit(t, repo, "commit", "-m", "initial")
	return repo
}

func openSecretScanRoot(t *testing.T, repo string) *repofs.Root {
	t.Helper()
	root, err := repofs.Open(repo)
	if err != nil {
		t.Fatalf("open repository root: %v", err)
	}
	return root
}

func runSecretScanGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	_ = runSecretScanGitOutput(t, repo, nil, args...)
}

func runSecretScanGitOutput(t *testing.T, repo string, input *strings.Reader, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if input != nil {
		command.Stdin = input
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git fixture command failed: %v: %s", err, output)
	}
	return string(output)
}

func writeSecretScanFile(t *testing.T, repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func assertRepositoryFinding(t *testing.T, findings []RepositorySecretFinding, path string) {
	t.Helper()
	for _, finding := range findings {
		if finding.File == path {
			return
		}
	}
	t.Fatalf("expected finding for %s, got %+v", path, findings)
}

func assertRepositoryScanRedacted(t *testing.T, findings []RepositorySecretFinding, candidate string) {
	t.Helper()
	text := fmt.Sprintf("%+v", findings)
	encoded, err := json.Marshal(findings)
	if err != nil {
		t.Fatalf("marshal findings: %v", err)
	}
	if strings.Contains(text, candidate) || strings.Contains(string(encoded), candidate) {
		t.Fatal("repository findings exposed candidate value")
	}
}

func testRepositorySecret() string {
	return "ghp_" + strings.Repeat("z", 36)
}

func TestEmergencyStop_executesAll(t *testing.T) {
	es := NewEmergencyStop()
	var calls []int

	es.Register(func(_ context.Context) error {
		calls = append(calls, 1)
		return nil
	})
	es.Register(func(_ context.Context) error {
		calls = append(calls, 2)
		return nil
	})
	es.Register(func(_ context.Context) error {
		calls = append(calls, 3)
		return nil
	})

	errs := es.Execute(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if len(calls) != 3 {
		t.Errorf("expected 3 handlers called, got %d", len(calls))
	}
}

func TestEmergencyStop_collectsErrors(t *testing.T) {
	es := NewEmergencyStop()

	es.Register(func(_ context.Context) error {
		return nil
	})
	es.Register(func(_ context.Context) error {
		return fmt.Errorf("handler 1 failed")
	})
	es.Register(func(_ context.Context) error {
		return fmt.Errorf("handler 2 failed")
	})

	errs := es.Execute(context.Background())
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "handler 1 failed") {
		t.Errorf("expected first error message, got %v", errs[0])
	}
	if !strings.Contains(errs[1].Error(), "handler 2 failed") {
		t.Errorf("expected second error message, got %v", errs[1])
	}
}

func TestEmergencyStop_emptyExecute(t *testing.T) {
	es := NewEmergencyStop()
	errs := es.Execute(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no errors from empty stop, got %v", errs)
	}
}

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxFilesPerJob != 0 {
		t.Errorf("expected MaxFilesPerJob=0, got %d", limits.MaxFilesPerJob)
	}
	if limits.MaxLinesPerFile != 500 {
		t.Errorf("expected MaxLinesPerFile=500, got %d", limits.MaxLinesPerFile)
	}
	if limits.MaxTotalLines != 2000 {
		t.Errorf("expected MaxTotalLines=2000, got %d", limits.MaxTotalLines)
	}
	if limits.MaxOpenPRsPerRepo != 3 {
		t.Errorf("expected MaxOpenPRsPerRepo=3, got %d", limits.MaxOpenPRsPerRepo)
	}
	if !limits.ForbidDelete {
		t.Error("expected ForbidDelete=true by default")
	}
}
