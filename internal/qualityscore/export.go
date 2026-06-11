/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/scoring-system.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-008-scoring-trust-quality.md
- docs/features/F-012-self-improvement-loop.md
*/
package qualityscore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
)

const (
	defaultWindowDays = 30
	staleTicketDays   = 7
	manualStart       = "<!-- BEGIN MANUAL NOTES -->"
	manualEnd         = "<!-- END MANUAL NOTES -->"
)

// Options controls quality score export.
type Options struct {
	RepoPath               string
	RepoID                 string
	DBPath                 string
	WindowDays             int
	Now                    time.Time
	CreateInterventionDebt bool
	DisableTicketCreation  bool
}

// Report summarizes an export run.
type Report struct {
	Path           string
	Grade          string
	Warnings       []string
	TicketsChanged []string
}

type evidence struct {
	repoPath        string
	repoID          string
	dbPath          string
	now             time.Time
	windowDays      int
	since           time.Time
	manualNotes     string
	scoreDBPresent  bool
	scores          []scoring.Score
	outcomes        []scoring.Outcome
	outcomeCounts   []scoring.OutcomeCount
	paceRows        []paceRow
	telemetryCounts []telemetry.RoleCategoryCount
	tickets         ticketSummary
	warnings        []string
	ticketsChanged  []string
}

type ticketSummary struct {
	Backlog              int
	InProgress           int
	Done                 int
	OpenInterventionDebt int
	InProgressTickets    []ticketInfo
	InterventionDebt     []ticketInfo
}

type ticketInfo struct {
	Path       string
	Status     string
	ID         string
	Title      string
	Kind       string
	WorkType   string
	DedupeKey  string
	ModifiedAt time.Time
}

type paceRow struct {
	RepoID          string
	Role            string
	JobID           string
	Outcome         string
	TerminalOutcome scoring.OutcomeType
	TurnCount       int
	ToolInvocations int
	LLMCalls        int
	WallMs          int64
	LimitStop       bool
}

// Export refreshes docs/QUALITY_SCORE.md from live scoring, telemetry, and
// repo-visible ticket evidence. Missing databases are rendered as insufficient
// evidence rather than treated as healthy.
func Export(ctx context.Context, opts Options) (Report, error) {
	if strings.TrimSpace(opts.RepoPath) == "" {
		opts.RepoPath = "."
	}
	repoPath, err := filepath.Abs(opts.RepoPath)
	if err != nil {
		return Report{}, fmt.Errorf("quality score: resolve repo path: %w", err)
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	} else {
		opts.Now = opts.Now.UTC()
	}
	if opts.WindowDays <= 0 {
		opts.WindowDays = defaultWindowDays
	}

	scorePath := filepath.Join(repoPath, "docs", "QUALITY_SCORE.md")
	existing, _ := os.ReadFile(scorePath)
	ev := evidence{
		repoPath:    repoPath,
		repoID:      strings.TrimSpace(opts.RepoID),
		dbPath:      strings.TrimSpace(opts.DBPath),
		now:         opts.Now,
		windowDays:  opts.WindowDays,
		since:       opts.Now.AddDate(0, 0, -opts.WindowDays),
		manualNotes: extractManualNotes(string(existing)),
	}

	if err := ev.collect(ctx); err != nil {
		return Report{}, err
	}
	if opts.CreateInterventionDebt && !opts.DisableTicketCreation {
		var changed []string
		regressionTickets, err := createRegressionTickets(repoPath, ev.repoID, ev.scores)
		if err != nil {
			return Report{}, err
		}
		changed = append(changed, regressionTickets...)
		outcomeTickets, err := createOutcomeSignalTickets(repoPath, ev.repoID, ev.outcomeCounts, opts.WindowDays)
		if err != nil {
			return Report{}, err
		}
		changed = append(changed, outcomeTickets...)
		staleTickets, err := createStaleTicketSignalTickets(repoPath, ev.repoID, ev.tickets.InProgressTickets, opts.Now)
		if err != nil {
			return Report{}, err
		}
		changed = append(changed, staleTickets...)
		ev.ticketsChanged = changed
		if len(changed) > 0 {
			tickets, err := scanTickets(repoPath)
			if err != nil {
				return Report{}, err
			}
			ev.tickets = tickets
		}
	}

	grade := ev.overallGrade()
	rendered := ev.render(grade)
	if err := os.MkdirAll(filepath.Dir(scorePath), 0o755); err != nil {
		return Report{}, fmt.Errorf("quality score: create docs dir: %w", err)
	}
	if err := os.WriteFile(scorePath, []byte(rendered), 0o644); err != nil {
		return Report{}, fmt.Errorf("quality score: write %s: %w", scorePath, err)
	}

	return Report{
		Path:           scorePath,
		Grade:          grade,
		Warnings:       ev.warnings,
		TicketsChanged: ev.ticketsChanged,
	}, nil
}

func (ev *evidence) collect(ctx context.Context) error {
	tickets, err := scanTickets(ev.repoPath)
	if err != nil {
		return err
	}
	ev.tickets = tickets

	if ev.dbPath == "" {
		ev.warnings = append(ev.warnings, "No SQLite database path was supplied; score evidence is insufficient.")
		return nil
	}
	if _, err := os.Stat(ev.dbPath); err != nil {
		if os.IsNotExist(err) {
			ev.warnings = append(ev.warnings, fmt.Sprintf("No SQLite database found at %s; score evidence is insufficient.", ev.dbPath))
			return nil
		}
		return fmt.Errorf("quality score: stat database: %w", err)
	}
	ev.scoreDBPresent = true

	scoreStore, err := scoring.OpenStore(ev.dbPath)
	if err != nil {
		if isDatabaseEvidenceUnavailable(err) {
			ev.warnings = append(ev.warnings, fmt.Sprintf("SQLite score evidence unavailable at %s; score evidence is insufficient. Run `mars-harness setup`, run `mars-harness register --repo <path>`, or pass --db with a writable SQLite path.", ev.dbPath))
			return nil
		}
		return err
	}
	defer scoreStore.Close()

	pairs, err := scoreStore.RoleReposWithOutcomes(ctx, ev.repoID, ev.since)
	if err != nil {
		return err
	}
	scoreByKey := map[string]scoring.Score{}
	for _, pair := range pairs {
		sc, err := scoreStore.ComputeScoreAt(ctx, pair.Role, pair.RepoID, ev.windowDays, ev.now)
		if err != nil {
			return err
		}
		scoreByKey[scoreKey(sc.Role, sc.RepoID)] = sc
	}

	cached, err := scoreStore.ListScores(ctx)
	if err != nil {
		return err
	}
	for _, sc := range cached {
		if !matchesRepo(ev.repoID, sc.RepoID) {
			continue
		}
		key := scoreKey(sc.Role, sc.RepoID)
		if _, ok := scoreByKey[key]; !ok {
			scoreByKey[key] = sc
		}
	}
	for _, sc := range scoreByKey {
		ev.scores = append(ev.scores, sc)
	}
	sort.Slice(ev.scores, func(i, j int) bool {
		if ev.scores[i].RepoID == ev.scores[j].RepoID {
			return ev.scores[i].Role < ev.scores[j].Role
		}
		return ev.scores[i].RepoID < ev.scores[j].RepoID
	})

	counts, err := scoreStore.OutcomeCounts(ctx, ev.repoID, ev.since)
	if err != nil {
		return err
	}
	ev.outcomeCounts = counts
	outcomes, err := scoreStore.OutcomesSince(ctx, ev.repoID, ev.since)
	if err != nil {
		return err
	}
	ev.outcomes = outcomes
	ev.collectPaceRows(ctx)

	telemetryStore, err := telemetry.OpenStore(ev.dbPath)
	if err != nil {
		ev.warnings = append(ev.warnings, fmt.Sprintf("Telemetry evidence unavailable: %v", err))
		return nil
	}
	defer telemetryStore.Close()
	telemetryCounts, err := telemetryStore.RoleCategoryCountsSince(ev.since)
	if err != nil {
		ev.warnings = append(ev.warnings, fmt.Sprintf("Telemetry evidence unavailable: %v", err))
		return nil
	}
	for _, count := range telemetryCounts {
		if matchesRepo(ev.repoID, count.RepoID) {
			ev.telemetryCounts = append(ev.telemetryCounts, count)
		}
	}
	sort.Slice(ev.telemetryCounts, func(i, j int) bool {
		if ev.telemetryCounts[i].Count == ev.telemetryCounts[j].Count {
			return telemetryKey(ev.telemetryCounts[i]) < telemetryKey(ev.telemetryCounts[j])
		}
		return ev.telemetryCounts[i].Count > ev.telemetryCounts[j].Count
	})
	return nil
}

func (ev *evidence) collectPaceRows(ctx context.Context) {
	if strings.TrimSpace(ev.dbPath) == "" || len(ev.outcomes) == 0 {
		return
	}
	traceStore, err := trace.OpenStore(ev.dbPath)
	if err != nil {
		ev.warnings = append(ev.warnings, fmt.Sprintf("Trace pace evidence unavailable: %v", err))
		return
	}
	defer traceStore.Close()

	for _, outcome := range ev.outcomes {
		if strings.TrimSpace(outcome.JobID) == "" {
			continue
		}
		rec, err := traceStore.GetLatestByJobID(ctx, outcome.JobID)
		if err != nil {
			ev.warnings = append(ev.warnings, fmt.Sprintf("Trace pace evidence unavailable for job %s: %v", outcome.JobID, err))
			continue
		}
		if rec == nil || strings.TrimSpace(rec.SummaryJSON) == "" {
			continue
		}
		var summary trace.Summary
		if err := json.Unmarshal([]byte(rec.SummaryJSON), &summary); err != nil {
			ev.warnings = append(ev.warnings, fmt.Sprintf("Trace pace summary for job %s could not be parsed: %v", outcome.JobID, err))
			continue
		}
		if summary.TurnCount == 0 && summary.ToolInvocations == 0 && summary.LLMCalls == 0 && summary.WallMs == 0 {
			continue
		}
		jobID := strings.TrimSpace(summary.JobID)
		if jobID == "" {
			jobID = outcome.JobID
		}
		ev.paceRows = append(ev.paceRows, paceRow{
			RepoID:          outcome.RepoID,
			Role:            outcome.Role,
			JobID:           jobID,
			Outcome:         strings.TrimSpace(summary.Outcome),
			TerminalOutcome: outcome.Type,
			TurnCount:       summary.TurnCount,
			ToolInvocations: summary.ToolInvocations,
			LLMCalls:        summary.LLMCalls,
			WallMs:          summary.WallMs,
			LimitStop:       isLimitStop(summary.Outcome, outcome.Type),
		})
	}
	sort.Slice(ev.paceRows, func(i, j int) bool {
		if ev.paceRows[i].RepoID != ev.paceRows[j].RepoID {
			return ev.paceRows[i].RepoID < ev.paceRows[j].RepoID
		}
		if ev.paceRows[i].Role != ev.paceRows[j].Role {
			return ev.paceRows[i].Role < ev.paceRows[j].Role
		}
		return ev.paceRows[i].JobID < ev.paceRows[j].JobID
	})
}

func isDatabaseEvidenceUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database directory") && strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "database at") && strings.Contains(msg, "is unavailable") ||
		strings.Contains(msg, "unable to open database file") ||
		strings.Contains(msg, "out of memory (14)")
}

func scanTickets(repoPath string) (ticketSummary, error) {
	var summary ticketSummary
	for _, status := range []string{"backlog", "in-progress", "done"} {
		dir := filepath.Join(repoPath, "docs", "tickets", status)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ticketSummary{}, fmt.Errorf("quality score: read %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "README.md" {
				continue
			}
			info, err := readTicketInfo(filepath.Join(dir, entry.Name()), filepath.Join("docs", "tickets", status, entry.Name()), status)
			if err != nil {
				return ticketSummary{}, err
			}
			switch status {
			case "backlog":
				summary.Backlog++
			case "in-progress":
				summary.InProgress++
				summary.InProgressTickets = append(summary.InProgressTickets, info)
			case "done":
				summary.Done++
			}
			if info.Kind == "intervention-debt" || info.WorkType == "intervention-debt" {
				summary.InterventionDebt = append(summary.InterventionDebt, info)
				if status != "done" {
					summary.OpenInterventionDebt++
				}
			}
		}
	}
	sort.Slice(summary.InProgressTickets, func(i, j int) bool {
		return summary.InProgressTickets[i].Path < summary.InProgressTickets[j].Path
	})
	sort.Slice(summary.InterventionDebt, func(i, j int) bool {
		return summary.InterventionDebt[i].Path < summary.InterventionDebt[j].Path
	})
	return summary, nil
}

func readTicketInfo(absPath, relPath, status string) (ticketInfo, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ticketInfo{}, fmt.Errorf("quality score: read ticket %s: %w", relPath, err)
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return ticketInfo{}, fmt.Errorf("quality score: stat ticket %s: %w", relPath, err)
	}
	fm := frontmatter(data)
	info := ticketInfo{
		Path:       relPath,
		Status:     status,
		ID:         fm["id"],
		Title:      fm["title"],
		Kind:       fm["kind"],
		WorkType:   fm["work_type"],
		DedupeKey:  fm["dedupe_key"],
		ModifiedAt: stat.ModTime().UTC(),
	}
	if info.Title == "" {
		info.Title = titleFromPath(relPath)
	}
	return info, nil
}

func frontmatter(data []byte) map[string]string {
	out := map[string]string{}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return out
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(line, " ") || !strings.Contains(line, ":") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return out
}

func createRegressionTickets(repoPath, repoID string, scores []scoring.Score) ([]string, error) {
	root, err := tools.NewRoot(repoPath)
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, sc := range scores {
		if sc.SampleSize < 5 || sc.Value >= 0.5 {
			continue
		}
		snapshot := telemetry.ScoreSnapshot{
			Role:       sc.Role,
			RepoID:     sc.RepoID,
			Value:      sc.Value,
			SampleSize: sc.SampleSize,
			WindowDays: sc.WindowDays,
		}
		proposal, ok := telemetry.TriageScore(snapshot)
		if !ok {
			continue
		}
		if proposal.RepoID == "" {
			proposal.RepoID = repoID
		}
		result, err := tools.CreateTicket(root, tools.TicketInput{
			Title:      fmt.Sprintf("Intervention debt: low score for %s", sc.Role),
			Priority:   "high",
			Complexity: "medium",
			Kind:       "intervention-debt",
			WorkType:   "intervention-debt",
			DedupeKey:  qualityDedupeKey(proposal.RepoID, sc.Role, sc.WindowDays),
			Metadata: map[string]string{
				"role":            sc.Role,
				"repo_id":         proposal.RepoID,
				"target":          string(proposal.Target),
				"category":        "score",
				"severity":        proposal.Severity,
				"confidence":      fmt.Sprintf("%.2f", proposal.Confidence),
				"score_value":     fmt.Sprintf("%.2f", sc.Value),
				"score_samples":   fmt.Sprintf("%d", sc.SampleSize),
				"evidence_window": fmt.Sprintf("%dd", sc.WindowDays),
				"origin_kind":     "quality_score_export",
			},
			Source: fmt.Sprintf("quality-score-export:%s:%s:%dd", proposal.RepoID, sc.Role, sc.WindowDays),
			Body:   regressionTicketBody(proposal, sc),
		})
		if err != nil {
			return nil, err
		}
		changed = append(changed, result.Output)
	}
	return changed, nil
}

func createOutcomeSignalTickets(repoPath, repoID string, counts []scoring.OutcomeCount, windowDays int) ([]string, error) {
	root, err := tools.NewRoot(repoPath)
	if err != nil {
		return nil, err
	}
	if windowDays <= 0 {
		windowDays = defaultWindowDays
	}
	window := fmt.Sprintf("%dd", windowDays)
	var changed []string
	for _, count := range counts {
		category, ok := categoryForOutcome(count.Type)
		if !ok || count.Count <= 0 {
			continue
		}
		eventRepoID := count.RepoID
		if eventRepoID == "" {
			eventRepoID = repoID
		}
		patternCount := count.Count
		if patternCount < telemetry.PatternThreshold {
			patternCount = telemetry.PatternThreshold
		}
		proposal := telemetry.TriagePattern(telemetry.Pattern{
			RepoID:   eventRepoID,
			Role:     count.Role,
			Category: category,
			Count:    patternCount,
			Window:   window,
		})
		proposal.Evidence = fmt.Sprintf("%d %s outcome(s) for repo %s role %s in %s", count.Count, count.Type, eventRepoID, count.Role, window)
		result, err := tools.CreateTicket(root, tools.TicketInput{
			Title:      fmt.Sprintf("Intervention debt: %s for %s", proposal.Title, count.Role),
			Priority:   priorityForSeverity(proposal.Severity),
			Complexity: "medium",
			Kind:       "intervention-debt",
			WorkType:   "intervention-debt",
			DedupeKey:  signalDedupeKey(eventRepoID, count.Role, proposal.Target, category, window),
			Metadata: map[string]string{
				"role":            count.Role,
				"repo_id":         eventRepoID,
				"target":          string(proposal.Target),
				"category":        string(category),
				"severity":        proposal.Severity,
				"confidence":      fmt.Sprintf("%.2f", proposal.Confidence),
				"evidence_window": window,
				"origin_kind":     "quality_score_outcome",
				"outcome_type":    string(count.Type),
				"outcome_count":   fmt.Sprintf("%d", count.Count),
			},
			Source: fmt.Sprintf("quality-score-outcome:%s:%s:%s:%s", eventRepoID, count.Role, count.Type, window),
			Body:   outcomeSignalTicketBody(proposal, count, window),
		})
		if err != nil {
			return nil, err
		}
		changed = append(changed, result.Output)
	}
	return changed, nil
}

func createStaleTicketSignalTickets(repoPath, repoID string, tickets []ticketInfo, now time.Time) ([]string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.AddDate(0, 0, -staleTicketDays)
	var stale []ticketInfo
	for _, ticket := range tickets {
		if ticket.ModifiedAt.IsZero() || ticket.ModifiedAt.After(cutoff) {
			continue
		}
		stale = append(stale, ticket)
	}
	if len(stale) == 0 {
		return nil, nil
	}
	root, err := tools.NewRoot(repoPath)
	if err != nil {
		return nil, err
	}
	role := "engineer"
	category := telemetry.CategoryStaleTicket
	window := fmt.Sprintf("%dd", staleTicketDays)
	proposal := telemetry.TriagePattern(telemetry.Pattern{
		RepoID:   repoID,
		Role:     role,
		Category: category,
		Count:    telemetry.PatternThreshold,
		Window:   window,
	})
	proposal.Evidence = staleTicketEvidence(stale, now)
	result, err := tools.CreateTicket(root, tools.TicketInput{
		Title:      fmt.Sprintf("Intervention debt: %s", proposal.Title),
		Priority:   priorityForSeverity(proposal.Severity),
		Complexity: "medium",
		Kind:       "intervention-debt",
		WorkType:   "intervention-debt",
		DedupeKey:  signalDedupeKey(repoID, role, proposal.Target, category, window),
		Metadata: map[string]string{
			"role":               role,
			"repo_id":            repoID,
			"target":             string(proposal.Target),
			"category":           string(category),
			"severity":           proposal.Severity,
			"confidence":         fmt.Sprintf("%.2f", proposal.Confidence),
			"evidence_window":    window,
			"origin_kind":        "quality_score_ticket_state",
			"stale_ticket_count": fmt.Sprintf("%d", len(stale)),
		},
		Source: fmt.Sprintf("quality-score-ticket-state:%s:%s", repoID, window),
		Body:   staleTicketSignalTicketBody(proposal, stale, now),
	})
	if err != nil {
		return nil, err
	}
	return []string{result.Output}, nil
}

func categoryForOutcome(outcome scoring.OutcomeType) (telemetry.FailureCategory, bool) {
	switch outcome {
	case scoring.OutcomeTimeout:
		return telemetry.CategoryToolTimeout, true
	case scoring.OutcomeGuardrailBlocked:
		return telemetry.CategoryGuardrailBlock, true
	case scoring.OutcomeHumanFollowup:
		return telemetry.CategoryHumanFollowup, true
	case scoring.OutcomeReverted:
		return telemetry.CategoryRevertedCommit, true
	default:
		return "", false
	}
}

func signalDedupeKey(repoID, role string, target telemetry.ImprovementTarget, category telemetry.FailureCategory, window string) string {
	return strings.Join([]string{
		"intervention-debt",
		normalizePart(repoID),
		normalizePart(role),
		normalizePart(string(target)),
		normalizePart(string(category)),
		normalizePart(window),
	}, ":")
}

func priorityForSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func outcomeSignalTicketBody(proposal telemetry.ImprovementProposal, count scoring.OutcomeCount, window string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Context\n\n")
	fmt.Fprintf(&b, "`mars-harness scores export` detected an intervention-debt outcome signal while refreshing `docs/QUALITY_SCORE.md`.\n\n")
	fmt.Fprintf(&b, "## Evidence\n\n")
	fmt.Fprintf(&b, "- Role: %s\n", count.Role)
	fmt.Fprintf(&b, "- Repo ID: %s\n", proposal.RepoID)
	fmt.Fprintf(&b, "- Outcome: %s\n", count.Type)
	fmt.Fprintf(&b, "- Count: %d\n", count.Count)
	fmt.Fprintf(&b, "- Evidence window: %s\n\n", window)
	fmt.Fprintf(&b, "## Recommendation\n\n%s\n\n", strings.TrimSpace(proposal.Suggestion))
	fmt.Fprintf(&b, "## Acceptance Criteria\n\n")
	fmt.Fprintf(&b, "### Functional (happy path)\n\n")
	fmt.Fprintf(&b, "- [ ] The originating outcome signal is linked to trace, commit, score, or ticket evidence where available.\n")
	fmt.Fprintf(&b, "- [ ] The harness workflow change prevents the same signal from recurring in the evidence window.\n\n")
	fmt.Fprintf(&b, "### Edge cases and negative paths\n\n")
	fmt.Fprintf(&b, "- [ ] Missing optional GitHub or commit metadata does not block local ticket creation.\n")
	fmt.Fprintf(&b, "- [ ] Existing matching intervention-debt tickets are updated instead of duplicated.\n\n")
	fmt.Fprintf(&b, "### Observability, docs, and regressions\n\n")
	fmt.Fprintf(&b, "- [ ] `docs/QUALITY_SCORE.md` and completion notes link the relevant verification evidence.")
	return b.String()
}

func staleTicketEvidence(tickets []ticketInfo, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d in-progress ticket(s) older than %d days as of %s", len(tickets), staleTicketDays, now.Format("2006-01-02"))
	for _, ticket := range tickets {
		fmt.Fprintf(&b, "\n- %s last modified %s", ticket.Path, ticket.ModifiedAt.Format("2006-01-02"))
	}
	return b.String()
}

func staleTicketSignalTicketBody(proposal telemetry.ImprovementProposal, tickets []ticketInfo, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Context\n\n")
	fmt.Fprintf(&b, "`mars-harness scores export` detected stale in-progress ticket state while refreshing `docs/QUALITY_SCORE.md`.\n\n")
	fmt.Fprintf(&b, "## Evidence\n\n%s\n\n", staleTicketEvidence(tickets, now))
	fmt.Fprintf(&b, "## Recommendation\n\n%s\n\n", strings.TrimSpace(proposal.Suggestion))
	fmt.Fprintf(&b, "## Acceptance Criteria\n\n")
	fmt.Fprintf(&b, "### Functional (happy path)\n\n")
	fmt.Fprintf(&b, "- [ ] Each stale in-progress ticket is completed, moved back to backlog with blocker evidence, or converted into focused intervention debt.\n")
	fmt.Fprintf(&b, "- [ ] The Engineer and Janitor flow prevents the same stale state from recurring.\n\n")
	fmt.Fprintf(&b, "### Edge cases and negative paths\n\n")
	fmt.Fprintf(&b, "- [ ] Tickets blocked by dependencies record the blocker explicitly instead of silently remaining active.\n")
	fmt.Fprintf(&b, "- [ ] Existing matching stale-ticket intervention debt is updated instead of duplicated.\n\n")
	fmt.Fprintf(&b, "### Observability, docs, and regressions\n\n")
	fmt.Fprintf(&b, "- [ ] Follow-up quality export no longer reports the same stale in-progress ticket set.")
	return b.String()
}

func qualityDedupeKey(repoID, role string, windowDays int) string {
	return strings.Join([]string{
		"intervention-debt",
		normalizePart(repoID),
		normalizePart(role),
		"process",
		"score",
		fmt.Sprintf("%dd", windowDays),
	}, ":")
}

func regressionTicketBody(proposal telemetry.ImprovementProposal, sc scoring.Score) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Context\n\n")
	fmt.Fprintf(&b, "`mars-harness scores export` detected a low role score while refreshing `docs/QUALITY_SCORE.md`.\n\n")
	fmt.Fprintf(&b, "## Evidence\n\n")
	fmt.Fprintf(&b, "- Role: %s\n", sc.Role)
	fmt.Fprintf(&b, "- Repo ID: %s\n", sc.RepoID)
	fmt.Fprintf(&b, "- Score snapshot: %.2f over %d samples in %dd\n", sc.Value, sc.SampleSize, sc.WindowDays)
	fmt.Fprintf(&b, "- Formula: %s\n\n", sc.Formula)
	fmt.Fprintf(&b, "## Recommendation\n\n%s\n\n", strings.TrimSpace(proposal.Suggestion))
	fmt.Fprintf(&b, "## Acceptance Criteria\n\n")
	fmt.Fprintf(&b, "### Functional (happy path)\n\n")
	fmt.Fprintf(&b, "- [ ] Root cause is linked to failed outcomes, trace evidence, telemetry, or ticket flow.\n")
	fmt.Fprintf(&b, "- [ ] Prompt, process, guardrail, skill, context, inference, or tool-policy fix is scoped before implementation.\n")
	fmt.Fprintf(&b, "- [ ] A follow-up score export no longer flags the same low-score window.\n\n")
	fmt.Fprintf(&b, "### Edge cases and negative paths\n\n")
	fmt.Fprintf(&b, "- [ ] Sparse scores below five samples remain informational and do not trigger debt.\n")
	fmt.Fprintf(&b, "- [ ] Existing matching intervention-debt tickets are updated instead of duplicated.\n\n")
	fmt.Fprintf(&b, "### Observability, docs, and regressions\n\n")
	fmt.Fprintf(&b, "- [ ] `docs/QUALITY_SCORE.md` links the relevant evidence and next action.\n")
	fmt.Fprintf(&b, "- [ ] Tests or dogfood evidence cover deterministic failure modes where available.")
	return b.String()
}

func (ev evidence) render(grade string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Quality Score\n\n")
	fmt.Fprintf(&b, "**Status:** Generated\n")
	fmt.Fprintf(&b, "**Updated:** %s\n", ev.now.Format("2006-01-02"))
	fmt.Fprintf(&b, "**Owner:** Project maintainers\n")
	fmt.Fprintf(&b, "**Generated by:** `mars-harness scores export --repo <path>`\n")
	if ev.repoID != "" {
		fmt.Fprintf(&b, "**Repo ID:** `%s`\n", ev.repoID)
	}
	if ev.dbPath != "" {
		fmt.Fprintf(&b, "**Source DB:** `%s`\n", ev.dbPath)
	}
	fmt.Fprintf(&b, "**Evidence window:** %dd\n\n", ev.windowDays)

	fmt.Fprintf(&b, "## Purpose\n\n")
	fmt.Fprintf(&b, "This file is the repo-visible quality artifact for Mars Harness evidence. It is generated from the scoring database, telemetry, ticket state, and preserved manual notes so agents can inspect quality without treating the dashboard as the source of truth.\n\n")

	fmt.Fprintf(&b, "## Grading Scale\n\n")
	fmt.Fprintf(&b, "| Grade | Meaning |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| A | Complete, tested, documented, and consistently healthy. |\n")
	fmt.Fprintf(&b, "| B | Functional with minor gaps or hardening work still open. |\n")
	fmt.Fprintf(&b, "| C | Partially healthy; meaningful implementation or proof work remains. |\n")
	fmt.Fprintf(&b, "| D | Unhealthy or under-proven; corrective work is needed. |\n")
	fmt.Fprintf(&b, "| F | Failing repeatedly or missing the expected quality surface. |\n")
	fmt.Fprintf(&b, "| Insufficient evidence | Live SQLite evidence is missing or too sparse to grade honestly. |\n\n")

	fmt.Fprintf(&b, "## Overall Roll-Up\n\n")
	fmt.Fprintf(&b, "| Area | Grade | Evidence | Next Action |\n| --- | --- | --- | --- |\n")
	for _, row := range ev.rollupRows(grade) {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", row.area, row.grade, row.evidence, row.next)
	}
	fmt.Fprintf(&b, "\n**Current overall grade: %s.**\n\n", grade)

	fmt.Fprintf(&b, "## Role Health\n\n")
	if len(ev.scores) == 0 {
		fmt.Fprintf(&b, "No role scores were available in the selected evidence window.\n\n")
	} else {
		fmt.Fprintf(&b, "| Repo | Role | Grade | Score | Samples | Window | Computed |\n| --- | --- | --- | ---: | ---: | ---: | --- |\n")
		for _, sc := range ev.scores {
			fmt.Fprintf(&b, "| %s | %s | %s | %.2f | %d | %dd | %s |\n",
				emptyDash(sc.RepoID), sc.Role, gradeForScore(sc.Value, sc.SampleSize), sc.Value, sc.SampleSize, sc.WindowDays, sc.ComputedAt.Format(time.RFC3339))
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Factory Pace\n\n")
	pace := summarizePace(ev.paceRows)
	if len(pace) == 0 {
		fmt.Fprintf(&b, "No trace pace evidence was available in the selected evidence window.\n\n")
	} else {
		fmt.Fprintf(&b, "| Repo | Role | Jobs | Avg Turns | Avg Tool Invocations | Avg LLM Calls | Avg Wall | Limit Stops | Pace Signal |\n| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
		for _, row := range pace {
			fmt.Fprintf(&b, "| %s | %s | %d | %.1f | %.1f | %.1f | %s | %d | %s |\n",
				emptyDash(row.RepoID), row.Role, row.Jobs, row.AvgTurns, row.AvgToolInvocations, row.AvgLLMCalls, formatSeconds(row.AvgWallMs), row.LimitStops, row.Signal)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Evidence Signals\n\n")
	outcomes := summarizeOutcomes(ev.outcomeCounts)
	remediation := summarizeRemediation(ev.outcomes)
	fmt.Fprintf(&b, "| Signal | Evidence |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| Role scores | %s |\n", roleScoreSignal(ev.scores))
	fmt.Fprintf(&b, "| Factory pace | %s |\n", paceSignal(pace))
	fmt.Fprintf(&b, "| Terminal outcomes | %s |\n", terminalOutcomeSignal(outcomes))
	fmt.Fprintf(&b, "| Stuck tickets | %s |\n", stuckTicketSignal(ev.tickets))
	fmt.Fprintf(&b, "| Failed dogfood | %s |\n", countSignal(outcomes.dogfoodFailures, "dogfood failures"))
	fmt.Fprintf(&b, "| Guardrail blocks | %s |\n", countSignal(outcomes.byType[scoring.OutcomeGuardrailBlocked], "guardrail blocks"))
	fmt.Fprintf(&b, "| Intervention debt | %s |\n", interventionDebtSignal(ev.tickets))
	fmt.Fprintf(&b, "| Check results | %d passed, %d failed |\n", outcomes.byType[scoring.OutcomeChecksPassed], outcomes.byType[scoring.OutcomeChecksFailed])
	fmt.Fprintf(&b, "| No-op runs | %s |\n", countSignal(outcomes.byType[scoring.OutcomeNoop], "no-op runs"))
	fmt.Fprintf(&b, "| Human follow-up | %s |\n", countSignal(outcomes.byType[scoring.OutcomeHumanFollowup], "human follow-up outcomes"))
	fmt.Fprintf(&b, "| Deterministic remediation | %s |\n", remediationSignal(remediation))
	fmt.Fprintf(&b, "| Top telemetry triage targets | %s |\n\n", telemetrySignal(ev.telemetryCounts))

	fmt.Fprintf(&b, "## Top Improvement Targets\n\n")
	for _, target := range ev.improvementTargets(grade, outcomes) {
		fmt.Fprintf(&b, "%d. %s\n", target.index, target.text)
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## Source And Target Contract\n\n")
	fmt.Fprintf(&b, "- Refresh this artifact with `mars-harness scores export --repo <path>`.\n")
	fmt.Fprintf(&b, "- The export reads role scores, terminal outcomes, tickets, telemetry, dogfood, guardrail blocks, no-op runs, human follow-up, deterministic remediation attempts, and check outcomes from the same evidence used by dashboard quality views.\n")
	fmt.Fprintf(&b, "- The dashboard may link to or display this data, but `docs/QUALITY_SCORE.md` remains the repo-visible source of truth for quality claims.\n")
	fmt.Fprintf(&b, "- The quality score separates shipped feature scenarios from enabler work; feature claims still require mapped BDD evidence.\n")
	fmt.Fprintf(&b, "- Low role scores and recurring failures are reported as improvement targets by default; pass `--create-intervention-debt` when ticket materialization is deliberately wanted.\n")
	fmt.Fprintf(&b, "- Missing optional telemetry leaves an explicit evidence warning instead of failing the export.\n\n")

	fmt.Fprintf(&b, "## Manual Notes\n\n")
	fmt.Fprintf(&b, "%s\n%s\n%s\n\n", manualStart, strings.TrimSpace(ev.manualNotes), manualEnd)

	fmt.Fprintf(&b, "## Generation Notes\n\n")
	if len(ev.warnings) == 0 {
		fmt.Fprintf(&b, "- No export warnings.\n")
	} else {
		for _, warning := range ev.warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	for _, output := range ev.ticketsChanged {
		fmt.Fprintf(&b, "- Regression ticket: %s\n", output)
	}
	return b.String()
}

type rollupRow struct {
	area     string
	grade    string
	evidence string
	next     string
}

func (ev evidence) rollupRows(overall string) []rollupRow {
	outcomes := summarizeOutcomes(ev.outcomeCounts)
	pace := summarizePace(ev.paceRows)
	return []rollupRow{
		{
			area:     "Role health",
			grade:    roleHealthGrade(ev.scores),
			evidence: roleScoreSignal(ev.scores),
			next:     roleHealthNext(ev.scores),
		},
		{
			area:     "Factory pace",
			grade:    paceGrade(pace),
			evidence: paceSignal(pace),
			next:     "Use trace pace rows to target high-turn or limit-stop roles before raising runtime limits.",
		},
		{
			area:     "Terminal outcomes and checks",
			grade:    outcomeGrade(outcomes),
			evidence: terminalOutcomeSignal(outcomes),
			next:     "Investigate failed checks, guardrail blocks, no-op runs, and human follow-up.",
		},
		{
			area:     "Ticket flow and intervention debt",
			grade:    ticketGrade(ev.tickets),
			evidence: fmt.Sprintf("%d backlog, %d in-progress, %d done, %d open intervention-debt", ev.tickets.Backlog, ev.tickets.InProgress, ev.tickets.Done, ev.tickets.OpenInterventionDebt),
			next:     "Drain in-progress and high-priority intervention debt before ordinary backlog work; keep medium/low intervention debt visible without blocking product progress.",
		},
		{
			area:     "Telemetry and dogfood",
			grade:    telemetryGrade(ev.telemetryCounts, outcomes.dogfoodFailures),
			evidence: fmt.Sprintf("%s; %s", telemetrySignal(ev.telemetryCounts), countSignal(outcomes.dogfoodFailures, "dogfood failures")),
			next:     "Promote recurring telemetry and dogfood failures into bounded remediation.",
		},
		{
			area:     "Evidence coverage",
			grade:    overall,
			evidence: evidenceCoverageSignal(ev),
			next:     "Run harness jobs with scoring enabled when evidence is insufficient.",
		},
	}
}

type outcomeSummary struct {
	byType          map[scoring.OutcomeType]int
	total           int
	positive        int
	negative        int
	dogfoodFailures int
}

type paceSummary struct {
	RepoID             string
	Role               string
	Jobs               int
	AvgTurns           float64
	AvgToolInvocations float64
	AvgLLMCalls        float64
	AvgWallMs          float64
	LimitStops         int
	Signal             string
}

type remediationSummary struct {
	Attempts        map[string]int
	Executions      map[string]int
	TotalAttempts   int
	TotalExecutions int
	Failed          int
	NoExecutor      int
}

type remediationDetails struct {
	Attempts []struct {
		RecipeID string `json:"recipe_id"`
		Status   string `json:"status"`
	} `json:"remediation_attempts"`
	Executions []struct {
		RecipeID string `json:"recipe_id"`
		Status   string `json:"status"`
	} `json:"remediation_executions"`
}

func summarizeOutcomes(counts []scoring.OutcomeCount) outcomeSummary {
	summary := outcomeSummary{byType: map[scoring.OutcomeType]int{}}
	for _, count := range counts {
		summary.byType[count.Type] += count.Count
		summary.total += count.Count
		if count.Role == "dogfood" && isNegative(count.Type) {
			summary.dogfoodFailures += count.Count
		}
		if isPositive(count.Type) {
			summary.positive += count.Count
		}
		if isNegative(count.Type) {
			summary.negative += count.Count
		}
	}
	return summary
}

func summarizePace(rows []paceRow) []paceSummary {
	type acc struct {
		repoID           string
		role             string
		jobs             int
		turns            int
		toolInvocations  int
		llmCalls         int
		wallMs           int64
		limitStops       int
		negativeOutcomes int
	}
	byKey := map[string]*acc{}
	for _, row := range rows {
		key := row.RepoID + "\x00" + row.Role
		a := byKey[key]
		if a == nil {
			a = &acc{repoID: row.RepoID, role: row.Role}
			byKey[key] = a
		}
		a.jobs++
		a.turns += row.TurnCount
		a.toolInvocations += row.ToolInvocations
		a.llmCalls += row.LLMCalls
		a.wallMs += row.WallMs
		if row.LimitStop {
			a.limitStops++
		}
		if isNegative(row.TerminalOutcome) {
			a.negativeOutcomes++
		}
	}
	out := make([]paceSummary, 0, len(byKey))
	for _, a := range byKey {
		if a.jobs == 0 {
			continue
		}
		summary := paceSummary{
			RepoID:             a.repoID,
			Role:               a.role,
			Jobs:               a.jobs,
			AvgTurns:           float64(a.turns) / float64(a.jobs),
			AvgToolInvocations: float64(a.toolInvocations) / float64(a.jobs),
			AvgLLMCalls:        float64(a.llmCalls) / float64(a.jobs),
			AvgWallMs:          float64(a.wallMs) / float64(a.jobs),
			LimitStops:         a.limitStops,
		}
		summary.Signal = paceRowSignal(summary, a.negativeOutcomes)
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RepoID != out[j].RepoID {
			return out[i].RepoID < out[j].RepoID
		}
		return out[i].Role < out[j].Role
	})
	return out
}

func paceRowSignal(row paceSummary, negativeOutcomes int) string {
	switch {
	case row.LimitStops > 0:
		return "limit-stop evidence"
	case row.AvgTurns >= 30 || row.AvgToolInvocations >= 20:
		return "high-turn baseline"
	case negativeOutcomes > 0:
		return "negative-outcome baseline"
	default:
		return "trace baseline"
	}
}

func isLimitStop(outcome string, terminal scoring.OutcomeType) bool {
	if terminal == scoring.OutcomeTimeout {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "max_turns", "max_tool_calls", "timeout", "circle_detected", "empty_response", "budget_exceeded":
		return true
	default:
		return false
	}
}

func summarizeRemediation(outcomes []scoring.Outcome) remediationSummary {
	summary := remediationSummary{
		Attempts:   map[string]int{},
		Executions: map[string]int{},
	}
	for _, outcome := range outcomes {
		if strings.TrimSpace(outcome.Details) == "" {
			continue
		}
		var details remediationDetails
		if err := json.Unmarshal([]byte(outcome.Details), &details); err != nil {
			continue
		}
		for _, attempt := range details.Attempts {
			key := remediationKey(attempt.RecipeID, attempt.Status)
			if key == "" {
				continue
			}
			summary.Attempts[key]++
			summary.TotalAttempts++
		}
		for _, execution := range details.Executions {
			key := remediationKey(execution.RecipeID, execution.Status)
			if key == "" {
				continue
			}
			summary.Executions[key]++
			summary.TotalExecutions++
			switch execution.Status {
			case "failed":
				summary.Failed++
			case "skipped_no_executor":
				summary.NoExecutor++
			}
		}
	}
	return summary
}

func (ev evidence) overallGrade() string {
	outcomes := summarizeOutcomes(ev.outcomeCounts)
	if !ev.scoreDBPresent || len(ev.scores) == 0 || outcomes.total == 0 {
		return "Insufficient evidence"
	}
	weighted, samples := weightedScore(ev.scores)
	if samples < 5 {
		return "Insufficient evidence"
	}
	score := weighted
	score -= float64(min(ev.tickets.InProgress, 5)) * 0.03
	score -= float64(min(ev.tickets.OpenInterventionDebt, 5)) * 0.02
	score -= float64(min(recurringTelemetryCount(ev.telemetryCounts), 4)) * 0.04
	score -= float64(min(outcomes.dogfoodFailures, 4)) * 0.04
	return gradeForScore(score, samples)
}

func roleHealthGrade(scores []scoring.Score) string {
	weighted, samples := weightedScore(scores)
	return gradeForScore(weighted, samples)
}

func paceGrade(pace []paceSummary) string {
	if len(pace) == 0 {
		return "Insufficient evidence"
	}
	var jobs int
	var limitStops int
	var highTurnRows int
	for _, row := range pace {
		jobs += row.Jobs
		limitStops += row.LimitStops
		if row.AvgTurns >= 30 || row.AvgToolInvocations >= 20 {
			highTurnRows++
		}
	}
	switch {
	case jobs == 0:
		return "Insufficient evidence"
	case limitStops > 0:
		return "D"
	case highTurnRows > 0:
		return "C"
	default:
		return "B"
	}
}

func outcomeGrade(outcomes outcomeSummary) string {
	if outcomes.total == 0 {
		return "Insufficient evidence"
	}
	rate := float64(outcomes.positive) / float64(outcomes.total)
	return gradeForScore(rate, outcomes.total)
}

func ticketGrade(tickets ticketSummary) string {
	if tickets.InProgress > 0 {
		return "D"
	}
	if tickets.OpenInterventionDebt > 5 {
		return "C"
	}
	if tickets.OpenInterventionDebt > 0 {
		return "B"
	}
	return "A"
}

func telemetryGrade(counts []telemetry.RoleCategoryCount, dogfoodFailures int) string {
	if recurringTelemetryCount(counts) > 0 || dogfoodFailures > 0 {
		return "D"
	}
	if len(counts) > 0 {
		return "B"
	}
	return "A"
}

func gradeForScore(value float64, samples int) string {
	if samples < 5 {
		return "Insufficient evidence"
	}
	switch {
	case value >= 0.9:
		return "A"
	case value >= 0.8:
		return "B"
	case value >= 0.65:
		return "C"
	case value >= 0.5:
		return "D"
	default:
		return "F"
	}
}

func weightedScore(scores []scoring.Score) (float64, int) {
	var sum float64
	var samples int
	for _, sc := range scores {
		if sc.SampleSize <= 0 {
			continue
		}
		sum += sc.Value * float64(sc.SampleSize)
		samples += sc.SampleSize
	}
	if samples == 0 {
		return 0, 0
	}
	return sum / float64(samples), samples
}

func (ev evidence) improvementTargets(grade string, outcomes outcomeSummary) []struct {
	index int
	text  string
} {
	var targets []string
	pace := summarizePace(ev.paceRows)
	if grade == "Insufficient evidence" {
		targets = append(targets, "Run harness jobs with scoring enabled or pass `--db` for the repo-specific SQLite database.")
	}
	for _, row := range pace {
		if row.LimitStops > 0 {
			targets = append(targets, fmt.Sprintf("Review `%s/%s` pace: %d limit stop(s) with %.1f average turns.", emptyDash(row.RepoID), row.Role, row.LimitStops, row.AvgTurns))
			continue
		}
		if row.AvgTurns >= 30 || row.AvgToolInvocations >= 20 {
			targets = append(targets, fmt.Sprintf("Review `%s/%s` pace: %.1f average turns and %.1f average tool invocations.", emptyDash(row.RepoID), row.Role, row.AvgTurns, row.AvgToolInvocations))
		}
	}
	for _, sc := range ev.scores {
		if sc.SampleSize >= 5 && sc.Value < 0.5 {
			targets = append(targets, fmt.Sprintf("Triage low `%s` score %.2f over %d samples.", sc.Role, sc.Value, sc.SampleSize))
		}
	}
	for _, count := range ev.telemetryCounts {
		if count.Count >= telemetry.PatternThreshold {
			proposal := telemetry.TriagePattern(telemetry.Pattern{
				RepoID:   count.RepoID,
				Role:     count.Role,
				Category: count.Category,
				Count:    count.Count,
				Window:   fmt.Sprintf("%dd", ev.windowDays),
			})
			targets = append(targets, fmt.Sprintf("Address `%s` telemetry for `%s`: %s", count.Category, count.Role, proposal.Suggestion))
		}
	}
	if ev.tickets.InProgress > 0 {
		targets = append(targets, fmt.Sprintf("Drain %d in-progress ticket(s) before starting new backlog work.", ev.tickets.InProgress))
	}
	if ev.tickets.OpenInterventionDebt > 0 {
		targets = append(targets, fmt.Sprintf("Resolve, downgrade, or leave non-blocking evidence for %d open intervention-debt ticket(s).", ev.tickets.OpenInterventionDebt))
	}
	if outcomes.byType[scoring.OutcomeChecksFailed] > 0 {
		targets = append(targets, fmt.Sprintf("Investigate %d failed check outcome(s).", outcomes.byType[scoring.OutcomeChecksFailed]))
	}
	if outcomes.byType[scoring.OutcomeNoop] > 0 {
		targets = append(targets, fmt.Sprintf("Review %d no-op run(s) where actionable work may have existed.", outcomes.byType[scoring.OutcomeNoop]))
	}
	remediation := summarizeRemediation(ev.outcomes)
	if remediation.NoExecutor > 0 {
		targets = append(targets, fmt.Sprintf("Add deterministic executors or downgrade %d auto-safe remediation attempt(s) that skipped without an executor.", remediation.NoExecutor))
	}
	if remediation.Failed > 0 {
		targets = append(targets, fmt.Sprintf("Investigate %d failed deterministic remediation execution(s).", remediation.Failed))
	}
	if len(targets) == 0 {
		targets = append(targets, "Keep score export in the release checklist and refresh after material changes.")
	}
	out := make([]struct {
		index int
		text  string
	}, 0, len(targets))
	for i, target := range targets {
		out = append(out, struct {
			index int
			text  string
		}{index: i + 1, text: target})
	}
	return out
}

func extractManualNotes(existing string) string {
	start := strings.Index(existing, manualStart)
	end := strings.Index(existing, manualEnd)
	if start >= 0 && end > start {
		return strings.TrimSpace(existing[start+len(manualStart) : end])
	}
	return "_No manual notes recorded. Keep human context here; `scores export` preserves this block._"
}

func matchesRepo(want, got string) bool {
	return want == "" || got == "" || want == got
}

func scoreKey(role, repoID string) string {
	return repoID + "\x00" + role
}

func telemetryKey(count telemetry.RoleCategoryCount) string {
	return count.RepoID + "\x00" + count.Role + "\x00" + string(count.Category)
}

func isPositive(typ scoring.OutcomeType) bool {
	switch typ {
	case scoring.OutcomePassed, scoring.OutcomeCommitted, scoring.OutcomeChecksPassed, scoring.OutcomeMerged:
		return true
	default:
		return false
	}
}

func isNegative(typ scoring.OutcomeType) bool {
	switch typ {
	case scoring.OutcomeChecksFailed,
		scoring.OutcomeGuardrailBlocked,
		scoring.OutcomeReverted,
		scoring.OutcomeHumanFollowup,
		scoring.OutcomeClosed,
		scoring.OutcomeFailed,
		scoring.OutcomeNoop,
		scoring.OutcomeTimeout:
		return true
	default:
		return false
	}
}

func roleScoreSignal(scores []scoring.Score) string {
	weighted, samples := weightedScore(scores)
	if samples == 0 {
		return "No scored role outcomes."
	}
	return fmt.Sprintf("%d scored samples, weighted score %.2f", samples, weighted)
}

func paceSignal(pace []paceSummary) string {
	if len(pace) == 0 {
		return "No trace pace evidence."
	}
	var jobs int
	var limitStops int
	var maxTurns float64
	var slowest string
	for _, row := range pace {
		jobs += row.Jobs
		limitStops += row.LimitStops
		if row.AvgTurns > maxTurns {
			maxTurns = row.AvgTurns
			slowest = fmt.Sprintf("%s/%s", emptyDash(row.RepoID), row.Role)
		}
	}
	if limitStops > 0 {
		return fmt.Sprintf("%d traced job(s), %d limit stop(s), slowest average %.1f turns at `%s`", jobs, limitStops, maxTurns, slowest)
	}
	return fmt.Sprintf("%d traced job(s), slowest average %.1f turns at `%s`", jobs, maxTurns, slowest)
}

func roleHealthNext(scores []scoring.Score) string {
	for _, sc := range scores {
		if sc.SampleSize >= 5 && sc.Value < 0.5 {
			return "Work low-score intervention-debt tickets before raising autonomy."
		}
	}
	return "Keep recording terminal outcomes for every role run."
}

func terminalOutcomeSignal(outcomes outcomeSummary) string {
	if outcomes.total == 0 {
		return "No terminal outcomes recorded."
	}
	return fmt.Sprintf("%d positive, %d negative, %d total", outcomes.positive, outcomes.negative, outcomes.total)
}

func stuckTicketSignal(tickets ticketSummary) string {
	if tickets.InProgress == 0 {
		return "No in-progress tickets."
	}
	names := make([]string, 0, min(len(tickets.InProgressTickets), 3))
	for _, ticket := range tickets.InProgressTickets {
		names = append(names, fmt.Sprintf("`%s`", ticket.Path))
		if len(names) == 3 {
			break
		}
	}
	return fmt.Sprintf("%d in-progress: %s", tickets.InProgress, strings.Join(names, ", "))
}

func interventionDebtSignal(tickets ticketSummary) string {
	return fmt.Sprintf("%d open intervention-debt, %d total intervention-debt", tickets.OpenInterventionDebt, len(tickets.InterventionDebt))
}

func countSignal(count int, noun string) string {
	if count == 0 {
		return "None recorded"
	}
	return fmt.Sprintf("%d %s", count, noun)
}

func telemetrySignal(counts []telemetry.RoleCategoryCount) string {
	if len(counts) == 0 {
		return "No telemetry triage targets"
	}
	parts := make([]string, 0, min(len(counts), 3))
	for _, count := range counts {
		parts = append(parts, fmt.Sprintf("`%s/%s` %s x%d", emptyDash(count.RepoID), count.Role, count.Category, count.Count))
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, "; ")
}

func remediationSignal(summary remediationSummary) string {
	if summary.TotalAttempts == 0 {
		return "No remediation attempts recorded."
	}
	parts := []string{fmt.Sprintf("%d attempt(s)", summary.TotalAttempts)}
	if len(summary.Attempts) > 0 {
		parts = append(parts, "attempts: "+topCountSignals(summary.Attempts))
	}
	if summary.TotalExecutions > 0 {
		parts = append(parts, fmt.Sprintf("%d execution(s): %s", summary.TotalExecutions, topCountSignals(summary.Executions)))
	}
	return strings.Join(parts, "; ")
}

func topCountSignals(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] == counts[keys[j]] {
			return keys[i] < keys[j]
		}
		return counts[keys[i]] > counts[keys[j]]
	})
	parts := make([]string, 0, min(len(keys), 3))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("`%s` x%d", key, counts[key]))
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, ", ")
}

func remediationKey(recipeID, status string) string {
	recipeID = strings.TrimSpace(recipeID)
	status = strings.TrimSpace(status)
	if recipeID == "" || status == "" {
		return ""
	}
	return recipeID + " " + status
}

func evidenceCoverageSignal(ev evidence) string {
	if !ev.scoreDBPresent {
		return "SQLite score database missing."
	}
	if len(ev.scores) == 0 {
		return "SQLite present but no role scores in the selected window."
	}
	return "SQLite score and telemetry evidence available."
}

func recurringTelemetryCount(counts []telemetry.RoleCategoryCount) int {
	var total int
	for _, count := range counts {
		if count.Count >= telemetry.PatternThreshold {
			total++
		}
	}
	return total
}

func normalizePart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '_' || r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}

func titleFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	parts := strings.SplitN(base, "-", 3)
	if len(parts) == 3 {
		base = parts[2]
	}
	return strings.ReplaceAll(base, "-", " ")
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func formatSeconds(ms float64) string {
	return fmt.Sprintf("%.1fs", ms/1000)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
