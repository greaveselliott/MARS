/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordDecisionUsesSessionRole(t *testing.T) {
	root, err := NewRoot(t.TempDir())
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly"})

	_, err = handleRecordDecision(ctx, root, json.RawMessage(`{"summary":"Choose scoped JIRA mirror","rationale":"Avoid global role state."}`))
	if err != nil {
		t.Fatalf("handleRecordDecision: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root.Abs(), ".harness", "learnings.yaml"))
	if err != nil {
		t.Fatalf("read learnings: %v", err)
	}
	if !strings.Contains(string(data), "role: cto-weekly") {
		t.Fatalf("decision did not record session role:\n%s", string(data))
	}
}

func TestRecordDecisionRejectsSymlinkedLearningsParent(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "learnings.yaml")
	if err := os.WriteFile(sentinel, []byte("outside\n"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, ".harness")); err != nil {
		t.Fatalf("create symlink parent: %v", err)
	}
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatalf("NewRoot: %v", err)
	}

	_, err = handleRecordDecision(context.Background(), root, json.RawMessage(`{"summary":"Remain contained"}`))
	if err == nil {
		t.Fatal("expected symlinked learnings parent to fail")
	}
	data, readErr := os.ReadFile(sentinel)
	if readErr != nil {
		t.Fatalf("read sentinel: %v", readErr)
	}
	if string(data) != "outside\n" {
		t.Fatalf("outside sentinel changed: %q", data)
	}
}
