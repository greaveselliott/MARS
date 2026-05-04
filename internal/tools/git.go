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
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const gitStatusSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {}
}`

const gitDiffSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "paths": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Optional paths relative to repo root to pass to git diff"
    }
  }
}`

const gitCommitSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "message": { "type": "string" },
    "paths": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Paths to stage; if empty, stages all changes (git add -A)"
    }
  },
  "required": ["message"]
}`

const gitBranchSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "name": { "type": "string" },
    "create": { "type": "boolean", "description": "If true (default), create branch with checkout -b" }
  },
  "required": ["name"]
}`

const gitPushSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "remote": { "type": "string", "description": "Remote name (default origin)" },
    "branch": { "type": "string", "description": "Branch to push (default current branch)" }
  }
}`

type gitDiffArgs struct {
	Paths []string `json:"paths"`
}

type gitCommitArgs struct {
	Message string   `json:"message"`
	Paths   []string `json:"paths"`
}

type gitBranchArgs struct {
	Name   string `json:"name"`
	Create *bool  `json:"create"`
}

type gitPushArgs struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

func registerGitTools(r *Registry) error {
	if err := r.Register("git_status", "Run git status --porcelain in the repository.", json.RawMessage(gitStatusSchema), handleGitStatus); err != nil {
		return err
	}
	if err := r.Register("git_diff", "Run git diff with optional path filters.", json.RawMessage(gitDiffSchema), handleGitDiff); err != nil {
		return err
	}
	if err := r.Register("git_commit", "Stage files and create a commit.", json.RawMessage(gitCommitSchema), handleGitCommit); err != nil {
		return err
	}
	if err := r.Register("git_branch", "Create or switch the current branch.", json.RawMessage(gitBranchSchema), handleGitBranch); err != nil {
		return err
	}
	return r.Register("git_push", "Push the current (or named) branch to a remote.", json.RawMessage(gitPushSchema), handleGitPush)
}

func handleGitStatus(ctx context.Context, root Root, _ json.RawMessage) (ToolResult, error) {
	tr, err := runGit(ctx, root, "status", "--porcelain")
	if err != nil {
		return tr, err
	}
	if tr.ExitCode != 0 {
		return tr, fmt.Errorf("git_status: %s", strings.TrimSpace(tr.Stderr))
	}
	return tr, nil
}

func handleGitDiff(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args gitDiffArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("git_diff: parse arguments: %w", err)
	}
	cmd := []string{"diff", "--"}
	for _, p := range args.Paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		abs, err := root.ResolvePath(p)
		if err != nil {
			return ToolResult{}, fmt.Errorf("git_diff: %w", err)
		}
		cmd = append(cmd, abs)
	}
	if len(cmd) == 2 {
		cmd = []string{"diff"}
	}
	tr, err := runGit(ctx, root, cmd...)
	if err != nil {
		return tr, err
	}
	if tr.ExitCode != 0 {
		return tr, fmt.Errorf("git_diff: %s", strings.TrimSpace(tr.Stderr))
	}
	return tr, nil
}

func handleGitCommit(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args gitCommitArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("git_commit: parse arguments: %w", err)
	}
	if strings.TrimSpace(args.Message) == "" {
		return ToolResult{}, fmt.Errorf("git_commit: field message is required")
	}
	if len(args.Paths) == 0 {
		if err := runGitExit0(ctx, root, "add", "-A"); err != nil {
			return ToolResult{}, err
		}
	} else {
		for _, p := range args.Paths {
			abs, err := root.ResolvePath(p)
			if err != nil {
				return ToolResult{}, fmt.Errorf("git_commit: %w", err)
			}
			if err := runGitExit0(ctx, root, "add", "--", abs); err != nil {
				return ToolResult{}, err
			}
		}
	}
	if err := runGitExit0(ctx, root, "commit", "-m", args.Message); err != nil {
		return ToolResult{}, err
	}
	tr, err := runGit(ctx, root, "log", "-1", "--oneline")
	if err != nil || tr.ExitCode != 0 {
		return ToolResult{Output: "commit completed"}, nil
	}
	return tr, nil
}

func handleGitBranch(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args gitBranchArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("git_branch: parse arguments: %w", err)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return ToolResult{}, fmt.Errorf("git_branch: field name is required")
	}
	create := true
	if args.Create != nil {
		create = *args.Create
	}
	if create {
		if err := runGitExit0(ctx, root, "checkout", "-b", name); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: fmt.Sprintf("created and checked out branch %q", name)}, nil
	}
	if err := runGitExit0(ctx, root, "checkout", name); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: fmt.Sprintf("checked out branch %q", name)}, nil
}

func handleGitPush(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args gitPushArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("git_push: parse arguments: %w", err)
	}
	remote := strings.TrimSpace(args.Remote)
	if remote == "" {
		remote = "origin"
	}
	branch := strings.TrimSpace(args.Branch)
	if branch == "" {
		out, err := runGit(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return out, err
		}
		if out.ExitCode != 0 {
			return out, fmt.Errorf("git_push: %s", strings.TrimSpace(out.Stderr))
		}
		branch = strings.TrimSpace(out.Output)
	}
	if branch == "" {
		return ToolResult{}, fmt.Errorf("git_push: could not determine current branch")
	}
	tr, err := runGit(ctx, root, "push", remote, branch)
	if err != nil {
		return tr, err
	}
	if tr.ExitCode != 0 {
		msg := strings.TrimSpace(tr.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(tr.Output)
		}
		return tr, fmt.Errorf("git_push: %s", msg)
	}
	return tr, nil
}

func runGitExit0(ctx context.Context, root Root, args ...string) error {
	tr, err := runGit(ctx, root, args...)
	if err != nil {
		return err
	}
	if tr.ExitCode != 0 {
		msg := strings.TrimSpace(tr.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(tr.Output)
		}
		return fmt.Errorf("git %s failed (exit %d): %s", strings.Join(args, " "), tr.ExitCode, msg)
	}
	return nil
}

func runGit(ctx context.Context, root Root, args ...string) (ToolResult, error) {
	full := append([]string{"-C", root.Abs()}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			return ToolResult{Stderr: stderr.String(), Output: stdout.String()}, fmt.Errorf("git: %w", err)
		}
	}
	outStr, truncOut := capString(stdout.String(), DefaultMaxToolOutputBytes/2)
	errStr, truncErr := capString(stderr.String(), DefaultMaxToolOutputBytes/2)
	return ToolResult{
		Output:    outStr,
		Stderr:    errStr,
		ExitCode:  exit,
		Truncated: truncOut || truncErr,
	}, nil
}
