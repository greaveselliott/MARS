/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package models

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/llm"
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
			require.Equal(t, ProviderOllama, candidate.Provider)
			require.NotEmpty(t, candidate.Model)
			require.Contains(t, candidate.Source, "ollama.com/library/qwen3.6")
		}
	}
	require.True(t, hasQwen36, "Qwen3.6 should be in the refresh shortlist")
	require.Contains(t, plan.BenchmarkCases, BenchmarkCase{
		Name:        "repo-ticket-completion-json",
		Description: "Model must read a repo ticket and return strict JSON naming the next completion gate.",
		Kind:        "repo_ticket",
		RepoBacked:  true,
	})
}

func TestEvaluate_requiresEndpointAndModel(t *testing.T) {
	_, err := Evaluate(context.Background(), Config{Model: "qwen3.6"})
	require.ErrorContains(t, err, "--endpoint is required")

	_, err = Evaluate(context.Background(), Config{Endpoint: "http://127.0.0.1:8080"})
	require.ErrorContains(t, err, "--model is required")
}

func TestEvaluateReportsCredentialReadFailure(t *testing.T) {
	repo := harnessRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", ".env.local"), []byte("not-an-env-assignment\n"), 0o600))
	t.Setenv("MARS_TEST_API_KEY", "")

	_, err := Evaluate(context.Background(), Config{
		RepoRoot:  repo,
		Endpoint:  "https://models.example.test/v1",
		Model:     "model-under-test",
		Provider:  ProviderOpenAI,
		APIKeyEnv: "MARS_TEST_API_KEY",
	})
	require.ErrorContains(t, err, "read credential env MARS_TEST_API_KEY")
	require.NotContains(t, err.Error(), "is not set")
}

func TestEvaluate_runsMechanicalCases(t *testing.T) {
	var calls int
	repoRoot := writeBenchmarkTicket(t, "MH-030")
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls++
		var req llm.ChatCompletionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		if calls == 1 {
			require.NotEmpty(t, req.Tools)
			return jsonResponse(llm.ChatCompletionResponse{
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
			}), nil
		}

		if calls == 2 {
			return jsonResponse(llm.ChatCompletionResponse{
				Model: req.Model,
				Choices: []llm.Choice{{
					Message: llm.Message{Content: `{"category":"timeout","risk":"stalled work","next_action":"repair queue recovery"}`},
				}},
				Usage: llm.Usage{PromptTokens: 20, CompletionTokens: 8, TotalTokens: 28},
			}), nil
		}

		return jsonResponse(llm.ChatCompletionResponse{
			Model: req.Model,
			Choices: []llm.Choice{{
				Message: llm.Message{Content: `{"ticket_id":"MH-030","work_type":"enabler","next_test":"go test ./internal/models","completion_gate":"report saved"}`},
			}},
			Usage: llm.Usage{PromptTokens: 30, CompletionTokens: 10, TotalTokens: 40},
		}), nil
	})}

	report, err := Evaluate(context.Background(), Config{
		Endpoint:        "http://models.test",
		Model:           "candidate",
		Provider:        ProviderOpenAICompatible,
		RepoRoot:        repoRoot,
		HardwareProfile: "medium",
		HTTPClient:      httpClient,
		Timeout:         time.Second,
	})
	require.NoError(t, err)

	require.Equal(t, "candidate", report.Model)
	require.Len(t, report.Cases, 3)
	require.Equal(t, 3, report.Summary.Passed)
	require.Equal(t, 0, report.Summary.Failed)
	require.Greater(t, report.Summary.TokensPerSec, 0.0)
	require.Equal(t, "medium", report.HardwareProfile)
	require.False(t, report.Promotion.SafeToPromote)
	require.Contains(t, report.Promotion.Reasons, "default promotion requires immutable source revision and SHA256")
}

func TestEvaluate_persistsReportJSON(t *testing.T) {
	var calls int
	repoRoot := writeBenchmarkTicket(t, "MH-030")
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return jsonResponse(llm.ChatCompletionResponse{
				Model: "qwen3.6:27b",
				Choices: []llm.Choice{{
					Message: llm.Message{ToolCalls: []llm.ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "inspect_file",
							Arguments: `{"path":"docs/design-docs/local-inference.md"}`,
						},
					}}},
				}},
			}), nil
		case 2:
			return jsonResponse(llm.ChatCompletionResponse{
				Model:   "qwen3.6:27b",
				Choices: []llm.Choice{{Message: llm.Message{Content: `{"category":"timeout","risk":"stalled","next_action":"retry"}`}}},
			}), nil
		default:
			return jsonResponse(llm.ChatCompletionResponse{
				Model:   "qwen3.6:27b",
				Choices: []llm.Choice{{Message: llm.Message{Content: `{"ticket_id":"MH-030","work_type":"enabler","next_test":"go test ./...","completion_gate":"promotion blocked"}`}}},
			}), nil
		}
	})}
	reportDir := filepath.Join(repoRoot, "docs", "generated", "model-evaluations")

	report, err := Evaluate(context.Background(), Config{
		Endpoint:   DefaultOllamaEndpoint,
		Model:      "qwen3.6:27b",
		Provider:   ProviderOllama,
		RepoRoot:   repoRoot,
		ReportsDir: reportDir,
		HTTPClient: httpClient,
		Timeout:    time.Second,
	})
	require.NoError(t, err)
	require.NotEmpty(t, report.ReportPath)

	data, err := os.ReadFile(report.ReportPath)
	require.NoError(t, err)
	var persisted Report
	require.NoError(t, json.Unmarshal(data, &persisted))
	require.Equal(t, report.Model, persisted.Model)
	require.Equal(t, ProviderOllama, persisted.Provider)
	require.Equal(t, "blocked", persisted.Promotion.Decision)
	require.Contains(t, persisted.Promotion.Reasons, "ad-hoc Ollama selections remain explicit overrides or candidates until pinned as default artifacts")
}

func TestEvaluate_failedBenchmarkCasesRemainVisible(t *testing.T) {
	var calls int
	repoRoot := writeBenchmarkTicket(t, "MH-030")
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return jsonResponse(llm.ChatCompletionResponse{
				Model: "candidate",
				Choices: []llm.Choice{{
					Message: llm.Message{Content: "I will not call the tool."},
				}},
			}), nil
		case 2:
			return jsonResponse(llm.ChatCompletionResponse{
				Model:   "candidate",
				Choices: []llm.Choice{{Message: llm.Message{Content: `not json`}}},
			}), nil
		default:
			return jsonResponse(llm.ChatCompletionResponse{
				Model:   "candidate",
				Choices: []llm.Choice{{Message: llm.Message{Content: `{"ticket_id":"WRONG","work_type":"enabler","next_test":"test","completion_gate":"gate"}`}}},
			}), nil
		}
	})}

	report, err := Evaluate(context.Background(), Config{
		Endpoint:   "http://models.test",
		Model:      "candidate",
		RepoRoot:   repoRoot,
		HTTPClient: httpClient,
		Timeout:    time.Second,
	})
	require.NoError(t, err)

	require.Equal(t, 0, report.Summary.Passed)
	require.Equal(t, 3, report.Summary.Failed)
	require.Contains(t, report.Cases[0].Detail, "response did not include a tool call")
	require.Contains(t, report.Cases[1].Detail, "response was not strict JSON")
	require.Contains(t, report.Cases[2].Detail, "did not match source ticket")
	require.Equal(t, "blocked", report.Promotion.Decision)
}

func writeBenchmarkTicket(t *testing.T, id string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "tickets", "backlog")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	body := `---
id: ` + id + `
work_type: enabler
---

# ` + id + `: Benchmark ticket

## Requirements
- Keep promotion evidence explicit.
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, id+"-benchmark.md"), []byte(body), 0o644))
	return root
}
