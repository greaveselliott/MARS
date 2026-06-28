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

	"github.com/greaveselliott/mars/internal/queue"
	"github.com/greaveselliott/mars/internal/remediation"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/telemetry"
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
	Error      string                         `json:"error,omitempty"`
	TraceID    string                         `json:"trace_id,omitempty"`
	Attempts   []remediationAttemptEvidence   `json:"remediation_attempts,omitempty"`
	Executions []remediationExecutionEvidence `json:"remediation_executions,omitempty"`
}

type remediationExecutionEvidence struct {
	RecipeID     string   `json:"recipe_id"`
	Status       string   `json:"status"`
	Command      string   `json:"command,omitempty"`
	UpdatedFiles []string `json:"updated_files,omitempty"`
	Error        string   `json:"error,omitempty"`
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

func (s *Server) executeReadyRemediation(ctx context.Context, plan remediation.Plan) []remediationExecutionEvidence {
	var executions []remediationExecutionEvidence
	for _, attempt := range plan.Attempts {
		if attempt.Status != remediation.AttemptReady {
			continue
		}
		if remediationAttemptHasExecutor(attempt) {
			executions = append(executions, s.executeGeneratedDocsUpdate(ctx, plan.Signal, attempt))
			continue
		}
		executions = append(executions, remediationExecutionEvidence{
			RecipeID: attempt.RecipeID,
			Status:   "skipped_no_executor",
			Error:    "no deterministic executor is registered for this auto-safe recipe",
		})
	}
	return executions
}

func (s *Server) executeGeneratedDocsUpdate(ctx context.Context, signal remediation.Signal, attempt remediation.Attempt) remediationExecutionEvidence {
	evidence := remediationExecutionEvidence{
		RecipeID: attempt.RecipeID,
		Status:   "failed",
	}
	if len(attempt.Commands) > 0 {
		evidence.Command = attempt.Commands[0]
	}
	if err := ctx.Err(); err != nil {
		evidence.Error = err.Error()
		return evidence
	}
	if strings.TrimSpace(signal.RepoPath) == "" {
		evidence.Error = "repo path unavailable for generated harness update"
		return evidence
	}
	updated, err := scanner.Upgrade(signal.RepoPath)
	if err != nil {
		evidence.Error = err.Error()
		return evidence
	}
	evidence.UpdatedFiles = updated
	if len(updated) == 0 {
		evidence.Status = "noop"
	} else {
		evidence.Status = "applied"
	}
	if s.dash != nil {
		payload, err := json.Marshal(evidence)
		if err == nil {
			s.dash.BroadcastEvent("remediation_execution", string(payload))
		}
	}
	return evidence
}

func remediationOutcomeDetails(errMsg, traceID string, plan remediation.Plan, executions []remediationExecutionEvidence) string {
	evidence := remediationPlanEvidence{
		Error:      strings.TrimSpace(errMsg),
		TraceID:    strings.TrimSpace(traceID),
		Attempts:   remediationAttemptEvidenceList(plan.Attempts),
		Executions: executions,
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

func remediationPlanHasExecutableReadyAttempt(plan remediation.Plan) bool {
	for _, attempt := range plan.Attempts {
		if attempt.Status == remediation.AttemptReady && remediationAttemptHasExecutor(attempt) {
			return true
		}
	}
	return false
}

func remediationAttemptHasExecutor(attempt remediation.Attempt) bool {
	return attempt.RecipeID == "generated-docs:update-missing-defaults"
}
