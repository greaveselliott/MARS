/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/local-inference.md
- docs/design-docs/context-efficiency.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
*/
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultMaxRetries = 5

// ContextSizeError reports a server-side context-window rejection (HTTP 400
// exceed_context_size_error from llama.cpp or compatible servers). It carries
// the server-measured prompt size and serving window when the body includes
// them, so the agent loop can clamp its budget to the actually served window
// and prune-and-retry instead of failing the job (AD-288).
type ContextSizeError struct {
	PromptTokens  int    // server-counted prompt tokens (0 when unknown)
	ContextWindow int    // server context window in tokens (0 when unknown)
	Body          string // raw error body for diagnostics
}

func (e *ContextSizeError) Error() string {
	return fmt.Sprintf("llm: context size exceeded (non-retryable): %s", e.Body)
}

// parseContextSizeError extracts n_prompt_tokens and n_ctx from a llama.cpp
// exceed_context_size_error body. Missing or malformed fields stay zero.
func parseContextSizeError(body string) *ContextSizeError {
	cerr := &ContextSizeError{Body: body}
	var payload struct {
		Error struct {
			NPromptTokens int `json:"n_prompt_tokens"`
			NCtx          int `json:"n_ctx"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err == nil {
		cerr.PromptTokens = payload.Error.NPromptTokens
		cerr.ContextWindow = payload.Error.NCtx
	}
	return cerr
}

// Config configures the HTTP client and retry behaviour.
type Config struct {
	BaseURL    string        // e.g. http://127.0.0.1:8080/v1 (no trailing slash required)
	APIKey     string        // optional; sent as Bearer when non-empty
	Model      string        // default model name when request.Model is empty
	Provider   string        // optional provider adapter: openai-compatible, anthropic, gemini, ...
	HTTPClient *http.Client  // optional; sensible default with Timeout when nil
	Timeout    time.Duration // per-attempt timeout when HTTPClient is nil
	MaxRetries int           // total attempts for retryable failures (default 3)
}

// Client speaks OpenAI-compatible chat completions over HTTP.
type Client struct {
	cfg      Config
	provider string
	prefix   string // normalized provider API base
}

// NewClient returns a Client using cfg. BaseURL must be non-empty.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("llm: BaseURL is required")
	}
	provider := normalizeProvider(cfg.Provider)
	prefix := normalizeClientPrefix(provider, cfg.BaseURL)
	if cfg.MaxRetries < 1 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.HTTPClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		cfg.HTTPClient = &http.Client{Timeout: timeout}
	}
	return &Client{cfg: cfg, provider: provider, prefix: prefix}, nil
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "custom":
		return "openai-compatible"
	case "openai", "openai-compatible", "ollama", "gemini", "mistral", "xai", "deepseek", "groq":
		return strings.ToLower(strings.TrimSpace(provider))
	case "anthropic", "claude":
		return "anthropic"
	case "cohere":
		return "cohere"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func normalizeClientPrefix(provider, baseURL string) string {
	prefix := strings.TrimRight(baseURL, "/")
	switch provider {
	case "anthropic", "cohere", "gemini":
		return prefix
	default:
		if !strings.HasSuffix(prefix, "/v1") {
			prefix = prefix + "/v1"
		}
		return prefix
	}
}

// ChatCompletion performs a non-streaming completion.
func (c *Client) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.cfg.Model
	}
	req.Stream = false
	if c.provider == "anthropic" {
		return c.anthropicChatCompletion(ctx, req)
	}
	if c.provider == "cohere" {
		return ChatCompletionResponse{}, fmt.Errorf("llm: provider cohere requires a native adapter fixture before runtime use")
	}

	var out ChatCompletionResponse
	if err := c.postJSON(ctx, "/chat/completions", req, &out); err != nil {
		return ChatCompletionResponse{}, err
	}
	if err := validateChatCompletionResponse(out); err != nil {
		return ChatCompletionResponse{}, err
	}
	return out, nil
}

// ChatCompletionStream streams completion chunks via onChunk until the stream ends or an error occurs.
func (c *Client) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest, onChunk func(ChatCompletionStreamChunk) error) error {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.cfg.Model
	}
	req.Stream = true
	if c.provider != "openai-compatible" && c.provider != "openai" && c.provider != "ollama" && c.provider != "gemini" && c.provider != "mistral" && c.provider != "xai" && c.provider != "deepseek" && c.provider != "groq" {
		return fmt.Errorf("llm: streaming is not implemented for provider %s", c.provider)
	}

	u := c.prefix + "/chat/completions"
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("llm: marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("llm: build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		if c.cfg.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		}

		resp, err := c.cfg.HTTPClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("llm: request failed (attempt %d/%d): %w", attempt+1, c.cfg.MaxRetries, err)
			if !retryableNetErr(err) || attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			if wait, ok := backoffDuration(attempt, 0); ok {
				if err := sleepCtx(ctx, wait); err != nil {
					return err
				}
			}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			retryAfter := parseRetryAfterSeconds(resp.Header.Get("Retry-After"))
			_ = drainAndClose(resp)
			lastErr = fmt.Errorf("llm: server returned %s (attempt %d/%d)", resp.Status, attempt+1, c.cfg.MaxRetries)
			if attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			wait, ok := backoffDuration(attempt, retryAfter)
			if ok {
				if err := sleepCtx(ctx, wait); err != nil {
					return err
				}
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
			return fmt.Errorf("llm: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}

		if err := readSSEStream(ctx, resp.Body, onChunk); err != nil {
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("llm: exhausted retries without response")
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, out any) error {
	u := c.prefix + path
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("llm: marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("llm: build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if c.cfg.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		}

		resp, err := c.cfg.HTTPClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("llm: request failed (attempt %d/%d): %w", attempt+1, c.cfg.MaxRetries, err)
			if !retryableNetErr(err) || attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			if wait, ok := backoffDuration(attempt, 0); ok {
				if err := sleepCtx(ctx, wait); err != nil {
					return err
				}
			}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			retryAfter := parseRetryAfterSeconds(resp.Header.Get("Retry-After"))
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("llm: server returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
			if attempt == c.cfg.MaxRetries-1 {
				return lastErr
			}
			wait, ok := backoffDuration(attempt, retryAfter)
			if ok {
				if err := sleepCtx(ctx, wait); err != nil {
					return err
				}
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, errRead := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
			if errRead != nil {
				return fmt.Errorf("llm: read error body: %w", errRead)
			}
			bodyStr := strings.TrimSpace(string(b))
			if resp.StatusCode == 400 && strings.Contains(bodyStr, "exceed") && strings.Contains(bodyStr, "context") {
				return parseContextSizeError(bodyStr)
			}
			return fmt.Errorf("llm: unexpected status %s: %s", resp.Status, bodyStr)
		}

		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(out); err != nil {
			_ = resp.Body.Close()
			return fmt.Errorf("llm: decode response: %w", err)
		}
		_ = resp.Body.Close()
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("llm: exhausted retries without response")
}

func readSSEStream(ctx context.Context, r io.Reader, onChunk func(ChatCompletionStreamChunk) error) error {
	sc := bufio.NewScanner(r)
	// Allow large lines for JSON payloads.
	const maxToken = 1 << 20
	buf := make([]byte, maxToken)
	sc.Buffer(buf, maxToken)

	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var chunk ChatCompletionStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("llm: decode stream chunk: %w", err)
		}
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("llm: read stream: %w", err)
	}
	return nil
}

func validateChatCompletionResponse(resp ChatCompletionResponse) error {
	if len(resp.Choices) == 0 {
		return fmt.Errorf("llm: empty choices in completion response")
	}
	msg := resp.Choices[0].Message
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name == "" {
			return fmt.Errorf("llm: tool call missing function name")
		}
		if !json.Valid([]byte(tc.Function.Arguments)) {
			return fmt.Errorf("llm: tool call %q has malformed JSON arguments", tc.Function.Name)
		}
	}
	return nil
}

func drainAndClose(resp *http.Response) error {
	if resp == nil {
		return nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.Body.Close()
}

func parseRetryAfterSeconds(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if sec, err := strconv.Atoi(h); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return 0
}

func backoffDuration(attempt int, retryAfter time.Duration) (time.Duration, bool) {
	if retryAfter > 0 {
		return retryAfter, true
	}
	ms := 500
	for i := 0; i < attempt && ms < 15000; i++ {
		ms *= 2
		if ms > 15000 {
			ms = 15000
		}
	}
	return time.Duration(ms) * time.Millisecond, true
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func retryableNetErr(err error) bool {
	// Conservative: treat timeouts and temporary errors as retryable.
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var te interface{ Temporary() bool }
	if errors.As(err, &te) && te.Temporary() {
		return true
	}
	// Unknown network errors: still retry a couple times.
	return true
}
