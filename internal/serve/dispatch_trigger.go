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
	"encoding/json"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/greaveselliott/mars-harness/internal/queue"
)

type dispatchTriggerPayload struct {
	Type              string                     `json:"type"`
	SourceRole        string                     `json:"source_role"`
	SourceJob         string                     `json:"source_job"`
	DecisionID        string                     `json:"decision_id"`
	DecisionKind      string                     `json:"decision_kind"`
	DecisionReason    string                     `json:"decision_reason"`
	TargetRole        string                     `json:"target_role"`
	SourceDisposition dispatchTriggerDisposition `json:"source_disposition"`
}

type dispatchTriggerDisposition struct {
	Status        string            `json:"status"`
	NextNeed      string            `json:"next_need,omitempty"`
	SuggestedRole string            `json:"suggested_role,omitempty"`
	TicketID      string            `json:"ticket_id,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	EvidenceLinks []string          `json:"evidence_links,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	Handoff       orgstate.Handoff  `json:"handoff,omitempty"`
	Feedback      orgstate.Feedback `json:"feedback,omitempty"`
}

func newDispatchTriggerPayload(job *queue.Job, decision orgstate.Decision, disposition orgstate.Disposition) dispatchTriggerPayload {
	return newDispatchTriggerPayloadForSource(job.Role, job.ID, decision, disposition)
}

func newDispatchTriggerPayloadForSource(sourceRole, sourceJob string, decision orgstate.Decision, disposition orgstate.Disposition) dispatchTriggerPayload {
	return dispatchTriggerPayload{
		Type:           "dispatch",
		SourceRole:     sourceRole,
		SourceJob:      sourceJob,
		DecisionID:     decision.ID,
		DecisionKind:   decision.DecisionKind,
		DecisionReason: decision.Reason,
		TargetRole:     decision.NextRole,
		SourceDisposition: dispatchTriggerDisposition{
			Status:        disposition.Status,
			NextNeed:      disposition.NextNeed,
			SuggestedRole: disposition.SuggestedRole,
			TicketID:      disposition.TicketID,
			Reason:        disposition.Reason,
			EvidenceLinks: append([]string{}, disposition.EvidenceLinks...),
			TraceID:       disposition.TraceID,
			Handoff:       disposition.Handoff,
			Feedback:      disposition.Feedback,
		},
	}
}

func sourceDispositionFromDispatchTrigger(repoID, raw string) (orgstate.Disposition, bool) {
	var payload dispatchTriggerPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return orgstate.Disposition{}, false
	}
	if payload.Type != "dispatch" ||
		strings.TrimSpace(payload.SourceRole) == "" ||
		strings.TrimSpace(payload.SourceJob) == "" {
		return orgstate.Disposition{}, false
	}
	status := strings.TrimSpace(payload.SourceDisposition.Status)
	if status == "" {
		status = "completed"
	}
	return orgstate.Disposition{
		JobID:         strings.TrimSpace(payload.SourceJob),
		RepoID:        repoID,
		Role:          strings.TrimSpace(payload.SourceRole),
		Status:        status,
		NextNeed:      strings.TrimSpace(payload.SourceDisposition.NextNeed),
		SuggestedRole: strings.TrimSpace(payload.SourceDisposition.SuggestedRole),
		TicketID:      strings.TrimSpace(payload.SourceDisposition.TicketID),
		Reason:        strings.TrimSpace(payload.SourceDisposition.Reason),
		EvidenceLinks: append([]string{}, payload.SourceDisposition.EvidenceLinks...),
		TraceID:       strings.TrimSpace(payload.SourceDisposition.TraceID),
		Handoff:       payload.SourceDisposition.Handoff,
		Feedback:      payload.SourceDisposition.Feedback,
	}, true
}
