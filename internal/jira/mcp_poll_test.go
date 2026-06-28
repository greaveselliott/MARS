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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greaveselliott/mars/internal/integrations"
)

func TestPollAtlassianMCPMirrorsScopedIssueAndClosesSession(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := atlassianMCPConfig(repoRoot)
	server := newFakeAtlassianMCPServer(t, fakeMCPOptions{
		Tools: []string{atlassianMCPSearchTool, atlassianMCPGetTool, "addCommentToJiraIssue"},
		Result: map[string]any{"issues": []map[string]any{{
			"key":         "DEMO-4181",
			"project":     "DEMO",
			"summary":     "Scoped opportunity",
			"description": "Mirror this labelled issue.",
			"created":     "2026-06-24T08:00:00.000+0000",
			"updated":     "2026-06-24T09:00:00.000+0000",
			"priority":    map[string]any{"name": "P1"},
			"status":      map[string]any{"name": "Ready for Dev"},
			"labels":      []string{"example-required-label"},
			"browseUrl":   "https://jira.example/browse/DEMO-4181",
		}}},
	})
	defer server.Close()
	cfg.Ingestion.JIRA.MCP.EndpointURL = server.URL

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
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("expected one created result, got %#v", results)
	}
	if results[0].LLMJobsEnqueued != 0 {
		t.Fatalf("jira mcp poll must not enqueue LLM jobs, got %d", results[0].LLMJobsEnqueued)
	}
	if len(results[0].Warnings) != 1 || results[0].Warnings[0] != "board_scope_not_enforced_by_provider" {
		t.Fatalf("expected board warning on created result, got %#v", results[0].Warnings)
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:poll-token"))
	if server.authorization != expectedAuth {
		t.Fatalf("unexpected auth header %q", server.authorization)
	}
	if !server.closed {
		t.Fatalf("mcp session was not closed")
	}
	if server.calledTool != atlassianMCPSearchTool {
		t.Fatalf("expected search read tool call, got %q", server.calledTool)
	}
	if strings.Contains(strings.ToLower(server.calledTool), "comment") {
		t.Fatalf("write-like tool was called: %s", server.calledTool)
	}
	if !strings.Contains(server.calledArgs["jql"].(string), `labels = "example-required-label"`) {
		t.Fatalf("scoped JQL did not include required label: %#v", server.calledArgs)
	}
	fields, ok := server.calledArgs["fields"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatalf("expected fields array for Atlassian MCP schema, got %#v", server.calledArgs["fields"])
	}
	ticket := onlyTicketContent(t, repoRoot)
	for _, want := range []string{`jira_key: "DEMO-4181"`, `priority: "P1"`, "Mirror this labelled issue."} {
		if !strings.Contains(ticket, want) {
			t.Fatalf("mcp ticket missing %q:\n%s", want, ticket)
		}
	}
}

func TestPollAtlassianMCPUsesBoardToolWhenAvailable(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := atlassianMCPConfig(repoRoot)
	server := newFakeAtlassianMCPServer(t, fakeMCPOptions{
		Tools: []string{atlassianMCPSearchTool, "getJiraBoardIssues"},
		Result: map[string]any{"issues": []map[string]any{{
			"key":     "DEMO-4182",
			"project": "DEMO",
			"summary": "Board scoped issue",
			"labels":  []string{"example-required-label"},
		}}},
	})
	defer server.Close()
	cfg.Ingestion.JIRA.MCP.EndpointURL = server.URL

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
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if server.calledTool != "getJiraBoardIssues" {
		t.Fatalf("expected board read tool call, got %q", server.calledTool)
	}
	if server.calledArgs["boardId"] != "board-example" {
		t.Fatalf("expected board id board-example, got %#v", server.calledArgs)
	}
	if len(results) != 1 || len(results[0].Warnings) != 0 {
		t.Fatalf("expected board-enforced result without warning, got %#v", results)
	}
}

func TestPollAtlassianMCPUsesStdioProxyWithoutStaticAuth(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := atlassianMCPConfig(repoRoot)
	cfg.Ingestion.JIRA.MCP.Proxy.Enabled = true
	cfg.Ingestion.JIRA.MCP.Proxy.Transport = integrations.JIRAMCPProxyTransportStdio
	cfg.Ingestion.JIRA.MCP.Proxy.Command = os.Args[0]
	cfg.Ingestion.JIRA.MCP.Proxy.Args = []string{"-test.run=TestJiraStdioMCPHelperProcess"}
	cfg.Ingestion.JIRA.MCP.Proxy.EnvPassthrough = []string{"JIRA_STDIO_MCP_HELPER"}

	results, err := Poll(context.Background(), PollConfig{
		Repository: Repository{ID: "repo-1", Path: repoRoot, Config: cfg},
		EnvLookup: func(name string) (string, bool) {
			if name == "JIRA_STDIO_MCP_HELPER" {
				return "1", true
			}
			return "", false
		},
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("expected one created result, got %#v", results)
	}
	ticket := onlyTicketContent(t, repoRoot)
	for _, want := range []string{`jira_key: "DEMO-9001"`, `priority: "P2"`, "Mirrored through stdio proxy."} {
		if !strings.Contains(ticket, want) {
			t.Fatalf("stdio mcp ticket missing %q:\n%s", want, ticket)
		}
	}
}

func TestJiraStdioMCPHelperProcess(t *testing.T) {
	if os.Getenv("JIRA_STDIO_MCP_HELPER") != "1" {
		return
	}
	defer os.Exit(0)
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id := req["id"]
		switch req["method"] {
		case "initialize":
			writeFakeStdioRPCResult(enc, id, map[string]any{"protocolVersion": "2025-06-18"})
		case "notifications/initialized":
		case "tools/list":
			writeFakeStdioRPCResult(enc, id, map[string]any{"tools": []map[string]any{{"name": atlassianMCPSearchTool}}})
		case "tools/call":
			writeFakeStdioRPCResult(enc, id, map[string]any{
				"content": []map[string]any{{
					"type": "text",
					"text": `{"issues":[{"key":"DEMO-9001","project":"DEMO","summary":"Stdio scoped opportunity","description":"Mirrored through stdio proxy.","priority":{"name":"P2"},"status":{"name":"Backlog"},"labels":["example-required-label"]}]}`,
				}},
			})
		default:
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": "unexpected method"}})
		}
	}
}

func TestPollAtlassianMCPDropsMissingRequiredLabel(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := atlassianMCPConfig(repoRoot)
	server := newFakeAtlassianMCPServer(t, fakeMCPOptions{
		Tools: []string{atlassianMCPSearchTool},
		Result: map[string]any{"issues": []map[string]any{{
			"key":     "DEMO-4183",
			"project": "DEMO",
			"summary": "Wrong label",
			"labels":  []string{"other"},
		}}},
	})
	defer server.Close()
	cfg.Ingestion.JIRA.MCP.EndpointURL = server.URL

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
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(results) != 1 || results[0].Status != StatusDropped || results[0].Reason != "scope_required_label_missing" {
		t.Fatalf("expected missing label drop, got %#v", results)
	}
	assertNoMarkdownTickets(t, repoRoot)
}

func TestPollAtlassianMCPMissingReadToolFailsClosedAndStopsProxy(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := atlassianMCPConfig(repoRoot)
	cfg.Ingestion.JIRA.MCP.Proxy.Enabled = true
	server := newFakeAtlassianMCPServer(t, fakeMCPOptions{
		Tools:  []string{"transitionJiraIssue"},
		Result: map[string]any{"issues": []map[string]any{}},
	})
	defer server.Close()
	cfg.Ingestion.JIRA.MCP.EndpointURL = server.URL
	var proxyStarted bool
	var proxyStopped bool

	_, err := Poll(context.Background(), PollConfig{
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
		HTTPClient: server.Client(),
		MCPProxyStarter: func(context.Context, integrations.JIRAMCPProxyConfig, func(string) (string, bool)) (func(context.Context) error, error) {
			proxyStarted = true
			return func(context.Context) error {
				proxyStopped = true
				return nil
			}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "missing required read tool") {
		t.Fatalf("expected missing read tool error, got %v", err)
	}
	if !proxyStarted || !proxyStopped {
		t.Fatalf("expected proxy start and stop, started=%t stopped=%t", proxyStarted, proxyStopped)
	}
	assertNoMarkdownTickets(t, repoRoot)
}

func atlassianMCPConfig(repoRoot string) integrations.Config {
	cfg := boardDrivenConfig(filepath.Base(repoRoot))
	cfg.Ingestion.JIRA.Provider = integrations.JIRAProviderAtlassianMCP
	cfg.Ingestion.JIRA.ProjectRepoMap = []integrations.ProjectRepoMapping{{Project: "DEMO", Repo: filepath.Base(repoRoot)}}
	cfg.Ingestion.JIRA.Auth.EmailEnv = "JIRA_EMAIL"
	cfg.Ingestion.JIRA.Auth.APITokenEnv = "JIRA_TOKEN"
	cfg.Ingestion.JIRA.BaseURL = "https://jira.example"
	cfg.Ingestion.JIRA.MCP.EndpointURL = "https://mcp.atlassian.com/v1/mcp"
	cfg.Ingestion.JIRA.MCP.CloudID = "cloud-example"
	cfg.Ingestion.JIRA.Scope.BoardID = "board-example"
	cfg.Ingestion.JIRA.Scope.AllowedWorkspaces = []string{"https://jira.example/jira/software/c/projects/DEMO/boards/board-example/backlog"}
	cfg.Ingestion.JIRA.Scope.RequiredLabels = []string{"example-required-label"}
	return cfg
}

type fakeMCPOptions struct {
	Tools  []string
	Result map[string]any
}

type fakeAtlassianMCPServer struct {
	*httptest.Server
	authorization string
	closed        bool
	calledTool    string
	calledArgs    map[string]any
}

func newFakeAtlassianMCPServer(t *testing.T, opts fakeMCPOptions) *fakeAtlassianMCPServer {
	t.Helper()
	state := &fakeAtlassianMCPServer{}
	state.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			state.authorization = got
		}
		if r.Method == http.MethodDelete {
			state.closed = r.Header.Get("Mcp-Session-Id") == "session-1"
			w.WriteHeader(http.StatusOK)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode mcp request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "session-1")
		switch req["method"] {
		case "initialize":
			writeFakeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": "2025-06-18"})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			tools := make([]map[string]any, 0, len(opts.Tools))
			for _, name := range opts.Tools {
				tools = append(tools, map[string]any{"name": name})
			}
			writeFakeRPCResult(t, w, req["id"], map[string]any{"tools": tools})
		case "tools/call":
			params, _ := req["params"].(map[string]any)
			state.calledTool, _ = params["name"].(string)
			state.calledArgs, _ = params["arguments"].(map[string]any)
			resultData, err := json.Marshal(opts.Result)
			if err != nil {
				t.Fatalf("marshal fake result: %v", err)
			}
			writeFakeRPCResult(t, w, req["id"], map[string]any{
				"content": []map[string]any{{"type": "text", "text": string(resultData)}},
			})
		default:
			t.Fatalf("unexpected mcp method %#v", req["method"])
		}
	}))
	return state
}

func writeFakeRPCResult(t *testing.T, w http.ResponseWriter, id any, result any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}); err != nil {
		t.Fatalf("encode fake rpc result: %v", err)
	}
}

func writeFakeStdioRPCResult(enc *json.Encoder, id any, result any) {
	_ = enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}
