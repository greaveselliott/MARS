/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-012-self-improvement-loop.md
*/
package serve

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/remediation"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
)

type remediationAttemptEvidence struct {
	RecipeID   string   `json:"recipe_id"`
	Status     string   `json:"status"`
	Safety     string   `json:"safety"`
	Reason     string   `json:"reason,omitempty"`
	Commands   []string `json:"commands,omitempty"`
	NextAction string   `json:"next_action,omitempty"`
}

type remediationPlanEvidence struct {
	Error    string                       `json:"error,omitempty"`
	TraceID  string                       `json:"trace_id,omitempty"`
	Attempts []remediationAttemptEvidence `json:"remediation_attempts,omitempty"`
}

func (s *Server) planJobFailureRemediation(ctx context.Context, job *queue.Job, cat telemetry.FailureCategory, msg, traceID string, broadcast bool) remediation.Plan {
	if job == nil {
		return remediation.Plan{}
	}
	return s.planFailureRemediation(ctx, remediation.Signal{
		Category: cat,
		Message:  msg,
		Role:     job.Role,
		RepoPath: s.repoPathForRemediation(ctx, job.RepoID),
		Phase:    "job_failed",
	}, job.ID, job.RepoID, traceID, broadcast)
}

func (s *Server) planEventRemediation(ctx context.Context, evt telemetry.Event, traceID string, broadcast bool) remediation.Plan {
	return s.planFailureRemediation(ctx, remediation.Signal{
		Category: evt.Category,
		Message:  evt.Message,
		Role:     evt.Role,
		RepoPath: s.repoPathForRemediation(ctx, evt.RepoID),
		Phase:    "telemetry_remediation",
	}, evt.JobID, evt.RepoID, traceID, broadcast)
}

func (s *Server) planFailureRemediation(_ context.Context, signal remediation.Signal, jobID, repoID, traceID string, broadcast bool) remediation.Plan {
	registry := s.remediators
	if len(registry.List()) == 0 {
		registry = remediation.DefaultRegistry()
	}
	plan := registry.Plan(signal)
	if len(plan.Attempts) == 0 {
		return plan
	}

	slog.Info("serve: deterministic remediation plan selected",
		"job_id", jobID,
		"repo_id", repoID,
		"role", signal.Role,
		"category", signal.Category,
		"attempts", len(plan.Attempts),
	)
	if broadcast && s.dash != nil {
		payload, err := json.Marshal(map[string]interface{}{
			"job_id":    jobID,
			"repo_id":   repoID,
			"role":      signal.Role,
			"category":  string(signal.Category),
			"phase":     signal.Phase,
			"trace_id":  traceID,
			"attempts":  remediationAttemptEvidenceList(plan.Attempts),
			"repo_path": signal.RepoPath,
		})
		if err == nil {
			s.dash.BroadcastEvent("remediation_plan", string(payload))
		}
	}
	return plan
}

func (s *Server) repoPathForRemediation(ctx context.Context, repoID string) string {
	if s == nil || s.repos == nil || strings.TrimSpace(repoID) == "" {
		return ""
	}
	rec, err := s.repos.FindByID(ctx, repoID)
	if err != nil || rec == nil {
		return ""
	}
	return rec.Path
}

func remediationOutcomeDetails(errMsg, traceID string, plan remediation.Plan) string {
	evidence := remediationPlanEvidence{
		Error:    strings.TrimSpace(errMsg),
		TraceID:  strings.TrimSpace(traceID),
		Attempts: remediationAttemptEvidenceList(plan.Attempts),
	}
	data, err := json.Marshal(evidence)
	if err != nil {
		return strings.TrimSpace(errMsg)
	}
	return string(data)
}

func remediationAttemptEvidenceList(attempts []remediation.Attempt) []remediationAttemptEvidence {
	if len(attempts) == 0 {
		return nil
	}
	out := make([]remediationAttemptEvidence, 0, len(attempts))
	for _, attempt := range attempts {
		out = append(out, remediationAttemptEvidence{
			RecipeID:   attempt.RecipeID,
			Status:     string(attempt.Status),
			Safety:     string(attempt.Safety),
			Reason:     attempt.Reason,
			Commands:   append([]string(nil), attempt.Commands...),
			NextAction: attempt.NextAction,
		})
	}
	return out
}

func remediationPlanHasReadyAttempt(plan remediation.Plan) bool {
	for _, attempt := range plan.Attempts {
		if attempt.Status == remediation.AttemptReady {
			return true
		}
	}
	return false
}
