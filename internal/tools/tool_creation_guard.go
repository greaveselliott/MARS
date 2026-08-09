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
	"fmt"
	"path/filepath"
	"strings"
)

const toolCreationGuardSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "expect_exception": { "type": "boolean", "description": "Whether a bypass exception is expected" },
    "tool_name": { "type": "string", "description": "Optional built-in tool name to check" }
  },
  "required": []
}`

type toolCreationGuardArgs struct {
	ExpectException bool   `json:"expect_exception"`
	ToolName        string `json:"tool_name"`
}

func registerToolCreationGuard(r *Registry) error {
	return r.Register("tool_creation_guard", "Audit whether built-in tool creation followed the governed tool_create and record_decision path.", json.RawMessage(toolCreationGuardSchema), handleToolCreationGuard)
}

func handleToolCreationGuard(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args toolCreationGuardArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("tool_creation_guard: parse arguments: %w", err)
	}
	toolName := strings.TrimSpace(args.ToolName)
	var checks []string
	checks = append(checks,
		guardContains(root, "docs/design-docs/tools-glossary.md", "New built-in tools must originate through `tool_create`"),
		guardContains(root, "docs/design-docs/tools-glossary.md", "`record_decision`"),
		guardContains(root, "docs/design-docs/delivery-operating-model.md", "Built-in tool creation must dogfood the meta-tool path"),
		guardContains(root, "docs/design-docs/dogfood-and-decisions.md", "bypassing `tool_create` breaks the doctrine it represents"),
		guardContains(root, "internal/scanner/init.go", "Tool creation path"),
		guardContains(root, "internal/docsconsistency/operating_rules_test.go", "TestToolCreationPathIsDocumented"),
	)
	if toolName != "" {
		checks = append(checks,
			guardFileExists(root, filepath.Join("internal", "tools", toolName+".go")),
			guardFileExists(root, filepath.Join("internal", "tools", toolName+"_test.go")),
			guardContains(root, "docs/design-docs/tools-glossary.md", "`"+toolName+"`"),
			guardContains(root, "internal/scanner/init.go", toolName),
		)
	}
	if args.ExpectException {
		checks = append(checks,
			guardContains(root, "docs/design-docs/dogfood-and-decisions.md", "Bypassing `tool_create`"),
			guardContains(root, "docs/design-docs/tools-glossary.md", "record_decision"),
		)
	}
	status := "ok"
	for _, check := range checks {
		if strings.HasPrefix(check, "FAIL:") {
			status = "needs_attention"
			break
		}
	}
	return ToolResult{Output: "# tool_creation_guard\nstatus: " + status + "\n\n" + strings.Join(checks, "\n")}, nil
}

func guardContains(root Root, rel, needle string) string {
	content, err := guardRead(root, rel)
	if err != nil {
		return "FAIL: " + rel + " unreadable: " + err.Error()
	}
	if strings.Contains(content, needle) {
		return "PASS: " + rel + " contains " + needle
	}
	return "FAIL: " + rel + " missing " + needle
}

func guardFileExists(root Root, rel string) string {
	if _, err := root.RepoFS().Stat(rel); err != nil {
		return "FAIL: " + rel + " missing"
	}
	return "PASS: " + rel + " exists"
}

func guardRead(root Root, rel string) (string, error) {
	b, err := root.RepoFS().ReadFile(rel)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
