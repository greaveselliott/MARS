package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient_requiresBaseURL(t *testing.T) {
	t.Parallel()
	_, err := NewClient(Config{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "BaseURL")
}

func TestChatCompletion_happyPath(t *testing.T) {
	t.Parallel()
	want := ChatCompletionResponse{
		ID: "cmpl-1",
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: "hello",
			},
			FinishReason: "stop",
		}},
		Usage: Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var got ChatCompletionRequest
		require.NoError(t, json.Unmarshal(b, &got))
		require.False(t, got.Stream)
		require.Equal(t, "m", got.Model)
		require.Len(t, got.Messages, 1)
		require.Equal(t, "user", got.Messages[0].Role)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, Model: "m", HTTPClient: srv.Client(), Timeout: 5 * time.Second, MaxRetries: 1})
	require.NoError(t, err)

	out, err := c.ChatCompletion(context.Background(), ChatCompletionRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, "hello", out.Choices[0].Message.Content)
}

func TestChatCompletion_includesToolsAndAuth(t *testing.T) {
	t.Parallel()
	var sawAuth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth.Store(r.Header.Get("Authorization"))
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var got ChatCompletionRequest
		require.NoError(t, json.Unmarshal(b, &got))
		require.Len(t, got.Tools, 1)
		require.Equal(t, "fn", got.Tools[0].Function.Name)

		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: "assistant", Content: "ok"}}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{
		Model: "any",
		Messages: []Message{
			{Role: "system", Content: "sys"},
		},
		Tools: []ToolDefinition{{
			Type: "function",
			Function: FunctionSpec{
				Name: "fn",
				Parameters: map[string]any{
					"type": "object",
				},
			},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "Bearer k", sawAuth.Load())
}

func TestChatCompletion_toolCallsParsed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{
				Message: Message{
					Role: "assistant",
					ToolCalls: []ToolCall{{
						ID:   "call_1",
						Type: "function",
						Function: FunctionCall{
							Name:      "git_status",
							Arguments: `{"path":"."}`,
						},
					}},
				},
				FinishReason: "tool_calls",
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	out, err := c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.NoError(t, err)
	require.Len(t, out.Choices[0].Message.ToolCalls, 1)
	require.Equal(t, "git_status", out.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestChatCompletion_malformedToolArguments(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{
				Message: Message{
					Role: "assistant",
					ToolCalls: []ToolCall{{
						ID:   "call_1",
						Type: "function",
						Function: FunctionCall{
							Name:      "git_status",
							Arguments: `not-json`,
						},
					}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "malformed JSON arguments")
}

func TestChatCompletion_retries500(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{Choices: []Choice{{Message: Message{Content: "ok"}}}})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 3})
	require.NoError(t, err)

	out, err := c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.NoError(t, err)
	require.Equal(t, "ok", out.Choices[0].Message.Content)
	require.EqualValues(t, 3, calls.Load())
}

func TestChatCompletion_retries429WithRetryAfter(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "rate", http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{Choices: []Choice{{Message: Message{Content: "ok"}}}})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 3})
	require.NoError(t, err)

	out, err := c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.NoError(t, err)
	require.Equal(t, "ok", out.Choices[0].Message.Content)
}

func TestChatCompletion_emptyBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode response")
}

func TestChatCompletion_invalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode response")
}

func TestChatCompletionStream_assemblesChunks(t *testing.T) {
	t.Parallel()
	payload := strings.Join([]string{
		"data: " + mustJSON(ChatCompletionStreamChunk{Choices: []StreamChoice{{Delta: StreamChoiceDelta{Content: "hel"}}}}),
		"",
		"data: " + mustJSON(ChatCompletionStreamChunk{Choices: []StreamChoice{{Delta: StreamChoiceDelta{Content: "lo"}}}}),
		"",
		"data: [DONE]",
		"",
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var got ChatCompletionRequest
		require.NoError(t, json.Unmarshal(b, &got))
		require.True(t, got.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	var buf strings.Builder
	err = c.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "x"}},
	}, func(chunk ChatCompletionStreamChunk) error {
		for _, ch := range chunk.Choices {
			buf.WriteString(ch.Delta.Content)
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "hello", buf.String())
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestChatCompletion_contextTimeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{Choices: []Choice{{}}})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 20 * time.Millisecond},
		MaxRetries: 1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = c.ChatCompletion(ctx, ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "request failed")
}

func TestChatCompletion_normalizesBaseURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{Choices: []Choice{{Message: Message{Content: "x"}}}})
	}))
	t.Cleanup(srv.Close)

	base := strings.TrimSuffix(srv.URL, "/")
	// Intentionally omit /v1 suffix; client should add it.
	c, err := NewClient(Config{BaseURL: base, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.NoError(t, err)
}

func TestChatCompletion_unexpectedStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status")
}

func TestChatCompletion_emptyChoices(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChatCompletionResponse{Choices: nil})
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 1})
	require.NoError(t, err)

	_, err = c.ChatCompletion(context.Background(), ChatCompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty choices")
}
