/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
*/
package codeintel

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestIndexSearchSnippetTraceAndImpact(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "go.mod", "module example.test/app\n\ngo 1.22\n")
	writeFile(t, repo, "internal/app/app.go", `package app

import "fmt"

type Runner struct{}

func (r *Runner) Run(name string) string {
	return helper(name)
}

func helper(name string) string {
	return fmt.Sprintf("hi %s", name)
}
`)
	writeFile(t, repo, "internal/app/app_test.go", `package app

import "testing"

func TestRunner(t *testing.T) {}
`)
	writeFile(t, repo, "docs/design-docs/tools-glossary.md", "The Runner Run behavior is documented for internal/app/app.go.\n")
	writeFile(t, repo, "docs/features/F-001-demo.md", "Scenario mentions internal/app/app.go and Runner.\n")
	writeFile(t, repo, "docs/tickets/backlog/T-001-demo.md", "Ticket covers helper in internal/app/app.go.\n")

	store, err := Open(repo, filepath.Join(t.TempDir(), "mars.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	idx, err := store.Index(context.Background(), IndexOptions{})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if idx.FilesIndexed == 0 || idx.Symbols == 0 {
		t.Fatalf("expected indexed files and symbols, got %+v", idx)
	}

	search, err := store.Search(context.Background(), SearchOptions{Query: "Runner Run", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(search.Results) == 0 {
		t.Fatalf("expected search results")
	}

	snippet, err := store.Snippet(context.Background(), "app.Runner.Run")
	if err != nil {
		t.Fatalf("Snippet: %v", err)
	}
	if snippet.Symbol.Path != "internal/app/app.go" || snippet.Source == "" {
		t.Fatalf("unexpected snippet: %+v", snippet)
	}

	trace, err := store.Trace(context.Background(), "app.Runner.Run", "outbound")
	if err != nil {
		t.Fatalf("Trace: %v", err)
	}
	if len(trace.Edges) == 0 {
		t.Fatalf("expected outbound trace edges")
	}

	impact, err := store.Impact(context.Background(), []string{"internal/app/app.go"}, "")
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}
	if len(impact.Symbols) == 0 || len(impact.Tests) == 0 || len(impact.Docs) == 0 || len(impact.Features) == 0 || len(impact.Tickets) == 0 {
		t.Fatalf("expected Mars-native impact links, got %+v", impact)
	}
}

func TestStatusDetectsStaleFiles(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "main.go", "package main\n\nfunc main() {}\n")
	store, err := Open(repo, filepath.Join(t.TempDir(), "mars.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if _, err := store.Index(context.Background(), IndexOptions{}); err != nil {
		t.Fatalf("Index: %v", err)
	}
	writeFile(t, repo, "main.go", "package main\n\nfunc main() { println(\"changed\") }\n")
	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Status != FreshnessStale {
		t.Fatalf("expected stale status, got %+v", status)
	}
}

func TestStatusDetectsNewFiles(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "main.go", "package main\n\nfunc main() {}\n")
	store, err := Open(repo, filepath.Join(t.TempDir(), "mars.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if _, err := store.Index(context.Background(), IndexOptions{}); err != nil {
		t.Fatalf("Index: %v", err)
	}
	writeFile(t, repo, "internal/newpkg/new.go", "package newpkg\n\nfunc NewThing() {}\n")
	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Status != FreshnessStale || status.NewFiles == 0 {
		t.Fatalf("expected stale status with new files, got %+v", status)
	}
}

func TestBuildContextRefreshesAndRendersBoundedGraphMap(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "go.mod", "module example.test/app\n\ngo 1.22\n")
	writeFile(t, repo, "internal/app/app.go", `package app

func Run() string {
	return "ok"
}
`)
	writeFile(t, repo, "internal/app/app_test.go", "package app\n\nfunc TestRun() {}\n")
	writeFile(t, repo, "docs/features/F-001-demo.md", "Scenario covers internal/app/app.go and Run.\n")
	initGitRepo(t, repo)

	result, err := BuildContext(context.Background(), repo, ContextOptions{
		Refresh:  true,
		DBPath:   filepath.Join(t.TempDir(), "mars.db"),
		MaxFiles: 100,
	})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if !result.Refreshed {
		t.Fatalf("expected context build to refresh missing index")
	}
	for _, want := range []string{
		"freshness: fresh",
		"index_refresh:",
		"changed_paths:",
		"internal/app/app.go",
		"impacted_symbols:",
		"app.Run",
		"likely_tests:",
		"docs:",
		"operator_hint: use code_search",
	} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("expected context text to contain %q:\n%s", want, result.Text)
		}
	}
}

func TestBuildContextUsesConfiguredDBPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	writeFile(t, repo, "main.go", "package main\n\nfunc main() {}\n")
	customDB := filepath.Join(t.TempDir(), "custom", "mars.db")

	result, err := BuildContext(context.Background(), repo, ContextOptions{Refresh: true, DBPath: customDB, MaxFiles: 100})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if !result.Refreshed {
		t.Fatalf("expected context build to refresh missing index")
	}
	if count := codeIntelTableCount(t, customDB); count == 0 {
		t.Fatalf("expected codeintel tables in configured DB")
	}
	if _, err := os.Stat(DefaultDBPath(repo)); !os.IsNotExist(err) {
		t.Fatalf("expected default DB path not to be used, stat err=%v", err)
	}
}

func TestBuildContextDoesNotAutoRefreshLargeStaleSet(t *testing.T) {
	repo := t.TempDir()
	staleDB := filepath.Join(t.TempDir(), "mars.db")
	store, err := Open(repo, staleDB)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	for i := 0; i < 260; i++ {
		writeFile(t, repo, filepath.Join("pkg", "p", "file"+string(rune('a'+(i%26)))+fmtInt(i)+".go"), "package p\n\nfunc F"+fmtInt(i)+"() {}\n")
	}
	if _, err := store.Index(context.Background(), IndexOptions{MaxFiles: 1000}); err != nil {
		t.Fatalf("Index: %v", err)
	}
	for i := 0; i < 260; i++ {
		writeFile(t, repo, filepath.Join("pkg", "p", "file"+string(rune('a'+(i%26)))+fmtInt(i)+".go"), "package p\n\nfunc F"+fmtInt(i)+"() string { return \"changed\" }\n")
	}
	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.StaleFiles != 260 {
		t.Fatalf("expected exact stale count, got %+v", status)
	}

	result, err := BuildContext(context.Background(), repo, ContextOptions{
		Refresh:               true,
		DBPath:                staleDB,
		MaxFiles:              1000,
		MaxAutoRefreshChanges: 250,
	})
	if err != nil {
		t.Fatalf("BuildContext stale DB: %v", err)
	}
	if result.Refreshed {
		t.Fatalf("expected stale configured DB not to auto-refresh")
	}
	if !strings.Contains(result.Text, "stale_files=260") || !strings.Contains(result.Text, "omitted: graph relationships are stale") {
		t.Fatalf("expected exact stale disclosure and omitted relationships, got:\n%s", result.Text)
	}
}

func TestImpactNoChangesReturnsEmptyImpact(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, repo, "docs/design-docs/mentions.md", "main.go is documented here.\n")
	initGitRepo(t, repo)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "-c", "user.name=Mars", "-c", "user.email=mars@example.test", "commit", "-m", "initial")
	store, err := Open(repo, filepath.Join(t.TempDir(), "mars.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if _, err := store.Index(context.Background(), IndexOptions{}); err != nil {
		t.Fatalf("Index: %v", err)
	}

	impact, err := store.Impact(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}
	if len(impact.ChangedPaths) != 0 || len(impact.Symbols) != 0 || len(impact.Docs) != 0 || len(impact.Features) != 0 || len(impact.Tickets) != 0 {
		t.Fatalf("expected empty impact for clean repo, got %+v", impact)
	}
}

func TestBuildContextPrioritizesSourcePathsOverGeneratedHarnessFiles(t *testing.T) {
	text := renderContext(Status{Status: FreshnessFresh}, IndexResult{}, ImpactResult{
		ChangedPaths: []string{
			".harness/manifest.yaml",
			"docs/features/F-001-demo.md",
			"internal/app/app.go",
		},
		Symbols: []Symbol{
			{QualifiedName: ".harness/manifest.yaml", Path: ".harness/manifest.yaml", StartLine: 1, EndLine: 10},
			{QualifiedName: "app.Run", Path: "internal/app/app.go", StartLine: 3, EndLine: 5},
		},
	}, false, ContextOptions{MaxChangedPaths: 2, MaxSymbols: 1})

	if !strings.Contains(text, "- internal/app/app.go\n") {
		t.Fatalf("expected source path to be retained:\n%s", text)
	}
	if !strings.Contains(text, "- app.Run internal/app/app.go:3-5") {
		t.Fatalf("expected source symbol to be retained:\n%s", text)
	}
}

func TestImpactPreflightArgsBoundsLargeDirtySets(t *testing.T) {
	graph := ContextResult{Impact: ImpactResult{ChangedPaths: []string{"main.go", "main_test.go"}}}
	args, ok := ImpactPreflightArgs(graph, 2)
	if !ok {
		t.Fatalf("expected bounded args")
	}
	if !strings.Contains(args, `"paths":["main.go","main_test.go"]`) {
		t.Fatalf("expected explicit paths, got %s", args)
	}
	if args, ok := ImpactPreflightArgs(graph, 1); ok || args != "" {
		t.Fatalf("expected large path set to skip preflight, got ok=%v args=%q", ok, args)
	}
	if args, ok := ImpactPreflightArgs(ContextResult{}, 1); !ok || args != `{}` {
		t.Fatalf("expected clean repo args, got ok=%v args=%q", ok, args)
	}
}

func codeIntelTableCount(t *testing.T, dbPath string) int {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name LIKE 'codeintel_%'`).Scan(&count); err != nil {
		t.Fatalf("table count: %v", err)
	}
	return count
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v\n%s", err, out)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
