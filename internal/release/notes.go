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
		return marker
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
	out = strings.TrimRight(out, "\x1e\n")
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	records := strings.Split(out, "\x1e")
	commits := make([]Commit, 0, len(records))
	for _, record := range records {
		record = strings.Trim(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x1f", 4)
		if len(fields) < 4 {
			continue
		}
		commit := Commit{
			Hash:    strings.TrimSpace(fields[0]),
			Short:   strings.TrimSpace(fields[1]),
			Subject: strings.TrimSpace(fields[2]),
			Body:    strings.TrimSpace(fields[3]),
		}
		commit.Type, commit.Scope, commit.Message, commit.Breaking = parseSubject(commit.Subject, commit.Body)
		if commit.Type == "release" {
			continue
		}
		commits = append(commits, commit)
	}
	return commits, nil
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
