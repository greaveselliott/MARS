/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/board-driven-integrations.md
- docs/features/F-013-board-driven-integrations.md
- docs/product-specs/product-surface.md
- docs/runbooks/atlassian-mcp-jira-intake.md
*/
package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// StdioConfig describes an ephemeral MCP stdio subprocess.
type StdioConfig struct {
	Command string
	Args    []string
	Env     []string
}

// StdioClient speaks newline-delimited MCP JSON-RPC over a subprocess stdio.
type StdioClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout <-chan rpcResponse
	errs   <-chan error

	nextID atomic.Int64
	writes sync.Mutex
	closed sync.Once
}

func StartStdio(ctx context.Context, cfg StdioConfig) (*StdioClient, error) {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		return nil, fmt.Errorf("mcp stdio: command is required")
	}
	cmd := exec.CommandContext(ctx, command, cfg.Args...)
	if len(cfg.Env) > 0 {
		cmd.Env = cfg.Env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: open stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: open stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: open stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp stdio: start %s: %w", command, err)
	}
	responses := make(chan rpcResponse, 8)
	errs := make(chan error, 1)
	go readStdioResponses(stdout, responses, errs)
	go io.Copy(io.Discard, stderr)
	return &StdioClient{cmd: cmd, stdin: stdin, stdout: responses, errs: errs}, nil
}

func (c *StdioClient) Initialize(ctx context.Context) error {
	_, err := c.doRPC(ctx, "initialize", map[string]any{
		"protocolVersion": c.protocolVersion(),
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mars",
			"version": "integration-client",
		},
	})
	if err != nil {
		return err
	}
	return c.doNotification(ctx, "notifications/initialized", map[string]any{})
}

func (c *StdioClient) ListTools(ctx context.Context) ([]Tool, error) {
	raw, err := c.doRPC(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("mcp stdio: parse tools/list result: %w", err)
	}
	return result.Tools, nil
}

func (c *StdioClient) CallTool(ctx context.Context, name string, args any) (json.RawMessage, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("mcp stdio: tool name is required")
	}
	return c.doRPC(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
}

func (c *StdioClient) Close(ctx context.Context) error {
	var err error
	c.closed.Do(func() {
		if c.stdin != nil {
			_ = c.stdin.Close()
		}
		done := make(chan error, 1)
		go func() {
			if c.cmd != nil && c.cmd.Process != nil {
				_ = c.cmd.Process.Kill()
			}
			if c.cmd != nil {
				done <- c.cmd.Wait()
				return
			}
			done <- nil
		}()
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case waitErr := <-done:
			if waitErr != nil && !strings.Contains(waitErr.Error(), "signal: killed") && !strings.Contains(waitErr.Error(), "process already finished") && !strings.Contains(waitErr.Error(), os.ErrProcessDone.Error()) {
				err = fmt.Errorf("mcp stdio: close process: %w", waitErr)
			}
		}
	})
	return err
}

func (c *StdioClient) doRPC(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	if err := c.write(ctx, rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-c.errs:
			return nil, fmt.Errorf("mcp stdio: read response: %w", err)
		case resp, ok := <-c.stdout:
			if !ok {
				return nil, fmt.Errorf("mcp stdio: subprocess stdout closed")
			}
			if !responseIDMatches(resp.ID, id) {
				continue
			}
			if resp.Error != nil {
				return nil, fmt.Errorf("mcp stdio: %s failed: %s", method, resp.Error.Message)
			}
			return resp.Result, nil
		}
	}
}

func (c *StdioClient) doNotification(ctx context.Context, method string, params any) error {
	return c.write(ctx, rpcRequest{JSONRPC: "2.0", Method: method, Params: params})
}

func (c *StdioClient) write(ctx context.Context, payload rpcRequest) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mcp stdio: encode %s request: %w", payload.Method, err)
	}
	c.writes.Lock()
	defer c.writes.Unlock()
	done := make(chan error, 1)
	go func() {
		_, writeErr := c.stdin.Write(append(data, '\n'))
		done <- writeErr
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("mcp stdio: write %s request: %w", payload.Method, err)
		}
		return nil
	}
}

func (c *StdioClient) protocolVersion() string {
	return defaultProtocolVersion
}

func readStdioResponses(stdout io.Reader, responses chan<- rpcResponse, errs chan<- error) {
	defer close(responses)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), defaultMaxResponse)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			errs <- err
			return
		}
		responses <- resp
	}
	if err := scanner.Err(); err != nil {
		errs <- err
	}
}

func responseIDMatches(raw json.RawMessage, want int64) bool {
	if len(raw) == 0 {
		return false
	}
	var id int64
	if err := json.Unmarshal(raw, &id); err == nil {
		return id == want
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text == strconv.FormatInt(want, 10)
	}
	return false
}
