package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

const ghStubSchema = `{
  "type": "object",
  "additionalProperties": true,
  "properties": {}
}`

func registerGitHubStubs(r *Registry) error {
	stub := func(name string) Handler {
		return func(_ context.Context, _ Root, _ json.RawMessage) (ToolResult, error) {
			return ToolResult{}, fmt.Errorf("%s: not implemented until milestone M4 (GitHub integration)", name)
		}
	}
	if err := r.Register("github_pr_create", "Create a pull request (stub until M4).", json.RawMessage(ghStubSchema), stub("github_pr_create")); err != nil {
		return err
	}
	if err := r.Register("github_pr_comment", "Comment on a pull request (stub until M4).", json.RawMessage(ghStubSchema), stub("github_pr_comment")); err != nil {
		return err
	}
	return r.Register("github_check_run", "Create or update a check run (stub until M4).", json.RawMessage(ghStubSchema), stub("github_check_run"))
}
