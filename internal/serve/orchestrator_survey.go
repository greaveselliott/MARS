package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

const (
	orchestratorSurveyInterval       = 15 * time.Minute
	orchestratorSurveyStuckAfter     = 6 * time.Hour
	orchestratorSurveyEvidenceWindow = 24 * time.Hour
	orchestratorSurveyDailyCap       = 3
)

type orchestratorSurveyReport struct {
	ReposTriaged      int
	JobsRouted        int
	TicketsTriaged    int
	StuckJobsFailed   int
	RecoveryJobsFixed int
}

func (s *Server) runOrchestratorSurvey(ctx context.Context) {
	s.runSingleOrchestratorSurvey(ctx, "startup")

	ticker := time.NewTicker(orchestratorSurveyInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runSingleOrchestratorSurvey(ctx, "watchdog")
		}
	}
}

func (s *Server) runSingleOrchestratorSurvey(ctx context.Context, source string) {
	surveyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	report, err := s.surveyOrchestrator(surveyCtx, source)
	if err != nil {
		slog.Warn("serve: orchestrator survey failed", "source", source, "err", err)
		return
	}
	if report.JobsRouted == 0 && report.TicketsTriaged == 0 && report.StuckJobsFailed == 0 && report.RecoveryJobsFixed == 0 {
		return
	}
	slog.Info("serve: orchestrator survey complete",
		"source", source,
		"repos", report.ReposTriaged,
		"jobs_routed", report.JobsRouted,
		"tickets_triaged", report.TicketsTriaged,
		"stuck_jobs_failed", report.StuckJobsFailed,
		"recovery_jobs_fixed", report.RecoveryJobsFixed,
	)
	if s.dash != nil {
		payload, _ := json.Marshal(report)
		s.dash.BroadcastEvent("orchestrator_survey", string(payload))
	}
}

func (s *Server) surveyOrchestrator(ctx context.Context, source string) (orchestratorSurveyReport, error) {
	var report orchestratorSurveyReport

	recovery, err := s.queue.RepairActiveRecoveryJobs(ctx, recoveryJobStaleAfter)
	if err != nil {
		return report, err
	}
	report.RecoveryJobsFixed = recovery.Total()

	stuck, err := s.queue.FailStuckRunningJobs(ctx, orchestratorSurveyStuckAfter, "orchestrator survey watchdog: running job exceeded stale window")
	if err != nil {
		return report, err
	}
	report.StuckJobsFailed = stuck

	repos, err := s.repos.List(ctx)
	if err != nil {
		return report, fmt.Errorf("orchestrator survey: list repos: %w", err)
	}
	if s.cfg.RepoScope != "" {
		repos = filterReposByPath(repos, s.cfg.RepoScope)
	}

	for _, rec := range repos {
		manifest, err := bundle.Load(rec.Path)
		if err != nil {
			slog.Warn("serve: orchestrator survey skipped repo; manifest load failed", "repo_id", rec.ID, "err", err)
			continue
		}
		report.ReposTriaged++
		report.JobsRouted += s.surveyTicketState(ctx, rec, manifest, source)
		report.JobsRouted += s.surveyRecentOutcomes(ctx, rec, manifest, source)
		report.TicketsTriaged += s.surveyTelemetryPatterns(ctx, rec, source)
		report.TicketsTriaged += s.surveyLowScores(ctx, rec, source)
	}

	return report, nil
}

func (s *Server) surveyTicketState(ctx context.Context, rec RepoRecord, manifest *bundle.Manifest, source string) int {
	routed := 0

	eligible, err := ticketstate.EligibleInProgress(rec.Path)
	if err != nil {
		slog.Warn("serve: orchestrator survey ticket ownership skipped", "repo_id", rec.ID, "err", err)
	} else if len(eligible) > 0 && hasManifestRole(manifest, "engineer") {
		t := eligible[0]
		if s.enqueueSurveyJob(ctx, rec, "engineer", "ticket_delivery", surveyJobSpec{
			Signal:           "eligible_in_progress_ticket",
			Reason:           "eligible in-progress ticket needs an owning engineer workspace",
			Tickets:          []ticketstate.Ticket{t},
			Source:           source,
			IdempotencyKey:   fmt.Sprintf("survey:ticket-owner:%s:%s", rec.ID, ticketKey(t)),
			ConcurrencyGroup: fmt.Sprintf("ticket:%s:%s", rec.ID, ticketKey(t)),
			DailyCap:         2,
		}) {
			routed++
		}
	}

	stale, err := ticketstate.StaleInProgress(rec.Path, time.Now().UTC(), ticketstate.DefaultStaleInProgressAfter)
	if err != nil {
		slog.Warn("serve: orchestrator survey stale ticket check skipped", "repo_id", rec.ID, "err", err)
	} else if len(stale) > 0 && hasManifestRole(manifest, "janitor") {
		if s.enqueueSurveyJob(ctx, rec, "janitor", "ticket_hygiene", surveyJobSpec{
			Signal:           "stale_in_progress_ticket",
			Reason:           "stale eligible in-progress tickets need janitor reconciliation",
			Tickets:          stale,
			Source:           source,
			IdempotencyKey:   fmt.Sprintf("survey:stale-ticket:%s", rec.ID),
			ConcurrencyGroup: fmt.Sprintf("ticket:%s:stale-in-progress", rec.ID),
			DailyCap:         orchestratorSurveyDailyCap,
		}) {
			routed++
		}
	}

	all, err := ticketstate.List(rec.Path)
	if err != nil {
		slog.Warn("serve: orchestrator survey ticket index skipped", "repo_id", rec.ID, "err", err)
		return routed
	}

	var blocked []ticketstate.Ticket
	for _, t := range all {
		if t.Status == ticketstate.StatusInProgress && t.Blocked() {
			blocked = append(blocked, t)
		}
	}
	if len(blocked) > 0 && hasManifestRole(manifest, "janitor") {
		if s.enqueueSurveyJob(ctx, rec, "janitor", "ticket_hygiene", surveyJobSpec{
			Signal:           "blocked_in_progress_ticket",
			Reason:           "blocked in-progress tickets need explicit dependency or resume evidence",
			Tickets:          blocked,
			Source:           source,
			IdempotencyKey:   fmt.Sprintf("survey:blocked-ticket:%s", rec.ID),
			ConcurrencyGroup: fmt.Sprintf("ticket:%s:blocked-in-progress", rec.ID),
			DailyCap:         1,
		}) {
			routed++
		}
	}

	if len(eligible) == 0 && hasManifestRole(manifest, "engineer") {
		if t, ok := firstBacklogInterventionDebt(all); ok {
			if s.enqueueSurveyJob(ctx, rec, "engineer", "intervention_debt", surveyJobSpec{
				Signal:           "intervention_debt_backlog",
				Reason:           "intervention-debt backlog is prioritized ahead of ordinary backlog",
				Tickets:          []ticketstate.Ticket{t},
				Source:           source,
				IdempotencyKey:   fmt.Sprintf("survey:intervention-debt:%s:%s", rec.ID, ticketKey(t)),
				ConcurrencyGroup: fmt.Sprintf("ticket:%s:%s", rec.ID, ticketKey(t)),
				DailyCap:         1,
			}) {
				routed++
			}
		}
	}

	return routed
}

func (s *Server) surveyRecentOutcomes(ctx context.Context, rec RepoRecord, manifest *bundle.Manifest, source string) int {
	if s.scoreStore == nil {
		return 0
	}
	counts, err := s.scoreStore.OutcomeCounts(ctx, rec.ID, time.Now().UTC().Add(-orchestratorSurveyEvidenceWindow))
	if err != nil {
		slog.Warn("serve: orchestrator survey outcome scan failed", "repo_id", rec.ID, "err", err)
		return 0
	}

	routed := 0
	for _, count := range counts {
		switch {
		case count.Type == scoring.OutcomeChecksFailed && hasManifestRole(manifest, "pipeline-fixer"):
			if s.enqueueSurveyJob(ctx, rec, "pipeline-fixer", "pipeline_repair", surveyJobSpec{
				Signal:           "failed_check",
				Reason:           "recent failed checks need deterministic pipeline repair",
				SourceRole:       count.Role,
				Outcome:          string(count.Type),
				Count:            count.Count,
				Source:           source,
				IdempotencyKey:   fmt.Sprintf("survey:failed-check:%s:%s", rec.ID, count.Role),
				ConcurrencyGroup: fmt.Sprintf("checks:%s:%s", rec.ID, count.Role),
				DailyCap:         2,
			}) {
				routed++
			}
		case count.Role == "dogfood" && isDogfoodFailureOutcome(count.Type) && hasManifestRole(manifest, "engineer"):
			if s.enqueueSurveyJob(ctx, rec, "engineer", "dogfood_failure", surveyJobSpec{
				Signal:           "dogfood_failure",
				Reason:           "recent dogfood failure should become reproducible engineering work",
				SourceRole:       count.Role,
				Outcome:          string(count.Type),
				Count:            count.Count,
				Source:           source,
				IdempotencyKey:   fmt.Sprintf("survey:dogfood-failure:%s", rec.ID),
				ConcurrencyGroup: fmt.Sprintf("dogfood:%s", rec.ID),
				DailyCap:         2,
			}) {
				routed++
			}
		case count.Type == scoring.OutcomeNoop && hasManifestRole(manifest, "janitor"):
			if s.enqueueSurveyJob(ctx, rec, "janitor", "ticket_hygiene", surveyJobSpec{
				Signal:           "silent_noop",
				Reason:           "recent no-op outcomes need ticket and plan state reconciliation",
				SourceRole:       count.Role,
				Outcome:          string(count.Type),
				Count:            count.Count,
				Source:           source,
				IdempotencyKey:   fmt.Sprintf("survey:no-op:%s:%s", rec.ID, count.Role),
				ConcurrencyGroup: fmt.Sprintf("noop:%s:%s", rec.ID, count.Role),
				DailyCap:         orchestratorSurveyDailyCap,
			}) {
				routed++
			}
		}
	}
	return routed
}

func (s *Server) surveyTelemetryPatterns(ctx context.Context, rec RepoRecord, source string) int {
	if s.telemetry == nil {
		return 0
	}
	patterns := s.telemetry.DetectPatternsFromStore()
	triaged := 0
	for _, pattern := range patterns {
		if pattern.RepoID != "" && pattern.RepoID != rec.ID {
			continue
		}
		proposal := telemetry.TriagePattern(pattern)
		if proposal.RepoID == "" {
			proposal.RepoID = rec.ID
		}
		origin := interventionDebtOrigin{
			Kind:           "orchestrator_survey_telemetry_pattern",
			EvidenceWindow: defaultString(pattern.Window, "24h"),
		}
		if s.telemStore != nil {
			evt, err := s.telemStore.LatestByRoleCategory(proposal.RepoID, proposal.Role, proposal.Category, time.Now().UTC().Add(-telemetry.PatternWindow))
			if err != nil {
				slog.Warn("serve: orchestrator survey telemetry evidence lookup failed", "repo_id", rec.ID, "role", proposal.Role, "category", proposal.Category, "err", err)
			} else {
				origin.Event = evt
			}
		}
		s.recordInterventionDebtTicket(ctx, rec.ID, proposal, origin)
		s.offerInterventionDebtEvolution(ctx, rec.ID, proposal, fmt.Sprintf("survey_telemetry_%s_%s", proposal.Target, proposal.Category))
		triaged++
	}
	return triaged
}

func (s *Server) surveyLowScores(ctx context.Context, rec RepoRecord, source string) int {
	if s.scoreStore == nil {
		return 0
	}
	scores, err := s.scoreStore.ListScores(ctx)
	if err != nil {
		slog.Warn("serve: orchestrator survey score scan failed", "repo_id", rec.ID, "err", err)
		return 0
	}
	triaged := 0
	for _, score := range scores {
		if score.RepoID != rec.ID {
			continue
		}
		proposal, ok := telemetry.TriageScore(telemetry.ScoreSnapshot{
			Role:       score.Role,
			RepoID:     score.RepoID,
			Value:      score.Value,
			SampleSize: score.SampleSize,
			WindowDays: score.WindowDays,
		})
		if !ok {
			continue
		}
		s.recordInterventionDebtTicket(ctx, rec.ID, proposal, interventionDebtOrigin{
			Kind:           "orchestrator_survey_score_snapshot",
			EvidenceWindow: fmt.Sprintf("%dd", score.WindowDays),
			Score: &telemetry.ScoreSnapshot{
				Role:       score.Role,
				RepoID:     score.RepoID,
				Value:      score.Value,
				SampleSize: score.SampleSize,
				WindowDays: score.WindowDays,
			},
			Message: source,
		})
		s.offerInterventionDebtEvolution(ctx, rec.ID, proposal, fmt.Sprintf("survey_score_%s", proposal.Target))
		triaged++
	}
	return triaged
}

type surveyJobSpec struct {
	Signal           string
	Reason           string
	Source           string
	SourceRole       string
	Outcome          string
	Count            int
	Tickets          []ticketstate.Ticket
	IdempotencyKey   string
	ConcurrencyGroup string
	DailyCap         int
}

func (s *Server) enqueueSurveyJob(ctx context.Context, rec RepoRecord, role, payloadMode string, spec surveyJobSpec) bool {
	if spec.DailyCap <= 0 {
		spec.DailyCap = orchestratorSurveyDailyCap
	}
	payload := map[string]any{
		"type":         "orchestrator.survey",
		"source":       defaultString(spec.Source, "survey"),
		"signal":       spec.Signal,
		"reason":       spec.Reason,
		"payload_mode": payloadMode,
	}
	if spec.SourceRole != "" {
		payload["source_role"] = spec.SourceRole
	}
	if spec.Outcome != "" {
		payload["outcome"] = spec.Outcome
	}
	if spec.Count > 0 {
		payload["count"] = spec.Count
	}
	if len(spec.Tickets) > 0 {
		payload["tickets"] = surveyTicketPayload(spec.Tickets)
	}
	trigger, _ := json.Marshal(payload)

	jobID, err := s.queue.Enqueue(ctx, queue.Job{
		RepoID:           rec.ID,
		Role:             role,
		Trigger:          string(trigger),
		PayloadMode:      payloadMode,
		ConcurrencyGroup: spec.ConcurrencyGroup,
		DailyCap:         spec.DailyCap,
		IdempotencyKey:   spec.IdempotencyKey,
	})
	if err != nil {
		slog.Warn("serve: orchestrator survey enqueue failed",
			"repo_id", rec.ID,
			"role", role,
			"signal", spec.Signal,
			"err", err,
		)
		return false
	}
	slog.Info("serve: orchestrator survey job routed",
		"repo_id", rec.ID,
		"role", role,
		"signal", spec.Signal,
		"job_id", jobID,
		"group", spec.ConcurrencyGroup,
	)
	return true
}

func surveyTicketPayload(tickets []ticketstate.Ticket) []map[string]string {
	out := make([]map[string]string, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, map[string]string{
			"id":            t.ID,
			"path":          t.RelPath,
			"title":         t.Title,
			"status":        t.Status,
			"kind":          t.Kind,
			"last_activity": t.LastActivityLabel(),
			"next_action":   t.NextAction,
		})
	}
	return out
}

func firstBacklogInterventionDebt(tickets []ticketstate.Ticket) (ticketstate.Ticket, bool) {
	for _, t := range tickets {
		if t.Status == ticketstate.StatusBacklog && t.Kind == "intervention-debt" {
			return t, true
		}
	}
	return ticketstate.Ticket{}, false
}

func ticketKey(t ticketstate.Ticket) string {
	key := strings.TrimSpace(t.ID)
	if key == "" {
		key = strings.TrimSpace(t.Name)
	}
	if key == "" {
		key = "unknown-ticket"
	}
	return normalizeDedupePart(key)
}

func hasManifestRole(manifest *bundle.Manifest, role string) bool {
	if manifest == nil {
		return false
	}
	_, ok := manifest.Roles[role]
	return ok
}

func isDogfoodFailureOutcome(outcome scoring.OutcomeType) bool {
	switch outcome {
	case scoring.OutcomeFailed, scoring.OutcomeChecksFailed, scoring.OutcomeTimeout, scoring.OutcomeNoop:
		return true
	default:
		return false
	}
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
