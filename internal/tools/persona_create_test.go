/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersonaCreateCreatesManualPromptRegistryAndManifestRole(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, ".harness", "manifest.yaml"), []byte(`name: test
description: test
orchestration_mode: dispatch
roles: {}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "docs", "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "docs", "roles", "ROLES.md"), []byte(`# Role Registry

| Role | Origin | Domain | Mode | Trigger sources | Schedule | Tools | Trust level | Guardrails | Model routing | Scoring signals | Escalation behavior |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
`), 0o644))
	root, err := NewRoot(rootDir)
	require.NoError(t, err)

	_, err = handlePersonaCreate(context.Background(), root, []byte(`{
	  "role_key":"research-lead",
	  "title":"Research Lead",
	  "scope":"deployed",
	  "domain":"planner",
	  "mode":"research-advisory",
	  "category":"optional-advisory",
	  "modus_operandi":"Turn uncertain questions into evidence-backed recommendations.",
	  "priorities":["evidence quality"],
	  "owns":["research memos"],
	  "does_not_own":["implementation"],
	  "best_feedback_format":["question, audience, constraints"],
	  "feedback_i_need":["the decision the research supports"],
	  "feedback_i_give":["recommendation with confidence"],
	  "stop_conditions":["the next action is implementation"],
	  "orchestrator_handoff":["route decisions back to CEO"],
	  "activate":true,
	  "tools":["file_read","grep","record_decision","job_disposition_record"]
	}`))
	require.NoError(t, err)

	manual, err := os.ReadFile(filepath.Join(rootDir, "docs", "roles", "personas", "research-lead.md"))
	require.NoError(t, err)
	require.Contains(t, string(manual), "# Research Lead Persona")
	require.Contains(t, string(manual), "## Owns")
	require.Contains(t, string(manual), "## Stop Conditions")

	prompt, err := os.ReadFile(filepath.Join(rootDir, ".harness", "roles", "research-lead.md"))
	require.NoError(t, err)
	require.Contains(t, string(prompt), "## Personal Guide")
	require.Contains(t, string(prompt), "### How I Like To Receive Feedback")

	registry, err := os.ReadFile(filepath.Join(rootDir, "docs", "roles", "ROLES.md"))
	require.NoError(t, err)
	require.Contains(t, string(registry), "| `research-lead` | custom | planner | `research-advisory` |")

	manifest, err := os.ReadFile(filepath.Join(rootDir, ".harness", "manifest.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(manifest), "research-lead:")
	require.Contains(t, string(manifest), "prompt: roles/research-lead.md")
}

func TestPersonaCreateRejectsMissingRequiredManualSections(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	root, err := NewRoot(rootDir)
	require.NoError(t, err)

	_, err = handlePersonaCreate(context.Background(), root, []byte(`{
	  "role_key":"research-lead",
	  "title":"Research Lead",
	  "scope":"deployed",
	  "domain":"planner",
	  "mode":"research-advisory",
	  "category":"optional-advisory",
	  "modus_operandi":"Turn uncertain questions into evidence-backed recommendations.",
	  "priorities":["evidence quality"],
	  "owns":[],
	  "does_not_own":["implementation"],
	  "best_feedback_format":["question, audience, constraints"],
	  "feedback_i_need":["the decision the research supports"],
	  "feedback_i_give":["recommendation with confidence"],
	  "stop_conditions":["the next action is implementation"],
	  "orchestrator_handoff":["route decisions back to CEO"]
	}`))
	require.ErrorContains(t, err, "owns is required")
}
