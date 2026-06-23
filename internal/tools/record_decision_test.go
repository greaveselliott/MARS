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
