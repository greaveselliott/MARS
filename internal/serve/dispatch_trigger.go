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
	TicketID      string            `json:"ticket_id,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	EvidenceLinks []string          `json:"evidence_links,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	Handoff       orgstate.Handoff  `json:"handoff,omitempty"`
	Feedback      orgstate.Feedback `json:"feedback,omitempty"`
}

func newDispatchTriggerPayload(job *queue.Job, decision orgstate.Decision, disposition orgstate.Disposition) dispatchTriggerPayload {
	return dispatchTriggerPayload{
		Type:           "dispatch",
		SourceRole:     job.Role,
		SourceJob:      job.ID,
		DecisionID:     decision.ID,
		DecisionKind:   decision.DecisionKind,
		DecisionReason: decision.Reason,
		TargetRole:     decision.NextRole,
		SourceDisposition: dispatchTriggerDisposition{
			Status:        disposition.Status,
			NextNeed:      disposition.NextNeed,
			TicketID:      disposition.TicketID,
			Reason:        disposition.Reason,
			EvidenceLinks: append([]string{}, disposition.EvidenceLinks...),
			TraceID:       disposition.TraceID,
			Handoff:       disposition.Handoff,
			Feedback:      disposition.Feedback,
		},
	}
}
