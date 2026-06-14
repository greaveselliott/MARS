/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-012-self-improvement-loop.md
*/
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/greaveselliott/mars-harness/internal/tools"
)

type interventionDebtOrigin struct {
	Kind           string
	EvidenceWindow string
	Event          *telemetry.Event
	Score          *telemetry.ScoreSnapshot
	TraceID        string
	Commit         string
	Outcome        string
	ToolName       string
	Message        string
}

type interventionDebtTicket struct {
	Path   string
	Output string
}

type interventionDebtSignal struct {
	Kind           string
	RepoID         string
	Role           string
	JobID          string
	Category       telemetry.FailureCategory
	Count          int
	EvidenceWindow string
	Event          *telemetry.Event
	TraceID        string
	Commit         string
	Outcome        string
	ToolName       string
	Message        string
}

type interventionDebtRoute string

const (
	interventionDebtRouteFoundationTelemetry interventionDebtRoute = "foundation_telemetry"
	interventionDebtRouteTargetBacklog       interventionDebtRoute = "target_backlog"
)

func (s *Server) recordInterventionDebtSignal(ctx context.Context, signal interventionDebtSignal) {
	signal.RepoID = strings.TrimSpace(signal.RepoID)
	signal.Role = strings.TrimSpace(signal.Role)
	if signal.RepoID == "" || signal.Role == "" {
		return
	}
	if signal.Category == "" {
		signal.Category = telemetry.Classify(signal.Message)
	}
	if signal.Event == nil && s.telemetry != nil && strings.TrimSpace(signal.Message) != "" && strings.TrimSpace(signal.JobID) != "" {
		evt := s.telemetry.Record(signal.JobID, signal.RepoID, signal.Role, signal.Message)
		signal.Event = &evt
		if signal.Category == "" || signal.Category == telemetry.CategoryUnknown {
			signal.Category = evt.Category
		}
	}

	proposal := interventionDebtProposalFromSignal(signal)
	origin := interventionDebtOrigin{
		Kind:           interventionDebtSignalKind(signal),
		EvidenceWindow: interventionDebtSignalWindow(signal),
		Event:          signal.Event,
		TraceID:        strings.TrimSpace(signal.TraceID),
		Commit:         strings.TrimSpace(signal.Commit),
		Outcome:        strings.TrimSpace(signal.Outcome),
		ToolName:       strings.TrimSpace(signal.ToolName),
		Message:        strings.TrimSpace(signal.Message),
	}
	if interventionDebtRouting(proposal) == interventionDebtRouteFoundationTelemetry {
		slog.Info("serve: foundation-owned intervention signal kept out of target backlog",
			"repo_id", signal.RepoID,
			"role", signal.Role,
			"target", proposal.Target,
			"category", proposal.Category,
		)
		if s.dash != nil {
			payload, _ := json.Marshal(map[string]string{
				"repo_id":  signal.RepoID,
				"role":     signal.Role,
				"target":   string(proposal.Target),
				"category": interventionDebtCategory(proposal),
				"result":   "local telemetry only; eligible for anonymous foundation reporting",
			})
			s.dash.BroadcastEvent("foundation_telemetry_signal", string(payload))
		}
		if foundationTelemetrySignalShouldOfferEvolution(signal) {
			s.offerInterventionDebtEvolution(ctx, signal.RepoID, proposal, "signal_"+interventionDebtCategory(proposal))
		}
		return
	}
	s.recordInterventionDebtTicket(ctx, signal.RepoID, proposal, origin)
	s.offerInterventionDebtEvolution(ctx, signal.RepoID, proposal, "signal_"+interventionDebtCategory(proposal))
}

func (s *Server) recordInterventionDebtTicket(ctx context.Context, repoID string, proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) {
	if strings.TrimSpace(proposal.RepoID) == "" {
		proposal.RepoID = repoID
	}
	if strings.TrimSpace(proposal.RepoID) == "" || s.repos == nil {
		return
	}
	if interventionDebtRouting(proposal) == interventionDebtRouteFoundationTelemetry {
		slog.Info("serve: foundation-owned telemetry pattern kept out of target backlog",
			"repo_id", proposal.RepoID,
			"role", proposal.Role,
			"target", proposal.Target,
			"category", proposal.Category,
		)
		if s.dash != nil {
			payload, _ := json.Marshal(map[string]string{
				"repo_id":  proposal.RepoID,
				"role":     proposal.Role,
				"target":   string(proposal.Target),
				"category": interventionDebtCategory(proposal),
				"result":   "local telemetry only; eligible for anonymous foundation reporting",
			})
			s.dash.BroadcastEvent("foundation_telemetry_signal", string(payload))
		}
		return
	}

	rec, err := s.repos.FindByID(ctx, proposal.RepoID)
	if err != nil || rec == nil {
		if err == nil {
			err = fmt.Errorf("repo %s not found", proposal.RepoID)
		}
		s.recordInterventionDebtFailure(proposal, origin, fmt.Errorf("resolve repo: %w", err))
		return
	}

	ticket, err := createInterventionDebtTicket(rec.Path, proposal, origin)
	if err != nil {
		s.recordInterventionDebtFailure(proposal, origin, err)
		return
	}

	slog.Info("serve: intervention-debt ticket recorded",
		"repo_id", proposal.RepoID,
		"role", proposal.Role,
		"target", proposal.Target,
		"category", proposal.Category,
		"ticket", ticket.Path,
	)

	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"repo_id":     proposal.RepoID,
			"role":        proposal.Role,
			"target":      string(proposal.Target),
			"category":    interventionDebtCategory(proposal),
			"severity":    proposal.Severity,
			"ticket_path": ticket.Path,
			"result":      ticket.Output,
		})
		s.dash.BroadcastEvent("intervention_debt_ticket", string(payload))
	}
}

func (s *Server) recordInterventionDebtFailure(proposal telemetry.ImprovementProposal, origin interventionDebtOrigin, err error) {
	slog.Error("serve: intervention-debt ticket creation failed",
		"repo_id", proposal.RepoID,
		"role", proposal.Role,
		"target", proposal.Target,
		"category", proposal.Category,
		"err", err,
	)
	if s.telemetry == nil {
		return
	}
	jobID := "intervention-debt"
	if origin.Event != nil && origin.Event.JobID != "" {
		jobID = origin.Event.JobID
	}
	s.telemetry.Record(jobID, proposal.RepoID, proposal.Role, fmt.Sprintf("intervention debt ticket creation failed: %v", err))
}

func createInterventionDebtTicket(repoPath string, proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) (interventionDebtTicket, error) {
	root, err := tools.NewRoot(repoPath)
	if err != nil {
		return interventionDebtTicket{}, err
	}

	input := tools.TicketInput{
		Title:      interventionDebtTitle(proposal),
		Priority:   interventionDebtPriority(proposal.Severity),
		Complexity: "medium",
		Kind:       "intervention-debt",
		DedupeKey:  interventionDebtDedupeKey(proposal, origin),
		Metadata:   interventionDebtMetadata(proposal, origin),
		Source:     interventionDebtSource(proposal, origin),
		Body:       interventionDebtBody(proposal, origin),
	}

	result, err := tools.CreateTicket(root, input)
	if err != nil {
		return interventionDebtTicket{}, err
	}
	return interventionDebtTicket{
		Path:   ticketPathFromToolOutput(result.Output),
		Output: result.Output,
	}, nil
}

func interventionDebtProposalFromSignal(signal interventionDebtSignal) telemetry.ImprovementProposal {
	count := signal.Count
	if count <= 0 {
		count = 1
	}
	window := interventionDebtSignalWindow(signal)
	proposal := telemetry.TriagePattern(telemetry.Pattern{
		RepoID:   signal.RepoID,
		Role:     signal.Role,
		Category: signal.Category,
		Count:    count,
		Window:   window,
	})
	evidence := strings.TrimSpace(interventionDebtSignalEvidence(signal))
	if evidence != "" {
		proposal.Evidence = evidence
	}
	return proposal
}

func interventionDebtBelongsToFoundation(proposal telemetry.ImprovementProposal) bool {
	return telemetry.ReportableFoundationCategory(proposal.Category)
}

func interventionDebtRouting(proposal telemetry.ImprovementProposal) interventionDebtRoute {
	switch proposal.Category {
	case telemetry.CategoryHumanFollowup, telemetry.CategoryRevertedCommit, telemetry.CategoryStaleTicket:
		return interventionDebtRouteTargetBacklog
	}
	if interventionDebtBelongsToFoundation(proposal) || proposal.Category == "" || proposal.Category == telemetry.CategoryManualStop {
		return interventionDebtRouteFoundationTelemetry
	}
	return interventionDebtRouteFoundationTelemetry
}

func foundationTelemetrySignalShouldOfferEvolution(signal interventionDebtSignal) bool {
	if signal.Category == telemetry.CategoryGuardrailLoop {
		return true
	}
	return signal.Count >= telemetry.PatternThreshold
}

func interventionDebtSignalWindow(signal interventionDebtSignal) string {
	window := strings.TrimSpace(signal.EvidenceWindow)
	if window == "" {
		window = "24h"
	}
	return window
}

func interventionDebtSignalKind(signal interventionDebtSignal) string {
	if kind := strings.TrimSpace(signal.Kind); kind != "" {
		return kind
	}
	switch signal.Category {
	case telemetry.CategoryGuardrailLoop:
		return "guardrail_loop"
	case telemetry.CategoryGuardrailBlock:
		return "guardrail_block"
	case telemetry.CategoryWorkspaceHygiene:
		return "workspace_hygiene_block"
	case telemetry.CategoryHumanFollowup:
		return "human_followup_commit"
	case telemetry.CategoryRevertedCommit:
		return "reverted_agent_commit"
	case telemetry.CategoryStaleTicket:
		return "stale_in_progress_ticket"
	case telemetry.CategoryManualStop:
		return "manual_stop"
	case telemetry.CategoryToolTimeout:
		return "timeout"
	case telemetry.CategoryCircleDetected:
		return "repeated_tool_loop"
	default:
		return "terminal_agent_result"
	}
}

func interventionDebtSignalEvidence(signal interventionDebtSignal) string {
	var parts []string
	kind := interventionDebtSignalKind(signal)
	parts = append(parts, fmt.Sprintf("%s signal for repo %s role %s category %s in %s",
		kind, signal.RepoID, signal.Role, signal.Category, interventionDebtSignalWindow(signal)))
	if signal.Count > 0 {
		parts = append(parts, fmt.Sprintf("count=%d", signal.Count))
	}
	if signal.JobID != "" {
		parts = append(parts, "job="+signal.JobID)
	}
	if signal.TraceID != "" {
		parts = append(parts, "trace="+signal.TraceID)
	}
	if signal.Commit != "" {
		parts = append(parts, "commit="+signal.Commit)
	}
	if signal.Outcome != "" {
		parts = append(parts, "outcome="+signal.Outcome)
	}
	if signal.ToolName != "" {
		parts = append(parts, "tool="+signal.ToolName)
	}
	if msg := strings.TrimSpace(signal.Message); msg != "" {
		parts = append(parts, "message="+msg)
	}
	return strings.Join(parts, "\n")
}

func interventionDebtTitle(proposal telemetry.ImprovementProposal) string {
	category := interventionDebtCategory(proposal)
	role := strings.TrimSpace(proposal.Role)
	if role == "" {
		role = "unknown-role"
	}
	title := strings.TrimSpace(proposal.Title)
	if title == "" {
		title = fmt.Sprintf("%s %s", proposal.Target, category)
	}
	return fmt.Sprintf("Intervention debt: %s for %s %s %s", title, role, proposal.Target, category)
}

func interventionDebtCategory(proposal telemetry.ImprovementProposal) string {
	if proposal.Category != "" {
		return string(proposal.Category)
	}
	return "score"
}

func interventionDebtPriority(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func interventionDebtDedupeKey(proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) string {
	window := strings.TrimSpace(origin.EvidenceWindow)
	if window == "" {
		window = "unknown-window"
	}
	parts := []string{
		"intervention-debt",
		proposal.RepoID,
		proposal.Role,
		string(proposal.Target),
		interventionDebtCategory(proposal),
		window,
	}
	for i, part := range parts {
		parts[i] = normalizeDedupePart(part)
	}
	return strings.Join(parts, ":")
}

func normalizeDedupePart(value string) string {
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
				b.WriteRune('-')
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

func interventionDebtMetadata(proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) map[string]string {
	metadata := map[string]string{
		"role":            proposal.Role,
		"repo_id":         proposal.RepoID,
		"target":          string(proposal.Target),
		"category":        interventionDebtCategory(proposal),
		"severity":        proposal.Severity,
		"confidence":      fmt.Sprintf("%.2f", proposal.Confidence),
		"evidence_window": strings.TrimSpace(origin.EvidenceWindow),
		"origin_kind":     strings.TrimSpace(origin.Kind),
	}
	if origin.Event != nil {
		metadata["origin_event_id"] = origin.Event.ID
		metadata["origin_job_id"] = origin.Event.JobID
	}
	if origin.TraceID != "" {
		metadata["trace_id"] = origin.TraceID
	}
	if origin.Commit != "" {
		metadata["commit"] = origin.Commit
	}
	if origin.Outcome != "" {
		metadata["outcome"] = origin.Outcome
	}
	if origin.ToolName != "" {
		metadata["tool"] = origin.ToolName
	}
	if origin.Score != nil {
		metadata["score_value"] = fmt.Sprintf("%.2f", origin.Score.Value)
		metadata["score_samples"] = fmt.Sprintf("%d", origin.Score.SampleSize)
	}
	return metadata
}

func interventionDebtSource(proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) string {
	if origin.Event != nil && origin.Event.ID != "" {
		return "telemetry:" + origin.Event.ID
	}
	if origin.Score != nil {
		return fmt.Sprintf("score:%s:%s:%dd", proposal.RepoID, proposal.Role, origin.Score.WindowDays)
	}
	if origin.TraceID != "" {
		return "trace:" + origin.TraceID
	}
	if origin.Commit != "" {
		return "commit:" + origin.Commit
	}
	if kind := strings.TrimSpace(origin.Kind); kind != "" {
		return "signal:" + kind
	}
	return "telemetry-triage"
}

func interventionDebtBody(proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Context\n\n")
	fmt.Fprintf(&b, "A harness self-improvement signal identified intervention debt that should become durable work before direct harness evolution. This ticket is generated from evidence and deduped by repo, role, target, category, and evidence window.\n\n")

	fmt.Fprintf(&b, "## Triage Metadata\n\n")
	fmt.Fprintf(&b, "- Kind: intervention-debt\n")
	if origin.Kind != "" {
		fmt.Fprintf(&b, "- Origin kind: %s\n", origin.Kind)
	}
	fmt.Fprintf(&b, "- Repo ID: %s\n", proposal.RepoID)
	fmt.Fprintf(&b, "- Role: %s\n", proposal.Role)
	fmt.Fprintf(&b, "- Proposal: %s\n", proposal.Title)
	fmt.Fprintf(&b, "- Target: %s\n", proposal.Target)
	fmt.Fprintf(&b, "- Category: %s\n", interventionDebtCategory(proposal))
	fmt.Fprintf(&b, "- Severity: %s\n", proposal.Severity)
	fmt.Fprintf(&b, "- Confidence: %.2f\n", proposal.Confidence)
	if origin.EvidenceWindow != "" {
		fmt.Fprintf(&b, "- Evidence window: %s\n", origin.EvidenceWindow)
	}
	if origin.Event != nil {
		fmt.Fprintf(&b, "- Origin event: %s\n", origin.Event.ID)
		fmt.Fprintf(&b, "- Origin job: %s\n", origin.Event.JobID)
	}
	if origin.TraceID != "" {
		fmt.Fprintf(&b, "- Trace ID: %s\n", origin.TraceID)
	}
	if origin.Commit != "" {
		fmt.Fprintf(&b, "- Commit: %s\n", origin.Commit)
	}
	if origin.Outcome != "" {
		fmt.Fprintf(&b, "- Outcome: %s\n", origin.Outcome)
	}
	if origin.ToolName != "" {
		fmt.Fprintf(&b, "- Tool: %s\n", origin.ToolName)
	}
	if origin.Score != nil {
		fmt.Fprintf(&b, "- Score snapshot: %.2f over %d samples in %dd\n", origin.Score.Value, origin.Score.SampleSize, origin.Score.WindowDays)
	}

	fmt.Fprintf(&b, "\n## Evidence\n\n%s\n\n", strings.TrimSpace(proposal.Evidence))
	if origin.Event != nil && strings.TrimSpace(origin.Event.Message) != "" {
		fmt.Fprintf(&b, "Latest event message:\n\n```text\n%s\n```\n\n", strings.TrimSpace(origin.Event.Message))
	} else if strings.TrimSpace(origin.Message) != "" {
		fmt.Fprintf(&b, "Latest signal message:\n\n```text\n%s\n```\n\n", strings.TrimSpace(origin.Message))
	}

	fmt.Fprintf(&b, "## Recommendation\n\n%s\n\n", strings.TrimSpace(proposal.Suggestion))

	if len(proposal.CandidateFiles) > 0 {
		fmt.Fprintf(&b, "## Candidate Files\n\n")
		for _, file := range proposal.CandidateFiles {
			fmt.Fprintf(&b, "- `%s`\n", file)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Acceptance Criteria\n\n")
	fmt.Fprintf(&b, "### Functional (happy path)\n\n")
	fmt.Fprintf(&b, "- [ ] Root cause is classified and linked to the affected harness surface.\n")
	fmt.Fprintf(&b, "- [ ] Fix is implemented through prompt, skill, process, guardrail, context, inference, manifest, or tool-policy changes as appropriate.\n")
	fmt.Fprintf(&b, "- [ ] Follow-up run or score evidence confirms the issue no longer repeats in the same window.\n\n")
	fmt.Fprintf(&b, "### Edge cases and negative paths\n\n")
	fmt.Fprintf(&b, "- [ ] Unknown or unsafe fixes remain ticketed rather than applied as unbounded direct evolution.\n")
	fmt.Fprintf(&b, "- [ ] New guardrails or tests prevent recurrence where the failure mode is deterministic.\n\n")
	fmt.Fprintf(&b, "### Observability, docs, and regressions\n\n")
	fmt.Fprintf(&b, "- [ ] Relevant design or product docs explain the change and why.\n")
	fmt.Fprintf(&b, "- [ ] Telemetry, score, or trace evidence is linked in the completion notes.\n")
	return b.String()
}

func ticketPathFromToolOutput(output string) string {
	start := strings.Index(output, "docs/tickets/")
	if start < 0 {
		return ""
	}
	end := start
	for end < len(output) {
		switch output[end] {
		case ' ', '\n', '\t', ')', '"', '\'':
			return output[start:end]
		default:
			end++
		}
	}
	return output[start:end]
}
