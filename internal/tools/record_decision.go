/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/learnings"
)

const recordDecisionSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "summary":   { "type": "string", "description": "One-line summary of the decision" },
    "rationale": { "type": "string", "description": "Why this decision was made (context, tradeoffs, alternatives considered)" }
  },
  "required": ["summary"]
}`

type recordDecisionArgs struct {
	Summary   string `json:"summary"`
	Rationale string `json:"rationale"`
}

// RecordDecisionRole is set per-job by the executor so the tool knows
// which role is recording the decision. Protected by the agent loop's
// sequential tool execution.
var RecordDecisionRole string

func registerRecordDecision(r *Registry) error {
	return r.Register(
		"record_decision",
		"Record an architectural or operational decision for future agent runs. Decisions are persisted in .harness/learnings.yaml and visible to all agents.",
		json.RawMessage(recordDecisionSchema),
		handleRecordDecision,
	)
}

func handleRecordDecision(_ context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args recordDecisionArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("record_decision: parse arguments: %w", err)
	}
	if strings.TrimSpace(args.Summary) == "" {
		return ToolResult{}, fmt.Errorf("record_decision: summary is required")
	}

	store := learnings.NewStore(root.Abs())
	role := RecordDecisionRole
	if role == "" {
		role = "unknown"
	}

	added, err := store.AddDecision(role, strings.TrimSpace(args.Summary), strings.TrimSpace(args.Rationale))
	if err != nil {
		return ToolResult{}, fmt.Errorf("record_decision: save: %w", err)
	}
	if !added {
		return ToolResult{Output: "decision already recorded (duplicate summary)"}, nil
	}

	slog.Info("record_decision: persisted", "role", role, "summary", args.Summary)
	return ToolResult{Output: fmt.Sprintf("decision recorded: %s", args.Summary)}, nil
}
