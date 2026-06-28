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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/llm"
)

const (
	ProviderOpenAICompatible = "openai-compatible"
	ProviderOllama           = "ollama"
	ProviderRegistry         = "registry"
)

// Candidate describes a model worth benchmarking before registry promotion.
type Candidate struct {
	Name        string   `json:"name"`
	Model       string   `json:"model,omitempty"`
	Provider    string   `json:"provider,omitempty"`
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
	RepoBacked  bool   `json:"repo_backed,omitempty"`
}

// Config controls an evaluation run against an OpenAI-compatible endpoint.
type Config struct {
	Endpoint        string
	Model           string
	Provider        string
	APIKey          string
	APIKeyEnv       string
	RepoRoot        string
	ReportsDir      string
	HardwareProfile string
	CandidateSource string
	Revision        string
	SHA256          string
	Cloud           bool
	HTTPClient      *http.Client
	Timeout         time.Duration
}

// Report summarizes a model evaluation run.
type Report struct {
	Model           string          `json:"model"`
	Provider        string          `json:"provider"`
	Endpoint        string          `json:"endpoint"`
	RepoRoot        string          `json:"repo_root,omitempty"`
	HardwareProfile string          `json:"hardware_profile,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     time.Time       `json:"completed_at"`
	ReportPath      string          `json:"report_path,omitempty"`
	Cases           []CaseResult    `json:"cases"`
	Summary         Summary         `json:"summary"`
	Promotion       PromotionReport `json:"promotion"`
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

// PromotionReport explains whether benchmark evidence is enough to change a default.
type PromotionReport struct {
	Decision         string                        `json:"decision"`
	SafeToPromote    bool                          `json:"safe_to_promote"`
	Candidate        PromotionCandidate            `json:"candidate"`
	ComparedTo       map[hardware.Tier]ModelRecord `json:"compared_to"`
	ComparableTiers  []hardware.Tier               `json:"comparable_tiers"`
	Reasons          []string                      `json:"reasons"`
	RequiredEvidence []string                      `json:"required_evidence,omitempty"`
}

// PromotionCandidate is the candidate metadata used by the promotion gate.
type PromotionCandidate struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	Source           string `json:"source,omitempty"`
	Revision         string `json:"revision,omitempty"`
	SHA256           string `json:"sha256,omitempty"`
	Local            bool   `json:"local"`
	Cloud            bool   `json:"cloud"`
	ExplicitOverride bool   `json:"explicit_override"`
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
				Model:       "qwen3.6:35b-a3b",
				Provider:    ProviderOllama,
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
				Model:       "qwen3.6:27b",
				Provider:    ProviderOllama,
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
				Model:       "laguna-xs.2:latest",
				Provider:    ProviderOllama,
				Role:        "local long-horizon coding candidate",
				Local:       true,
				MinMemoryGB: 22,
				Context:     "128K",
				Why:         "33B total/3B active model positioned for local agentic coding and long-horizon work.",
				Risks:       []string{"Newer and less proven than Qwen defaults", "Needs harness-specific failure-mode data"},
				Source:      "https://ollama.com/library/laguna-xs.2",
			},
			{
				Name:     "GLM-5.1",
				Model:    "glm-5.1:latest",
				Provider: ProviderOllama,
				Role:     "optional cloud/remote quality candidate",
				Local:    false,
				Cloud:    true,
				Context:  "198K",
				Why:      "Strong self-reported agentic engineering results; useful as optional remote benchmark target.",
				Risks:    []string{"Cloud-only in Ollama listing", "Not a default for local/private operation"},
				Source:   "https://ollama.com/library/glm-5.1",
			},
			{
				Name:        "Mistral Medium 3.5",
				Model:       "mistral-medium3.5:latest",
				Provider:    ProviderOllama,
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
			{
				Name:        "repo-ticket-completion-json",
				Description: "Model must read a repo ticket and return strict JSON naming the next completion gate.",
				Kind:        "repo_ticket",
				RepoBacked:  true,
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

// NormalizeProvider returns the supported provider name used in reports and overrides.
func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "":
		return ProviderOpenAICompatible
	case ProviderOpenAICompatible, "custom", "openai_compatible":
		return ProviderOpenAICompatible
	case "openai", "chatgpt":
		return ProviderOpenAI
	case "anthropic", "claude":
		return ProviderAnthropic
	case "google", "gemini":
		return ProviderGemini
	case "mistral":
		return ProviderMistral
	case "xai", "grok":
		return ProviderXAI
	case "deepseek":
		return ProviderDeepSeek
	case "groq":
		return ProviderGroq
	case "cohere":
		return ProviderCohere
	case ProviderOllama:
		return ProviderOllama
	case ProviderRegistry:
		return ProviderRegistry
	default:
		return strings.ToLower(strings.TrimSpace(provider))
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
	provider := NormalizeProvider(cfg.Provider)
	repoRoot := strings.TrimSpace(cfg.RepoRoot)
	if repoRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			repoRoot = cwd
		}
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" && strings.TrimSpace(cfg.APIKeyEnv) != "" {
		var ok bool
		apiKey, ok = LookupCredential(repoRoot, cfg.APIKeyEnv)
		if !ok {
			return Report{}, fmt.Errorf("models evaluate: credential env %s is not set — export %s=<secret> and retry", cfg.APIKeyEnv, cfg.APIKeyEnv)
		}
	}

	client, err := llm.NewClient(llm.Config{
		BaseURL:    cfg.Endpoint,
		APIKey:     apiKey,
		Provider:   provider,
		Model:      cfg.Model,
		HTTPClient: cfg.HTTPClient,
		Timeout:    timeout,
	})
	if err != nil {
		return Report{}, err
	}

	started := time.Now().UTC()
	report := Report{
		Model:           cfg.Model,
		Provider:        provider,
		Endpoint:        cfg.Endpoint,
		RepoRoot:        repoRoot,
		HardwareProfile: strings.TrimSpace(cfg.HardwareProfile),
		StartedAt:       started,
	}

	report.Cases = append(report.Cases, runToolCallCase(ctx, client, cfg.Model))
	report.Cases = append(report.Cases, runStructuredJSONCase(ctx, client, cfg.Model))
	report.Cases = append(report.Cases, runRepoTicketCase(ctx, client, cfg.Model, repoRoot))
	report.CompletedAt = time.Now().UTC()
	report.Summary = summarize(report.Cases, report.CompletedAt.Sub(started))
	report.Promotion = promotionForReport(report, cfg)
	if strings.TrimSpace(cfg.ReportsDir) != "" {
		report.ReportPath = filepath.Join(cfg.ReportsDir, reportFilename(report.Provider, report.Model, report.StartedAt))
		if _, err := writeReport(report, cfg.ReportsDir); err != nil {
			return Report{}, err
		}
	}
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

func runRepoTicketCase(ctx context.Context, client *llm.Client, model, repoRoot string) CaseResult {
	start := time.Now()
	result := CaseResult{Name: "repo-ticket-completion-json", Duration: time.Since(start)}
	ticketPath, ticketBody, err := findBenchmarkTicket(repoRoot)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	ticketID := ticketIDFromBody(ticketBody)
	req := llm.ChatCompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "Return only strict JSON. Do not wrap it in markdown.",
			},
			{
				Role: "user",
				Content: fmt.Sprintf(`This is a MARS repo ticket from %s:

%s

Return {"ticket_id":"...","work_type":"...","next_test":"...","completion_gate":"..."} with short string values.`, ticketPath, truncate(ticketBody, 4000)),
			},
		},
	}
	resp, err := client.ChatCompletion(ctx, req)
	result.Duration = time.Since(start)
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
	for _, key := range []string{"ticket_id", "work_type", "next_test", "completion_gate"} {
		if strings.TrimSpace(parsed[key]) == "" {
			result.Detail = "missing JSON field " + key
			return result
		}
	}
	if ticketID != "" && !strings.EqualFold(strings.TrimSpace(parsed["ticket_id"]), ticketID) {
		result.Detail = fmt.Sprintf("ticket_id %q did not match source ticket %q", parsed["ticket_id"], ticketID)
		return result
	}
	result.Passed = true
	result.Detail = "repo-backed ticket completion JSON parsed with required fields"
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

func findBenchmarkTicket(repoRoot string) (string, string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", "", fmt.Errorf("repo-backed benchmark: repo root is required")
	}
	for _, dir := range []string{
		filepath.Join(repoRoot, "docs", "tickets", "in-progress"),
		filepath.Join(repoRoot, "docs", "tickets", "backlog"),
		filepath.Join(repoRoot, "docs", "tickets", "done"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return "", "", fmt.Errorf("repo-backed benchmark: read %s: %w", path, err)
			}
			return path, string(data), nil
		}
	}
	return "", "", fmt.Errorf("repo-backed benchmark: no ticket markdown found under docs/tickets/{in-progress,backlog,done} — run against a harness-initialized repo or pass --repo <path>")
}

func ticketIDFromBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "id:")), `"'`)
		}
		if strings.HasPrefix(line, "# ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if before, _, ok := strings.Cut(heading, ":"); ok {
				return strings.TrimSpace(before)
			}
		}
	}
	return ""
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n...[truncated]"
}

func writeReport(report Report, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("models evaluate: create report directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, reportFilename(report.Provider, report.Model, report.StartedAt))
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("models evaluate: create report %s: %w", path, err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return "", fmt.Errorf("models evaluate: write report %s: %w", path, err)
	}
	return path, nil
}

func reportFilename(provider, model string, ts time.Time) string {
	name := sanitizeFilename(provider + "-" + model)
	if name == "" {
		name = "model"
	}
	return fmt.Sprintf("%s-%s.json", ts.UTC().Format("20060102T150405Z"), name)
}

func sanitizeFilename(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func promotionForReport(report Report, cfg Config) PromotionReport {
	comparedTo := make(map[hardware.Tier]ModelRecord)
	for tier, spec := range hardware.DefaultModels(hardware.ProfileMedium) {
		comparedTo[tier] = modelRecord(spec)
	}
	known := knownCandidate(cfg.Model)
	source := strings.TrimSpace(cfg.CandidateSource)
	if source == "" {
		source = known.Source
	}
	provider := NormalizeProvider(cfg.Provider)
	cloud := cfg.Cloud || (known.Cloud && !known.Local)
	local := !cloud
	if known.Local {
		local = true
	}
	candidate := PromotionCandidate{
		Provider:         provider,
		Model:            cfg.Model,
		Source:           source,
		Revision:         strings.TrimSpace(cfg.Revision),
		SHA256:           strings.TrimSpace(cfg.SHA256),
		Local:            local,
		Cloud:            cloud,
		ExplicitOverride: provider == ProviderOllama || provider == ProviderOpenAICompatible,
	}

	var reasons []string
	if report.Summary.Failed > 0 {
		reasons = append(reasons, fmt.Sprintf("%d/%d benchmark cases failed", report.Summary.Failed, report.Summary.Total))
	}
	if cloud && !local {
		reasons = append(reasons, "cloud-only candidates cannot be promoted into local zero-config defaults")
	}
	if candidate.Revision == "" || candidate.SHA256 == "" {
		reasons = append(reasons, "default promotion requires immutable source revision and SHA256")
	}
	if provider == ProviderOllama && (candidate.Revision == "" || candidate.SHA256 == "") {
		reasons = append(reasons, "ad-hoc Ollama selections remain explicit overrides or candidates until pinned as default artifacts")
	}

	promotion := PromotionReport{
		Decision:        "safe",
		SafeToPromote:   true,
		Candidate:       candidate,
		ComparedTo:      comparedTo,
		ComparableTiers: comparableTiers(known, cfg.Model),
		Reasons:         []string{"candidate passed benchmark gates and includes pinned artifact metadata"},
	}
	if len(reasons) > 0 {
		promotion.Decision = "blocked"
		promotion.SafeToPromote = false
		promotion.Reasons = reasons
		promotion.RequiredEvidence = []string{
			"passing harness benchmark report against current defaults",
			"hardware-fit timing and token-throughput evidence",
			"immutable artifact revision",
			"SHA256 checksum for the exact promoted artifact",
			"design/product rationale for any default change",
		}
	}
	return promotion
}

func knownCandidate(model string) Candidate {
	normalized := strings.ToLower(strings.TrimSpace(model))
	for _, candidate := range DefaultPlan(time.Now()).Candidates {
		if strings.EqualFold(candidate.Model, model) || strings.Contains(normalized, strings.ToLower(candidate.Name)) {
			return candidate
		}
		if candidate.Model != "" && strings.Contains(normalized, strings.Split(strings.ToLower(candidate.Model), ":")[0]) {
			return candidate
		}
	}
	return Candidate{Provider: NormalizeProvider(""), Local: true}
}

func comparableTiers(candidate Candidate, model string) []hardware.Tier {
	role := strings.ToLower(candidate.Role + " " + model)
	var tiers []hardware.Tier
	if strings.Contains(role, "fast") {
		tiers = append(tiers, hardware.TierFast)
	}
	if strings.Contains(role, "reasoning") {
		tiers = append(tiers, hardware.TierReasoning)
	}
	if strings.Contains(role, "coding") || strings.Contains(role, "code") {
		tiers = append(tiers, hardware.TierCoding)
	}
	if len(tiers) == 0 {
		tiers = append(tiers, hardware.TierCoding, hardware.TierReasoning)
	}
	return tiers
}
