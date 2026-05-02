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
}

type interventionDebtTicket struct {
	Path   string
	Output string
}

func (s *Server) recordInterventionDebtTicket(ctx context.Context, repoID string, proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) {
	if strings.TrimSpace(proposal.RepoID) == "" {
		proposal.RepoID = repoID
	}
	if strings.TrimSpace(proposal.RepoID) == "" || s.repos == nil {
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
	return "telemetry-triage"
}

func interventionDebtBody(proposal telemetry.ImprovementProposal, origin interventionDebtOrigin) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Context\n\n")
	fmt.Fprintf(&b, "Telemetry triage identified intervention debt that should become durable work before direct harness evolution. This ticket is generated from evidence and deduped by repo, role, target, category, and evidence window.\n\n")

	fmt.Fprintf(&b, "## Triage Metadata\n\n")
	fmt.Fprintf(&b, "- Kind: intervention-debt\n")
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
	if origin.Score != nil {
		fmt.Fprintf(&b, "- Score snapshot: %.2f over %d samples in %dd\n", origin.Score.Value, origin.Score.SampleSize, origin.Score.WindowDays)
	}

	fmt.Fprintf(&b, "\n## Evidence\n\n%s\n\n", strings.TrimSpace(proposal.Evidence))
	if origin.Event != nil && strings.TrimSpace(origin.Event.Message) != "" {
		fmt.Fprintf(&b, "Latest event message:\n\n```text\n%s\n```\n\n", strings.TrimSpace(origin.Event.Message))
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
