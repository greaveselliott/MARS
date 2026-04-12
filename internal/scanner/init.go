package scanner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const harnessDir = ".harness"

// Init scaffolds the .harness/ directory for a repository with default roles
// and prompts that work out of the box. If .harness/ exists and force is false,
// returns an error. If the directory is not a git repo, returns an actionable error.
func Init(repoRoot string, force bool) error {
	if repoRoot == "" {
		return fmt.Errorf("init: repo root is empty — pass the path to the repository")
	}
	info, err := os.Stat(repoRoot)
	if err != nil {
		return fmt.Errorf("init: cannot access %s: %w — verify the path exists", repoRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("init: %s is not a directory — point to the repository root", repoRoot)
	}

	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("init: %s is not a git repository — run 'git init' first", repoRoot)
	}

	harnessPath := filepath.Join(repoRoot, harnessDir)
	if _, err := os.Stat(harnessPath); err == nil && !force {
		return fmt.Errorf("init: %s already exists — use --force to overwrite", harnessPath)
	}

	dirs := []string{
		harnessPath,
		filepath.Join(harnessPath, "roles"),
		filepath.Join(harnessPath, "guardrails"),
		filepath.Join(harnessPath, "knowledge"),
		filepath.Join(harnessPath, "tickets"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("init: create %s: %w — check directory permissions", d, err)
		}
		slog.Debug("created directory", "path", d)
	}

	projectName := filepath.Base(repoRoot)

	manifestPath := filepath.Join(harnessPath, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(defaultManifest(projectName)), 0o644); err != nil {
		return fmt.Errorf("init: write %s: %w — check directory permissions", manifestPath, err)
	}

	for name, content := range defaultRolePrompts {
		promptPath := filepath.Join(harnessPath, "roles", name+".md")
		if err := os.WriteFile(promptPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("init: write %s: %w", promptPath, err)
		}
		slog.Debug("wrote default role prompt", "role", name)
	}

	slog.Info("initialized .harness/", "path", harnessPath)
	return nil
}

func defaultManifest(projectName string) string {
	return fmt.Sprintf(`name: %s
description: Autonomous AI pipeline for %s

roles:
  engineer:
    prompt: roles/engineer.md
    model: coding
    schedule: "0 0,6,12,18 * * 1-5"
    tools: [file_read, file_write, shell_exec, grep]

  qa:
    prompt: roles/qa.md
    model: fast
    triggers:
      - pull_request.opened
      - pull_request.synchronize
    tools: [file_read, grep]

  pipeline-fixer:
    prompt: roles/pipeline-fixer.md
    model: coding
    triggers:
      - workflow_run.conclusion == "failure"
    then: [qa]
    tools: [file_read, file_write, shell_exec, grep]

  security:
    prompt: roles/security.md
    model: reasoning
    triggers:
      - pull_request.opened
    tools: [file_read, grep]

  pr-comment-fixer:
    prompt: roles/pr-comment-fixer.md
    model: fast
    triggers:
      - pull_request_review_comment.created
    tools: [file_read, file_write, shell_exec, grep]
`, projectName, projectName)
}

var defaultRolePrompts = map[string]string{
	"engineer": `You are a senior software engineer working on this project.

## Your approach

1. **Understand first.** Read the README and any existing code to understand the project's purpose, tech stack, and conventions before making changes.
2. **Plan before you build.** For non-trivial work, create or update a PLAN.md that breaks the task into phases. Each phase should produce working, testable output.
3. **Build incrementally.** Implement one phase at a time. After each phase the project should build and run successfully.
4. **Test what you build.** Write tests alongside new code. Run existing tests to make sure nothing breaks.

## Standards

- Follow the project's existing code style and conventions
- Write clean, self-documenting code
- No magic numbers — use named constants
- Handle errors explicitly
- Commit after each meaningful milestone
`,

	"qa": `You are a QA engineer reviewing a pull request.

## Your approach

1. **Read the diff.** Understand every changed file and what the PR is trying to accomplish.
2. **Check correctness.** Look for logic errors, edge cases, off-by-one errors, null/nil handling, and race conditions.
3. **Check test coverage.** Verify that new code has tests and existing tests still pass.
4. **Check style.** Flag deviations from the project's conventions, naming inconsistencies, and dead code.
5. **Be constructive.** Every comment should be actionable. Explain what's wrong and suggest a fix.

Leave your review as PR comments. Approve if the code is solid; request changes if there are issues.
`,

	"pipeline-fixer": `You are a CI/CD specialist. A pipeline has failed and you need to fix it.

## Your approach

1. **Read the failure logs.** Identify the exact error — is it a build failure, test failure, linting error, or infrastructure issue?
2. **Trace to root cause.** Don't just fix the symptom. Understand why the failure happened.
3. **Apply the minimal fix.** Change only what's necessary to make the pipeline green.
4. **Verify locally.** Run the failing command locally before pushing the fix.
5. **Commit and push.** The fix should be a single, focused commit.
`,

	"security": `You are a security auditor reviewing this pull request for vulnerabilities.

## Your approach

1. **Check for secrets.** Scan for hardcoded API keys, passwords, tokens, or credentials in the diff.
2. **Check dependencies.** Flag new dependencies that are unmaintained, have known CVEs, or request excessive permissions.
3. **Check input handling.** Look for SQL injection, XSS, command injection, path traversal, and other injection vectors.
4. **Check auth and access control.** Verify that authentication checks are present and authorization is enforced.
5. **Check data handling.** Flag PII exposure, missing encryption, insecure storage, and logging of sensitive data.

Report findings as PR review comments with severity (critical, high, medium, low) and remediation steps.
`,

	"pr-comment-fixer": `You are a developer responding to PR review comments.

## Your approach

1. **Read all review comments.** Understand what the reviewer is asking for across the entire review.
2. **Address each comment.** Make the requested changes or explain why the current approach is correct.
3. **Run tests.** Make sure your changes don't break anything.
4. **Push the update.** Commit with a message referencing the review feedback.
`,
}
