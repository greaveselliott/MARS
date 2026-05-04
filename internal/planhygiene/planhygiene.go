/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-improvement.md
- docs/features/F-001-delivery-operating-model.md
*/
package planhygiene

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Issue describes one actionable exec-plan hygiene problem.
type Issue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Fix     string `json:"fix"`
}

// Report summarizes active-plan hygiene for a repository.
type Report struct {
	Issues []Issue `json:"issues"`
}

// OK returns true when no active-plan hygiene issues were found.
func (r Report) OK() bool {
	return len(r.Issues) == 0
}

// Summary returns a short operator-facing report.
func (r Report) Summary() string {
	if r.OK() {
		return "active-plan hygiene is clean"
	}
	if len(r.Issues) == 1 {
		return "active-plan hygiene issue: " + r.Issues[0].Message
	}
	return fmt.Sprintf("active-plan hygiene found %d issues: %s", len(r.Issues), r.Issues[0].Message)
}

// Remediation returns the first concrete remediation command or instruction.
func (r Report) Remediation() string {
	for _, issue := range r.Issues {
		if strings.TrimSpace(issue.Fix) != "" {
			return issue.Fix
		}
	}
	return "repair docs/exec-plans, then run 'go test ./internal/docsconsistency/...'"
}

// Config controls active-plan hygiene checks.
type Config struct {
	RepoPath             string
	Now                  time.Time
	StaleVerificationAge time.Duration
}

// CheckRepo checks active-plan hygiene without mutating the repository.
func CheckRepo(repoPath string) (Report, error) {
	return Check(Config{RepoPath: repoPath})
}

// Check checks active-plan hygiene without mutating the repository.
func Check(cfg Config) (Report, error) {
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return Report{}, fmt.Errorf("active-plan hygiene: repo path is empty")
	}
	if cfg.Now.IsZero() {
		cfg.Now = time.Now()
	}
	if cfg.StaleVerificationAge == 0 {
		cfg.StaleVerificationAge = 60 * 24 * time.Hour
	}

	absRepo, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return Report{}, fmt.Errorf("active-plan hygiene: resolve repo path: %w", err)
	}
	execRoot := filepath.Join(absRepo, "docs", "exec-plans")
	if _, err := os.Stat(execRoot); err != nil {
		if os.IsNotExist(err) {
			return Report{Issues: []Issue{{
				Path:    "docs/exec-plans",
				Message: "exec-plan directory is missing",
				Fix:     "run 'mars-harness init --repo <path>' or restore docs/exec-plans from source control",
			}}}, nil
		}
		return Report{}, fmt.Errorf("active-plan hygiene: stat docs/exec-plans: %w", err)
	}

	plans, err := collectPlans(absRepo, execRoot)
	if err != nil {
		return Report{}, err
	}
	tickets, err := collectTicketLocations(absRepo)
	if err != nil {
		return Report{}, err
	}

	var report Report
	activeDirPlans := filterPlans(plans, func(p planFile) bool { return p.lifecycle == "active" })
	activeStatusPlans := filterPlans(plans, func(p planFile) bool { return p.status == "active" })

	if len(activeDirPlans) != 1 {
		report.add("docs/exec-plans/active",
			fmt.Sprintf("docs/exec-plans/active must contain exactly one markdown exec plan, got %d", len(activeDirPlans)),
			"keep one active/current-operating-plan.md; move waiting plans to docs/exec-plans/backlog/ and historical plans to docs/exec-plans/superseded/")
	}
	if len(activeStatusPlans) != 1 {
		report.add("docs/exec-plans",
			fmt.Sprintf("exactly one exec plan must declare **Status:** Active, got %d", len(activeStatusPlans)),
			"update docs/exec-plans/active/current-operating-plan.md to declare **Status:** Active and remove active status from every other plan")
	}

	for _, p := range plans {
		switch p.lifecycle {
		case "active":
			if p.status != "active" {
				report.add(p.rel, "active exec plans must declare **Status:** Active", "move superseded or waiting plans out of docs/exec-plans/active/")
			}
			if !validPriority(p.priority) {
				report.add(p.rel, "active exec plans must declare **Priority:** P0-P4", "add **Priority:** P0-P4 near the top of the active plan")
			}
			report.Issues = append(report.Issues, checkActivePlanContent(p, tickets, cfg.Now, cfg.StaleVerificationAge)...)
		case "backlog":
			if p.status != "backlog" {
				report.add(p.rel, "backlog exec plans must declare **Status:** Backlog", "set **Status:** Backlog or move the plan to the matching lifecycle directory")
			}
			if !validPriority(p.priority) {
				report.add(p.rel, "backlog exec plans must declare **Priority:** P0-P4", "add **Priority:** P1-P4 before the dependency metadata")
			}
		default:
			if p.status == "active" {
				report.add(p.rel, "exec plan declares active status outside docs/exec-plans/active", "move the plan to docs/exec-plans/active/current-operating-plan.md or change its status")
			}
			if p.status == "backlog" {
				report.add(p.rel, "waiting exec plans must live in docs/exec-plans/backlog", "move this plan to docs/exec-plans/backlog/ and keep **Priority:** metadata")
			}
		}

		if p.status == "superseded" && !hasCurrentPlanPointer(p.text) {
			report.add(p.rel, "superseded exec plans must point to the current active plan", "add a note linking to docs/exec-plans/active/current-operating-plan.md")
		}
	}

	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Path == report.Issues[j].Path {
			return report.Issues[i].Message < report.Issues[j].Message
		}
		return report.Issues[i].Path < report.Issues[j].Path
	})
	return report, nil
}

type planFile struct {
	rel       string
	lifecycle string
	status    string
	priority  string
	text      string
}

func collectPlans(repoRoot, execRoot string) ([]planFile, error) {
	var plans []planFile
	err := filepath.WalkDir(execRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		lifecycle := planLifecycle(rel)
		status := normalizeStatus(metadataValue(text, "Status"))
		priority := strings.TrimSpace(metadataValue(text, "Priority"))
		if lifecycle == "root" && status == "" && priority == "" {
			return nil
		}
		plans = append(plans, planFile{
			rel:       rel,
			lifecycle: lifecycle,
			status:    status,
			priority:  priority,
			text:      text,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("active-plan hygiene: walk docs/exec-plans: %w", err)
	}
	return plans, nil
}

func planLifecycle(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 || parts[0] != "docs" || parts[1] != "exec-plans" {
		return "root"
	}
	switch parts[2] {
	case "active", "backlog", "completed", "superseded":
		return parts[2]
	default:
		return "root"
	}
}

func metadataValue(text, label string) string {
	re := regexp.MustCompile(`(?mi)^\*\*` + regexp.QuoteMeta(label) + `:\*\*\s*(.+?)\s*$`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func normalizeStatus(status string) string {
	lower := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.HasPrefix(lower, "active"):
		return "active"
	case strings.HasPrefix(lower, "backlog"), strings.HasPrefix(lower, "waiting"), strings.HasPrefix(lower, "planned"):
		return "backlog"
	case strings.HasPrefix(lower, "superseded"):
		return "superseded"
	case strings.HasPrefix(lower, "completed"), strings.HasPrefix(lower, "complete"), strings.HasPrefix(lower, "done"):
		return "completed"
	default:
		return lower
	}
}

func validPriority(priority string) bool {
	return regexp.MustCompile(`(?i)^P[0-4]\b`).MatchString(strings.TrimSpace(priority))
}

func hasCurrentPlanPointer(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "current-operating-plan.md") &&
		(strings.Contains(lower, "active/") || strings.Contains(lower, "docs/exec-plans/active"))
}

func filterPlans(plans []planFile, keep func(planFile) bool) []planFile {
	var out []planFile
	for _, p := range plans {
		if keep(p) {
			out = append(out, p)
		}
	}
	return out
}

func (r *Report) add(path, message, fix string) {
	r.Issues = append(r.Issues, Issue{Path: filepath.ToSlash(path), Message: message, Fix: fix})
}

var (
	ticketIDRE              = regexp.MustCompile(`\bMH-\d{3,}\b`)
	dateRE                  = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	relativeTimeLanguageRE  = regexp.MustCompile(`(?i)\b(latest|recently|currently|today|yesterday|this week|this month|now)\b`)
	verificationLanguageRE  = regexp.MustCompile(`(?i)\b(verification|verified|checks?|evidence|review|observed)\b`)
	unresolvedPlaceholderRE = regexp.MustCompile(`\bTBD\b`)
)

func checkActivePlanContent(p planFile, tickets map[string]string, now time.Time, maxAge time.Duration) []Issue {
	var issues []Issue
	lines := strings.Split(p.text, "\n")
	for i, line := range lines {
		lineNo := i + 1
		if unresolvedPlaceholderRE.MatchString(line) {
			issues = append(issues, Issue{
				Path:    fmt.Sprintf("%s:%d", p.rel, lineNo),
				Message: "active plan contains unresolved TBD placeholder",
				Fix:     "replace TBD with a concrete value, blocker, or explicit 'None'",
			})
		}
		if relativeTimeLanguageRE.MatchString(line) && !dateRE.MatchString(line) {
			issues = append(issues, Issue{
				Path:    fmt.Sprintf("%s:%d", p.rel, lineNo),
				Message: "active plan uses relative status language without an absolute date",
				Fix:     "replace relative language with an absolute YYYY-MM-DD date or a durable source-of-truth pointer",
			})
		}
		if verificationLanguageRE.MatchString(line) {
			for _, match := range dateRE.FindAllString(line, -1) {
				parsed, err := time.Parse("2006-01-02", match)
				if err != nil {
					continue
				}
				if now.Sub(parsed) > maxAge {
					issues = append(issues, Issue{
						Path:    fmt.Sprintf("%s:%d", p.rel, lineNo),
						Message: fmt.Sprintf("active plan verification note from %s is stale", match),
						Fix:     "rerun the named checks, update the dated evidence, or move old verification notes to completed/superseded history",
					})
				}
			}
		}
		issues = append(issues, checkTicketStatusClaims(p.rel, lineNo, line, tickets)...)
	}
	return issues
}

func checkTicketStatusClaims(rel string, lineNo int, line string, tickets map[string]string) []Issue {
	ids := ticketIDRE.FindAllString(line, -1)
	if len(ids) == 0 {
		return nil
	}
	claim := claimedTicketLocation(line)
	if claim == "" {
		return nil
	}
	var issues []Issue
	for _, id := range ids {
		actual := tickets[id]
		if actual == "" || actual == claim {
			continue
		}
		issues = append(issues, Issue{
			Path:    fmt.Sprintf("%s:%d", rel, lineNo),
			Message: fmt.Sprintf("active plan lists %s as %s but ticket is %s", id, claim, actual),
			Fix:     "update the active plan ticket-state section after moving tickets between backlog, in-progress, and done",
		})
	}
	return issues
}

func claimedTicketLocation(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "in-progress"), strings.Contains(lower, "in progress"):
		return "in-progress"
	case strings.Contains(lower, "backlog"):
		return "backlog"
	case strings.Contains(lower, "done"), strings.Contains(lower, "completed"):
		return "done"
	default:
		return ""
	}
}

func collectTicketLocations(repoRoot string) (map[string]string, error) {
	locations := map[string]string{}
	for _, dir := range []string{"backlog", "in-progress", "done"} {
		path := filepath.Join(repoRoot, "docs", "tickets", dir)
		entries, err := os.ReadDir(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("active-plan hygiene: read docs/tickets/%s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			id := ticketIDRE.FindString(entry.Name())
			if id == "" {
				continue
			}
			locations[id] = dir
		}
	}
	return locations, nil
}
