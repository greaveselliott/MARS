package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"

	"github.com/greaveselliott/mars-harness/internal/agent"
	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/config"
	ctx "github.com/greaveselliott/mars-harness/internal/context"
	"github.com/greaveselliott/mars-harness/internal/doctor"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/serve"
	"github.com/greaveselliott/mars-harness/internal/setup"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/ui"
)

var version = "0.0.1-dev"

var (
	commit = "unknown"
	date   = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:   "mars-harness",
		Short: "Autonomous AI delivery system",
		Long:  "Mars Harness — self-hosted autonomous AI delivery. Run setup to get started.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(versionCmd())
	root.AddCommand(startCmd())
	root.AddCommand(runCmd())
	root.AddCommand(setupCmd())
	root.AddCommand(initCmd())
	root.AddCommand(scanCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(registerCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(placeholderCmd("scores", "Accuracy and value scoring dashboard"))
	root.AddCommand(placeholderCmd("trust", "Progressive autonomy level management"))

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		Long:  "Load the .harness/ bundle from --repo and execute the named role.",
		Args:  cobra.ExactArgs(1),
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

	manifest, err := bundle.Load(opts.repoPath)
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

	rolePrompt, err := manifest.RolePrompt(opts.repoPath, opts.roleName)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}

	assemblyInput := ctx.Input{
		RoleScope:  opts.roleName,
		RolePrompt: rolePrompt,
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
		router, endpoint, err = autoStartInference(sigCtx, opts.roleName)
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

	root, err := tools.NewRoot(opts.repoPath)
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

	tw.WriteHandoff(opts.roleName, role.Then)

	return nil
}

// autoStartInference loads config, detects hardware, and starts llama-server
// for the requested role via the inference Router. Returns the router (for
// cleanup) and the base URL of the running server.
func autoStartInference(ctx context.Context, roleName string) (*inference.Router, string, error) {
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

	binaryPath := filepath.Join(baseDir, "bin", "llama-server")
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, "", fmt.Errorf("llama-server not found at %s — run 'mars-harness setup' first", binaryPath)
	}

	hw := hardware.Detect()
	modelSet := hardware.DefaultModels(hw.Profile)

	roleMapping := map[string]hardware.Tier{
		"engineer":       hardware.TierCoding,
		"pipeline-fixer": hardware.TierCoding,
		"reviewer":       hardware.TierReasoning,
		"qa":             hardware.TierCoding,
		"documenter":     hardware.TierFast,
		"release":        hardware.TierFast,
		"triager":        hardware.TierFast,
		"onboarder":      hardware.TierFast,
		"auditor":        hardware.TierReasoning,
		"backlog":        hardware.TierFast,
		"evolution":      hardware.TierReasoning,
	}

	router := inference.NewRouter(inference.RouterConfig{
		BinaryPath:  binaryPath,
		Models:      modelSet,
		RoleMapping: roleMapping,
		ModelsDir:   modelsDir,
	})

	endpoint, err := router.ServerForRole(ctx, roleName)
	if err != nil {
		return nil, "", err
	}

	slog.Info("inference ready", "role", roleName, "endpoint", endpoint)
	return router, endpoint, nil
}

func setupCmd() *cobra.Command {
	var (
		skipDownload bool
		skipGitHub   bool
		testMode     bool
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "First-time setup wizard",
		Long:  "Create ~/.mars-harness/, detect hardware, download models, and configure GitHub integration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := setup.Run(setup.Config{
				SkipDownload: skipDownload,
				SkipGitHub:   skipGitHub,
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
	cmd.Flags().BoolVar(&skipGitHub, "skip-github", false, "Skip GitHub App configuration")
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
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing .harness/ directory")

	return cmd
}

func scanCmd() *cobra.Command {
	var (
		repoPath   string
		genTickets bool
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a repository for gaps and generate starter tickets",
		Long:  "Walk the file tree to find missing tests, TODOs, missing CI, and large functions. Optionally generate .harness/tickets/.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("scan: cannot determine working directory: %w", err)
				}
			}

			sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			result, err := scanner.Scan(sigCtx, scanner.Config{RepoRoot: repoPath})
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
				ticketDir := filepath.Join(repoPath, ".harness", "tickets")
				if err := scanner.GenerateTickets(result.Findings, ticketDir); err != nil {
					return err
				}
				fmt.Printf("Tickets written to %s\n", ticketDir)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().BoolVar(&genTickets, "tickets", false, "Generate ticket files in .harness/tickets/")

	return cmd
}

func doctorCmd() *cobra.Command {
	var (
		configPath string
		dbPath     string
		skipRemote bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Health check for GPU, models, and config",
		Long:  "Run diagnostic checks on Go version, config, models, database, llama-server, and disk space.",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := doctor.Run(doctor.Config{
				ConfigPath: configPath,
				DBPath:     dbPath,
				SkipRemote: skipRemote,
				JSONOutput: jsonOutput,
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to database file (default: ~/.mars-harness/db/mars.db)")
	cmd.Flags().BoolVar(&skipRemote, "skip-remote", false, "Skip remote connectivity checks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")

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
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".mars-harness", "db", "mars.db")
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("serve: create db directory: %w", err)
			}

			webhookSecret := os.Getenv("MARS_HARNESS_WEBHOOK_SECRET")
			dashboardAddr := fmt.Sprintf(":%d", cfg.DashboardPort)

			serve.Cleanup(cfg.WebhookPort, dbPath, cfg.DashboardPort)

			srv, err := serve.New(serve.Config{
				WebhookAddr:   webhookAddr,
				WebhookSecret: webhookSecret,
				DBPath:        dbPath,
				Concurrency:   concurrency,
				ModelsDir:     cfg.ModelsDir,
				BinDir:        cfg.BinDir,
				DashboardAddr: dashboardAddr,
			})
			if err != nil {
				return err
			}

			sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

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
		Long:  "Register a local repository that has a .harness/manifest.yaml so the orchestrator can manage it.",
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

			if _, err := bundle.Load(absPath); err != nil {
				return fmt.Errorf("register: %s does not contain a valid .harness/manifest.yaml — run `mars-harness init` first: %w",
					absPath, err)
			}

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".mars-harness", "db", "mars.db")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/mars.db)")

	return cmd
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database at %s: %w — check path and permissions", path, err)
	}
	return db, nil
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

			if _, err := os.Stat(filepath.Join(absPath, ".harness", "manifest.yaml")); os.IsNotExist(err) {
				tw.WriteAssistant("No .harness/ found — initialising with default pipeline...")
				if initErr := scanner.Init(absPath, force); initErr != nil {
					tw.WriteError(fmt.Sprintf("init failed: %v", initErr))
					return initErr
				}
			}

			if _, err := bundle.Load(absPath); err != nil {
				tw.WriteError(fmt.Sprintf("manifest invalid: %v", err))
				return err
			}

			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				slog.Warn("config load failed, using defaults", "err", err)
			}

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".mars-harness", "db", "mars.db")
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("start: create db directory: %w", err)
			}

			webhookAddr := fmt.Sprintf(":%d", cfg.WebhookPort)
			dashboardAddr := fmt.Sprintf(":%d", cfg.DashboardPort)

			serve.Cleanup(cfg.WebhookPort, dbPath, cfg.DashboardPort)

			srv, err := serve.New(serve.Config{
				WebhookAddr:   webhookAddr,
				DBPath:        dbPath,
				Concurrency:   concurrency,
				ModelsDir:     cfg.ModelsDir,
				BinDir:        cfg.BinDir,
				DashboardAddr: dashboardAddr,
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

			return srv.Start(sigCtx)
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of concurrent agent workers (1 = sequential pipeline)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/mars.db)")
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
