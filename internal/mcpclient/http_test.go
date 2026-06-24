/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
*/
package mcpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPClientLifecycle(t *testing.T) {
	t.Parallel()
	var sawAuth bool
	var sawSession bool
	var closed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer test-token" {
			sawAuth = true
		}
		switch r.Method {
		case http.MethodDelete:
			closed = r.Header.Get("Mcp-Session-Id") == "session-1"
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodPost:
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		if req["method"] != "initialize" && r.Header.Get("Mcp-Session-Id") == "session-1" {
			sawSession = true
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "session-1")
		switch req["method"] {
		case "initialize":
			writeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": defaultProtocolVersion})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeRPCResult(t, w, req["id"], map[string]any{"tools": []map[string]any{{"name": "searchJiraIssuesUsingJql"}}})
		case "tools/call":
			writeRPCResult(t, w, req["id"], map[string]any{"content": []map[string]any{{"type": "text", "text": `{"ok":true}`}}})
		default:
			t.Fatalf("unexpected method %v", req["method"])
		}
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, Authorization: "Bearer test-token", HTTPClient: server.Client()}
	require.NoError(t, client.Initialize(context.Background()))
	tools, err := client.ListTools(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Tool{{Name: "searchJiraIssuesUsingJql"}}, tools)
	raw, err := client.CallTool(context.Background(), "searchJiraIssuesUsingJql", map[string]any{"jql": "project = DEMO"})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"content"`)
	require.NoError(t, client.Close(context.Background()))
	require.True(t, sawAuth)
	require.True(t, sawSession)
	require.True(t, closed)
}

func TestHTTPClientParsesSSEToolResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		w.Header().Set("Content-Type", "text/event-stream")
		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result":  map[string]any{"tools": []map[string]any{{"name": "getJiraIssue"}}},
		}
		data, err := json.Marshal(response)
		require.NoError(t, err)
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	tools, err := client.ListTools(context.Background())
	require.NoError(t, err)
	require.Equal(t, "getJiraIssue", tools[0].Name)
}

func TestHTTPClientReportsRPCError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"error": map[string]any{
				"code":    -32000,
				"message": "permission denied",
			},
		}))
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := client.ListTools(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestHTTPClientTruncatesHTTPErrorBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, strings.Repeat("x", 800), http.StatusUnauthorized)
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := client.ListTools(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 401")
	require.Contains(t, err.Error(), "[truncated]")
}

func TestStdioClientLifecycle(t *testing.T) {
	t.Parallel()
	client, err := StartStdio(context.Background(), StdioConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestStdioClientHelperProcess"},
		Env:     append(os.Environ(), "MCPCLIENT_STDIO_HELPER=1"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close(context.Background()))
	})
	require.NoError(t, client.Initialize(context.Background()))
	tools, err := client.ListTools(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Tool{{Name: "searchJiraIssuesUsingJql"}}, tools)
	raw, err := client.CallTool(context.Background(), "searchJiraIssuesUsingJql", map[string]any{"jql": "project = DEMO"})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"content"`)
}

func TestStdioClientRequiresCommand(t *testing.T) {
	t.Parallel()
	_, err := StartStdio(context.Background(), StdioConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "command is required")
}

func TestStdioClientRejectsEmptyToolName(t *testing.T) {
	t.Parallel()
	client, err := StartStdio(context.Background(), StdioConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestStdioClientHelperProcess"},
		Env:     append(os.Environ(), "MCPCLIENT_STDIO_HELPER=1"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close(context.Background()))
	})
	_, err = client.CallTool(context.Background(), " ", map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "tool name is required")
}

func TestStdioClientReportsRPCError(t *testing.T) {
	t.Parallel()
	client, err := StartStdio(context.Background(), StdioConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestStdioClientHelperProcess"},
		Env:     append(os.Environ(), "MCPCLIENT_STDIO_HELPER=1", "MCPCLIENT_STDIO_HELPER_MODE=rpc-error"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close(context.Background()))
	})
	_, err = client.ListTools(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestStdioClientReportsMalformedOutput(t *testing.T) {
	t.Parallel()
	client, err := StartStdio(context.Background(), StdioConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestStdioClientHelperProcess"},
		Env:     append(os.Environ(), "MCPCLIENT_STDIO_HELPER=1", "MCPCLIENT_STDIO_HELPER_MODE=malformed"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close(context.Background()))
	})
	_, err = client.ListTools(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "read response")
}

func TestResponseIDMatches(t *testing.T) {
	t.Parallel()
	require.True(t, responseIDMatches(json.RawMessage(`7`), 7))
	require.True(t, responseIDMatches(json.RawMessage(`"7"`), 7))
	require.False(t, responseIDMatches(json.RawMessage(`8`), 7))
	require.False(t, responseIDMatches(nil, 7))
	require.False(t, responseIDMatches(json.RawMessage(`{}`), 7))
}

func TestStdioClientHelperProcess(t *testing.T) {
	if os.Getenv("MCPCLIENT_STDIO_HELPER") != "1" {
		return
	}
	defer os.Exit(0)
	switch os.Getenv("MCPCLIENT_STDIO_HELPER_MODE") {
	case "malformed":
		_, _ = os.Stdout.WriteString("not-json\n")
		select {}
	}
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
			writeStdioRPCResult(enc, id, map[string]any{"protocolVersion": defaultProtocolVersion})
		case "notifications/initialized":
		case "tools/list":
			if os.Getenv("MCPCLIENT_STDIO_HELPER_MODE") == "rpc-error" {
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": "permission denied"}})
				continue
			}
			writeStdioRPCResult(enc, id, map[string]any{"tools": []map[string]any{{"name": "searchJiraIssuesUsingJql"}}})
		case "tools/call":
			writeStdioRPCResult(enc, id, map[string]any{"content": []map[string]any{{"type": "text", "text": `{"ok":true}`}}})
		default:
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": "unexpected method"}})
		}
	}
}

func writeStdioRPCResult(enc *json.Encoder, id any, result any) {
	_ = enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeRPCResult(t *testing.T, w http.ResponseWriter, id any, result any) {
	t.Helper()
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}))
}
