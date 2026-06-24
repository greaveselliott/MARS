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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketDir(t *testing.T) (string, Root) {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/in-review", "docs/tickets/done"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, sub), 0o755))
	}
	root, err := NewRoot(dir)
	require.NoError(t, err)
	return dir, root
}

func writeTicket(t *testing.T, dir, status, filename, title string) {
	t.Helper()
	content := "---\nid: " + strings.TrimSuffix(strings.Split(filename, "-")[0]+"-"+strings.Split(filename, "-")[1], ".md") +
		"\ntitle: " + title + "\npriority: high\n---\n\n# " + title + "\n"
	path := filepath.Join(dir, "docs", "tickets", status, filename)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestTicketCreate_basic(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	_ = dir

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "Implement scoring system",
		"priority": "high",
		"body":     "## Context\nTest context.\n\n## Requirements\nTest reqs.\n\n## Acceptance criteria\n- [ ] Works",
	})

	result, err := handleTicketCreate(context.Background(), root, args)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-001")
	assert.Contains(t, result.Output, "docs/tickets/backlog/T-001-implement-scoring-system.md")

	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-implement-scoring-system.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "title: Implement scoring system")
	assert.Contains(t, string(data), "priority: high")
	assert.Contains(t, string(data), "work_type: feature")
	assert.Contains(t, string(data), "end_to_end_evidence: required")
	assert.Contains(t, string(data), `owner: "TBD"`)
	assert.Contains(t, string(data), `last_attempt: "TBD"`)
	assert.Contains(t, string(data), `blocker: "none"`)
	assert.Contains(t, string(data), "blocked_by: []")
	assert.Contains(t, string(data), `trace_id: "TBD"`)
	assert.Contains(t, string(data), `next_action: "TBD"`)
	assert.Contains(t, string(data), "## Context")
}

func TestTicketCreate_writesBDDOperatingModelMetadata(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	result, err := CreateTicket(root, TicketInput{
		Title:            "Implement first scenario",
		Priority:         "high",
		Complexity:       "small",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S001"},
		EndToEndEvidence: "required",
		EvidenceLinks:    []string{"go test ./internal/serve -run TestBDDOperatingModel"},
		VerifiedBy:       "go test",
		Owner:            "engineer",
		LastAttempt:      "2026-05-03",
		Blocker:          "waiting on dependency",
		BlockedBy:        []string{"T-002"},
		TraceID:          "trace-123",
		NextAction:       "land T-002 first",
		Source:           "current-operating-plan.md — scenario F-001-S001",
		Body:             "## Context\nScenario work.\n\n## Requirements\nImplement it.\n\n## Acceptance criteria\n- [ ] Scenario passes",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-001")

	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-implement-first-scenario.md"))
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, `bdd_scenarios: ["F-001-S001"]`)
	assert.Contains(t, text, `evidence_links: ["go test ./internal/serve -run TestBDDOperatingModel"]`)
	assert.Contains(t, text, `verified_by: "go test"`)
	assert.Contains(t, text, `owner: "engineer"`)
	assert.Contains(t, text, `last_attempt: "2026-05-03"`)
	assert.Contains(t, text, `blocker: "waiting on dependency"`)
	assert.Contains(t, text, `blocked_by: ["T-002"]`)
	assert.Contains(t, text, `trace_id: "trace-123"`)
	assert.Contains(t, text, `next_action: "land T-002 first"`)
}

func TestTicketCreate_stripsDuplicateBodyTitleHeading(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	_, err := CreateTicket(root, TicketInput{
		Title:    "Implement player movement",
		Priority: "high",
		Body:     "# T-999: Implement player movement\n\n## Context\nBuild movement.\n\n## Requirements\nMove left and right.\n\n## Acceptance criteria\n- [ ] Works",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-implement-player-movement.md"))
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(data), "# T-001: Implement player movement"))
	require.NotContains(t, string(data), "# T-999: Implement player movement")
}

func TestTicketCreate_dedupesIndependentFeatureTicketsForSameBDDScenario(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	first := TicketInput{
		Title:        "Create walking skeleton shell",
		Priority:     "high",
		WorkType:     "feature",
		BDDScenarios: []string{"F-001-S001"},
		Source:       "current-operating-plan.md — scenario F-001-S001",
		Body:         "## Context\nFirst slice.\n\n## Requirements\nBuild it.\n\n## Acceptance criteria\n- [ ] Scenario passes",
	}
	result, err := CreateTicket(root, first)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-001")

	second := TicketInput{
		Title:        "Implement player movement",
		Priority:     "high",
		WorkType:     "feature",
		BDDScenarios: []string{"F-001-S001"},
		Source:       "current-operating-plan.md — scenario F-001-S001",
		Body:         "## Context\nSame scenario.\n\n## Requirements\nBuild another part.\n\n## Acceptance criteria\n- [ ] Scenario passes",
	}
	result, err = CreateTicket(root, second)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "DUPLICATE")
	assert.Contains(t, result.Output, "F-001-S001")

	entries, err := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	second.DependsOn = []string{"T-001"}
	result, err = CreateTicket(root, second)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-002")
}

func TestTicketCreate_dedupesActiveFeatureTicketsForOverlappingBDDScenario(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	result, err := CreateTicket(root, TicketInput{
		Title:        "Implement browser access and playfield display",
		Priority:     "high",
		WorkType:     "feature",
		BDDScenarios: []string{"F-001-S002", "F-001-S003"},
		Source:       "current-operating-plan.md",
		Body:         "## Context\nEarly product batch.\n\n## Requirements\nBuild it.\n\n## Acceptance criteria\n- [ ] Scenarios pass",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-001")

	result, err = CreateTicket(root, TicketInput{
		Title:        "Dogfood pre-flight missing core gameplay mechanics",
		Priority:     "high",
		WorkType:     "feature",
		BDDScenarios: []string{"F-001-S003"},
		Source:       "dogfood test 2026-05-22",
		Body:         "## Context\nDogfood observed missing gameplay.\n\n## Requirements\nFix it.\n\n## Acceptance criteria\n- [ ] Scenario passes",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "DUPLICATE")
	assert.Contains(t, result.Output, "F-001-S003")
	assert.Contains(t, result.Output, "T-001")

	entries, err := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestTicketCreate_interventionDebtWritesMetadata(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	result, err := CreateTicket(root, TicketInput{
		Title:      "Intervention debt: engineer context context_overflow",
		Priority:   "high",
		Complexity: "medium",
		Kind:       "intervention-debt",
		DedupeKey:  "intervention-debt:repo-1:engineer:context:context_overflow:24h",
		Metadata: map[string]string{
			"role":     "engineer",
			"repo_id":  "repo-1",
			"severity": "high",
		},
		Source: "telemetry:evt-1",
		Body:   "## Context\nTelemetry.\n\n## Acceptance Criteria\n- [ ] Fixed",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-001")

	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-intervention-debt-engineer-context-context-overflow.md"))
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, "kind: intervention-debt")
	assert.Contains(t, text, "work_type: intervention-debt")
	assert.Contains(t, text, "end_to_end_evidence: not_applicable")
	assert.Contains(t, text, `dedupe_key: "intervention-debt:repo-1:engineer:context:context_overflow:24h"`)
	assert.Contains(t, text, "metadata:")
	assert.Contains(t, text, `  role: "engineer"`)
	assert.Contains(t, text, `  severity: "high"`)
}

func TestTicketCreate_interventionDebtDedupeUpdatesExisting(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	first := TicketInput{
		Title:     "Intervention debt: engineer context context_overflow",
		Priority:  "high",
		Kind:      "intervention-debt",
		DedupeKey: "intervention-debt:repo-1:engineer:context:context_overflow:24h",
		Metadata:  map[string]string{"origin_event_id": "evt-1"},
		Source:    "telemetry:evt-1",
		Body:      "## Context\nTelemetry.\n\n## Acceptance Criteria\n- [ ] Fixed",
	}
	_, err := CreateTicket(root, first)
	require.NoError(t, err)

	second := first
	second.Source = "telemetry:evt-2"
	second.Metadata = map[string]string{"origin_event_id": "evt-2"}
	result, err := CreateTicket(root, second)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "UPDATED")

	entries, err := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "same intervention-debt dedupe key should not create another ticket")

	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", entries[0].Name()))
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, "## Latest Triage Update")
	assert.Contains(t, text, "source: telemetry:evt-2")
	assert.Contains(t, text, "origin_event_id: evt-2")
}

func TestTicketCreate_interventionDebtDedupeCompactsRepeatedUpdates(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	input := TicketInput{
		Title:     "Intervention debt: engineer guardrail guardrail_block",
		Priority:  "medium",
		Kind:      "intervention-debt",
		DedupeKey: "intervention-debt:repo-1:engineer:guardrail:guardrail_block:24h",
		Metadata:  map[string]string{"origin_event_id": "evt-1"},
		Source:    "telemetry:evt-1",
		Body:      "## Context\nTelemetry.\n\n## Acceptance Criteria\n- [ ] Fixed",
	}
	_, err := CreateTicket(root, input)
	require.NoError(t, err)

	for i := 2; i <= 8; i++ {
		next := input
		next.Source = "telemetry:evt-" + strconv.Itoa(i)
		next.Metadata = map[string]string{"origin_event_id": "evt-" + strconv.Itoa(i)}
		_, err := CreateTicket(root, next)
		require.NoError(t, err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", entries[0].Name()))
	require.NoError(t, err)
	text := string(data)
	require.Equal(t, maxLatestTriageUpdates, strings.Count(text, "## Latest Triage Update"))
	require.Contains(t, text, "## Earlier Triage Updates Compacted")
	require.Contains(t, text, "4 earlier triage update(s) compacted")
	require.Contains(t, text, "source: telemetry:evt-8")
	require.NotContains(t, text, "source: telemetry:evt-2")
}

func TestTicketCreate_duplicateBlocked(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	writeTicket(t, dir, "done", "T-003-implement-scoring-system.md", "Implement scoring system")

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "Implement scoring system",
		"priority": "high",
		"body":     "## Context\nDuplicate.\n\n## Requirements\nDupe.\n\n## Acceptance criteria\n- [ ] Dupe",
	})

	result, err := handleTicketCreate(context.Background(), root, args)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "DUPLICATE")
	assert.Contains(t, result.Output, "T-003-implement-scoring-system.md")

	entries, _ := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	assert.Empty(t, entries, "no new file should be created for a duplicate")
}

func TestTicketCreate_duplicateSubsetMatch(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	writeTicket(t, dir, "backlog", "T-005-implement-wave-progression-system.md", "Implement wave progression system")

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "Implement wave progression",
		"priority": "high",
		"body":     "## Context\nShorter title.\n\n## Requirements\nTest.\n\n## Acceptance criteria\n- [ ] Works",
	})

	result, err := handleTicketCreate(context.Background(), root, args)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "DUPLICATE")
}

func TestTicketCreate_autoNumbering(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	writeTicket(t, dir, "done", "T-007-player-movement.md", "Implement player movement")

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "Implement health system",
		"priority": "medium",
		"body":     "## Context\nNew ticket.\n\n## Requirements\nHealth.\n\n## Acceptance criteria\n- [ ] Health works",
	})

	result, err := handleTicketCreate(context.Background(), root, args)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "T-008")
}

func TestTicketCreate_rejectsEmptyTitle(t *testing.T) {
	t.Parallel()
	_, root := setupTicketDir(t)

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "",
		"priority": "high",
		"body":     "## Context\nTest.\n\n## Acceptance criteria\n- [ ] Works",
	})

	_, err := handleTicketCreate(context.Background(), root, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestTicketCreate_rejectsEmptyBody(t *testing.T) {
	t.Parallel()
	_, root := setupTicketDir(t)

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "Valid title",
		"priority": "high",
		"body":     "",
	})

	_, err := handleTicketCreate(context.Background(), root, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "body is required")
}

func TestTicketCreate_acceptsQuotedJSONListBDDScenarios(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)

	result, err := handleTicketCreate(context.Background(), root, []byte(`{
		"title":"Implement CLI",
		"priority":"high",
		"work_type":"feature",
		"bdd_scenarios":"[\"F-001-S002\"]",
		"blocked_by":"[\"T-001\"]",
		"depends_on":"[\"T-001\"]",
		"evidence_links":"[\"go test ./...\"]",
		"body":"## Context\nTest.\n\n## Acceptance criteria\n- [ ] Works"
	}`))

	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-001")
	data, err := os.ReadFile(filepath.Join(dir, "docs", "tickets", "backlog", "T-001-implement-cli.md"))
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, `bdd_scenarios: ["F-001-S002"]`)
	assert.Contains(t, text, `blocked_by: ["T-001"]`)
	assert.Contains(t, text, `depends_on: [T-001]`)
	assert.Contains(t, text, `evidence_links: ["go test ./..."]`)
}

func TestTicketCreate_parseHintForInvalidBDDScenarioString(t *testing.T) {
	t.Parallel()
	_, root := setupTicketDir(t)

	_, err := handleTicketCreate(context.Background(), root, []byte(`{
		"title":"Implement CLI",
		"priority":"high",
		"work_type":"feature",
		"bdd_scenarios":"F-001-S002",
		"body":"## Context\nTest.\n\n## Acceptance criteria\n- [ ] Works"
	}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bdd_scenarios must be a JSON array or a quoted JSON-array string")
	assert.Contains(t, err.Error(), `"bdd_scenarios":["F-001-S002"]`)
}

func TestTicketCreate_differentTopicAllowed(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	writeTicket(t, dir, "done", "T-001-implement-scoring-system.md", "Implement scoring system")

	args, _ := json.Marshal(map[string]interface{}{
		"title":    "Implement collision detection",
		"priority": "high",
		"body":     "## Context\nDifferent topic.\n\n## Requirements\nCollision.\n\n## Acceptance criteria\n- [ ] Collides",
	})

	result, err := handleTicketCreate(context.Background(), root, args)
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-002")
}

func TestNormalizeToWords(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected []string
	}{
		{"Implement wave progression system", []string{"implement", "wave", "progression", "system"}},
		{"implement-wave-progression", []string{"implement", "wave", "progression"}},
		{"Add the basic scoring", []string{"add", "basic", "scoring"}},
		{"", nil},
	}
	for _, tt := range tests {
		words := normalizeToWords(tt.input)
		assert.Equal(t, tt.expected, words, "input: %q", tt.input)
	}
}

func TestIsSubsetMatch(t *testing.T) {
	t.Parallel()
	assert.True(t, isSubsetMatch(
		[]string{"implement", "wave", "progression"},
		[]string{"implement", "wave", "progression", "system"},
	))
	assert.True(t, isSubsetMatch(
		[]string{"implement", "scoring", "system", "component"},
		[]string{"implement", "scoring", "system"},
	))
	assert.False(t, isSubsetMatch(
		[]string{"implement", "collision", "detection"},
		[]string{"implement", "scoring", "system"},
	))
	assert.False(t, isSubsetMatch(
		[]string{"implement", "health", "endpoint", "inventory", "api"},
		[]string{"implement", "list", "items", "endpoint", "inventory", "api"},
	))
	assert.False(t, isSubsetMatch(nil, []string{"implement"}))
	assert.False(t, isSubsetMatch([]string{"implement"}, nil))
}

func TestTicketCreate_allowsDistinctAPIEndpointTitles(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	writeTicket(t, dir, "backlog", "T-001-implement-health-endpoint-for-inventory-api.md",
		"Implement health endpoint for inventory API")

	result, err := CreateTicket(root, TicketInput{
		Title:        "Implement list items endpoint for inventory API",
		Priority:     "high",
		WorkType:     "feature",
		BDDScenarios: []string{"F-001-S002"},
		Source:       "feature contract batch",
		Body:         "## Context\nSecond endpoint.\n\n## Requirements\nList items.\n\n## Acceptance criteria\n- [ ] GET /items",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-002")

	entries, err := os.ReadDir(filepath.Join(dir, "docs", "tickets", "backlog"))
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestTicketCreate_allowsSiblingEnablerTitlesAgainstDoneTicket(t *testing.T) {
	t.Parallel()
	dir, root := setupTicketDir(t)
	writeTicket(t, dir, "done", "T-040-extract-ticket-lifecycle-policy-domain-into-policy-ticket-go-ad-287-step-5.md",
		"Extract ticket-lifecycle policy domain into policy_ticket.go (AD-287 step 5)")

	result, err := CreateTicket(root, TicketInput{
		Title:    "Extract review-gates policy domain into policy_review.go (AD-287 step 6)",
		Priority: "high",
		WorkType: "enabler",
		Source:   "AD-287 extraction sequence",
		Body:     "## Context\nNext domain.\n\n## Requirements\nExtract review gates.\n\n## Acceptance criteria\n- [ ] Suite green",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Output, "created ticket T-041")
}

func TestSlugify(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "implement-wave-progression-system", slugify("Implement wave progression system"))
	assert.Equal(t, "add-collision-detection", slugify("Add collision detection"))
}

func TestScanExistingTickets_emptyDir(t *testing.T) {
	t.Parallel()
	dir, _ := setupTicketDir(t)
	tickets, err := scanExistingTickets(dir)
	require.NoError(t, err)
	assert.Empty(t, tickets)
}

func TestScanExistingTickets_findsAcrossStatuses(t *testing.T) {
	t.Parallel()
	dir, _ := setupTicketDir(t)
	writeTicket(t, dir, "backlog", "T-001-alpha.md", "Alpha")
	writeTicket(t, dir, "in-progress", "T-002-beta.md", "Beta")
	writeTicket(t, dir, "in-review", "T-003-gamma.md", "Gamma")
	writeTicket(t, dir, "done", "T-004-delta.md", "Delta")

	tickets, err := scanExistingTickets(dir)
	require.NoError(t, err)
	assert.Len(t, tickets, 4)

	statuses := map[string]bool{}
	for _, tk := range tickets {
		statuses[tk.Status] = true
	}
	assert.True(t, statuses["backlog"])
	assert.True(t, statuses["in-progress"])
	assert.True(t, statuses["in-review"])
	assert.True(t, statuses["done"])
}

func TestReadTicketTitle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "ticket.md")
	content := "---\nid: T-001\ntitle: My Great Ticket\npriority: high\n---\n\n# T-001: My Great Ticket\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	title := readTicketTitle(path)
	assert.Equal(t, "My Great Ticket", title)
}

func TestTitleFromFilename(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "implement wave progression system", titleFromFilename("T-005-implement-wave-progression-system.md"))
	assert.Equal(t, "alpha", titleFromFilename("T-001-alpha.md"))
}
