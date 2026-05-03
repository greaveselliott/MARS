package mcpstdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/buildinfo"
	"github.com/greaveselliott/mars-harness/internal/tools"
)

const protocolVersion = "2024-11-05"

// Server exposes the Mars Harness built-in registry as an MCP-style JSON-RPC
// stdio tool server for any MCP-compatible host or local harness agent.
type Server struct {
	Registry *tools.Registry
	Executor *tools.Executor
	Root     tools.Root
	Allow    []string
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Serve reads newline-delimited JSON-RPC messages from r and writes responses to
// w. Notifications are accepted and do not produce responses.
func (s Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	if s.Registry == nil {
		return fmt.Errorf("mcp: registry is nil")
	}
	if s.Executor == nil {
		return fmt.Errorf("mcp: executor is nil")
	}
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 8*1024*1024)
	enc := json.NewEncoder(w)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if err := enc.Encode(rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "parse error: " + err.Error()},
			}); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		resp := s.handle(ctx, req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s Server) handle(ctx context.Context, req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	result, err := s.dispatch(ctx, req)
	if err != nil {
		resp.Error = &rpcError{Code: -32000, Message: err.Error()}
		return resp
	}
	resp.Result = result
	return resp
}

func (s Server) dispatch(ctx context.Context, req rpcRequest) (any, error) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mars-harness",
				"version": buildinfo.DefaultVersion,
			},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return s.listTools()
	case "tools/call":
		return s.callTool(ctx, req.Params)
	default:
		return nil, fmt.Errorf("method %q is not supported", req.Method)
	}
}

func (s Server) listTools() (any, error) {
	allow := s.Allow
	if len(allow) == 0 {
		allow = s.Registry.Names()
	}
	defs, err := s.Registry.Definitions(allow)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		out = append(out, map[string]any{
			"name":        def.Function.Name,
			"description": def.Function.Description,
			"inputSchema": def.Function.Parameters,
		})
	}
	return map[string]any{"tools": out}, nil
}

func (s Server) callTool(ctx context.Context, raw json.RawMessage) (any, error) {
	var params callParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("tools/call: parse params: %w", err)
	}
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, fmt.Errorf("tools/call: name is required")
	}
	args := strings.TrimSpace(string(params.Arguments))
	if args == "" || args == "null" {
		args = "{}"
	}
	allow := s.Allow
	if len(allow) == 0 {
		allow = s.Registry.Names()
	}
	res, err := s.Executor.Execute(ctx, s.Root, allow, name, args)
	text := res.FormatForModel()
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return map[string]any{
			"isError": true,
			"content": []map[string]string{{
				"type": "text",
				"text": text,
			}},
		}, nil
	}
	return map[string]any{
		"content": []map[string]string{{
			"type": "text",
			"text": text,
		}},
	}, nil
}
