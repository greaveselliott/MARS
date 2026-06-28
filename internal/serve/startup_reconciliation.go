/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/greaveselliott/mars/internal/bundle"
	"github.com/greaveselliott/mars/internal/orchestration"
	"github.com/greaveselliott/mars/internal/orgstate"
	"github.com/greaveselliott/mars/internal/queue"
	ticketstate "github.com/greaveselliott/mars/internal/tickets"
)

type StartupAction string

const (
	StartupActionSeededCEO          StartupAction = "seeded_ceo"
	StartupActionResumedLifecycle   StartupAction = "resumed_lifecycle"
	StartupActionRecoveredStaleJob  StartupAction = "recovered_stale_job"
	StartupActionRoutedExistingWork StartupAction = "routed_existing_ticket"
	StartupActionRefusedAmbiguous   StartupAction = "refused_ambiguous_state"
)

type StartupReconciliation struct {
	Action     StartupAction
	Evidence   []string
	JobID      string
	Role       string
	ShouldSeed bool
}

func (r StartupReconciliation) Summary() string {
	action := strings.TrimSpace(string(r.Action))
	if action == "" {
		action = string(StartupActionRefusedAmbiguous)
	}
	var parts []string
	parts = append(parts, "startup_action="+action)
	if r.Role != "" {
		parts = append(parts, "role="+r.Role)
	}
	if r.JobID != "" {
		parts = append(parts, "job="+r.JobID)
	}
	if len(r.Evidence) > 0 {
		parts = append(parts, "evidence="+strings.Join(r.Evidence, "; "))
	}
	return strings.Join(parts, " ")
}

// ReconcileStartup decides whether `mars start --repo` should seed a
// new CEO job or resume existing lifecycle state. It runs before Server.Start
// so bootstrap seeding cannot get ahead of stale-job and ticket recovery.
func (s *Server) ReconcileStartup(ctx context.Context, repoID, repoPath string, forceNewLifecycle bool) (StartupReconciliation, error) {
	if forceNewLifecycle {
		return StartupReconciliation{
			Action:     StartupActionSeededCEO,
			Evidence:   []string{"--new-lifecycle requested"},
			Role:       "ceo",
			ShouldSeed: true,
		}, nil
	}

	reset, err := s.queue.ResetOrphansForRepo(ctx, repoID, "interrupted by previous process before startup reconciliation")
	if err != nil {
		return StartupReconciliation{}, err
	}
	active, err := s.queue.ActiveJobsForRepo(ctx, repoID, 20)
	if err != nil {
		return StartupReconciliation{}, err
	}
	if len(active) > 0 {
		job := active[0]
		return StartupReconciliation{
			Action:   startupActionForRecovered(reset, StartupActionResumedLifecycle),
			Evidence: []string{fmt.Sprintf("active %s job %s already queued", job.Role, job.ID)},
			JobID:    job.ID,
			Role:     job.Role,
		}, nil
	}

	if rec, ok, err := s.reconcileReviewRework(ctx, repoID); err != nil || ok {
		if err != nil {
			return StartupReconciliation{}, err
		}
		if reset > 0 && rec.Action == StartupActionRoutedExistingWork {
			rec.Action = StartupActionRecoveredStaleJob
			rec.Evidence = append([]string{fmt.Sprintf("recovered %d stale job(s)", reset)}, rec.Evidence...)
		}
		return rec, nil
	}

	if rec, ok, err := s.reconcileOpenTickets(ctx, repoID, repoPath); err != nil || ok {
		if err != nil {
			return StartupReconciliation{}, err
		}
		if reset > 0 && rec.Action == StartupActionRoutedExistingWork {
			rec.Action = StartupActionRecoveredStaleJob
			rec.Evidence = append([]string{fmt.Sprintf("recovered %d stale job(s)", reset)}, rec.Evidence...)
		}
		return rec, nil
	}

	if reason, ok := repoHasUncommittedSurveyBlocker(ctx, repoPath); ok {
		return StartupReconciliation{
			Action:   StartupActionRefusedAmbiguous,
			Evidence: []string{fmt.Sprintf("dirty workspace without deterministic ticket route: %s", reason)},
		}, nil
	}

	if rec, ok, err := s.reconcileRecentDisposition(ctx, repoID, repoPath); err != nil || ok {
		if err != nil {
			return StartupReconciliation{}, err
		}
		if reset > 0 && rec.Action == StartupActionResumedLifecycle {
			rec.Action = StartupActionRecoveredStaleJob
			rec.Evidence = append([]string{fmt.Sprintf("recovered %d stale job(s)", reset)}, rec.Evidence...)
		}
		return rec, nil
	}

	if reset > 0 {
		return StartupReconciliation{
			Action:   StartupActionRefusedAmbiguous,
			Evidence: []string{fmt.Sprintf("recovered %d stale job(s) but found no deterministic ticket or disposition route", reset)},
		}, nil
	}

	return StartupReconciliation{
		Action:     StartupActionSeededCEO,
		Evidence:   []string{"no active jobs, tickets, or dispositions found"},
		Role:       "ceo",
		ShouldSeed: true,
	}, nil
}

func startupActionForRecovered(recovered int, otherwise StartupAction) StartupAction {
	if recovered > 0 {
		return StartupActionRecoveredStaleJob
	}
	return otherwise
}

func (s *Server) reconcileOpenTickets(ctx context.Context, repoID, repoPath string) (StartupReconciliation, bool, error) {
	inProgress, err := ticketstate.EligibleInProgress(repoPath)
	if err != nil {
		return StartupReconciliation{}, false, fmt.Errorf("startup reconciliation: inspect in-progress tickets: %w", err)
	}
	if len(inProgress) > 0 {
		t := inProgress[0]
		jobID, err := s.enqueueStartupRole(ctx, repoID, "engineer", "routed_existing_ticket", t.ID, map[string]string{
			"signal":      "eligible_in_progress_ticket",
			"ticket_id":   t.ID,
			"ticket_path": t.RelPath,
			"ask":         "Resume the active in-progress ticket from the current repository state; do not restart planning.",
		})
		if err != nil {
			return StartupReconciliation{}, false, err
		}
		return StartupReconciliation{
			Action:   StartupActionRoutedExistingWork,
			Evidence: []string{fmt.Sprintf("in-progress ticket %s at %s", t.ID, t.RelPath)},
			JobID:    jobID,
			Role:     "engineer",
		}, true, nil
	}

	inReview, err := ticketstate.ListStatus(repoPath, ticketstate.StatusInReview)
	if err != nil {
		return StartupReconciliation{}, false, fmt.Errorf("startup reconciliation: inspect in-review tickets: %w", err)
	}
	if len(inReview) > 0 {
		t := inReview[0]
		jobID, err := s.enqueueStartupRole(ctx, repoID, "qa", "resumed_lifecycle", t.ID, map[string]string{
			"signal":      "in_review_ticket",
			"ticket_id":   t.ID,
			"ticket_path": t.RelPath,
			"ask":         "Resume review for the in-review ticket from the current repository state.",
		})
		if err != nil {
			return StartupReconciliation{}, false, err
		}
		return StartupReconciliation{
			Action:   StartupActionResumedLifecycle,
			Evidence: []string{fmt.Sprintf("in-review ticket %s at %s", t.ID, t.RelPath)},
			JobID:    jobID,
			Role:     "qa",
		}, true, nil
	}
	return StartupReconciliation{}, false, nil
}

func (s *Server) reconcileReviewRework(ctx context.Context, repoID string) (StartupReconciliation, bool, error) {
	if s.orgStore == nil {
		return StartupReconciliation{}, false, nil
	}
	recent, err := s.orgStore.RecentDispositions(ctx, repoID, 10)
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	for _, d := range recent {
		if !isEngineerReworkDisposition(d) {
			continue
		}
		decision := orgstate.Decision{
			JobID:        d.JobID,
			RepoID:       repoID,
			SourceRole:   d.Role,
			TicketID:     d.TicketID,
			NextNeed:     d.NextNeed,
			NextRole:     "engineer",
			DecisionKind: "startup_reconciliation",
			Reason:       "resuming pinned implementation rework from latest review disposition",
		}
		recorded, err := s.orgStore.RecordDecision(ctx, decision)
		if err != nil {
			return StartupReconciliation{}, false, err
		}
		trigger, err := json.Marshal(newDispatchTriggerPayloadForSource(d.Role, d.JobID, recorded, d))
		if err != nil {
			return StartupReconciliation{}, false, err
		}
		jobID, err := s.queue.Enqueue(ctx, queue.Job{
			RepoID:           repoID,
			Role:             "engineer",
			Trigger:          string(trigger),
			IdempotencyKey:   fmt.Sprintf("startup:rework:%s:%s:%s", repoID, d.TicketID, d.JobID),
			ConcurrencyGroup: fmt.Sprintf("ticket:%s:%s", repoID, d.TicketID),
		})
		if err != nil {
			return StartupReconciliation{}, false, err
		}
		return StartupReconciliation{
			Action:   StartupActionRoutedExistingWork,
			Evidence: []string{fmt.Sprintf("changes_requested pinned ticket %s from %s", d.TicketID, d.Role)},
			JobID:    jobID,
			Role:     "engineer",
		}, true, nil
	}
	return StartupReconciliation{}, false, nil
}

func isEngineerReworkDisposition(d orgstate.Disposition) bool {
	return strings.EqualFold(strings.TrimSpace(d.Status), "changes_requested") &&
		strings.EqualFold(strings.TrimSpace(d.NextNeed), "implementation_rework") &&
		strings.EqualFold(strings.TrimSpace(d.Feedback.ForRole), "engineer") &&
		strings.TrimSpace(d.TicketID) != ""
}

func isPinnedEngineerReworkSource(d orgstate.Disposition) bool {
	if !strings.EqualFold(strings.TrimSpace(d.Status), "changes_requested") ||
		!strings.EqualFold(strings.TrimSpace(d.NextNeed), "implementation_rework") ||
		strings.TrimSpace(d.TicketID) == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(d.Feedback.ForRole), "engineer") ||
		strings.EqualFold(strings.TrimSpace(d.SuggestedRole), "engineer") ||
		strings.EqualFold(strings.TrimSpace(d.Handoff.TargetRole), "engineer") {
		return true
	}
	return false
}

func (s *Server) reconcileRecentDisposition(ctx context.Context, repoID, repoPath string) (StartupReconciliation, bool, error) {
	if s.orgStore == nil {
		return StartupReconciliation{}, false, nil
	}
	recent, err := s.orgStore.RecentDispositions(ctx, repoID, 5)
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	if len(recent) == 0 {
		return StartupReconciliation{}, false, nil
	}
	manifest, err := bundle.Load(repoPath)
	if err != nil {
		return StartupReconciliation{}, false, fmt.Errorf("startup reconciliation: load manifest: %w", err)
	}
	snap, snapErr := snapshotTickets(repoPath)
	ticketHash := ""
	if snapErr == nil {
		ticketHash = snap.routingHash()
	}
	decisions, err := s.orgStore.RecentDecisions(ctx, repoID, 20)
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	latest := recent[0]
	decision, err := orchestration.Decide(orchestration.Input{
		Disposition:     latest,
		Manifest:        manifest,
		RecentDecisions: decisions,
		TicketStateHash: ticketHash,
	})
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	if snapErr == nil {
		decision = enforceEngineerTicketPrerequisite(decision, snap, manifest, &latest)
		decision = enforceReleaseRequiresCompletedFeatureScenarios(decision, snap, manifest, repoPath)
	}
	if strings.TrimSpace(decision.NextRole) == "" {
		return StartupReconciliation{
			Action:   StartupActionRefusedAmbiguous,
			Evidence: []string{fmt.Sprintf("latest disposition %s/%s has no deterministic next role: %s", latest.Role, latest.Status, decision.StopReason)},
		}, true, nil
	}
	recorded, err := s.orgStore.RecordDecision(ctx, decision)
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	trigger, err := json.Marshal(newDispatchTriggerPayloadForSource(latest.Role, latest.JobID, recorded, latest))
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	jobID, err := s.queue.Enqueue(ctx, queue.Job{
		RepoID:         repoID,
		Role:           recorded.NextRole,
		Trigger:        string(trigger),
		IdempotencyKey: fmt.Sprintf("startup:resume:%s:%s:%s", repoID, latest.JobID, recorded.NextRole),
	})
	if err != nil {
		return StartupReconciliation{}, false, err
	}
	return StartupReconciliation{
		Action:   StartupActionResumedLifecycle,
		Evidence: []string{fmt.Sprintf("latest disposition %s/%s routed by %s", latest.Role, latest.Status, recorded.Reason)},
		JobID:    jobID,
		Role:     recorded.NextRole,
	}, true, nil
}

func (s *Server) enqueueStartupRole(ctx context.Context, repoID, role, signal, ticketID string, payload map[string]string) (string, error) {
	payload["type"] = "startup_reconciliation"
	payload["source"] = "mars start"
	payload["target_role"] = role
	trigger, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	keyParts := []string{"startup", signal, repoID, role}
	if strings.TrimSpace(ticketID) != "" {
		keyParts = append(keyParts, ticketID)
	}
	return s.queue.Enqueue(ctx, queue.Job{
		RepoID:           repoID,
		Role:             role,
		Trigger:          string(trigger),
		IdempotencyKey:   strings.Join(keyParts, ":"),
		ConcurrencyGroup: startupConcurrencyGroup(repoID, ticketID),
	})
}

func startupConcurrencyGroup(repoID, ticketID string) string {
	if strings.TrimSpace(ticketID) == "" {
		return ""
	}
	return fmt.Sprintf("ticket:%s:%s", repoID, ticketID)
}
