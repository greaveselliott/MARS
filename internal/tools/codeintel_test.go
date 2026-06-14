/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeIntelToolsRegistered(t *testing.T) {
	reg, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	for _, name := range []string{"code_index", "code_search", "code_snippet", "code_trace", "code_impact"} {
		if _, _, ok := reg.Lookup(name); !ok {
			t.Fatalf("expected %s to be registered", name)
		}
	}
}

func TestCodeIntelToolsRunThroughExecutor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	writeCodeIntelToolRepoFile(t, repo, "main.go", "package main\n\nfunc main() { helper() }\n\nfunc helper() {}\n")
	root, err := NewRoot(repo)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	customDB := filepath.Join(t.TempDir(), "custom", "mars.db")
	root = root.WithDBPath(customDB)
	reg, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	exec := NewExecutor(reg)
	allow := []string{"code_index", "code_search", "code_snippet", "code_trace", "code_impact"}
	if _, err := exec.Execute(context.Background(), root, allow, "code_index", `{}`); err != nil {
		t.Fatalf("code_index: %v", err)
	}
	if _, err := os.Stat(customDB); err != nil {
		t.Fatalf("expected custom DB to be used: %v", err)
	}
	defaultDB := filepath.Join(home, ".mars-harness", "db", filepath.Base(repo), "mars.db")
	if _, err := os.Stat(defaultDB); !os.IsNotExist(err) {
		t.Fatalf("expected default DB not to be used, stat err=%v", err)
	}
	search, err := exec.Execute(context.Background(), root, allow, "code_search", `{"query":"helper"}`)
	if err != nil {
		t.Fatalf("code_search: %v", err)
	}
	if !strings.Contains(search.Output, "helper") {
		t.Fatalf("expected helper in search output, got %s", search.Output)
	}
	impact, err := exec.Execute(context.Background(), root, allow, "code_impact", `{"paths":["main.go"]}`)
	if err != nil {
		t.Fatalf("code_impact: %v", err)
	}
	if !strings.Contains(impact.Output, "main.go") {
		t.Fatalf("expected main.go in impact output, got %s", impact.Output)
	}
}

func writeCodeIntelToolRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
