package operatingmodel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Drift reports a missing or stale operating-model artifact in a target repo.
type Drift struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Report summarizes whether a repo carries the mirrored BDD-led delivery model.
type Report struct {
	Missing []Drift `json:"missing"`
	Stale   []Drift `json:"stale"`
}

// OK returns true when no mirrored operating-model drift was found.
func (r Report) OK() bool {
	return len(r.Missing) == 0 && len(r.Stale) == 0
}

// Summary returns a short operator-facing drift summary.
func (r Report) Summary() string {
	if r.OK() {
		return "operating model is current"
	}
	var parts []string
	if len(r.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", len(r.Missing)))
	}
	if len(r.Stale) > 0 {
		parts = append(parts, fmt.Sprintf("%d stale", len(r.Stale)))
	}
	return "operating model drift detected: " + strings.Join(parts, ", ")
}

type requiredArtifact struct {
	path    string
	needles []string
}

var requiredArtifacts = []requiredArtifact{
	{path: "AGENTS.md", needles: []string{"BDD feature contracts", "walking skeleton", "Conversation system record"}},
	{path: "docs/QUALITY_SCORE.md", needles: []string{"shipped feature scenarios", "enabler work"}},
	{path: "docs/design-docs/delivery-operating-model.md", needles: []string{"AD-074", "BDD-Led Goal-Driven Walking-Skeleton"}},
	{path: "docs/design-docs/conversation-as-system-record.md", needles: []string{"AD-086", "Conversation As System Record", "Chat summaries can help humans catch up"}},
	{path: "docs/goals/README.md", needles: []string{"Goal Schema", "Dedupe Key", "Autonomous Goal Rule"}},
	{path: "docs/goals/active.md", needles: []string{"G-001", "Status: active", "Hypothesis"}},
	{path: "docs/goals/observations.md", needles: []string{"weak/noisy evidence"}},
	{path: "docs/goals/superseded.md", needles: []string{"Superseded Goals"}},
	{path: "docs/features/README.md", needles: []string{"BDD Feature Contracts", "Scenario Schedule", "Given/When/Then"}},
	{path: "docs/features/F-001-delivery-operating-model.md", needles: []string{"Feature ID: F-001", "Scenario Schedule", "Given", "When", "Then"}},
	{path: "docs/exec-plans/README.md", needles: []string{"**Goals:**", "**BDD Feature:**", "**Scenario Schedule:**", "**Walking Skeleton Slice:**"}},
	{path: "docs/exec-plans/active/current-operating-plan.md", needles: []string{"**Goals:**", "**BDD Feature:**", "**Current Failing Scenario:**"}},
	{path: "docs/tickets/README.md", needles: []string{"work_type", "bdd_scenarios", "end_to_end_evidence", "verified_by"}},
	{path: ".harness/knowledge/context-glossary.yaml", needles: []string{"goals, BDD, feature contracts, planning, feedback, or quality evidence"}},
}

// CheckRepo checks generated target harness artifacts without mutating them.
func CheckRepo(repoPath string) (Report, error) {
	if strings.TrimSpace(repoPath) == "" {
		return Report{}, fmt.Errorf("operating model: repo path is empty")
	}
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return Report{}, fmt.Errorf("operating model: resolve repo path: %w", err)
	}
	var report Report
	for _, artifact := range requiredArtifactsForRepo(absRepo) {
		absPath := filepath.Join(absRepo, filepath.FromSlash(artifact.path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				report.Missing = append(report.Missing, Drift{Path: artifact.path, Message: "file is missing"})
				continue
			}
			return Report{}, fmt.Errorf("operating model: read %s: %w", artifact.path, err)
		}
		text := string(data)
		for _, needle := range artifact.needles {
			if !containsFold(text, needle) {
				report.Stale = append(report.Stale, Drift{
					Path:    artifact.path,
					Message: fmt.Sprintf("missing required marker %q", needle),
				})
			}
		}
	}
	return report, nil
}

func requiredArtifactsForRepo(absRepo string) []requiredArtifact {
	if !IsFoundationHarnessRepo(absRepo) {
		return requiredArtifacts
	}
	filtered := make([]requiredArtifact, 0, len(requiredArtifacts))
	for _, artifact := range requiredArtifacts {
		if artifact.path == ".harness/knowledge/context-glossary.yaml" {
			continue
		}
		filtered = append(filtered, artifact)
	}
	return filtered
}

// IsFoundationHarnessRepo reports whether repoPath is the mars-harness source
// repo. The source harness records doctrine in AGENTS.md and docs rather than
// in generated target .harness metadata.
func IsFoundationHarnessRepo(absRepo string) bool {
	data, err := os.ReadFile(filepath.Join(absRepo, "go.mod"))
	if err != nil {
		return false
	}
	if !strings.Contains(string(data), "module github.com/greaveselliott/mars-harness") {
		return false
	}
	if _, err := os.Stat(filepath.Join(absRepo, "internal", "scanner", "init.go")); err != nil {
		return false
	}
	return true
}

func containsFold(text, needle string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(needle))
}
