package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketDir(t *testing.T) (string, Root) {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"docs/tickets/backlog", "docs/tickets/in-progress", "docs/tickets/done"} {
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
	assert.Contains(t, string(data), "## Context")
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
	assert.Contains(t, text, `dedupe_key: "public-example"`)
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
	assert.False(t, isSubsetMatch(nil, []string{"implement"}))
	assert.False(t, isSubsetMatch([]string{"implement"}, nil))
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
	writeTicket(t, dir, "done", "T-003-gamma.md", "Gamma")

	tickets, err := scanExistingTickets(dir)
	require.NoError(t, err)
	assert.Len(t, tickets, 3)

	statuses := map[string]bool{}
	for _, tk := range tickets {
		statuses[tk.Status] = true
	}
	assert.True(t, statuses["backlog"])
	assert.True(t, statuses["in-progress"])
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
