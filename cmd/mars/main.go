/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-smoke-validation.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/dashboard.md
- docs/design-docs/dogfood-matrix.md
- docs/design-docs/github-app-integration.md
- docs/design-docs/harness-glossary.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/local-inference.md
- docs/design-docs/release-versioning.md
- docs/design-docs/self-reflective-telemetry.md
- docs/validation/README.md
- docs/validation/agent-smoke/README.md
- docs/product-specs/product-surface.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
- docs/features/F-012-self-improvement-loop.md
- docs/roles/ROLES.md
*/
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"

	"github.com/greaveselliott/mars/internal/agent"
	"github.com/greaveselliott/mars/internal/buildinfo"
	"github.com/greaveselliott/mars/internal/bundle"
	"github.com/greaveselliott/mars/internal/codeintel"
	"github.com/greaveselliott/mars/internal/config"
	ctx "github.com/greaveselliott/mars/internal/context"
	"github.com/greaveselliott/mars/internal/docsync"
	"github.com/greaveselliott/mars/internal/doctor"
	"github.com/greaveselliott/mars/internal/foundationtelemetry"
	gh "github.com/greaveselliott/mars/internal/github"
	"github.com/greaveselliott/mars/internal/githubauth"
	"github.com/greaveselliott/mars/internal/guardrails"
	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/inference"
	"github.com/greaveselliott/mars/internal/llm"
	"github.com/greaveselliott/mars/internal/mcpstdio"
	"github.com/greaveselliott/mars/internal/models"
	"github.com/greaveselliott/mars/internal/qualityscore"
	"github.com/greaveselliott/mars/internal/release"
	"github.com/greaveselliott/mars/internal/safety"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/scoring"
	"github.com/greaveselliott/mars/internal/selfupdate"
	"github.com/greaveselliott/mars/internal/serve"
	"github.com/greaveselliott/mars/internal/setup"
	"github.com/greaveselliott/mars/internal/shellpath"
	"github.com/greaveselliott/mars/internal/telemetry"
	"github.com/greaveselliott/mars/internal/tools"
	"github.com/greaveselliott/mars/internal/trace"
	"github.com/greaveselliott/mars/internal/trust"
	"github.com/greaveselliott/mars/internal/ui"
	"github.com/greaveselliott/mars/internal/updatecheck"
	foundationvalidation "github.com/greaveselliott/mars/internal/validation"
)

var version = buildinfo.DefaultVersion

var (
	commit           = "unknown"
	date             = "unknown"
	jsonErrorWritten bool
)

func main() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		if jsonRequestedFromArgs(os.Args[1:]) {
			if !jsonErrorWritten {
				_ = writeJSONError(err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var showVersion bool
	root := &cobra.Command{
		Use:           "mars",
		Short:         "Autonomous AI delivery system",
		Long:          "MARS — self-hosted autonomous AI delivery. Run setup to get started.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				printVersion(cmd.OutOrStdout())
				return nil
			}
			return cmd.Help()
		},
	}
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version, OS, and architecture")

	root.AddCommand(versionCmd())
	root.AddCommand(updateCmd())
	root.AddCommand(startCmd())
	root.AddCommand(runCmd())
	root.AddCommand(setupCmd())
	root.AddCommand(initCmd())
	root.AddCommand(ejectCmd())
	root.AddCommand(upgradeCmd())
	root.AddCommand(scanCmd())
	root.AddCommand(serveCmd())
	root.AddCommand(registerCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(scoresCmd())
	root.AddCommand(telemetryCmd())
	root.AddCommand(trustCmd())
	root.AddCommand(toolsCmd())
	root.AddCommand(codeIntelCmd())
	root.AddCommand(mcpCmd())
	root.AddCommand(modelsCmd())
	root.AddCommand(guardrailsCmd())
	root.AddCommand(releaseCmd())
	root.AddCommand(checksCmd())
	root.AddCommand(docsyncCmd())
	root.AddCommand(validationCmd())
	root.AddCommand(pathCmd())
	root.AddCommand(authCmd())

	return root
}

func validationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validation",
		Short: "Run and check foundation validation evidence",
		Long: `Run MARS foundation validation helpers.

These commands are source-maintainer gates for repo-owned validation evidence.
They complement, but do not replace, full clean-project lifecycle sweeps.`,
	}
	cmd.AddCommand(validationAgentSmokeCmd())
	return cmd
}

func validationAgentSmokeCmd() *cobra.Command {
	var opts foundationvalidation.AgentSmokeOptions
	cmd := &cobra.Command{
		Use:   "agent-smoke",
		Short: "Run compartmentalised role smoke tests against ephemeral targets",
		Long: `Generate fresh ephemeral target repositories through foundation tools and
run compartmentalised smoke validation cases for MARS roles through
the server job execution path.

Successful runs are discarded by default. Failed runs are retained for
diagnosis unless --discard-failed is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.HarnessRoot == "" {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("validation agent-smoke: determine working directory: %w", err)
				}
				opts.HarnessRoot = wd
			}
			report, err := foundationvalidation.RunAgentSmoke(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if opts.JSON {
				if writeErr := writeJSON(cmd.OutOrStdout(), report); writeErr != nil {
					return writeErr
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), report.Summary())
				for _, result := range report.Results {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s/%s %s %s", result.Role, result.CaseID, result.ProjectType, result.Status)
					if result.ExecutionMode != "" {
						fmt.Fprintf(cmd.OutOrStdout(), " mode=%s", result.ExecutionMode)
					}
					if result.TerminalDisposition != "" {
						fmt.Fprintf(cmd.OutOrStdout(), " disposition=%s", result.TerminalDisposition)
					}
					if result.FailureClass != "" {
						fmt.Fprintf(cmd.OutOrStdout(), " (%s)", result.FailureClass)
					}
					if result.RunPath != "" && !result.Discarded {
						fmt.Fprintf(cmd.OutOrStdout(), " run=%s", result.RunPath)
					}
					fmt.Fprintln(cmd.OutOrStdout())
				}
			}
			if !report.OK() {
				return fmt.Errorf("validation agent-smoke: %d case(s) failed", report.Failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Role, "role", "", "Run only cases for this role")
	cmd.Flags().StringVar(&opts.CaseID, "case", "", "Run one case by ID")
	cmd.Flags().StringVar(&opts.ProjectType, "project-type", "", "Run cases for one project type")
	cmd.Flags().StringVar(&opts.Suite, "suite", "fast", "Suite to run: fast, default, full, or held-out")
	cmd.Flags().IntVar(&opts.Parallel, "parallel", 1, "Maximum cases to run concurrently")
	cmd.Flags().StringVar(&opts.Cycle, "cycle", "", "Stable cycle key for rotating fast/held-out selections")
	cmd.Flags().IntVar(&opts.MaxTurns, "max-turns", 32, "Maximum role turns for live execution")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "Per-case timeout")
	cmd.Flags().StringVar(&opts.ModelEndpoint, "model-endpoint", "", "Optional real OpenAI-compatible model endpoint override; fake or scripted endpoints are not validation evidence")
	cmd.Flags().BoolVar(&opts.SingleServer, "single-server", true, "Use one local inference server for all selected roles; pass --single-server=false for tiered routing")
	cmd.Flags().StringVar(&opts.SingleTier, "single-server-tier", "coding", "Local model tier for --single-server: coding, reasoning, or fast")
	cmd.Flags().BoolVar(&opts.FixtureOnly, "fixture-only", false, "Generate and lint ephemeral targets without running the role")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Write JSON report")
	cmd.Flags().StringVar(&opts.ReportPath, "report", "", "Optional Markdown report path")
	cmd.Flags().BoolVar(&opts.KeepRuns, "keep-runs", false, "Keep successful ephemeral run directories")
	cmd.Flags().BoolVar(&opts.CleanupOnly, "cleanup-only", false, "Remove retained agent-smoke run directories and exit")
	cmd.Flags().BoolVar(&opts.DiscardFailed, "discard-failed", false, "Discard failed run directories after recording results")
	cmd.Flags().StringVar(&opts.Root, "root", "", "Parent directory for ephemeral agent-smoke runs")
	return cmd
}

func docsyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docsync",
		Short: "Audit code-to-documentation metadata",
		Long: `Audit source files for top-of-file MarsDocSync metadata and verify
that associated documentation paths exist and match the universal code map.`,
	}
	cmd.AddCommand(docsyncAuditCmd())
	return cmd
}

func docsyncAuditCmd() *cobra.Command {
	var (
		repoPath string
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Check MarsDocSync metadata on source files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("docsync audit: cannot determine working directory: %w", err)
				}
			}
			report, err := docsync.Audit(docsync.Config{RepoRoot: repoPath})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			fmt.Fprintln(cmd.OutOrStdout(), report.Summary())
			for _, finding := range report.Findings {
				fmt.Fprintf(cmd.OutOrStdout(), "FAIL: %s: %s\n", finding.Path, finding.Message)
			}
			if !report.OK() {
				return fmt.Errorf("docsync audit: %d findings", len(report.Findings))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Status: ok")
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write audit report as JSON")
	return cmd
}

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Expose MARS tools through Model Context Protocol",
		Long: `Expose the registered MARS tool registry through standard MCP
transports so any MCP-compatible client or local harness agent can attach to the
foundation or deployed harness without depending on a model provider.`,
	}
	cmd.AddCommand(mcpServeCmd())
	return cmd
}

func gitChangedPaths(repoRoot string) (map[string]bool, error) {
	paths := map[string]bool{}
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); os.IsNotExist(err) {
		return filesystemPaths(repoRoot)
	}
	cmd := exec.Command("git", "status", "--porcelain=v1", "-z")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git status in %s failed: %w\n%s", repoRoot, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git status in %s failed: %w", repoRoot, err)
	}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry == "" || len(entry) < 4 {
			continue
		}
		status := entry[:2]
		path := strings.TrimSpace(entry[3:])
		if path != "" {
			paths[path] = true
		}
		if (status[0] == 'R' || status[0] == 'C') && i+1 < len(entries) {
			i++
		}
	}
	return paths, nil
}

func filesystemPaths(repoRoot string) (map[string]bool, error) {
	paths := map[string]bool{}
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == repoRoot {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		paths[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect filesystem paths in %s: %w", repoRoot, err)
	}
	return paths, nil
}

func commitGeneratedHarnessBaseline(repoRoot string, preInitChanges map[string]bool) (bool, error) {
	postInitChanges, err := gitChangedPaths(repoRoot)
	if err != nil {
		return false, err
	}
	var generated []string
	for path := range postInitChanges {
		if !preInitChanges[path] {
			generated = append(generated, path)
		}
	}
	if len(generated) == 0 {
		return false, nil
	}
	if err := runStartGit(repoRoot, append([]string{"add", "--"}, generated...)...); err != nil {
		return false, err
	}
	if err := runStartGit(repoRoot, "diff", "--cached", "--quiet"); err == nil {
		return false, nil
	}
	if err := runStartGit(repoRoot,
		"-c", "user.name=MARS",
		"-c", "user.email=mars@example.invalid",
		"commit", "--no-gpg-sign", "-m", "chore(harness): initialize mars harness",
	); err != nil {
		return false, err
	}
	return true, nil
}

func runStartGit(repoRoot string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s in %s failed: %w\n%s", strings.Join(args, " "), repoRoot, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func mcpServeCmd() *cobra.Command {
	var (
		repoPath  string
		allow     string
		role      string
		trustLvl  string
		maxOutput int
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run a stdio MCP server for registered MARS tools",
		Long: `Run a newline-delimited JSON-RPC stdio MCP server. The server exposes
registered MARS tools via tools/list and tools/call, using the same
executor, trust policy, repository root, and JSON arguments as agent runs.

Configure MCP clients to launch this command as a local stdio server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			absRepo, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("mcp serve: resolve repo %s: %w", repoPath, err)
			}
			root, err := tools.NewRoot(absRepo)
			if err != nil {
				return err
			}
			registry, err := tools.DefaultRegistry()
			if err != nil {
				return err
			}
			executor := tools.NewExecutor(registry)
			if maxOutput > 0 {
				executor.MaxOutput = maxOutput
			}
			executor.Session = &tools.Session{
				Role:       role,
				TrustLevel: trustLvl,
			}
			allowlist := splitCSV(allow)
			server := mcpstdio.Server{
				Registry: registry,
				Executor: executor,
				Root:     root,
				Allow:    allowlist,
			}
			return server.Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Repository root for root-scoped tool execution")
	cmd.Flags().StringVar(&allow, "allowlist", "", "Comma-separated tool allowlist; empty exposes every registered tool")
	cmd.Flags().StringVar(&role, "role", "mcp-client", "Role label used for session policy")
	cmd.Flags().StringVar(&trustLvl, "trust", "observer", "Trust level used for mutating-tool policy")
	cmd.Flags().IntVar(&maxOutput, "max-output-bytes", 0, "Maximum combined tool output bytes; 0 uses the executor default")
	return cmd
}

func toolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Inspect and execute registered MARS tools",
		Long: `Inspect and execute registered MARS tools through the same registry,
allowlist, trust policy, repository root, and JSON argument path used by agent
runs. This gives operators and external LLM shells a first-class bridge to
mirrored tools such as tool_create without reaching into Go package internals.`,
	}
	cmd.AddCommand(toolsListCmd())
	cmd.AddCommand(toolsRunCmd())
	return cmd
}

func toolsListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered built-in tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := tools.DefaultRegistry()
			if err != nil {
				return err
			}
			names := registry.Names()
			if jsonOut {
				defs, err := registry.Definitions(names)
				if err != nil {
					return err
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(defs)
			}
			for _, name := range names {
				fmt.Fprintln(cmd.OutOrStdout(), name)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write OpenAI-style tool definitions as JSON")
	return cmd
}

func toolsRunCmd() *cobra.Command {
	var (
		repoPath  string
		argsJSON  string
		allow     string
		role      string
		trustLvl  string
		maxOutput int
		jsonOut   bool
	)
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Execute one registered built-in tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("tools run: tool name is empty")
			}
			absRepo, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("tools run: resolve repo %s: %w", repoPath, err)
			}
			root, err := tools.NewRoot(absRepo)
			if err != nil {
				return err
			}
			registry, err := tools.DefaultRegistry()
			if err != nil {
				return err
			}
			executor := tools.NewExecutor(registry)
			if maxOutput > 0 {
				executor.MaxOutput = maxOutput
			}
			executor.Session = &tools.Session{
				Role:       role,
				TrustLevel: trustLvl,
			}
			allowlist := []string{name}
			if strings.TrimSpace(allow) != "" {
				allowlist = splitCSV(allow)
			}
			res, err := executor.Execute(cmd.Context(), root, allowlist, name, argsJSON)
			if err != nil {
				return err
			}
			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"tool":      name,
					"output":    res.Output,
					"stderr":    res.Stderr,
					"exit_code": res.ExitCode,
					"truncated": res.Truncated,
					"is_binary": res.IsBinary,
					"duration":  res.Duration.String(),
				})
			}
			if formatted := res.FormatForModel(); formatted != "" {
				fmt.Fprintln(cmd.OutOrStdout(), formatted)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Repository root for root-scoped tool execution")
	cmd.Flags().StringVar(&argsJSON, "args-json", "{}", "JSON object arguments for the tool")
	cmd.Flags().StringVar(&allow, "allowlist", "", "Comma-separated allowlist; defaults to the named tool")
	cmd.Flags().StringVar(&role, "role", "cli-tool", "Role label used for session policy")
	cmd.Flags().StringVar(&trustLvl, "trust", "observer", "Trust level used for mutating-tool policy")
	cmd.Flags().IntVar(&maxOutput, "max-output-bytes", 0, "Maximum combined tool output bytes; 0 uses the executor default")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write structured JSON output")
	return cmd
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Inspect and evaluate model candidates",
	}
	cmd.AddCommand(modelsEligibleCmd())
	cmd.AddCommand(modelsListCmd())
	cmd.AddCommand(modelsEvaluateCmd())
	cmd.AddCommand(modelsOverrideCmd())
	cmd.AddCommand(modelsCredentialsCmd())
	return cmd
}

func modelsListCmd() *cobra.Command {
	var (
		provider string
		eligible bool
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List model candidates from a provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer silenceLogsForJSON(jsonOut)()
			if eligible {
				report := models.EvaluateLocalBundles(hardware.Detect())
				if jsonOut {
					return writeJSON(os.Stdout, report.Bundles)
				}
				printEligibleBundles(report)
				return nil
			}
			switch models.NormalizeProvider(provider) {
			case models.ProviderOllama:
				rows, err := models.ListOllamaModels(cmd.Context(), nil)
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(os.Stdout, rows)
				}
				fmt.Println("Ollama models")
				for _, row := range rows {
					fmt.Printf("  - %s", row.Name)
					if row.Size != "" {
						fmt.Printf(" size=%s", row.Size)
					}
					if row.Modified != "" {
						fmt.Printf(" modified=%s", row.Modified)
					}
					fmt.Println()
				}
				return nil
			case models.ProviderRegistry:
				rows := hardware.DefaultModels(hardware.ProfileMedium)
				if jsonOut {
					return writeJSON(os.Stdout, rows)
				}
				fmt.Println("Medium-profile registry defaults")
				for tier, spec := range rows {
					fmt.Printf("  - %s: %s %s %s revision=%s sha256=%s\n", tier, spec.Name, spec.Params, spec.Quant, spec.Revision, spec.SHA256)
				}
				return nil
			default:
				return fmt.Errorf("models list: unsupported provider %q — use ollama or registry", provider)
			}
		},
	}
	cmd.Flags().StringVar(&provider, "provider", models.ProviderRegistry, "Provider to list: registry or ollama")
	cmd.Flags().BoolVar(&eligible, "eligible", false, "Show only local bundle eligibility from detected hardware")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func modelsEligibleCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "eligible",
		Short: "Show local model bundles eligible for detected hardware",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer silenceLogsForJSON(jsonOut)()
			report := models.EvaluateLocalBundles(hardware.Detect())
			if jsonOut {
				return writeJSON(os.Stdout, report)
			}
			printEligibleBundles(report)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func modelsEvaluateCmd() *cobra.Command {
	var (
		endpoint        string
		model           string
		provider        string
		apiKeyEnv       string
		repoRoot        string
		reportDir       string
		saveReport      bool
		revision        string
		sha256          string
		candidateSource string
		cloud           bool
		timeout         time.Duration
		jsonOut         bool
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
			provider = models.NormalizeProvider(provider)
			if provider == models.ProviderOllama && endpoint == "" && model != "" {
				endpoint = models.DefaultOllamaEndpoint
			}
			if endpoint == "" || model == "" {
				plan := models.DefaultPlan(time.Now())
				if jsonOut {
					return writeJSON(os.Stdout, plan)
				}
				printModelEvaluationPlan(plan)
				return nil
			}
			absRepo, err := filepath.Abs(repoRoot)
			if err != nil {
				return fmt.Errorf("models evaluate: resolve repo %s: %w", repoRoot, err)
			}
			resolvedReportDir := ""
			if saveReport {
				resolvedReportDir = reportDir
				if !filepath.IsAbs(resolvedReportDir) {
					resolvedReportDir = filepath.Join(absRepo, resolvedReportDir)
				}
			}
			hw := hardware.Detect()

			report, err := models.Evaluate(cmd.Context(), models.Config{
				Endpoint:        endpoint,
				Model:           model,
				Provider:        provider,
				APIKeyEnv:       apiKeyEnv,
				RepoRoot:        absRepo,
				ReportsDir:      resolvedReportDir,
				HardwareProfile: string(hw.Profile),
				CandidateSource: candidateSource,
				Revision:        revision,
				SHA256:          sha256,
				Cloud:           cloud,
				Timeout:         timeout,
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
	cmd.Flags().StringVar(&provider, "provider", models.ProviderOpenAICompatible, "Provider label: openai-compatible, ollama, openai, anthropic, gemini, mistral, xai, deepseek, groq, or cohere")
	cmd.Flags().StringVar(&apiKeyEnv, "api-key-env", "", "Environment variable containing the provider API key")
	cmd.Flags().StringVar(&repoRoot, "repo", ".", "Repo root for repo-backed benchmark cases")
	cmd.Flags().StringVar(&reportDir, "report-dir", filepath.Join("docs", "generated", "model-evaluations"), "Directory for persisted evaluation reports")
	cmd.Flags().BoolVar(&saveReport, "save-report", true, "Persist live evaluation reports")
	cmd.Flags().StringVar(&revision, "revision", "", "Immutable artifact revision for promotion checks")
	cmd.Flags().StringVar(&sha256, "sha256", "", "Artifact SHA256 for promotion checks")
	cmd.Flags().StringVar(&candidateSource, "source", "", "Candidate source URL or artifact reference")
	cmd.Flags().BoolVar(&cloud, "cloud", false, "Mark candidate as cloud-only/remote")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Per-request timeout")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func modelsOverrideCmd() *cobra.Command {
	var (
		repoRoot    string
		tier        string
		role        string
		routing     string
		localBundle string
		provider    string
		model       string
		endpoint    string
		apiKeyEnv   string
		reason      string
		jsonOut     bool
	)
	cmd := &cobra.Command{
		Use:   "override",
		Short: "Set a repo-owned model override for one tier or role",
		RunE: func(cmd *cobra.Command, args []string) error {
			absRepo, err := filepath.Abs(repoRoot)
			if err != nil {
				return fmt.Errorf("models override: resolve repo %s: %w", repoRoot, err)
			}
			path, err := models.SetModelOverride(absRepo, tier, role, models.ModelOverride{
				Routing:     routing,
				LocalBundle: localBundle,
				Provider:    provider,
				Model:       model,
				Endpoint:    endpoint,
				APIKeyEnv:   apiKeyEnv,
				Reason:      reason,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, map[string]string{"path": path})
			}
			fmt.Printf("Wrote model override: %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo", ".", "Target repo root")
	cmd.Flags().StringVar(&tier, "tier", "", "Tier to override: fast, reasoning, or coding")
	cmd.Flags().StringVar(&role, "role", "", "Role name to override")
	cmd.Flags().StringVar(&routing, "routing", "", "Routing mode: local, cloud, or defer")
	cmd.Flags().StringVar(&localBundle, "local-bundle", "", "Local bundle: auto, local-cpu-q3, local-balanced-q4, or local-quality-q8")
	cmd.Flags().StringVar(&provider, "provider", models.ProviderOllama, "Provider: ollama, openai-compatible, openai, anthropic, gemini, mistral, xai, deepseek, groq, or cohere")
	cmd.Flags().StringVar(&model, "model", "", "Provider model name")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "OpenAI-compatible endpoint; Ollama defaults to local Ollama")
	cmd.Flags().StringVar(&apiKeyEnv, "api-key-env", "", "Environment variable containing the provider API key")
	cmd.Flags().StringVar(&reason, "reason", "", "Operator rationale saved with the override")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func modelsCredentialsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manage local model provider credentials",
	}
	cmd.AddCommand(modelsCredentialsWriteLocalEnvCmd())
	return cmd
}

func modelsCredentialsWriteLocalEnvCmd() *cobra.Command {
	var (
		repoRoot  string
		apiKeyEnv string
		yes       bool
		jsonOut   bool
	)
	cmd := &cobra.Command{
		Use:   "write-local-env",
		Short: "Write a provider credential from the process environment into ignored .harness/.env.local",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes && !jsonOut && !ui.IsTerminal(os.Stdin) {
				return fmt.Errorf("models credentials: non-interactive use requires --yes")
			}
			absRepo, err := filepath.Abs(repoRoot)
			if err != nil {
				return fmt.Errorf("models credentials: resolve repo %s: %w", repoRoot, err)
			}
			localPath, examplePath, err := models.WriteLocalCredential(absRepo, apiKeyEnv)
			if err != nil {
				if jsonOut {
					return writeJSONError(err)
				}
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, map[string]string{"status": "ok", "local_env": localPath, "example": examplePath})
			}
			fmt.Printf("Wrote local credential env file: %s\n", localPath)
			fmt.Printf("Updated example env names: %s\n", examplePath)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo", ".", "Target repo root")
	cmd.Flags().StringVar(&apiKeyEnv, "api-key-env", "", "Environment variable containing the provider API key")
	cmd.Flags().BoolVar(&yes, "yes", false, "Do not prompt; fail with remediation when required input is missing")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func guardrailsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guardrails",
		Short: "Inspect and install repository safety guardrails",
	}
	cmd.AddCommand(guardrailsSecretScanCmd())
	cmd.AddCommand(guardrailsInstallHooksCmd())
	return cmd
}

type cliSecretFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Pattern string `json:"pattern"`
	Match   string `json:"match"`
}

func guardrailsSecretScanCmd() *cobra.Command {
	var (
		repoRoot string
		staged   bool
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "secret-scan",
		Short: "Scan repository files for common secret patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			absRepo, err := filepath.Abs(repoRoot)
			if err != nil {
				return fmt.Errorf("guardrails secret-scan: resolve repo %s: %w", repoRoot, err)
			}
			findings, err := runCLISecretScan(absRepo, staged)
			if err != nil {
				return err
			}
			if jsonOut {
				_ = writeJSON(os.Stdout, map[string]any{"status": secretScanStatus(findings), "findings": findings})
			} else if len(findings) == 0 {
				fmt.Println("No secrets detected.")
			} else {
				for _, finding := range findings {
					fmt.Printf("%s:%d %s %s\n", finding.File, finding.Line, finding.Pattern, finding.Match)
				}
			}
			if len(findings) > 0 {
				return fmt.Errorf("guardrails secret-scan: %d finding(s)", len(findings))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo", ".", "Repository root")
	cmd.Flags().BoolVar(&staged, "staged", false, "Scan staged files only")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func guardrailsInstallHooksCmd() *cobra.Command {
	var (
		repoRoot string
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "Install optional git hooks for MARS guardrails",
		RunE: func(cmd *cobra.Command, args []string) error {
			absRepo, err := filepath.Abs(repoRoot)
			if err != nil {
				return fmt.Errorf("guardrails install-hooks: resolve repo %s: %w", repoRoot, err)
			}
			path, changed, err := installSecretScanHook(absRepo)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, map[string]any{"status": "ok", "hook": path, "changed": changed})
			}
			if changed {
				fmt.Printf("Installed MARS pre-commit secret scan hook: %s\n", path)
			} else {
				fmt.Printf("MARS pre-commit secret scan hook already installed: %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoRoot, "repo", ".", "Repository root")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func secretScanStatus(findings []cliSecretFinding) string {
	if len(findings) > 0 {
		return "blocked"
	}
	return "ok"
}

func runCLISecretScan(repoRoot string, staged bool) ([]cliSecretFinding, error) {
	paths, err := secretScanCandidatePaths(repoRoot, staged)
	if err != nil {
		return nil, err
	}
	var findings []cliSecretFinding
	for _, rel := range paths {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" || rel == ".harness/.env.local" || strings.HasPrefix(rel, ".git/") {
			continue
		}
		abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		for _, hit := range safety.ScanForSecrets(rel, string(data)) {
			findings = append(findings, cliSecretFinding{
				File:    hit.File,
				Line:    hit.Line,
				Pattern: hit.Pattern,
				Match:   "[REDACTED]",
			})
		}
	}
	return findings, nil
}

func secretScanCandidatePaths(repoRoot string, staged bool) ([]string, error) {
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); os.IsNotExist(err) {
		paths, err := filesystemPaths(repoRoot)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(paths))
		for path := range paths {
			out = append(out, path)
		}
		sort.Strings(out)
		return out, nil
	}
	args := []string{"ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	if staged {
		args = []string{"diff", "--cached", "--name-only", "-z", "--diff-filter=ACMRT"}
	}
	gitCmd := exec.Command("git", args...)
	gitCmd.Dir = repoRoot
	out, err := gitCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("guardrails secret-scan: git %s failed: %w", strings.Join(args, " "), err)
	}
	var paths []string
	for _, rel := range strings.Split(string(out), "\x00") {
		rel = strings.TrimSpace(rel)
		if rel != "" {
			paths = append(paths, rel)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func installSecretScanHook(repoRoot string) (string, bool, error) {
	gitDir := filepath.Join(repoRoot, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		return "", false, fmt.Errorf("guardrails install-hooks: %s is missing — run inside a git checkout", gitDir)
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return "", false, fmt.Errorf("guardrails install-hooks: create %s: %w", hooksDir, err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	const begin = "# BEGIN MARS SECRET SCAN\n"
	const end = "# END MARS SECRET SCAN\n"
	block := begin + "mars guardrails secret-scan --repo " + shellQuote(repoRoot) + " --staged\n" + end
	existing, _ := os.ReadFile(hookPath)
	content := string(existing)
	if strings.Contains(content, begin) && strings.Contains(content, end) {
		start := strings.Index(content, begin)
		stop := strings.Index(content, end) + len(end)
		updated := content[:start] + block + content[stop:]
		if updated == content {
			return hookPath, false, nil
		}
		if err := os.WriteFile(hookPath, []byte(updated), 0o755); err != nil {
			return "", false, fmt.Errorf("guardrails install-hooks: write %s: %w", hookPath, err)
		}
		return hookPath, true, nil
	}
	var b strings.Builder
	if content == "" {
		b.WriteString("#!/bin/sh\n")
	} else {
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(block)
	if err := os.WriteFile(hookPath, []byte(b.String()), 0o755); err != nil {
		return "", false, fmt.Errorf("guardrails install-hooks: write %s: %w", hookPath, err)
	}
	return hookPath, true, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeJSONError(err error) error {
	jsonErrorWritten = true
	_ = writeJSON(os.Stdout, map[string]any{
		"status":      "error",
		"error":       err.Error(),
		"remediation": remediationForError(err),
	})
	return err
}

func jsonRequestedFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}

func remediationForError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown flag: --api-key"):
		return "rerun with --api-key-env <ENV_NAME>; raw API key values are never accepted"
	case strings.Contains(msg, "--api-key-env is required"):
		return "rerun with --api-key-env <ENV_NAME>; raw API key values are never accepted"
	case strings.Contains(msg, "credential env") && strings.Contains(msg, "is not set"):
		return "export the named environment variable, then rerun mars models credentials write-local-env --repo <repo> --api-key-env <ENV_NAME> --yes --json"
	case strings.Contains(msg, "environment variable") && strings.Contains(msg, "is not set"):
		return "export the named environment variable, then rerun mars models credentials write-local-env --repo <repo> --api-key-env <ENV_NAME> --yes --json"
	case strings.Contains(msg, "missing model file(s)") || (strings.Contains(msg, "local model bundle") && strings.Contains(msg, "missing")):
		return "run mars setup --inference local --local-bundle auto --download --yes --json, then rerun the command"
	case strings.Contains(msg, "--log-file path") || strings.Contains(msg, "--db path"):
		return "pass a writable runtime artifact path outside the target repo or use the documented default"
	case strings.Contains(msg, "non-interactive") && strings.Contains(msg, "--yes"):
		return "rerun with --yes and all required flags, or run interactively in a TTY"
	default:
		return "fix the reported input or environment issue, then rerun the same command"
	}
}

func silenceLogsForJSON(enabled bool) func() {
	if !enabled {
		return func() {}
	}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return func() { slog.SetDefault(old) }
}

func printEligibleBundles(report models.EligibilityReport) {
	fmt.Println("Eligible local model bundles")
	fmt.Printf("Hardware: profile=%s os=%s arch=%s ram=%dMiB gpus=%d\n\n",
		report.Hardware.Profile, report.Hardware.OS, report.Hardware.Arch, report.Hardware.RAMMiB, len(report.Hardware.GPUs))
	fmt.Printf("%-20s %-10s %-9s %s\n", "BUNDLE", "PROFILE", "STATUS", "REASON")
	for _, row := range report.Bundles {
		status := "disabled"
		if row.Eligible {
			status = "eligible"
		}
		if row.Selected {
			status = "selected"
		}
		fmt.Printf("%-20s %-10s %-9s %s\n", row.ID, row.Profile, status, row.DisabledReason)
	}
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
		ref := c.Name
		if c.Provider != "" && c.Model != "" {
			ref = fmt.Sprintf("%s [%s:%s]", c.Name, c.Provider, c.Model)
		}
		fmt.Printf("  - %s (%s,%s%s): %s\n", ref, c.Role, mode, mem, c.Why)
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
	fmt.Println("\nRun live evaluation with: mars models evaluate --endpoint <url> --model <name>")
}

func printModelEvaluationReport(report models.Report) {
	fmt.Printf("Model evaluation: %s\n", report.Model)
	fmt.Printf("Provider: %s\n", report.Provider)
	fmt.Printf("Endpoint: %s\n", report.Endpoint)
	if report.HardwareProfile != "" {
		fmt.Printf("Hardware profile: %s\n", report.HardwareProfile)
	}
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
	fmt.Printf("\nPromotion: %s\n", strings.ToUpper(report.Promotion.Decision))
	for _, reason := range report.Promotion.Reasons {
		fmt.Printf("  - %s\n", reason)
	}
	if report.ReportPath != "" {
		fmt.Printf("\nReport saved: %s\n", report.ReportPath)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, OS, and architecture",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion(cmd.OutOrStdout())
		},
	}
}

func printVersion(out io.Writer) {
	fmt.Fprintln(out, versionLine())
}

func versionLine() string {
	return fmt.Sprintf("mars %s %s/%s commit=%s built=%s", version, runtime.GOOS, runtime.GOARCH, commit, date)
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the CLI binary or a deployed target harness",
		Long: `Update MARS surfaces with one verb.

Use "mars update tool" to reinstall or upgrade the installed CLI.
Use "mars update harness --repo <path>" to update the .harness/
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
		Long: `Check version drift for the installed mars tool and a target repo's
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
		sourceUpdate  bool
		dryRun        bool
		jsonOut       bool
	)
	cmd := &cobra.Command{
		Use:     "tool",
		Aliases: []string{"binary", "cli"},
		Short:   "Reinstall or upgrade the mars command",
		Long: `Reinstall the mars command without changing directories.

By default this acquires the canonical platform archive, verifies its signed
checksum, workflow identity, commit, metadata, and structure, then durably
replaces the currently running mars binary. Use --source for source-development
updates through go install, or pass --version main which selects the source path
automatically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			method := selfupdate.UpdateMethod("")
			if sourceUpdate {
				method = selfupdate.MethodSource
			}
			cfg := selfupdate.Config{
				Version:        updateVersion,
				CurrentVersion: version,
				CurrentCommit:  commit,
				InstallDir:     installDir,
				Method:         method,
				DryRun:         dryRun,
			}
			plan, err := selfupdate.Run(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, plan)
			}
			fmt.Printf("mars update tool\n")
			fmt.Printf("Method: %s\n", plan.Method)
			fmt.Printf("Version: %s\n", plan.Version)
			fmt.Printf("Install dir: %s\n", plan.InstallDir)
			if len(plan.Command) > 0 {
				fmt.Printf("Command: GOBIN=%s %s\n", plan.InstallDir, strings.Join(plan.Command, " "))
			}
			if plan.AssetName != "" {
				fmt.Printf("Archive: %s\n", plan.AssetName)
			}
			if plan.ShellPath.InstallDir != "" {
				fmt.Printf("Shell PATH: %s\n", plan.ShellPath.Message)
				if plan.ShellPath.ProfilePath != "" {
					fmt.Printf("Profile: %s\n", plan.ShellPath.ProfilePath)
				}
				if plan.ShellPath.ReloadHint != "" {
					fmt.Printf("Reload: %s\n", plan.ShellPath.ReloadHint)
				}
			}
			if dryRun {
				fmt.Printf("Dry run: no changes made\n")
				return nil
			}
			fmt.Printf("Installed: %s\n", plan.BinaryPath)
			fmt.Printf("Run: mars version\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&updateVersion, "version", selfupdate.DefaultVersion, "Release or source version to install, e.g. latest, v0.5.3, or main")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Install directory; default is the current mars binary directory")
	cmd.Flags().BoolVar(&sourceUpdate, "source", false, "Use go install instead of signed release archives")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the update plan without downloading or replacing the binary")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func pathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Configure shell PATH for the installed command",
	}
	cmd.AddCommand(pathSetupCmd())
	return cmd
}

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Configure and check external authentication used by MARS",
	}
	cmd.AddCommand(authGitHubCmd())
	return cmd
}

func authGitHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Configure and check GitHub auth for private MARS releases",
	}
	cmd.AddCommand(authGitHubCheckCmd())
	cmd.AddCommand(authGitHubSetupCmd())
	return cmd
}

func authGitHubCheckCmd() *cobra.Command {
	var (
		configPath string
		jsonOut    bool
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check GitHub auth for private MARS release assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := githubauth.Check(cmd.Context(), githubauth.Options{ConfigPath: configPath})
			if jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				printGitHubAuthReport(cmd.OutOrStdout(), report)
			}
			if report.Status != githubauth.StatusOK {
				return fmt.Errorf("auth github check: %s — %s", report.Message, report.NextAction)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.mars/config.yaml)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func authGitHubSetupCmd() *cobra.Command {
	var (
		configPath string
		token      string
		jsonOut    bool
	)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Prepare GitHub auth for private MARS release assets",
		Long: `Prepare GitHub auth for private MARS release assets.

The recommended path is to authenticate GitHub CLI once with "gh auth login",
then run this command. Setup saves a verified GitHub CLI token as an owner-only
local fallback for future update runs. Headless installs may pass --token or set
GH_TOKEN or GITHUB_TOKEN instead. Token values are never printed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := githubauth.Options{ConfigPath: configPath}
			if strings.TrimSpace(token) != "" {
				path := strings.TrimSpace(configPath)
				if path == "" {
					path = config.DefaultPath()
				}
				opts = githubauth.Options{
					ConfigPath:   path,
					ConfigToken:  strings.TrimSpace(token),
					DisableGHCLI: true,
					Env:          func(string) string { return "" },
				}
				report, err := githubauth.Setup(cmd.Context(), opts)
				if err != nil {
					return err
				}
				if jsonOut {
					if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
						return err
					}
				} else {
					printGitHubAuthReport(cmd.OutOrStdout(), report)
				}
				if report.Status != githubauth.StatusOK {
					return fmt.Errorf("auth github setup: %s — %s", report.Message, report.NextAction)
				}
				return nil
			}
			report, err := githubauth.Setup(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				printGitHubAuthReport(cmd.OutOrStdout(), report)
			}
			if report.Status != githubauth.StatusOK {
				return fmt.Errorf("auth github setup: %s — %s", report.Message, report.NextAction)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.mars/config.yaml)")
	cmd.Flags().StringVar(&token, "token", "", "Persist a GitHub token in ~/.mars/config.yaml for headless installs; prefer GitHub CLI auth when possible")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func printGitHubAuthReport(w io.Writer, report githubauth.Report) {
	fmt.Fprintln(w, "GitHub private release auth")
	fmt.Fprintf(w, "Status: %s\n", report.Status)
	fmt.Fprintf(w, "Auth source: %s\n", report.AuthSource)
	fmt.Fprintf(w, "Repo access: %s\n", report.RepoAccess)
	fmt.Fprintf(w, "Release access: %s\n", report.ReleaseAccess)
	fmt.Fprintf(w, "Message: %s\n", report.Message)
	if report.NextAction != "" {
		fmt.Fprintf(w, "Next action: %s\n", report.NextAction)
	}
}

func pathSetupCmd() *cobra.Command {
	var (
		installDir string
		shellName  string
		dryRun     bool
		jsonOut    bool
	)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Add the mars install directory to the current user's shell PATH",
		Long:  "Detect Fish, Zsh, Bash, POSIX sh, or Csh/Tcsh and write an idempotent shell profile snippet so mars works in new terminals.",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := shellpath.Ensure(shellpath.Config{
				InstallDir: installDir,
				ShellPath:  shellName,
				DryRun:     dryRun,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, result)
			}
			fmt.Printf("mars path setup\n")
			fmt.Printf("Install dir: %s\n", result.InstallDir)
			fmt.Printf("Shell: %s\n", result.Shell)
			fmt.Printf("Status: %s\n", result.Message)
			if result.ProfilePath != "" {
				fmt.Printf("Profile: %s\n", result.ProfilePath)
			}
			if result.ReloadHint != "" {
				fmt.Printf("Reload: %s\n", result.ReloadHint)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Directory containing mars; default resolves from current executable or Go bin")
	cmd.Flags().StringVar(&shellName, "shell", "", "Shell path/name to configure; default is $SHELL")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be configured without writing profile files")
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
	fmt.Println("MARS Update Check")
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
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(releaseBackfillNotesCmd())
	cmd.AddCommand(releaseNotesCmd())
	return cmd
}

func releaseBackfillNotesCmd() *cobra.Command {
	var (
		repoPath   string
		minVersion string
		maxVersion string
		dryRun     bool
		check      bool
	)
	cmd := &cobra.Command{
		Use:   "backfill-notes",
		Short: "Backfill historical release narrative sections",
		Long: `Backfill existing CHANGELOG.md entries that have mars release
markers so historical releases use the current Impact, Why, and What Changed
narrative format. Marker commits define each release range. Use --dry-run to
preview without writing and --check to fail when CHANGELOG.md is stale.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("release backfill-notes: cannot determine working directory: %w", err)
				}
			}
			result, err := release.BackfillNotes(cmd.Context(), release.BackfillConfig{
				RepoRoot:   repoPath,
				MinVersion: minVersion,
				MaxVersion: maxVersion,
				DryRun:     dryRun,
				Check:      check,
			})
			if err != nil && len(result.Entries) == 0 && len(result.Changed) == 0 {
				return err
			}
			printReleaseBackfillResult(cmd.OutOrStdout(), result, dryRun, check)
			return err
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().StringVar(&minVersion, "min-version", "", "Minimum release version to backfill, inclusive")
	cmd.Flags().StringVar(&maxVersion, "max-version", "", "Maximum release version to backfill, inclusive")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview backfill without writing CHANGELOG.md")
	cmd.Flags().BoolVar(&check, "check", false, "Fail if any selected release entries need backfill")
	return cmd
}

func printReleaseBackfillResult(out io.Writer, result release.BackfillResult, dryRun, check bool) {
	fmt.Fprintf(out, "Release backfill-notes: checked %d entries, changed %d\n", len(result.Entries), len(result.Changed))
	if len(result.Changed) > 0 {
		fmt.Fprintln(out, "Changed releases:")
		for _, version := range result.Changed {
			fmt.Fprintf(out, "  %s\n", version)
		}
	}
	switch {
	case check && len(result.Changed) == 0:
		fmt.Fprintln(out, "Status: ok")
	case dryRun:
		fmt.Fprintln(out, "Dry run: no files written")
	case len(result.UpdatedFiles) > 0:
		fmt.Fprintln(out, "Updated files:")
		for _, path := range result.UpdatedFiles {
			fmt.Fprintf(out, "  %s\n", path)
		}
	}
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

func checksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checks",
		Short: "Run and record local Mars checks",
	}
	cmd.AddCommand(checksRunCmd())
	return cmd
}

func checksRunCmd() *cobra.Command {
	var (
		repoPath string
		dbPath   string
		name     string
		role     string
	)
	cmd := &cobra.Command{
		Use:   "run --repo <path> --name <check-name> -- <command> [args...]",
		Short: "Run a local check and record the result for Mars orchestration",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(repoPath) == "" {
				return fmt.Errorf("checks run: --repo is required")
			}
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("checks run: --name is required")
			}
			if strings.TrimSpace(role) == "" {
				role = "engineer"
			}
			repoAbs, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("checks run: resolve repo path: %w", err)
			}
			resolvedDB, repoID, err := resolveRepoDBAndID(repoAbs, dbPath)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(resolvedDB), 0o755); err != nil {
				return fmt.Errorf("checks run: create database directory: %w", err)
			}
			store, err := scoring.OpenStore(resolvedDB)
			if err != nil {
				return err
			}
			defer store.Close()

			started := time.Now().UTC()
			localCmd := exec.CommandContext(cmd.Context(), args[0], args[1:]...)
			localCmd.Dir = repoAbs
			localCmd.Stdout = cmd.OutOrStdout()
			localCmd.Stderr = cmd.ErrOrStderr()
			runErr := localCmd.Run()
			exitCode := 0
			if runErr != nil {
				exitCode = 1
				var exitErr *exec.ExitError
				if errors.As(runErr, &exitErr) {
					exitCode = exitErr.ExitCode()
				}
			}
			outcomeType := scoring.OutcomeChecksPassed
			if runErr != nil {
				outcomeType = scoring.OutcomeChecksFailed
			}
			details, _ := json.Marshal(map[string]any{
				"name":        name,
				"command":     args,
				"exit_code":   exitCode,
				"duration_ms": time.Since(started).Milliseconds(),
			})
			if err := store.RecordOutcome(cmd.Context(), scoring.Outcome{
				JobID:      "local-check:" + name + ":" + started.Format("20060102T150405Z"),
				RepoID:     repoID,
				Role:       role,
				Type:       outcomeType,
				Details:    string(details),
				RecordedAt: time.Now().UTC(),
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Recorded check %q as %s for repo %s role %s\n", name, outcomeType, repoID, role)
			if runErr != nil {
				return fmt.Errorf("checks run: command failed with exit code %d: %w", exitCode, runErr)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository whose check is being recorded")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().StringVar(&name, "name", "", "Stable local check name")
	cmd.Flags().StringVar(&role, "role", "engineer", "Role to attribute check outcome to")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func runCmd() *cobra.Command {
	var (
		repoPath      string
		modelEndpoint string
		traceFlag     bool
		debug         bool
		logFile       string
		dryRun        bool
		noInit        bool
		codeIntelFlag string
		budget        int
		maxTurns      int
	)

	cmd := &cobra.Command{
		Use:   "run <role>",
		Short: "Run an agent role against a repository",
		Long: `Load the .harness/ bundle from --repo and execute the named role.

If .harness/manifest.yaml is missing, the same scaffold as 'mars init'
is applied automatically (requires a git repository). Use --no-init with
--dry-run when inspecting an uninitialized target without writing harness
scaffolding. The source-only foundation-maintainer role may run from the
mars source repo with --dry-run --no-init without creating a source
manifest.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]
			return executeRun(runOpts{
				roleName:      roleName,
				repoPath:      repoPath,
				modelEndpoint: modelEndpoint,
				trace:         traceFlag,
				debug:         debug,
				logFile:       logFile,
				dryRun:        dryRun,
				noInit:        noInit,
				codeIntelFlag: codeIntelFlag,
				budget:        budget,
				maxTurns:      maxTurns,
			})
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (required)")
	cmd.Flags().StringVar(&modelEndpoint, "model-endpoint", "", "Override LLM endpoint (e.g. http://127.0.0.1:8080)")
	cmd.Flags().BoolVar(&traceFlag, "trace", false, "Enable verbose execution trace output (compatibility alias for --debug on run)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Stream verbose trace and logs inline instead of using the TTY dashboard")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Write verbose command logs to this file (default ~/.mars/traces/logs/<timestamp>-run.log)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print assembled system prompt and exit without calling the LLM")
	cmd.Flags().BoolVar(&noInit, "no-init", false, "Do not auto-initialize missing .harness/ before running; useful with --dry-run for observer-safe previews")
	cmd.Flags().StringVar(&codeIntelFlag, "code-intel", "", "Enable automatic code graph context and loop maintenance: true or false (default from config/env)")
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
	debug         bool
	logFile       string
	dryRun        bool
	noInit        bool
	codeIntelFlag string
	budget        int
	maxTurns      int
}

const foundationMaintainerRoleName = "foundation-maintainer"

type runtimeDisplay struct {
	out       io.Writer
	dashboard *ui.TerminalDashboard
	jobViews  ui.JobViewFactory
	logger    *ui.InstalledLogger
	debug     bool
}

func newRuntimeDisplay(command, logFile string, debug bool, out, logInline io.Writer, provider ui.StatusProvider, opts ui.DashboardOptions) (*runtimeDisplay, error) {
	path := strings.TrimSpace(logFile)
	if path == "" {
		var err error
		path, err = ui.DefaultLogPath(command, time.Now())
		if err != nil {
			return nil, err
		}
	}
	if opts.Command == "" {
		opts.Command = command
	}
	opts.LogPath = path
	dash := ui.NewTerminalDashboard(out, provider, opts)

	display := &runtimeDisplay{out: out, dashboard: dash, debug: debug}
	switch {
	case debug:
		display.jobViews = ui.NewDebugJobViewFactory(out, false, false)
	case dash.Active():
		display.jobViews = dash
	default:
		display.jobViews = ui.NewPlainJobViewFactory(out)
	}

	var logDash *ui.TerminalDashboard
	if !debug && dash.Active() {
		logDash = dash
	}
	logger, err := ui.InstallCommandLogger(ui.LoggingConfig{
		Command:   command,
		LogPath:   path,
		Debug:     debug,
		Inline:    logInline,
		Dashboard: logDash,
	})
	if err != nil {
		return nil, err
	}
	display.logger = logger
	return display, nil
}

func (d *runtimeDisplay) Start() {
	if d != nil && d.dashboard != nil && d.dashboard.Active() && !d.debug {
		d.dashboard.Start()
	}
}

func (d *runtimeDisplay) Close() error {
	if d == nil {
		return nil
	}
	if d.dashboard != nil && d.dashboard.Active() && !d.debug {
		d.dashboard.Stop()
	}
	if d.logger != nil {
		return d.logger.Close()
	}
	return nil
}

func (d *runtimeDisplay) Event(kind, msg string) {
	if d == nil {
		return
	}
	if d.dashboard != nil && d.dashboard.Active() && !d.debug {
		d.dashboard.AddEvent(kind, msg)
		return
	}
	fmt.Fprintf(d.out, "mars: %s\n", msg)
}

func (d *runtimeDisplay) Error(msg string) {
	if d == nil {
		return
	}
	if d.dashboard != nil && d.dashboard.Active() && !d.debug {
		d.dashboard.AddWarning(msg)
		return
	}
	fmt.Fprintf(d.out, "mars: error: %s\n", msg)
}

func loadRunProfile(repoRoot, roleName string, sourceFoundationRole bool) (*bundle.Manifest, bundle.RoleConfig, string, []guardrails.Rule, []bundle.KnowledgeRoute, []bundle.SkillDef, error) {
	if sourceFoundationRole {
		return loadSourceFoundationRunProfile(repoRoot)
	}

	manifest, err := bundle.Load(repoRoot)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, err
	}
	role, ok := manifest.Roles[roleName]
	if !ok {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, fmt.Errorf("role %q not found in manifest; check .harness/manifest.yaml", roleName)
	}
	rolePrompt, err := manifest.RolePrompt(repoRoot, roleName)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, err
	}
	guardRules, err := manifest.LoadGuardrails(repoRoot, roleName)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, err
	}
	knowledgeDefs, err := manifest.LoadKnowledgeRoutes(repoRoot, roleName)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, err
	}
	skillDefs, err := bundle.LoadSkills(repoRoot, roleName)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, fmt.Errorf("load skills: %w", err)
	}
	return manifest, role, rolePrompt, guardRules, knowledgeDefs, skillDefs, nil
}

func loadSourceFoundationRunProfile(repoRoot string) (*bundle.Manifest, bundle.RoleConfig, string, []guardrails.Rule, []bundle.KnowledgeRoute, []bundle.SkillDef, error) {
	role := sourceFoundationRoleConfig()
	manifest := &bundle.Manifest{
		Name:              "mars-foundation",
		Description:       "Source-only foundation operating model for mars maintainers",
		OrchestrationMode: "dispatch",
		Roles: map[string]bundle.RoleConfig{
			foundationMaintainerRoleName: role,
		},
	}

	promptPath := filepath.Join(repoRoot, "docs", "roles", "personas", "foundation-maintainer.md")
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, fmt.Errorf("run: read source-only foundation role packet %s: %w", promptPath, err)
	}
	rolePrompt := strings.TrimSpace(string(prompt))
	if rolePrompt == "" {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, fmt.Errorf("run: source-only foundation role packet %s is empty", promptPath)
	}

	skillDefs, err := bundle.LoadSkills(repoRoot, foundationMaintainerRoleName)
	if err != nil {
		return nil, bundle.RoleConfig{}, "", nil, nil, nil, fmt.Errorf("load source foundation skills: %w", err)
	}
	return manifest, role, rolePrompt, sourceFoundationGuardrails(), sourceFoundationKnowledgeRoutes(), skillDefs, nil
}

func sourceFoundationRoleConfig() bundle.RoleConfig {
	return bundle.RoleConfig{
		Prompt:     "docs/roles/personas/foundation-maintainer.md",
		Domain:     "maintainer",
		Mode:       "foundation-build",
		Model:      "reasoning",
		TrustLevel: string(trust.LevelContributor),
		Tools: []string{
			"file_read", "file_write", "shell_exec", "dependency_sync", "mars_cli",
			"grep", "code_index", "code_search", "code_snippet", "code_trace",
			"code_impact", "workspace_hygiene", "github_auth_check", "record_decision",
			"ticket_create", "tool_create", "persona_create", "task_trace_summarize",
			"docsync_audit", "git_status", "git_diff", "git_commit", "git_push",
			"job_disposition_record", "release_orchestrate", "github_release_status",
			"git_release_guard",
		},
		MaxTurns: 50,
	}
}

func sourceFoundationGuardrails() []guardrails.Rule {
	now := time.Now().UTC()
	return []guardrails.Rule{
		{
			ID:        "foundation-classify-ownership",
			Name:      "Classify Foundation And Deployed Ownership",
			Severity:  guardrails.SeverityAdvisory,
			Scope:     foundationMaintainerRoleName,
			Message:   "Before proposing or applying changes, classify each finding as foundation-owned, deployed-owned, mirrored doctrine, or evidence-only.",
			CreatedAt: now,
		},
		{
			ID:        "foundation-no-demo-doctrine",
			Name:      "Validation Evidence Is Not Product Doctrine",
			Severity:  guardrails.SeverityAdvisory,
			Scope:     foundationMaintainerRoleName,
			Message:   "Use validation projects as evidence only; generalize reusable rules before changing foundation docs, generated target defaults, tools, or role guidance.",
			CreatedAt: now,
		},
		{
			ID:        "foundation-release-discipline",
			Name:      "Foundation Release Discipline",
			Severity:  guardrails.SeverityAdvisory,
			Scope:     foundationMaintainerRoleName,
			Message:   "Follow the active F-018 plan before changing source release state. During T-065 through T-067, validate and push bounded semantic checkpoints without release-note commits or version changes, run only the pinned publication-disabled snapshot, and do not tag, upload, sign, announce, or publish.",
			CreatedAt: now,
		},
	}
}

func sourceFoundationKnowledgeRoutes() []bundle.KnowledgeRoute {
	return []bundle.KnowledgeRoute{
		{When: "foundation mode, repo rules, client adapters, or source work", Paths: "AGENTS.md, docs/roles/personas/foundation-maintainer.md, docs/design-docs/harness-glossary.md"},
		{When: "role domains, role modes, source-only roles, or role registry", Paths: "docs/design-docs/harness-operating-model.md, docs/roles/ROLES.md"},
		{When: "foundation/deployed ownership, live feedback, validation loops, or doctrine drift", Paths: "docs/design-docs/foundation-deployed-harness-architecture.md, docs/design-docs/delivery-operating-model.md, docs/design-docs/dogfood-matrix.md"},
		{When: "documentation sync, code documentation metadata, or generated target guidance", Paths: "docs/design-docs/documentation-sync-architecture.md, docs/design-docs/code-documentation-map.md, docs/design-docs/cli-tool-skill-sync.md"},
		{When: "release notes, tags, GitHub releases, assets, or update flow", Paths: "docs/design-docs/release-versioning.md, .harness/skills/release-publication/SKILL.md"},
		{When: "active work, tickets, goals, or execution plans", Paths: "docs/goals/active.md, docs/exec-plans/active/current-operating-plan.md, docs/tickets/README.md, docs/tickets/backlog/"},
	}
}

func isMarsSourceRepo(repoRoot string) bool {
	goMod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil || !goModDeclaresModule(string(goMod), "github.com/greaveselliott/mars") {
		return false
	}
	for _, rel := range []string{
		filepath.Join("cmd", "mars", "main.go"),
		filepath.Join("internal", "scanner", "init.go"),
		"AGENTS.md",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
			return false
		}
	}
	return true
}

func goModDeclaresModule(text, modulePath string) bool {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" && fields[1] == modulePath {
			return true
		}
	}
	return false
}

func resolveCodeIntelRuntime(flagValue string, cfg config.Config) (codeintel.Runtime, error) {
	if raw := strings.TrimSpace(flagValue); raw != "" {
		enabled, err := parseCodeIntelBool(raw)
		if err != nil {
			return codeintel.Runtime{}, err
		}
		return codeintel.NewRuntime(enabled, "flag"), nil
	}
	if raw := strings.TrimSpace(config.Env("MARS_CODE_INTEL_ENABLED")); raw != "" {
		enabled, err := parseCodeIntelBool(raw)
		if err != nil {
			return codeintel.Runtime{}, err
		}
		return codeintel.NewRuntime(enabled, "env"), nil
	}
	return codeintel.NewRuntime(cfg.CodeIntel.Enabled, "config"), nil
}

func parseCodeIntelBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on", "enabled":
		return true, nil
	case "0", "false", "no", "off", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("code-intel: expected true or false, got %q", raw)
	}
}

func buildRoleCodeGraphContext(repoRoot string, roleTools []string, runtime codeintel.Runtime) (codeintel.ContextResult, error) {
	if !runtime.Enabled || !codeintel.ToolAllowed(roleTools) {
		return codeintel.ContextResult{}, nil
	}
	result, err := codeintel.BuildContext(context.Background(), repoRoot, codeintel.ContextOptions{Refresh: true})
	if err != nil {
		return codeintel.ContextResult{}, err
	}
	return result, nil
}

func codeGraphRunPreflight(roleTools []string, graph codeintel.ContextResult, graphErr error, runtime codeintel.Runtime) []agent.PreflightToolCall {
	if !runtime.Enabled || graphErr != nil || graph.Status.Status != codeintel.FreshnessFresh || !tools.Allowlisted("code_impact", roleTools) {
		return nil
	}
	args, ok := codeintel.ImpactPreflightArgs(graph, 0)
	if !ok {
		return nil
	}
	return []agent.PreflightToolCall{{
		Name:      "code_impact",
		ArgsJSON:  args,
		Rationale: "Mars code graph preflight: inspect changed files, likely tests, docs, feature contracts, and tickets before broad repository exploration.",
	}}
}

func codeGraphRuntimeForTrace(runtime codeintel.Runtime, roleTools []string, graphErr error) codeintel.Runtime {
	if !runtime.Enabled {
		return runtime
	}
	if !codeintel.ToolAllowed(roleTools) {
		runtime.Mode = codeintel.ModeDisabled
		return runtime
	}
	if graphErr != nil {
		runtime.Mode = codeintel.ModeUnavailable
	}
	return runtime
}

func codeGraphRunMaintenanceEnabled(roleTools []string, graph codeintel.ContextResult, graphErr error) bool {
	return graphErr == nil && graph.Status.Status == codeintel.FreshnessFresh && tools.Allowlisted("code_index", roleTools)
}

func executeRun(opts runOpts) error {
	absRepo, err := filepath.Abs(opts.repoPath)
	if err != nil {
		tw := ui.NewTraceWriter(os.Stdout, false, false)
		tw.WriteError(err.Error())
		return fmt.Errorf("run: resolve repo path: %w", err)
	}
	if err := validateRuntimeArtifactPathOutsideRepo("run", "--log-file", opts.logFile, absRepo, "default ~/.mars/traces/logs/<timestamp>-run.log"); err != nil {
		return err
	}

	sourceFoundationRole := opts.roleName == foundationMaintainerRoleName
	if sourceFoundationRole && !isMarsSourceRepo(absRepo) {
		return fmt.Errorf("run: role %q is source-only for the mars foundation repo; %s is not a mars source checkout. Use a generated target role for deployed harness work, or rerun from the mars source repository", foundationMaintainerRoleName, absRepo)
	}

	manifestPath := filepath.Join(absRepo, ".harness", "manifest.yaml")
	if opts.noInit && !sourceFoundationRole {
		if _, err := os.Stat(manifestPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("run: inspect harness manifest at %s: %w", manifestPath, err)
			}
			msg := fmt.Sprintf("run: .harness/manifest.yaml is missing in %s and --no-init was set; no files were written. Run `mars init --repo %s` to scaffold the target, or rerun without --no-init when initialization is intended.", absRepo, absRepo)
			if opts.dryRun {
				fmt.Println("── dry-run: observer-safe no-init ──")
				fmt.Println(msg)
				fmt.Printf("Role: %s\n", opts.roleName)
				fmt.Println("── end dry-run ──")
				return nil
			}
			return errors.New(msg)
		}
	}

	debug := opts.debug || opts.trace
	display, err := newRuntimeDisplay("run", opts.logFile, debug, os.Stdout, os.Stderr, nil, ui.DashboardOptions{
		Title:    "MARS",
		RepoPath: absRepo,
		Controls: "Ctrl+C cancel",
	})
	if err != nil {
		return err
	}
	display.Start()
	defer display.Close()
	cfg, cfgErr := config.Load(config.DefaultPath())
	if cfgErr != nil {
		slog.Warn("config load failed, using defaults", "err", cfgErr)
		cfg = config.Defaults()
	}
	codeIntelRuntime, err := resolveCodeIntelRuntime(opts.codeIntelFlag, cfg)
	if err != nil {
		return err
	}
	tw := display.jobViews.NewJobView(ui.JobViewMeta{
		JobID:    fmt.Sprintf("run-%s", opts.roleName),
		RepoID:   absRepo,
		RepoPath: absRepo,
		Role:     opts.roleName,
	})

	if !opts.noInit && !sourceFoundationRole {
		preInitChanges, err := gitChangedPaths(absRepo)
		if err != nil {
			tw.WriteError(fmt.Sprintf("inspect pre-init git status: %v", err))
			return err
		}

		didInit, err := scanner.EnsureHarness(absRepo, false)
		if err != nil {
			tw.WriteError(err.Error())
			return err
		}
		if didInit {
			tw.WriteAssistant("Auto-initialised .harness/ with default pipeline — continuing.")
			committed, err := commitGeneratedHarnessBaseline(absRepo, preInitChanges)
			if err != nil {
				tw.WriteError(fmt.Sprintf("commit generated harness baseline: %v", err))
				return err
			}
			if committed {
				tw.WriteAssistant("Committed generated harness baseline so the role starts from a clean scaffold.")
			}
		}
	}

	manifest, role, rolePrompt, guardRules, knowledgeDefs, skillDefs, err := loadRunProfile(absRepo, opts.roleName, sourceFoundationRole)
	if err != nil {
		tw.WriteError(err.Error())
		return err
	}

	handoff := manifest.DisplayHandoff(opts.roleName)
	tw.WriteHeader(opts.roleName, role.Model, role.Tools, handoff)

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
	var knowledgeRoutes []ctx.KnowledgeRoute
	for _, kr := range knowledgeDefs {
		knowledgeRoutes = append(knowledgeRoutes, ctx.KnowledgeRoute{When: kr.When, Paths: kr.Paths})
	}
	var skills []ctx.Skill
	for _, sd := range skillDefs {
		skills = append(skills, ctx.Skill{Name: sd.Name, Scope: sd.Scope, Body: sd.Body})
	}

	codeGraph, codeGraphErr := buildRoleCodeGraphContext(absRepo, role.Tools, codeIntelRuntime)
	codeGraphContext := codeGraph.Text
	if codeGraphErr != nil {
		tw.WriteAssistant(fmt.Sprintf("Code graph context unavailable: %v", codeGraphErr))
		codeGraphContext = codeintel.UnavailableContext(codeGraphErr)
	}

	assemblyInput := ctx.Input{
		RoleScope:        opts.roleName,
		RolePrompt:       rolePrompt,
		Guardrails:       promptGuardrails,
		KnowledgeRoutes:  knowledgeRoutes,
		Skills:           skills,
		CodeGraphContext: codeGraphContext,
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
		_ = display.Close()
		display.dashboard = nil
		display.logger = nil
		fmt.Println("── dry-run: assembled system prompt ──")
		fmt.Println(systemPrompt)
		fmt.Println("── end system prompt ──")
		fmt.Printf("\nRole: %s | Tools: %v\n", opts.roleName, role.Tools)
		return nil
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	endpoint := opts.modelEndpoint
	modelName := role.Model
	provider := models.ProviderOpenAICompatible
	apiKey := ""
	var router *inference.Router

	if endpoint == "" {
		override, ok, overrideErr := models.ResolveModelOverride(absRepo, opts.roleName, role.Model)
		if overrideErr != nil {
			tw.WriteError(fmt.Sprintf("model override failed: %v", overrideErr))
			return overrideErr
		}
		if ok {
			switch override.Routing {
			case models.RoutingCloud:
				route, err := models.ResolveProviderRoute(absRepo, models.ProviderRoute{
					Routing:   models.RoutingCloud,
					Provider:  override.Provider,
					Model:     override.Model,
					Endpoint:  override.Endpoint,
					APIKeyEnv: override.APIKeyEnv,
				})
				if err != nil {
					tw.WriteError(fmt.Sprintf("model route failed: %v", err))
					return err
				}
				endpoint = route.Endpoint
				modelName = route.Model
				provider = route.Provider
				apiKey = route.APIKey
				slog.Info("model override selected",
					"role", opts.roleName,
					"provider", provider,
					"model", modelName,
					"endpoint", endpoint,
				)
			case models.RoutingLocal:
				var clientCfg llm.Config
				router, clientCfg, err = autoStartInference(sigCtx, opts.roleName, role.Model, override.LocalBundle)
				if err != nil {
					tw.WriteError(fmt.Sprintf("inference startup failed: %v", err))
					return err
				}
				defer router.StopAll()
				endpoint = clientCfg.BaseURL
				modelName = clientCfg.Model
				provider = clientCfg.Provider
				apiKey = clientCfg.APIKey
			case models.RoutingDefer:
				return fmt.Errorf("run: model routing is deferred — configure local or cloud routing before running role %q", opts.roleName)
			default:
				return fmt.Errorf("run: unsupported model routing %q", override.Routing)
			}
		} else {
			var clientCfg llm.Config
			router, clientCfg, err = autoStartInference(sigCtx, opts.roleName, role.Model, models.LocalBundleAuto)
			if err != nil {
				tw.WriteError(fmt.Sprintf("inference startup failed: %v", err))
				return err
			}
			defer router.StopAll()
			endpoint = clientCfg.BaseURL
			modelName = clientCfg.Model
			provider = clientCfg.Provider
			apiKey = clientCfg.APIKey
		}
	}

	tw.WriteReady()

	client, err := llm.NewClient(llm.Config{
		BaseURL:  endpoint,
		Model:    modelName,
		Provider: provider,
		APIKey:   apiKey,
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
		ToolCounts:   map[string]int{},
	}
	if codeIntelRuntime.Enabled && codeintel.ToolAllowed(role.Tools) {
		codeintel.RecordContextCounters(executor.Session.ToolCounts, codeGraph, codeGraphErr)
	}

	root, err := tools.NewRoot(absRepo)
	if err != nil {
		tw.WriteError(fmt.Sprintf("invalid repo root: %v", err))
		return err
	}

	recorder := trace.NewRecorder(nil)
	traceDBPath := defaultDBPath(absRepo)
	if err := os.MkdirAll(filepath.Dir(traceDBPath), 0o755); err != nil {
		tw.WriteError(fmt.Sprintf("trace store: %v", err))
		return fmt.Errorf("run: create trace db directory: %w", err)
	}
	traceStore, err := trace.OpenStore(traceDBPath)
	if err != nil {
		tw.WriteError(fmt.Sprintf("trace store: %v", err))
		return fmt.Errorf("run: open trace store: %w", err)
	}
	defer traceStore.Close()
	traceRuntime := codeGraphRuntimeForTrace(codeIntelRuntime, role.Tools, codeGraphErr)

	result, err := agent.Run(sigCtx, agent.Params{
		Completer:           client,
		Registry:            registry,
		Executor:            executor,
		Root:                root,
		Allowlist:           role.Tools,
		SystemPrompt:        systemPrompt,
		UserMessage:         "Begin your task. Inspect the repository and take action.",
		Preflight:           codeGraphRunPreflight(role.Tools, codeGraph, codeGraphErr, codeIntelRuntime),
		MaintainCodeGraph:   codeIntelRuntime.Enabled && codeGraphRunMaintenanceEnabled(role.Tools, codeGraph, codeGraphErr),
		CodeIntelMode:       traceRuntime.Mode,
		CodeIntelModeSource: traceRuntime.Source,
		Config: agent.LoopConfig{
			Model:       modelName,
			MaxTurns:    opts.maxTurns,
			TokenBudget: opts.budget,
		},
		JobID:      fmt.Sprintf("%s-%s", manifest.Name, opts.roleName),
		Trace:      recorder,
		TraceStore: traceStore,
		UI:         tw,
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

	tw.WriteHandoff(opts.roleName, handoff)

	return nil
}

// autoStartInference loads config, detects hardware, and starts llama-server
// for the requested role via the inference Router. Returns the router (for
// cleanup) and the base URL of the running server.
func autoStartInference(ctx context.Context, roleName, modelHint, localBundle string) (*inference.Router, llm.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, llm.Config{}, fmt.Errorf("cannot determine home directory: %w", err)
	}
	baseDir := filepath.Join(home, ".mars")

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
		return nil, llm.Config{}, fmt.Errorf("llama-server not found at %s — run 'mars setup' first", binaryPath)
	}

	hw := hardware.Detect()
	bundle, _, err := models.ResolveLocalBundle(hw, localBundle)
	if err != nil {
		return nil, llm.Config{}, err
	}

	router := inference.NewRouter(inference.RouterConfig{
		BinaryPath:  binaryPath,
		Models:      bundle.Models,
		RoleMapping: inference.DefaultRoleTierMapping(),
		ModelsDir:   modelsDir,
		Tuning:      inferenceTuningFromConfig(cfg),
	})

	clientCfg, err := router.ClientConfigForRoleModel(ctx, roleName, modelHint)
	if err != nil {
		return nil, llm.Config{}, err
	}

	slog.Info("inference ready", "role", roleName, "endpoint", clientCfg.BaseURL, "local_bundle", bundle.ID)
	return router, clientCfg, nil
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
		skipDownload  bool
		download      bool
		skipGitHub    bool
		enableGitHub  bool
		testMode      bool
		dryRun        bool
		installDir    string
		inferenceMode string
		localBundle   string
		yes           bool
		jsonOut       bool
		plain         bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "First-time setup wizard",
		Long:  "Create ~/.mars/, detect hardware, install local inference, and download pinned models. GitHub integration is optional.",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer silenceLogsForJSON(jsonOut)()
			result, err := setup.Run(setup.Config{
				SkipDownload: skipDownload,
				SkipGitHub:   skipGitHub,
				EnableGitHub: enableGitHub,
				TestMode:     testMode,
				DryRun:       dryRun,
				InstallDir:   installDir,
				Inference:    inferenceMode,
				LocalBundle:  localBundle,
			})
			if err != nil {
				if jsonOut {
					return writeJSONError(err)
				}
				return err
			}
			_ = download
			_ = yes
			_ = plain
			if jsonOut {
				return writeJSON(os.Stdout, map[string]any{
					"status":        "ok",
					"steps_run":     result.StepsRun,
					"steps_skipped": result.StepsSkipped,
				})
			}
			fmt.Printf("Setup complete: %d steps run, %d skipped\n", result.StepsRun, result.StepsSkipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&skipDownload, "skip-download", false, "Skip model download")
	cmd.Flags().BoolVar(&download, "download", false, "Download selected local model bundle artifacts")
	cmd.Flags().BoolVar(&skipGitHub, "skip-github", false, "Skip GitHub private release auth and optional GitHub integration checks")
	cmd.Flags().BoolVar(&enableGitHub, "github", false, "Configure optional GitHub status/check integration")
	cmd.Flags().BoolVar(&testMode, "test-mode", false, "Skip downloads and external services")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print steps without executing")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Directory containing mars for shell PATH setup; default resolves automatically")
	cmd.Flags().StringVar(&inferenceMode, "inference", models.RoutingLocal, "Inference mode: local, cloud, or defer")
	cmd.Flags().StringVar(&localBundle, "local-bundle", models.LocalBundleAuto, "Local bundle: auto, local-cpu-q3, local-balanced-q4, or local-quality-q8")
	cmd.Flags().BoolVar(&yes, "yes", false, "Do not prompt; fail with remediation when required input is missing")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	cmd.Flags().BoolVar(&plain, "plain", false, "Disable styling and animation")

	return cmd
}

func initCmd() *cobra.Command {
	var (
		repoPath      string
		force         bool
		modelRouting  string
		localBundle   string
		cloudProvider string
		cloudModel    string
		cloudEndpoint string
		apiKeyEnv     string
		yes           bool
		jsonOut       bool
		plain         bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold .harness/ in a repository",
		Long:  "Create the .harness/ directory with manifest.yaml, roles/, guardrails/, and knowledge/ subdirectories.",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer silenceLogsForJSON(jsonOut)()
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("init: cannot determine working directory: %w", err)
				}
			}
			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("init: resolve path: %w", err)
			}
			preInitChanges, err := gitChangedPaths(absPath)
			if err != nil {
				return fmt.Errorf("init: inspect pre-init git status: %w", err)
			}
			if err := scanner.Init(absPath, force); err != nil {
				if jsonOut {
					return writeJSONError(err)
				}
				return err
			}
			routePath := ""
			if strings.TrimSpace(modelRouting) != "" {
				path, err := writeInitModelRouting(absPath, modelRouting, localBundle, cloudProvider, cloudModel, cloudEndpoint, apiKeyEnv)
				if err != nil {
					if jsonOut {
						return writeJSONError(err)
					}
					return err
				}
				routePath = path
			}
			_ = yes
			_ = plain
			out := cmd.OutOrStdout()
			if !jsonOut {
				fmt.Fprintf(out, "Initialized .harness/ in %s\n", absPath)
				if routePath != "" {
					fmt.Fprintf(out, "Wrote model routing in %s\n", routePath)
				}
			}
			committed, err := commitGeneratedHarnessBaseline(absPath, preInitChanges)
			if err != nil {
				return fmt.Errorf("init: commit generated harness baseline: %w", err)
			}
			if jsonOut {
				return writeJSON(os.Stdout, map[string]any{
					"status":             "ok",
					"repo":               absPath,
					"model_routing_path": routePath,
					"committed":          committed,
				})
			}
			if committed {
				fmt.Fprintf(out, "Committed generated harness baseline in %s\n", absPath)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "Refresh missing scaffold and rewrite manifest if .harness/ already exists")
	cmd.Flags().StringVar(&modelRouting, "model-routing", "", "Model routing mode: local, cloud, or defer")
	cmd.Flags().StringVar(&localBundle, "local-bundle", models.LocalBundleAuto, "Local bundle: auto, local-cpu-q3, local-balanced-q4, or local-quality-q8")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider: openai, anthropic, gemini, mistral, xai, deepseek, groq, cohere, or openai-compatible")
	cmd.Flags().StringVar(&cloudModel, "cloud-model", "", "Cloud provider model name")
	cmd.Flags().StringVar(&cloudEndpoint, "cloud-endpoint", "", "Cloud provider endpoint; required for openai-compatible custom routes")
	cmd.Flags().StringVar(&apiKeyEnv, "api-key-env", "", "Environment variable containing the provider API key")
	cmd.Flags().BoolVar(&yes, "yes", false, "Do not prompt; fail with remediation when required input is missing")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	cmd.Flags().BoolVar(&plain, "plain", false, "Disable styling and animation")

	return cmd
}

func writeInitModelRouting(repoRoot, routing, localBundle, cloudProvider, cloudModel, cloudEndpoint, apiKeyEnv string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(routing)) {
	case models.RoutingLocal:
		if _, _, err := models.ResolveLocalBundle(hardware.Detect(), localBundle); err != nil {
			return "", fmt.Errorf("init: %w", err)
		}
		return models.SetDefaultModelRouting(repoRoot, models.ModelOverride{
			Routing:     models.RoutingLocal,
			LocalBundle: localBundle,
			Reason:      "configured by mars init",
		})
	case models.RoutingCloud:
		if strings.TrimSpace(cloudProvider) == "" {
			return "", fmt.Errorf("init: --cloud-provider is required when --model-routing cloud")
		}
		if strings.TrimSpace(cloudModel) == "" {
			return "", fmt.Errorf("init: --cloud-model is required when --model-routing cloud")
		}
		path, err := models.SetDefaultModelRouting(repoRoot, models.ModelOverride{
			Routing:   models.RoutingCloud,
			Provider:  cloudProvider,
			Model:     cloudModel,
			Endpoint:  cloudEndpoint,
			APIKeyEnv: apiKeyEnv,
			Reason:    "configured by mars init",
		})
		if err != nil {
			return "", err
		}
		envName := strings.TrimSpace(apiKeyEnv)
		if envName == "" {
			if spec, ok := models.ProviderSpecByName(cloudProvider); ok {
				envName = spec.DefaultAPIKeyEnv
			}
		}
		if _, err := models.EnsureEnvExample(repoRoot, envName); err != nil {
			return "", err
		}
		return path, nil
	case models.RoutingDefer:
		return models.SetDefaultModelRouting(repoRoot, models.ModelOverride{
			Routing: models.RoutingDefer,
			Reason:  "model routing deferred by mars init",
		})
	default:
		return "", fmt.Errorf("init: unsupported --model-routing %q — use local, cloud, or defer", routing)
	}
}

type ejectDBResult struct {
	DBPath       string
	Removed      []string
	Missing      []string
	Pruned       []string
	Unregistered bool
	KeptShared   bool
}

func ejectCmd() *cobra.Command {
	var (
		repoPath       string
		dbPath         string
		apply          bool
		confirm        string
		keepDB         bool
		deleteSharedDB bool
	)

	cmd := &cobra.Command{
		Use:     "eject",
		Aliases: []string{"kill-switch", "uninstall"},
		Short:   "Remove MARS from a target repo",
		Long: `Remove the deployed MARS surface from a target repository and
delete its associated per-repo SQLite database. The command is a dry run by
default; destructive removal requires --apply and --confirm <repo-name>.

This removes working-tree harness traces such as .harness/, generated planning
docs, tickets, feature contracts, AGENTS.md, VERSION, and CHANGELOG.md. It does
not rewrite git history.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("eject: cannot determine working directory: %w", err)
				}
			}
			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("eject: resolve path: %w", err)
			}
			repoName := filepath.Base(absPath)
			if apply && strings.TrimSpace(confirm) != repoName {
				return fmt.Errorf("eject: destructive apply requires --confirm %s", repoName)
			}

			result, err := scanner.Eject(absPath, scanner.EjectOptions{Apply: apply})
			if err != nil {
				return err
			}
			dbResult := ejectDBResult{}
			if !keepDB {
				dbResult, err = ejectDatabase(cmd.Context(), absPath, dbPath, apply, deleteSharedDB)
				if err != nil {
					return err
				}
			}

			printEjectResult(cmd.OutOrStdout(), result, dbResult, apply, keepDB, repoName)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to associated SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().BoolVar(&apply, "apply", false, "Actually remove files and database artifacts; default is dry-run")
	cmd.Flags().StringVar(&confirm, "confirm", "", "Required with --apply; must equal the target repo directory name")
	cmd.Flags().BoolVar(&keepDB, "keep-db", false, "Remove repo files but leave the associated database untouched")
	cmd.Flags().BoolVar(&deleteSharedDB, "delete-shared-db", false, "Allow deleting the legacy shared ~/.mars/db/mars.db database")
	return cmd
}

func ejectDatabase(ctx context.Context, repoAbs, dbPath string, apply, deleteSharedDB bool) (ejectDBResult, error) {
	if dbPath == "" {
		dbPath = defaultDBPath(repoAbs)
	}
	absDB, err := filepath.Abs(dbPath)
	if err != nil {
		return ejectDBResult{}, fmt.Errorf("eject: resolve db path: %w", err)
	}
	result := ejectDBResult{DBPath: absDB}

	legacyAbs, _ := filepath.Abs(legacyDBPath())
	if filepath.Clean(absDB) == filepath.Clean(legacyAbs) && !deleteSharedDB {
		result.KeptShared = true
		if apply {
			removed, err := unregisterRepoFromDB(ctx, repoAbs, absDB)
			if err != nil {
				return result, err
			}
			result.Unregistered = removed
		}
		return result, nil
	}

	for _, path := range []string{absDB, absDB + "-shm", absDB + "-wal"} {
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			result.Missing = append(result.Missing, path)
			continue
		} else if err != nil {
			return result, fmt.Errorf("eject: inspect database artifact %s: %w", path, err)
		}
		result.Removed = append(result.Removed, path)
		if apply {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return result, fmt.Errorf("eject: remove database artifact %s: %w", path, err)
			}
		}
	}

	if apply {
		dir := filepath.Dir(absDB)
		pruned, err := removeDirIfEmpty(dir)
		if err != nil {
			return result, fmt.Errorf("eject: prune database directory %s: %w", dir, err)
		}
		if pruned {
			result.Pruned = append(result.Pruned, dir)
		}
	}
	return result, nil
}

func unregisterRepoFromDB(ctx context.Context, repoAbs, dbPath string) (bool, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("eject: inspect shared database %s: %w", dbPath, err)
	}
	db, err := openDB(dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()
	registry, err := serve.NewRepoRegistry(db)
	if err != nil {
		return false, err
	}
	repos, err := registry.List(ctx)
	if err != nil {
		return false, err
	}
	for _, rec := range repos {
		if rec.Path == repoAbs {
			return true, registry.Remove(ctx, rec.ID)
		}
	}
	return false, nil
}

func removeDirIfEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(entries) > 0 {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}

func printEjectResult(w io.Writer, result scanner.EjectResult, dbResult ejectDBResult, apply, keepDB bool, repoName string) {
	mode := "dry-run"
	action := "Would remove"
	if apply {
		mode = "applied"
		action = "Removed"
	}
	fmt.Fprintf(w, "MARS eject %s for %s\n", mode, result.RepoRoot)
	fmt.Fprintf(w, "\nRepo artifacts\n")
	printPathList(w, action, result.Removed)
	if len(result.Pruned) > 0 {
		printPathList(w, "Pruned empty directory", result.Pruned)
	}
	if len(result.Missing) > 0 {
		printPathList(w, "Already absent", result.Missing)
	}

	fmt.Fprintf(w, "\nDatabase artifacts\n")
	if keepDB {
		fmt.Fprintln(w, "  Kept database because --keep-db was set")
	} else if dbResult.KeptShared {
		fmt.Fprintf(w, "  Kept shared database %s; pass --delete-shared-db to delete it\n", dbResult.DBPath)
		if apply && dbResult.Unregistered {
			fmt.Fprintln(w, "  Removed this repo from the shared registry")
		}
	} else {
		printPathList(w, action, dbResult.Removed)
		if len(dbResult.Pruned) > 0 {
			printPathList(w, "Pruned empty directory", dbResult.Pruned)
		}
		if len(dbResult.Missing) > 0 {
			printPathList(w, "Already absent", dbResult.Missing)
		}
	}

	if !apply {
		fmt.Fprintf(w, "\nRun with --apply --confirm %s to delete these artifacts.\n", repoName)
	}
}

func printPathList(w io.Writer, label string, paths []string) {
	if len(paths) == 0 {
		fmt.Fprintf(w, "  %s: none\n", label)
		return
	}
	for _, path := range paths {
		fmt.Fprintf(w, "  %s: %s\n", label, path)
	}
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

If .harness/manifest.yaml is missing, mars scaffolds it first (same as init; requires git).`,
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
			preInitChanges, err := gitChangedPaths(absPath)
			if err != nil {
				return fmt.Errorf("scan: inspect pre-init git status: %w", err)
			}
			didInit, err := scanner.EnsureHarness(absPath, false)
			if err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			if didInit {
				committed, err := commitGeneratedHarnessBaseline(absPath, preInitChanges)
				if err != nil {
					return fmt.Errorf("scan: commit generated harness baseline: %w", err)
				}
				if committed {
					fmt.Printf("Committed generated harness baseline in %s\n", absPath)
				}
			}

			parentCtx := cmd.Context()
			if parentCtx == nil {
				parentCtx = context.Background()
			}
			sigCtx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
			defer stop()

			result, err := scanner.Scan(sigCtx, scanner.Config{RepoRoot: absPath})
			if err != nil {
				return err
			}

			fmt.Printf("Language: %s | Framework: %s\n", result.Language, result.Framework)
			fmt.Printf("CI: %v | Tests: %v | README: %v | LICENSE: %v\n",
				result.HasCI, result.HasTests, result.HasReadme, result.HasLicense)
			cfg, cfgErr := config.Load(config.DefaultPath())
			if cfgErr != nil {
				slog.Warn("config load failed, using defaults", "err", cfgErr)
				cfg = config.Defaults()
			}
			codeIntelRuntime, runtimeErr := resolveCodeIntelRuntime("", cfg)
			if runtimeErr != nil {
				return runtimeErr
			}
			if !codeIntelRuntime.Enabled {
				fmt.Printf("Code intel: disabled\n")
			} else if status, err := codeIntelScanStatus(sigCtx, absPath); err == nil {
				fmt.Printf("Code intel: %s | Files: %d | Stale: %d | New: %d | Symbols: %d | Edges: %d\n",
					status.Status, status.Files, status.StaleFiles, status.NewFiles, status.Symbols, status.Edges)
				if status.Message != "" {
					fmt.Printf("Code intel message: %s\n", status.Message)
				}
			} else {
				fmt.Printf("Code intel: unavailable (%v)\n", err)
			}
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

func codeIntelScanStatus(ctx context.Context, repoPath string) (codeintel.Status, error) {
	store, err := codeintel.Open(repoPath, "")
	if err != nil {
		return codeintel.Status{}, err
	}
	defer store.Close()
	return store.Status(ctx)
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
				fmt.Println("MARS Doctor")
				fmt.Println("───────────────────")
				fmt.Print(doctor.FormatText(results))
			}

			if doctor.HasFailures(results) {
				return fmt.Errorf("doctor: one or more checks failed — see above for remediation")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.mars/config.yaml)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to database file (default: ~/.mars/db/{repo}/mars.db)")
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
				if isUnavailableDatabaseError(err) {
					fmt.Fprintf(cmd.OutOrStdout(), "No scores recorded yet. %s\n", databaseEvidenceRemediation(path, err))
					return nil
				}
				return err
			}
			defer store.Close()
			scores, err := store.ListScores(cmd.Context())
			if err != nil {
				return err
			}
			if len(scores) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No scores recorded yet.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-18s %-8s %-7s %-8s %s\n", "REPO", "ROLE", "SCORE", "SAMPLES", "WINDOW", "COMPUTED")
			for _, sc := range scores {
				fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-18s %-8.2f %-7d %-8d %s\n",
					sc.RepoID, sc.Role, sc.Value, sc.SampleSize, sc.WindowDays, sc.ComputedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Target repository path (default: shared legacy database)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	cmd.AddCommand(scoresExportCmd())
	return cmd
}

func scoresExportCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	var windowDays int
	var createInterventionDebt bool
	var noTicket bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Refresh docs/QUALITY_SCORE.md from live evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoAbs, resolvedDB, repoID, err := resolveQualityExportPaths(repoPath, dbPath)
			if err != nil {
				return err
			}
			report, err := qualityscore.Export(cmd.Context(), qualityscore.Options{
				RepoPath:               repoAbs,
				RepoID:                 repoID,
				DBPath:                 resolvedDB,
				WindowDays:             windowDays,
				CreateInterventionDebt: createInterventionDebt,
				DisableTicketCreation:  noTicket,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Exported quality score to %s\n", report.Path)
			fmt.Printf("Overall grade: %s\n", report.Grade)
			for _, warning := range report.Warnings {
				fmt.Printf("Warning: %s\n", warning)
			}
			for _, ticket := range report.TicketsChanged {
				fmt.Printf("%s\n", ticket)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Target repository path")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().IntVar(&windowDays, "window-days", 30, "Scoring and telemetry evidence window in days")
	cmd.Flags().BoolVar(&createInterventionDebt, "create-intervention-debt", false, "Create or update deduped intervention-debt tickets from score and outcome evidence")
	cmd.Flags().BoolVar(&noTicket, "no-ticket", false, "Deprecated: ticket creation is disabled by default unless --create-intervention-debt is set")
	_ = cmd.Flags().MarkHidden("no-ticket")
	return cmd
}

func codeIntelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code-intel",
		Short: "Measure and benchmark Mars code graph assistance",
		Long:  "Inspect local code-intel evidence from Mars SQLite traces and run model-free graph benchmarks.",
	}
	cmd.AddCommand(codeIntelMetricsCmd())
	cmd.AddCommand(codeIntelBenchmarkCmd())
	return cmd
}

func codeIntelMetricsCmd() *cobra.Command {
	var (
		repoPath   string
		dbPath     string
		windowDays int
		jsonOut    bool
	)
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Summarize persisted local code-intel efficiency evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(repoPath) == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("code-intel metrics: cannot determine working directory: %w", err)
				}
			}
			absRepo, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("code-intel metrics: resolve repo path: %w", err)
			}
			if strings.TrimSpace(dbPath) == "" {
				dbPath = defaultDBPath(absRepo)
			}
			if err := validateRuntimeArtifactPathOutsideRepo("code-intel metrics", "--db", dbPath, absRepo, defaultDBPath(absRepo)); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("code-intel metrics: create db directory: %w", err)
			}
			report, err := codeintel.Metrics(cmd.Context(), codeintel.MetricsOptions{
				RepoPath:   absRepo,
				DBPath:     dbPath,
				WindowDays: windowDays,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Code-intel metrics: %s\n", report.RepoPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Window: %dd | Jobs: %d\n", report.WindowDays, report.Jobs)
			fmt.Fprintf(cmd.OutOrStdout(), "Graph jobs: enabled=%d disabled=%d unavailable=%d unused=%d\n",
				report.GraphEnabledJobs, report.GraphDisabledJobs, report.GraphUnavailableJobs, report.UnusedGraphJobs)
			fmt.Fprintf(cmd.OutOrStdout(), "Code-intel calls: %d | output_bytes=%d | context_bytes=%d | refreshes=%d\n",
				report.CodeIntelToolCalls, report.CodeIntelOutputBytes, report.CodeIntelContextBytes, report.CodeIntelRefreshes)
			fmt.Fprintf(cmd.OutOrStdout(), "Broad exploration: calls=%d output_bytes=%d bulk_reads=%d shell_searches=%d\n",
				report.BroadExplorationCalls, report.BroadExplorationBytes, report.BulkFileReadCalls, report.BroadShellSearchCalls)
			fmt.Fprintf(cmd.OutOrStdout(), "Runtime: llm_calls=%d tool_invocations=%d tokens=%d wall_ms=%d\n",
				report.LLMCalls, report.ToolInvocations, report.TokenEstimate, report.WallMs)
			if sources := codeintel.SortedModeSources(report.ModeSources); len(sources) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Mode sources: %s\n", strings.Join(sources, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().IntVar(&windowDays, "window-days", 30, "Number of days of persisted traces to aggregate")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func codeIntelBenchmarkCmd() *cobra.Command {
	var (
		repoPath      string
		dbPath        string
		caseName      string
		trials        int
		reportPath    string
		jsonOut       bool
		changedPaths  string
		expectedFiles string
		expectedTests string
		expectedDocs  string
	)
	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run a local no-model control/treatment code graph benchmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(repoPath) == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("code-intel benchmark: cannot determine working directory: %w", err)
				}
			}
			absRepo, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("code-intel benchmark: resolve repo path: %w", err)
			}
			if strings.TrimSpace(dbPath) == "" {
				dbPath = defaultDBPath(absRepo)
			}
			if err := validateRuntimeArtifactPathOutsideRepo("code-intel benchmark", "--db", dbPath, absRepo, defaultDBPath(absRepo)); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("code-intel benchmark: create db directory: %w", err)
			}
			if err := validateRuntimeArtifactPathOutsideRepo("code-intel benchmark", "--report", reportPath, absRepo, "a path outside the target repo"); err != nil {
				return err
			}
			report, err := codeintel.Benchmark(cmd.Context(), codeintel.BenchmarkOptions{
				RepoPath:      absRepo,
				DBPath:        dbPath,
				Case:          caseName,
				Trials:        trials,
				ChangedPaths:  splitCSV(changedPaths),
				ExpectedFiles: splitCSV(expectedFiles),
				ExpectedTests: splitCSV(expectedTests),
				ExpectedDocs:  splitCSV(expectedDocs),
			})
			if err != nil {
				return err
			}
			if strings.TrimSpace(reportPath) != "" {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
					return fmt.Errorf("code-intel benchmark: create report directory: %w", err)
				}
				if err := os.WriteFile(reportPath, append(data, '\n'), 0o644); err != nil {
					return fmt.Errorf("code-intel benchmark: write report: %w", err)
				}
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Code-intel benchmark: %s\n", report.RepoPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Case: %s | Trials: %d | Local only: %v\n", report.Case, report.Trials, report.LocalOnly)
			fmt.Fprintf(cmd.OutOrStdout(), "Control: avg_duration_ms=%d avg_context_bytes=%d\n",
				report.Summary.ControlAvgDurationMS, report.Summary.ControlAvgContextBytes)
			fmt.Fprintf(cmd.OutOrStdout(), "Treatment: avg_duration_ms=%d avg_context_bytes=%d\n",
				report.Summary.TreatmentAvgDurationMS, report.Summary.TreatmentAvgContextBytes)
			if report.Summary.ExpectedFilesHitRate > 0 || strings.TrimSpace(expectedFiles) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Expected files hit rate: %.2f\n", report.Summary.ExpectedFilesHitRate)
			}
			if report.Summary.ExpectedTestsHitRate > 0 || strings.TrimSpace(expectedTests) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Expected tests hit rate: %.2f\n", report.Summary.ExpectedTestsHitRate)
			}
			if report.Summary.ExpectedDocsHitRate > 0 || strings.TrimSpace(expectedDocs) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Expected docs hit rate: %.2f\n", report.Summary.ExpectedDocsHitRate)
			}
			if strings.TrimSpace(reportPath) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Report: %s\n", reportPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().StringVar(&caseName, "case", "current", "Benchmark case name")
	cmd.Flags().IntVar(&trials, "trials", 2, "Number of control/treatment trials")
	cmd.Flags().StringVar(&changedPaths, "changed-paths", "", "Comma-separated changed paths to evaluate instead of the current git diff")
	cmd.Flags().StringVar(&expectedFiles, "expected-files", "", "Comma-separated changed paths expected in impact output")
	cmd.Flags().StringVar(&expectedTests, "expected-tests", "", "Comma-separated test paths expected in impact output")
	cmd.Flags().StringVar(&expectedDocs, "expected-docs", "", "Comma-separated doc paths expected in impact output")
	cmd.Flags().StringVar(&reportPath, "report", "", "Optional JSON report path outside the target repo")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func telemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Inspect and exchange anonymous foundation telemetry",
		Long: `Inspect local telemetry state, preview anonymous aggregate reports,
enqueue reports into the local outbox, send pending reports to a configured
collector, or run a local foundation telemetry collector.`,
	}
	cmd.AddCommand(telemetryStatusCmd())
	cmd.AddCommand(telemetryPreviewCmd())
	cmd.AddCommand(telemetryExportCmd())
	cmd.AddCommand(telemetrySendCmd())
	cmd.AddCommand(telemetryCollectCmd())
	cmd.AddCommand(telemetryTriageFoundationCmd())
	return cmd
}

func telemetryStatusCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show anonymous telemetry reporting status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				return err
			}
			_, resolvedDB, _, err := resolveQualityExportPaths(repoPath, dbPath)
			if err != nil {
				return err
			}
			store, err := telemetry.OpenStore(resolvedDB)
			if err != nil {
				return err
			}
			defer store.Close()
			stats, err := store.OutboxStats(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Anonymous telemetry reporting: %s\n", defaultStringCLI(cfg.Telemetry.Reporting, "off"))
			if cfg.Telemetry.Endpoint != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Collector endpoint: %s\n", cfg.Telemetry.Endpoint)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Collector endpoint: not configured")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Local DB: %s\n", resolvedDB)
			fmt.Fprintf(cmd.OutOrStdout(), "Outbox: pending=%d sent=%d failed=%d\n", stats["pending"], stats["sent"], stats["failed"])
			if strings.TrimSpace(cfg.Telemetry.Reporting) == "" || strings.EqualFold(cfg.Telemetry.Reporting, "off") {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: disabled by default; no network calls will be made.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Target repository path")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	return cmd
}

func telemetryPreviewCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview the exact anonymous telemetry payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, _, closeFn, err := buildAnonymousTelemetryReport(cmd.Context(), repoPath, dbPath)
			if closeFn != nil {
				defer closeFn()
			}
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Target repository path")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	return cmd
}

func telemetryExportCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	var anonymous bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Enqueue an anonymous telemetry report in the local outbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !anonymous {
				return fmt.Errorf("telemetry export: pass --anonymous to export the sanitized aggregate report")
			}
			report, store, closeFn, err := buildAnonymousTelemetryReport(cmd.Context(), repoPath, dbPath)
			if closeFn != nil {
				defer closeFn()
			}
			if err != nil {
				return err
			}
			rec, err := store.EnqueueAnonymousReport(cmd.Context(), report)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Enqueued anonymous telemetry report %s (%s)\n", rec.ID, rec.PayloadHash)
			return writeJSON(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Target repository path")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().BoolVar(&anonymous, "anonymous", false, "Export sanitized anonymous aggregate telemetry")
	return cmd
}

func telemetrySendCmd() *cobra.Command {
	var repoPath string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send pending anonymous telemetry reports to the configured collector",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				return err
			}
			if !strings.EqualFold(strings.TrimSpace(cfg.Telemetry.Reporting), "anonymous") {
				fmt.Fprintln(cmd.OutOrStdout(), "Anonymous telemetry reporting is off; no network calls made.")
				return nil
			}
			if strings.TrimSpace(cfg.Telemetry.Endpoint) == "" {
				return fmt.Errorf("telemetry send: telemetry.endpoint is required when telemetry.reporting is anonymous")
			}
			report, store, closeFn, err := buildAnonymousTelemetryReport(cmd.Context(), repoPath, dbPath)
			if closeFn != nil {
				defer closeFn()
			}
			if err != nil {
				return err
			}
			if _, err := store.EnqueueAnonymousReport(cmd.Context(), report); err != nil {
				return err
			}
			pending, err := store.PendingReports(cmd.Context(), time.Now().UTC(), 25)
			if err != nil {
				return err
			}
			sent := 0
			for _, rec := range pending {
				if err := sendAnonymousTelemetryReport(cmd.Context(), cfg, rec); err != nil {
					_ = store.MarkReportFailed(cmd.Context(), rec.ID, time.Now().UTC().Add(time.Hour), err)
					fmt.Fprintf(cmd.OutOrStderr(), "Warning: send %s failed: %v\n", rec.ID, err)
					continue
				}
				if err := store.MarkReportSent(cmd.Context(), rec.ID); err != nil {
					return err
				}
				sent++
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sent %d anonymous telemetry report(s).\n", sent)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Target repository path")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	return cmd
}

func telemetryCollectCmd() *cobra.Command {
	var addr string
	var storage string
	var dbPath string
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Run a local anonymous foundation telemetry collector",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(addr) == "" {
				addr = ":9092"
			}
			switch strings.ToLower(strings.TrimSpace(storage)) {
			case "", "sqlite":
			default:
				return fmt.Errorf("telemetry collect: storage %q is not implemented in v1; use sqlite", storage)
			}
			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".mars", "db", "foundation-telemetry", "intake.db")
			}
			store, err := foundationtelemetry.OpenSQLiteStore(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()

			srv := &http.Server{Addr: addr, Handler: foundationtelemetry.Handler(store)}
			go func() {
				<-cmd.Context().Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutdownCtx)
			}()
			fmt.Fprintf(cmd.OutOrStdout(), "Foundation telemetry collector listening on %s, db=%s\n", addr, dbPath)
			err = srv.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":9092", "Collector listen address")
	cmd.Flags().StringVar(&storage, "storage", "sqlite", "Collector storage backend (v1: sqlite)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Collector SQLite database path")
	return cmd
}

func telemetryTriageFoundationCmd() *cobra.Command {
	var dbPath string
	var repoPath string
	var windowDays int
	cmd := &cobra.Command{
		Use:   "triage-foundation",
		Short: "Create MARS source tickets from repeated anonymous telemetry patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return fmt.Errorf("telemetry triage-foundation: --db is required")
			}
			if repoPath == "" {
				repoPath = "."
			}
			absRepo, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("telemetry triage-foundation: resolve repo: %w", err)
			}
			store, err := foundationtelemetry.OpenSQLiteStore(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			since := time.Now().UTC().AddDate(0, 0, -windowDays)
			patterns, err := store.PatternsSince(cmd.Context(), since)
			if err != nil {
				return err
			}
			root, err := tools.NewRoot(absRepo)
			if err != nil {
				return err
			}
			created := 0
			for _, pattern := range patterns {
				if pattern.InstallWindowCount < 2 && len(pattern.HarnessVersions) < 2 {
					continue
				}
				result, err := tools.CreateTicket(root, tools.TicketInput{
					Title:      fmt.Sprintf("Foundation telemetry: %s %s %s", pattern.Category, pattern.Target, pattern.Signature),
					Priority:   foundationTelemetryPriority(pattern.Severity),
					Complexity: "medium",
					Kind:       "intervention-debt",
					DedupeKey:  "foundation-telemetry:" + pattern.Signature,
					Source:     "foundation-telemetry:" + pattern.Signature,
					Metadata: map[string]string{
						"target":               pattern.Target,
						"category":             pattern.Category,
						"severity":             pattern.Severity,
						"signature":            pattern.Signature,
						"report_count":         fmt.Sprintf("%d", pattern.ReportCount),
						"install_window_count": fmt.Sprintf("%d", pattern.InstallWindowCount),
					},
					Body: foundationTelemetryTicketBody(pattern),
				})
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), result.Output)
				created++
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Triaged %d foundation telemetry pattern(s).\n", created)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Foundation collector SQLite database path")
	cmd.Flags().StringVar(&repoPath, "repo", ".", "MARS source repository path")
	cmd.Flags().IntVar(&windowDays, "window-days", 30, "Foundation telemetry lookback window")
	return cmd
}

func buildAnonymousTelemetryReport(ctx context.Context, repoPath, dbPath string) (foundationtelemetry.AnonymousReport, *telemetry.Store, func(), error) {
	repoAbs, resolvedDB, repoID, err := resolveQualityExportPaths(repoPath, dbPath)
	if err != nil {
		return foundationtelemetry.AnonymousReport{}, nil, nil, err
	}
	store, err := telemetry.OpenStore(resolvedDB)
	if err != nil {
		return foundationtelemetry.AnonymousReport{}, nil, nil, err
	}
	closeFn := func() { _ = store.Close() }
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		closeFn()
		return foundationtelemetry.AnonymousReport{}, nil, nil, err
	}
	interval := telemetryReportInterval(cfg)
	windowEnd := time.Now().UTC()
	windowStart := windowEnd.Add(-interval)
	seed, err := telemetry.LoadOrCreateReportKeySeed(filepath.Dir(config.DefaultPath()))
	if err != nil {
		closeFn()
		return foundationtelemetry.AnonymousReport{}, nil, nil, err
	}
	hw := hardware.Detect()
	roles, orchestrationMode := anonymousRoleMetadata(repoAbs)
	generatedVersion := ""
	if metadata, err := scanner.ReadHarnessMetadata(repoAbs); err == nil {
		generatedVersion = metadata.GeneratorVersion
	}
	report, err := store.BuildAnonymousReport(telemetry.AnonymousReportOptions{
		RepoID:                  repoID,
		ReportKeySeed:           seed,
		HarnessVersion:          version,
		GeneratedHarnessVersion: generatedVersion,
		OS:                      runtime.GOOS,
		Arch:                    runtime.GOARCH,
		HardwareTier:            string(hw.Profile),
		OrchestrationMode:       orchestrationMode,
		WindowStart:             windowStart,
		WindowEnd:               windowEnd,
		Roles:                   roles,
	})
	if err != nil {
		closeFn()
		return foundationtelemetry.AnonymousReport{}, nil, nil, err
	}
	_ = ctx
	return report, store, closeFn, nil
}

func telemetryReportInterval(cfg config.Config) time.Duration {
	interval, err := time.ParseDuration(strings.TrimSpace(cfg.Telemetry.ReportInterval))
	if err != nil || interval <= 0 {
		return 24 * time.Hour
	}
	return interval
}

func anonymousRoleMetadata(repoAbs string) (map[string]telemetry.RoleMetadata, string) {
	roles := map[string]telemetry.RoleMetadata{}
	orchestrationMode := "unknown"
	manifest, err := bundle.Load(repoAbs)
	if err != nil {
		return roles, orchestrationMode
	}
	orchestrationMode = strings.TrimSpace(manifest.OrchestrationMode)
	if orchestrationMode == "" {
		orchestrationMode = "legacy"
	}
	for name, role := range manifest.Roles {
		roles[name] = telemetry.RoleMetadata{
			Domain: role.Domain,
			Mode:   role.Mode,
		}
	}
	return roles, orchestrationMode
}

func sendAnonymousTelemetryReport(ctx context.Context, cfg config.Config, rec telemetry.OutboxRecord) error {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Telemetry.Endpoint), "/") + foundationtelemetry.ReportsPath
	body := []byte(fmt.Sprintf(`{"reports":[%s]}`, rec.PayloadJSON))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telemetry send: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(cfg.Telemetry.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telemetry send: post report: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("telemetry send: collector returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return nil
}

func foundationTelemetryPriority(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func foundationTelemetryTicketBody(pattern foundationtelemetry.AggregatedPattern) string {
	return fmt.Sprintf(`## Context

Anonymous deployed-harness telemetry reported a repeated foundation-owned failure pattern. Raw target telemetry remains local; this ticket is created from sanitized aggregate collector data only.

## Triage Metadata

- Signature: %s
- Category: %s
- Target: %s
- Severity: %s
- Report count: %d
- Install-window count: %d
- First seen: %s
- Last seen: %s
- Harness versions: %s

## Acceptance Criteria

- [ ] Root cause is classified against the MARS foundation surface.
- [ ] The fix lands in source harness code, generated target doctrine, prompt/skill/tool policy, or docs as appropriate.
- [ ] New evidence proves target repos no longer need to create local intervention-debt tickets for this failure pattern.
`,
		pattern.Signature,
		pattern.Category,
		pattern.Target,
		pattern.Severity,
		pattern.ReportCount,
		pattern.InstallWindowCount,
		pattern.FirstSeen.Format(time.RFC3339),
		pattern.LastSeen.Format(time.RFC3339),
		strings.Join(pattern.HarnessVersions, ", "),
	)
}

func defaultStringCLI(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
				if isUnavailableDatabaseError(err) {
					fmt.Fprintf(cmd.OutOrStdout(), "No trust entries recorded yet. %s\n", databaseEvidenceRemediation(path, err))
					return nil
				}
				return err
			}
			defer store.Close()
			entries, err := store.List(cmd.Context())
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No trust entries recorded yet.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-18s %-12s %-7s %s\n", "REPO", "ROLE", "LEVEL", "TRIALS", "UPDATED")
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "%-36s %-18s %-12s %-7d %s\n",
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

func resolveWebhookActorIDs(cliIDs, yamlIDs []int64) ([]int64, error) {
	selected := append([]int64(nil), cliIDs...)
	source := "--webhook-actor-id"
	if len(selected) == 0 {
		source = "MARS_WEBHOOK_ALLOWED_ACTOR_IDS"
		raw := strings.TrimSpace(config.Env(source))
		if raw != "" {
			for _, value := range strings.Split(raw, ",") {
				id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("webhook actor policy: %s contains invalid numeric actor ID %q; use comma-separated positive GitHub actor IDs", source, value)
				}
				selected = append(selected, id)
			}
		} else {
			source = "webhook_allowed_actor_ids"
			selected = append(selected, yamlIDs...)
		}
	}
	result, err := gh.NormalizeWebhookActorIDs(selected)
	if err != nil {
		return nil, fmt.Errorf("webhook actor policy from %s: %w", source, err)
	}
	return result, nil
}

func serveCmd() *cobra.Command {
	var (
		webhookAddr     string
		webhookActorIDs []int64
		concurrency     int
		dbPath          string
		debug           bool
		logFile         string
		codeIntelFlag   string
		dashboardOrigin string
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
				webhookAddr = fmt.Sprintf("127.0.0.1:%d", cfg.WebhookPort)
			}

			if dbPath == "" {
				dbPath = legacyDBPath()
				slog.Info("serve: using shared DB — for per-repo isolation, use 'start --repo' or pass --db explicitly")
			}
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("serve: create db directory: %w", err)
			}
			codeIntelRuntime, err := resolveCodeIntelRuntime(codeIntelFlag, cfg)
			if err != nil {
				return err
			}

			webhookSecret, err := gh.ResolveWebhookSecret(config.Env("MARS_WEBHOOK_SECRET"))
			if err != nil {
				return err
			}
			actors, err := resolveWebhookActorIDs(webhookActorIDs, cfg.WebhookAllowedActorIDs)
			if err != nil {
				return err
			}
			dashboardAddr := fmt.Sprintf("127.0.0.1:%d", cfg.DashboardPort)
			trustedOrigin := strings.TrimSpace(dashboardOrigin)
			if trustedOrigin == "" {
				trustedOrigin = strings.TrimSpace(config.Env("MARS_DASHBOARD_TRUSTED_ORIGIN"))
			}

			display, err := newRuntimeDisplay("serve", logFile, debug, cmd.ErrOrStderr(), cmd.ErrOrStderr(), nil, ui.DashboardOptions{
				Title:        "MARS",
				DashboardURL: "http://" + dashboardAddr,
				Controls:     "[p] pause  [r] restart  [s] scan  [q] quit  [h] help",
			})
			if err != nil {
				return err
			}
			defer display.Close()

			serve.Cleanup(cfg.WebhookPort, dbPath, cfg.DashboardPort)

			srv, err := serve.New(serve.Config{
				WebhookAddr:            webhookAddr,
				WebhookSecret:          webhookSecret,
				WebhookAllowedActorIDs: actors,
				DBPath:                 dbPath,
				Concurrency:            concurrency,
				ModelsDir:              cfg.ModelsDir,
				BinDir:                 cfg.BinDir,
				DashboardAddr:          dashboardAddr,
				DashboardControlSecret: config.Env("MARS_DASHBOARD_CONTROL_SECRET"),
				DashboardTrustedOrigin: trustedOrigin,
				PerformanceProfile:     cfg.PerformanceProfile,
				InferenceTuning:        inferenceTuningFromConfig(cfg),
				RequireModelPreflight:  true,
				JobViews:               display.jobViews,
				CodeIntelDisabled:      !codeIntelRuntime.Enabled,
				CodeIntelSource:        codeIntelRuntime.Source,
			})
			if err != nil {
				return err
			}
			if display.dashboard != nil {
				display.dashboard.SetStatusProvider(srv)
			}
			display.Start()
			display.Event("info", "orchestrator starting")

			parentCtx := cmd.Context()
			if parentCtx == nil {
				parentCtx = context.Background()
			}
			sigCtx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
			defer stop()

			var notifier ui.StatusNotifier
			if display.dashboard != nil && display.dashboard.Active() && !debug {
				notifier = display.dashboard
			} else if ui.IsTerminal(os.Stderr) {
				sb := ui.NewStatusBar(os.Stderr, srv)
				sb.Start()
				defer sb.Stop()
				notifier = sb
			}

			kl := ui.NewKeyListener(srv, stop, notifier)
			kl.Start(sigCtx)
			defer kl.Stop()

			return srv.Start(sigCtx)
		},
	}

	cmd.Flags().StringVar(&webhookAddr, "addr", "", "Loopback address to listen on (default 127.0.0.1:<webhook_port>)")
	cmd.Flags().Int64SliceVar(&webhookActorIDs, "webhook-actor-id", nil, "Trusted numeric GitHub actor ID; repeat for multiple actors (overrides env and YAML)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 2, "Number of concurrent agent workers")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/mars.db)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Stream verbose trace and logs inline instead of using the TTY dashboard")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Write verbose command logs to this file (default ~/.mars/traces/logs/<timestamp>-serve.log)")
	cmd.Flags().StringVar(&codeIntelFlag, "code-intel", "", "Enable automatic code graph context and loop maintenance: true or false (default from config/env)")
	cmd.Flags().StringVar(&dashboardOrigin, "dashboard-trusted-origin", "", "Exact HTTPS reverse-proxy origin for authenticated dashboard access (overrides MARS_DASHBOARD_TRUSTED_ORIGIN; listener stays loopback-only)")

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

If .harness/manifest.yaml is missing, mars runs the same scaffold as
'mars init' automatically (requires a git repository).`,
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
			usingDefaultDBPath := strings.TrimSpace(dbPath) == ""
			if usingDefaultDBPath {
				dbPath = defaultDBPath(absPath)
			} else {
				dbPath = strings.TrimSpace(dbPath)
			}
			if err := validateRuntimeArtifactPathOutsideRepo("register", "--db", dbPath, absPath, defaultDBPath(absPath)); err != nil {
				return err
			}
			if usingDefaultDBPath {
				if _, err := os.Stat(legacyDBPath()); err == nil {
					if _, err := os.Stat(dbPath); os.IsNotExist(err) {
						slog.Warn("register: legacy shared database exists at " + legacyDBPath() + " but per-repo DB does not yet exist — starting fresh. Copy the legacy DB to " + dbPath + " if you want to preserve history.")
					}
				}
			}

			preInitChanges, err := gitChangedPaths(absPath)
			if err != nil {
				return fmt.Errorf("register: inspect pre-init git status: %w", err)
			}

			didInit, err := scanner.EnsureHarness(absPath, false)
			if err != nil {
				return fmt.Errorf("register: %w", err)
			}
			if didInit {
				fmt.Fprintf(os.Stderr, "Auto-initialised .harness/ in %s\n", absPath)
				committed, err := commitGeneratedHarnessBaseline(absPath, preInitChanges)
				if err != nil {
					return fmt.Errorf("register: commit generated harness baseline: %w", err)
				}
				if committed {
					fmt.Fprintf(os.Stderr, "Committed generated harness baseline in %s\n", absPath)
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")

	return cmd
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database at %s: %w — check path and permissions", path, err)
	}
	return db, nil
}

func isMissingDatabaseDirError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "database directory") && strings.Contains(msg, "does not exist")
}

func isUnavailableDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return isMissingDatabaseDirError(err) ||
		strings.Contains(msg, "database at") && strings.Contains(msg, "is unavailable") ||
		strings.Contains(msg, "unable to open database file") ||
		strings.Contains(msg, "out of memory (14)")
}

func databaseEvidenceRemediation(dbPath string, err error) string {
	reason := "database is unavailable"
	if isMissingDatabaseDirError(err) {
		reason = "database directory does not exist"
	}
	return fmt.Sprintf("%s for %s — run `mars setup`, run `mars register --repo <path>`, or pass --db with a writable SQLite path", reason, dbPath)
}

// defaultDBPath returns the per-repo database path: ~/.mars/db/{repo-slug}/mars.db.
// Each repo gets its own SQLite file so queue, telemetry, and scheduling are isolated.
func defaultDBPath(repoAbsPath string) string {
	home, _ := os.UserHomeDir()
	repoSlug := filepath.Base(repoAbsPath)
	return filepath.Join(home, ".mars", "db", repoSlug, "mars.db")
}

func validateRuntimeArtifactPathOutsideRepo(command, flag, path, repoAbsPath, suggested string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%s: resolve %s path: %w", command, flag, err)
	}
	absRepo, err := filepath.Abs(repoAbsPath)
	if err != nil {
		return fmt.Errorf("%s: resolve target repo path: %w", command, err)
	}
	rel, err := filepath.Rel(absRepo, absPath)
	if err != nil {
		return fmt.Errorf("%s: compare %s path to target repo: %w", command, flag, err)
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && !filepath.IsAbs(rel)) {
		return fmt.Errorf("%s: %s path %s is inside target repo %s; use %s or pass a writable path outside the repo so runtime artifacts cannot dirty the project worktree", command, flag, absPath, absRepo, suggested)
	}
	return nil
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

func resolveQualityExportPaths(repoArg, dbPath string) (string, string, string, error) {
	repoArg = strings.TrimSpace(repoArg)
	if repoArg == "" {
		repoArg = "."
	}
	abs, err := filepath.Abs(repoArg)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve repo path: %w", err)
	}
	if dbPath == "" {
		dbPath = defaultDBPath(abs)
	}

	repoID := ""
	if _, err := os.Stat(dbPath); err == nil {
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
	} else if !os.IsNotExist(err) {
		return "", "", "", fmt.Errorf("resolve quality export database: %w", err)
	}

	return abs, dbPath, repoID, nil
}

// legacyDBPath returns the pre-rename shared database path: ~/.mars-harness/db/mars.db.
func legacyDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mars-harness", "db", "mars.db")
}

func startCmd() *cobra.Command {
	var (
		repoPath          string
		concurrency       int
		dbPath            string
		force             bool
		exitAfterSeed     bool
		newLifecycle      bool
		debug             bool
		logFile           string
		codeIntelFlag     string
		modelEndpoint     string
		webhookAddrFlag   string
		dashboardAddrFlag string
		dashboardOrigin   string
		remote            string
		branch            string
		webhookActorIDs   []int64
		yes               bool
		jsonOut           bool
		plain             bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Bootstrap and run the full autonomous pipeline",
		Long: `Initialise .harness/ if needed, register the repo, reconcile any
existing lifecycle state, and start the orchestrator. Bootstrap order is exec
plan first, then feature contracts, then tickets, then delivery. Dispatch
routes deterministic role dispositions directly and uses Orchestrator for
ambiguous handoffs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer silenceLogsForJSON(jsonOut)()
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
			if err := validateRuntimeArtifactPathOutsideRepo("start", "--log-file", logFile, absPath, "default ~/.mars/traces/logs/<timestamp>-start.log"); err != nil {
				return err
			}
			usingDefaultDBPath := strings.TrimSpace(dbPath) == ""
			if usingDefaultDBPath {
				dbPath = defaultDBPath(absPath)
			} else {
				dbPath = strings.TrimSpace(dbPath)
			}
			if err := validateRuntimeArtifactPathOutsideRepo("start", "--db", dbPath, absPath, defaultDBPath(absPath)); err != nil {
				return err
			}
			if usingDefaultDBPath {
				if _, err := os.Stat(legacyDBPath()); err == nil {
					if _, err := os.Stat(dbPath); os.IsNotExist(err) {
						slog.Warn("start: legacy shared database exists at " + legacyDBPath() + " but per-repo DB does not yet exist — starting fresh. Copy the legacy DB to " + dbPath + " if you want to preserve history.")
					}
				}
			}

			displayOut := cmd.OutOrStdout()
			displayLog := cmd.ErrOrStderr()
			if jsonOut {
				displayOut = io.Discard
				displayLog = io.Discard
			}
			display, err := newRuntimeDisplay("start", logFile, debug, displayOut, displayLog, nil, ui.DashboardOptions{
				Title:    "MARS",
				RepoPath: absPath,
				Controls: "[p] pause  [r] restart  [s] scan  [q] quit  [h] help",
			})
			if err != nil {
				return err
			}
			defer display.Close()

			preInitChanges, err := gitChangedPaths(absPath)
			if err != nil {
				display.Error(fmt.Sprintf("inspect pre-init git status: %v", err))
				return err
			}

			didInit, err := scanner.EnsureHarness(absPath, force)
			if err != nil {
				display.Error(fmt.Sprintf("init failed: %v", err))
				return err
			}
			if didInit {
				display.Event("init", "No .harness/ found — initialised with default pipeline...")
				committed, err := commitGeneratedHarnessBaseline(absPath, preInitChanges)
				if err != nil {
					display.Error(fmt.Sprintf("commit generated harness baseline: %v", err))
					return err
				}
				if committed {
					display.Event("git", "Committed generated harness baseline so bootstrap agents start from a clean scaffold.")
				}
			}

			cfg, err := config.Load(config.DefaultPath())
			if err != nil {
				slog.Warn("config load failed, using defaults", "err", err)
			}
			codeIntelRuntime, err := resolveCodeIntelRuntime(codeIntelFlag, cfg)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("start: create db directory: %w", err)
			}

			webhookAddr := strings.TrimSpace(webhookAddrFlag)
			if webhookAddr == "" {
				webhookAddr = fmt.Sprintf("127.0.0.1:%d", cfg.WebhookPort)
			}
			dashboardAddr := strings.TrimSpace(dashboardAddrFlag)
			if dashboardAddr == "" {
				dashboardAddr = fmt.Sprintf("127.0.0.1:%d", cfg.DashboardPort)
			}
			trustedOrigin := strings.TrimSpace(dashboardOrigin)
			if trustedOrigin == "" {
				trustedOrigin = strings.TrimSpace(config.Env("MARS_DASHBOARD_TRUSTED_ORIGIN"))
			}
			actors, err := resolveWebhookActorIDs(webhookActorIDs, cfg.WebhookAllowedActorIDs)
			if err != nil {
				return err
			}
			webhookSecret, err := gh.ResolveWebhookSecret(config.Env("MARS_WEBHOOK_SECRET"))
			if err != nil {
				return err
			}
			allowHTTPFallback := strings.TrimSpace(webhookAddrFlag) == "" && strings.TrimSpace(dashboardAddrFlag) == ""

			if config.Env("MARS_SKIP_START_CLEANUP") != "1" {
				serve.CleanupScopedLifecycle(dbPath)
			}

			srv, err := serve.New(serve.Config{
				WebhookAddr:            webhookAddr,
				WebhookSecret:          webhookSecret,
				WebhookAllowedActorIDs: actors,
				DBPath:                 dbPath,
				Concurrency:            concurrency,
				ModelsDir:              cfg.ModelsDir,
				BinDir:                 cfg.BinDir,
				DashboardAddr:          dashboardAddr,
				DashboardControlSecret: config.Env("MARS_DASHBOARD_CONTROL_SECRET"),
				DashboardTrustedOrigin: trustedOrigin,
				RepoScope:              absPath,
				PerformanceProfile:     cfg.PerformanceProfile,
				InferenceTuning:        inferenceTuningFromConfig(cfg),
				ModelEndpoint:          modelEndpoint,
				RequireModelPreflight:  modelEndpoint == "",
				EphemeralHTTPFallback:  allowHTTPFallback,
				JobViews:               display.jobViews,
				CodeIntelDisabled:      !codeIntelRuntime.Enabled,
				CodeIntelSource:        codeIntelRuntime.Source,
			})
			if err != nil {
				display.Error(fmt.Sprintf("orchestrator init: %v", err))
				return err
			}
			if display.dashboard != nil {
				display.dashboard.SetStatusProvider(srv)
			}
			display.Start()
			if display.dashboard != nil && display.dashboard.Active() {
				display.dashboard.AddEvent("info", "dashboard http://"+dashboardAddr)
			}

			parentCtx := cmd.Context()
			if parentCtx == nil {
				parentCtx = context.Background()
			}
			sigCtx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
			defer stop()

			repoID, err := srv.Repos().Register(sigCtx, absPath, remote, branch)
			if err != nil {
				display.Error(fmt.Sprintf("register: %v", err))
				return err
			}
			display.Event("repo", fmt.Sprintf("Registered repo %s (ID: %s)", filepath.Base(absPath), repoID))

			startup, err := srv.ReconcileStartup(sigCtx, repoID, absPath, newLifecycle)
			if err != nil {
				display.Error(fmt.Sprintf("startup reconciliation: %v", err))
				return err
			}
			display.Event("startup", startup.Summary())
			if startup.Action == serve.StartupActionRefusedAmbiguous {
				return fmt.Errorf("start: %s; inspect the repo/DB or rerun with --new-lifecycle to intentionally seed CEO", startup.Summary())
			}
			if startup.ShouldSeed {
				triggerJSON := fmt.Sprintf(`{"type":"bootstrap","source":"mars start","startup_action":%q}`, startup.Action)
				var jobID string
				if newLifecycle {
					jobID, err = srv.SeedJob(sigCtx, repoID, "ceo", triggerJSON)
				} else {
					jobID, err = srv.SeedBootstrapJob(sigCtx, repoID, "ceo", triggerJSON)
				}
				if err != nil {
					display.Error(fmt.Sprintf("seed CEO: %v", err))
					return err
				}
				display.Event("queue", fmt.Sprintf("Seeded CEO agent (job %s) — bootstrap order: exec plan → features → tickets → delivery; Orchestrator selects each next role", jobID))
			} else if startup.JobID != "" {
				display.Event("queue", fmt.Sprintf("Resuming lifecycle with %s job %s", startup.Role, startup.JobID))
			}

			if exitAfterSeed {
				if err := srv.Stop(context.Background()); err != nil {
					return err
				}
				_ = yes
				_ = plain
				if jsonOut {
					return writeJSON(os.Stdout, map[string]any{
						"status":  "ok",
						"repo":    absPath,
						"repo_id": repoID,
						"seeded":  startup.ShouldSeed,
					})
				}
				return nil
			}

			var notifier ui.StatusNotifier
			if display.dashboard != nil && display.dashboard.Active() && !debug {
				notifier = display.dashboard
			} else if ui.IsTerminal(os.Stderr) {
				sb := ui.NewStatusBar(os.Stderr, srv)
				sb.Start()
				defer sb.Stop()
				notifier = sb
			}

			kl := ui.NewKeyListener(srv, stop, notifier)
			kl.Start(sigCtx)
			defer kl.Stop()

			return srv.Start(sigCtx)
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the target repository (default: current directory)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of concurrent agent workers (1 = sequential pipeline)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars/db/{repo}/mars.db)")
	cmd.Flags().BoolVar(&force, "force", false, "Force re-init .harness/ even if it exists")
	cmd.Flags().BoolVar(&newLifecycle, "new-lifecycle", false, "Intentionally seed a fresh CEO lifecycle even when resumable state exists")
	cmd.Flags().BoolVar(&debug, "debug", false, "Stream verbose trace and logs inline instead of using the TTY dashboard")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Write verbose command logs to this file (default ~/.mars/traces/logs/<timestamp>-start.log)")
	cmd.Flags().StringVar(&codeIntelFlag, "code-intel", "", "Enable automatic code graph context and loop maintenance: true or false (default from config/env)")
	cmd.Flags().StringVar(&modelEndpoint, "model-endpoint", "", "Optional real OpenAI-compatible model endpoint override; skips local llama-server startup. Fake or scripted endpoints are not live validation evidence")
	cmd.Flags().StringVar(&webhookAddrFlag, "addr", "", "Loopback webhook/control listen address (default 127.0.0.1:<webhook_port>; scoped start falls back to a loopback ephemeral port on conflict)")
	cmd.Flags().StringVar(&dashboardAddrFlag, "dashboard-addr", "", "Loopback dashboard listen address (default 127.0.0.1:<dashboard_port>; scoped start falls back to a loopback ephemeral port on conflict)")
	cmd.Flags().StringVar(&dashboardOrigin, "dashboard-trusted-origin", "", "Exact HTTPS reverse-proxy origin for authenticated dashboard access (overrides MARS_DASHBOARD_TRUSTED_ORIGIN; listener stays loopback-only)")
	cmd.Flags().StringVar(&remote, "remote", "", "Exact GitHub owner/repo allowed for webhooks; empty preserves an existing registration")
	cmd.Flags().StringVar(&branch, "branch", "", "Exact case-sensitive branch allowed for webhooks; empty preserves an existing registration or defaults new registrations to main")
	cmd.Flags().Int64SliceVar(&webhookActorIDs, "webhook-actor-id", nil, "Trusted numeric GitHub actor ID; repeat for multiple actors (overrides env and YAML)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Do not prompt; fail with remediation when required input is missing")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output for deterministic setup/seed paths")
	cmd.Flags().BoolVar(&plain, "plain", false, "Disable styling and animation")
	cmd.Flags().BoolVar(&exitAfterSeed, "exit-after-seed", false, "Exit after init/register/seed; intended for deterministic smoke tests")
	_ = cmd.Flags().MarkHidden("exit-after-seed")

	return cmd
}
