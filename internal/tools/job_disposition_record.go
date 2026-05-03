package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

const jobDispositionRecordSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "status": { "type": "string", "enum": ["completed", "approved", "blocked", "in_review", "changes_requested", "no_work", "failed", "ambiguous"], "description": "Structured job outcome for orchestration routing." },
    "next_need": { "type": "string", "description": "What kind of work is needed next, e.g. qa_review, implementation_rework, architecture_review, ticket_shaping, goal_decision, release_blocked, or liveness." },
    "suggested_role": { "type": "string", "description": "Optional manifest role key suggested for the next dispatch. The server validates it before enqueueing." },
    "ticket_id": { "type": "string", "description": "Ticket ID this disposition concerns, e.g. MH-047." },
    "reason": { "type": "string", "description": "Brief reason for the disposition and next need." },
    "evidence_links": { "type": "array", "items": { "type": "string" }, "description": "Evidence commands, paths, traces, reports, or URLs supporting this disposition." },
    "approval_id": { "type": "string", "description": "Approval record ID when the next move is review or approval." },
    "work_product_ids": { "type": "array", "items": { "type": "string" }, "description": "Work product IDs produced by this job." },
    "blocked_by": { "type": "array", "items": { "type": "string" }, "description": "Ticket IDs or blockers preventing progress." },
    "trace_id": { "type": "string", "description": "Trace ID when known." }
  },
  "required": ["status"]
}`

func registerJobDispositionRecord(r *Registry) error {
	return r.Register(
		"job_disposition_record",
		"Record a structured job disposition for dispatch-mode orchestration routing.",
		json.RawMessage(jobDispositionRecordSchema),
		handleJobDispositionRecord,
	)
}

func handleJobDispositionRecord(ctx context.Context, _ Root, raw json.RawMessage) (ToolResult, error) {
	session, ok := SessionFromContext(ctx)
	if !ok {
		return ToolResult{}, fmt.Errorf("job_disposition_record: no tool session available")
	}
	if session.DispositionRecorder == nil {
		return ToolResult{}, fmt.Errorf("job_disposition_record: this run is not configured to record dispositions")
	}
	if err := session.DispositionRecorder(ctx, raw); err != nil {
		return ToolResult{}, fmt.Errorf("job_disposition_record: %w", err)
	}
	return ToolResult{Output: "job disposition recorded"}, nil
}
