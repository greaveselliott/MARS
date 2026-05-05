/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/release-versioning.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/greaveselliott/mars-harness/internal/githubauth"
)

const githubAuthCheckSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {},
  "required": []
}`

type githubAuthCheckArgs struct {
}

func registerGithubAuthCheck(r *Registry) error {
	return r.Register("github_auth_check", "Check private GitHub release authentication readiness without revealing tokens.", json.RawMessage(githubAuthCheckSchema), handleGithubAuthCheck)
}

func handleGithubAuthCheck(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args githubAuthCheckArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("github_auth_check: parse arguments: %w", err)
	}
	_ = root
	_ = args
	report := githubauth.Check(ctx, githubauth.Options{})
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ToolResult{}, fmt.Errorf("github_auth_check: marshal report: %w", err)
	}
	if report.Status != githubauth.StatusOK {
		return ToolResult{Output: string(out)}, fmt.Errorf("github_auth_check: %s", report.Message)
	}
	return ToolResult{Output: string(out)}, nil
}
