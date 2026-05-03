package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssemble_fullSections(t *testing.T) {
	t.Parallel()
	in := Input{
		RoleScope:  "engineer",
		RolePrompt: "You are the engineer role.",
		Guardrails: []Guardrail{
			{Scope: "", Title: "Global", Body: "Be honest."},
			{Scope: "engineer", Title: "Eng", Body: "Run tests."},
		},
		KnowledgeRoutes: []KnowledgeRoute{
			{When: "CI failures", Paths: "docs/ci.md"},
		},
		Trigger:     "Ticket: fix login.",
		RepoSummary: "src/\n  main.go",
	}
	out, stats, err := Assemble(in)
	require.NoError(t, err)
	require.Contains(t, out, "## ROLE")
	require.Contains(t, out, "## GUARDRAILS")
	require.Contains(t, out, "## KNOWLEDGE ROUTES")
	require.Contains(t, out, "When working on CI failures, read docs/ci.md")
	require.Contains(t, out, "## TRIGGER CONTEXT")
	require.Contains(t, out, "## REPO SUMMARY")
	require.Contains(t, out, "### Global\nBe honest.")
	require.Greater(t, len(stats), 0)
}

func TestAssemble_scopeFiltersGuardrails(t *testing.T) {
	t.Parallel()
	in := Input{
		RoleScope:  "engineer",
		RolePrompt: "Role.",
		Guardrails: []Guardrail{
			{Scope: "engineer", Title: "E", Body: "eng only"},
			{Scope: "security", Title: "S", Body: "sec only"},
		},
	}
	out, _, err := Assemble(in)
	require.NoError(t, err)
	require.Contains(t, out, "eng only")
	require.NotContains(t, out, "sec only")
}

func TestAssemble_omitsEmptyOptionalSections(t *testing.T) {
	t.Parallel()
	in := Input{
		RoleScope:  "engineer",
		RolePrompt: "Only role.",
	}
	out, _, err := Assemble(in)
	require.NoError(t, err)
	require.NotContains(t, out, "## KNOWLEDGE ROUTES")
	require.NotContains(t, out, "## TRIGGER CONTEXT")
	require.NotContains(t, out, "## REPO SUMMARY")
	require.Contains(t, out, "## ROLE")
}

func TestAssemble_missingRoleFile(t *testing.T) {
	t.Parallel()
	in := Input{
		RolePromptPath: filepath.Join(t.TempDir(), "does-not-exist.md"),
	}
	_, _, err := Assemble(in)
	require.Error(t, err)
	require.Contains(t, err.Error(), "read role prompt")
}

func TestAssemble_emptyRolePrompt(t *testing.T) {
	t.Parallel()
	_, _, err := Assemble(Input{RoleScope: "x"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "role prompt is empty")
}

func TestAssemble_budgetTruncatesLowerPriority(t *testing.T) {
	t.Parallel()
	role := "ROLE_FIXED_MARKER"
	repo := strings.Repeat("R", 2000)
	trigger := strings.Repeat("T", 2000)
	in := Input{
		RoleScope:   "engineer",
		RolePrompt:  role,
		Trigger:     trigger,
		RepoSummary: repo,
		TokenBudget: 120,
	}
	out, _, err := Assemble(in)
	require.NoError(t, err)
	require.Contains(t, out, role)
	require.Contains(t, out, "truncated")
	require.Less(t, strings.Count(out, "R"), strings.Count(repo, "R"))
}

func TestAssemble_roleFromFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "role.md")
	require.NoError(t, os.WriteFile(p, []byte("  from file\n"), 0o644))
	out, _, err := Assemble(Input{RolePromptPath: p})
	require.NoError(t, err)
	require.Contains(t, out, "from file")
}

func TestAssemble_ticketIndex(t *testing.T) {
	t.Parallel()
	in := Input{
		RoleScope:   "coo",
		RolePrompt:  "You are the COO.",
		TicketIndex: "Existing tickets (3 total):\n- [done] T-001-player-movement.md\n- [in-progress] T-002-shooting.md\n- [backlog] T-003-scoring.md",
	}
	out, stats, err := Assemble(in)
	require.NoError(t, err)
	require.Contains(t, out, "## TICKET INDEX")
	require.Contains(t, out, "T-001-player-movement.md")
	require.Contains(t, out, "T-003-scoring.md")

	var found bool
	for _, s := range stats {
		if s.Name == "tickets" {
			found = true
			require.Greater(t, s.Tokens, 0)
		}
	}
	require.True(t, found, "expected 'tickets' section in stats")
}

func TestAssemble_ticketIndexOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	in := Input{
		RoleScope:  "coo",
		RolePrompt: "You are the COO.",
	}
	out, _, err := Assemble(in)
	require.NoError(t, err)
	require.NotContains(t, out, "## TICKET INDEX")
}

func TestAssemble_payloadModeInTriggerContext(t *testing.T) {
	t.Parallel()
	in := Input{
		RoleScope:   "janitor",
		RolePrompt:  "You are the janitor.",
		PayloadMode: "ticket_hygiene",
		Trigger:     `{"type":"orchestrator.survey"}`,
	}
	out, _, err := Assemble(in)
	require.NoError(t, err)
	require.Contains(t, out, "## TRIGGER CONTEXT")
	require.Contains(t, out, "payload_mode: ticket_hygiene")
	require.Contains(t, out, `"type":"orchestrator.survey"`)
}
