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
	"fmt"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/integrations"
)

func resolveEnvBackedConfig(cfg integrations.Config, lookup func(string) (string, bool)) (integrations.Config, error) {
	if lookup == nil || !cfg.JIRAEnabled() {
		return cfg, nil
	}
	jiraCfg := cfg.Ingestion.JIRA
	var err error
	if jiraCfg.BaseURLEnv != "" {
		if jiraCfg.BaseURL, err = requiredIntegrationEnvValue(jiraCfg.BaseURLEnv, "ingestion.jira.base_url_env", lookup); err != nil {
			return cfg, err
		}
		jiraCfg.BaseURL = strings.TrimRight(jiraCfg.BaseURL, "/")
	}
	if jiraCfg.MCP.CloudIDEnv != "" {
		if jiraCfg.MCP.CloudID, err = requiredIntegrationEnvValue(jiraCfg.MCP.CloudIDEnv, "ingestion.jira.mcp.cloud_id_env", lookup); err != nil {
			return cfg, err
		}
	}
	if jiraCfg.MCP.SiteURLEnv != "" {
		if jiraCfg.MCP.SiteURL, err = requiredIntegrationEnvValue(jiraCfg.MCP.SiteURLEnv, "ingestion.jira.mcp.site_url_env", lookup); err != nil {
			return cfg, err
		}
		jiraCfg.MCP.SiteURL = strings.TrimRight(jiraCfg.MCP.SiteURL, "/")
	}
	if jiraCfg.Scope.AllowedWorkspacesEnv != "" {
		value, err := requiredIntegrationEnvValue(jiraCfg.Scope.AllowedWorkspacesEnv, "ingestion.jira.scope.allowed_workspaces_env", lookup)
		if err != nil {
			return cfg, err
		}
		jiraCfg.Scope.AllowedWorkspaces = cleanStringList(append(jiraCfg.Scope.AllowedWorkspaces, splitEnvList(value)...))
	}
	if jiraCfg.Scope.BoardIDEnv != "" {
		if jiraCfg.Scope.BoardID, err = requiredIntegrationEnvValue(jiraCfg.Scope.BoardIDEnv, "ingestion.jira.scope.board_id_env", lookup); err != nil {
			return cfg, err
		}
	}
	if jiraCfg.Fields.SprintEnv != "" {
		if jiraCfg.Fields.Sprint, err = requiredIntegrationEnvValue(jiraCfg.Fields.SprintEnv, "ingestion.jira.fields.sprint_env", lookup); err != nil {
			return cfg, err
		}
	}
	if jiraCfg.Fields.RankEnv != "" {
		if jiraCfg.Fields.Rank, err = requiredIntegrationEnvValue(jiraCfg.Fields.RankEnv, "ingestion.jira.fields.rank_env", lookup); err != nil {
			return cfg, err
		}
	}
	if jiraCfg.Fields.EpicLinkEnv != "" {
		if jiraCfg.Fields.EpicLink, err = requiredIntegrationEnvValue(jiraCfg.Fields.EpicLinkEnv, "ingestion.jira.fields.epic_link_env", lookup); err != nil {
			return cfg, err
		}
	}
	if jiraCfg.Fields.StoryPointsEnv != "" {
		if jiraCfg.Fields.StoryPoints, err = requiredIntegrationEnvValue(jiraCfg.Fields.StoryPointsEnv, "ingestion.jira.fields.story_points_env", lookup); err != nil {
			return cfg, err
		}
	}
	cfg.Ingestion.JIRA = jiraCfg
	return cfg, nil
}

func resolveEnvBackedRepositories(repos []Repository, lookup func(string) (string, bool)) ([]Repository, error) {
	out := make([]Repository, len(repos))
	copy(out, repos)
	for i := range out {
		cfg, err := resolveEnvBackedConfig(out[i].Config, lookup)
		if err != nil {
			return nil, err
		}
		out[i].Config = cfg
	}
	return out, nil
}

func requiredIntegrationEnvValue(envName, field string, lookup func(string) (string, bool)) (string, error) {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return "", fmt.Errorf("jira: configured env-backed field %s is empty", field)
	}
	value, ok := lookup(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("jira: configured env-backed field %s requires env var %s", field, envName)
	}
	return strings.TrimSpace(value), nil
}

func splitEnvList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
}
