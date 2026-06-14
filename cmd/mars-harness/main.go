/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/dashboard.md
- docs/design-docs/harness-glossary.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/release-versioning.md
- docs/design-docs/self-reflective-telemetry.md
- docs/product-specs/product-surface.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-009-release-update-lifecycle.md
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
	"strings"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"

	"github.com/greaveselliott/mars-harness/internal/agent"
	"github.com/greaveselliott/mars-harness/internal/buildinfo"
	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/codeintel"
	"github.com/greaveselliott/mars-harness/internal/config"
	ctx "github.com/greaveselliott/mars-harness/internal/context"
	"github.com/greaveselliott/mars-harness/internal/docsync"
	"github.com/greaveselliott/mars-harness/internal/doctor"
	"github.com/greaveselliott/mars-harness/internal/foundationtelemetry"
	"github.com/greaveselliott/mars-harness/internal/githubauth"
	"github.com/greaveselliott/mars-harness/internal/guardrails"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/mcpstdio"
	"github.com/greaveselliott/mars-harness/internal/models"
	"github.com/greaveselliott/mars-harness/internal/qualityscore"
	"github.com/greaveselliott/mars-harness/internal/release"
	"github.com/greaveselliott/mars-harness/internal/safety"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/selfupdate"
	"github.com/greaveselliott/mars-harness/internal/serve"
	"github.com/greaveselliott/mars-harness/internal/setup"
	"github.com/greaveselliott/mars-harness/internal/shellpath"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/greaveselliott/mars-harness/internal/ui"
	"github.com/greaveselliott/mars-harness/internal/updatecheck"
	foundationvalidation "github.com/greaveselliott/mars-harness/internal/validation"
)

var version = buildinfo.DefaultVersion

var (
	commit = "unknown"
	date   = "unknown"
)

func main() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var showVersion bool
	root := &cobra.Command{
		Use:           "mars-harness",
		Short:         "Autonomous AI delivery system",
		Long:          "Mars Harness — self-hosted autonomous AI delivery. Run setup to get started.",
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
		Long: `Run Mars Harness foundation validation helpers.

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
run compartmentalised smoke validation cases for Mars Harness roles through
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
	cmd.Flags().IntVar(&opts.MaxTurns, "max-turns", 6, "Maximum role turns for live execution")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "Per-case timeout")
	cmd.Flags().StringVar(&opts.ModelEndpoint, "model-endpoint", "", "Optional OpenAI-compatible model endpoint for live smoke execution")
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
		Short: "Expose Mars Harness tools through Model Context Protocol",
		Long: `Expose the registered Mars Harness tool registry through standard MCP
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
		"-c", "user.name=Mars Harness",
		"-c", "user.email=mars-harness@example.invalid",
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
		Short: "Run a stdio MCP server for registered Mars Harness tools",
		Long: `Run a newline-delimited JSON-RPC stdio MCP server. The server exposes
registered Mars Harness tools via tools/list and tools/call, using the same
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
		Short: "Inspect and execute registered Mars Harness tools",
		Long: `Inspect and execute registered Mars Harness tools through the same registry,
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
	cmd.AddCommand(modelsListCmd())
	cmd.AddCommand(modelsEvaluateCmd())
	cmd.AddCommand(modelsOverrideCmd())
	return cmd
}

func modelsListCmd() *cobra.Command {
	var (
		provider string
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List model candidates from a provider",
		RunE: func(cmd *cobra.Command, args []string) error {
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
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func modelsEvaluateCmd() *cobra.Command {
	var (
		endpoint        string
		model           string
		provider        string
		apiKey          string
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
				APIKey:          apiKey,
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
	cmd.Flags().StringVar(&provider, "provider", models.ProviderOpenAICompatible, "Provider label: openai-compatible or ollama")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Optional API key for the endpoint")
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
		repoRoot string
		tier     string
		role     string
		provider string
		model    string
		endpoint string
		reason   string
		jsonOut  bool
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
				Provider: provider,
				Model:    model,
				Endpoint: endpoint,
				Reason:   reason,
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
	cmd.Flags().StringVar(&provider, "provider", models.ProviderOllama, "Provider: ollama or openai-compatible")
	cmd.Flags().StringVar(&model, "model", "", "Provider model name")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "OpenAI-compatible endpoint; Ollama defaults to local Ollama")
	cmd.Flags().StringVar(&reason, "reason", "", "Operator rationale saved with the override")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func writeJSON(w io.Writer, v any) error {
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
	fmt.Println("\nRun live evaluation with: mars-harness models evaluate --endpoint <url> --model <name>")
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
	return fmt.Sprintf("mars-harness %s %s/%s commit=%s built=%s", version, runtime.GOOS, runtime.GOARCH, commit, date)
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
		sourceUpdate  bool
		dryRun        bool
		jsonOut       bool
	)
	cmd := &cobra.Command{
		Use:     "tool",
		Aliases: []string{"binary", "cli"},
		Short:   "Reinstall or upgrade the mars-harness command",
		Long: `Reinstall the mars-harness command without changing directories.

By default this downloads the platform release asset, verifies checksums.txt,
and atomically replaces the binary in the directory that contains the currently
running mars-harness binary. Use --source for source-development updates through
go install, or pass --version main which selects the source path automatically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			method := selfupdate.UpdateMethod("")
			if sourceUpdate {
				method = selfupdate.MethodSource
			}
			cfg := selfupdate.Config{
				Version:    updateVersion,
				InstallDir: installDir,
				Method:     method,
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
			fmt.Printf("Method: %s\n", plan.Method)
			fmt.Printf("Version: %s\n", plan.Version)
			fmt.Printf("Install dir: %s\n", plan.InstallDir)
			if len(plan.Command) > 0 {
				fmt.Printf("Command: GOBIN=%s %s\n", plan.InstallDir, strings.Join(plan.Command, " "))
			}
			if plan.AssetName != "" {
				fmt.Printf("Asset: %s\n", plan.AssetName)
				if plan.ReleaseTag == selfupdate.DefaultVersion {
					fmt.Printf("Release metadata: %s\n", selfupdate.DefaultLatestReleaseURL)
				} else {
					fmt.Printf("Download: %s\n", plan.DownloadURL)
					fmt.Printf("Checksums: %s\n", plan.ChecksumsURL)
				}
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
			fmt.Printf("Run: mars-harness version\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&updateVersion, "version", selfupdate.DefaultVersion, "Release or source version to install, e.g. latest, v0.5.3, or main")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Install directory; default is the current mars-harness binary directory")
	cmd.Flags().BoolVar(&sourceUpdate, "source", false, "Use go install instead of checksum-verified release assets")
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
		Short: "Configure and check external authentication used by Mars Harness",
	}
	cmd.AddCommand(authGitHubCmd())
	return cmd
}

func authGitHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Configure and check GitHub auth for private Mars Harness releases",
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
		Short: "Check GitHub auth for private Mars Harness release assets",
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
	cmd.Flags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.mars-harness/config.yaml)")
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
		Short: "Prepare GitHub auth for private Mars Harness release assets",
		Long: `Prepare GitHub auth for private Mars Harness release assets.

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
	cmd.Flags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.mars-harness/config.yaml)")
	cmd.Flags().StringVar(&token, "token", "", "Persist a GitHub token in ~/.mars-harness/config.yaml for headless installs; prefer GitHub CLI auth when possible")
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
		Short: "Add the mars-harness install directory to the current user's shell PATH",
		Long:  "Detect Fish, Zsh, Bash, POSIX sh, or Csh/Tcsh and write an idempotent shell profile snippet so mars-harness works in new terminals.",
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
			fmt.Printf("mars-harness path setup\n")
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
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Directory containing mars-harness; default resolves from current executable or Go bin")
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
	cmd.AddCommand(releaseAuditCmd())
	cmd.AddCommand(releaseBackfillNotesCmd())
	cmd.AddCommand(releaseNotesCmd())
	cmd.AddCommand(releasePublishAssetsCmd())
	cmd.AddCommand(releaseVerifyAssetsCmd())
	return cmd
}

func releaseAuditCmd() *cobra.Command {
	var (
		repoPath       string
		githubRepoFull string
		limit          int
		jsonOut        bool
	)
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Detect notes-only or missing GitHub releases for recent tags",
		Long: `Audit the newest local vX.Y.Z tags against GitHub Releases and report
versions whose release object is missing or has incomplete binary assets
(notes-only releases). Each finding names the exact publish-assets backfill
command. When tags or the GitHub API are unavailable the audit reports the
blocker and exits successfully, because the GitHub mirror is optional
infrastructure.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("release audit: cannot determine working directory: %w", err)
				}
			}
			result, err := release.Audit(cmd.Context(), release.AuditConfig{
				RepoRoot:     repoPath,
				RepoFullName: githubRepoFull,
				Limit:        limit,
			})
			if err != nil {
				return err
			}
			return printReleaseAuditResult(cmd.OutOrStdout(), result, jsonOut)
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().StringVar(&githubRepoFull, "github-repo", selfupdate.DefaultRepoFullName, "GitHub repository in owner/name form")
	cmd.Flags().IntVar(&limit, "limit", 10, "Newest version tags to audit")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func printReleaseAuditResult(out io.Writer, result release.AuditResult, jsonOut bool) error {
	if jsonOut {
		if err := writeJSON(out, result); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(out, "Release audit: %s\n", result.RepoFullName)
		if result.Skipped {
			fmt.Fprintf(out, "Skipped: %s\n", result.SkipReason)
			fmt.Fprintln(out, "Record this blocker if release mirror health was expected to be verifiable.")
			return nil
		}
		fmt.Fprintf(out, "Checked: %s\n", strings.Join(result.Checked, ", "))
		for _, finding := range result.Findings {
			fmt.Fprintf(out, "FINDING (%s): %s\n", finding.Class, finding.TagName)
			if len(finding.Missing) > 0 {
				fmt.Fprintf(out, "  Missing: %s\n", strings.Join(finding.Missing, ", "))
			}
			fmt.Fprintf(out, "  Fix: %s\n", finding.Remediation)
		}
		if len(result.Findings) == 0 {
			fmt.Fprintln(out, "Status: ok")
		}
	}
	if len(result.Findings) > 0 {
		return fmt.Errorf("release audit: %d release(s) missing assets or release objects", len(result.Findings))
	}
	return nil
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
		Long: `Backfill existing CHANGELOG.md entries that have mars-harness release
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

func releaseVerifyAssetsCmd() *cobra.Command {
	var (
		repoFullName  string
		verifyVersion string
		releaseURL    string
		distDir       string
		jsonOut       bool
	)
	cmd := &cobra.Command{
		Use:   "verify-assets",
		Short: "Verify release binary assets",
		Long: `Verify that local release assets or a GitHub Release contain all Mars
Harness binary assets and checksums.txt before announcing an installer or
self-update release.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(distDir) != "" {
				report, err := release.VerifyLocalAssets(distDir, verifyVersion)
				if err != nil {
					return err
				}
				return printReleaseAssetReport(cmd.OutOrStdout(), report, jsonOut)
			}
			url := strings.TrimSpace(releaseURL)
			if url == "" {
				url = selfupdate.ReleaseAPIURL(repoFullName, verifyVersion)
			}
			report, err := selfupdate.VerifyReleaseAssets(cmd.Context(), nil, url)
			if err != nil {
				return err
			}
			return printReleaseAssetReport(cmd.OutOrStdout(), report, jsonOut)
		},
	}
	cmd.Flags().StringVar(&repoFullName, "repo", selfupdate.DefaultRepoFullName, "GitHub repository in owner/name form")
	cmd.Flags().StringVar(&verifyVersion, "version", selfupdate.DefaultVersion, "Release tag to verify, e.g. latest or v0.12.0")
	cmd.Flags().StringVar(&releaseURL, "release-url", "", "GitHub-compatible release metadata URL override")
	cmd.Flags().StringVar(&distDir, "dist", "", "Local release asset directory to verify instead of GitHub release metadata")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Write JSON output")
	return cmd
}

func releasePublishAssetsCmd() *cobra.Command {
	var (
		repoPath       string
		version        string
		distDir        string
		upload         string
		githubRepoFull string
	)
	cmd := &cobra.Command{
		Use:   "publish-assets",
		Short: "Build and optionally mirror release assets locally",
		Long: `Build linux/darwin amd64/arm64 release binaries from the release-note
commit, generate checksums.txt, verify local assets, and optionally mirror the
assets to GitHub Releases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				var err error
				repoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("release publish-assets: cannot determine working directory: %w", err)
				}
			}
			result, err := release.PublishAssets(cmd.Context(), release.PublishAssetsConfig{
				RepoRoot:   repoPath,
				Version:    version,
				DistDir:    distDir,
				Upload:     upload,
				GitHubRepo: githubRepoFull,
				Stdout:     cmd.OutOrStdout(),
				Stderr:     cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Release assets: %s\n", result.TagName)
			fmt.Fprintf(out, "Dist: %s\n", result.DistDir)
			fmt.Fprintf(out, "Assets: %s\n", strings.Join(assetBaseNames(result.Assets), ", "))
			fmt.Fprintf(out, "Checksums: %s\n", result.ChecksumsPath)
			switch {
			case result.Uploaded:
				fmt.Fprintln(out, "GitHub mirror: uploaded")
			case result.UploadSkipped:
				fmt.Fprintln(out, "GitHub mirror: skipped")
			default:
				fmt.Fprintln(out, "GitHub mirror: disabled")
			}
			fmt.Fprintln(out, "Status: ok")
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().StringVar(&version, "version", "", "Release tag to publish, for example v1.2.3")
	cmd.Flags().StringVar(&distDir, "dist", filepath.Join("dist", "releases"), "Local release asset directory")
	cmd.Flags().StringVar(&upload, "upload", "none", "Optional mirror mode: none, github, or auto")
	cmd.Flags().StringVar(&githubRepoFull, "github-repo", selfupdate.DefaultRepoFullName, "GitHub repository in owner/name form for upload modes")
	_ = cmd.MarkFlagRequired("version")
	return cmd
}

func printReleaseAssetReport(out io.Writer, report selfupdate.ReleaseAssetReport, jsonOut bool) error {
	if jsonOut {
		return writeJSON(out, report)
	}
	fmt.Fprintf(out, "Release assets: %s\n", report.TagName)
	if report.URL != "" {
		fmt.Fprintf(out, "URL: %s\n", report.URL)
	}
	fmt.Fprintf(out, "Required: %s\n", strings.Join(report.Required, ", "))
	if len(report.Found) > 0 {
		fmt.Fprintf(out, "Found: %s\n", strings.Join(report.Found, ", "))
	}
	if report.OK {
		fmt.Fprintln(out, "Status: ok")
		return nil
	}
	fmt.Fprintf(out, "Missing: %s\n", strings.Join(report.Missing, ", "))
	return fmt.Errorf("release verify-assets: release %s is missing required assets", report.TagName)
}

func assetBaseNames(paths []string) []string {
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}
	return names
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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

If .harness/manifest.yaml is missing, the same scaffold as 'mars-harness init'
is applied automatically (requires a git repository). Use --no-init with
--dry-run when inspecting an uninitialized target without writing harness
scaffolding. The source-only foundation-maintainer role may run from the
mars-harness source repo with --dry-run --no-init without creating a source
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
	cmd.Flags().StringVar(&logFile, "log-file", "", "Write verbose command logs to this file (default ~/.mars-harness/traces/logs/<timestamp>-run.log)")
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
	fmt.Fprintf(d.out, "mars-harness: %s\n", msg)
}

func (d *runtimeDisplay) Error(msg string) {
	if d == nil {
		return
	}
	if d.dashboard != nil && d.dashboard.Active() && !d.debug {
		d.dashboard.AddWarning(msg)
		return
	}
	fmt.Fprintf(d.out, "mars-harness: error: %s\n", msg)
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
		Name:              "mars-harness-foundation",
		Description:       "Source-only foundation operating model for mars-harness maintainers",
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
			"file_read", "file_write", "shell_exec", "dependency_sync", "mars_harness_cli",
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
			Message:   "After semantic source changes, keep docs synchronized, generate release notes, push trunk, publish local release assets, optionally mirror them to GitHub, and record any asset blocker.",
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

func isMarsHarnessSourceRepo(repoRoot string) bool {
	goMod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil || !strings.Contains(string(goMod), "module github.com/greaveselliott/mars-harness") {
		return false
	}
	for _, rel := range []string{
		filepath.Join("cmd", "mars-harness", "main.go"),
		filepath.Join("internal", "scanner", "init.go"),
		"AGENTS.md",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
			return false
		}
	}
	return true
}

func resolveCodeIntelRuntime(flagValue string, cfg config.Config) (codeintel.Runtime, error) {
	if raw := strings.TrimSpace(flagValue); raw != "" {
		enabled, err := parseCodeIntelBool(raw)
		if err != nil {
			return codeintel.Runtime{}, err
		}
		return codeintel.NewRuntime(enabled, "flag"), nil
	}
	if raw := strings.TrimSpace(os.Getenv("MARS_HARNESS_CODE_INTEL_ENABLED")); raw != "" {
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
	if err := validateRuntimeArtifactPathOutsideRepo("run", "--log-file", opts.logFile, absRepo, "default ~/.mars-harness/traces/logs/<timestamp>-run.log"); err != nil {
		return err
	}

	sourceFoundationRole := opts.roleName == foundationMaintainerRoleName
	if sourceFoundationRole && !isMarsHarnessSourceRepo(absRepo) {
		return fmt.Errorf("run: role %q is source-only for the mars-harness foundation repo; %s is not a mars-harness source checkout. Use a generated target role for deployed harness work, or rerun from the mars-harness source repository", foundationMaintainerRoleName, absRepo)
	}

	manifestPath := filepath.Join(absRepo, ".harness", "manifest.yaml")
	if opts.noInit && !sourceFoundationRole {
		if _, err := os.Stat(manifestPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("run: inspect harness manifest at %s: %w", manifestPath, err)
			}
			msg := fmt.Sprintf("run: .harness/manifest.yaml is missing in %s and --no-init was set; no files were written. Run `mars-harness init --repo %s` to scaffold the target, or rerun without --no-init when initialization is intended.", absRepo, absRepo)
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
		Title:    "Mars Harness",
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
	var router *inference.Router

	if endpoint == "" {
		override, ok, overrideErr := models.ResolveModelOverride(absRepo, opts.roleName, role.Model)
		if overrideErr != nil {
			tw.WriteError(fmt.Sprintf("model override failed: %v", overrideErr))
			return overrideErr
		}
		if ok {
			endpoint = override.Endpoint
			modelName = override.Model
			slog.Info("model override selected",
				"role", opts.roleName,
				"provider", override.Provider,
				"model", modelName,
				"endpoint", endpoint,
			)
		} else {
			router, endpoint, err = autoStartInference(sigCtx, opts.roleName, role.Model)
			if err != nil {
				tw.WriteError(fmt.Sprintf("inference startup failed: %v", err))
				return err
			}
			defer router.StopAll()
		}
	}

	tw.WriteReady()

	client, err := llm.NewClient(llm.Config{
		BaseURL: endpoint,
		Model:   modelName,
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
		skipGitHub   bool
		enableGitHub bool
		testMode     bool
		dryRun       bool
		installDir   string
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "First-time setup wizard",
		Long:  "Create ~/.mars-harness/, detect hardware, install local inference, and download pinned models. GitHub integration is optional.",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := setup.Run(setup.Config{
				SkipDownload: skipDownload,
				SkipGitHub:   skipGitHub,
				EnableGitHub: enableGitHub,
				TestMode:     testMode,
				DryRun:       dryRun,
				InstallDir:   installDir,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Setup complete: %d steps run, %d skipped\n", result.StepsRun, result.StepsSkipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&skipDownload, "skip-download", false, "Skip model download")
	cmd.Flags().BoolVar(&skipGitHub, "skip-github", false, "Skip GitHub private release auth and optional GitHub integration checks")
	cmd.Flags().BoolVar(&enableGitHub, "github", false, "Configure optional GitHub status/check integration")
	cmd.Flags().BoolVar(&testMode, "test-mode", false, "Skip downloads and external services")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print steps without executing")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "Directory containing mars-harness for shell PATH setup; default resolves automatically")

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
			absPath, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("init: resolve path: %w", err)
			}
			preInitChanges, err := gitChangedPaths(absPath)
			if err != nil {
				return fmt.Errorf("init: inspect pre-init git status: %w", err)
			}
			if err := scanner.Init(absPath, force); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Initialized .harness/ in %s\n", absPath)
			committed, err := commitGeneratedHarnessBaseline(absPath, preInitChanges)
			if err != nil {
				return fmt.Errorf("init: commit generated harness baseline: %w", err)
			}
			if committed {
				fmt.Fprintf(out, "Committed generated harness baseline in %s\n", absPath)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "Path to the repository (default: current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "Refresh missing scaffold and rewrite manifest if .harness/ already exists")

	return cmd
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
		Short:   "Remove Mars Harness from a target repo",
		Long: `Remove the deployed Mars Harness surface from a target repository and
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to associated SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
	cmd.Flags().BoolVar(&apply, "apply", false, "Actually remove files and database artifacts; default is dry-run")
	cmd.Flags().StringVar(&confirm, "confirm", "", "Required with --apply; must equal the target repo directory name")
	cmd.Flags().BoolVar(&keepDB, "keep-db", false, "Remove repo files but leave the associated database untouched")
	cmd.Flags().BoolVar(&deleteSharedDB, "delete-shared-db", false, "Allow deleting the legacy shared ~/.mars-harness/db/mars.db database")
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
	fmt.Fprintf(w, "Mars Harness eject %s for %s\n", mode, result.RepoRoot)
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
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
				dbPath = filepath.Join(home, ".mars-harness", "db", "foundation-telemetry", "intake.db")
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
		Short: "Create Mars Harness source tickets from repeated anonymous telemetry patterns",
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
	cmd.Flags().StringVar(&repoPath, "repo", ".", "Mars Harness source repository path")
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

- [ ] Root cause is classified against the Mars Harness foundation surface.
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

func serveCmd() *cobra.Command {
	var (
		webhookAddr   string
		concurrency   int
		dbPath        string
		debug         bool
		logFile       string
		codeIntelFlag string
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
			codeIntelRuntime, err := resolveCodeIntelRuntime(codeIntelFlag, cfg)
			if err != nil {
				return err
			}

			webhookSecret := os.Getenv("MARS_HARNESS_WEBHOOK_SECRET")
			dashboardAddr := fmt.Sprintf(":%d", cfg.DashboardPort)

			display, err := newRuntimeDisplay("serve", logFile, debug, cmd.ErrOrStderr(), cmd.ErrOrStderr(), nil, ui.DashboardOptions{
				Title:        "Mars Harness",
				DashboardURL: "http://localhost" + dashboardAddr,
				Controls:     "[p] pause  [r] restart  [s] scan  [q] quit  [h] help",
			})
			if err != nil {
				return err
			}
			defer display.Close()

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
				JobViews:           display.jobViews,
				CodeIntelDisabled:  !codeIntelRuntime.Enabled,
				CodeIntelSource:    codeIntelRuntime.Source,
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

	cmd.Flags().StringVar(&webhookAddr, "addr", "", "Address to listen on (default from config webhook_port)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 2, "Number of concurrent agent workers")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/mars.db)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Stream verbose trace and logs inline instead of using the TTY dashboard")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Write verbose command logs to this file (default ~/.mars-harness/traces/logs/<timestamp>-serve.log)")
	cmd.Flags().StringVar(&codeIntelFlag, "code-intel", "", "Enable automatic code graph context and loop maintenance: true or false (default from config/env)")

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
	return fmt.Sprintf("%s for %s — run `mars-harness setup`, run `mars-harness register --repo <path>`, or pass --db with a writable SQLite path", reason, dbPath)
}

// defaultDBPath returns the per-repo database path: ~/.mars-harness/db/{repo-slug}/mars.db.
// Each repo gets its own SQLite file so queue, telemetry, and scheduling are isolated.
func defaultDBPath(repoAbsPath string) string {
	home, _ := os.UserHomeDir()
	repoSlug := filepath.Base(repoAbsPath)
	return filepath.Join(home, ".mars-harness", "db", repoSlug, "mars.db")
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

// legacyDBPath returns the old shared database path: ~/.mars-harness/db/mars.db.
func legacyDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mars-harness", "db", "mars.db")
}

func startCmd() *cobra.Command {
	var (
		repoPath      string
		concurrency   int
		dbPath        string
		force         bool
		exitAfterSeed bool
		debug         bool
		logFile       string
		codeIntelFlag string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Bootstrap and run the full autonomous pipeline",
		Long: `Initialise .harness/ if needed, register the repo, seed the CEO agent,
and start the orchestrator. Bootstrap order is exec plan first, then feature
contracts, then tickets, then delivery. Dispatch routes deterministic role
dispositions directly and uses Orchestrator for ambiguous handoffs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if err := validateRuntimeArtifactPathOutsideRepo("start", "--log-file", logFile, absPath, "default ~/.mars-harness/traces/logs/<timestamp>-start.log"); err != nil {
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

			display, err := newRuntimeDisplay("start", logFile, debug, cmd.OutOrStdout(), cmd.ErrOrStderr(), nil, ui.DashboardOptions{
				Title:    "Mars Harness",
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

			webhookAddr := fmt.Sprintf(":%d", cfg.WebhookPort)
			dashboardAddr := fmt.Sprintf(":%d", cfg.DashboardPort)

			if os.Getenv("MARS_HARNESS_SKIP_START_CLEANUP") != "1" {
				serve.Cleanup(cfg.WebhookPort, dbPath, cfg.DashboardPort)
			}

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
				JobViews:           display.jobViews,
				CodeIntelDisabled:  !codeIntelRuntime.Enabled,
				CodeIntelSource:    codeIntelRuntime.Source,
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
				display.dashboard.AddEvent("info", "dashboard http://localhost"+dashboardAddr)
			}

			parentCtx := cmd.Context()
			if parentCtx == nil {
				parentCtx = context.Background()
			}
			sigCtx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
			defer stop()

			repoID, err := srv.Repos().Register(sigCtx, absPath, "", "main")
			if err != nil {
				display.Error(fmt.Sprintf("register: %v", err))
				return err
			}
			display.Event("repo", fmt.Sprintf("Registered repo %s (ID: %s)", filepath.Base(absPath), repoID))

			triggerJSON := `{"type":"bootstrap","source":"mars-harness start"}`
			jobID, err := srv.SeedBootstrapJob(sigCtx, repoID, "ceo", triggerJSON)
			if err != nil {
				display.Error(fmt.Sprintf("seed CEO: %v", err))
				return err
			}
			display.Event("queue", fmt.Sprintf("Seeded CEO agent (job %s) — bootstrap order: exec plan → features → tickets → delivery; Orchestrator selects each next role", jobID))

			if exitAfterSeed {
				return srv.Stop(context.Background())
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
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database (default ~/.mars-harness/db/{repo}/mars.db)")
	cmd.Flags().BoolVar(&force, "force", false, "Force re-init .harness/ even if it exists")
	cmd.Flags().BoolVar(&debug, "debug", false, "Stream verbose trace and logs inline instead of using the TTY dashboard")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Write verbose command logs to this file (default ~/.mars-harness/traces/logs/<timestamp>-start.log)")
	cmd.Flags().StringVar(&codeIntelFlag, "code-intel", "", "Enable automatic code graph context and loop maintenance: true or false (default from config/env)")
	cmd.Flags().BoolVar(&exitAfterSeed, "exit-after-seed", false, "Exit after init/register/seed; intended for deterministic smoke tests")
	_ = cmd.Flags().MarkHidden("exit-after-seed")

	return cmd
}
