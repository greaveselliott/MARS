package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"

	"github.com/greaveselliott/mars-harness/internal/agent"
	"github.com/greaveselliott/mars-harness/internal/buildinfo"
	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/config"
	ctx "github.com/greaveselliott/mars-harness/internal/context"
	"github.com/greaveselliott/mars-harness/internal/doctor"
	"github.com/greaveselliott/mars-harness/internal/guardrails"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/models"
	"github.com/greaveselliott/mars-harness/internal/release"
	"github.com/greaveselliott/mars-harness/internal/safety"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/selfupdate"
	"github.com/greaveselliott/mars-harness/internal/serve"
	"github.com/greaveselliott/mars-harness/internal/setup"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/greaveselliott/mars-harness/internal/ui"
	"github.com/greaveselliott/mars-harness/internal/updatecheck"
)

var version = buildinfo.DefaultVersion

var (
	commit = "unknown"
	date   = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:           "mars-harness",
		Short:         "Autonomous AI delivery system",
		Long:          "Mars Harness — self-hosted autonomous AI delivery. Run setup to get started.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(versionCmd())
	root.AddCommand(updateCmd())
	root.AddCommand(startCmd())
	root.AddCommand(runCmd())
	root.AddCommand(setupCmd())
	root.AddCommand(initCmd())
	root.AddCommand(upgradeCmd())
	root.AddCommand(scanCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(registerCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(scoresCmd())
	root.AddCommand(trustCmd())
	root.AddCommand(modelsCmd())
	root.AddCommand(releaseCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Inspect and evaluate model candidates",
	}
	cmd.AddCommand(modelsEvaluateCmd())
	return cmd
}

func modelsEvaluateCmd() *cobra.Command {
	var (
		endpoint string
		model    string
		apiKey   string
		timeout  time.Duration
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "evaluate",
		Short: "Evaluate or plan model-candidate benchmarks",
		Long: `Evaluate a model through an OpenAI-compatible endpoint.

With --endpoint and --model, this runs the mechanical benchmark pack used to
screen model candidates before registry promotion. Without those flags, it
prints the current refresh plan, candidate shortlist, benchmark cases, and
promotion rules. Defaults are not changed by this command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if endpoint == "" || model == "" {
				plan := models.DefaultPlan(time.Now())
				if jsonOut {
					return writeJSON(os.Stdout, plan)
				}
				printModelEvaluationPlan(plan)
				return nil
			}

			report, err := models.Evaluate(cmd.Context(), models.Config{
				Endpoint: endpoint,
				Model:    model,
				APIKey:   apiKey,
				Timeout:  timeout,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, report)
			}
			printModelEvaluationReport(report)
			return nil
		},
	}
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "OpenAI-compatible base URL to evaluate")
	cmd.Flags().StringVar(&model, "model", "", "Model name to evaluate")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Optional API key for the endpoint")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Per-request timeout")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func writeJSON(w *os.File, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printModelEvaluationPlan(plan models.Plan) {
	fmt.Println("Model evaluation plan")
	fmt.Printf("Generated: %s\n\n", plan.GeneratedAt.Format(time.RFC3339))

	fmt.Println("Current medium-profile defaults:")
	for tier, spec := range plan.CurrentDefaults {
		fmt.Printf("  %s: %s %s %s context=%d repo=%s revision=%s\n",
			tier, spec.Name, spec.Params, spec.Quant, spec.ContextLen, spec.Repo, spec.Revision)
	}

	fmt.Println("\nCandidate shortlist:")
	for _, c := range plan.Candidates {
		mode := "local"
		if c.Cloud && !c.Local {
			mode = "cloud"
		} else if c.Cloud && c.Local {
			mode = "local/cloud"
		}
		mem := ""
		if c.MinMemoryGB > 0 {
			mem = fmt.Sprintf(" min_memory=%dGB", c.MinMemoryGB)
		}
		fmt.Printf("  - %s (%s,%s%s): %s\n", c.Name, c.Role, mode, mem, c.Why)
		fmt.Printf("    source: %s\n", c.Source)
	}

	fmt.Println("\nBenchmark cases:")
	for _, c := range plan.BenchmarkCases {
		fmt.Printf("  - %s [%s]: %s\n", c.Name, c.Kind, c.Description)
	}

	fmt.Println("\nPromotion rules:")
	for _, rule := range plan.PromotionRules {
		fmt.Printf("  - %s\n", rule)
	}
	fmt.Println("\nRun live evaluation with: mars-harness models evaluate --endpoint <url> --model <name>")
}

func printModelEvaluationReport(report models.Report) {
	fmt.Printf("Model evaluation: %s\n", report.Model)
	fmt.Printf("Endpoint: %s\n", report.Endpoint)
	fmt.Printf("Wall time: %s\n", report.Summary.WallTime.Round(time.Millisecond))
	if report.Summary.TokensPerSec > 0 {
		fmt.Printf("Tokens/sec: %.2f\n", report.Summary.TokensPerSec)
	}
	fmt.Printf("Passed: %d/%d\n\n", report.Summary.Passed, report.Summary.Total)
	for _, c := range report.Cases {
		status := "FAIL"
		if c.Passed {
			status = "PASS"
		}
		fmt.Printf("  %s %s %s", status, c.Name, c.Duration.Round(time.Millisecond))
		if c.TotalTokens > 0 {
			fmt.Printf(" tokens=%d", c.TotalTokens)
		}
		fmt.Println()
		if c.Error != "" {
			fmt.Printf("    error: %s\n", c.Error)
		}
		if c.Detail != "" {
			fmt.Printf("    detail: %s\n", c.Detail)
		}
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, OS, and architecture",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mars-harness %s %s/%s commit=%s built=%s\n", version, runtime.GOOS, runtime.GOARCH, commit, date)
		},
	}
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the CLI binary or a deployed target harness",
		Long: `Update Mars Harness surfaces with one verb.

Use "mars-harness update tool" to reinstall or upgrade the installed CLI.
Use "mars-harness update harness --repo <path>" to update the .harness/
bundle deployed into a target repository.`,
	}
	cmd.AddCommand(updateCheckCmd())
	cmd.AddCommand(updateToolCmd())
	cmd.AddCommand(updateHarnessCmd())
	return cmd
}

func updateCheckCmd() *cobra.Command {
	var (
		repoPath         string
		latestReleaseURL string
		skipRemote       bool
		jsonOut          bool
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check whether the CLI or deployed target harness is behind",
		Long: `Check version drift for the installed mars-harness tool and a target repo's
deployed .harness/ metadata. Remote release failures are reported as unknown so
local target checks still complete.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("update check: cannot determine working directory: %w", err)
				}
			}
			report, err := updatecheck.Run(cmd.Context(), updatecheck.Config{
				CurrentVersion:   version,
				RepoPath:         repoPath,
				LatestReleaseURL: latestReleaseURL,
				SkipRemote:       skipRemote,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, report)
			}
			printUpdateCheck(report)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to target repository (default: current directory)")
	cmd.Flags().StringVar(&latestReleaseURL, "latest-release-url", selfupdate.DefaultLatestReleaseURL, "GitHub-compatible latest release endpoint")
	cmd.Flags().BoolVar(&skipRemote, "skip-remote", false, "Skip remote latest-release check")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func updateToolCmd() *cobra.Command {
	var (
		updateVersion string
		installDir    string
		dryRun        bool
		jsonOut       bool
	)
	cmd := &cobra.Command{
		Use:     "tool",
		Aliases: []string{"binary", "cli"},
		Short:   "Reinstall or upgrade the mars-harness command",
		Long: `Reinstall the mars-harness command without changing directories.

This source-based updater uses go install and writes into the directory that
contains the currently running mars-harness binary by default. It is intended
for source-development installs until release-asset based updates are complete.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := selfupdate.Config{
				Version:    updateVersion,
				InstallDir: installDir,
				DryRun:     dryRun,
			}
			plan, err := selfupdate.Run(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, plan)
			}
			fmt.Printf("mars-harness update tool\n")
			fmt.Printf("Version: %s\n", plan.Version)
			fmt.Printf("Install dir: %s\n", plan.InstallDir)
			fmt.Printf("Command: GOBIN=%s %s\n", plan.InstallDir, strings.Join(plan.Command, " "))
			if dryRun {
				fmt.Printf("Dry run: no changes made\n")
				return nil
			}
			fmt.Printf("Installed: %s\n", plan.BinaryPath)
			fmt.Printf("Run: mars-harness version\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&updateVersion, "version", selfupdate.DefaultVersion, "Go module version to install, e.g. latest, main, or v0.5.3")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Install directory; default is the current mars-harness binary directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the update plan without running go install")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func updateHarnessCmd() *cobra.Command {
	var repoPath string
	cmd := &cobra.Command{
		Use:     "harness",
		Aliases: []string{"target", "bundle"},
		Short:   "Update a deployed target .harness/ bundle",
		Long:    "Fill missing defaults in a target project's .harness/ without overwriting user-owned agent configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarnessUpgrade(repoPath, "update harness")
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	return cmd
}

func printUpdateCheck(report updatecheck.Report) {
	fmt.Println("Mars Harness Update Check")
	for _, component := range []updatecheck.Component{report.Tool, report.Harness} {
		fmt.Printf("  %s: %s", component.Name, component.Status)
		if component.CurrentVersion != "" {
			fmt.Printf(" current=%s", component.CurrentVersion)
		}
		if component.LatestVersion != "" {
			fmt.Printf(" latest=%s", component.LatestVersion)
		}
		fmt.Printf(" — %s\n", component.Message)
		if component.Command != "" {
			fmt.Printf("    run: %s\n", component.Command)
		}
	}
	if len(report.Actions) == 0 {
		fmt.Println("  actions: none")
		return
	}
	fmt.Println("  actions:")
	for _, action := range report.Actions {
		fmt.Printf("    %s\n", action)
	}
}

func releaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage semantic versions and patch notes",
	}
	cmd.AddCommand(releaseNotesCmd())
	return cmd
}

func releaseNotesCmd() *cobra.Command {
	var (
		repoPath string
		bump     string
		dryRun   bool
	)
	cmd := &cobra.Command{
		Use:   "notes",
		Short: "Generate semantic-versioned patch notes",
		Long: `Generate patch notes from semantic commits on main.

The command reads VERSION, finds commits since the latest release marker or
version tag, chooses a semantic version bump, updates VERSION, and prepends a
CHANGELOG.md entry. Use --dry-run to preview without writing files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("release notes: cannot determine working directory: %w", err)
				}
			}
			result, err := release.Prepare(cmd.Context(), release.Config{
				RepoRoot: repoPath,
				Bump:     release.Bump(bump),
				DryRun:   dryRun,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Release notes: %s -> %s (%s, %d commits)\n",
				result.PreviousVersion, result.NextVersion, result.Bump, len(result.Commits))
			if result.BaseRef != "" {
				fmt.Printf("Base: %s\n", result.BaseRef)
			}
			if dryRun {
				fmt.Println()
				fmt.Print(result.Entry)
				return nil
			}
			fmt.Println("Updated files:")
			for _, path := range result.UpdatedFiles {
				fmt.Printf("  %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().StringVar(&bump, "bump", string(release.BumpAuto), "Version bump: auto, major, minor, or patch")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview release notes without writing files")
	return cmd
}

func runCmd() *cobra.Command {
	var (
		repoPath      string
		modelEndpoint string
		traceFlag     bool
		dryRun        bool
		budget        int
		maxTurns      int
	)

	cmd := &cobra.Command{
		Use:   "run <role>",
		Short: "Run an agent role against a repository",
		Long: `Load the .harness/ bundle from --repo and execute the named role.

If .harness/manifest.yaml is missing, the same scaffold as 'mars-harness init'
is applied automatically (requires a git repository).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]
			return executeRun(runOpts{
				roleName:      roleName,
				repoPath:      repoPath,
				modelEndpoint: modelEndpoint,
				trace:         traceFlag,
				dryRun:        dryRun,
				budget:        budget,
				maxTurns:      maxTurns,
			})
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (required)")
	cmd.Flags().StringVar(&modelEndpoint, "model-endpoint", "", "Override LLM endpoint (e.g. http://127.0.0.1:8080)")
	cmd.Flags().BoolVar(&traceFlag, "trace", false, "Enable verbose execution trace output")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print assembled system prompt and exit without calling the LLM")
	cmd.Flags().IntVar(&budget, "budget", 0, "Token budget (0 = unlimited)")
	cmd.Flags().IntVar(&maxTurns, "max-turns", 50, "Maximum LLM round-trips")
	_ = cmd.MarkFlagRequired("repo")

	return cmd
}

type runOpts struct {
	roleName      string
	repoPath      string
	modelEndpoint string
	trace         bool
	dryRun        bool
	budget        int
	maxTurns      int
}

func executeRun(opts runOpts) error {
	tw := ui.NewTraceWriter(os.Stdout, false, false)

	absRepo, err := filepath.Abs(opts.repoPath)
	if err != nil {
		tw.WriteError(err.Error())
		return fmt.Errorf("run: resolve repo path: %w", err)
	}

	didInit, err := scanner.EnsureHarness(absRepo, false)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}
	if didInit {
		tw.WriteAssistant("Auto-initialised .harness/ with default pipeline — continuing.")
	}

	manifest, err := bundle.Load(absRepo)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}

	role, ok := manifest.Roles[opts.roleName]
	if !ok {
		msg := fmt.Sprintf("role %q not found in manifest; check .harness/manifest.yaml", opts.roleName)
		tw.WriteError(msg)
		return fmt.Errorf(msg)
	}

	tw.WriteHeader(opts.roleName, role.Model, role.Tools, role.Then)

	rolePrompt, err := manifest.RolePrompt(absRepo, opts.roleName)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}

	guardRules, err := manifest.LoadGuardrails(absRepo, opts.roleName)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}
	guardEngine, err := guardrails.New(guardRules)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}
	var promptGuardrails []ctx.Guardrail
	for _, r := range guardRules {
		body := r.Message
		if body == "" {
			body = r.Pattern
		}
		promptGuardrails = append(promptGuardrails, ctx.Guardrail{Scope: r.Scope, Title: r.Name, Body: body})
	}
	knowledgeDefs, err := manifest.LoadKnowledgeRoutes(absRepo, opts.roleName)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}
	var knowledgeRoutes []ctx.KnowledgeRoute
	for _, kr := range knowledgeDefs {
		knowledgeRoutes = append(knowledgeRoutes, ctx.KnowledgeRoute{When: kr.When, Paths: kr.Paths})
	}
	skillDefs, err := bundle.LoadSkills(absRepo, opts.roleName)
	if err != nil {
		tw.WriteError(fmt.Sprintf("load skills: %v", err))
		return err
	}
	var skills []ctx.Skill
	for _, sd := range skillDefs {
		skills = append(skills, ctx.Skill{Name: sd.Name, Scope: sd.Scope, Body: sd.Body})
	}

	assemblyInput := ctx.Input{
		RoleScope:       opts.roleName,
		RolePrompt:      rolePrompt,
		Guardrails:      promptGuardrails,
		KnowledgeRoutes: knowledgeRoutes,
		Skills:          skills,
	}
	if opts.budget > 0 {
		assemblyInput.TokenBudget = opts.budget
	}

	systemPrompt, stats, err := ctx.Assemble(assemblyInput)
	if err != nil {
		tw.WriteError(fmt.Sprintf("context assembly failed: %v", err))
		return err
	}

	if opts.trace {
		slog.Info("context assembled",
			"role", opts.roleName,
			"sections", len(stats),
		)
		for _, s := range stats {
			slog.Info("context section", "name", s.Name, "tokens", s.Tokens)
		}
	}

	if opts.dryRun {
		fmt.Println("── dry-run: assembled system prompt ──")
		fmt.Println(systemPrompt)
		fmt.Println("── end system prompt ──")
		fmt.Printf("\nRole: %s | Tools: %v\n", opts.roleName, role.Tools)
		return nil
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	endpoint := opts.modelEndpoint
	var router *inference.Router

	if endpoint == "" {
		router, endpoint, err = autoStartInference(sigCtx, opts.roleName, role.Model)
		if err != nil {
			tw.WriteError(fmt.Sprintf("inference startup failed: %v", err))
			return err
		}
		defer router.StopAll()
	}

	tw.WriteReady()

	client, err := llm.NewClient(llm.Config{
		BaseURL: endpoint,
		Model:   role.Model,
	})
	if err != nil {
		tw.WriteError(fmt.Sprintf("failed to create LLM client: %v", err))
		return err
	}

	registry, err := tools.DefaultRegistry()
	if err != nil {
		tw.WriteError(fmt.Sprintf("failed to load tool registry: %v", err))
		return err
	}
	executor := tools.NewExecutor(registry)
	executor.Session = &tools.Session{
		Role:         opts.roleName,
		JobID:        fmt.Sprintf("%s-%s", manifest.Name, opts.roleName),
		RepoID:       absRepo,
		TrustLevel:   string(trust.LevelContributor),
		Guardrails:   guardEngine,
		SafetyLimits: safety.DefaultLimits(),
	}

	root, err := tools.NewRoot(absRepo)
	if err != nil {
		tw.WriteError(fmt.Sprintf("invalid repo root: %v", err))
		return err
	}

	recorder := trace.NewRecorder(nil)

	result, err := agent.Run(sigCtx, agent.Params{
		Completer:    client,
		Registry:     registry,
		Executor:     executor,
		Root:         root,
		Allowlist:    role.Tools,
		SystemPrompt: systemPrompt,
		UserMessage:  "Begin your task. Inspect the repository and take action.",
		Config: agent.LoopConfig{
			Model:       role.Model,
			MaxTurns:    opts.maxTurns,
			TokenBudget: opts.budget,
		},
		JobID: fmt.Sprintf("%s-%s", manifest.Name, opts.roleName),
		Trace: recorder,
		UI:    tw,
	})
	if err != nil {
		tw.WriteError(fmt.Sprintf("agent error: %v", err))
		return err
	}

	tw.WriteSummary(
		opts.roleName,
		string(result.EndReason),
		result.LLMCalls,
		result.ToolInvocations,
		result.WallTime,
		result.TokenEstimate,
	)

	if result.Err != nil {
		tw.WriteError(fmt.Sprintf("run ended with error: %v", result.Err))
	}
	if err := agent.NonSuccessError(result); err != nil {
		tw.WriteError(err.Error())
		return fmt.Errorf("run: %w", err)
	}

	tw.WriteHandoff(opts.roleName, role.Then)

	return nil
}

// autoStartInference loads config, detects hardware, and starts llama-server
// for the requested role via the inference Router. Returns the router (for
// cleanup) and the base URL of the running server.
func autoStartInference(ctx context.Context, roleName, modelHint string) (*inference.Router, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	baseDir := filepath.Join(home, ".mars-harness")

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		slog.Warn("config load failed, using defaults", "err", err)
	}

	modelsDir := filepath.Join(baseDir, "models")
	if cfg.ModelsDir != "" {
		modelsDir = cfg.ModelsDir
	}
	binDir := filepath.Join(baseDir, "bin")
	if cfg.BinDir != "" {
		binDir = cfg.BinDir
	}

	binaryPath := filepath.Join(binDir, "llama-server")
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, "", fmt.Errorf("llama-server not found at %s — run 'mars-harness setup' first", binaryPath)
	}

	hw := hardware.Detect()
	modelSet := hardware.DefaultModelsForHardware(hw, cfg.PerformanceProfile)

	router := inference.NewRouter(inference.RouterConfig{
		BinaryPath:  binaryPath,
		Models:      modelSet,
		RoleMapping: inference.DefaultRoleTierMapping(),
		ModelsDir:   modelsDir,
		Tuning:      inferenceTuningFromConfig(cfg),
	})

	endpoint, err := router.ServerForRoleModel(ctx, roleName, modelHint)
	if err != nil {
		return nil, "", err
	}

	slog.Info("inference ready", "role", roleName, "endpoint", endpoint)
	return router, endpoint, nil
}

func inferenceTuningFromConfig(cfg config.Config) inference.ServerTuning {
	return inference.ServerTuning{
		Threads:        cfg.LlamaThreads,
		ThreadsBatch:   cfg.LlamaThreadsBatch,
		Parallel:       cfg.LlamaParallel,
		BatchSize:      cfg.LlamaBatchSize,
		UBatchSize:     cfg.LlamaUBatchSize,
		FlashAttention: cfg.LlamaFlashAttention,
		MLock:          cfg.LlamaMLock,
	}
}

func setupCmd() *cobra.Command {
	var (
		skipDownload bool
		enableGitHub bool
		testMode     bool
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "First-time setup wizard",
		Long:  "Create ~/.mars-harness/, detect hardware, install local inference, and download pinned models. GitHub integration is optional.",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := setup.Run(setup.Config{
				SkipDownload: skipDownload,
				EnableGitHub: enableGitHub,
				TestMode:     testMode,
				DryRun:       dryRun,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Setup complete: %d steps run, %d skipped\n", result.StepsRun, result.StepsSkipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&skipDownload, "skip-download", false, "Skip model download")
	cmd.Flags().BoolVar(&enableGitHub, "github", false, "Configure optional GitHub status/check integration")
	cmd.Flags().BoolVar(&testMode, "test-mode", false, "Skip downloads and external services")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print steps without executing")

	return cmd
}

func initCmd() *cobra.Command {
	var (
		repoPath string
		force    bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold .harness/ in a repository",
		Long:  "Create the .harness/ directory with manifest.yaml, roles/, guardrails/, and knowledge/ subdirectories.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("init: cannot determine working directory: %w", err)
				}
			}
			if err := scanner.Init(repoPath, force); err != nil {
				return err
			}
			fmt.Printf("Initialized .harness/ in %s\n", repoPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "Refresh missing scaffold and rewrite manifest if .harness/ already exists")

	return cmd
}

func upgradeCmd() *cobra.Command {
	var repoPath string

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Fill missing .harness/ defaults without overwriting user agents",
		Long: `Fill missing default files in an existing target project's .harness/.
Existing manifest.yaml, role prompts, knowledge routes, guardrails, target
AGENTS.md, tickets, exec-plans, design-docs, and references are user-owned and
preserved.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarnessUpgrade(repoPath, "upgrade")
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	return cmd
}

func runHarnessUpgrade(repoPath, verb string) error {
	if repoPath == "" {
		var err error
		repoPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("%s: cannot determine working directory: %w", verb, err)
		}
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("%s: resolve path: %w", verb, err)
	}

	updated, err := scanner.Upgrade(absPath)
	if err != nil {
		return err
	}

	fmt.Printf("Updated .harness/ in %s (%d files updated)\n", absPath, len(updated))
	for _, f := range updated {
		fmt.Printf("  %s\n", f)
	}
	return nil
}

func scanCmd() *cobra.Command {
	var (
		repoPath   string
		genTickets bool
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a repository for gaps and generate starter tickets",
		Long: `Walk the file tree to find missing tests, TODOs, missing CI, and large functions. Optionally generate docs/tickets/backlog entries.

If .harness/manifest.yaml is missing, mars-harness scaffolds it first (same as init; requires git).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("scan: cannot determine working directory: %w", err)
				}
			}

			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("scan: resolve path: %w", err)
			}
			if _, err := scanner.EnsureHarness(absPath, false); err != nil {
				return fmt.Errorf("scan: %w", err)
			}

			sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			result, err := scanner.Scan(sigCtx, scanner.Config{RepoRoot: absPath})
			if err != nil {
				return err
			}

			fmt.Printf("Language: %s | Framework: %s\n", result.Language, result.Framework)
			fmt.Printf("CI: %v | Tests: %v | README: %v | LICENSE: %v\n",
				result.HasCI, result.HasTests, result.HasReadme, result.HasLicense)
			fmt.Printf("Findings: %d\n", len(result.Findings))

			for _, f := range result.Findings {
				fmt.Printf("  [%s] %s: %s\n", f.Severity, f.Type, f.Description)
			}

			if genTickets && len(result.Findings) > 0 {
				if err := scanner.GenerateTickets(result.Findings, absPath); err != nil {
					return err
				}
				fmt.Printf("Tickets written to %s\n", filepath.Join(absPath, "docs", "tickets", "backlog"))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().BoolVar(&genTickets, "tickets", false, "Generate ticket files in docs/tickets/backlog/")

	return cmd
}

func doctorCmd() *cobra.Command {
	var (
		configPath string
		dbPath     string
		repoPath   string
		skipRemote bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Health check for GPU, models, and config",
		Long:  "Run diagnostic checks on Go version, config, models, database, llama-server, and disk space.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" && repoPath != "" {
				absPath, err := filepath.Abs(repoPath)
				if err == nil {
					dbPath = defaultDBPath(absPath)
				}
			}
			results := doctor.Run(doctor.Config{
				ConfigPath:     configPath,
				DBPath:         dbPath,
				RepoPath:       repoPath,
				CurrentVersion: version,
				SkipRemote:     skipRemote,
				JSONOutput:     jsonOutput,
			})

			if jsonOutput {
				out, err := doctor.FormatJSON(results)
				if err != nil {
					return err
				}
				fmt.Println(out)
			} else {
				fmt.Println("Mars Harness Doctor")
				fmt.Println("───────────────────")
				fmt.Print(doctor.FormatText(results))
			}

			if doctor.HasFailures(results) {
				return fmt.Errorf("doctor: one or more checks failed — see above for remediation")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.mars-harness/config.yaml)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to database file (default: ~/.mars-harness/db/{repo}/mars.db)")
	cmd.Flags().StringVar(&repoPath, "repo", "", "Target repository — used to locate the per-repo database")
	cmd.Flags().BoolVar(&skipRemote, "skip-remote", false, "Skip remote connectivity checks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")

	return cmd
}

func scoresCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "scores",
		Short: "Show trunk-native accuracy scores",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _, err := resolveRepoDBAndID(repoPath, dbPath)
			if err != nil {
				return err
			}
			store, err := scoring.OpenStore(path)
			if err != nil {
				return err
			}
			defer store.Close()
			scores, err := store.ListScores(cmd.Context())
			if err != nil {
				return err
			}
			if len(scores) == 0 {
				fmt.Println("No scores recorded yet.")
				return nil
			}
			fmt.Printf("%-36s %-18s %-8s %-7s %-8s %s\n", "REPO", "ROLE", "SCORE", "SAMPLES", "WINDOW", "COMPUTED")
			for _, sc := range scores {
				fmt.Printf("%-36s %-18s %-8.2f %-7d %-8dd %s\n",
					sc.RepoID, sc.Role, sc.Value, sc.SampleSize, sc.WindowDays, sc.ComputedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Target repository path (default: shared legacy database)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	return cmd
}

func trustCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Show and set progressive autonomy levels",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _, err := resolveRepoDBAndID(repoPath, dbPath)
			if err != nil {
				return err
			}
			store, err := trust.OpenStore(path)
			if err != nil {
				return err
			}
			defer store.Close()
			entries, err := store.List(cmd.Context())
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("No trust entries recorded yet.")
				return nil
			}
			fmt.Printf("%-36s %-18s %-12s %-7s %s\n", "REPO", "ROLE", "LEVEL", "TRIALS", "UPDATED")
			for _, e := range entries {
				fmt.Printf("%-36s %-18s %-12s %-7d %s\n",
					e.RepoID, e.Role, e.Level, e.TrialRuns, e.UpdatedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Target repository path (default: shared legacy database)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	cmd.AddCommand(trustSetCmd())
	return cmd
}

func trustSetCmd() *cobra.Command {
	var reason string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "set <role> <repo> <observer|contributor|autonomous>",
		Short: "Set a role's trust level for a repository",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]
			level := trust.Level(strings.TrimSpace(args[2]))
			switch level {
			case trust.LevelObserver, trust.LevelContributor, trust.LevelAutonomous:
			default:
				return fmt.Errorf("trust set: invalid level %q", args[2])
			}
			if strings.TrimSpace(reason) == "" {
				return fmt.Errorf("trust set: --reason is required for auditability")
			}
			path, repoID, err := resolveRepoDBAndID(args[1], dbPath)
			if err != nil {
				return err
			}
			store, err := trust.OpenStore(path)
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.SetWithReason(cmd.Context(), roleName, repoID, level, reason); err != nil {
				return err
			}
			fmt.Printf("Set %s/%s to %s\n", repoID, roleName, level)
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for the trust override (required)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	return cmd
}

func serveCmd() *cobra.Command {
	var (
		webhookAddr string
		concurrency int
		dbPath      string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the autonomous pipeline server",
		Long:  "Run the orchestrator: receive webhooks, fire cron schedules, and execute agent roles.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				slog.Warn("config load failed, using defaults", "err", err)
			}

			if webhookAddr == "" {
				webhookAddr = fmt.Sprintf(":%d", cfg.WebhookPort)
			}

			if dbPath == "" {
				dbPath = legacyDBPath()
				slog.Info("serve: using shared DB — for per-repo isolation, use 'start --repo' or pass --db explicitly")
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("serve: create db directory: %w", err)
			}

			webhookSecret := os.Getenv("MARS_HARNESS_WEBHOOK_SECRET")
			dashboardAddr := fmt.Sprintf(":%d", cfg.DashboardPort)

			serve.Cleanup(cfg.WebhookPort, dbPath, cfg.DashboardPort)

			srv, err := serve.New(serve.Config{
				WebhookAddr:        webhookAddr,
				WebhookSecret:      webhookSecret,
				DBPath:             dbPath,
				Concurrency:        concurrency,
				ModelsDir:          cfg.ModelsDir,
				BinDir:             cfg.BinDir,
				DashboardAddr:      dashboardAddr,
				PerformanceProfile: cfg.PerformanceProfile,
				InferenceTuning:    inferenceTuningFromConfig(cfg),
			})
			if err != nil {
				return err
			}

			sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			sb := ui.NewStatusBar(os.Stderr, srv)
			sb.Start()
			defer sb.Stop()

			kl := ui.NewKeyListener(srv, stop, sb)
			kl.Start(sigCtx)
			defer kl.Stop()

			return srv.Start(sigCtx)
		},
	}

	cmd.Flags().StringVar(&webhookAddr, "addr", "", "Address to listen on (default from config webhook_port)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 2, "Number of concurrent agent workers")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/mars.db)")

	return cmd
}

func registerCmd() *cobra.Command {
	var (
		repoPath string
		remote   string
		branch   string
		dbPath   string
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a repository for autonomous management",
		Long: `Register a local repository so the orchestrator can manage it.

If .harness/manifest.yaml is missing, mars-harness runs the same scaffold as
'mars-harness init' automatically (requires a git repository).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("register: cannot determine working directory: %w", err)
				}
			}

			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("register: resolve path: %w", err)
			}

			didInit, err := scanner.EnsureHarness(absPath, false)
			if err != nil {
				return fmt.Errorf("register: %w", err)
			}
			if didInit {
				fmt.Fprintf(os.Stderr, "Auto-initialised .harness/ in %s\n", absPath)
			}

			if dbPath == "" {
				dbPath = defaultDBPath(absPath)
				if _, err := os.Stat(legacyDBPath()); err == nil {
					if _, err := os.Stat(dbPath); os.IsNotExist(err) {
						slog.Warn("register: legacy shared database exists at " + legacyDBPath() + " but per-repo DB does not yet exist — starting fresh. Copy the legacy DB to " + dbPath + " if you want to preserve history.")
					}
				}
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("register: create db directory: %w", err)
			}

			db, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			registry, err := serve.NewRepoRegistry(db)
			if err != nil {
				return err
			}

			if branch == "" {
				branch = "main"
			}

			id, err := registry.Register(context.Background(), absPath, remote, branch)
			if err != nil {
				return err
			}

			fmt.Printf("Registered repo %s\n  ID: %s\n  Path: %s\n  Remote: %s\n  Branch: %s\n",
				filepath.Base(absPath), id, absPath, remote, branch)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().StringVar(&remote, "remote", "", "GitHub owner/repo (e.g. myorg/myrepo)")
	cmd.Flags().StringVar(&branch, "branch", "main", "Default branch name")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")

	return cmd
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database at %s: %w — check path and permissions", path, err)
	}
	return db, nil
}

// defaultDBPath returns the per-repo database path: ~/.mars-harness/db/{repo-slug}/mars.db.
// Each repo gets its own SQLite file so queue, telemetry, and scheduling are isolated.
func defaultDBPath(repoAbsPath string) string {
	home, _ := os.UserHomeDir()
	repoSlug := filepath.Base(repoAbsPath)
	return filepath.Join(home, ".mars-harness", "db", repoSlug, "mars.db")
}

func resolveRepoDBAndID(repoArg, dbPath string) (string, string, error) {
	repoArg = strings.TrimSpace(repoArg)
	if dbPath != "" && repoArg == "" {
		return dbPath, "", nil
	}
	if repoArg == "" {
		return legacyDBPath(), "", nil
	}
	abs, err := filepath.Abs(repoArg)
	if err != nil {
		return "", "", fmt.Errorf("resolve repo path: %w", err)
	}
	if dbPath == "" {
		dbPath = defaultDBPath(abs)
	}
	repoID := repoArg
	db, err := openDB(dbPath)
	if err == nil {
		defer db.Close()
		reg, regErr := serve.NewRepoRegistry(db)
		if regErr == nil {
			repos, listErr := reg.List(context.Background())
			if listErr == nil {
				for _, rec := range repos {
					if rec.Path == abs {
						repoID = rec.ID
						break
					}
				}
			}
		}
	}
	return dbPath, repoID, nil
}

// legacyDBPath returns the old shared database path: ~/.mars-harness/db/mars.db.
func legacyDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mars-harness", "db", "mars.db")
}

func startCmd() *cobra.Command {
	var (
		repoPath    string
		concurrency int
		dbPath      string
		force       bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Bootstrap and run the full autonomous pipeline",
		Long: `Initialise .harness/ if needed, register the repo, seed the CEO agent,
and start the orchestrator. The CEO plans strategy, hands off to CTO,
then COO creates tickets, the engineer builds, QA reviews — the full chain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			tw := ui.NewTraceWriter(os.Stdout, false, false)

			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("start: cannot determine working directory: %w", err)
				}
			}
			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("start: resolve path: %w", err)
			}

			didInit, err := scanner.EnsureHarness(absPath, force)
			if err != nil {
				tw.WriteError(fmt.Sprintf("init failed: %v", err))
				return err
			}
			if didInit {
				tw.WriteAssistant("No .harness/ found — initialised with default pipeline...")
			}

			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				slog.Warn("config load failed, using defaults", "err", err)
			}

			if dbPath == "" {
				dbPath = defaultDBPath(absPath)
				if _, err := os.Stat(legacyDBPath()); err == nil {
					if _, err := os.Stat(dbPath); os.IsNotExist(err) {
						slog.Warn("start: legacy shared database exists at " + legacyDBPath() + " but per-repo DB does not yet exist — starting fresh. Copy the legacy DB to " + dbPath + " if you want to preserve history.")
					}
				}
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("start: create db directory: %w", err)
			}

			webhookAddr := fmt.Sprintf(":%d", cfg.WebhookPort)
			dashboardAddr := fmt.Sprintf(":%d", cfg.DashboardPort)

			serve.Cleanup(cfg.WebhookPort, dbPath, cfg.DashboardPort)

			srv, err := serve.New(serve.Config{
				WebhookAddr:        webhookAddr,
				DBPath:             dbPath,
				Concurrency:        concurrency,
				ModelsDir:          cfg.ModelsDir,
				BinDir:             cfg.BinDir,
				DashboardAddr:      dashboardAddr,
				RepoScope:          absPath,
				PerformanceProfile: cfg.PerformanceProfile,
				InferenceTuning:    inferenceTuningFromConfig(cfg),
			})
			if err != nil {
				tw.WriteError(fmt.Sprintf("orchestrator init: %v", err))
				return err
			}

			sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			repoID, err := srv.Repos().Register(sigCtx, absPath, "", "main")
			if err != nil {
				tw.WriteError(fmt.Sprintf("register: %v", err))
				return err
			}
			tw.WriteAssistant(fmt.Sprintf("Registered repo %s (ID: %s)", filepath.Base(absPath), repoID))

			triggerJSON := `{"type":"bootstrap","source":"mars-harness start"}`
			jobID, err := srv.SeedJob(sigCtx, repoID, "ceo", triggerJSON)
			if err != nil {
				tw.WriteError(fmt.Sprintf("seed CEO: %v", err))
				return err
			}
			tw.WriteAssistant(fmt.Sprintf("Seeded CEO agent (job %s) — pipeline will cascade: CEO → CTO → COO → Engineer → QA → Security → Deps", jobID))

			sb := ui.NewStatusBar(os.Stderr, srv)
			sb.Start()
			defer sb.Stop()

			kl := ui.NewKeyListener(srv, stop, sb)
			kl.Start(sigCtx)
			defer kl.Stop()

			return srv.Start(sigCtx)
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of concurrent agent workers (1 = sequential pipeline)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
	cmd.Flags().BoolVar(&force, "force", false, "Force re-init .harness/ even if it exists")

	return cmd
}

func placeholderCmd(name, description string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: description,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s: not yet implemented\n", name)
		},
	}
}
