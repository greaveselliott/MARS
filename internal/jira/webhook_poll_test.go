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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/integrations"
)

func TestWebhookDisabledWithoutEnabledConfigReturnsNotFound(t *testing.T) {
	handler := WebhookHandler(WebhookConfig{
		Repositories: func(context.Context) ([]Repository, error) {
			return []Repository{{ID: "repo-1", Path: t.TempDir(), Config: integrations.Defaults()}}, nil
		},
		EnvLookup: func(string) (string, bool) { return "", false },
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected disabled webhook to return 404, got %d", rec.Code)
	}
}

func TestWebhookRequiresSignatureAndMirrorsMappedIssue(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	body := webhookPayload(t, map[string]any{
		"key":  "DEMO-20",
		"self": "https://jira.example/browse/DEMO-20?token=raw-url-token",
		"fields": map[string]any{
			"project":            map[string]any{"key": "DEMO"},
			"summary":            "Mirror signed webhook",
			"description":        "Build webhook path with token=raw-body-token",
			"created":            "2026-06-23T08:00:00.000+0000",
			"updated":            "2026-06-23T09:00:00.000+0000",
			"priority":           map[string]any{"name": "P1"},
			"status":             map[string]any{"name": "Ready for Dev"},
			"customfield_sprint": []any{map[string]any{"name": "Sprint 1", "state": "active"}},
			"customfield_rank":   "0|abc:",
			"customfield_epic":   "DEMO-EPIC",
			"blocked_by":         []any{"DEMO-19"},
		},
	})
	handler := WebhookHandler(WebhookConfig{
		Repositories: func(context.Context) ([]Repository, error) {
			return []Repository{{ID: "repo-1", Path: repoRoot, Config: cfg}}, nil
		},
		EnvLookup: func(name string) (string, bool) {
			if name == "JIRA_WEBHOOK_SECRET" {
				return "webhook-secret", true
			}
			return "", false
		},
	})

	unsigned := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))
	unsigned.Header.Set("X-Atlassian-Webhook-Identifier", "delivery-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, unsigned)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected unsigned webhook to fail, got %d", rec.Code)
	}
	assertNoMarkdownTickets(t, repoRoot)

	signed := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))
	signed.Header.Set("X-Atlassian-Webhook-Identifier", "delivery-2")
	signed.Header.Set(jiraSignatureHeader, SignWebhookPayloadForTest("webhook-secret", body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, signed)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected signed webhook to succeed, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"llm_jobs_enqueued":0`) {
		t.Fatalf("webhook response must prove no LLM enqueue, got %s", rec.Body.String())
	}
	if got := countMarkdownTickets(t, repoRoot); got != 1 {
		t.Fatalf("expected one mirrored ticket, got %d", got)
	}
	ticket := onlyTicketContent(t, repoRoot)
	for _, leaked := range []string{"raw-url-token", "raw-body-token"} {
		if strings.Contains(ticket, leaked) {
			t.Fatalf("ticket leaked secret-like value %q:\n%s", leaked, ticket)
		}
	}
	for _, want := range []string{`token=[redacted]`, `jira_key: "DEMO-20"`, `blocked_by: ["DEMO-19"]`} {
		if !strings.Contains(ticket, want) {
			t.Fatalf("ticket missing %q:\n%s", want, ticket)
		}
	}

	duplicate := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))
	duplicate.Header.Set("X-Atlassian-Webhook-Identifier", "delivery-2")
	duplicate.Header.Set(jiraSignatureHeader, SignWebhookPayloadForTest("webhook-secret", body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, duplicate)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duplicate webhook to return 409, got %d", rec.Code)
	}
	if got := countMarkdownTickets(t, repoRoot); got != 1 {
		t.Fatalf("duplicate webhook created tickets, got %d", got)
	}
}

func TestWebhookUsesMappedRepoSecret(t *testing.T) {
	repoOne := t.TempDir()
	repoTwo := t.TempDir()
	cfgOne := boardDrivenConfig(filepath.Base(repoOne))
	cfgOne.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "ALPHA", Repo: filepath.Base(repoOne)}}
	cfgOne.Ingestion.JIRA.WebhookSecretEnv = "ALPHA_SECRET"
	cfgTwo := boardDrivenConfig(filepath.Base(repoTwo))
	cfgTwo.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "DEMO", Repo: filepath.Base(repoTwo)}}
	cfgTwo.Ingestion.JIRA.WebhookSecretEnv = "ZOO_SECRET"
	body := webhookPayload(t, map[string]any{
		"key": "ALPHA-21",
		"fields": map[string]any{
			"project": map[string]any{"key": "ALPHA"},
			"summary": "Repo-specific secret",
		},
	})
	handler := WebhookHandler(WebhookConfig{
		Repositories: func(context.Context) ([]Repository, error) {
			return []Repository{
				{ID: "repo-1", Path: repoOne, Config: cfgOne},
				{ID: "repo-2", Path: repoTwo, Config: cfgTwo},
			}, nil
		},
		EnvLookup: func(name string) (string, bool) {
			switch name {
			case "ALPHA_SECRET":
				return "alpha-secret", true
			case "ZOO_SECRET":
				return "zoo-secret", true
			default:
				return "", false
			}
		},
	})

	wrongSecret := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))
	wrongSecret.Header.Set("X-Atlassian-Webhook-Identifier", "delivery-secret-1")
	wrongSecret.Header.Set(jiraSignatureHeader, SignWebhookPayloadForTest("zoo-secret", body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, wrongSecret)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected wrong repo secret to fail, got %d", rec.Code)
	}
	assertNoMarkdownTickets(t, repoOne)
	assertNoMarkdownTickets(t, repoTwo)

	rightSecret := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))
	rightSecret.Header.Set("X-Atlassian-Webhook-Identifier", "delivery-secret-2")
	rightSecret.Header.Set(jiraSignatureHeader, SignWebhookPayloadForTest("alpha-secret", body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, rightSecret)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected mapped repo secret to succeed, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := countMarkdownTickets(t, repoOne); got != 1 {
		t.Fatalf("expected repo one ticket, got %d", got)
	}
	assertNoMarkdownTickets(t, repoTwo)
}

func TestPollFetchesAndMirrorsMappedIssueWithoutQueueWork(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	cfg.Ingestion.JIRA.Auth.EmailEnv = "JIRA_EMAIL"
	cfg.Ingestion.JIRA.Auth.APITokenEnv = "JIRA_TOKEN"
	var sawAuth bool
	cfg.Ingestion.JIRA.BaseURL = "https://jira.example"
	cfg.Ingestion.JIRA.JQL = "project = DEMO"
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected poll method %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected poll path %s", r.URL.Path)
		}
		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode poll body: %v", err)
		}
		if got := requestBody["jql"]; got != "project = DEMO" {
			t.Fatalf("unexpected poll jql %q", got)
		}
		user, pass, ok := r.BasicAuth()
		sawAuth = ok && user == "agent@example.com" && pass == "poll-token"
		body := SearchPayloadForTest(map[string]any{
			"key": "DEMO-30",
			"fields": map[string]any{
				"project":     map[string]any{"key": "DEMO"},
				"summary":     "Mirror poll result",
				"description": "Poll-created requirement",
				"created":     "2026-06-23T08:00:00.000+0000",
				"updated":     "2026-06-23T10:00:00.000+0000",
				"priority":    map[string]any{"name": "P2"},
				"status":      map[string]any{"name": "To Do"},
			},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    r,
		}, nil
	})}

	results, err := Poll(context.Background(), PollConfig{
		Repository: Repository{ID: "repo-1", Path: repoRoot, Config: cfg},
		EnvLookup: func(name string) (string, bool) {
			switch name {
			case "JIRA_EMAIL":
				return "agent@example.com", true
			case "JIRA_TOKEN":
				return "poll-token", true
			default:
				return "", false
			}
		},
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if !sawAuth {
		t.Fatalf("poll request did not use configured env-backed basic auth")
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("expected one created result, got %#v", results)
	}
	if results[0].LLMJobsEnqueued != 0 {
		t.Fatalf("jira poll must not enqueue LLM jobs, got %d", results[0].LLMJobsEnqueued)
	}
	ticket := onlyTicketContent(t, repoRoot)
	for _, want := range []string{`jira_key: "DEMO-30"`, `priority: "P2"`, "Poll-created requirement"} {
		if !strings.Contains(ticket, want) {
			t.Fatalf("poll ticket missing %q:\n%s", want, ticket)
		}
	}
}

func TestPollMissingEnvNamesSecretVarNotValue(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	cfg.Ingestion.JIRA.BaseURL = "https://jira.example"
	cfg.Ingestion.JIRA.Auth.EmailEnv = "JIRA_EMAIL"
	cfg.Ingestion.JIRA.Auth.APITokenEnv = "JIRA_TOKEN"
	_, err := Poll(context.Background(), PollConfig{
		Repository: Repository{ID: "repo-1", Path: repoRoot, Config: cfg},
		EnvLookup: func(name string) (string, bool) {
			if name == "JIRA_EMAIL" {
				return "agent@example.com", true
			}
			return "", false
		},
	})
	if err == nil {
		t.Fatal("expected missing token env to fail closed")
	}
	if !strings.Contains(err.Error(), "JIRA_TOKEN") {
		t.Fatalf("expected error to name missing env var, got %v", err)
	}
	if strings.Contains(err.Error(), "poll-token") {
		t.Fatalf("error leaked secret value: %v", err)
	}
}

func webhookPayload(t *testing.T, issue map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"issue": issue})
	if err != nil {
		t.Fatalf("marshal webhook: %v", err)
	}
	return body
}

func onlyTicketContent(t *testing.T, repoRoot string) string {
	t.Helper()
	var contents []string
	root := filepath.Join(repoRoot, "docs", "tickets")
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		contents = append(contents, readFile(t, path))
		return nil
	})
	if len(contents) != 1 {
		t.Fatalf("expected exactly one ticket content, got %d", len(contents))
	}
	return contents[0]
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
