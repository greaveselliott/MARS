/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
*/
package integrations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_missingConfigDefaultsDisabled(t *testing.T) {
	t.Parallel()
	cfg, err := Load(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Version)
	require.Equal(t, FlowProfileCEOLed, cfg.FlowProfile)
	require.False(t, cfg.BoardDriven())
	require.False(t, cfg.JIRAEnabled())
	require.False(t, cfg.JiraEnabled())
	require.False(t, cfg.FigmaEnabled())
	require.False(t, cfg.DeliveryEnabled())
	require.False(t, cfg.PullRequestDelivery())
	require.Empty(t, cfg.EnabledSections())
	require.False(t, cfg.SectionEnabled(SectionJIRA))
	require.Equal(t, DeliveryModeTrunk, cfg.Delivery.Mode)
}

func TestLoad_boardDrivenConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeConfig(t, root, `version: 1
flow_profile: board-driven
ingestion:
  jira:
    enabled: true
    base_url: https://jira.example.invalid
    auth:
      email_env: JIRA_EMAIL
      api_token_env: JIRA_API_TOKEN
    webhook_secret_env: JIRA_WEBHOOK_SECRET
    poll_interval: 2m
    jql: project = DEMO
    project_repo_map:
      - { project: DEMO, repo: owner/repo }
    scope:
      allowed_workspaces:
        - https://jira.example/jira/software/c/projects/DEMO/boards/42/backlog
      required_labels:
        - allowed-intake
    fields:
      sprint: <jira-custom-field-id>
      rank: <jira-custom-field-id>
    prioritisation:
      ready_statuses: ["Ready"]
design_sources:
  figma:
    enabled: true
    token_env: FIGMA_TOKEN
delivery:
  mode: pull_request
  min_trust: autonomous
`)

	cfg, err := Load(root)
	require.NoError(t, err)
	require.True(t, cfg.BoardDriven())
	require.True(t, cfg.JIRAEnabled())
	require.True(t, cfg.JiraEnabled())
	require.True(t, cfg.FigmaEnabled())
	require.True(t, cfg.DeliveryEnabled())
	require.True(t, cfg.PullRequestDelivery())
	require.Equal(t, []string{SectionJIRA, SectionFigma, SectionDelivery}, cfg.EnabledSections())
	require.True(t, cfg.SectionEnabled(SectionJIRA))
	require.True(t, cfg.SectionEnabled(SectionFigma))
	require.True(t, cfg.SectionEnabled(SectionDelivery))
	require.False(t, cfg.SectionEnabled("unknown"))
	require.Equal(t, "2m", cfg.Ingestion.JIRA.PollInterval)
	require.Equal(t, []string{"https://jira.example/jira/software/c/projects/DEMO/boards/42/backlog"}, cfg.Ingestion.JIRA.Scope.AllowedWorkspaces)
	require.Equal(t, []string{"allowed-intake"}, cfg.Ingestion.JIRA.Scope.RequiredLabels)
	require.Equal(t, []string{"Ready"}, cfg.Ingestion.JIRA.Prioritisation.ReadyStatuses)
	require.Equal(t, []string{"priority", "rank", "age"}, cfg.Ingestion.JIRA.Prioritisation.Order)
	require.Equal(t, "https://api.figma.com", cfg.DesignSources.Figma.BaseURL)
	require.Equal(t, "autonomous", cfg.Delivery.MinTrust)
}

func TestLoad_unknownProfileFailsSafe(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeConfig(t, root, `version: 1
flow_profile: experimental
ingestion:
  jira:
    enabled: true
delivery:
  mode: pull_request
`)

	cfg, err := Load(root)
	require.NoError(t, err)
	require.Equal(t, FlowProfileCEOLed, cfg.FlowProfile)
	require.False(t, cfg.BoardDriven())
	require.False(t, cfg.JIRAEnabled())
	require.False(t, cfg.DeliveryEnabled())
	require.False(t, cfg.PullRequestDelivery())
	require.Empty(t, cfg.EnabledSections())
}

func TestLoad_emptyProfileFailsSafe(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeConfig(t, root, `version: 1
flow_profile:
`)

	cfg, err := Load(root)
	require.NoError(t, err)
	require.Equal(t, FlowProfileCEOLed, cfg.FlowProfile)
	require.False(t, cfg.BoardDriven())
}

func TestLoad_toleratesUnknownFields(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeConfig(t, root, `version: 1
unknown: value
flow_profile: board-driven
ingestion:
  jira:
    enabled: false
    another_unknown: value
`)

	cfg, err := Load(root)
	require.NoError(t, err)
	require.True(t, cfg.BoardDriven())
}

func TestConfig_suppressesOnlyPlanningSchedulesWhenBoardDriven(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	require.False(t, cfg.SuppressesSchedule("ceo"))

	cfg.FlowProfile = FlowProfileBoardDriven
	for _, role := range []string{"ceo", "coo", "head-of-strategy", "cto-weekly"} {
		require.True(t, cfg.SuppressesSchedule(role), role)
	}
	for _, role := range []string{"engineer", "qa", "security", "dogfood", "release-manager", "janitor", "orchestrator"} {
		require.False(t, cfg.SuppressesSchedule(role), role)
	}
}

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, integrationsPath), []byte(body), 0o644))
}
