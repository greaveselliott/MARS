/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
*/
package jira

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/integrations"
)

const (
	jiraOwnedStartMarker = "<!-- mars:jira-owned:start -->"
	jiraOwnedEndMarker   = "<!-- mars:jira-owned:end -->"
	scopedMarker         = "<!-- mars:scoped-marker -->"
)

var ticketIDPattern = regexp.MustCompile(`\bT-(\d+)\b`)

var jiraFrontmatterOrder = []string{
	"priority",
	"blocked_by",
	"jira_key",
	"jira_updated",
	"jira_created",
	"sprint",
	"sprint_active",
	"rank",
	"jira_status",
	"epic",
}

func MirrorRawIssue(ctx context.Context, repos []Repository, raw RawIssue) (MirrorResult, error) {
	resolved, result := resolveRepository(repos, raw.Key, raw.Project)
	if result.Status != "" {
		return result, nil
	}
	issue := raw.Normalize(resolved.Config.Ingestion.JIRA.Fields)
	return mirrorIssue(ctx, resolved, issue)
}

func MirrorIssue(ctx context.Context, repos []Repository, issue Issue) (MirrorResult, error) {
	issue = issue.Sanitized()
	resolved, result := resolveRepository(repos, issue.Key, issue.Project)
	if result.Status != "" {
		return result, nil
	}
	return mirrorIssue(ctx, resolved, issue)
}

func mirrorIssue(ctx context.Context, repo Repository, issue Issue) (MirrorResult, error) {
	if err := ctx.Err(); err != nil {
		return MirrorResult{}, err
	}
	issue = issue.Sanitized()
	result := MirrorResult{
		JiraKey:         issue.Key,
		RepoID:          repo.ID,
		RepoPath:        repo.Path,
		LLMJobsEnqueued: 0,
	}
	if issue.Key == "" || issue.Project == "" {
		result.Status = StatusDropped
		result.Reason = "missing_issue_identity"
		return result, nil
	}
	if ok, reason := scopeAllowsIssue(repo.Config.Ingestion.JIRA, issue); !ok {
		result.Status = StatusDropped
		result.Reason = reason
		return result, nil
	}

	existing, err := findTicketByJiraKey(repo.Path, issue.Key)
	if err != nil {
		return result, err
	}
	if existing == "" {
		rel, err := createMirroredTicket(repo.Path, issue)
		if err != nil {
			return result, err
		}
		result.Status = StatusCreated
		result.Created = true
		result.TicketPath = rel
		return result, nil
	}

	rel, err := reconcileMirroredTicket(repo.Path, existing, issue)
	if err != nil {
		return result, err
	}
	result.Status = StatusReconciled
	result.TicketPath = rel
	return result, nil
}

func resolveRepository(repos []Repository, jiraKey, project string) (Repository, MirrorResult) {
	project = strings.TrimSpace(project)
	if project == "" {
		project = projectFromKey(jiraKey)
	}
	base := MirrorResult{JiraKey: strings.TrimSpace(jiraKey), LLMJobsEnqueued: 0}
	var enabled int
	var projectMappings int
	var matches []Repository
	for _, repo := range repos {
		if !repo.Config.JIRAEnabled() {
			continue
		}
		enabled++
		for _, mapping := range repo.Config.Ingestion.JIRA.ProjectRepoMap {
			mappedProject := strings.TrimSpace(mapping.Project)
			mappedRepo := strings.TrimSpace(mapping.Repo)
			if mappedProject == "" || mappedRepo == "" {
				continue
			}
			if !sameProject(mappedProject, project) {
				continue
			}
			projectMappings++
			if repoMatches(mappedRepo, repo) {
				matches = append(matches, repo)
			}
		}
	}
	if enabled == 0 {
		base.Status = StatusDisabled
		base.Reason = "jira_ingestion_disabled"
		return Repository{}, base
	}
	if project == "" || projectMappings == 0 {
		base.Status = StatusDropped
		base.Reason = "unmapped_project"
		return Repository{}, base
	}
	if len(matches) == 0 {
		base.Status = StatusDropped
		base.Reason = "unmapped_repo"
		return Repository{}, base
	}
	if len(matches) > 1 {
		base.Status = StatusDropped
		base.Reason = "ambiguous_project_repo_map"
		return Repository{}, base
	}
	return matches[0], MirrorResult{}
}

func sameProject(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func repoMatches(configured string, repo Repository) bool {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return false
	}
	candidates := []string{
		repo.ID,
		repo.Path,
		filepath.Clean(repo.Path),
		filepath.Base(repo.Path),
		repo.Remote,
	}
	if abs, err := filepath.Abs(repo.Path); err == nil {
		candidates = append(candidates, abs, filepath.Clean(abs))
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == configured {
			return true
		}
	}
	return false
}

func findTicketByJiraKey(repoRoot, jiraKey string) (string, error) {
	var matches []string
	for _, status := range []string{"backlog", "in-progress", "in-review", "done"} {
		dir := filepath.Join(repoRoot, "docs", "tickets", status)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("jira: read ticket dir %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || entry.Name() == "README.md" {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			frontmatter, err := readFrontmatter(path)
			if err != nil {
				return "", err
			}
			if strings.Trim(frontmatter["jira_key"], `"'`) == jiraKey {
				rel, _ := filepath.Rel(repoRoot, path)
				matches = append(matches, filepath.ToSlash(rel))
			}
		}
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return "", fmt.Errorf("jira: jira_key %s is mirrored by multiple tickets: %s", jiraKey, strings.Join(matches, ", "))
	}
	if len(matches) == 0 {
		return "", nil
	}
	return matches[0], nil
}

func createMirroredTicket(repoRoot string, issue Issue) (string, error) {
	id, err := nextTicketID(repoRoot)
	if err != nil {
		return "", err
	}
	title := issue.Summary
	if title == "" {
		title = issue.Key
	}
	filename := fmt.Sprintf("%s-%s.md", id, slugify(title))
	rel := filepath.ToSlash(filepath.Join("docs", "tickets", "backlog", filename))
	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("jira: create ticket dir: %w", err)
	}
	var b strings.Builder
	b.WriteString("---\n")
	for _, line := range initialFrontmatter(id, title, issue) {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s: %s\n\n", id, title)
	b.WriteString(renderJiraSourceSection(issue))
	b.WriteString("\n\n## Harness Scope\n\n")
	b.WriteString(scopedMarker)
	b.WriteString("\nNot scoped yet. `cto-weekly` owns scoping after board selection.\n\n")
	b.WriteString("## Agent Notes\n\n")
	if err := os.WriteFile(abs, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("jira: write ticket %s: %w", abs, err)
	}
	return rel, nil
}

func reconcileMirroredTicket(repoRoot, rel string, issue Issue) (string, error) {
	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("jira: read mirrored ticket %s: %w", abs, err)
	}
	updated, err := updateFrontmatter(data, jiraFrontmatter(issue))
	if err != nil {
		return "", err
	}
	updated = updateJiraSection(updated, issue)
	if bytes.Equal(data, updated) {
		return rel, nil
	}
	if err := os.WriteFile(abs, updated, 0o644); err != nil {
		return "", fmt.Errorf("jira: update mirrored ticket %s: %w", abs, err)
	}
	return rel, nil
}

func nextTicketID(repoRoot string) (string, error) {
	next := 1
	root := filepath.Join(repoRoot, "docs", "tickets")
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		match := ticketIDPattern.FindStringSubmatch(d.Name())
		if len(match) != 2 {
			return nil
		}
		n, _ := strconv.Atoi(match[1])
		if n >= next {
			next = n + 1
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("jira: scan ticket IDs: %w", err)
	}
	return fmt.Sprintf("T-%03d", next), nil
}

func initialFrontmatter(id, title string, issue Issue) []string {
	lines := []string{
		"id: " + id,
		"title: " + yamlScalar(title),
		"priority: " + yamlScalar(issue.Priority),
		"complexity: medium",
		"work_type: feature",
		"bdd_scenarios: []",
		"end_to_end_evidence: required",
		"evidence_links: []",
		"verified_by: " + yamlScalar("TBD"),
		"owner: " + yamlScalar("cto-weekly"),
		"last_attempt: " + yamlScalar("TBD"),
		"blocker: " + yamlScalar("none"),
		"blocked_by: " + yamlInlineList(issue.BlockedBy),
		"trace_id: " + yamlScalar("TBD"),
		"next_action: " + yamlScalar("Await cto-weekly scoping after board-driven selection."),
	}
	for _, key := range []string{"jira_key", "jira_updated", "jira_created", "sprint", "sprint_active", "rank", "jira_status", "epic"} {
		lines = append(lines, key+": "+jiraFrontmatter(issue)[key])
	}
	lines = append(lines,
		"source: jira",
		"created: "+time.Now().Format("2006-01-02"),
		"depends_on: []",
	)
	return lines
}

func jiraFrontmatter(issue Issue) map[string]string {
	issue = issue.Sanitized()
	return map[string]string{
		"priority":      yamlScalar(issue.Priority),
		"blocked_by":    yamlInlineList(issue.BlockedBy),
		"jira_key":      yamlScalar(issue.Key),
		"jira_updated":  yamlScalar(issue.Updated),
		"jira_created":  yamlScalar(issue.Created),
		"sprint":        yamlScalar(issue.Sprint),
		"sprint_active": strconv.FormatBool(issue.SprintActive),
		"rank":          yamlScalar(issue.Rank),
		"jira_status":   yamlScalar(issue.Status),
		"epic":          yamlScalar(issue.Epic),
	}
}

func renderJiraSourceSection(issue Issue) string {
	issue = issue.Sanitized()
	var b strings.Builder
	b.WriteString("## JIRA Source\n\n")
	b.WriteString(jiraOwnedStartMarker)
	b.WriteString("\n")
	fmt.Fprintf(&b, "- JIRA key: %s\n", issue.Key)
	fmt.Fprintf(&b, "- Project: %s\n", issue.Project)
	fmt.Fprintf(&b, "- Summary: %s\n", valueOrUnknown(issue.Summary))
	fmt.Fprintf(&b, "- Status: %s\n", valueOrUnknown(issue.Status))
	fmt.Fprintf(&b, "- Priority: %s\n", valueOrUnknown(issue.Priority))
	fmt.Fprintf(&b, "- Sprint: %s\n", valueOrUnknown(issue.Sprint))
	fmt.Fprintf(&b, "- Sprint active: %t\n", issue.SprintActive)
	fmt.Fprintf(&b, "- Rank: %s\n", valueOrUnknown(issue.Rank))
	fmt.Fprintf(&b, "- Epic: %s\n", valueOrUnknown(issue.Epic))
	fmt.Fprintf(&b, "- Created: %s\n", valueOrUnknown(issue.Created))
	fmt.Fprintf(&b, "- Updated: %s\n", valueOrUnknown(issue.Updated))
	if issue.URL != "" {
		fmt.Fprintf(&b, "- Source URL: %s\n", issue.URL)
	}
	if len(issue.BlockedBy) > 0 {
		fmt.Fprintf(&b, "- Blocked by: %s\n", strings.Join(issue.BlockedBy, ", "))
	}
	if len(issue.Labels) > 0 {
		fmt.Fprintf(&b, "- Labels: %s\n", strings.Join(issue.Labels, ", "))
	}
	b.WriteString("\n### Requirements\n\n")
	if issue.Description == "" {
		b.WriteString("_No JIRA description supplied._\n")
	} else {
		b.WriteString(issue.Description)
		if !strings.HasSuffix(issue.Description, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString(jiraOwnedEndMarker)
	return b.String()
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func scopeAllowsIssue(cfg integrations.JIRAConfig, issue Issue) (bool, string) {
	if len(cfg.Scope.AllowedWorkspaces) > 0 && !workspaceAllowed(cfg.Scope.AllowedWorkspaces, cfg.BaseURL, issue) {
		return false, "scope_workspace_mismatch"
	}
	if len(cfg.Scope.RequiredLabels) > 0 && !hasRequiredLabels(issue.Labels, cfg.Scope.RequiredLabels) {
		return false, "scope_required_label_missing"
	}
	return true, ""
}

func workspaceAllowed(allowed []string, baseURL string, issue Issue) bool {
	for _, workspace := range allowed {
		if workspaceMatchesIssue(workspace, baseURL, issue) {
			return true
		}
	}
	return false
}

func workspaceMatchesIssue(workspace, baseURL string, issue Issue) bool {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return false
	}
	if strings.EqualFold(workspace, issue.Project) {
		return true
	}
	allowedURL, allowedURLParsed := parseMaybeURL(workspace)
	if !allowedURLParsed {
		return strings.EqualFold(workspace, hostFromURL(issue.URL)) ||
			strings.EqualFold(workspace, hostFromURL(baseURL))
	}
	issueHost := hostFromURL(issue.URL)
	if isAtlassianAPIGatewayHost(issueHost) && hostFromURL(baseURL) != "" {
		issueHost = hostFromURL(baseURL)
	}
	if issueHost == "" {
		issueHost = hostFromURL(baseURL)
	}
	if allowedURL.Host != "" && !strings.EqualFold(allowedURL.Host, issueHost) {
		return false
	}
	if project := projectFromWorkspacePath(allowedURL.Path); project != "" {
		return strings.EqualFold(project, issue.Project)
	}
	return true
}

func parseMaybeURL(value string) (*url.URL, bool) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, false
	}
	return parsed, true
}

func hostFromURL(value string) string {
	parsed, ok := parseMaybeURL(value)
	if !ok {
		return ""
	}
	return parsed.Host
}

func isAtlassianAPIGatewayHost(host string) bool {
	return strings.EqualFold(strings.TrimSpace(host), "api.atlassian.com")
}

func projectFromWorkspacePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func hasRequiredLabels(labels, required []string) bool {
	present := map[string]bool{}
	for _, label := range labels {
		present[strings.ToLower(strings.TrimSpace(label))] = true
	}
	for _, label := range required {
		if !present[strings.ToLower(strings.TrimSpace(label))] {
			return false
		}
	}
	return true
}

func updateJiraSection(data []byte, issue Issue) []byte {
	text := string(data)
	start := strings.Index(text, jiraOwnedStartMarker)
	end := strings.Index(text, jiraOwnedEndMarker)
	if start >= 0 && end > start {
		replaceStart := start
		prefix := text[:start]
		if heading := strings.LastIndex(prefix, "\n## JIRA Source"); heading >= 0 {
			replaceStart = heading + 1
		}
		replaceEnd := end + len(jiraOwnedEndMarker)
		return []byte(text[:replaceStart] + renderJiraSourceSection(issue) + text[replaceEnd:])
	}
	insert := renderJiraSourceSection(issue) + "\n\n"
	firstHeadingEnd := strings.Index(text, "\n\n")
	if firstHeadingEnd < 0 {
		return []byte(insert + text)
	}
	return []byte(text[:firstHeadingEnd+2] + insert + text[firstHeadingEnd+2:])
}

func readFrontmatter(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("jira: read frontmatter %s: %w", path, err)
	}
	front, _, ok := splitFrontmatter(data)
	if !ok {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(front), "\n") {
		if strings.HasPrefix(line, " ") || strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

func updateFrontmatter(data []byte, fields map[string]string) ([]byte, error) {
	front, body, ok := splitFrontmatter(data)
	if !ok {
		return data, fmt.Errorf("jira: mirrored ticket is missing YAML frontmatter")
	}
	lines := strings.Split(string(front), "\n")
	remaining := map[string]string{}
	for key, value := range fields {
		remaining[key] = value
	}
	for i, line := range lines {
		if strings.HasPrefix(line, " ") {
			continue
		}
		key, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value, owned := remaining[key]
		if !owned {
			continue
		}
		lines[i] = key + ": " + value
		delete(remaining, key)
	}
	for _, key := range jiraFrontmatterOrder {
		value, ok := remaining[key]
		if !ok {
			continue
		}
		lines = append(lines, key+": "+value)
		delete(remaining, key)
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(strings.TrimRight(strings.Join(lines, "\n"), "\n"))
	b.WriteString("\n---")
	b.Write(body)
	return []byte(b.String()), nil
}

func splitFrontmatter(data []byte) ([]byte, []byte, bool) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, false
	}
	rest := data[len("---\n"):]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx < 0 {
		return nil, nil, false
	}
	front := rest[:idx]
	body := rest[idx+len("\n---"):]
	return front, body, true
}

func yamlScalar(value string) string {
	return strconv.Quote(redactSensitiveText(strings.TrimSpace(value)))
}

func yamlInlineList(values []string) string {
	values = cleanStringList(values)
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, yamlScalar(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func enabledJIRARepos(repos []Repository) []Repository {
	var out []Repository
	for _, repo := range repos {
		if repo.Config.JIRAEnabled() {
			out = append(out, repo)
		}
	}
	return out
}
