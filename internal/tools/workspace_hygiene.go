/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const workspaceHygieneSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "mode": { "type": "string", "description": "Audit mode: audit, pre_job, pre_dependency, or post_dependency." },
    "paths": { "type": "array", "description": "Optional repo-relative path filters to focus the audit." }
  },
  "required": []
}`

type workspaceHygieneArgs struct {
	Mode  string   `json:"mode"`
	Paths []string `json:"paths"`
}

func registerWorkspaceHygiene(r *Registry) error {
	return r.Register("workspace_hygiene", "Audit repository workspace hygiene before agent jobs or dependency mutations.", json.RawMessage(workspaceHygieneSchema), handleWorkspaceHygiene)
}

func handleWorkspaceHygiene(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args workspaceHygieneArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("workspace_hygiene: parse arguments: %w", err)
	}
	report, err := AuditWorkspaceHygiene(ctx, root, WorkspaceHygieneOptions{
		Mode:  args.Mode,
		Paths: args.Paths,
	})
	if err != nil {
		return ToolResult{}, err
	}
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ToolResult{}, fmt.Errorf("workspace_hygiene: marshal report: %w", err)
	}
	if report.Blocking && workspaceHygieneModeBlocks(report.Mode) {
		return ToolResult{Output: string(out)}, fmt.Errorf("workspace_hygiene_blocked: %s", report.Message)
	}
	return ToolResult{Output: string(out)}, nil
}

// WorkspaceHygieneOptions controls deterministic workspace hygiene audits.
type WorkspaceHygieneOptions struct {
	Mode  string
	Paths []string
}

// WorkspaceHygieneReport is the shared report shape used by the tool, doctor,
// scanner, dependency sync, and server pre-job gate.
type WorkspaceHygieneReport struct {
	Status     string                    `json:"status"`
	Mode       string                    `json:"mode"`
	Blocking   bool                      `json:"blocking"`
	Findings   []WorkspaceHygieneFinding `json:"findings"`
	RecipeID   string                    `json:"recipe_id,omitempty"`
	Message    string                    `json:"message"`
	NextAction string                    `json:"next_action,omitempty"`
}

// WorkspaceHygieneFinding describes one deterministic workspace hygiene issue.
type WorkspaceHygieneFinding struct {
	Type       string   `json:"type"`
	Path       string   `json:"path,omitempty"`
	Severity   string   `json:"severity"`
	Blocking   bool     `json:"blocking"`
	Message    string   `json:"message"`
	RecipeID   string   `json:"recipe_id,omitempty"`
	NextAction string   `json:"next_action,omitempty"`
	Paths      []string `json:"paths,omitempty"`
}

const (
	workspaceHygieneModeAudit          = "audit"
	workspaceHygieneModePreJob         = "pre_job"
	workspaceHygieneModePreDependency  = "pre_dependency"
	workspaceHygieneModePostDependency = "post_dependency"

	workspaceRecipeAddIgnore        = "workspace-hygiene:add-ignore"
	workspaceRecipeTrackGenerated   = "workspace-hygiene:tracked-generated"
	workspaceRecipeGeneratedDirty   = "workspace-hygiene:generated-dirty"
	workspaceRecipeForbiddenDelete  = "workspace-hygiene:forbidden-delete"
	workspaceRecipeLargeGenerated   = "workspace-hygiene:large-generated-diff"
	workspaceRecipeDependencyReview = "workspace-hygiene:dependency-review"
)

var generatedWorkspaceDirs = []string{
	"node_modules",
	".next",
	"dist",
	"build",
	"coverage",
	"target",
	"vendor",
	".venv",
	"__pycache__",
}

var generatedDirIgnoreHints = map[string][]string{
	"node_modules": {"node_modules/"},
	".next":        {".next/"},
	"dist":         {"dist/"},
	"build":        {"build/"},
	"coverage":     {"coverage/"},
	"target":       {"target/"},
	"vendor":       {"vendor/"},
	".venv":        {".venv/"},
	"__pycache__":  {"__pycache__/"},
}

// GeneratedWorkspaceDirs returns directories that are treated as generated
// dependency or build output for context and hygiene checks.
func GeneratedWorkspaceDirs() []string {
	out := append([]string(nil), generatedWorkspaceDirs...)
	sort.Strings(out)
	return out
}

// IsGeneratedWorkspacePath reports whether rel is under a generated dependency
// or build-output directory.
func IsGeneratedWorkspacePath(rel string) bool {
	rel = cleanRepoPath(rel)
	if rel == "." || rel == "" {
		return false
	}
	for _, dir := range generatedWorkspaceDirs {
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}

// AuditWorkspaceHygiene inspects repo-visible state for generated dependency
// and build churn without mutating the repository.
func AuditWorkspaceHygiene(ctx context.Context, root Root, opts WorkspaceHygieneOptions) (WorkspaceHygieneReport, error) {
	mode := normalizeWorkspaceHygieneMode(opts.Mode)
	scope := normalizeHygieneScope(opts.Paths)
	report := WorkspaceHygieneReport{
		Status:  "clean",
		Mode:    mode,
		Message: "workspace hygiene is clean",
	}

	findings := append([]WorkspaceHygieneFinding{}, missingIgnoreFindings(root, mode, scope)...)
	gitFindings, err := gitWorkspaceHygieneFindings(ctx, root, mode, scope)
	if err != nil {
		return report, err
	}
	findings = append(findings, gitFindings...)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Blocking != findings[j].Blocking {
			return findings[i].Blocking
		}
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity > findings[j].Severity
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Type < findings[j].Type
	})

	report.Findings = findings
	for _, f := range findings {
		if f.Blocking {
			report.Blocking = true
			report.Status = "blocked"
			report.RecipeID = f.RecipeID
			report.NextAction = f.NextAction
			report.Message = f.Message
			return report, nil
		}
	}
	if len(findings) > 0 {
		report.Status = "safe"
		report.Message = fmt.Sprintf("workspace hygiene has %d non-blocking finding(s)", len(findings))
	}
	return report, nil
}

func normalizeWorkspaceHygieneMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case workspaceHygieneModePreJob:
		return workspaceHygieneModePreJob
	case workspaceHygieneModePreDependency:
		return workspaceHygieneModePreDependency
	case workspaceHygieneModePostDependency:
		return workspaceHygieneModePostDependency
	default:
		return workspaceHygieneModeAudit
	}
}

func workspaceHygieneModeBlocks(mode string) bool {
	switch normalizeWorkspaceHygieneMode(mode) {
	case workspaceHygieneModePreJob, workspaceHygieneModePreDependency, workspaceHygieneModePostDependency:
		return true
	default:
		return false
	}
}

func normalizeHygieneScope(paths []string) []string {
	var out []string
	for _, p := range paths {
		p = cleanRepoPath(p)
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func hygienePathInScope(rel string, scope []string) bool {
	if len(scope) == 0 {
		return true
	}
	rel = cleanRepoPath(rel)
	for _, p := range scope {
		if rel == p || strings.HasPrefix(rel, p+"/") || strings.HasPrefix(p, rel+"/") {
			return true
		}
	}
	return false
}

func missingIgnoreFindings(root Root, mode string, scope []string) []WorkspaceHygieneFinding {
	required := requiredGeneratedIgnores(root)
	if len(required) == 0 {
		return nil
	}
	ignored := loadGitignoreCoverage(root)
	var findings []WorkspaceHygieneFinding
	for _, dir := range required {
		if !hygienePathInScope(dir, scope) || ignoreCoversGeneratedDir(ignored, dir) {
			continue
		}
		exists := pathExists(root, dir)
		blocking := mode == workspaceHygieneModePreDependency || mode == workspaceHygieneModePostDependency || (mode == workspaceHygieneModePreJob && exists)
		severity := "medium"
		if blocking {
			severity = "high"
		}
		hint := generatedDirIgnoreHints[dir][0]
		findings = append(findings, WorkspaceHygieneFinding{
			Type:       "missing_generated_ignore",
			Path:       ".gitignore",
			Severity:   severity,
			Blocking:   blocking,
			Message:    fmt.Sprintf("generated directory %s is not ignored", dir),
			RecipeID:   workspaceRecipeAddIgnore,
			NextAction: fmt.Sprintf("Add %q to .gitignore, commit the ignore policy, then rerun workspace_hygiene.", hint),
			Paths:      []string{dir},
		})
	}
	return findings
}

func requiredGeneratedIgnores(root Root) []string {
	var required []string
	if repoFileExists(root, "package.json") {
		required = append(required, "node_modules")
		if repoFileExists(root, "next.config.js") || repoFileExists(root, "next.config.mjs") || repoFileExists(root, "next.config.ts") {
			required = append(required, ".next")
		}
	}
	if repoFileExists(root, "Cargo.toml") {
		required = append(required, "target")
	}
	if repoFileExists(root, "composer.json") {
		required = append(required, "vendor")
	}
	if repoFileExists(root, "Gemfile") {
		required = append(required, "vendor")
	}
	return compactStrings(required)
}

func loadGitignoreCoverage(root Root) []string {
	abs, err := root.ResolvePath(".gitignore")
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "/")
		line = strings.TrimSuffix(line, "/")
		lines = append(lines, line)
	}
	return lines
}

func ignoreCoversGeneratedDir(ignoreLines []string, dir string) bool {
	dir = strings.Trim(dir, "/")
	for _, line := range ignoreLines {
		line = strings.Trim(line, "/")
		if line == dir || line == "**/"+dir || strings.HasSuffix(line, "/"+dir) {
			return true
		}
	}
	return false
}

func gitWorkspaceHygieneFindings(ctx context.Context, root Root, mode string, scope []string) ([]WorkspaceHygieneFinding, error) {
	var findings []WorkspaceHygieneFinding
	tracked, err := runGit(ctx, root, "ls-files")
	if err != nil {
		return nil, err
	}
	if tracked.ExitCode == 0 {
		grouped := generatedPathsFromLines(tracked.Output, scope)
		if len(grouped) > 0 {
			findings = append(findings, WorkspaceHygieneFinding{
				Type:       "tracked_generated_path",
				Severity:   "high",
				Blocking:   workspaceHygieneModeBlocks(mode),
				Message:    fmt.Sprintf("generated paths are tracked by git: %s", strings.Join(grouped, ", ")),
				RecipeID:   workspaceRecipeTrackGenerated,
				NextAction: "Move generated dependency/build output out of version control in an explicit human-reviewed commit, then rerun workspace_hygiene.",
				Paths:      grouped,
			})
		}
	}

	status, err := runGit(ctx, root, "status", "--porcelain", "-uall")
	if err != nil {
		return nil, err
	}
	if status.ExitCode == 0 {
		generatedDirty, deletions := parseHygieneGitStatus(status.Output, scope)
		if len(generatedDirty) > 0 {
			findings = append(findings, WorkspaceHygieneFinding{
				Type:       "generated_dirty_worktree",
				Severity:   "high",
				Blocking:   workspaceHygieneModeBlocks(mode),
				Message:    fmt.Sprintf("generated dependency/build output is dirty: %s", strings.Join(firstNStrings(generatedDirty, 8), ", ")),
				RecipeID:   workspaceRecipeGeneratedDirty,
				NextAction: "Stop ordinary agent work. Commit intended source or lockfile changes separately, fix ignore policy if needed, and perform explicit cleanup before retrying.",
				Paths:      generatedDirty,
			})
		}
		if len(deletions) > 0 {
			findings = append(findings, WorkspaceHygieneFinding{
				Type:       "forbidden_deletion",
				Severity:   "high",
				Blocking:   workspaceHygieneModeBlocks(mode),
				Message:    fmt.Sprintf("worktree contains deletion(s): %s", strings.Join(firstNStrings(deletions, 8), ", ")),
				RecipeID:   workspaceRecipeForbiddenDelete,
				NextAction: "Do not continue autonomous work over deletions. Restore, intentionally commit, or explicitly approve the deletion before retrying.",
				Paths:      deletions,
			})
		}
	}

	large, err := largeGeneratedDiffs(ctx, root, scope)
	if err != nil {
		return nil, err
	}
	if len(large) > 0 {
		findings = append(findings, WorkspaceHygieneFinding{
			Type:       "large_generated_diff",
			Severity:   "high",
			Blocking:   workspaceHygieneModeBlocks(mode),
			Message:    fmt.Sprintf("generated path diff exceeds safe context/blast-radius limits: %s", strings.Join(firstNStrings(large, 8), ", ")),
			RecipeID:   workspaceRecipeLargeGenerated,
			NextAction: "Use workspace_hygiene and dependency_sync rather than exposing generated diffs to the model; fix ignore/tracking policy before retrying.",
			Paths:      large,
		})
	}
	return findings, nil
}

func generatedPathsFromLines(output string, scope []string) []string {
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		rel := cleanRepoPath(line)
		if rel == "" || !hygienePathInScope(rel, scope) || !IsGeneratedWorkspacePath(rel) {
			continue
		}
		dir := generatedRoot(rel)
		if dir != "" {
			seen[dir] = true
		}
	}
	return sortedKeys(seen)
}

func parseHygieneGitStatus(output string, scope []string) ([]string, []string) {
	generatedDirty := map[string]bool{}
	deletions := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 4 {
			continue
		}
		code := line[:2]
		path := cleanRepoPath(strings.TrimSpace(line[3:]))
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = cleanRepoPath(parts[len(parts)-1])
		}
		if path == "" || !hygienePathInScope(path, scope) {
			continue
		}
		if strings.Contains(code, "D") {
			deletions[path] = true
		}
		if IsGeneratedWorkspacePath(path) {
			generatedDirty[path] = true
		}
	}
	return sortedKeys(generatedDirty), sortedKeys(deletions)
}

func largeGeneratedDiffs(ctx context.Context, root Root, scope []string) ([]string, error) {
	const largeGeneratedDiffLineLimit = 500
	large := map[string]bool{}
	numstat, err := runGit(ctx, root, "diff", "--numstat", "HEAD", "--")
	if err != nil {
		return nil, err
	}
	if numstat.ExitCode == 0 {
		for _, line := range strings.Split(numstat.Output, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			path := cleanRepoPath(strings.Join(fields[2:], " "))
			if path == "" || !IsGeneratedWorkspacePath(path) || !hygienePathInScope(path, scope) {
				continue
			}
			lines := atoiHygieneDiffField(fields[0]) + atoiHygieneDiffField(fields[1])
			if lines > largeGeneratedDiffLineLimit {
				large[path] = true
			}
		}
	}
	untracked, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	if untracked.ExitCode == 0 {
		for _, line := range strings.Split(untracked.Output, "\n") {
			rel := cleanRepoPath(line)
			if rel == "" || !IsGeneratedWorkspacePath(rel) || !hygienePathInScope(rel, scope) {
				continue
			}
			if countFileLines(root, rel) > largeGeneratedDiffLineLimit {
				large[rel] = true
			}
		}
	}
	return sortedKeys(large), nil
}

func generatedRoot(rel string) string {
	rel = cleanRepoPath(rel)
	for _, dir := range generatedWorkspaceDirs {
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return dir
		}
	}
	return ""
}

func countFileLines(root Root, rel string) int {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return 0
	}
	lines := strings.Count(string(data), "\n")
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		lines++
	}
	return lines
}

func pathExists(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	_, err = os.Stat(abs)
	return err == nil
}

func compactStrings(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	return sortedKeys(seen)
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func firstNStrings(values []string, n int) []string {
	if len(values) <= n {
		return values
	}
	out := append([]string(nil), values[:n]...)
	out = append(out, fmt.Sprintf("... +%d more", len(values)-n))
	return out
}

func atoiHygieneDiffField(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func generatedPathspecExcludes() []string {
	excludes := make([]string, 0, len(generatedWorkspaceDirs))
	for _, dir := range generatedWorkspaceDirs {
		excludes = append(excludes, ":(exclude)"+dir)
	}
	return excludes
}

func filterGeneratedPaths(paths []string) (kept []string, generated []string) {
	for _, path := range paths {
		path = cleanRepoPath(path)
		if path == "" {
			continue
		}
		if IsGeneratedWorkspacePath(path) {
			generated = append(generated, path)
			continue
		}
		kept = append(kept, path)
	}
	return kept, generated
}

func isGeneratedDirName(name string) bool {
	return slices.Contains(generatedWorkspaceDirs, strings.Trim(name, "/"))
}
