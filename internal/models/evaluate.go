package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/llm"
)

// Candidate describes a model worth benchmarking before registry promotion.
type Candidate struct {
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Local       bool     `json:"local"`
	Cloud       bool     `json:"cloud"`
	MinMemoryGB int      `json:"min_memory_gb,omitempty"`
	Context     string   `json:"context"`
	Why         string   `json:"why"`
	Risks       []string `json:"risks,omitempty"`
	Source      string   `json:"source"`
}

// Plan is the model-refresh evaluation plan for the current harness build.
type Plan struct {
	GeneratedAt     time.Time                     `json:"generated_at"`
	CurrentDefaults map[hardware.Tier]ModelRecord `json:"current_defaults"`
	Candidates      []Candidate                   `json:"candidates"`
	BenchmarkCases  []BenchmarkCase               `json:"benchmark_cases"`
	PromotionRules  []string                      `json:"promotion_rules"`
}

// ModelRecord is the model registry shape exposed by evaluation output.
type ModelRecord struct {
	Name       string `json:"name"`
	Params     string `json:"params"`
	Quant      string `json:"quant"`
	ContextLen int    `json:"context_len"`
	RAMMinMiB  int    `json:"ram_min_mib"`
	Repo       string `json:"repo"`
	Revision   string `json:"revision"`
	File       string `json:"file"`
	SHA256     string `json:"sha256"`
}

// BenchmarkCase defines one deterministic model probe.
type BenchmarkCase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

// Config controls an evaluation run against an OpenAI-compatible endpoint.
type Config struct {
	Endpoint string
	Model    string
	APIKey   string
	Timeout  time.Duration
}

// Report summarizes a model evaluation run.
type Report struct {
	Model       string       `json:"model"`
	Endpoint    string       `json:"endpoint"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at"`
	Cases       []CaseResult `json:"cases"`
	Summary     Summary      `json:"summary"`
}

// CaseResult captures timing and mechanical pass/fail signals for one case.
type CaseResult struct {
	Name             string        `json:"name"`
	Passed           bool          `json:"passed"`
	Duration         time.Duration `json:"duration"`
	PromptTokens     int           `json:"prompt_tokens,omitempty"`
	CompletionTokens int           `json:"completion_tokens,omitempty"`
	TotalTokens      int           `json:"total_tokens,omitempty"`
	Error            string        `json:"error,omitempty"`
	Detail           string        `json:"detail,omitempty"`
}

// Summary aggregates evaluation signals.
type Summary struct {
	Passed       int           `json:"passed"`
	Failed       int           `json:"failed"`
	Total        int           `json:"total"`
	WallTime     time.Duration `json:"wall_time"`
	TokensPerSec float64       `json:"tokens_per_sec,omitempty"`
}

// DefaultPlan returns the current model refresh plan without contacting any endpoint.
func DefaultPlan(now time.Time) Plan {
	current := make(map[hardware.Tier]ModelRecord)
	for tier, spec := range hardware.DefaultModels(hardware.ProfileMedium) {
		current[tier] = modelRecord(spec)
	}

	return Plan{
		GeneratedAt:     now.UTC(),
		CurrentDefaults: current,
		Candidates: []Candidate{
			{
				Name:        "Qwen3.6 35B-A3B Coding",
				Role:        "coding/reasoning candidate",
				Local:       true,
				MinMemoryGB: 22,
				Context:     "256K",
				Why:         "Newest Qwen local candidate with explicit agentic coding and thinking-preservation claims.",
				Risks:       []string{"Needs GGUF/llama.cpp validation", "Tool-call behavior must be benchmarked before promotion"},
				Source:      "https://ollama.com/library/qwen3.6",
			},
			{
				Name:        "Qwen3.6 27B",
				Role:        "balanced coding candidate",
				Local:       true,
				MinMemoryGB: 17,
				Context:     "256K",
				Why:         "Smaller Qwen3.6 option likely to fit more Apple Silicon machines than the 35B coding variants.",
				Risks:       []string{"Quality may trail 35B coding variant", "Must compare against current Qwen3-Coder default"},
				Source:      "https://ollama.com/library/qwen3.6",
			},
			{
				Name:        "Laguna XS.2",
				Role:        "local long-horizon coding candidate",
				Local:       true,
				MinMemoryGB: 22,
				Context:     "128K",
				Why:         "33B total/3B active model positioned for local agentic coding and long-horizon work.",
				Risks:       []string{"Newer and less proven than Qwen defaults", "Needs harness-specific failure-mode data"},
				Source:      "https://ollama.com/library/laguna-xs.2",
			},
			{
				Name:    "GLM-5.1",
				Role:    "optional cloud/remote quality candidate",
				Local:   false,
				Cloud:   true,
				Context: "198K",
				Why:     "Strong self-reported agentic engineering results; useful as optional remote benchmark target.",
				Risks:   []string{"Cloud-only in Ollama listing", "Not a default for local/private operation"},
				Source:  "https://ollama.com/library/glm-5.1",
			},
			{
				Name:        "Mistral Medium 3.5",
				Role:        "large remote/high-memory candidate",
				Local:       true,
				Cloud:       true,
				MinMemoryGB: 80,
				Context:     "256K",
				Why:         "Recent 128B model merging reasoning and coding, useful as an upper-bound comparison.",
				Risks:       []string{"Too large for ordinary local default", "Must be treated as high-memory or remote only"},
				Source:      "https://ollama.com/library/mistral-medium-3.5",
			},
		},
		BenchmarkCases: []BenchmarkCase{
			{
				Name:        "tool-call-json",
				Description: "Model must call a simple inspect_file tool with parseable JSON arguments.",
				Kind:        "tool_call",
			},
			{
				Name:        "structured-triage-json",
				Description: "Model must return strict JSON classifying a failing harness run.",
				Kind:        "json",
			},
		},
		PromotionRules: []string{
			"Do not promote from newest/model-card claims alone.",
			"Candidate must beat or match current defaults on harness benchmark pass rate.",
			"Candidate must have acceptable tokens/sec and memory use on the target hardware profile.",
			"Candidate must support reliable tool-call JSON for mutating roles.",
			"Default registry entries must be pinned by immutable source revision and SHA256.",
			"Cloud-only candidates may be optional remote profiles but not local defaults.",
		},
	}
}

// Evaluate runs the mechanical benchmark pack against one OpenAI-compatible model endpoint.
func Evaluate(ctx context.Context, cfg Config) (Report, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return Report{}, fmt.Errorf("models evaluate: --endpoint is required for live evaluation")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return Report{}, fmt.Errorf("models evaluate: --model is required for live evaluation")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	client, err := llm.NewClient(llm.Config{
		BaseURL: cfg.Endpoint,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Timeout: timeout,
	})
	if err != nil {
		return Report{}, err
	}

	started := time.Now().UTC()
	report := Report{
		Model:     cfg.Model,
		Endpoint:  cfg.Endpoint,
		StartedAt: started,
	}

	report.Cases = append(report.Cases, runToolCallCase(ctx, client, cfg.Model))
	report.Cases = append(report.Cases, runStructuredJSONCase(ctx, client, cfg.Model))
	report.CompletedAt = time.Now().UTC()
	report.Summary = summarize(report.Cases, report.CompletedAt.Sub(started))
	return report, nil
}

func modelRecord(spec hardware.ModelSpec) ModelRecord {
	return ModelRecord{
		Name:       spec.Name,
		Params:     spec.Params,
		Quant:      spec.Quant,
		ContextLen: spec.ContextLen,
		RAMMinMiB:  spec.RAMMinMiB,
		Repo:       spec.Repo,
		Revision:   spec.Revision,
		File:       spec.File,
		SHA256:     spec.SHA256,
	}
}

func runToolCallCase(ctx context.Context, client *llm.Client, model string) CaseResult {
	start := time.Now()
	req := llm.ChatCompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are being evaluated for tool-call reliability. Use the provided tool exactly once.",
			},
			{
				Role:    "user",
				Content: "Inspect docs/design-docs/local-inference.md and explain nothing else.",
			},
		},
		Tools: []llm.ToolDefinition{
			{
				Type: "function",
				Function: llm.FunctionSpec{
					Name:        "inspect_file",
					Description: "Inspect one repo-relative file.",
					Parameters: map[string]any{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]any{
							"path": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	result := CaseResult{Name: "tool-call-json", Duration: time.Since(start)}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	addUsage(&result, resp.Usage)
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		result.Detail = "response did not include a tool call"
		return result
	}
	call := resp.Choices[0].Message.ToolCalls[0]
	if call.Function.Name != "inspect_file" {
		result.Detail = fmt.Sprintf("unexpected tool %q", call.Function.Name)
		return result
	}
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		result.Detail = "tool arguments were not parseable JSON: " + err.Error()
		return result
	}
	if args.Path == "" {
		result.Detail = "tool arguments omitted path"
		return result
	}
	result.Passed = true
	result.Detail = "tool call and JSON arguments parsed"
	return result
}

func runStructuredJSONCase(ctx context.Context, client *llm.Client, model string) CaseResult {
	start := time.Now()
	req := llm.ChatCompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "Return only strict JSON. Do not wrap it in markdown.",
			},
			{
				Role: "user",
				Content: `A harness engineer job timed out after creating two in-progress tickets and no commit.
Return {"category": "...", "risk": "...", "next_action": "..."} with short string values.`,
			},
		},
	}
	resp, err := client.ChatCompletion(ctx, req)
	result := CaseResult{Name: "structured-triage-json", Duration: time.Since(start)}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	addUsage(&result, resp.Usage)
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		result.Detail = "response was not strict JSON: " + err.Error()
		return result
	}
	for _, key := range []string{"category", "risk", "next_action"} {
		if strings.TrimSpace(parsed[key]) == "" {
			result.Detail = "missing JSON field " + key
			return result
		}
	}
	result.Passed = true
	result.Detail = "strict JSON parsed with required fields"
	return result
}

func addUsage(result *CaseResult, usage llm.Usage) {
	result.PromptTokens = usage.PromptTokens
	result.CompletionTokens = usage.CompletionTokens
	result.TotalTokens = usage.TotalTokens
}

func summarize(results []CaseResult, wall time.Duration) Summary {
	summary := Summary{Total: len(results), WallTime: wall}
	var totalTokens int
	for _, result := range results {
		if result.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		totalTokens += result.TotalTokens
	}
	if wall > 0 && totalTokens > 0 {
		summary.TokensPerSec = float64(totalTokens) / wall.Seconds()
	}
	return summary
}
