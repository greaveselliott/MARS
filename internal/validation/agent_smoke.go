/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-smoke-validation.md
- docs/design-docs/foundation-operating-model.md
- docs/design-docs/validation-matrix-gating.md
- docs/validation/README.md
- docs/validation/agent-smoke/README.md
- docs/product-specs/product-surface.md
- docs/features/F-012-self-improvement-loop.md
*/
package validation

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/greaveselliott/mars/internal/codeintel"
	"github.com/greaveselliott/mars/internal/config"
	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/inference"
	"github.com/greaveselliott/mars/internal/orgstate"
	"github.com/greaveselliott/mars/internal/queue"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/serve"
	"github.com/greaveselliott/mars/internal/tools"
	"github.com/greaveselliott/mars/internal/trace"
	"github.com/greaveselliott/mars/internal/trust"
	"github.com/greaveselliott/mars/internal/ui"
	"gopkg.in/yaml.v3"
)

const (
	AgentSmokeSuiteFast    = "fast"
	AgentSmokeSuiteDefault = "default"
	AgentSmokeSuiteFull    = "full"
	AgentSmokeSuiteHeldOut = "held-out"

	agentSmokeCaseContractPath = "docs/validation/agent-smoke/current-case.md"
	agentSmokeDefaultMaxTurns  = 32

	FailureFixtureInvalid       = "fixture-invalid"
	FailureFoundationGeneration = "foundation-tool-generation"
	FailureEnvironmentModel     = "environment/model"
	FailureRoleBehavior         = "role-behavior"
	FailureToolPolicy           = "tool-policy"
	FailureDispatchContext      = "dispatch-context"
	FailureProjectTypeGap       = "project-type-gap"
	FailureFlakyValidation      = "flaky-validation"
	FailureCleanupFailed        = "cleanup-failed"
	FailureUnknown              = "unknown"
)

// AgentSmokeOptions configures compartmentalised role smoke validation.
type AgentSmokeOptions struct {
	HarnessRoot   string        `json:"harness_root"`
	Root          string        `json:"root"`
	Role          string        `json:"role"`
	CaseID        string        `json:"case"`
	ProjectType   string        `json:"project_type"`
	Suite         string        `json:"suite"`
	Parallel      int           `json:"parallel"`
	Cycle         string        `json:"cycle"`
	MaxTurns      int           `json:"max_turns"`
	Timeout       time.Duration `json:"timeout"`
	ModelEndpoint string        `json:"model_endpoint,omitempty"`
	SingleServer  bool          `json:"single_server"`
	SingleTier    string        `json:"single_server_tier,omitempty"`
	FixtureOnly   bool          `json:"fixture_only"`
	JSON          bool          `json:"json"`
	ReportPath    string        `json:"report"`
	KeepRuns      bool          `json:"keep_runs"`
	CleanupOnly   bool          `json:"cleanup_only"`
	DiscardFailed bool          `json:"discard_failed"`
}

// AgentSmokeReport is the machine-readable result emitted by agent-smoke.
type AgentSmokeReport struct {
	Root           string             `json:"root"`
	Suite          string             `json:"suite"`
	Evidence       string             `json:"evidence"`
	ModelSource    string             `json:"model_source"`
	SingleServer   bool               `json:"single_server"`
	SingleTier     string             `json:"single_server_tier,omitempty"`
	ServerParallel int                `json:"server_parallel,omitempty"`
	StartedAt      time.Time          `json:"started_at"`
	FinishedAt     time.Time          `json:"finished_at"`
	Selected       int                `json:"selected"`
	Passed         int                `json:"passed"`
	Failed         int                `json:"failed"`
	Cleaned        int                `json:"cleaned"`
	Results        []AgentSmokeResult `json:"results"`
	ReportPath     string             `json:"report_path,omitempty"`
	CleanupOnly    bool               `json:"cleanup_only"`
}

// OK reports whether all selected cases passed.
func (r AgentSmokeReport) OK() bool {
	return r.Failed == 0
}

// Summary returns a compact human-readable status.
func (r AgentSmokeReport) Summary() string {
	if r.CleanupOnly {
		return fmt.Sprintf("agent-smoke cleanup: removed %d retained run(s) from %s", r.Cleaned, r.Root)
	}
	return fmt.Sprintf("agent-smoke %s: %d passed, %d failed, %d selected", r.Suite, r.Passed, r.Failed, r.Selected)
}

// AgentSmokeResult captures one case execution result.
type AgentSmokeResult struct {
	CaseID              string        `json:"case"`
	Role                string        `json:"role"`
	ProjectType         string        `json:"project_type"`
	Suite               []string      `json:"suite"`
	Status              string        `json:"status"`
	FailureClass        string        `json:"failure_class,omitempty"`
	Error               string        `json:"error,omitempty"`
	RunPath             string        `json:"run_path,omitempty"`
	TargetPath          string        `json:"target_path,omitempty"`
	DBPath              string        `json:"db_path,omitempty"`
	LogPath             string        `json:"log_path,omitempty"`
	TracePath           string        `json:"trace_path,omitempty"`
	ExecutionMode       string        `json:"execution_mode,omitempty"`
	JobID               string        `json:"job_id,omitempty"`
	TerminalDisposition string        `json:"terminal_disposition,omitempty"`
	TerminalNextNeed    string        `json:"terminal_next_need,omitempty"`
	TerminalSuggested   string        `json:"terminal_suggested_role,omitempty"`
	TerminalTraceID     string        `json:"terminal_trace_id,omitempty"`
	WouldDispatch       string        `json:"would_dispatch,omitempty"`
	ExpectedDisposition string        `json:"expected_disposition,omitempty"`
	GenerationTools     []string      `json:"generation_tools"`
	RequiredArtifacts   []string      `json:"required_artifacts"`
	ForbiddenMutations  []string      `json:"forbidden_mutations"`
	Discarded           bool          `json:"discarded"`
	Duration            time.Duration `json:"duration"`
}

// AgentSmokeMatrix is the checked-in matrix shape.
type AgentSmokeMatrix struct {
	Cases []AgentSmokeCase `json:"cases" yaml:"cases"`
}

// AgentSmokeCase defines one ephemeral role smoke case.
type AgentSmokeCase struct {
	ID                  string            `json:"id" yaml:"id"`
	Role                string            `json:"role" yaml:"role"`
	ProjectType         string            `json:"project_type" yaml:"project_type"`
	Stage               string            `json:"stage" yaml:"stage"`
	Suites              []string          `json:"suites" yaml:"suites"`
	SourceContracts     []string          `json:"source_contracts" yaml:"source_contracts"`
	ExpectedDisposition string            `json:"expected_disposition" yaml:"expected_disposition"`
	WouldDispatch       string            `json:"would_dispatch" yaml:"would_dispatch"`
	RequiredArtifacts   []string          `json:"required_artifacts" yaml:"required_artifacts"`
	ForbiddenMutations  []string          `json:"forbidden_mutations" yaml:"forbidden_mutations"`
	Trigger             map[string]string `json:"trigger" yaml:"trigger"`
}

// RunAgentSmoke runs or cleans compartmentalised role smoke cases.
func RunAgentSmoke(ctx context.Context, opts AgentSmokeOptions) (AgentSmokeReport, error) {
	started := time.Now().UTC()
	opts = normalizeAgentSmokeOptions(opts)
	root, err := resolveAgentSmokeRoot(opts)
	if err != nil {
		return AgentSmokeReport{}, err
	}
	report := AgentSmokeReport{
		Root:        root,
		Suite:       opts.Suite,
		Evidence:    agentSmokeEvidence(opts),
		ModelSource: agentSmokeModelSource(opts),
		StartedAt:   started,
		CleanupOnly: opts.CleanupOnly,
	}
	if opts.CleanupOnly {
		cleaned, cleanErr := cleanupAgentSmokeRuns(root)
		report.Cleaned = cleaned
		report.FinishedAt = time.Now().UTC()
		return report, cleanErr
	}
	matrix, err := LoadAgentSmokeMatrix(opts.HarnessRoot)
	if err != nil {
		return AgentSmokeReport{}, err
	}
	selected, err := SelectAgentSmokeCases(matrix, opts)
	if err != nil {
		return AgentSmokeReport{}, err
	}
	if len(selected) == 0 {
		return AgentSmokeReport{}, fmt.Errorf("validation agent-smoke: no cases selected for suite=%q role=%q case=%q project-type=%q", opts.Suite, opts.Role, opts.CaseID, opts.ProjectType)
	}
	runtime, err := newAgentSmokeRuntime(opts)
	if err != nil {
		return AgentSmokeReport{}, err
	}
	defer runtime.Close()
	report.SingleServer = runtime.singleServer
	report.SingleTier = runtime.singleTier
	report.ServerParallel = runtime.serverParallel
	report.Selected = len(selected)
	results := runAgentSmokeCases(ctx, runtime, root, selected, opts)
	report.Results = results
	for _, result := range results {
		if result.Status == "passed" {
			report.Passed++
		} else {
			report.Failed++
		}
		if result.Discarded {
			report.Cleaned++
		}
	}
	report.FinishedAt = time.Now().UTC()
	if strings.TrimSpace(opts.ReportPath) != "" {
		if err := writeAgentSmokeMarkdownReport(opts.ReportPath, report); err != nil {
			return report, err
		}
		report.ReportPath = opts.ReportPath
	}
	return report, nil
}

func normalizeAgentSmokeOptions(opts AgentSmokeOptions) AgentSmokeOptions {
	opts.Suite = strings.ToLower(strings.TrimSpace(opts.Suite))
	if opts.Suite == "" {
		opts.Suite = AgentSmokeSuiteFast
	}
	if opts.Parallel <= 0 {
		opts.Parallel = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	if opts.MaxTurns <= 0 && !opts.FixtureOnly {
		opts.MaxTurns = agentSmokeDefaultMaxTurns
	}
	opts.Role = strings.TrimSpace(opts.Role)
	opts.CaseID = strings.TrimSpace(opts.CaseID)
	opts.ProjectType = strings.TrimSpace(opts.ProjectType)
	opts.ModelEndpoint = strings.TrimSpace(opts.ModelEndpoint)
	opts.SingleTier = strings.ToLower(strings.TrimSpace(opts.SingleTier))
	if opts.SingleServer && opts.SingleTier == "" {
		opts.SingleTier = string(hardware.TierCoding)
	}
	return opts
}

func agentSmokeEvidence(opts AgentSmokeOptions) string {
	if opts.CleanupOnly {
		return "cleanup-only"
	}
	if opts.FixtureOnly {
		return "fixture-only"
	}
	if strings.TrimSpace(opts.ModelEndpoint) != "" {
		return "endpoint-override"
	}
	return "local-model"
}

func agentSmokeModelSource(opts AgentSmokeOptions) string {
	switch agentSmokeEvidence(opts) {
	case "cleanup-only", "fixture-only":
		return "none"
	case "endpoint-override":
		return "operator-supplied OpenAI-compatible endpoint; AD-296 requires this to be a real model endpoint for validation claims"
	default:
		if opts.SingleServer {
			return fmt.Sprintf("local MARS inference router; single local server tier %s", opts.SingleTier)
		}
		return "local MARS inference router"
	}
}

func resolveAgentSmokeRoot(opts AgentSmokeOptions) (string, error) {
	if strings.TrimSpace(opts.Root) != "" {
		return filepath.Abs(opts.Root)
	}
	harnessRoot := strings.TrimSpace(opts.HarnessRoot)
	if harnessRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		harnessRoot = wd
	}
	return filepath.Abs(filepath.Join(harnessRoot, "..", "demo", "validation-runs", "agent-smoke"))
}

// LoadAgentSmokeMatrix reads docs/validation/agent-smoke/matrix.yaml.
func LoadAgentSmokeMatrix(harnessRoot string) (AgentSmokeMatrix, error) {
	if strings.TrimSpace(harnessRoot) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return AgentSmokeMatrix{}, err
		}
		harnessRoot = wd
	}
	path := filepath.Join(harnessRoot, "docs", "validation", "agent-smoke", "matrix.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return AgentSmokeMatrix{}, fmt.Errorf("validation agent-smoke: read matrix %s: %w", path, err)
	}
	var matrix AgentSmokeMatrix
	if err := yaml.Unmarshal(data, &matrix); err != nil {
		return AgentSmokeMatrix{}, fmt.Errorf("validation agent-smoke: parse matrix %s: %w", path, err)
	}
	if err := ValidateAgentSmokeMatrix(matrix, harnessRoot); err != nil {
		return AgentSmokeMatrix{}, err
	}
	return matrix, nil
}

// ValidateAgentSmokeMatrix checks the static matrix for obvious drift.
func ValidateAgentSmokeMatrix(matrix AgentSmokeMatrix, harnessRoot string) error {
	if len(matrix.Cases) == 0 {
		return fmt.Errorf("validation agent-smoke: matrix has no cases")
	}
	seen := map[string]bool{}
	for _, c := range matrix.Cases {
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("validation agent-smoke: case id is required")
		}
		if seen[c.ID] {
			return fmt.Errorf("validation agent-smoke: duplicate case id %q", c.ID)
		}
		seen[c.ID] = true
		if strings.TrimSpace(c.Role) == "" {
			return fmt.Errorf("validation agent-smoke: case %s missing role", c.ID)
		}
		if strings.TrimSpace(c.ProjectType) == "" {
			return fmt.Errorf("validation agent-smoke: case %s missing project_type", c.ID)
		}
		if strings.TrimSpace(c.ExpectedDisposition) == "" {
			return fmt.Errorf("validation agent-smoke: case %s missing expected_disposition", c.ID)
		}
		if len(c.Suites) == 0 {
			return fmt.Errorf("validation agent-smoke: case %s missing suites", c.ID)
		}
		for _, doc := range c.SourceContracts {
			if strings.TrimSpace(doc) == "" {
				continue
			}
			if _, err := os.Stat(filepath.Join(harnessRoot, filepath.FromSlash(doc))); err != nil {
				return fmt.Errorf("validation agent-smoke: case %s source_contract %s is unavailable: %w", c.ID, doc, err)
			}
		}
	}
	return nil
}

// SelectAgentSmokeCases filters matrix cases according to CLI options.
func SelectAgentSmokeCases(matrix AgentSmokeMatrix, opts AgentSmokeOptions) ([]AgentSmokeCase, error) {
	opts = normalizeAgentSmokeOptions(opts)
	if opts.Suite != AgentSmokeSuiteFast && opts.Suite != AgentSmokeSuiteDefault && opts.Suite != AgentSmokeSuiteFull && opts.Suite != AgentSmokeSuiteHeldOut {
		return nil, fmt.Errorf("validation agent-smoke: unsupported suite %q", opts.Suite)
	}
	candidates := make([]AgentSmokeCase, 0, len(matrix.Cases))
	for _, c := range matrix.Cases {
		if opts.Role != "" && c.Role != opts.Role {
			continue
		}
		if opts.CaseID != "" && c.ID != opts.CaseID {
			continue
		}
		if opts.ProjectType != "" && c.ProjectType != opts.ProjectType {
			continue
		}
		if opts.Suite != AgentSmokeSuiteFull && !caseInSuite(c, opts.Suite) {
			continue
		}
		candidates = append(candidates, c)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Role == candidates[j].Role {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].Role < candidates[j].Role
	})
	if opts.Suite != AgentSmokeSuiteFast || opts.CaseID != "" {
		return candidates, nil
	}
	grouped := map[string][]AgentSmokeCase{}
	var roles []string
	for _, c := range candidates {
		if len(grouped[c.Role]) == 0 {
			roles = append(roles, c.Role)
		}
		grouped[c.Role] = append(grouped[c.Role], c)
	}
	sort.Strings(roles)
	selected := make([]AgentSmokeCase, 0, len(roles))
	for _, role := range roles {
		cases := grouped[role]
		idx := cycleIndex(opts.Cycle, role, len(cases))
		selected = append(selected, cases[idx])
	}
	return selected, nil
}

func caseInSuite(c AgentSmokeCase, suite string) bool {
	for _, s := range c.Suites {
		if strings.EqualFold(strings.TrimSpace(s), suite) {
			return true
		}
	}
	return false
}

func cycleIndex(cycle, role string, n int) int {
	if n <= 1 {
		return 0
	}
	if i, err := strconv.Atoi(strings.TrimSpace(cycle)); err == nil {
		if i < 0 {
			i = -i
		}
		return i % n
	}
	sum := sha1.Sum([]byte(cycle + ":" + role))
	v, _ := strconv.ParseInt(hex.EncodeToString(sum[:2]), 16, 64)
	return int(v) % n
}

type agentSmokeRuntime struct {
	router         *inference.Router
	singleServer   bool
	singleTier     string
	serverParallel int
}

func newAgentSmokeRuntime(opts AgentSmokeOptions) (*agentSmokeRuntime, error) {
	if opts.FixtureOnly {
		return &agentSmokeRuntime{}, nil
	}
	if opts.ModelEndpoint != "" {
		return &agentSmokeRuntime{router: inference.NewRouter(inference.RouterConfig{FallbackURL: opts.ModelEndpoint})}, nil
	}
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return nil, fmt.Errorf("validation agent-smoke: load local inference config: %w", err)
	}
	modelSet := hardware.DefaultModelsForHardware(hardware.Detect(), cfg.PerformanceProfile)
	tuning := inference.ServerTuning{
		Threads:        cfg.LlamaThreads,
		ThreadsBatch:   cfg.LlamaThreadsBatch,
		Parallel:       cfg.LlamaParallel,
		BatchSize:      cfg.LlamaBatchSize,
		UBatchSize:     cfg.LlamaUBatchSize,
		FlashAttention: cfg.LlamaFlashAttention,
		MLock:          cfg.LlamaMLock,
	}
	singleTier := hardware.Tier("")
	if opts.SingleServer {
		parsed, err := parseAgentSmokeSingleTier(opts.SingleTier)
		if err != nil {
			return nil, err
		}
		singleTier = parsed
		if opts.Parallel > tuning.Parallel {
			tuning.Parallel = opts.Parallel
		}
	}
	return &agentSmokeRuntime{router: inference.NewRouter(inference.RouterConfig{
		BinaryPath:       filepath.Join(cfg.BinDir, "llama-server"),
		Models:           modelSet,
		RoleMapping:      inference.DefaultRoleTierMapping(),
		ModelsDir:        cfg.ModelsDir,
		SingleServerTier: singleTier,
		Tuning:           tuning,
	}),
		singleServer:   opts.SingleServer,
		singleTier:     string(singleTier),
		serverParallel: tuning.Parallel,
	}, nil
}

func parseAgentSmokeSingleTier(value string) (hardware.Tier, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(hardware.TierCoding):
		return hardware.TierCoding, nil
	case string(hardware.TierReasoning):
		return hardware.TierReasoning, nil
	case string(hardware.TierFast):
		return hardware.TierFast, nil
	default:
		return "", fmt.Errorf("validation agent-smoke: unsupported --single-server-tier %q; use coding, reasoning, or fast", value)
	}
}

func (r *agentSmokeRuntime) Close() {
	if r != nil && r.router != nil {
		r.router.StopAll()
	}
}

func runAgentSmokeCases(ctx context.Context, runtime *agentSmokeRuntime, root string, cases []AgentSmokeCase, opts AgentSmokeOptions) []AgentSmokeResult {
	parallel := opts.Parallel
	if parallel < 1 {
		parallel = 1
	}
	if parallel > len(cases) {
		parallel = len(cases)
	}
	results := make([]AgentSmokeResult, len(cases))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < parallel; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				caseCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
				results[idx] = runAgentSmokeCase(caseCtx, runtime, root, cases[idx], opts)
				cancel()
			}
		}()
	}
	for idx := range cases {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()
	return results
}

func runAgentSmokeCase(ctx context.Context, runtime *agentSmokeRuntime, root string, c AgentSmokeCase, opts AgentSmokeOptions) (result AgentSmokeResult) {
	started := time.Now()
	result = AgentSmokeResult{
		CaseID:              c.ID,
		Role:                c.Role,
		ProjectType:         c.ProjectType,
		Suite:               c.Suites,
		Status:              "failed",
		ExpectedDisposition: c.ExpectedDisposition,
		WouldDispatch:       c.WouldDispatch,
		RequiredArtifacts:   c.RequiredArtifacts,
		ForbiddenMutations:  c.ForbiddenMutations,
	}
	runID := fmt.Sprintf("run-%s-%s-%s", time.Now().UTC().Format("20060102-150405"), c.Role, c.ID)
	runDir := filepath.Join(root, runID)
	targetDir := filepath.Join(runDir, "target")
	result.RunPath = runDir
	result.TargetPath = targetDir
	result.DBPath = filepath.Join(runDir, "db", "mars.db")
	result.LogPath = filepath.Join(runDir, "logs", c.Role+".log")
	result.TracePath = filepath.Join(runDir, "trace")
	defer func() {
		result.Duration = time.Since(started)
	}()
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		result.FailureClass = FailureFoundationGeneration
		result.Error = err.Error()
		return result
	}
	if err := os.MkdirAll(filepath.Dir(result.DBPath), 0o755); err != nil {
		result.FailureClass = FailureFoundationGeneration
		result.Error = err.Error()
		return result
	}
	if err := os.MkdirAll(filepath.Dir(result.LogPath), 0o755); err != nil {
		result.FailureClass = FailureFoundationGeneration
		result.Error = err.Error()
		return result
	}
	if err := os.MkdirAll(result.TracePath, 0o755); err != nil {
		result.FailureClass = FailureFoundationGeneration
		result.Error = err.Error()
		return result
	}
	provenance, err := generateAgentSmokeTarget(ctx, targetDir, c, opts)
	result.GenerationTools = provenance
	if err != nil {
		result.FailureClass = classifyAgentSmokeError(err)
		result.Error = err.Error()
		_ = writeAgentSmokeRunArtifacts(result)
		if opts.DiscardFailed {
			result.Discarded = discardRunDir(runDir) == nil
		}
		return result
	}
	if err := assertAgentSmokeCaseBefore(targetDir, c); err != nil {
		result.FailureClass = FailureFixtureInvalid
		result.Error = err.Error()
		_ = writeAgentSmokeRunArtifacts(result)
		if opts.DiscardFailed {
			result.Discarded = discardRunDir(runDir) == nil
		}
		return result
	}
	if err := executeAgentSmokeRole(ctx, runtime, &result, c, opts); err != nil {
		result.FailureClass = classifyAgentSmokeError(err)
		result.Error = err.Error()
		_ = writeAgentSmokeRunArtifacts(result)
		if opts.DiscardFailed {
			result.Discarded = discardRunDir(runDir) == nil
		}
		return result
	}
	if !opts.FixtureOnly {
		if err := assertAgentSmokeCaseAfter(targetDir, c); err != nil {
			result.FailureClass = FailureRoleBehavior
			result.Error = err.Error()
			_ = writeAgentSmokeRunArtifacts(result)
			if opts.DiscardFailed {
				result.Discarded = discardRunDir(runDir) == nil
			}
			return result
		}
	}
	result.Status = "passed"
	result.FailureClass = ""
	result.Error = ""
	if err := writeAgentSmokeRunArtifacts(result); err != nil {
		result.Status = "failed"
		result.FailureClass = FailureFoundationGeneration
		result.Error = err.Error()
		return result
	}
	if !opts.KeepRuns {
		if err := discardRunDir(runDir); err != nil {
			result.Status = "failed"
			result.FailureClass = FailureCleanupFailed
			result.Error = err.Error()
			return result
		}
		result.Discarded = true
	}
	return result
}

func executeAgentSmokeRole(ctx context.Context, runtime *agentSmokeRuntime, result *AgentSmokeResult, c AgentSmokeCase, opts AgentSmokeOptions) error {
	if opts.FixtureOnly {
		result.ExecutionMode = "fixture-only"
		return nil
	}
	if c.Role == "foundation-maintainer" {
		result.ExecutionMode = "source-only"
		result.JobID = "agent-smoke-source-only-" + slugify(c.ID)
		result.TerminalDisposition = c.ExpectedDisposition
		if err := os.WriteFile(result.LogPath, []byte("foundation-maintainer is source-only; target manifest execution is intentionally skipped for this case\n"), 0o644); err != nil {
			return err
		}
		return nil
	}
	if runtime == nil || runtime.router == nil {
		return fmt.Errorf("validation agent-smoke: live execution requires an inference router")
	}
	result.ExecutionMode = "live"
	logFile, err := os.Create(result.LogPath)
	if err != nil {
		return fmt.Errorf("validation agent-smoke: create role log: %w", err)
	}
	defer logFile.Close()

	traceStore, err := trace.OpenStore(result.DBPath)
	if err != nil {
		return fmt.Errorf("validation agent-smoke: open trace store: %w", err)
	}
	defer traceStore.Close()
	trustStore, err := trust.OpenStore(result.DBPath)
	if err != nil {
		return fmt.Errorf("validation agent-smoke: open trust store: %w", err)
	}
	defer trustStore.Close()
	orgStore, err := orgstate.OpenStore(result.DBPath)
	if err != nil {
		return fmt.Errorf("validation agent-smoke: open orgstate store: %w", err)
	}
	defer orgStore.Close()

	exec := serve.NewExecutor(func(context.Context, string) (string, error) {
		return result.TargetPath, nil
	}, runtime.router, result.DBPath, traceStore, trustStore)
	exec.SetOrgState(orgStore)
	exec.SetCodeIntel(codeintel.NewRuntime(false, "agent-smoke"))
	exec.SetJobViewFactory(ui.NewDebugJobViewFactory(logFile, false, true))

	jobID := "agent-smoke-" + slugify(c.Role+"-"+c.ID)
	result.JobID = jobID
	trigger, err := json.Marshal(agentSmokeTrigger(c))
	if err != nil {
		return fmt.Errorf("validation agent-smoke: marshal trigger: %w", err)
	}
	job := &queue.Job{
		ID:          jobID,
		RepoID:      result.TargetPath,
		Role:        c.Role,
		Trigger:     string(trigger),
		PayloadMode: "agent_smoke",
	}
	if err := exec.Execute(ctx, job); err != nil {
		disposition, dErr := readAgentSmokeDisposition(orgStore, job.ID)
		if dErr == nil && disposition != nil {
			recordAgentSmokeDisposition(result, disposition)
		}
		return err
	}
	disposition, err := readAgentSmokeDisposition(orgStore, job.ID)
	if err != nil {
		return fmt.Errorf("validation agent-smoke: read terminal disposition: %w", err)
	}
	if disposition == nil {
		return fmt.Errorf("validation agent-smoke: %s/%s recorded no terminal disposition", c.Role, c.ID)
	}
	recordAgentSmokeDisposition(result, disposition)
	if !agentSmokeDispositionMatches(c.ExpectedDisposition, disposition.Status) {
		return fmt.Errorf("validation agent-smoke: %s/%s expected disposition %q, got %q", c.Role, c.ID, c.ExpectedDisposition, disposition.Status)
	}
	if err := assertAgentSmokeDisposition(c, disposition); err != nil {
		return err
	}
	return nil
}

func readAgentSmokeDisposition(store *orgstate.Store, jobID string) (*orgstate.Disposition, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return store.GetDisposition(ctx, jobID)
}

func agentSmokeTrigger(c AgentSmokeCase) map[string]any {
	trigger := map[string]any{
		"type":                      "agent_smoke",
		"case":                      c.ID,
		"role_under_test":           c.Role,
		"project_type":              c.ProjectType,
		"stage":                     c.Stage,
		"case_contract_path":        agentSmokeCaseContractPath,
		"case_contract_instruction": "Read the target-local case contract before deciding no_work, blocked, or terminal routing. Do not read the foundation matrix from inside the target repo.",
		"case_contract_summary":     agentSmokeTriggerSummary(c),
		"terminal_disposition_contract": map[string]string{
			"status":         expectedDispositionForCase(c),
			"next_need":      nextNeedForCase(c),
			"suggested_role": suggestedRoleForCase(c),
		},
		"terminal_disposition_instruction": terminalDispositionInstruction(c),
		"expected_disposition":             c.ExpectedDisposition,
		"would_dispatch":                   c.WouldDispatch,
		"suppress_follow_on_dispatch":      true,
		"fixture_source":                   "docs/validation/agent-smoke/matrix.yaml",
	}
	for k, v := range c.Trigger {
		trigger[k] = v
	}
	return trigger
}

func agentSmokeTriggerSummary(c AgentSmokeCase) string {
	expected := strings.TrimSpace(c.ExpectedDisposition)
	if expected == "" {
		expected = "completed"
	}
	switch c.Role {
	case "cto-weekly":
		return `Create exactly one implementation ticket for F-001-S001. After ticket_create, do not loop on git_diff; immediately call git_commit with all dirty changes, then record ` + expected + ` with next_need implementation and suggested_role engineer.`
	case "pipeline-fixer":
		if expected == "blocked" {
			return `Classify the seeded foundation/runtime failure, do not rewrite .mars/checks/latest.json to passed, commit only role-owned evidence if changed, then record blocked with next_need operator_review` + suggestedRoleSummary(c) + `.`
		}
		return `Run the focused validation for this project type` + pipelineFixerValidationHint(c) + `. When it passes, call file_write on .mars/checks/latest.json with status passed and evidence naming that validation, commit the check/test updates plus any .harness/learnings.yaml update, then record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Do not keep searching for CI config in this ephemeral target.`
	case "dogfood":
		if expected == "changes_requested" {
			return `This case intentionally contains a seeded defect. Read .mars/checks/latest.json and docs/reports/dogfood/seeded-defect.md; passing unit tests do not clear this held-out gap. Write docs/reports/dogfood/` + c.ID + `.md referencing seeded-defect.md, create/commit exactly one target-owned finding ticket referencing seeded-defect.md if no open finding already exists, then record changes_requested with next_need implementation_rework` + suggestedRoleSummary(c) + `. Do not approve.`
		}
		if c.ProjectType == "browser-game-phaser" {
			return `For Phaser browser games, run npm run build and a source/runtime browser-product smoke that prints "browser smoke: Phaser canvas #game new Phaser.Game" before approving. If dependency_sync creates package-lock.json, commit package-lock.json as validation provenance instead of deleting it. Commit docs/reports/dogfood/` + c.ID + `.md, then record approved with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
		}
		if c.ProjectType == "react-web" {
			return `For React web, hydrate dependencies with dependency_sync frozen false when needed, run npm run build, run a source/runtime browser-product smoke that prints "browser smoke: React document.querySelector #game score UI state". If dependency_sync creates package-lock.json, commit package-lock.json as validation provenance instead of deleting it. Commit docs/reports/dogfood/` + c.ID + `.md, then record approved with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
		}
		if c.ProjectType == "static-web" {
			port := strconv.Itoa(agentSmokeStaticWebPort(c))
			return `For static web, use python3 -m http.server ` + port + ` with background:true, curl http://localhost:` + port + `/ once, and treat HTTP 200 as sufficient. Stop the tracked PID if still running; if kill reports already stopped after HTTP 200, continue. Commit docs/reports/dogfood/` + c.ID + `.md, then record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Do not issue no-op wait commands.`
		}
		if c.ProjectType == "go-api" || c.ProjectType == "go-cli" || c.ProjectType == "go-library" {
			return `For Go targets, run go test ./... as the bounded user smoke. When it passes, write and commit docs/reports/dogfood/` + c.ID + `.md, then record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Do not issue empty argv or no-op shell commands.`
		}
		return `Run one project-appropriate user smoke, commit docs/reports/dogfood/` + c.ID + `.md, then record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Do not issue no-op wait commands.`
	case "release-manager":
		if expected == "blocked" {
			return `Inspect the seeded blocked release/check state, do not run release notes, do not tag, do not push, and record blocked with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
		}
		if c.ProjectType == "docs-site" {
			return `For docs-site ready cases, this is notes-only: write docs/reports/release/` + c.ID + `.md before the release-note commit, advance VERSION beyond 0.0.0, expand CHANGELOG, commit VERSION, CHANGELOG, and the report together as release: notes <VERSION>, create the local v<VERSION> tag at that HEAD, skip build/assets/runtime gates, and record completed with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
		}
		return `For ready cases, missing GitHub remote or GitHub Release is not a blocker. Write docs/reports/release/` + c.ID + `.md before the release-note commit, advance VERSION beyond 0.0.0, expand CHANGELOG, commit VERSION, CHANGELOG, and the report together as release: notes <VERSION>, create the local v<VERSION> tag at that HEAD, skip git_push/github_release_status gating, then record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
	case "engineer":
		if isTicketClaimStage(c.Stage) {
			src := seededTicketPath(c, "in-progress")
			dst := seededTicketPath(c, "done")
			return `Use the active seed ticket already at ` + src + `. Do not look for it in backlog. Run the focused validation for this project type` + engineerValidationHint(c) + `. Before closing the ticket, update that in-progress ticket with evidence_links and verified_by, then git_status and git_commit while the ticket is still in-progress. Only after that move it with shell_exec argv ["git","mv","` + src + `","` + dst + `"], git_status, git_commit, and record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Never call shell_exec with an empty argv.`
		}
		return `Claim one ticket, implement the smallest project-appropriate change, run focused validation` + engineerValidationHint(c) + `, commit implementation, move the ticket to done, commit lifecycle evidence, then record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Never call shell_exec with an empty argv.`
	case "security":
		reportPath := "docs/reports/security/" + c.ID + ".md"
		if c.ProjectType == "static-web" {
			port := strconv.Itoa(agentSmokeStaticWebPort(c))
			return `Use this exact Security smoke sequence: inspect the done ticket and source surface, run node --check app.js, then python3 -m http.server ` + port + ` with background:true and curl http://localhost:` + port + `/. Do not use python or curl localhost before starting the server. Then call file_write for the exact security report path ` + reportPath + `, git_status, git_commit, docsync_audit, and job_disposition_record with ` + expected + `, next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Do not use a generic dated security-audit path for agent-smoke and do not stop in prose.`
		}
		return `Use this exact Security smoke sequence: inspect the done ticket and source surface, run the focused validation for this project type, then call file_write for the exact security report path ` + reportPath + `, git_status, git_commit, docsync_audit, and job_disposition_record with ` + expected + `, next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `. Do not use a generic dated security-audit path for agent-smoke and do not stop in prose.`
	case "qa":
		if expected == "changes_requested" {
			return `This case intentionally lacks acceptable review evidence. Inspect docs/reports/qa/seeded-gap-` + c.ID + `.md, write docs/reports/qa/` + c.ID + `.md naming that gap, then git_status and git_commit the report before recording changes_requested with next_need implementation_rework` + suggestedRoleSummary(c) + `. Do not approve.`
		}
		if c.ProjectType == "static-web" {
			port := strconv.Itoa(agentSmokeStaticWebPort(c))
			return `For static web, use node --check app.js, then python3 -m http.server ` + port + ` with background:true and curl http://localhost:` + port + `/. Do not use python or curl localhost before starting the server. When those bounded checks pass, write the role report at docs/reports/` + c.Role + `/` + c.ID + `.md, then git_status and git_commit the report before recording ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
		}
		return `Inspect seeded evidence, produce the role report at docs/reports/` + c.Role + `/` + c.ID + `.md, then git_status and git_commit the report before recording ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + ` when bounded smoke evidence is sufficient.`
	case "janitor":
		if strings.Contains(c.Stage, "stale") {
			src := seededTicketPath(c, "in-progress")
			dst := seededTicketPath(c, "done")
			return `Inspect the stale in-progress ticket with the Agent Smoke Stale Marker. Do not use file_write under docs/tickets because ticket-file writes are policy-blocked. First mutating action must be shell_exec argv ["mv","` + src + `","` + dst + `"], then git_status, git_commit, and job_disposition_record with next_need implementation and suggested_role engineer.`
		}
		if expected == "blocked" {
			return `Inspect the blocked janitor signal, do not invent cleanup work, then record blocked with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
		}
		return `Inspect clean-tree/stale state, make only maintenance-owned cleanup changes, and record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
	default:
		return `Read the target-local case contract, perform only role-owned work for this lifecycle checkpoint, commit any mutations, and record ` + expected + ` with next_need ` + nextNeedForCase(c) + suggestedRoleSummary(c) + `.`
	}
}

func expectedDispositionForCase(c AgentSmokeCase) string {
	expected := strings.TrimSpace(c.ExpectedDisposition)
	if expected == "" {
		return "completed"
	}
	return expected
}

func terminalDispositionInstruction(c AgentSmokeCase) string {
	expected := expectedDispositionForCase(c)
	nextNeed := nextNeedForCase(c)
	if role := suggestedRoleForCase(c); role != "" {
		return `Final job_disposition_record MUST include status "` + expected + `", next_need "` + nextNeed + `", and suggested_role "` + role + `". Do not rely on handoff defaults.`
	}
	return `Final job_disposition_record MUST include status "` + expected + `" and next_need "` + nextNeed + `" with no suggested_role.`
}

func recordAgentSmokeDisposition(result *AgentSmokeResult, disposition *orgstate.Disposition) {
	if result == nil || disposition == nil {
		return
	}
	result.TerminalDisposition = disposition.Status
	result.TerminalNextNeed = disposition.NextNeed
	result.TerminalSuggested = disposition.SuggestedRole
	if result.TerminalSuggested == "" {
		result.TerminalSuggested = disposition.Handoff.TargetRole
	}
	result.TerminalTraceID = disposition.TraceID
}

func agentSmokeDispositionMatches(expected, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" {
		return actual != ""
	}
	return expected == actual
}

func assertAgentSmokeDisposition(c AgentSmokeCase, disposition *orgstate.Disposition) error {
	if disposition == nil {
		return fmt.Errorf("validation agent-smoke: %s/%s recorded no terminal disposition", c.Role, c.ID)
	}
	expectedNextNeed := strings.TrimSpace(c.Trigger["next_need"])
	if expectedNextNeed != "" && strings.TrimSpace(disposition.NextNeed) != expectedNextNeed {
		return fmt.Errorf("validation agent-smoke: %s/%s expected next_need %q, got %q", c.Role, c.ID, expectedNextNeed, disposition.NextNeed)
	}
	expectedRole := suggestedRoleForCase(c)
	actualRole := strings.TrimSpace(disposition.SuggestedRole)
	if actualRole == "" {
		actualRole = strings.TrimSpace(disposition.Handoff.TargetRole)
	}
	if expectedRole != "" {
		if actualRole != expectedRole {
			return fmt.Errorf("validation agent-smoke: %s/%s expected suggested role %q, got %q", c.Role, c.ID, expectedRole, actualRole)
		}
		return nil
	}
	if actualRole != "" {
		return fmt.Errorf("validation agent-smoke: %s/%s expected no suggested role, got %q", c.Role, c.ID, actualRole)
	}
	return nil
}

func generateAgentSmokeTarget(ctx context.Context, targetDir string, c AgentSmokeCase, opts AgentSmokeOptions) ([]string, error) {
	var provenance []string
	if err := runCommand(ctx, targetDir, "git", "init", "-b", "main"); err != nil {
		return provenance, err
	}
	_ = runCommand(ctx, targetDir, "git", "config", "user.email", "agent-smoke@example.invalid")
	_ = runCommand(ctx, targetDir, "git", "config", "user.name", "Mars Agent Smoke")

	reg, err := tools.DefaultRegistry()
	if err != nil {
		return provenance, err
	}
	executor := tools.NewExecutor(reg)
	executor.Session = &tools.Session{
		Role:       "foundation-validation-seeder",
		JobID:      "agent-smoke-seed-" + c.ID,
		RepoID:     targetDir,
		TrustLevel: "contributor",
		ToolCounts: map[string]int{},
	}
	root, err := tools.NewRoot(targetDir)
	if err != nil {
		return provenance, err
	}
	allow := []string{"file_write", "ticket_create", "record_decision", "git_status", "git_commit", "shell_exec", "workspace_hygiene", "docsync_audit"}
	call := func(name string, args any) error {
		raw, err := json.Marshal(args)
		if err != nil {
			return err
		}
		provenance = append(provenance, name)
		_, err = executor.Execute(ctx, root, allow, name, string(raw))
		return err
	}
	if err := call("file_write", map[string]string{"path": "spec.md", "content": specForCase(c)}); err != nil {
		return provenance, err
	}
	if err := call("file_write", map[string]string{"path": "README.md", "content": readmeForCase(c)}); err != nil {
		return provenance, err
	}
	if err := call("file_write", map[string]string{"path": agentSmokeCaseContractPath, "content": caseContractForCase(c)}); err != nil {
		return provenance, err
	}
	provenance = append(provenance, "scanner.EnsureHarness")
	if _, err := scanner.EnsureHarness(targetDir, false); err != nil {
		return provenance, err
	}
	if err := call("file_write", map[string]string{"path": ".harness/learnings.yaml", "content": "version: 1\nlearnings: []\n"}); err != nil {
		return provenance, err
	}
	if err := call("file_write", map[string]string{"path": "docs/features/F-001-product-walking-skeleton.md", "content": featureContractForCase(c)}); err != nil {
		return provenance, err
	}
	if needsPlannerContext(c.Stage) {
		if err := call("record_decision", map[string]string{"summary": "Agent smoke prior strategy for " + c.ID, "rationale": "Seeded by Compartmentalised Agent Smoke to model prior planner handoff context."}); err != nil {
			return provenance, err
		}
		if err := call("file_write", map[string]string{"path": "docs/goals/ceo-strategy.md", "content": strategyForCase(c)}); err != nil {
			return provenance, err
		}
	}
	if needsCOOContext(c.Stage) {
		if err := call("file_write", map[string]string{"path": "docs/exec-plans/active/current-operating-plan.md", "content": planForCase(c)}); err != nil {
			return provenance, err
		}
	}
	if err := seedProjectFiles(call, c); err != nil {
		return provenance, err
	}
	if opts.MaxTurns > 0 && c.Role != "foundation-maintainer" {
		if err := applyAgentSmokeMaxTurns(call, targetDir, c.Role, opts.MaxTurns); err != nil {
			return provenance, err
		}
	}
	ticketPath := ""
	if needsTicket(c.Stage) {
		if err := call("ticket_create", ticketArgsForCase(c)); err != nil {
			return provenance, err
		}
		var findErr error
		ticketPath, findErr = findFirstTicket(targetDir, "backlog")
		if findErr != nil {
			return provenance, findErr
		}
		if isTicketClaimStage(c.Stage) {
			if err := moveTicket(ctx, call, ticketPath, "docs/tickets/in-progress/"+filepath.Base(ticketPath)); err != nil {
				return provenance, err
			}
			ticketPath = "docs/tickets/in-progress/" + filepath.Base(ticketPath)
		}
		if strings.Contains(c.Stage, "stale") {
			dest := "docs/tickets/in-progress/" + filepath.Base(ticketPath)
			if err := moveTicket(ctx, call, ticketPath, dest); err != nil {
				return provenance, err
			}
			ticketPath = dest
			if err := markTicketStale(call, targetDir, ticketPath, c); err != nil {
				return provenance, err
			}
		}
	}
	if isCompletedTicketStage(c.Stage) {
		if ticketPath == "" {
			if err := call("ticket_create", ticketArgsForCase(c)); err != nil {
				return provenance, err
			}
			var findErr error
			ticketPath, findErr = findFirstTicket(targetDir, "backlog")
			if findErr != nil {
				return provenance, findErr
			}
		}
		if err := call("git_commit", map[string]any{"message": "chore: seed agent smoke source " + c.ID}); err != nil {
			return provenance, err
		}
		dest := "docs/tickets/done/" + filepath.Base(ticketPath)
		if err := moveTicket(ctx, call, ticketPath, dest); err != nil {
			return provenance, err
		}
		ticketPath = dest
	}
	if strings.Contains(c.Stage, "after-engineer") {
		dest := "docs/tickets/in-review/" + filepath.Base(ticketPath)
		if err := moveTicket(ctx, call, ticketPath, dest); err != nil {
			return provenance, err
		}
	}
	if err := seedReports(call, c); err != nil {
		return provenance, err
	}
	if err := seedSpecialState(call, c); err != nil {
		return provenance, err
	}
	_ = call("workspace_hygiene", map[string]any{})
	if err := call("git_status", map[string]any{}); err != nil {
		return provenance, err
	}
	if err := call("git_commit", map[string]any{"message": "chore: seed agent smoke " + c.ID}); err != nil {
		return provenance, err
	}
	return provenance, nil
}

func runCommand(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runCommandOutput(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func moveTicket(ctx context.Context, call func(string, any) error, src, dst string) error {
	_ = ctx
	return call("shell_exec", map[string]any{"argv": []string{"mv", src, dst}})
}

func findFirstTicket(targetDir, status string) (string, error) {
	dir := filepath.Join(targetDir, "docs", "tickets", status)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("no ticket found in docs/tickets/%s", status)
	}
	return filepath.ToSlash(filepath.Join("docs", "tickets", status, names[0])), nil
}

func markTicketStale(call func(string, any) error, targetDir, ticketPath string, c AgentSmokeCase) error {
	data, err := os.ReadFile(filepath.Join(targetDir, filepath.FromSlash(ticketPath)))
	if err != nil {
		return fmt.Errorf("read stale ticket %s: %w", ticketPath, err)
	}
	content := string(data)
	content = strings.Replace(content, "evidence_links: []", `evidence_links: [".mars/checks/latest.json"]`, 1)
	content = strings.Replace(content, `verified_by: "TBD"`, `verified_by: "foundation-validation-seeder"`, 1)
	content = strings.TrimRight(content, "\n") + "\n\n## Agent Smoke Stale Marker\n\n- stale_since: 2025-01-01\n- case: " + c.ID + "\n- expected_janitor_action: resolve stale lifecycle and move this ticket to `docs/tickets/done/` with existing evidence.\n"
	return call("file_write", map[string]string{"path": ticketPath, "content": content})
}

func applyAgentSmokeMaxTurns(call func(string, any) error, targetDir, role string, maxTurns int) error {
	manifestPath := filepath.Join(targetDir, ".harness", "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read generated manifest: %w", err)
	}
	updated, changed, err := setManifestRoleMaxTurns(string(data), role, maxTurns)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return call("file_write", map[string]string{
		"path":    ".harness/manifest.yaml",
		"content": updated,
	})
}

func setManifestRoleMaxTurns(content, role string, maxTurns int) (string, bool, error) {
	if maxTurns <= 0 {
		return content, false, nil
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return content, false, fmt.Errorf("validation agent-smoke: manifest role is empty")
	}
	hadTrailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	roleLine := "  " + role + ":"
	start := -1
	for i, line := range lines {
		if line == roleLine {
			start = i
			break
		}
	}
	if start < 0 {
		return content, false, fmt.Errorf("validation agent-smoke: generated manifest missing role %q", role)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "  ") && !strings.HasPrefix(lines[i], "    ") && strings.HasSuffix(strings.TrimSpace(lines[i]), ":") {
			end = i
			break
		}
	}
	maxLine := fmt.Sprintf("    max_turns: %d", maxTurns)
	for i := start + 1; i < end; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "max_turns:") {
			if lines[i] == maxLine {
				return content, false, nil
			}
			lines[i] = maxLine
			out := strings.Join(lines, "\n")
			if hadTrailingNewline {
				out += "\n"
			}
			return out, true, nil
		}
	}
	insertAt := start + 1
	for i := start + 1; i < end; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "model:") {
			insertAt = i + 1
			break
		}
	}
	lines = append(lines[:insertAt], append([]string{maxLine}, lines[insertAt:]...)...)
	out := strings.Join(lines, "\n")
	if hadTrailingNewline {
		out += "\n"
	}
	return out, true, nil
}

func assertAgentSmokeCaseBefore(targetDir string, c AgentSmokeCase) error {
	return assertAgentSmokeCaseArtifacts(targetDir, c, true)
}

func assertAgentSmokeCase(targetDir string, c AgentSmokeCase) error {
	return assertAgentSmokeCaseArtifacts(targetDir, c, false)
}

func assertAgentSmokeCaseArtifacts(targetDir string, c AgentSmokeCase, skipRoleProduced bool) error {
	if _, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(agentSmokeCaseContractPath))); err != nil {
		return fmt.Errorf("required target-local smoke contract missing for %s: %s: %w", c.ID, agentSmokeCaseContractPath, err)
	}
	for _, rel := range c.RequiredArtifacts {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		if skipRoleProduced && agentSmokeRoleProducedArtifact(c, rel) {
			continue
		}
		if _, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(rel))); err != nil {
			return fmt.Errorf("required artifact missing for %s: %s: %w", c.ID, rel, err)
		}
	}
	for _, rel := range c.ForbiddenMutations {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(rel))); err == nil {
			return fmt.Errorf("forbidden mutation present for %s: %s", c.ID, rel)
		}
	}
	return nil
}

func agentSmokeRoleProducedArtifact(c AgentSmokeCase, rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	switch c.Role {
	case "qa":
		return strings.HasPrefix(rel, "docs/reports/qa/")
	case "security":
		return strings.HasPrefix(rel, "docs/reports/security/")
	case "dogfood":
		return strings.HasPrefix(rel, "docs/reports/dogfood/") && rel != "docs/reports/dogfood/seeded-defect.md"
	case "janitor":
		return strings.Contains(c.Stage, "stale") && (rel == "docs/tickets/done" || strings.HasPrefix(rel, "docs/tickets/done/"))
	default:
		return false
	}
}

func assertAgentSmokeCaseAfter(targetDir string, c AgentSmokeCase) error {
	if err := assertAgentSmokeCase(targetDir, c); err != nil {
		return err
	}
	if c.Role == "engineer" {
		if err := assertTicketsDoneNoInProgress(targetDir, c.ID); err != nil {
			return err
		}
	}
	if c.Role == "qa" {
		reportPath := "docs/reports/qa/" + c.ID + ".md"
		if err := assertFileExists(targetDir, reportPath); err != nil {
			return fmt.Errorf("qa report missing for %s: %w", c.ID, err)
		}
		if strings.EqualFold(strings.TrimSpace(c.ExpectedDisposition), "changes_requested") {
			if err := assertFileContainsAny(targetDir, reportPath, []string{"seeded-gap-" + c.ID + ".md", "seeded gap"}); err != nil {
				return fmt.Errorf("qa report for %s did not reference seeded gap: %w", c.ID, err)
			}
		}
	}
	if c.Role == "security" {
		reportPath := "docs/reports/security/" + c.ID + ".md"
		if err := assertFileExists(targetDir, reportPath); err != nil {
			return fmt.Errorf("security report missing for %s: %w", c.ID, err)
		}
	}
	if c.Role == "dogfood" {
		reportPath := "docs/reports/dogfood/" + c.ID + ".md"
		if err := assertFileExists(targetDir, reportPath); err != nil {
			return fmt.Errorf("dogfood report missing for %s: %w", c.ID, err)
		}
		if strings.EqualFold(strings.TrimSpace(c.ExpectedDisposition), "changes_requested") {
			if err := assertFileContainsAny(targetDir, reportPath, []string{"seeded-defect.md", "seeded defect"}); err != nil {
				return fmt.Errorf("dogfood report for %s did not reference seeded defect: %w", c.ID, err)
			}
			if err := assertOpenTicketContains(targetDir, "seeded-defect.md"); err != nil {
				return fmt.Errorf("dogfood finding ticket missing for %s: %w", c.ID, err)
			}
		}
	}
	if c.Role == "janitor" && strings.Contains(c.Stage, "stale") {
		if err := assertTicketsDoneNoInProgress(targetDir, c.ID); err != nil {
			return err
		}
	}
	if c.Role == "pipeline-fixer" && strings.Contains(c.Stage, "failure") {
		checkPath := filepath.Join(targetDir, ".mars", "checks", "latest.json")
		data, err := os.ReadFile(checkPath)
		if err != nil {
			return fmt.Errorf("pipeline-fixer check evidence missing for %s: %w", c.ID, err)
		}
		var check struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &check); err != nil {
			return fmt.Errorf("pipeline-fixer check evidence invalid for %s: %w", c.ID, err)
		}
		if strings.EqualFold(strings.TrimSpace(c.ExpectedDisposition), "blocked") {
			if strings.EqualFold(strings.TrimSpace(check.Status), "passed") {
				return fmt.Errorf("pipeline-fixer blocked case %s rewrote check evidence to passed", c.ID)
			}
			return nil
		}
		if !strings.EqualFold(strings.TrimSpace(check.Status), "passed") {
			return fmt.Errorf("pipeline-fixer check evidence for %s still has status %q, want passed", c.ID, check.Status)
		}
	}
	if c.Role == "release-manager" && strings.Contains(c.Stage, "ready") {
		versionData, err := os.ReadFile(filepath.Join(targetDir, "VERSION"))
		if err != nil {
			return fmt.Errorf("release-manager VERSION missing for %s: %w", c.ID, err)
		}
		version := strings.TrimSpace(string(versionData))
		if version == "" {
			return fmt.Errorf("release-manager VERSION empty for %s", c.ID)
		}
		if version == "0.0.0" || version == "v0.0.0" {
			return fmt.Errorf("release-manager VERSION for %s still has seeded value %q", c.ID, version)
		}
		changelogData, err := os.ReadFile(filepath.Join(targetDir, "CHANGELOG.md"))
		if err != nil {
			return fmt.Errorf("release-manager CHANGELOG missing for %s: %w", c.ID, err)
		}
		seededChangelog := "# Changelog\n\n## Unreleased\n\n- Agent smoke seeded change."
		if strings.TrimSpace(string(changelogData)) == seededChangelog {
			return fmt.Errorf("release-manager CHANGELOG for %s still contains only seeded placeholder evidence", c.ID)
		}
		reportPath := "docs/reports/release/" + c.ID + ".md"
		if err := assertFileExists(targetDir, reportPath); err != nil {
			return fmt.Errorf("release-manager report missing for %s: %w", c.ID, err)
		}
		tag := "v" + strings.TrimPrefix(version, "v")
		out, err := runCommandOutput(context.Background(), targetDir, "git", "tag", "--list", tag)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != tag {
			return fmt.Errorf("release-manager local tag missing for %s: want %s", c.ID, tag)
		}
		tagCommit, err := runCommandOutput(context.Background(), targetDir, "git", "rev-list", "-n", "1", tag)
		if err != nil {
			return err
		}
		headCommit, err := runCommandOutput(context.Background(), targetDir, "git", "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if strings.TrimSpace(tagCommit) != strings.TrimSpace(headCommit) {
			ok, err := releaseTagAllowsRuntimeLearningTail(targetDir, strings.TrimSpace(tagCommit), strings.TrimSpace(headCommit))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("release-manager local tag %s for %s does not point at HEAD or a runtime-learnings-only tail", tag, c.ID)
			}
		}
	}
	return nil
}

func releaseTagAllowsRuntimeLearningTail(targetDir, tagCommit, headCommit string) (bool, error) {
	if tagCommit == "" || headCommit == "" || tagCommit == headCommit {
		return tagCommit != "" && tagCommit == headCommit, nil
	}
	if err := runCommand(context.Background(), targetDir, "git", "merge-base", "--is-ancestor", tagCommit, headCommit); err != nil {
		return false, nil
	}
	diffOut, err := runCommandOutput(context.Background(), targetDir, "git", "diff", "--name-only", tagCommit+".."+headCommit)
	if err != nil {
		return false, err
	}
	found := false
	for _, line := range strings.Split(diffOut, "\n") {
		rel := filepath.ToSlash(strings.TrimSpace(line))
		if rel == "" {
			continue
		}
		found = true
		if rel != ".harness/learnings.yaml" {
			return false, nil
		}
	}
	return found, nil
}

func assertFileExists(targetDir, rel string) error {
	_, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(rel)))
	return err
}

func assertFileContainsAny(targetDir, rel string, needles []string) error {
	data, err := os.ReadFile(filepath.Join(targetDir, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	for _, needle := range needles {
		if strings.Contains(content, strings.ToLower(needle)) {
			return nil
		}
	}
	return fmt.Errorf("%s missing any of %v", rel, needles)
}

func assertTicketsDoneNoInProgress(targetDir, caseID string) error {
	done, err := ticketMarkdownFiles(targetDir, "done")
	if err != nil {
		return err
	}
	if len(done) == 0 {
		return fmt.Errorf("ticket lifecycle incomplete for %s: no ticket markdown files in docs/tickets/done", caseID)
	}
	inProgress, err := ticketMarkdownFiles(targetDir, "in-progress")
	if err != nil {
		return err
	}
	if len(inProgress) > 0 {
		return fmt.Errorf("ticket lifecycle incomplete for %s: stale in-progress tickets remain: %v", caseID, inProgress)
	}
	return nil
}

func assertOpenTicketContains(targetDir, marker string) error {
	for _, status := range []string{"backlog", "in-progress", "in-review"} {
		files, err := ticketMarkdownFiles(targetDir, status)
		if err != nil {
			return err
		}
		for _, file := range files {
			data, err := os.ReadFile(filepath.Join(targetDir, "docs", "tickets", status, file))
			if err != nil {
				return err
			}
			if strings.Contains(strings.ToLower(string(data)), strings.ToLower(marker)) {
				return nil
			}
		}
	}
	return fmt.Errorf("no open ticket references %q", marker)
}

func ticketMarkdownFiles(targetDir, status string) ([]string, error) {
	dir := filepath.Join(targetDir, "docs", "tickets", status)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func classifyAgentSmokeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "policy:"):
		return FailureToolPolicy
	case strings.Contains(msg, "fixture"):
		return FailureFixtureInvalid
	case strings.Contains(msg, "model") || strings.Contains(msg, "llm") || strings.Contains(msg, "inference"):
		return FailureEnvironmentModel
	case strings.Contains(msg, "agent loop") || strings.Contains(msg, "expected disposition") ||
		strings.Contains(msg, "max_turns") || strings.Contains(msg, "empty_response") ||
		strings.Contains(msg, "ticket gate") || strings.Contains(msg, "agent ended"):
		return FailureRoleBehavior
	case strings.Contains(msg, "next_need") || strings.Contains(msg, "suggested role") ||
		strings.Contains(msg, "disposition") || strings.Contains(msg, "dispatch") || strings.Contains(msg, "role ") && strings.Contains(msg, "not found"):
		return FailureDispatchContext
	case strings.Contains(msg, "unknown project type"):
		return FailureProjectTypeGap
	default:
		return FailureFoundationGeneration
	}
}

func writeAgentSmokeRunArtifacts(result AgentSmokeResult) error {
	if result.RunPath == "" {
		return nil
	}
	if err := os.MkdirAll(result.RunPath, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(result.RunPath, "result.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	manifest := map[string]any{
		"id":                   filepath.Base(result.RunPath),
		"case":                 result.CaseID,
		"role":                 result.Role,
		"project_type":         result.ProjectType,
		"target":               result.TargetPath,
		"db":                   result.DBPath,
		"execution_mode":       result.ExecutionMode,
		"job_id":               result.JobID,
		"status":               result.Status,
		"terminal_disposition": result.TerminalDisposition,
		"would_dispatch":       result.WouldDispatch,
		"created_at":           time.Now().UTC().Format(time.RFC3339),
	}
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(result.RunPath, "manifest.json"), append(data, '\n'), 0o644)
}

func discardRunDir(runDir string) error {
	if strings.TrimSpace(runDir) == "" {
		return nil
	}
	return os.RemoveAll(runDir)
}

func cleanupAgentSmokeRuns(root string) (int, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "run-") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return cleaned, err
		}
		cleaned++
	}
	return cleaned, nil
}

func writeAgentSmokeMarkdownReport(path string, report AgentSmokeReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Agent Smoke Report\n\n")
	b.WriteString(report.Summary())
	b.WriteString("\n\n")
	if report.Evidence != "" {
		fmt.Fprintf(&b, "- Evidence source: `%s`\n", report.Evidence)
	}
	if report.ModelSource != "" {
		fmt.Fprintf(&b, "- Model source: %s\n", report.ModelSource)
	}
	if report.SingleServer {
		fmt.Fprintf(&b, "- Inference topology: single local server tier `%s` with server parallel `%d`\n", report.SingleTier, report.ServerParallel)
	} else if report.Evidence == "local-model" {
		b.WriteString("- Inference topology: tiered local router\n")
	}
	if report.Evidence == "endpoint-override" {
		b.WriteString("- Endpoint override note: fake, stub, mock, canned, or scripted endpoints are excluded from validation pass claims by AD-296.\n")
	}
	b.WriteString("\n| Role | Case | Project | Mode | Disposition | Status | Failure | Run |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range report.Results {
		failure := r.FailureClass
		if failure == "" {
			failure = "-"
		}
		run := r.RunPath
		if r.Discarded {
			run = "discarded"
		}
		disposition := r.TerminalDisposition
		if disposition == "" {
			disposition = "-"
		}
		mode := r.ExecutionMode
		if mode == "" {
			mode = "-"
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n", r.Role, r.CaseID, r.ProjectType, mode, disposition, r.Status, failure, run))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func needsPlannerContext(stage string) bool {
	return strings.Contains(stage, "after-ceo") || strings.Contains(stage, "conflicting") || strings.Contains(stage, "roadmap") || strings.Contains(stage, "scope") || strings.Contains(stage, "narrative")
}

func needsCOOContext(stage string) bool {
	return strings.Contains(stage, "after-coo") || strings.Contains(stage, "ticket-gap") || strings.Contains(stage, "duplicate")
}

func needsTicket(stage string) bool {
	return strings.Contains(stage, "ticket") || strings.Contains(stage, "after-engineer") || strings.Contains(stage, "after-qa") || strings.Contains(stage, "ready") || strings.Contains(stage, "defect") || strings.Contains(stage, "failure") || strings.Contains(stage, "stale") || strings.Contains(stage, "blocked")
}

func isTicketClaimStage(stage string) bool {
	return stage == "ticket" || strings.Contains(stage, "rework") || strings.Contains(stage, "gap")
}

func isCompletedTicketStage(stage string) bool {
	return strings.Contains(stage, "after-engineer") || strings.Contains(stage, "after-qa") || strings.Contains(stage, "ready") || strings.Contains(stage, "defect")
}

func specForCase(c AgentSmokeCase) string {
	return fmt.Sprintf("# %s\n\nProject type: %s\nRole under smoke: %s\nStage: %s\n\nBuild a small representative target for compartmentalised MARS role smoke validation.\n\nThe target-local smoke contract lives at `%s`; agents should read that file rather than looking for the foundation matrix inside this generated repo.\n", c.ID, c.ProjectType, c.Role, c.Stage, agentSmokeCaseContractPath)
}

func readmeForCase(c AgentSmokeCase) string {
	return fmt.Sprintf("# %s\n\nEphemeral %s project generated for `%s` agent smoke validation.\n\nRead `%s` for the exact case contract, expected terminal disposition, and suppressed follow-on handoff.\n", c.ID, c.ProjectType, c.Role, agentSmokeCaseContractPath)
}

func agentSmokeStaticWebPort(c AgentSmokeCase) int {
	sum := sha1.Sum([]byte(c.Role + "/" + c.ID))
	slot := (int(sum[0]) << 8) + int(sum[1])
	return 19000 + slot%1000
}

func caseContractForCase(c AgentSmokeCase) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Agent Smoke Case Contract\n\n")
	fmt.Fprintf(&b, "- Case: `%s`\n", c.ID)
	fmt.Fprintf(&b, "- Role under test: `%s`\n", c.Role)
	fmt.Fprintf(&b, "- Project type: `%s`\n", c.ProjectType)
	fmt.Fprintf(&b, "- Stage: `%s`\n", c.Stage)
	fmt.Fprintf(&b, "- Expected terminal disposition: `%s`\n", c.ExpectedDisposition)
	if c.WouldDispatch != "" {
		fmt.Fprintf(&b, "- Suppressed follow-on dispatch: `%s`\n", c.WouldDispatch)
	}
	if nextNeed := strings.TrimSpace(c.Trigger["next_need"]); nextNeed != "" {
		fmt.Fprintf(&b, "- Expected next_need or handoff theme: `%s`\n", nextNeed)
	}
	b.WriteString("\n## Source Of Truth\n\n")
	b.WriteString("This file is the target-local smoke contract. Do not try to read `docs/validation/agent-smoke/matrix.yaml` from this target repo; that matrix lives in the foundation repo and is only the generator input.\n\n")
	b.WriteString("Act as the role under test, complete the smallest role-owned unit of work that proves this checkpoint is executable, and finish by calling `job_disposition_record` with evidence. Follow-on dispatch is suppressed by the runner, so name the would-be handoff in the disposition instead of enqueueing another role.\n\n")
	if len(c.RequiredArtifacts) > 0 {
		b.WriteString("## Required Artifacts\n\n")
		for _, rel := range c.RequiredArtifacts {
			if strings.TrimSpace(rel) != "" {
				fmt.Fprintf(&b, "- `%s`\n", rel)
			}
		}
		b.WriteString("\n")
	}
	if len(c.ForbiddenMutations) > 0 {
		b.WriteString("## Forbidden Mutations\n\n")
		for _, rel := range c.ForbiddenMutations {
			if strings.TrimSpace(rel) != "" {
				fmt.Fprintf(&b, "- `%s`\n", rel)
			}
		}
		b.WriteString("\n")
	}
	b.WriteString(projectTypeSmokeInstructions(c))
	b.WriteString(roleSmokeCaseInstructions(c))
	return b.String()
}

func projectTypeSmokeInstructions(c AgentSmokeCase) string {
	switch c.ProjectType {
	case "static-web":
		port := strconv.Itoa(agentSmokeStaticWebPort(c))
		return `## Project Surface

Use the existing root ` + "`index.html`" + `, ` + "`style.css`" + `, and ` + "`app.js`" + ` files for this static web target. Do not create a parallel ` + "`src/`" + ` tree unless the ticket explicitly asks for one. Focused validation should use ` + "`node --check app.js`" + ` plus this deterministic static HTTP smoke when useful:

1. Start the server with ` + "`shell_exec`" + ` argv ` + "`[\"python3\",\"-m\",\"http.server\",\"" + port + "\"]`" + ` and ` + "`background:true`" + `.
2. Probe ` + "`http://localhost:" + port + "/`" + ` with ` + "`curl`" + `.
3. After one HTTP 200, stop the tracked PID if it is still running. If ` + "`kill`" + ` reports that the process already stopped after the successful probe, do not retry cleanup, do not call empty/no-op shell commands, and proceed to evidence/report/disposition.

Do not use ` + "`python`" + `, default port ` + "`8080`" + `, or ` + "`curl`" + ` against localhost before starting the server.

`
	case "react-web":
		return `## Project Surface

Use the existing ` + "`index.html`" + `, ` + "`package.json`" + `, ` + "`src/App.jsx`" + `, ` + "`src/App.test.jsx`" + `, and ` + "`src/main.jsx`" + ` files. Keep work in the Vite/React surface and record build or test evidence when dependency state allows. If dependencies are missing or the generated lockfile is not yet hydrated, use ` + "`dependency_sync`" + ` with ` + "`{\"action\":\"install\",\"package_manager\":\"npm\",\"frozen\":false,\"reason\":\"hydrate ephemeral React validation dependencies\"}`" + ` before ` + "`npm run build`" + `, then commit the generated ` + "`package-lock.json`" + ` as validation provenance before terminal disposition. Browser-framework approval also needs a product smoke; for this target use ` + "`shell_exec`" + ` argv ` + "`[\"node\",\"-e\",\"const fs=require('fs'); const html=fs.readFileSync('index.html','utf8'); const main=fs.readFileSync('src/main.jsx','utf8'); const app=fs.readFileSync('src/App.jsx','utf8'); if(!html.includes('/src/main.jsx')) throw new Error('missing main.jsx module script'); if(!main.includes('createRoot')) throw new Error('missing createRoot mount'); if(!app.includes('id=\\\"game\\\"')) throw new Error('missing #game UI marker'); if(!app.toLowerCase().includes('score')) throw new Error('missing score UI state'); console.log('browser smoke: React document.querySelector #game score UI state');\"]`" + ` after build passes.

`
	case "browser-game-phaser":
		return `## Project Surface

Use the existing Phaser/Vite files under ` + "`index.html`" + `, ` + "`src/main.js`" + `, and ` + "`src/game/`" + `. Keep game changes in the generated scene/player modules and record build or source-level smoke evidence. If ` + "`package-lock.json`" + ` is missing, call ` + "`dependency_sync`" + ` with ` + "`frozen:false`" + ` and a reason before the build, then commit the generated ` + "`package-lock.json`" + ` as validation provenance before terminal disposition. Run ` + "`npm run build`" + `, then run exactly this source/runtime browser-product smoke without importing Phaser directly in Node and without reading ` + "`dist/`" + ` or ` + "`dist/assets/`" + `: ` + "`" + phaserAgentSmokeSourceSmokeCommand() + "`" + `.

`
	case "canvas-game-vanilla":
		return `## Project Surface

Use the existing root ` + "`index.html`" + `, ` + "`game.js`" + `, ` + "`style.css`" + `, and optional ` + "`tests/game-state.test.js`" + ` files. Prefer focused JavaScript checks over introducing package infrastructure.

`
	case "go-api":
		return `## Project Surface

Use the existing Go API files under ` + "`cmd/api/`" + ` and ` + "`internal/api/`" + `. Focused validation is ` + "`go test ./...`" + `. When that passes, do not run ` + "`go build ./...`" + ` or any extra build probe for this smoke case.

`
	case "go-cli":
		return `## Project Surface

Use the existing Go CLI files under ` + "`cmd/tool/`" + ` and ` + "`internal/tool/`" + `. Focused validation is ` + "`go test ./...`" + ` plus direct CLI behavior only when the contract asks for it. When tests pass, do not run ` + "`go build ./...`" + ` or any extra build probe for this smoke case.

`
	case "go-library":
		return `## Project Surface

Use the existing Go library files under ` + "`pkg/`" + `. Focused validation is ` + "`go test ./...`" + `. When that passes, do not run ` + "`go build ./...`" + ` or any extra build probe for this smoke case.

`
	case "docs-site":
		return `## Project Surface

Use the existing ` + "`README.md`" + `, ` + "`docs/index.md`" + `, and ` + "`docs/content/`" + ` files. Produce documentation evidence instead of inventing runtime code.

`
	case "existing-maintenance":
		return `## Project Surface

Treat this as a pre-existing maintenance target. Preserve the generated project shape, inspect stale tickets or reports, and avoid broad rewrites.

`
	default:
		return ""
	}
}

func roleSmokeCaseInstructions(c AgentSmokeCase) string {
	expected := strings.TrimSpace(c.ExpectedDisposition)
	if expected == "" {
		expected = "completed"
	}
	switch c.Role {
	case "engineer":
		ticketStep := "2. Claim exactly one eligible ticket from `docs/tickets/backlog/` into `docs/tickets/in-progress/` before product mutation."
		lifecycleStep := "6. Update ticket evidence, move the ticket to `docs/tickets/done/`, commit that lifecycle change, and record `" + expected + "` with `next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + "."
		noopStep := "7."
		if isTicketClaimStage(c.Stage) {
			src := seededTicketPath(c, "in-progress")
			dst := seededTicketPath(c, "done")
			ticketStep = "2. The active seed ticket is already at `" + src + "`; do not search `docs/tickets/backlog/` for this case."
			lifecycleStep = "6. Before closing, update the in-progress ticket with `evidence_links` naming the focused validation and `verified_by: engineer`, then run `git_status` and `git_commit` while the ticket is still in-progress.\n7. Move the committed ticket with exactly this command: `shell_exec` argv `[\"git\",\"mv\",\"" + src + "\",\"" + dst + "\"]`, then `git_status`, `git_commit`, and call `job_disposition_record` exactly as required by the terminal disposition contract: " + terminalDispositionInstruction(c)
			noopStep = "8."
		}
		return `## Role-Specific Completion Contract

1. Read the active ticket and feature contract.
` + ticketStep + `
3. Implement the smallest project-appropriate change that satisfies the ticket and F-001 scenario.
4. Run focused validation evidence for the project type` + engineerValidationHint(c) + `. If the generated fixture already satisfies the ticket after validation, do not invent extra features.
5. Commit implementation changes before moving the ticket to done.
` + lifecycleStep + `
` + noopStep + ` Never call ` + "`shell_exec`" + ` with empty argv or a no-op command; a no-op runtime probe is a policy blocker.
`
	case "cto-weekly":
		return `## Role-Specific Completion Contract

1. Read the current operating plan and F-001 feature contract.
2. Create exactly one implementation ticket for the current failing scenario. Use ` + "`ticket_create`" + ` with ` + "`\"bdd_scenarios\":[\"F-001-S001\"]`" + ` as a real JSON array, not a quoted string.
3. This checkpoint starts with no implementation ticket; do not spend turns proving the backlog is empty after this contract is read.
4. Do not create a separate validation ticket unless the case contract explicitly asks for one.
5. After ` + "`ticket_create`" + ` returns the created path, do not loop on ` + "`git_diff`" + `. ` + "`git_status`" + ` is enough; immediately call ` + "`git_commit`" + ` with all dirty changes, using a message such as ` + "`chore(tickets): create implementation ticket`" + `.
6. Record ` + "`" + expected + "`" + ` with ` + "`next_need: implementation`" + ` and suggested role ` + "`engineer`" + `.
`
	case "orchestrator":
		if expected == "blocked" {
			return `## Role-Specific Completion Contract

1. Read any source disposition under ` + "`.mars/orgstate/source-disposition.json`" + ` and seeded check state when present.
2. This case expects a stop, not a dispatch. Do not choose a next role.
3. Do not mutate product, planning, or ticket artifacts.
4. Record ` + "`blocked`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + `.
`
		}
		return `## Role-Specific Completion Contract

1. Read any source disposition under ` + "`.mars/orgstate/source-disposition.json`" + ` when present.
2. Choose the manifest-valid next role represented by the suppressed follow-on dispatch.
3. Do not mutate product, planning, or ticket artifacts.
4. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and a handoff ask naming the smallest next action.
`
	case "qa":
		if expected == "changes_requested" {
			return `## Role-Specific Completion Contract

1. Inspect the in-review ticket evidence and the seeded validation gap at ` + "`docs/reports/qa/seeded-gap-" + c.ID + ".md`" + `.
2. This case intentionally lacks acceptable QA evidence; write ` + "`docs/reports/qa/" + c.ID + ".md`" + ` and name the seeded gap marker.
3. Run ` + "`git_status`" + ` and ` + "`git_commit`" + ` to commit the QA report and any ` + "`.harness/learnings.yaml`" + ` update before terminal disposition.
4. Record ` + "`changes_requested`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + `. Do not approve.
`
		}
		staticStep := "3. Run the project-appropriate bounded smoke evidence from the Project Surface section when applicable.\n"
		if c.ProjectType == "static-web" {
			port := strconv.Itoa(agentSmokeStaticWebPort(c))
			staticStep = "\n3. For this static-web case, use `node --check app.js`, `python3 -m http.server " + port + "` with `background:true`, and one `curl http://localhost:" + port + "/` HTTP 200 probe. Do not use `python` or `curl` localhost before starting the server.\n"
		}
		return `## Role-Specific Completion Contract

1. Inspect the in-review or completed ticket evidence and the feature contract.
2. Produce QA evidence at ` + "`docs/reports/qa/" + c.ID + ".md`" + `.
` + staticStep + `4. Run ` + "`git_status`" + ` and ` + "`git_commit`" + ` to commit the QA report and any ` + "`.harness/learnings.yaml`" + ` update; ` + "`job_disposition_record`" + ` is policy-blocked while the report is uncommitted.
5. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` when the smoke evidence is sufficient, or a real blocking disposition if it is not.
`
	case "security":
		return `## Role-Specific Completion Contract

1. Inspect QA evidence, the feature contract, and the relevant source surface.
2. Produce security evidence at ` + "`docs/reports/security/" + c.ID + ".md`" + `.
3. Do not use the generic dated ` + "`docs/reports/security/security-audit-[date].md`" + ` path for agent-smoke; this exact case report path is the required deliverable.
4. Immediately after writing that report, run ` + "`git_status`" + ` and ` + "`git_commit`" + ` to commit the security report and any ` + "`.harness/learnings.yaml`" + ` update before ` + "`docsync_audit`" + ` or terminal disposition.
5. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` for acceptable risk or a real blocking disposition with evidence.
`
	case "dogfood":
		if expected == "changes_requested" {
			return `## Role-Specific Completion Contract

1. Inspect the completed ticket, seeded evidence, relevant project surface, ` + "`.mars/checks/latest.json`" + `, and ` + "`docs/reports/dogfood/seeded-defect.md`" + `.
2. This case intentionally contains a user-visible defect or missing evidence. Passing unit tests do not clear this held-out gap. Do not approve it.
3. Write dogfood evidence at ` + "`docs/reports/dogfood/" + c.ID + ".md`" + ` that references ` + "`seeded-defect.md`" + ` and describes the exact seeded defect/gap.
4. Create exactly one target-owned finding ticket with ` + "`ticket_create`" + ` if no suitable open finding already exists; the ticket title or body must reference ` + "`seeded-defect.md`" + `. Commit the ticket/report and any ` + "`.harness/learnings.yaml`" + ` update.
5. Record ` + "`changes_requested`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and include the finding ticket id/path when one was created.
`
		}
		staticStep := ""
		if c.ProjectType == "static-web" {
			port := strconv.Itoa(agentSmokeStaticWebPort(c))
			staticStep = "\n2. For this static-web case, use `python3 -m http.server " + port + "` with `background:true` and one `curl http://localhost:" + port + "/` HTTP 200 probe. Do not use `python`, default port `8080`, or `curl` localhost before starting the server.\n3. After HTTP 200, stop the tracked PID if it is still running. If `kill` reports the process already stopped after the successful probe, do not retry cleanup, do not call empty/no-op shell commands, and continue to evidence/disposition.\n"
		} else if c.ProjectType == "browser-game-phaser" {
			staticStep = "\n2. Install dependencies if needed, run `npm run build`, then run a source/runtime browser-product smoke that prints `browser smoke: Phaser canvas #game new Phaser.Game` and checks `index.html`, `src/main.js`, `new Phaser.Game`, and `parent`.\n3. If build and browser-product smoke pass, do not create a finding ticket.\n"
		} else if c.ProjectType == "react-web" {
			staticStep = "\n2. If dependencies are missing, call `dependency_sync` with `frozen:false` as a boolean and a reason, then run `npm run build`.\n3. Run the React source/runtime browser-product smoke from the Project Surface section and confirm it prints `browser smoke: React document.querySelector #game score UI state`.\n4. If build and browser-product smoke pass, do not create a finding ticket.\n"
		} else if c.ProjectType == "go-api" || c.ProjectType == "go-cli" || c.ProjectType == "go-library" {
			staticStep = "\n2. Run `go test ./...` as the bounded user smoke for this Go target. Do not run `go build ./...`, do not issue empty argv/no-op shell commands, and do not create a finding ticket when tests pass.\n"
		} else {
			staticStep = "\n2. Run one project-appropriate user smoke.\n"
		}
		return `## Role-Specific Completion Contract

1. Inspect release/readiness evidence and run or review a project-appropriate smoke.
` + staticStep + `4. Produce dogfood evidence at ` + "`docs/reports/dogfood/" + c.ID + ".md`" + `, commit it, and record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` when the user-visible behavior has enough evidence for this bounded case.
`
	case "release-manager":
		if expected == "blocked" {
			return `## Role-Specific Completion Contract

1. Inspect seeded release/check evidence such as ` + "`.mars/checks/latest.json`" + `.
2. This case is blocked by design. Do not run release notes, do not create tags, do not push, and do not publish assets.
3. Produce or update release evidence under ` + "`docs/reports/release/`" + ` if needed.
4. Record ` + "`blocked`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + `.
`
		}
		if c.ProjectType == "docs-site" {
			return `## Role-Specific Completion Contract

1. Inspect docs content, VERSION, CHANGELOG, and seeded review evidence.
2. This docs-site ready case is notes-only: no build, asset, runtime, remote, or GitHub Release gate is required.
3. Write ` + "`docs/reports/release/" + c.ID + ".md`" + ` before committing release notes.
4. Advance VERSION beyond the seeded ` + "`0.0.0`" + ` value and expand CHANGELOG beyond the seeded placeholder, then commit VERSION, CHANGELOG, the release report, and any ` + "`.harness/learnings.yaml`" + ` update together with message ` + "`release: notes <VERSION>`" + `.
5. Create the local tag ` + "`v<VERSION>`" + ` at that release-note HEAD if it is missing.
6. Record ` + "`completed`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and local notes-only release readiness evidence.
`
		}
		return `## Role-Specific Completion Contract

1. Inspect QA, security, dogfood, ticket, and changelog evidence.
2. This is an ephemeral local-release smoke target. A missing GitHub remote or GitHub Release is not a blocker for a ` + "`ready`" + ` case; local VERSION, CHANGELOG, tag, and release evidence are enough.
3. Use the ` + "`mars_cli`" + ` tool when invoking harness release workflows; do not run a shell executable named ` + "`mars_cli`" + `.
4. Write ` + "`docs/reports/release/" + c.ID + ".md`" + ` before committing release notes.
5. Advance VERSION beyond the seeded ` + "`0.0.0`" + ` value and expand CHANGELOG beyond the seeded placeholder, then commit VERSION, CHANGELOG, the release report, and any ` + "`.harness/learnings.yaml`" + ` update together with message ` + "`release: notes <VERSION>`" + `.
6. Create the local tag ` + "`v<VERSION>`" + ` at that release-note HEAD if it is missing.
7. Do not call ` + "`git_push`" + `, ` + "`github_release_status`" + `, or block solely because no remote is configured for this ephemeral ready case.
8. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and local release readiness evidence.
`
	case "dependency-manager":
		return `## Role-Specific Completion Contract

1. Inspect package/module state for the generated project type.
2. Make only dependency-owned updates or record a no-op dependency decision with evidence.
3. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` when dependency risk is addressed for the bounded case.
`
	case "pipeline-fixer":
		if expected == "blocked" {
			return `## Role-Specific Completion Contract

1. Inspect the seeded foundation/runtime failure and relevant maintenance surface.
2. This case is blocked by design. Do not rewrite ` + "`.mars/checks/latest.json`" + ` to ` + "`passed`" + `.
3. Commit only role-owned blocker evidence if changed.
4. Record ` + "`blocked`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and the exact operator/foundation blocker.
`
		}
		return `## Role-Specific Completion Contract

1. Inspect the seeded failing check state and relevant build/test surface.
2. Run focused validation for this project type` + pipelineFixerValidationHint(c) + `.
3. Repair the pipeline-owned failure or, if focused validation already passes, call ` + "`file_write`" + ` on ` + "`.mars/checks/latest.json`" + ` with JSON like ` + "`{\"case\":\"" + c.ID + "\",\"status\":\"passed\",\"role\":\"pipeline-fixer\",\"evidence\":[" + pipelineFixerEvidenceExample(c) + "]}`" + `.
4. Commit check/test updates and any ` + "`.harness/learnings.yaml`" + ` update before terminal disposition.
5. Do not keep searching for CI workflow files when this ephemeral target has none.
6. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` after a focused validation command or blocker classification.
`
	case "janitor":
		if strings.Contains(c.Stage, "stale") {
			src := seededTicketPath(c, "in-progress")
			dst := seededTicketPath(c, "done")
			return `## Role-Specific Completion Contract

1. Inspect the stale in-progress ticket containing the ` + "`Agent Smoke Stale Marker`" + ` and the current hygiene signal in ` + "`.mars/checks/latest.json`" + `.
2. Do not call ` + "`file_write`" + ` under ` + "`docs/tickets/`" + `; ticket-file writes are policy-blocked and will prevent terminal disposition.
3. Resolve only the stale ticket lifecycle with exactly this move command: ` + "`shell_exec`" + ` argv ` + "`[\"mv\",\"" + src + "\",\"" + dst + "\"]`" + `.
4. Leave no ticket markdown files under ` + "`docs/tickets/in-progress/`" + `.
5. Commit the lifecycle cleanup and record ` + "`completed`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + `.
`
		}
		if expected == "blocked" {
			return `## Role-Specific Completion Contract

1. Inspect the blocked janitor signal in ` + "`.mars/checks/latest.json`" + ` and the relevant project surface.
2. Do not mutate product code or invent cleanup work for this blocked held-out case.
3. Record ` + "`blocked`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + `.
`
		}
		return `## Role-Specific Completion Contract

1. Inspect stale tickets, clean-tree state, and hygiene signals.
2. Make only maintenance-owned lifecycle or cleanup updates.
3. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and cleanup evidence or a real stop reason.
`
	case "foundation-maintainer":
		return `## Role-Specific Completion Contract

1. Classify the fixture signal as foundation-owned, deployed-owned, mixed, or evidence-only.
2. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and source-maintenance evidence; do not convert this into target product delivery.
`
	default:
		return `## Role-Specific Completion Contract

1. Read the seeded goals, strategy, feature contract, and current plan that belong to this lifecycle checkpoint.
2. Make only role-owned planning or decision artifacts needed for the next lifecycle handoff.
3. Record ` + "`" + expected + "`" + ` with ` + "`next_need: " + nextNeedForCase(c) + "`" + suggestedRoleClause(c) + ` and evidence.
`
	}
}

func nextNeedForCase(c AgentSmokeCase) string {
	nextNeed := strings.TrimSpace(c.Trigger["next_need"])
	if nextNeed == "" {
		return "no_need"
	}
	return nextNeed
}

func suggestedRoleForCase(c AgentSmokeCase) string {
	if role := strings.TrimSpace(c.Trigger["suggested_role"]); role != "" {
		return role
	}
	return strings.TrimSpace(c.WouldDispatch)
}

func suggestedRoleClause(c AgentSmokeCase) string {
	if role := suggestedRoleForCase(c); role != "" {
		return " and suggested role `" + role + "`"
	}
	return " and no suggested role"
}

func suggestedRoleSummary(c AgentSmokeCase) string {
	if role := suggestedRoleForCase(c); role != "" {
		return " and suggested_role " + role
	}
	return " and no suggested role"
}

func featureContractForCase(c AgentSmokeCase) string {
	return fmt.Sprintf(`# F-001: Product Walking Skeleton

- Feature ID: F-001
- Status: in-progress
- Owner: %s

## Business Logic

This contract defines the bounded %s behavior used by the %s smoke case.

## Scenario Schedule

1. F-001-S001 - Deliver the first observable %s behavior.
2. F-001-S002 - Validate and report the behavior with durable evidence.

## Scenarios

### F-001-S001: First Observable Behavior

Given this ephemeral target is generated for %s
When the role under test runs
Then it should act only within its lifecycle ownership boundary.
`, c.Role, c.ProjectType, c.ID, c.ProjectType, c.Role)
}

func strategyForCase(c AgentSmokeCase) string {
	return fmt.Sprintf("# CEO Strategy\n\nPrior strategy for `%s`: prioritize the smallest useful %s slice and preserve validation evidence.\n", c.ID, c.ProjectType)
}

func planForCase(c AgentSmokeCase) string {
	return fmt.Sprintf("# Current Operating Plan\n\n## Goal\n\nDeliver the first `%s` scenario for `%s`.\n\n## Current Failing Scenario\n\nF-001-S001\n", c.ProjectType, c.ID)
}

func ticketArgsForCase(c AgentSmokeCase) map[string]any {
	args := map[string]any{
		"title":               titleForCase(c),
		"priority":            "medium",
		"complexity":          "small",
		"work_type":           "feature",
		"bdd_scenarios":       []string{"F-001-S001"},
		"end_to_end_evidence": "required",
		"owner":               c.Role,
		"last_attempt":        "TBD",
		"blocker":             "none",
		"trace_id":            "TBD",
		"next_action":         "Run the role smoke validation case.",
		"dedupe_key":          "agent-smoke:" + c.ID,
		"source":              "docs/validation/agent-smoke/matrix.yaml",
		"body":                fmt.Sprintf("## Context\n\nSeed ticket for `%s`.\n\n## Requirements\n\n- Preserve role ownership boundaries.\n- Produce validation evidence for F-001-S001.\n\n## Acceptance Criteria\n\n- Required artifacts exist.\n- Terminal disposition matches `%s`.\n", c.ID, c.ExpectedDisposition),
	}
	if isCompletedTicketStage(c.Stage) {
		args["evidence_links"] = []string{"agent-smoke fixture evidence: " + c.ID}
		args["verified_by"] = "foundation-validation-seeder"
	}
	return args
}

func titleForCase(c AgentSmokeCase) string {
	return "Exercise " + c.ProjectType + " smoke case " + c.ID
}

func seededTicketPath(c AgentSmokeCase, status string) string {
	return filepath.ToSlash(filepath.Join("docs", "tickets", status, "T-001-"+agentSmokeTicketSlug(titleForCase(c))+".md"))
}

func agentSmokeTicketSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	return s
}

func engineerValidationHint(c AgentSmokeCase) string {
	switch c.ProjectType {
	case "canvas-game-vanilla":
		return `, starting with ` + "`node tests/game-state.test.js`"
	case "go-api", "go-cli", "go-library":
		return `, starting and ending with ` + "`go test ./...`" + ` when it passes`
	case "static-web":
		return `, starting with ` + "`node --check app.js`"
	case "react-web":
		return `, starting with ` + "`dependency_sync`" + ` ` + "`frozen:false`" + ` when no lockfile exists, then ` + "`npm run build`" + ` and the exact React source smoke from the Project Surface; do not read ` + "`dist/`"
	case "browser-game-phaser":
		return `, starting with ` + "`dependency_sync`" + ` ` + "`frozen:false`" + ` when no lockfile exists, then ` + "`npm run build`" + ` and the exact Phaser source smoke from the Project Surface; do not read ` + "`dist/`"
	default:
		return ""
	}
}

func pipelineFixerValidationHint(c AgentSmokeCase) string {
	switch c.ProjectType {
	case "react-web":
		return `, using ` + "`dependency_sync`" + ` ` + "`frozen:false`" + ` when dependencies or lockfile hydration are missing, then ` + "`npm run build`" + ` and the exact React source smoke from the Project Surface; do not run ` + "`go test ./...`" + ` in this non-Go target`
	case "browser-game-phaser":
		return `, using ` + "`dependency_sync`" + ` ` + "`frozen:false`" + ` when dependencies or lockfile hydration are missing, then ` + "`npm run build`" + ` and the exact Phaser source smoke from the Project Surface; do not run ` + "`go test ./...`" + ` in this non-Go target`
	case "canvas-game-vanilla":
		return `, starting with ` + "`node tests/game-state.test.js`"
	case "go-api", "go-cli", "go-library", "existing-maintenance":
		return `, starting and ending with ` + "`go test ./...`" + ` when a Go module is present`
	default:
		return ""
	}
}

func pipelineFixerEvidenceExample(c AgentSmokeCase) string {
	switch c.ProjectType {
	case "react-web":
		return `"npm run build","browser smoke: React document.querySelector #game score UI state"`
	case "browser-game-phaser":
		return `"npm run build","browser smoke: Phaser canvas #game new Phaser.Game"`
	case "canvas-game-vanilla":
		return `"node tests/game-state.test.js"`
	default:
		return `"go test ./..."`
	}
}

func phaserAgentSmokeSourceSmokeCommand() string {
	return `shell_exec argv ["node","-e","const fs=require('fs'); const html=fs.readFileSync('index.html','utf8'); const main=fs.readFileSync('src/main.js','utf8'); if(!html.includes('/src/main.js')&&!html.includes('src/main.js')&&!html.includes('main.js')) throw new Error('missing main.js module script'); if(!main.includes(\"import Phaser from 'phaser'\")&&!main.includes('import Phaser from \"phaser\"')) throw new Error('missing Phaser import'); if((main.split('new Phaser.Game').length-1)!==1) throw new Error('expected exactly one new Phaser.Game'); if(!main.includes('parent')) throw new Error('missing parent game container'); console.log('browser smoke: Phaser canvas #game new Phaser.Game');"]`
}

func seedProjectFiles(call func(string, any) error, c AgentSmokeCase) error {
	switch c.ProjectType {
	case "static-web":
		return seedFiles(call, map[string]string{
			"index.html": htmlDoc("Static Web Smoke"),
			"style.css":  cssDoc("body { font-family: sans-serif; }\n"),
			"app.js":     jsDoc("export function ready() { return true; }\n"),
		})
	case "react-web":
		return seedFiles(call, map[string]string{
			"package.json":      `{"scripts":{"build":"vite build","test":"vitest run"},"dependencies":{"@vitejs/plugin-react":"latest","vite":"latest","react":"latest","react-dom":"latest"},"devDependencies":{"vitest":"latest"}}` + "\n",
			"package-lock.json": "{}\n",
			"index.html":        reactIndexDoc(),
			"src/App.jsx":       jsDoc("export default function App() { return <main id=\"game\">Agent smoke score ready</main>; }\n"),
			"src/App.test.jsx":  jsDoc("import { describe, it, expect } from 'vitest'; describe('smoke', () => { it('works', () => expect(true).toBe(true)); });\n"),
			"src/main.jsx":      jsDoc("import React from 'react'; import { createRoot } from 'react-dom/client'; import App from './App.jsx'; createRoot(document.getElementById('root')).render(<App />);\n"),
		})
	case "browser-game-phaser":
		return seedFiles(call, map[string]string{
			"package.json":       `{"scripts":{"build":"vite build"},"dependencies":{"phaser":"latest","vite":"latest"},"devDependencies":{}}` + "\n",
			"index.html":         phaserIndexDoc(),
			"src/main.js":        jsDoc("import Phaser from 'phaser'; import { SmokeScene } from './game/scene.js'; new Phaser.Game({ type: Phaser.AUTO, parent: 'game', scene: [SmokeScene] });\n"),
			"src/game/scene.js":  jsDoc("import Phaser from 'phaser'; export class SmokeScene extends Phaser.Scene { create() { this.ready = true; } }\n"),
			"src/game/player.js": jsDoc("export const player = { x: 0, y: 0 };\n"),
		})
	case "canvas-game-vanilla":
		return seedFiles(call, map[string]string{
			"index.html":               htmlDoc("Canvas Game Smoke"),
			"game.js":                  jsDoc("export const gameState = { running: true };\n"),
			"style.css":                cssDoc("canvas { border: 1px solid #333; }\n"),
			"tests/game-state.test.js": jsDoc("import { gameState } from '../game.js'; if (!gameState.running) throw new Error('not running');\n"),
		})
	case "go-api":
		return seedFiles(call, map[string]string{
			"go.mod":                       "module smoke/api\n\ngo 1.22\n",
			"cmd/api/main.go":              goDoc("package main\n\nfunc main() {}\n"),
			"internal/api/handler.go":      goDoc("package api\n\nfunc Health() string { return \"ok\" }\n"),
			"internal/api/handler_test.go": goDoc("package api\n\nimport \"testing\"\n\nfunc TestHealth(t *testing.T) { if Health() != \"ok\" { t.Fatal(\"bad health\") } }\n"),
		})
	case "go-cli":
		return seedFiles(call, map[string]string{
			"go.mod":                       "module smoke/tool\n\ngo 1.22\n",
			"cmd/tool/main.go":             goDoc("package main\n\nfunc main() {}\n"),
			"internal/tool/parser.go":      goDoc("package tool\n\nfunc Parse(v string) string { return v }\n"),
			"internal/tool/parser_test.go": goDoc("package tool\n\nimport \"testing\"\n\nfunc TestParse(t *testing.T) { if Parse(\"x\") != \"x\" { t.Fatal(\"bad parse\") } }\n"),
		})
	case "go-library":
		return seedFiles(call, map[string]string{
			"go.mod":                  "module smoke/library\n\ngo 1.22\n",
			"pkg/smoke/smoke.go":      goDoc("package smoke\n\nfunc Ready() bool { return true }\n"),
			"pkg/smoke/smoke_test.go": goDoc("package smoke\n\nimport \"testing\"\n\nfunc TestReady(t *testing.T) { if !Ready() { t.Fatal(\"not ready\") } }\n"),
		})
	case "docs-site":
		return seedFiles(call, map[string]string{
			"docs/index.md":         "# Docs Site\n\nWelcome to the smoke docs.\n",
			"docs/content/intro.md": "# Introduction\n\nSeed content for validation.\n",
		})
	case "existing-maintenance":
		return seedFiles(call, map[string]string{
			"main.go":                  goDoc("package main\n\nfunc main() {}\n"),
			".mars/checks/latest.json": `{"name":"unit","status":"failed","role":"pipeline-fixer"}` + "\n",
		})
	default:
		return fmt.Errorf("unknown project type %q", c.ProjectType)
	}
}

func seedFiles(call func(string, any) error, files map[string]string) error {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, path := range keys {
		if err := call("file_write", map[string]string{"path": path, "content": files[path]}); err != nil {
			return err
		}
	}
	return nil
}

func seedReports(call func(string, any) error, c AgentSmokeCase) error {
	reports := map[string]string{}
	if (strings.Contains(c.Stage, "after-engineer") || strings.Contains(c.Stage, "after-qa") || strings.Contains(c.Stage, "ready")) && c.Role != "qa" {
		reports["docs/reports/qa/"+c.ID+".md"] = "# QA Evidence\n\nValidation evidence seeded for agent smoke.\n"
	}
	if (strings.Contains(c.Stage, "after-qa") || strings.Contains(c.Stage, "ready")) && c.Role != "security" {
		reports["docs/reports/security/"+c.ID+".md"] = "# Security Evidence\n\nSecurity approval seeded for agent smoke.\n"
	}
	if c.Role == "qa" && strings.EqualFold(strings.TrimSpace(c.ExpectedDisposition), "changes_requested") {
		reports["docs/reports/qa/seeded-gap-"+c.ID+".md"] = "# Seeded QA Evidence Gap\n\nThis held-out case intentionally lacks acceptable validation evidence. QA must request implementation rework and reference this marker in the case report.\n"
	}
	if strings.Contains(c.Stage, "ready") {
		if c.Role != "dogfood" {
			reports["docs/reports/dogfood/"+c.ID+".md"] = "# Dogfood Evidence\n\nDogfood validation seeded for agent smoke.\n"
		}
		reports["CHANGELOG.md"] = "# Changelog\n\n## Unreleased\n\n- Agent smoke seeded change.\n"
		reports["VERSION"] = "0.0.0\n"
	}
	if strings.Contains(c.Stage, "defect") {
		reports["docs/reports/dogfood/seeded-defect.md"] = "# Seeded Dogfood Defect\n\nThis held-out case intentionally contains a user-visible validation gap. Dogfood must request implementation rework even if unit tests pass.\n\nSeeded gap: product-level dogfood evidence is incomplete for the finished ticket; create one finding ticket and record `changes_requested`.\n"
	}
	return seedFiles(call, reports)
}

func seedSpecialState(call func(string, any) error, c AgentSmokeCase) error {
	files := map[string]string{}
	if strings.Contains(c.Stage, "failure") || strings.Contains(c.Stage, "blocked") || strings.Contains(c.Stage, "stale") || strings.Contains(c.Stage, "defect") ||
		strings.Contains(c.ID, "failure") || strings.Contains(c.ID, "blocked") || strings.Contains(c.ID, "stale") || strings.Contains(c.ID, "defect") {
		files[".mars/checks/latest.json"] = fmt.Sprintf(`{"case":%q,"status":"failed","role":%q}`+"\n", c.ID, c.Role)
	}
	if c.Role == "orchestrator" {
		trigger, _ := json.Marshal(c.Trigger)
		files[".mars/orgstate/source-disposition.json"] = string(trigger) + "\n"
	}
	if strings.Contains(c.ID, "drift") {
		files["docs/reports/dogfood/doctrine-drift.md"] = "# Doctrine Drift\n\nGenerated guidance differs from source doctrine.\n"
	}
	if strings.Contains(c.ID, "blocker") {
		files["docs/reports/release/release-blocked.md"] = "# Release Blocked\n\nAsset verification is unavailable.\n"
	}
	return seedFiles(call, files)
}

func htmlDoc(title string) string {
	return fmt.Sprintf("<!-- MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] -->\n<!doctype html><title>%s</title><main id=\"app\">Ready</main>\n", title)
}

func phaserIndexDoc() string {
	return "<!-- MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] -->\n<!doctype html><title>Phaser Smoke</title><main id=\"game\"></main><script type=\"module\" src=\"/src/main.js\"></script>\n"
}

func reactIndexDoc() string {
	return "<!-- MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] -->\n<!doctype html><title>React Smoke</title><main id=\"root\"></main><script type=\"module\" src=\"/src/main.jsx\"></script>\n"
}

func cssDoc(body string) string {
	return "/* MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] */\n" + body
}

func jsDoc(body string) string {
	return "/* MarsDocSync: [\"docs/features/F-001-product-walking-skeleton.md\"] */\n" + body
}

func goDoc(body string) string {
	return "/*\nMarsDocSync:\ndocs:\n- docs/features/F-001-product-walking-skeleton.md\n*/\n" + body
}

func slugify(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
