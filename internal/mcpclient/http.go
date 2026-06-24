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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultProtocolVersion = "2025-06-18"
	defaultMaxResponse     = 8 << 20
)

// Client speaks MCP over streamable HTTP for short-lived integration jobs.
type Client struct {
	Endpoint        string
	Authorization   string
	ProtocolVersion string
	HTTPClient      *http.Client
	MaxResponse     int64

	sessionID string
	nextID    atomic.Int64
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// Session is the common lifecycle used by short-lived MCP integration jobs.
type Session interface {
	Initialize(context.Context) error
	ListTools(context.Context) ([]Tool, error)
	CallTool(context.Context, string, any) (json.RawMessage, error)
	Close(context.Context) error
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Initialize(ctx context.Context) error {
	_, err := c.doRPC(ctx, "initialize", map[string]any{
		"protocolVersion": c.protocolVersion(),
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mars-harness",
			"version": "integration-client",
		},
	})
	if err != nil {
		return err
	}
	return c.doNotification(ctx, "notifications/initialized", map[string]any{})
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	raw, err := c.doRPC(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("mcp: parse tools/list result: %w", err)
	}
	return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args any) (json.RawMessage, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("mcp: tool name is required")
	}
	return c.doRPC(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
}

func (c *Client) Close(ctx context.Context) error {
	if strings.TrimSpace(c.sessionID) == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint(), nil)
	if err != nil {
		return fmt.Errorf("mcp: build close request: %w", err)
	}
	c.applyHeaders(req, false)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("mcp: close session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp: close session returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) doRPC(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	resp, err := c.send(ctx, rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}, true)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcp: %s failed: %s", method, resp.Error.Message)
	}
	return resp.Result, nil
}

func (c *Client) doNotification(ctx context.Context, method string, params any) error {
	_, err := c.send(ctx, rpcRequest{JSONRPC: "2.0", Method: method, Params: params}, false)
	return err
}

func (c *Client) send(ctx context.Context, payload rpcRequest, expectResponse bool) (rpcResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rpcResponse{}, fmt.Errorf("mcp: encode %s request: %w", payload.Method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return rpcResponse{}, fmt.Errorf("mcp: build %s request: %w", payload.Method, err)
	}
	c.applyHeaders(req, true)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return rpcResponse{}, fmt.Errorf("mcp: %s request failed: %w", payload.Method, err)
	}
	defer resp.Body.Close()
	if sessionID := strings.TrimSpace(resp.Header.Get("Mcp-Session-Id")); sessionID != "" {
		c.sessionID = sessionID
	}
	if !expectResponse && resp.StatusCode == http.StatusAccepted {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return rpcResponse{}, nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponse()))
	if err != nil {
		return rpcResponse{}, fmt.Errorf("mcp: read %s response: %w", payload.Method, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rpcResponse{}, fmt.Errorf("mcp: %s returned HTTP %d: %s", payload.Method, resp.StatusCode, truncateForError(data))
	}
	if !expectResponse && len(strings.TrimSpace(string(data))) == 0 {
		return rpcResponse{}, nil
	}
	parsed, err := parseRPCResponse(resp.Header.Get("Content-Type"), data)
	if err != nil {
		return rpcResponse{}, fmt.Errorf("mcp: parse %s response: %w", payload.Method, err)
	}
	return parsed, nil
}

func (c *Client) applyHeaders(req *http.Request, body bool) {
	if body {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	req.Header.Set("MCP-Protocol-Version", c.protocolVersion())
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	if strings.TrimSpace(c.Authorization) != "" {
		req.Header.Set("Authorization", strings.TrimSpace(c.Authorization))
	}
}

func (c *Client) endpoint() string {
	return strings.TrimSpace(c.Endpoint)
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Client) protocolVersion() string {
	if strings.TrimSpace(c.ProtocolVersion) != "" {
		return strings.TrimSpace(c.ProtocolVersion)
	}
	return defaultProtocolVersion
}

func (c *Client) maxResponse() int64 {
	if c.MaxResponse > 0 {
		return c.MaxResponse
	}
	return defaultMaxResponse
}

func parseRPCResponse(contentType string, data []byte) (rpcResponse, error) {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if strings.EqualFold(mediaType, "text/event-stream") {
		data = firstSSEData(data)
	}
	var resp rpcResponse
	if err := json.Unmarshal(bytes.TrimSpace(data), &resp); err != nil {
		return rpcResponse{}, err
	}
	return resp, nil
}

func firstSSEData(data []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var b strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			b.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
		if strings.TrimSpace(line) == "" && b.Len() > 0 {
			break
		}
	}
	if b.Len() == 0 {
		return data
	}
	return []byte(b.String())
}

func truncateForError(data []byte) string {
	text := strings.TrimSpace(string(data))
	if len(text) > 512 {
		text = text[:512] + "...[truncated]"
	}
	return text
}
