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

const defaultMaxRetries = 3

// Config configures the HTTP client and retry behaviour.
type Config struct {
	BaseURL    string        // e.g. http://127.0.0.1:8080/v1 (no trailing slash required)
	APIKey     string        // optional; sent as Bearer when non-empty
	Model      string        // default model name when request.Model is empty
	HTTPClient *http.Client  // optional; sensible default with Timeout when nil
	Timeout    time.Duration // per-attempt timeout when HTTPClient is nil
	MaxRetries int           // total attempts for retryable failures (default 3)
}

// Client speaks OpenAI-compatible chat completions over HTTP.
type Client struct {
	cfg    Config
	prefix string // normalized base + /v1 if needed
}

// NewClient returns a Client using cfg. BaseURL must be non-empty.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("llm: BaseURL is required")
	}
	prefix := strings.TrimRight(cfg.BaseURL, "/")
	if !strings.HasSuffix(prefix, "/v1") {
		prefix = prefix + "/v1"
	}
	if cfg.MaxRetries < 1 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.HTTPClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		cfg.HTTPClient = &http.Client{Timeout: timeout}
	}
	return &Client{cfg: cfg, prefix: prefix}, nil
}

// ChatCompletion performs a non-streaming completion.
func (c *Client) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.cfg.Model
	}
	req.Stream = false

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
			return fmt.Errorf("llm: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(b)))
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
	ms := 100
	for i := 0; i < attempt && ms < 5000; i++ {
		ms *= 2
		if ms > 5000 {
			ms = 5000
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
