/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/tools-glossary.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/docsync"
)

const simpleWorkflowSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "mode": { "type": "string", "description": "Optional mode such as reference, audit, status, plan, or summarize." },
    "notes": { "type": "string", "description": "Optional task notes or transcript excerpt to include in the analysis." }
  }
}`

type simpleWorkflowArgs struct {
	Mode  string `json:"mode"`
	Notes string `json:"notes"`
}

func registerReleaseOrchestrate(r *Registry) error {
	return r.Register(
		"release_orchestrate",
		"Plan and preflight the complete Mars Harness release ritual from semantic commit through pushed tag and verified assets.",
		json.RawMessage(simpleWorkflowSchema),
		handleReleaseOrchestrate,
	)
}

func registerGithubReleaseStatus(r *Registry) error {
	return r.Register(
		"github_release_status",
		"Summarize local GitHub release readiness signals and the commands needed to inspect remote workflow, release, tag, and asset state.",
		json.RawMessage(simpleWorkflowSchema),
		handleGithubReleaseStatus,
	)
}

func registerArchitectureAudit(r *Registry) error {
	return r.Register(
		"architecture_audit",
		"Audit architecture documentation against current CLI, generated harness layout, tool registry, and known runtime boundaries.",
		json.RawMessage(simpleWorkflowSchema),
		handleArchitectureAudit,
	)
}

func registerHarnessDoctrineSync(r *Registry) error {
	return r.Register(
		"harness_doctrine_sync",
		"Audit mirrored foundation and deployed harness doctrine for glossary, tools, operating-model, and generated-target consistency.",
		json.RawMessage(simpleWorkflowSchema),
		handleHarnessDoctrineSync,
	)
}

func registerDocSyncAudit(r *Registry) error {
	return r.Register(
		"docsync_audit",
		"Audit source files for MarsDocSync metadata and associated documentation freshness pointers.",
		json.RawMessage(simpleWorkflowSchema),
		handleDocSyncAudit,
	)
}

func registerGitReleaseGuard(r *Registry) error {
	return r.Register(
		"git_release_guard",
		"Check local git/version release invariants before or after release-note generation.",
		json.RawMessage(simpleWorkflowSchema),
		handleGitReleaseGuard,
	)
}

func registerToolInventoryAudit(r *Registry) error {
	return r.Register(
		"tool_inventory_audit",
		"Compare registered built-in tools, mutating policy, tools glossary, and generated target guidance.",
		json.RawMessage(simpleWorkflowSchema),
		handleToolInventoryAudit,
	)
}

func registerTaskTraceSummarize(r *Registry) error {
	return r.Register(
		"task_trace_summarize",
		"Summarize a recent work trace and identify repeated processes that should become formal tools.",
		json.RawMessage(simpleWorkflowSchema),
		handleTaskTraceSummarize,
	)
}

func handleReleaseOrchestrate(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	args, err := parseSimpleWorkflowArgs(raw)
	if err != nil {
		return ToolResult{}, fmt.Errorf("release_orchestrate: %w", err)
	}
	version := readOptional(root, "VERSION")
	status := gitOutput(ctx, root, "status", "--short")
	head := gitOutput(ctx, root, "log", "-3", "--oneline")
	out := strings.TrimSpace(fmt.Sprintf(`# release_orchestrate

Purpose: formalize the end-to-end Mars Harness release ritual. Use this tool before mutating release state, then use `+"`mars_harness_cli`"+` and git tools for the actual steps unless this tool has been explicitly extended to execute them.

Current VERSION: %s
Git status:
%s

Recent commits:
%s

Sequence:
1. Ensure git status is clean except the intended semantic change.
2. Commit the semantic change with a conventional message.
3. Run `+"`mars_harness_cli`"+` with args ["release", "notes", "--repo", ".", "--bump", "auto"].
4. Review VERSION, CHANGELOG.md, and buildinfo changes.
5. Commit generated files as `+"`release: notes X.Y.Z`"+`.
6. Push main.
7. Tag the release-note commit as `+"`vX.Y.Z`"+` and push the tag.
8. Wait for the Release workflow to complete.
9. Run `+"`mars_harness_cli`"+` with args ["release", "verify-assets", "--version", "vX.Y.Z"].
10. If verification fails, record the blocker before treating release work as complete.
`, strings.TrimSpace(version), nonEmpty(status, "(clean)"), nonEmpty(head, "(no commits)")))
	if strings.TrimSpace(args.Notes) != "" {
		out += "\nTask notes:\n" + strings.TrimSpace(args.Notes)
	}
	return ToolResult{Output: out}, nil
}

func handleGithubReleaseStatus(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	if _, err := parseSimpleWorkflowArgs(raw); err != nil {
		return ToolResult{}, fmt.Errorf("github_release_status: %w", err)
	}
	tags := gitOutput(ctx, root, "tag", "--list", "v*", "--sort=-version:refname")
	head := gitOutput(ctx, root, "show", "--no-patch", "--decorate", "--oneline", "HEAD")
	out := fmt.Sprintf(`# github_release_status

Local HEAD:
%s

Recent local release tags:
%s

Remote inspection commands:
- `+"`gh run list --repo <owner/name> --limit 10`"+`
- `+"`gh run view <run-id> --repo <owner/name> --json status,conclusion,url`"+`
- `+"`gh release view vX.Y.Z --repo <owner/name> --json tagName,name,assets,url,isDraft,isPrerelease,publishedAt`"+`
- `+"`mars-harness release verify-assets --version vX.Y.Z`"+`

Interpretation:
- Tag exists but release is 404: wait for the Release workflow or inspect the workflow failure.
- Release exists but assets are missing: wait for upload completion, rerun failed release jobs, or record the blocker.
- Local tag and remote tag disagree: stop and resolve tag drift before publishing installer guidance.
`, nonEmpty(head, "(unknown)"), firstLines(tags, 10))
	return ToolResult{Output: out}, nil
}

func handleArchitectureAudit(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	if _, err := parseSimpleWorkflowArgs(raw); err != nil {
		return ToolResult{}, fmt.Errorf("architecture_audit: %w", err)
	}
	arch := readOptional(root, "ARCHITECTURE.md")
	findings := checkContains("ARCHITECTURE.md", arch, []string{
		"mars-harness update tool",
		"mars_harness_cli",
		".harness/metadata.yaml",
		"docs/QUALITY_SCORE.md",
		"BDD-led",
		"it does not clone a fresh working directory per job",
	})
	stale := checkNotContains("ARCHITECTURE.md", arch, []string{
		"CLI --> STOP",
		"policies/*.yaml",
		"knowledge-routes.yaml",
		"bundle.lock.json",
	})
	findings = append(findings, stale...)
	return ToolResult{Output: renderAudit("architecture_audit", findings)}, nil
}

func handleHarnessDoctrineSync(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	if _, err := parseSimpleWorkflowArgs(raw); err != nil {
		return ToolResult{}, fmt.Errorf("harness_doctrine_sync: %w", err)
	}
	checks := []string{}
	for _, item := range []struct {
		path  string
		terms []string
	}{
		{"AGENTS.md", []string{"Operating model", "Mirrored tools", "docs/design-docs/tools-glossary.md", "docs/design-docs/documentation-sync-architecture.md", "docs/design-docs/cli-tool-skill-sync.md"}},
		{"docs/design-docs/harness-glossary.md", []string{"Symbiotic operating-model change", "Formalized tool creation trigger"}},
		{"docs/design-docs/tools-glossary.md", []string{"release_orchestrate", "docsync_audit", "tool_creation_guard", "tool_inventory_audit", "task_trace_summarize"}},
		{"docs/design-docs/delivery-operating-model.md", []string{"formalized tools", "repeated process", "docsync_audit", "documentation-sync-architecture.md", "cli-tool-skill-sync.md"}},
		{"docs/design-docs/documentation-sync-architecture.md", []string{"AD-102", "Universal Operating Model", "docsync_audit"}},
		{"docs/design-docs/cli-tool-skill-sync.md", []string{"AD-103", "mars_harness_cli", "repo shortcut map", "skills"}},
		{"internal/scanner/init.go", []string{"release_orchestrate", "docsync_audit", "documentation-sync-architecture.md", "cli-tool-skill-sync.md", "Formalized tool creation trigger"}},
	} {
		checks = append(checks, checkContains(item.path, readOptional(root, item.path), item.terms)...)
	}
	return ToolResult{Output: renderAudit("harness_doctrine_sync", checks)}, nil
}

func handleDocSyncAudit(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	if _, err := parseSimpleWorkflowArgs(raw); err != nil {
		return ToolResult{}, fmt.Errorf("docsync_audit: %w", err)
	}
	report, err := docsync.Audit(docsync.Config{RepoRoot: root.Abs()})
	if err != nil {
		return ToolResult{}, err
	}
	var lines []string
	lines = append(lines, "# docsync_audit", "", report.Summary())
	for _, finding := range report.Findings {
		lines = append(lines, "FAIL: "+finding.Path+": "+finding.Message)
	}
	if report.OK() {
		lines = append(lines, "Status: ok")
	}
	return ToolResult{Output: strings.Join(lines, "\n")}, nil
}

func handleGitReleaseGuard(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	if _, err := parseSimpleWorkflowArgs(raw); err != nil {
		return ToolResult{}, fmt.Errorf("git_release_guard: %w", err)
	}
	version := strings.TrimSpace(readOptional(root, "VERSION"))
	head := gitOutput(ctx, root, "log", "-1", "--pretty=%s")
	tagExists := false
	if version != "" {
		tagExists = gitExit0(ctx, root, "rev-parse", "--verify", "v"+version)
	}
	status := gitOutput(ctx, root, "status", "--short")
	var checks []string
	checks = append(checks, passFail(version != "", "VERSION is present", "VERSION is missing"))
	checks = append(checks, passFail(strings.HasPrefix(head, "release: notes ") || strings.TrimSpace(status) != "", "HEAD is release-note commit or worktree has unreleased changes", "HEAD is not a release-note commit and worktree is clean"))
	if version != "" {
		checks = append(checks, passFail(tagExists, "tag v"+version+" exists locally", "tag v"+version+" is missing locally"))
	}
	checks = append(checks, "git status:\n"+nonEmpty(status, "(clean)"))
	return ToolResult{Output: "# git_release_guard\n\n" + strings.Join(checks, "\n")}, nil
}

func handleToolInventoryAudit(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	if _, err := parseSimpleWorkflowArgs(raw); err != nil {
		return ToolResult{}, fmt.Errorf("tool_inventory_audit: %w", err)
	}
	reg, err := DefaultRegistry()
	if err != nil {
		return ToolResult{}, err
	}
	names := reg.Names()
	glossary := readOptional(root, "docs/design-docs/tools-glossary.md")
	generated := readOptional(root, "internal/scanner/init.go")
	var checks []string
	for _, name := range names {
		checks = append(checks, passFail(strings.Contains(glossary, "`"+name+"`"), "glossary includes "+name, "glossary missing "+name))
		if isMirroredWorkflowTool(name) || name == "mars_harness_cli" || name == "tool_create" {
			checks = append(checks, passFail(strings.Contains(generated, name), "generated harness includes "+name, "generated harness missing "+name))
		}
	}
	var mutating []string
	for _, name := range names {
		if mutatingTools[name] {
			mutating = append(mutating, name)
		}
	}
	sort.Strings(mutating)
	out := "# tool_inventory_audit\n\nRegistered tools:\n" + strings.Join(names, ", ") + "\n\nMutating tools:\n" + strings.Join(mutating, ", ") + "\n\n" + strings.Join(checks, "\n")
	return ToolResult{Output: out}, nil
}

func handleTaskTraceSummarize(_ context.Context, _ Root, raw json.RawMessage) (ToolResult, error) {
	args, err := parseSimpleWorkflowArgs(raw)
	if err != nil {
		return ToolResult{}, fmt.Errorf("task_trace_summarize: %w", err)
	}
	notes := strings.TrimSpace(args.Notes)
	if notes == "" {
		notes = "No notes were provided. Paste a concise transcript, command log, or task summary into `notes` for a stronger summary."
	}
	candidates := []string{
		"Repeated release/version/tag/asset steps -> release_orchestrate or github_release_status.",
		"Repeated doc-vs-code freshness checks -> architecture_audit or harness_doctrine_sync.",
		"Repeated registry/glossary/allowlist checks -> tool_inventory_audit.",
		"Repeated git version invariant checks -> git_release_guard.",
		"Repeated manual summaries of commands, files, tests, commits, and blockers -> task_trace_summarize.",
	}
	out := "# task_trace_summarize\n\nTrace notes:\n" + notes + "\n\nCandidate formal tools:\n- " + strings.Join(candidates, "\n- ")
	return ToolResult{Output: out}, nil
}

func parseSimpleWorkflowArgs(raw json.RawMessage) (simpleWorkflowArgs, error) {
	var args simpleWorkflowArgs
	if len(raw) == 0 {
		return args, nil
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return args, fmt.Errorf("parse arguments: %w", err)
	}
	return args, nil
}

func readOptional(root Root, rel string) string {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	return string(b)
}

func gitOutput(ctx context.Context, root Root, args ...string) string {
	tr, err := runGit(ctx, root, args...)
	if err != nil {
		return err.Error()
	}
	if tr.ExitCode != 0 {
		return strings.TrimSpace(tr.Stderr)
	}
	return strings.TrimSpace(tr.Output)
}

func gitExit0(ctx context.Context, root Root, args ...string) bool {
	tr, err := runGit(ctx, root, args...)
	return err == nil && tr.ExitCode == 0
}

func checkContains(path, content string, terms []string) []string {
	var out []string
	for _, term := range terms {
		out = append(out, passFail(strings.Contains(content, term), path+" contains "+term, path+" missing "+term))
	}
	return out
}

func checkNotContains(path, content string, terms []string) []string {
	var out []string
	for _, term := range terms {
		out = append(out, passFail(!strings.Contains(content, term), path+" omits stale "+term, path+" still contains stale "+term))
	}
	return out
}

func renderAudit(name string, checks []string) string {
	if len(checks) == 0 {
		return "# " + name + "\n\nNo checks ran."
	}
	return "# " + name + "\n\n" + strings.Join(checks, "\n")
}

func passFail(ok bool, pass, fail string) string {
	if ok {
		return "PASS: " + pass
	}
	return "FAIL: " + fail
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return strings.TrimSpace(s)
}

func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "(none)"
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func isMirroredWorkflowTool(name string) bool {
	return slices.Contains([]string{
		"release_orchestrate",
		"github_release_status",
		"architecture_audit",
		"harness_doctrine_sync",
		"docsync_audit",
		"git_release_guard",
		"tool_creation_guard",
		"tool_inventory_audit",
		"task_trace_summarize",
	}, name)
}
