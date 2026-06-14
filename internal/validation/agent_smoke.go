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

	"github.com/greaveselliott/mars-harness/internal/codeintel"
	"github.com/greaveselliott/mars-harness/internal/config"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/serve"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/greaveselliott/mars-harness/internal/ui"
	"gopkg.in/yaml.v3"
)

const (
	AgentSmokeSuiteFast    = "fast"
	AgentSmokeSuiteDefault = "default"
	AgentSmokeSuiteFull    = "full"
	AgentSmokeSuiteHeldOut = "held-out"

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
		opts.MaxTurns = 6
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
			return fmt.Sprintf("local Mars Harness inference router; single local server tier %s", opts.SingleTier)
		}
		return "local Mars Harness inference router"
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
	if err := assertAgentSmokeCase(targetDir, c); err != nil {
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
	if err := assertAgentSmokeCase(targetDir, c); err != nil {
		result.FailureClass = FailureRoleBehavior
		result.Error = err.Error()
		_ = writeAgentSmokeRunArtifacts(result)
		if opts.DiscardFailed {
			result.Discarded = discardRunDir(runDir) == nil
		}
		return result
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
		ID:      jobID,
		RepoID:  result.TargetPath,
		Role:    c.Role,
		Trigger: string(trigger),
	}
	if err := exec.Execute(ctx, job); err != nil {
		disposition, dErr := orgStore.GetDisposition(ctx, job.ID)
		if dErr == nil && disposition != nil {
			recordAgentSmokeDisposition(result, disposition)
		}
		return err
	}
	disposition, err := orgStore.GetDisposition(ctx, job.ID)
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
	return nil
}

func agentSmokeTrigger(c AgentSmokeCase) map[string]any {
	trigger := map[string]any{
		"type":                        "agent_smoke",
		"case":                        c.ID,
		"role_under_test":             c.Role,
		"project_type":                c.ProjectType,
		"stage":                       c.Stage,
		"expected_disposition":        c.ExpectedDisposition,
		"would_dispatch":              c.WouldDispatch,
		"suppress_follow_on_dispatch": true,
		"fixture_source":              "docs/validation/agent-smoke/matrix.yaml",
	}
	for k, v := range c.Trigger {
		trigger[k] = v
	}
	return trigger
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
	provenance = append(provenance, "scanner.EnsureHarness")
	if _, err := scanner.EnsureHarness(targetDir, false); err != nil {
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

func moveTicket(ctx context.Context, call func(string, any) error, src, dst string) error {
	_ = ctx
	return call("shell_exec", map[string]any{"argv": []string{"git", "mv", src, dst}})
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

func assertAgentSmokeCase(targetDir string, c AgentSmokeCase) error {
	for _, rel := range c.RequiredArtifacts {
		if strings.TrimSpace(rel) == "" {
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
	case strings.Contains(msg, "disposition") || strings.Contains(msg, "dispatch") || strings.Contains(msg, "role ") && strings.Contains(msg, "not found"):
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
	return strings.Contains(stage, "after-engineer") || strings.Contains(stage, "after-qa") || strings.Contains(stage, "ready") || strings.Contains(stage, "defect") || strings.Contains(stage, "stale")
}

func specForCase(c AgentSmokeCase) string {
	return fmt.Sprintf("# %s\n\nProject type: %s\nRole under smoke: %s\nStage: %s\n\nBuild a small representative target for compartmentalised Mars Harness role smoke validation.\n", c.ID, c.ProjectType, c.Role, c.Stage)
}

func readmeForCase(c AgentSmokeCase) string {
	return fmt.Sprintf("# %s\n\nEphemeral %s project generated for `%s` agent smoke validation.\n", c.ID, c.ProjectType, c.Role)
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
			"src/App.jsx":       jsDoc("export default function App() { return <main>Agent smoke</main>; }\n"),
			"src/App.test.jsx":  jsDoc("import { describe, it, expect } from 'vitest'; describe('smoke', () => { it('works', () => expect(true).toBe(true)); });\n"),
			"src/main.jsx":      jsDoc("import App from './App.jsx'; void App;\n"),
		})
	case "browser-game-phaser":
		return seedFiles(call, map[string]string{
			"package.json":       `{"scripts":{"build":"vite build"},"dependencies":{"phaser":"latest","vite":"latest"},"devDependencies":{}}` + "\n",
			"src/main.js":        jsDoc("import Phaser from 'phaser'; import { SmokeScene } from './game/scene.js'; new Phaser.Game({ type: Phaser.HEADLESS, scene: [SmokeScene] });\n"),
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
	if strings.Contains(c.Stage, "after-engineer") || strings.Contains(c.Stage, "after-qa") || strings.Contains(c.Stage, "ready") {
		reports["docs/reports/qa/"+c.ID+".md"] = "# QA Evidence\n\nValidation evidence seeded for agent smoke.\n"
	}
	if strings.Contains(c.Stage, "after-qa") || strings.Contains(c.Stage, "ready") {
		reports["docs/reports/security/"+c.ID+".md"] = "# Security Evidence\n\nSecurity approval seeded for agent smoke.\n"
	}
	if strings.Contains(c.Stage, "ready") {
		reports["docs/reports/dogfood/"+c.ID+".md"] = "# Dogfood Evidence\n\nDogfood validation seeded for agent smoke.\n"
		reports["CHANGELOG.md"] = "# Changelog\n\n## Unreleased\n\n- Agent smoke seeded change.\n"
		reports["VERSION"] = "0.0.0\n"
	}
	return seedFiles(call, reports)
}

func seedSpecialState(call func(string, any) error, c AgentSmokeCase) error {
	files := map[string]string{}
	if strings.Contains(c.Stage, "failure") || strings.Contains(c.Stage, "blocked") || strings.Contains(c.Stage, "stale") ||
		strings.Contains(c.ID, "failure") || strings.Contains(c.ID, "blocked") || strings.Contains(c.ID, "stale") {
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
