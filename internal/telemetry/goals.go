/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GoalRecordResult describes where telemetry triage was written.
type GoalRecordResult struct {
	Path      string
	ID        string
	Status    string
	DedupeKey string
	Updated   bool
}

// RecordGoalFromProposal writes structured telemetry triage into docs/goals.
// Actionable, confident proposals become active goals; weak/noisy proposals
// become observations. Duplicate evidence updates the existing goal file.
func RecordGoalFromProposal(repoPath string, proposal ImprovementProposal, now time.Time) (GoalRecordResult, error) {
	if strings.TrimSpace(repoPath) == "" {
		return GoalRecordResult{}, fmt.Errorf("telemetry goals: repo path is empty")
	}
	if now.IsZero() {
		now = time.Now()
	}
	goalsDir := filepath.Join(repoPath, "docs", "goals")
	if err := os.MkdirAll(goalsDir, 0o755); err != nil {
		return GoalRecordResult{}, fmt.Errorf("telemetry goals: create docs/goals: %w", err)
	}
	dedupeKey := goalDedupeKey(proposal)
	for _, name := range []string{"active.md", "observations.md", "superseded.md"} {
		path := filepath.Join(goalsDir, name)
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return GoalRecordResult{}, fmt.Errorf("telemetry goals: read %s: %w", name, err)
		}
		if strings.Contains(string(data), "Dedupe Key: "+dedupeKey) {
			update := fmt.Sprintf("\n### Evidence Update - %s\n\n- Dedupe Key: %s\n- Evidence: %s\n- Suggestion: %s\n", now.Format("2006-01-02"), dedupeKey, proposal.Evidence, proposal.Suggestion)
			if err := os.WriteFile(path, append(data, []byte(update)...), 0o644); err != nil {
				return GoalRecordResult{}, fmt.Errorf("telemetry goals: update %s: %w", name, err)
			}
			return GoalRecordResult{Path: filepath.ToSlash(filepath.Join("docs", "goals", name)), Status: statusFromGoalFile(name), DedupeKey: dedupeKey, Updated: true}, nil
		}
	}

	actionable := proposal.Confidence >= 0.75 && proposal.Target != "" && proposal.Target != TargetUnknown
	name := "observations.md"
	status := "observation"
	idPrefix := "OBS"
	if actionable {
		name = "active.md"
		status = "active"
		idPrefix = "G"
	}
	path := filepath.Join(goalsDir, name)
	data, err := ensureGoalFile(path, name)
	if err != nil {
		return GoalRecordResult{}, err
	}
	id := nextGoalID(string(data), idPrefix)
	entry := renderGoalEntry(id, status, dedupeKey, proposal, now)
	if err := os.WriteFile(path, append(data, []byte(entry)...), 0o644); err != nil {
		return GoalRecordResult{}, fmt.Errorf("telemetry goals: write %s: %w", name, err)
	}
	return GoalRecordResult{Path: filepath.ToSlash(filepath.Join("docs", "goals", name)), ID: id, Status: status, DedupeKey: dedupeKey}, nil
}

func goalDedupeKey(p ImprovementProposal) string {
	parts := []string{"telemetry", p.RepoID, p.Role, string(p.Category), string(p.Target), p.Title}
	for i, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		part = strings.ReplaceAll(part, " ", "-")
		if part == "" {
			part = "unknown"
		}
		parts[i] = part
	}
	return strings.Join(parts, ":")
}

func ensureGoalFile(path, name string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("telemetry goals: read %s: %w", name, err)
	}
	var header string
	switch name {
	case "active.md":
		header = "# Active Goals\n"
	case "observations.md":
		header = "# Observations\n\nweak/noisy evidence lives here until it is actionable.\n"
	default:
		header = "# Superseded Goals\n"
	}
	return []byte(header + "\n"), nil
}

var goalIDRE = regexp.MustCompile(`(?m)^## ((?:G|OBS)-(\d+)):`)

func nextGoalID(text, prefix string) string {
	maxID := 0
	for _, match := range goalIDRE.FindAllStringSubmatch(text, -1) {
		if !strings.HasPrefix(match[1], prefix+"-") {
			continue
		}
		var n int
		_, _ = fmt.Sscanf(match[2], "%d", &n)
		if n > maxID {
			maxID = n
		}
	}
	return fmt.Sprintf("%s-%03d", prefix, maxID+1)
}

func renderGoalEntry(id, status, dedupeKey string, p ImprovementProposal, now time.Time) string {
	category := "operational"
	if p.Target == TargetGuardrail {
		category = "safety"
	}
	if p.Target == TargetInference {
		category = "quality"
	}
	confidence := "low"
	if p.Confidence >= 0.75 {
		confidence = "high"
	} else if p.Confidence >= 0.55 {
		confidence = "medium"
	}
	priority := "P2"
	if p.Severity == "critical" {
		priority = "P0"
	} else if p.Severity == "high" {
		priority = "P1"
	}
	return fmt.Sprintf(`
## %s: %s

- ID: %s
- Status: %s
- Category: %s
- Priority: %s
- Confidence: %s
- Source: telemetry
- Dedupe Key: %s
- Hypothesis: Fixing %s for role %s will improve autonomous delivery outcomes.
- Success Evidence: %s
- Falsification Evidence: The same telemetry pattern recurs after the fix.
- Competes With: None recorded
- Supports: G-001
- Last Reviewed: %s
- Review Trigger: New telemetry with dedupe key %s
- Owner: CEO
- Evidence: %s
- Suggested Action: %s
`, id, p.Title, id, status, category, priority, confidence, dedupeKey, p.Target, p.Role, p.Suggestion, now.Format("2006-01-02"), dedupeKey, p.Evidence, p.Suggestion)
}

func statusFromGoalFile(name string) string {
	switch name {
	case "active.md":
		return "active"
	case "observations.md":
		return "observation"
	default:
		return "superseded"
	}
}
