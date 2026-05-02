package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/stretchr/testify/require"
)

func TestDefaultPlan_containsCandidatesAndPromotionRules(t *testing.T) {
	plan := DefaultPlan(time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))

	require.NotEmpty(t, plan.CurrentDefaults)
	require.NotEmpty(t, plan.Candidates)
	require.NotEmpty(t, plan.BenchmarkCases)
	require.Contains(t, plan.PromotionRules, "Do not promote from newest/model-card claims alone.")

	var hasQwen36 bool
	for _, candidate := range plan.Candidates {
		if candidate.Name == "Qwen3.6 35B-A3B Coding" {
			hasQwen36 = true
			require.True(t, candidate.Local)
			require.Contains(t, candidate.Source, "ollama.com/library/qwen3.6")
		}
	}
	require.True(t, hasQwen36, "Qwen3.6 should be in the refresh shortlist")
}

func TestEvaluate_requiresEndpointAndModel(t *testing.T) {
	_, err := Evaluate(context.Background(), Config{Model: "qwen3.6"})
	require.ErrorContains(t, err, "--endpoint is required")

	_, err = Evaluate(context.Background(), Config{Endpoint: "http://127.0.0.1:8080"})
	require.ErrorContains(t, err, "--model is required")
}

func TestEvaluate_runsMechanicalCases(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls++
		var req llm.ChatCompletionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		w.Header().Set("Content-Type", "application/json")

		if calls == 1 {
			require.NotEmpty(t, req.Tools)
			_ = json.NewEncoder(w).Encode(llm.ChatCompletionResponse{
				Model: req.Model,
				Choices: []llm.Choice{{
					Message: llm.Message{
						ToolCalls: []llm.ToolCall{{
							ID:   "call-1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "inspect_file",
								Arguments: `{"path":"docs/design-docs/local-inference.md"}`,
							},
						}},
					},
				}},
				Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(llm.ChatCompletionResponse{
			Model: req.Model,
			Choices: []llm.Choice{{
				Message: llm.Message{Content: `{"category":"timeout","risk":"stalled work","next_action":"repair queue recovery"}`},
			}},
			Usage: llm.Usage{PromptTokens: 20, CompletionTokens: 8, TotalTokens: 28},
		})
	}))
	defer srv.Close()

	report, err := Evaluate(context.Background(), Config{
		Endpoint: srv.URL,
		Model:    "candidate",
		Timeout:  time.Second,
	})
	require.NoError(t, err)

	require.Equal(t, "candidate", report.Model)
	require.Len(t, report.Cases, 2)
	require.Equal(t, 2, report.Summary.Passed)
	require.Equal(t, 0, report.Summary.Failed)
	require.Greater(t, report.Summary.TokensPerSec, 0.0)
}
