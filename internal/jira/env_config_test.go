/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
- docs/runbooks/atlassian-mcp-jira-intake.md
*/
package jira

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEnvBackedConfigUsesIDEnvVars(t *testing.T) {
	cfg := boardDrivenConfig("repo")
	cfg.Ingestion.JIRA.BaseURL = ""
	cfg.Ingestion.JIRA.BaseURLEnv = "JIRA_BASE_URL"
	cfg.Ingestion.JIRA.MCP.CloudIDEnv = "JIRA_CLOUD_ID"
	cfg.Ingestion.JIRA.MCP.SiteURLEnv = "JIRA_SITE_URL"
	cfg.Ingestion.JIRA.Scope.AllowedWorkspacesEnv = "JIRA_ALLOWED_WORKSPACES"
	cfg.Ingestion.JIRA.Scope.BoardIDEnv = "JIRA_BOARD_ID"
	cfg.Ingestion.JIRA.Fields.SprintEnv = "JIRA_FIELD_SPRINT"
	cfg.Ingestion.JIRA.Fields.RankEnv = "JIRA_FIELD_RANK"
	cfg.Ingestion.JIRA.Fields.EpicLinkEnv = "JIRA_FIELD_EPIC"
	cfg.Ingestion.JIRA.Fields.StoryPointsEnv = "JIRA_FIELD_STORY_POINTS"

	resolved, err := resolveEnvBackedConfig(cfg, func(name string) (string, bool) {
		values := map[string]string{
			"JIRA_BASE_URL":           "https://jira.example.invalid",
			"JIRA_CLOUD_ID":           "cloud-example-env",
			"JIRA_SITE_URL":           "https://jira.example.invalid",
			"JIRA_ALLOWED_WORKSPACES": "https://jira.example.invalid",
			"JIRA_BOARD_ID":           "board-example-env",
			"JIRA_FIELD_SPRINT":       "customfield_sprint_env",
			"JIRA_FIELD_RANK":         "customfield_rank_env",
			"JIRA_FIELD_EPIC":         "customfield_epic_env",
			"JIRA_FIELD_STORY_POINTS": "customfield_story_points_env",
		}
		value, ok := values[name]
		return value, ok
	})
	if err != nil {
		t.Fatalf("resolveEnvBackedConfig: %v", err)
	}
	jiraCfg := resolved.Ingestion.JIRA
	if jiraCfg.BaseURL != "https://jira.example.invalid" {
		t.Fatalf("base url not resolved: %#v", jiraCfg)
	}
	if jiraCfg.MCP.CloudID != "cloud-example-env" {
		t.Fatalf("cloud id not resolved: %#v", jiraCfg.MCP)
	}
	if jiraCfg.MCP.SiteURL != "https://jira.example.invalid" {
		t.Fatalf("site url not resolved: %#v", jiraCfg.MCP)
	}
	if jiraCfg.Scope.BoardID != "board-example-env" {
		t.Fatalf("board id not resolved: %#v", jiraCfg.Scope)
	}
	if len(jiraCfg.Scope.AllowedWorkspaces) != 2 {
		t.Fatalf("allowed workspaces not resolved: %#v", jiraCfg.Scope.AllowedWorkspaces)
	}
	if jiraCfg.Fields.Sprint != "customfield_sprint_env" ||
		jiraCfg.Fields.Rank != "customfield_rank_env" ||
		jiraCfg.Fields.EpicLink != "customfield_epic_env" ||
		jiraCfg.Fields.StoryPoints != "customfield_story_points_env" {
		t.Fatalf("field IDs not resolved: %#v", jiraCfg.Fields)
	}
}

func TestResolveEnvBackedConfigFailsClosedWhenIDEnvMissing(t *testing.T) {
	cfg := boardDrivenConfig("repo")
	cfg.Ingestion.JIRA.MCP.CloudIDEnv = "JIRA_CLOUD_ID"
	_, err := resolveEnvBackedConfig(cfg, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatalf("expected missing env-backed ID to fail closed")
	}
}

func TestPollAtlassianMCPUsesEnvBackedIDs(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := atlassianMCPConfig(repoRoot)
	cfg.Ingestion.JIRA.MCP.CloudID = ""
	cfg.Ingestion.JIRA.MCP.CloudIDEnv = "JIRA_CLOUD_ID"
	cfg.Ingestion.JIRA.BaseURL = ""
	cfg.Ingestion.JIRA.BaseURLEnv = "JIRA_BASE_URL"
	cfg.Ingestion.JIRA.MCP.SiteURL = ""
	cfg.Ingestion.JIRA.MCP.SiteURLEnv = "JIRA_SITE_URL"
	cfg.Ingestion.JIRA.Scope.AllowedWorkspaces = nil
	cfg.Ingestion.JIRA.Scope.AllowedWorkspacesEnv = "JIRA_ALLOWED_WORKSPACES"
	cfg.Ingestion.JIRA.Scope.BoardID = ""
	cfg.Ingestion.JIRA.Scope.BoardIDEnv = "JIRA_BOARD_ID"
	server := newFakeAtlassianMCPServer(t, fakeMCPOptions{
		Tools: []string{atlassianMCPSearchTool, "getJiraBoardIssues"},
		Result: map[string]any{"issues": []map[string]any{{
			"key":     "DEMO-9100",
			"project": "DEMO",
			"self":    "https://api.atlassian.com/ex/jira/cloud-example-env/rest/api/3/issue/9100",
			"summary": "Env-backed ID issue",
			"labels":  []string{"example-required-label"},
		}}},
	})
	defer server.Close()
	cfg.Ingestion.JIRA.MCP.EndpointURL = server.URL

	results, err := Poll(context.Background(), PollConfig{
		Repository: Repository{ID: "repo-1", Path: repoRoot, Config: cfg},
		EnvLookup: func(name string) (string, bool) {
			values := map[string]string{
				"JIRA_EMAIL":              "agent@example.com",
				"JIRA_TOKEN":              "poll-token",
				"JIRA_BASE_URL":           "https://jira.example.invalid",
				"JIRA_CLOUD_ID":           "cloud-example-env",
				"JIRA_SITE_URL":           "https://jira.example.invalid",
				"JIRA_ALLOWED_WORKSPACES": "https://jira.example.invalid",
				"JIRA_BOARD_ID":           "board-example-env",
			}
			value, ok := values[name]
			return value, ok
		},
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("expected env-backed issue to create, got %#v", results)
	}
	if server.calledArgs["cloudId"] != "cloud-example-env" {
		t.Fatalf("expected env-backed cloud id, got %#v", server.calledArgs)
	}
	if server.calledArgs["boardId"] != "board-example-env" {
		t.Fatalf("expected env-backed board id, got %#v", server.calledArgs)
	}
	ticket := onlyTicketContent(t, repoRoot)
	if !containsAll(ticket, `jira_key: "DEMO-9100"`, "Env-backed ID issue") {
		t.Fatalf("ticket missing env-backed issue content:\n%s", ticket)
	}
}

func TestWebhookUsesEnvBackedWorkspaceContainment(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	cfg.Ingestion.JIRA.BaseURL = "https://jira.example"
	cfg.Ingestion.JIRA.Scope.AllowedWorkspacesEnv = "JIRA_ALLOWED_WORKSPACES"
	issue := Issue{
		Key:         "DEMO-ENV",
		Project:     "DEMO",
		URL:         "https://api.atlassian.com/ex/jira/cloud-example-env/rest/api/3/issue/1",
		Summary:     "Env workspace",
		Description: "Env workspace containment.",
		Labels:      []string{"allowed-intake"},
	}
	cfg.Ingestion.JIRA.Scope.RequiredLabels = []string{"allowed-intake"}
	repos, err := resolveEnvBackedRepositories([]Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, func(name string) (string, bool) {
		if name == "JIRA_ALLOWED_WORKSPACES" {
			return "https://jira.example/jira/software/c/projects/DEMO/boards/board-example-env/backlog", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("resolveEnvBackedRepositories: %v", err)
	}
	result, err := MirrorIssue(context.Background(), repos, issue)
	if err != nil {
		t.Fatalf("MirrorIssue: %v", err)
	}
	if result.Status != StatusCreated {
		t.Fatalf("expected env-backed workspace issue to create, got %#v", result)
	}
}

func containsAll(text string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(text, value) {
			return false
		}
	}
	return true
}
