/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
*/
package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Bump string

const (
	BumpAuto  Bump = "auto"
	BumpMajor Bump = "major"
	BumpMinor Bump = "minor"
	BumpPatch Bump = "patch"
)

var (
	ErrNoChanges = errors.New("release: no commits found since the last release marker")
	markerRE     = regexp.MustCompile(`<!-- mars-harness-release: version=([^ ]+) commit=([0-9a-fA-F]+) -->`)
	subjectRE    = regexp.MustCompile(`^([a-zA-Z0-9_-]+)(\([^)]+\))?(!)?:\s+(.+)$`)
	narrativeRE  = regexp.MustCompile(`(?i)^\s*(impact|why|what|what changed)\s*:\s*(.*)$`)
)

type Config struct {
	RepoRoot string
	Bump     Bump
	DryRun   bool
	Now      time.Time
}

type Commit struct {
	Hash     string
	Short    string
	Subject  string
	Body     string
	Type     string
	Scope    string
	Message  string
	Breaking bool
}

type releaseNarrativeProfile struct {
	Impact string
	Why    string
	What   string
}

type Result struct {
	PreviousVersion SemVer
	NextVersion     SemVer
	Bump            Bump
	BaseRef         string
	Head            string
	Entry           string
	Commits         []Commit
	UpdatedFiles    []string
}

func Prepare(ctx context.Context, cfg Config) (Result, error) {
	repoRoot := strings.TrimSpace(cfg.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return Result{}, fmt.Errorf("release: resolve repo path: %w", err)
	}
	if cfg.Now.IsZero() {
		cfg.Now = time.Now()
	}
	if cfg.Bump == "" {
		cfg.Bump = BumpAuto
	}

	previous, err := readVersion(absRepo)
	if err != nil {
		return Result{}, err
	}
	baseRef := findBaseRef(ctx, absRepo, previous)
	commits, err := readCommits(ctx, absRepo, baseRef)
	if err != nil {
		return Result{}, err
	}
	if len(commits) == 0 {
		return Result{}, ErrNoChanges
	}
	bump := cfg.Bump
	if bump == BumpAuto {
		bump = inferBump(commits)
	}
	if bump != BumpMajor && bump != BumpMinor && bump != BumpPatch {
		return Result{}, fmt.Errorf("release: invalid bump %q, expected auto, major, minor, or patch", cfg.Bump)
	}

	next := previous.Bump(bump)
	head, err := git(ctx, absRepo, "rev-parse", "--short=12", "HEAD")
	if err != nil {
		return Result{}, err
	}
	entry := renderEntry(absRepo, next, cfg.Now, head, commits)
	result := Result{
		PreviousVersion: previous,
		NextVersion:     next,
		Bump:            bump,
		BaseRef:         baseRef,
		Head:            strings.TrimSpace(head),
		Entry:           entry,
		Commits:         commits,
	}
	if cfg.DryRun {
		return result, nil
	}

	files, err := writeReleaseFiles(absRepo, next, entry)
	if err != nil {
		return Result{}, err
	}
	result.UpdatedFiles = files
	return result, nil
}

func readVersion(repoRoot string) (SemVer, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if os.IsNotExist(err) {
		return SemVer{}, nil
	}
	if err != nil {
		return SemVer{}, fmt.Errorf("release: read VERSION: %w", err)
	}
	return ParseSemVer(string(data))
}

func findBaseRef(ctx context.Context, repoRoot string, current SemVer) string {
	if marker := latestChangelogMarker(repoRoot); marker != "" {
		if gitOK(ctx, repoRoot, "merge-base", "--is-ancestor", marker, "HEAD") {
			return marker
		}
	}
	currentTag := "v" + current.String()
	if current != (SemVer{}) && gitOK(ctx, repoRoot, "rev-parse", "--verify", "--quiet", currentTag) {
		return currentTag
	}
	if tag, err := git(ctx, repoRoot, "describe", "--tags", "--match", "v[0-9]*", "--abbrev=0"); err == nil {
		return strings.TrimSpace(tag)
	}
	return ""
}

func latestChangelogMarker(repoRoot string) string {
	data, err := os.ReadFile(filepath.Join(repoRoot, "CHANGELOG.md"))
	if err != nil {
		return ""
	}
	match := markerRE.FindSubmatch(data)
	if match == nil {
		return ""
	}
	return string(match[2])
}

func readCommits(ctx context.Context, repoRoot, baseRef string) ([]Commit, error) {
	args := []string{"log", "--reverse", "--format=%H%x1f%h%x1f%s%x1f%b%x1e"}
	if baseRef != "" {
		args = append(args, baseRef+"..HEAD")
	}
	out, err := git(ctx, repoRoot, args...)
	if err != nil {
		return nil, err
	}
	return parseGitLogCommits(out), nil
}

func parseSubject(subject, body string) (typ, scope, message string, breaking bool) {
	match := subjectRE.FindStringSubmatch(subject)
	if match == nil {
		return "other", "", subject, strings.Contains(body, "BREAKING CHANGE")
	}
	typ = strings.ToLower(match[1])
	scope = strings.Trim(match[2], "()")
	message = match[4]
	breaking = match[3] == "!" || strings.Contains(body, "BREAKING CHANGE")
	return typ, scope, message, breaking
}

func inferBump(commits []Commit) Bump {
	bump := BumpPatch
	for _, commit := range commits {
		if commit.Breaking {
			return BumpMajor
		}
		if commit.Type == "feat" {
			bump = BumpMinor
		}
	}
	return bump
}

func renderEntry(repoRoot string, version SemVer, now time.Time, head string, commits []Commit) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "## [%s] - %s\n", version.String(), now.Format("2006-01-02"))
	fmt.Fprintf(&buf, "<!-- mars-harness-release: version=%s commit=%s -->\n\n", version.String(), strings.TrimSpace(head))

	if summary := renderReleaseNarrative(commits); summary != "" {
		buf.WriteString(summary)
		buf.WriteString("\n\n")
	}

	groups := groupCommits(commits)
	order := []string{"Breaking Changes", "Features", "Fixes", "Documentation", "Maintenance", "Tests", "Other"}
	for _, group := range order {
		items := groups[group]
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].Short < items[j].Short })
		fmt.Fprintf(&buf, "### %s\n", group)
		for _, commit := range items {
			scope := ""
			if commit.Scope != "" {
				scope = fmt.Sprintf("**%s:** ", commit.Scope)
			}
			fmt.Fprintf(&buf, "- %s%s (%s)\n", scope, sentence(commit.Message), commit.Short)
		}
		buf.WriteString("\n")
	}
	if evidence := renderDeliveryEvidence(repoRoot, commits); evidence != "" {
		buf.WriteString(evidence)
		buf.WriteString("\n")
	}
	return strings.TrimRight(buf.String(), "\n") + "\n"
}

func renderReleaseNarrative(commits []Commit) string {
	if len(commits) == 0 {
		return ""
	}

	groups := groupCommits(commits)
	groupOrder := []string{"Breaking Changes", "Features", "Fixes", "Documentation", "Maintenance", "Tests", "Other"}
	ordered := make([]Commit, 0, len(commits))
	for _, group := range groupOrder {
		items := groups[group]
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].Short < items[j].Short })
		ordered = append(ordered, items...)
	}
	if len(ordered) == 0 {
		return ""
	}

	var buf bytes.Buffer
	writeNarrativeSection(&buf, "Impact", ordered, releaseImpactLine)
	writeNarrativeSection(&buf, "Why", ordered, releaseWhyLine)
	writeNarrativeSection(&buf, "What Changed", ordered, releaseWhatLine)
	return strings.TrimRight(buf.String(), "\n")
}

func writeNarrativeSection(buf *bytes.Buffer, title string, commits []Commit, lineFor func(Commit) string) {
	if len(commits) == 0 {
		return
	}
	if buf.Len() > 0 {
		buf.WriteString("\n")
	}
	fmt.Fprintf(buf, "### %s\n", title)
	for _, commit := range commits {
		fmt.Fprintf(buf, "- %s\n", lineFor(commit))
	}
}

func releaseImpactLine(commit Commit) string {
	if value := commitNarrativeField(commit, "impact"); value != "" {
		return scopedNarrative(commit, value)
	}
	if profile, ok := releaseProfile(commit); ok {
		return scopedNarrative(commit, profile.Impact)
	}
	change := releaseChangePhrase(commit)
	if commit.Breaking {
		return scopedNarrative(commit, "Operators may need to account for compatibility-changing work: "+change+".")
	}
	switch commit.Type {
	case "feat":
		return scopedNarrative(commit, "Operators gain new capability: "+change+".")
	case "fix", "perf":
		return scopedNarrative(commit, "Operators see improved reliability because "+change+".")
	case "docs":
		return scopedNarrative(commit, "Operators and future agents get clearer guidance because "+change+".")
	case "test":
		return scopedNarrative(commit, "The release carries stronger evidence because "+change+".")
	case "chore", "build", "ci", "refactor", "style":
		return scopedNarrative(commit, "Maintainers get a healthier project surface because "+change+".")
	default:
		return scopedNarrative(commit, "The release includes visible project movement: "+change+".")
	}
}

func releaseWhyLine(commit Commit) string {
	if value := commitNarrativeField(commit, "why"); value != "" {
		return scopedNarrative(commit, value)
	}
	if profile, ok := releaseProfile(commit); ok {
		return scopedNarrative(commit, profile.Why)
	}
	change := releaseChangePhrase(commit)
	switch {
	case commit.Breaking:
		return scopedNarrative(commit, "This matters because compatibility-changing work must be called out before operators upgrade.")
	case commit.Type == "feat":
		return scopedNarrative(commit, "This matters because "+change+" was missing from the shipped capability set.")
	case commit.Type == "fix" || commit.Type == "perf":
		return scopedNarrative(commit, "This matters because "+change+" closes a failure mode or degraded path.")
	case commit.Type == "docs":
		return scopedNarrative(commit, "This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.")
	case commit.Type == "test":
		return scopedNarrative(commit, "This matters because the project needs durable evidence that the behavior keeps working.")
	case commit.Type == "chore" || commit.Type == "build" || commit.Type == "ci" || commit.Type == "refactor" || commit.Type == "style":
		return scopedNarrative(commit, "This matters because project health work keeps future delivery predictable.")
	default:
		return scopedNarrative(commit, "This matters because the release should explain why "+change+" belongs in the shipped state.")
	}
}

func releaseWhatLine(commit Commit) string {
	if value := commitNarrativeField(commit, "what"); value != "" {
		return scopedNarrative(commit, fmt.Sprintf("%s (%s).", strings.TrimSuffix(ensureSentence(value), "."), commit.Short))
	}
	if profile, ok := releaseProfile(commit); ok {
		return scopedNarrative(commit, fmt.Sprintf("%s (%s).", strings.TrimSuffix(ensureSentence(profile.What), "."), commit.Short))
	}
	return scopedNarrative(commit, fmt.Sprintf("Changed %s (%s).", releaseChangePhrase(commit), commit.Short))
}

func releaseProfile(commit Commit) (releaseNarrativeProfile, bool) {
	text := normalizedReleaseText(commit)
	scope := strings.ToLower(strings.TrimSpace(commit.Scope))

	switch {
	case isStructuredDispatchChange(scope, text):
		return releaseNarrativeProfile{
			Impact: "Operators and agents get a more reliable delivery loop because handoff and feedback now travel as first-class runtime data through Orchestrator dispatch.",
			Why:    "This matters because operating-model shifts lose value when the next owner, expected correction, or supporting evidence only exists in free-form transcript text.",
			What:   "Dispatch triggers now carry the source disposition, including status, next need, ticket ID, reason, evidence links, trace ID, handoff, and feedback, so Orchestrator can validate one target owner before enqueueing follow-up work.",
		}, true
	case isPersonaOperatingModelChange(scope, text):
		return releaseNarrativeProfile{
			Impact: "Agents get clearer role ownership because foundation personas now spell out boundaries, feedback shape, stop conditions, and Orchestrator handoff expectations.",
			Why:    "This matters because autonomous routing depends on explicit ownership contracts; prompt prose alone leaves downstream roles guessing who should act next.",
			What:   "Canonical persona definitions now render checked role manuals and prompt Personal Guides so generated guidance, reviews, and dispatch handoffs share the same source of truth.",
		}, true
	case isDocumentationSyncChange(scope, text):
		return releaseNarrativeProfile{
			Impact: "Operators and agents get stronger no-stale-docs enforcement because documentation sync is described and validated as part of the delivery workflow.",
			Why:    "This matters because behavior changes become risky when code, BDD contracts, design docs, generated target guidance, and release notes drift apart.",
			What:   "The release documentation path now ties changed source files to associated docs, docsync evidence, and generated target doctrine instead of treating docs as an after-the-fact checklist.",
		}, true
	case isCLIToolSkillSyncChange(scope, text):
		return releaseNarrativeProfile{
			Impact: "Operators and agents get a more trustworthy CLI surface because command behavior, mirrored tool docs, repo shortcuts, generated target guidance, and skills stay synchronized.",
			Why:    "This matters because CLI changes can otherwise ship while agents continue using stale tool contracts or workflow instructions.",
			What:   "The release workflow now treats CLI tool and skill synchronization as required release evidence whenever command flags, outputs, or workflows change.",
		}, true
	case isOperatingModelChange(scope, text):
		return releaseNarrativeProfile{
			Impact: "Operators and future agents get clearer delivery behavior because an operating-model rule, boundary, or workflow contract is now explicit in repo-owned guidance.",
			Why:    "This matters because autonomous work needs durable routing, evidence, and ownership rules rather than relying on chat memory or implicit handoffs.",
			What:   "The operating-model guidance was updated so adjacent docs, roles, tools, evidence paths, and generated target defaults describe the new workflow consistently.",
		}, true
	default:
		return releaseNarrativeProfile{}, false
	}
}

func normalizedReleaseText(commit Commit) string {
	parts := []string{commit.Type, commit.Scope, commit.Message, commit.Subject, commit.Body}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func isStructuredDispatchChange(scope, text string) bool {
	return (scope == "orchestration" || strings.Contains(text, "orchestrator") || strings.Contains(text, "dispatch")) &&
		((strings.Contains(text, "structured") && strings.Contains(text, "handoff")) ||
			strings.Contains(text, "source_disposition") ||
			(strings.Contains(text, "handoff") && strings.Contains(text, "feedback") && strings.Contains(text, "dispatch")))
}

func isPersonaOperatingModelChange(scope, text string) bool {
	return scope == "personas" ||
		strings.Contains(text, "persona manual") ||
		strings.Contains(text, "personal guide") ||
		strings.Contains(text, "foundation agent manual")
}

func isDocumentationSyncChange(scope, text string) bool {
	return scope == "docsync" ||
		strings.Contains(text, "documentation sync") ||
		strings.Contains(text, "docsync") ||
		strings.Contains(text, "marsdocsync") ||
		strings.Contains(text, "no stale documentation")
}

func isCLIToolSkillSyncChange(scope, text string) bool {
	return (scope == "cli" || strings.Contains(text, "mars_harness_cli")) &&
		((strings.Contains(text, "tool") && strings.Contains(text, "skill") && strings.Contains(text, "sync")) ||
			strings.Contains(text, "repo-shortcut"))
}

func isOperatingModelChange(scope, text string) bool {
	return scope == "operating-model" ||
		strings.Contains(text, "operating model") ||
		strings.Contains(text, "bdd-led") ||
		strings.Contains(text, "business logic is first-class bdd") ||
		strings.Contains(text, "symbiotic workflow")
}

func releaseChangePhrase(commit Commit) string {
	message := strings.TrimSpace(commit.Message)
	if message == "" {
		message = strings.TrimSpace(commit.Subject)
	}
	message = strings.TrimSuffix(message, ".")
	return lowerFirst(message)
}

func scopedNarrative(commit Commit, value string) string {
	value = ensureSentence(value)
	if commit.Scope == "" {
		return value
	}
	return fmt.Sprintf("**%s:** %s", commit.Scope, value)
}

func commitNarrativeField(commit Commit, key string) string {
	fields := parseNarrativeFields(commit.Body)
	return fields[key]
}

func parseNarrativeFields(body string) map[string]string {
	fields := make(map[string]string)
	var current string
	for _, line := range strings.Split(body, "\n") {
		if match := narrativeRE.FindStringSubmatch(line); match != nil {
			current = canonicalNarrativeKey(match[1])
			fields[current] = strings.TrimSpace(match[2])
			continue
		}
		if current == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if fields[current] != "" {
			fields[current] += " "
		}
		fields[current] += trimmed
	}
	return fields
}

func canonicalNarrativeKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "what changed" {
		return "what"
	}
	return key
}

func ensureSentence(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Update."
	}
	last := value[len(value)-1]
	if last == '.' || last == '!' || last == '?' {
		return sentence(value)
	}
	return sentence(value) + "."
}

func lowerFirst(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func renderDeliveryEvidence(repoRoot string, commits []Commit) string {
	ids := ticketIDsFromCommits(commits)
	if len(ids) == 0 {
		return ""
	}
	var shipped []string
	var enablers []string
	for _, id := range ids {
		ticket, ok := readDoneTicketSummary(repoRoot, id)
		if !ok {
			continue
		}
		switch ticket.WorkType {
		case "feature":
			if len(ticket.BDDScenarios) > 0 {
				shipped = append(shipped, fmt.Sprintf("%s: %s", id, strings.Join(ticket.BDDScenarios, ", ")))
			}
		case "enabler", "research", "docs", "intervention-debt":
			enablers = append(enablers, fmt.Sprintf("%s: %s", id, ticket.Title))
		}
	}
	if len(shipped) == 0 && len(enablers) == 0 {
		return ""
	}
	sort.Strings(shipped)
	sort.Strings(enablers)
	var buf bytes.Buffer
	buf.WriteString("### Delivery Evidence\n")
	for _, item := range shipped {
		fmt.Fprintf(&buf, "- Shipped feature scenarios: %s\n", item)
	}
	for _, item := range enablers {
		fmt.Fprintf(&buf, "- Enabler work: %s\n", item)
	}
	return buf.String()
}

var ticketIDRE = regexp.MustCompile(`\b(?:MH|T)-\d{3}\b`)

func ticketIDsFromCommits(commits []Commit) []string {
	set := make(map[string]bool)
	for _, commit := range commits {
		for _, id := range ticketIDRE.FindAllString(commit.Subject+"\n"+commit.Body, -1) {
			set[id] = true
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

type doneTicketSummary struct {
	Title        string
	WorkType     string
	BDDScenarios []string
}

func readDoneTicketSummary(repoRoot, id string) (doneTicketSummary, bool) {
	matches, err := filepath.Glob(filepath.Join(repoRoot, "docs", "tickets", "done", id+"-*.md"))
	if err != nil || len(matches) == 0 {
		return doneTicketSummary{}, false
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return doneTicketSummary{}, false
	}
	fields := parseReleaseFrontmatter(string(data))
	return doneTicketSummary{
		Title:        strings.Trim(fields["title"], `"'`),
		WorkType:     strings.Trim(strings.ToLower(fields["work_type"]), `"'`),
		BDDScenarios: parseInlineList(fields["bdd_scenarios"]),
	}, true
}

func parseReleaseFrontmatter(text string) map[string]string {
	fields := make(map[string]string)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return fields
}

func parseInlineList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "[]")
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func groupCommits(commits []Commit) map[string][]Commit {
	groups := map[string][]Commit{}
	for _, commit := range commits {
		group := "Other"
		switch commit.Type {
		case "feat":
			group = "Features"
		case "fix", "perf":
			group = "Fixes"
		case "docs":
			group = "Documentation"
		case "test":
			group = "Tests"
		case "chore", "build", "ci", "refactor", "style":
			group = "Maintenance"
		}
		if commit.Breaking {
			group = "Breaking Changes"
		}
		groups[group] = append(groups[group], commit)
	}
	return groups
}

func sentence(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Update"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func writeReleaseFiles(repoRoot string, next SemVer, entry string) ([]string, error) {
	var updated []string
	versionPath := filepath.Join(repoRoot, "VERSION")
	if err := os.WriteFile(versionPath, []byte(next.String()+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("release: write VERSION: %w", err)
	}
	updated = append(updated, "VERSION")

	if err := updateBuildInfoVersion(repoRoot, next); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "internal", "buildinfo", "version.go")); err == nil {
		updated = append(updated, "internal/buildinfo/version.go")
	}

	changelogPath := filepath.Join(repoRoot, "CHANGELOG.md")
	existing, err := os.ReadFile(changelogPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("release: read CHANGELOG.md: %w", err)
	}
	nextChangelog := insertChangelogEntry(string(existing), entry)
	if err := os.WriteFile(changelogPath, []byte(nextChangelog), 0o644); err != nil {
		return nil, fmt.Errorf("release: write CHANGELOG.md: %w", err)
	}
	updated = append(updated, "CHANGELOG.md")
	return updated, nil
}

func updateBuildInfoVersion(repoRoot string, next SemVer) error {
	path := filepath.Join(repoRoot, "internal", "buildinfo", "version.go")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("release: read internal/buildinfo/version.go: %w", err)
	}
	old := regexp.MustCompile(`DefaultVersion = "[0-9]+\.[0-9]+\.[0-9]+"`)
	replacement := fmt.Sprintf(`DefaultVersion = "%s"`, next.String())
	content := string(data)
	if !old.MatchString(content) {
		return fmt.Errorf("release: internal/buildinfo/version.go does not contain a DefaultVersion semantic version")
	}
	content = old.ReplaceAllString(content, replacement)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("release: write internal/buildinfo/version.go: %w", err)
	}
	return nil
}

func insertChangelogEntry(existing, entry string) string {
	entry = strings.TrimRight(entry, "\n")
	if strings.TrimSpace(existing) == "" {
		return changelogHeader() + "\n\n" + entry + "\n"
	}
	if !strings.HasPrefix(existing, "# Changelog") {
		existing = changelogHeader() + "\n" + existing
	}
	lines := strings.Split(existing, "\n")
	insertAt := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(line, "## [") {
			insertAt = i
			break
		}
	}
	prefix := strings.TrimRight(strings.Join(lines[:insertAt], "\n"), "\n")
	suffix := strings.TrimLeft(strings.Join(lines[insertAt:], "\n"), "\n")
	if suffix == "" {
		return prefix + "\n\n" + entry + "\n"
	}
	return prefix + "\n\n" + entry + "\n\n" + suffix
}

func changelogHeader() string {
	return "# Changelog\n\nPatch notes are generated with `mars-harness release notes` from semantic commits on `main`."
}

func gitOK(ctx context.Context, repoRoot string, args ...string) bool {
	_, err := git(ctx, repoRoot, args...)
	return err == nil
}

func git(ctx context.Context, repoRoot string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("release: git %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
