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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/integrations"
)

func TestLiveAtlassianMCPJiraIntake(t *testing.T) {
	if os.Getenv("MARS_JIRA_LIVE") != "1" {
		t.Skip("set MARS_JIRA_LIVE=1 to run live Atlassian MCP Jira intake smoke")
	}
	repoRoot := t.TempDir()
	repoName := filepath.Base(repoRoot)
	siteURLEnv := envNameOrDefault("MARS_JIRA_SITE_URL_ENV", "MARS_JIRA_SITE_URL")
	if strings.TrimSpace(os.Getenv(siteURLEnv)) == "" {
		t.Fatalf("live Atlassian MCP smoke requires %s to contain the Jira site URL", siteURLEnv)
	}
	cfg := integrations.Defaults()
	cfg.FlowProfile = integrations.FlowProfileBoardDriven
	cfg.Ingestion.JIRA.Enabled = true
	cfg.Ingestion.JIRA.Provider = integrations.JIRAProviderAtlassianMCP
	cfg.Ingestion.JIRA.BaseURLEnv = siteURLEnv
	cfg.Ingestion.JIRA.Auth.EmailEnv = envNameOrDefault("MARS_JIRA_EMAIL_ENV", "JIRA_USERNAME")
	cfg.Ingestion.JIRA.Auth.APITokenEnv = envNameOrDefault("MARS_JIRA_API_TOKEN_ENV", "JIRA_API_TOKEN")
	cfg.Ingestion.JIRA.Auth.BearerTokenEnv = strings.TrimSpace(os.Getenv("MARS_JIRA_BEARER_ENV"))
	cfg.Ingestion.JIRA.MCP.EndpointURL = envOrDefault("MARS_JIRA_MCP_ENDPOINT", "https://mcp.atlassian.com/v1/mcp")
	cfg.Ingestion.JIRA.MCP.CloudID = strings.TrimSpace(os.Getenv("MARS_JIRA_CLOUD_ID"))
	cfg.Ingestion.JIRA.MCP.SiteURLEnv = siteURLEnv
	cfg.Ingestion.JIRA.MCP.Timeout = "30s"
	if os.Getenv("MARS_JIRA_MCP_STDIO_PROXY") == "1" {
		cfg.Ingestion.JIRA.MCP.Proxy.Enabled = true
		cfg.Ingestion.JIRA.MCP.Proxy.Transport = integrations.JIRAMCPProxyTransportStdio
		cfg.Ingestion.JIRA.MCP.Proxy.Command = envOrDefault("MARS_JIRA_MCP_PROXY_COMMAND", "npx")
		cfg.Ingestion.JIRA.MCP.Proxy.Args = mcpRemoteArgs(cfg.Ingestion.JIRA.MCP.EndpointURL)
	}
	cfg.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "DEMO", Repo: repoName}}
	cfg.Ingestion.JIRA.Scope.AllowedWorkspacesEnv = envNameOrDefault("MARS_JIRA_ALLOWED_WORKSPACES_ENV", "MARS_JIRA_ALLOWED_WORKSPACES")
	cfg.Ingestion.JIRA.Scope.RequiredLabels = []string{"example-required-label"}
	cfg.Ingestion.JIRA.Scope.BoardIDEnv = envNameOrDefault("MARS_JIRA_BOARD_ID_ENV", "MARS_JIRA_BOARD_ID")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	results, err := Poll(ctx, PollConfig{
		Repository: Repository{ID: "live-repo", Path: repoRoot, Config: cfg},
	})
	if err != nil {
		t.Fatalf("live atlassian mcp poll failed: %v", err)
	}
	for _, result := range results {
		if result.Status == StatusCreated || result.Status == StatusReconciled {
			if result.LLMJobsEnqueued != 0 {
				t.Fatalf("jira live poll enqueued LLM work: %#v", result)
			}
			if got := countMarkdownTickets(t, repoRoot); got == 0 {
				t.Fatalf("live poll reported mirrored ticket but none exist: %#v", results)
			}
			return
		}
	}
	t.Fatalf("live atlassian mcp poll returned no mirrored scoped issues; results=%#v", results)
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envNameOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func mcpRemoteArgs(endpoint string) []string {
	if raw := strings.TrimSpace(os.Getenv("MARS_JIRA_MCP_PROXY_ARGS")); raw != "" {
		return strings.Fields(raw)
	}
	return []string{"-y", "mcp-remote", endpoint}
}
