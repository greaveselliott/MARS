/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import "fmt"

// ImprovementTarget is the harness surface most likely to need improvement.
type ImprovementTarget string

const (
	TargetPrompt     ImprovementTarget = "prompt"
	TargetSkill      ImprovementTarget = "skill"
	TargetProcess    ImprovementTarget = "process"
	TargetGuardrail  ImprovementTarget = "guardrail"
	TargetContext    ImprovementTarget = "context"
	TargetInference  ImprovementTarget = "inference"
	TargetManifest   ImprovementTarget = "manifest"
	TargetToolPolicy ImprovementTarget = "tool_policy"
	TargetUnknown    ImprovementTarget = "unknown"
)

// ImprovementProposal is a self-reflective triage output. It translates raw
// telemetry into a concrete harness surface that can be reviewed or evolved.
type ImprovementProposal struct {
	Role           string            `json:"role"`
	RepoID         string            `json:"repo_id,omitempty"`
	Category       FailureCategory   `json:"category,omitempty"`
	Target         ImprovementTarget `json:"target"`
	Severity       string            `json:"severity"`
	Title          string            `json:"title"`
	Suggestion     string            `json:"suggestion"`
	Evidence       string            `json:"evidence"`
	CandidateFiles []string          `json:"candidate_files,omitempty"`
	Confidence     float64           `json:"confidence"`
}

// ScoreSnapshot is the scoring subset needed for self-reflection without
// coupling telemetry to the scoring package.
type ScoreSnapshot struct {
	Role       string
	RepoID     string
	Value      float64
	SampleSize int
	WindowDays int
}

// TriagePattern classifies a recurring telemetry pattern into an improvement
// proposal. It does not mutate anything; orchestration decides whether to
// create a ticket, record an evolution, or only surface the signal.
func TriagePattern(p Pattern) ImprovementProposal {
	count := p.Count
	if count <= 0 {
		count = PatternThreshold
	}
	window := p.Window
	if window == "" {
		window = "24h"
	}

	proposal := ImprovementProposal{
		RepoID:     p.RepoID,
		Role:       p.Role,
		Category:   p.Category,
		Severity:   severityForCount(count),
		Evidence:   fmt.Sprintf("%d %s failures in %s", count, p.Category, window),
		Confidence: 0.7,
	}

	switch p.Category {
	case CategoryContextOverflow, CategoryBudgetExceeded:
		proposal.Target = TargetContext
		proposal.Title = "Reduce context load"
		proposal.Suggestion = fmt.Sprintf("Role %q repeatedly exceeded context or budget limits; review knowledge routes, glossary routing, ticket index size, and role prompt breadth before increasing model context.", p.Role)
		proposal.CandidateFiles = []string{".harness/knowledge/context-glossary.yaml", fmt.Sprintf(".harness/roles/%s.md", p.Role)}

	case CategoryLLMUnreachable, CategoryInferenceCrash:
		proposal.Target = TargetInference
		proposal.Title = "Stabilize inference"
		proposal.Suggestion = fmt.Sprintf("Role %q hit repeated inference availability failures; review local model profile, llama server tuning, restart policy, and doctor checks before changing the role prompt.", p.Role)
		proposal.Confidence = 0.8

	case CategoryModelUnavailable:
		proposal.Target = TargetInference
		proposal.Title = "Install or route model tier"
		proposal.Suggestion = fmt.Sprintf("Role %q could not find its configured local model tier; run setup for the active performance profile, verify model checksums, or configure an explicit remote fallback before retrying jobs.", p.Role)
		proposal.CandidateFiles = []string{"~/.mars-harness/config.yaml", ".harness/manifest.yaml"}
		proposal.Confidence = 0.85

	case CategoryToolTimeout:
		proposal.Target = TargetToolPolicy
		proposal.Title = "Fix slow tool workflow"
		proposal.Suggestion = fmt.Sprintf("Role %q repeatedly timed out in tools; inspect long-running commands, background execution guidance, tool timeout policy, and repo-specific test commands.", p.Role)
		proposal.CandidateFiles = []string{fmt.Sprintf(".harness/roles/%s.md", p.Role), ".harness/manifest.yaml"}

	case CategoryCircleDetected, CategoryMaxTurns:
		proposal.Target = TargetSkill
		proposal.Title = "Capture missing workflow skill"
		proposal.Suggestion = fmt.Sprintf("Role %q repeatedly looped or exhausted turns; inspect traces for a reusable missing workflow and create or update a compact scoped skill before expanding the role prompt.", p.Role)
		proposal.CandidateFiles = []string{
			fmt.Sprintf(".harness/skills/%s-workflow/SKILL.md", p.Role),
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
		}

	case CategoryManifestError:
		proposal.Target = TargetManifest
		proposal.Title = "Repair harness manifest"
		proposal.Suggestion = fmt.Sprintf("Role %q repeatedly failed on bundle or manifest loading; validate role names, prompt paths, tool allowlists, guardrail refs, and knowledge route refs.", p.Role)
		proposal.CandidateFiles = []string{".harness/manifest.yaml"}
		proposal.Confidence = 0.85

	case CategoryTicketGate:
		proposal.Target = TargetProcess
		proposal.Title = "Fix ticket completion workflow"
		proposal.Suggestion = fmt.Sprintf("Role %q repeatedly ended without completing an existing in-progress ticket; inspect traces, role guidance, trust level, ticket gate rules, and target ticket state before retrying the same queue item.", p.Role)
		proposal.CandidateFiles = []string{
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
			".harness/manifest.yaml",
			"docs/tickets/in-progress/",
		}
		proposal.Confidence = 0.85

	case CategoryGuardrailBlock:
		proposal.Target = TargetGuardrail
		proposal.Title = "Calibrate guardrail workflow"
		proposal.Suggestion = fmt.Sprintf("Role %q hit guardrail or tool-policy blocks; inspect the blocked operation, the relevant guardrail, trust level, and role guidance before loosening enforcement.", p.Role)
		proposal.CandidateFiles = []string{
			".harness/guardrails/",
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
			".harness/manifest.yaml",
		}
		proposal.Confidence = 0.8

	case CategoryHumanFollowup:
		proposal.Target = TargetProcess
		proposal.Title = "Reduce human follow-up"
		proposal.Suggestion = fmt.Sprintf("Role %q needed human follow-up after agent work; inspect the follow-up commit or review evidence, then update role guidance, a compact skill, tests, or guardrails so the same correction is captured by the harness.", p.Role)
		proposal.CandidateFiles = []string{
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
			fmt.Sprintf(".harness/skills/%s-workflow/SKILL.md", p.Role),
			".harness/guardrails/",
		}
		proposal.Confidence = 0.75

	case CategoryRevertedCommit:
		proposal.Target = TargetProcess
		proposal.Title = "Prevent reverted agent commits"
		proposal.Suggestion = fmt.Sprintf("Role %q produced work that was reverted; inspect the reverted commit, trace, ticket evidence, and quality gates before letting the role repeat that workflow.", p.Role)
		proposal.CandidateFiles = []string{
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
			fmt.Sprintf(".harness/skills/%s-workflow/SKILL.md", p.Role),
			".harness/guardrails/",
		}
		proposal.Confidence = 0.8

	case CategoryStaleTicket:
		proposal.Target = TargetProcess
		proposal.Title = "Drain stale in-progress work"
		proposal.Suggestion = fmt.Sprintf("Role %q has stale in-progress ticket state; inspect blockers, ticket handoff rules, janitor recovery, and whether the ticket should move back to backlog with explicit blocker evidence.", p.Role)
		proposal.CandidateFiles = []string{
			"docs/tickets/in-progress/",
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
			".harness/manifest.yaml",
		}
		proposal.Confidence = 0.8

	case CategoryManualStop:
		proposal.Target = TargetProcess
		proposal.Title = "Remove manual stop trigger"
		proposal.Suggestion = fmt.Sprintf("Role %q was manually stopped or cancelled; inspect the trace and operator reason, then tighten the role stop condition, timeout, recovery policy, or escalation behavior before retrying unchanged.", p.Role)
		proposal.CandidateFiles = []string{
			fmt.Sprintf(".harness/roles/%s.md", p.Role),
			".harness/manifest.yaml",
		}
		proposal.Confidence = 0.7

	case CategoryUnknown:
		proposal.Target = TargetProcess
		proposal.Title = "Classify unknown failure"
		proposal.Suggestion = fmt.Sprintf("Role %q has recurring unknown failures; inspect traces and add classifier coverage, deterministic remediation, or a clearer role/process rule.", p.Role)
		proposal.Confidence = 0.55

	default:
		proposal.Target = TargetUnknown
		proposal.Title = "Review uncategorized telemetry"
		proposal.Suggestion = fmt.Sprintf("Role %q has recurring %s telemetry; inspect traces and decide whether this belongs in prompt, skill, process, guardrail, tool, or model configuration.", p.Role, p.Category)
		proposal.Confidence = 0.5
	}

	return proposal
}

// TriagePatterns maps multiple patterns to improvement proposals.
func TriagePatterns(patterns []Pattern) []ImprovementProposal {
	out := make([]ImprovementProposal, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, TriagePattern(p))
	}
	return out
}

// TriageScore turns a low rolling score into a self-improvement proposal.
// Returns false when the score has too few samples or is healthy enough.
func TriageScore(sc ScoreSnapshot) (ImprovementProposal, bool) {
	if sc.SampleSize < 5 || sc.Value >= 0.5 {
		return ImprovementProposal{}, false
	}
	window := sc.WindowDays
	if window <= 0 {
		window = 30
	}
	return ImprovementProposal{
		Role:       sc.Role,
		RepoID:     sc.RepoID,
		Target:     TargetProcess,
		Severity:   "high",
		Title:      "Low role score",
		Suggestion: fmt.Sprintf("Role %q has score %.2f over %d samples in %dd; triage failed outcomes, intervention debt, prompt scope, reusable skills, guardrails, tool policy, and model tier before raising autonomy.", sc.Role, sc.Value, sc.SampleSize, window),
		Evidence:   fmt.Sprintf("score %.2f, samples %d, window %dd", sc.Value, sc.SampleSize, window),
		CandidateFiles: []string{
			fmt.Sprintf(".harness/roles/%s.md", sc.Role),
			fmt.Sprintf(".harness/skills/%s-workflow/SKILL.md", sc.Role),
			".harness/manifest.yaml",
		},
		Confidence: 0.8,
	}, true
}

func severityForCount(count int) string {
	switch {
	case count >= PatternThreshold*3:
		return "critical"
	case count >= PatternThreshold*2:
		return "high"
	default:
		return "medium"
	}
}
