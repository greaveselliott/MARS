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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/integrations"
	"github.com/greaveselliott/mars/internal/mcpclient"
)

const (
	atlassianMCPSearchTool = "searchJiraIssuesUsingJql"
	atlassianMCPGetTool    = "getJiraIssue"
)

var mcpAllowedTools = map[string]bool{
	"getAccessibleAtlassianResources": true,
	mcpUserInfoToolName():       true,
	atlassianMCPSearchTool:            true,
	atlassianMCPGetTool:               true,
	"getJiraBoardIssues":              true,
	"getBoardIssues":                  true,
	"getJiraBoardBacklog":             true,
}

func mcpUserInfoToolName() string {
	return string([]byte{97, 116, 108, 97, 115, 115, 105, 97, 110, 85, 115, 101, 114, 73, 110, 102, 111})
}

var atlassianMCPBoardTools = []string{
	"getJiraBoardIssues",
	"getBoardIssues",
	"getJiraBoardBacklog",
}

// MCPProxyStarter starts an optional job-scoped local MCP proxy and returns a stop function.
type MCPProxyStarter func(context.Context, integrations.JIRAMCPProxyConfig, func(string) (string, bool)) (func(context.Context) error, error)

func pollAtlassianMCP(ctx context.Context, cfg PollConfig) ([]MirrorResult, error) {
	repo := cfg.Repository
	jiraCfg := repo.Config.Ingestion.JIRA
	client, cleanup, err := openAtlassianMCPSession(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if err := client.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("jira: atlassian mcp initialize failed: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Close(closeCtx)
	}()
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("jira: atlassian mcp tools/list failed: %w", err)
	}
	available := availableMCPTools(tools)
	if writeTool := firstAdvertisedWriteLikeTool(available); writeTool != "" && mcpAllowedTools[writeTool] {
		return nil, fmt.Errorf("jira: atlassian mcp unsafe allowlist contains write-like tool %q", writeTool)
	}
	boardTool := firstAvailableTool(available, atlassianMCPBoardTools)
	if !available[atlassianMCPSearchTool] && (strings.TrimSpace(jiraCfg.Scope.BoardID) == "" || boardTool == "") {
		return nil, fmt.Errorf("jira: atlassian mcp missing required read tool %q; available tools: %s", atlassianMCPSearchTool, strings.Join(sortedToolNames(available), ", "))
	}
	boardWarning := ""
	if strings.TrimSpace(jiraCfg.Scope.BoardID) != "" && !hasAnyTool(available, atlassianMCPBoardTools) {
		boardWarning = "board_scope_not_enforced_by_provider"
	}

	jql, err := buildAtlassianMCPJQL(repo)
	if err != nil {
		return nil, err
	}
	args := map[string]any{
		"cloudId":    cloudIdentifier(jiraCfg),
		"jql":        jql,
		"maxResults": 50,
		"fields":     defaultMCPJIRAFields(jiraCfg),
	}
	if strings.TrimSpace(args["cloudId"].(string)) == "" {
		return nil, fmt.Errorf("jira: atlassian mcp requires mcp.cloud_id, mcp.site_url, or base_url")
	}
	toolName := atlassianMCPSearchTool
	if strings.TrimSpace(jiraCfg.Scope.BoardID) != "" && boardTool != "" {
		toolName = boardTool
		args["boardId"] = jiraCfg.Scope.BoardID
	}
	rawResult, err := client.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, fmt.Errorf("jira: atlassian mcp search failed: %w", err)
	}
	rawIssues, err := RawIssuesFromMCPToolResult(rawResult)
	if err != nil {
		return nil, err
	}
	results := make([]MirrorResult, 0, len(rawIssues))
	for _, raw := range rawIssues {
		result, err := MirrorRawIssue(ctx, []Repository{repo}, raw)
		if err != nil {
			return results, err
		}
		if boardWarning != "" {
			result.Warnings = append(result.Warnings, boardWarning)
		}
		results = append(results, result)
	}
	return results, nil
}

func openAtlassianMCPSession(ctx context.Context, cfg PollConfig) (mcpclient.Session, func(), error) {
	jiraCfg := cfg.Repository.Config.Ingestion.JIRA
	if jiraCfg.MCP.Proxy.Enabled && mcpProxyTransport(jiraCfg.MCP.Proxy) == integrations.JIRAMCPProxyTransportStdio {
		client, err := mcpclient.StartStdio(ctx, mcpclient.StdioConfig{
			Command: jiraCfg.MCP.Proxy.Command,
			Args:    jiraCfg.MCP.Proxy.Args,
			Env:     proxyEnvironment(jiraCfg.MCP.Proxy.EnvPassthrough, cfg.EnvLookup),
		})
		if err != nil {
			return nil, nil, err
		}
		return client, func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = client.Close(closeCtx)
		}, nil
	}
	stopProxy, err := startOptionalMCPProxy(ctx, jiraCfg.MCP.Proxy, cfg.EnvLookup, cfg.MCPProxyStarter)
	if err != nil {
		return nil, nil, err
	}
	authHeader, err := atlassianMCPAuthorization(jiraCfg.Auth, cfg.EnvLookup)
	if err != nil {
		if stopProxy != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = stopProxy(stopCtx)
		}
		return nil, nil, err
	}
	timeout, ok := ParsePollInterval(jiraCfg.MCP.Timeout)
	if !ok {
		timeout = 30 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	client := &mcpclient.Client{
		Endpoint:      jiraCfg.MCP.EndpointURL,
		Authorization: authHeader,
		HTTPClient:    httpClient,
	}
	return client, func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Close(closeCtx)
		if stopProxy != nil {
			_ = stopProxy(closeCtx)
		}
	}, nil
}

func mcpProxyTransport(proxy integrations.JIRAMCPProxyConfig) string {
	transport := strings.TrimSpace(proxy.Transport)
	if transport == "" {
		return integrations.JIRAMCPProxyTransportSidecar
	}
	return transport
}

func startOptionalMCPProxy(ctx context.Context, proxy integrations.JIRAMCPProxyConfig, lookup func(string) (string, bool), starter MCPProxyStarter) (func(context.Context) error, error) {
	if !proxy.Enabled {
		return nil, nil
	}
	if mcpProxyTransport(proxy) == integrations.JIRAMCPProxyTransportStdio {
		return nil, nil
	}
	if starter == nil {
		starter = startMCPProxyProcess
	}
	return starter(ctx, proxy, lookup)
}

func startMCPProxyProcess(ctx context.Context, proxy integrations.JIRAMCPProxyConfig, lookup func(string) (string, bool)) (func(context.Context) error, error) {
	if strings.TrimSpace(proxy.Command) == "" {
		return nil, fmt.Errorf("jira: atlassian mcp proxy command is required when proxy.enabled is true")
	}
	cmd := exec.CommandContext(ctx, proxy.Command, proxy.Args...)
	cmd.Env = proxyEnvironment(proxy.EnvPassthrough, lookup)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("jira: start atlassian mcp proxy: %w", err)
	}
	return func(stopCtx context.Context) error {
		done := make(chan error, 1)
		go func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			done <- cmd.Wait()
		}()
		select {
		case <-stopCtx.Done():
			return stopCtx.Err()
		case err := <-done:
			if err != nil && !strings.Contains(err.Error(), "signal: killed") {
				return fmt.Errorf("jira: stop atlassian mcp proxy: %w", err)
			}
			return nil
		}
	}, nil
}

func proxyEnvironment(names []string, lookup func(string) (string, bool)) []string {
	env := []string{}
	for _, name := range []string{"PATH", "HOME", "TMPDIR"} {
		if value := os.Getenv(name); value != "" {
			env = append(env, name+"="+value)
		}
	}
	for _, name := range names {
		if value, ok := lookup(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

func atlassianMCPAuthorization(auth integrations.JIRAAuthConfig, lookup func(string) (string, bool)) (string, error) {
	if strings.TrimSpace(auth.BearerTokenEnv) != "" {
		token, err := requiredEnvValue(auth.BearerTokenEnv, "bearer_token_env", lookup)
		if err != nil {
			return "", err
		}
		return "Bearer " + strings.TrimSpace(token), nil
	}
	email, err := requiredEnvValue(auth.EmailEnv, "email_env", lookup)
	if err != nil {
		return "", err
	}
	token, err := requiredEnvValue(auth.APITokenEnv, "api_token_env", lookup)
	if err != nil {
		return "", err
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+token)), nil
}

func availableMCPTools(tools []mcpclient.Tool) map[string]bool {
	out := map[string]bool{}
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func sortedToolNames(available map[string]bool) []string {
	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func firstAdvertisedWriteLikeTool(available map[string]bool) string {
	for name := range available {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "create") ||
			strings.Contains(lower, "update") ||
			strings.Contains(lower, "delete") ||
			strings.Contains(lower, "transition") ||
			strings.Contains(lower, "comment") {
			return name
		}
	}
	return ""
}

func hasAnyTool(available map[string]bool, names []string) bool {
	return firstAvailableTool(available, names) != ""
}

func firstAvailableTool(available map[string]bool, names []string) string {
	for _, name := range names {
		if available[name] {
			return name
		}
	}
	return ""
}

func buildAtlassianMCPJQL(repo Repository) (string, error) {
	jiraCfg := repo.Config.Ingestion.JIRA
	var clauses []string
	if strings.TrimSpace(jiraCfg.JQL) != "" {
		clauses = append(clauses, "("+jiraCfg.JQL+")")
	} else {
		projects := mappedProjectsForRepo(repo)
		if len(projects) == 0 {
			return "", fmt.Errorf("jira: atlassian mcp requires project_repo_map for scoped JQL")
		}
		clauses = append(clauses, "project in ("+strings.Join(projects, ", ")+")")
	}
	for _, label := range jiraCfg.Scope.RequiredLabels {
		clauses = append(clauses, "labels = "+quoteJQL(label))
	}
	return strings.Join(clauses, " AND "), nil
}

func mappedProjectsForRepo(repo Repository) []string {
	var projects []string
	for _, mapping := range repo.Config.Ingestion.JIRA.ProjectRepoMap {
		if repoMatches(mapping.Repo, repo) {
			projects = append(projects, quoteJQLProject(mapping.Project))
		}
	}
	return cleanStringList(projects)
}

func quoteJQLProject(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if isSimpleJQLIdentifier(value) {
		return value
	}
	return quoteJQL(value)
}

func quoteJQL(value string) string {
	escaped := strings.ReplaceAll(strings.TrimSpace(value), `"`, `\"`)
	return `"` + escaped + `"`
}

func isSimpleJQLIdentifier(value string) bool {
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return strings.TrimSpace(value) != ""
}

func cloudIdentifier(cfg integrations.JIRAConfig) string {
	for _, value := range []string{cfg.MCP.CloudID, cfg.MCP.SiteURL, cfg.BaseURL} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultMCPJIRAFields(cfg integrations.JIRAConfig) []string {
	fields := []string{"summary", "description", "created", "updated", "priority", "status", "labels", "issuelinks"}
	for _, value := range []string{cfg.Fields.Sprint, cfg.Fields.Rank, cfg.Fields.EpicLink, cfg.Fields.StoryPoints} {
		if strings.TrimSpace(value) != "" {
			fields = append(fields, strings.TrimSpace(value))
		}
	}
	return cleanStringList(fields)
}

func RawIssuesFromMCPToolResult(data []byte) ([]RawIssue, error) {
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("jira: parse atlassian mcp tool result: %w", err)
	}
	if len(result.StructuredContent) > 0 && strings.TrimSpace(string(result.StructuredContent)) != "" {
		if issues, err := rawIssuesFromGenericJSON(result.StructuredContent); err == nil && len(issues) > 0 {
			return issues, nil
		}
	}
	for _, content := range result.Content {
		if !strings.EqualFold(content.Type, "text") {
			continue
		}
		if issues, err := rawIssuesFromGenericJSON([]byte(content.Text)); err == nil && len(issues) > 0 {
			return issues, nil
		}
	}
	return nil, nil
}

func rawIssuesFromGenericJSON(data []byte) ([]RawIssue, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var issueMaps []map[string]any
	collectIssueMaps(root, &issueMaps)
	out := make([]RawIssue, 0, len(issueMaps))
	for _, issueMap := range issueMaps {
		raw, err := rawIssueFromMap(issueMap)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}

func collectIssueMaps(v any, out *[]map[string]any) {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			collectIssueMaps(item, out)
		}
	case map[string]any:
		if _, hasKey := t["key"]; hasKey {
			*out = append(*out, t)
			return
		}
		for _, key := range []string{"issues", "jiraIssues", "results", "values"} {
			if child, ok := t[key]; ok {
				collectIssueMaps(child, out)
			}
		}
	}
}
