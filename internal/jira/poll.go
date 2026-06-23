/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
*/
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PollConfig struct {
	Repository Repository
	EnvLookup  func(string) (string, bool)
	HTTPClient *http.Client
}

func Poll(ctx context.Context, cfg PollConfig) ([]MirrorResult, error) {
	repo := cfg.Repository
	if !repo.Config.JIRAEnabled() {
		return []MirrorResult{{
			Status:          StatusDisabled,
			Reason:          "jira_ingestion_disabled",
			LLMJobsEnqueued: 0,
		}}, nil
	}
	if cfg.EnvLookup == nil {
		cfg.EnvLookup = os.LookupEnv
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := buildSearchRequest(ctx, repo, cfg.EnvLookup)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira: poll request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("jira: read poll response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira: poll returned HTTP %d", resp.StatusCode)
	}
	rawIssues, err := RawIssuesFromSearchPayload(body)
	if err != nil {
		return nil, err
	}
	results := make([]MirrorResult, 0, len(rawIssues))
	for _, raw := range rawIssues {
		result, err := MirrorRawIssue(ctx, []Repository{repo}, raw)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func buildSearchRequest(ctx context.Context, repo Repository, lookup func(string) (string, bool)) (*http.Request, error) {
	jiraCfg := repo.Config.Ingestion.JIRA
	baseURL := strings.TrimRight(strings.TrimSpace(jiraCfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("jira: poll disabled - ingestion.jira.base_url is required")
	}
	email, err := requiredEnvValue(jiraCfg.Auth.EmailEnv, "email_env", lookup)
	if err != nil {
		return nil, err
	}
	token, err := requiredEnvValue(jiraCfg.Auth.APITokenEnv, "api_token_env", lookup)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(baseURL + "/rest/api/3/search/jql")
	if err != nil {
		return nil, fmt.Errorf("jira: invalid base_url: %w", err)
	}
	query := endpoint.Query()
	if strings.TrimSpace(jiraCfg.JQL) != "" {
		query.Set("jql", jiraCfg.JQL)
	}
	query.Set("maxResults", "50")
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("jira: build poll request: %w", err)
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func requiredEnvValue(envName, field string, lookup func(string) (string, bool)) (string, error) {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return "", fmt.Errorf("jira: poll disabled - ingestion.jira.auth.%s is required", field)
	}
	value, ok := lookup(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("jira: poll disabled - set env var %s", envName)
	}
	return value, nil
}

func SearchPayloadForTest(issues ...map[string]any) []byte {
	body, _ := json.Marshal(map[string]any{"issues": issues})
	return body
}

func ParsePollInterval(value string) (time.Duration, bool) {
	interval, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || interval <= 0 {
		return 0, false
	}
	return interval, true
}
