---
id: MH-002
title: Implement tool registry and core tools
priority: high
complexity: large
source: delivery-schedule M1.2
created: 2026-04-11
---

# MH-002: Implement tool registry and core tools

## Context

The tool system is what gives the agent runtime the ability to interact with repos, files, and external systems. Each tool has a JSON Schema definition, an executor, and an allowlist check.

Reference: [docs/design-docs/agent-runtime.md](../../design-docs/agent-runtime.md) (AD-005)

## Requirements

### Tool registry (`internal/tools/registry.go`)
- Typed tool registry: register tools with name, description, JSON Schema for parameters
- Generate tool definitions array for LLM requests
- Look up tool by name for execution
- Allowlist enforcement: only tools listed in role config are available

### Tool executor (`internal/tools/executor.go`)
- Execute a tool call: parse arguments, run the tool function, capture output
- Timeout enforcement per tool call
- Structured result: output string, error string, exit code (for shell tools)

### Core tools (each its own file with tests)
- `file_read`: read file contents with optional line range
- `file_write`: write or create a file
- `file_search`: glob-based file search returning matching paths
- `grep`: regex search across files (use Go stdlib regexp)
- `shell_exec`: run a shell command with timeout, capture stdout/stderr
- `git_status`: run `git status --porcelain`
- `git_diff`: run `git diff` with optional path filter
- `git_commit`: stage files and commit with message
- `git_branch`: create and checkout a branch
- `git_push`: push current branch to remote
- `github_pr_create`: stub (real implementation in M4)
- `github_pr_comment`: stub (real implementation in M4)
- `github_check_run`: stub (real implementation in M4)

## Affected Files

- `internal/tools/registry.go`, `registry_test.go`
- `internal/tools/executor.go`, `executor_test.go`
- `internal/tools/file_read.go`, `file_read_test.go`
- `internal/tools/file_write.go`, `file_write_test.go`
- `internal/tools/file_search.go`, `file_search_test.go`
- `internal/tools/grep.go`, `grep_test.go`
- `internal/tools/shell_exec.go`, `shell_exec_test.go`
- `internal/tools/git.go`, `git_test.go`
- `internal/tools/github_stub.go`

## Acceptance Criteria

### Functional (happy path)
- [ ] Registry returns correct JSON Schema definitions for all registered tools
- [ ] `file_read` reads a file and returns contents
- [ ] `file_read` with line range returns only specified lines
- [ ] `file_write` creates a new file
- [ ] `file_write` overwrites an existing file
- [ ] `file_search` finds files matching a glob pattern
- [ ] `grep` finds lines matching a regex across multiple files
- [ ] `shell_exec` runs a command and captures stdout and stderr
- [ ] `git_commit` stages and commits files
- [ ] `git_branch` creates and checks out a branch
- [ ] Allowlist blocks tools not in the role config

### Edge cases and negative paths
- [ ] `file_read` on non-existent file returns descriptive error
- [ ] `file_write` to a path outside the working directory is blocked (sandbox boundary)
- [ ] `shell_exec` with a command that times out is killed and returns timeout error
- [ ] `shell_exec` with a command that produces >1MB output is truncated
- [ ] `grep` with invalid regex returns descriptive error
- [ ] Tool call with unknown tool name returns "tool not found" error
- [ ] Tool call with wrong argument types returns descriptive error

### Non-goals
- [ ] GitHub tools are stubs only (MH-009 in M4 implements them)
- [ ] Sandbox enforcement is not in this ticket (M5b)

### Observability, docs, and regressions
- [ ] Each tool has unit tests in its own test file
- [ ] Registry test verifies schema generation matches expected format
- [ ] Allowlist test verifies blocked tools cannot be executed

## Notes

All file tools operate on a working directory passed as context. The sandbox (M5b) will enforce this at the process level; for now, tools check the path prefix themselves.
