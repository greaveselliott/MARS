/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-smoke-validation.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/github-app-integration.md
- docs/design-docs/dashboard.md
- docs/design-docs/local-inference.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/tools-glossary.md
- docs/validation/README.md
- docs/validation/agent-smoke/README.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-011-optional-github-integration.md
- docs/features/F-017-open-source-publication.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-012-self-improvement-loop.md
*/
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const marsCLISchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "mode": {
      "type": "string",
      "enum": ["reference", "run"],
      "description": "Use reference for exhaustive CLI guidance, or run to execute mars with structured argv"
    },
    "args": {
      "type": "array",
      "items": { "type": "string" },
      "description": "mars arguments excluding the binary name, e.g. [\"doctor\", \"--repo\", \".\", \"--json\"]"
    },
    "repo": {
      "type": "string",
      "description": "Optional repository path under the current workspace. When set, appends --repo <absolute path> for commands that support --repo and do not already include it."
    },
    "timeout_seconds": {
      "type": "integer",
      "description": "Per-invocation timeout in seconds (1-600, default 60)"
    },
    "background": {
      "type": "boolean",
      "description": "Start mars in the background. Use for long-running serve/start runs."
    }
  }
}`

type marsCLIArgs struct {
	Mode           string   `json:"mode"`
	Args           []string `json:"args"`
	Repo           string   `json:"repo"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Background     bool     `json:"background"`
}

type rawMarsCLIArgs struct {
	Mode           string          `json:"mode"`
	Args           json.RawMessage `json:"args"`
	Repo           string          `json:"repo"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	Background     bool            `json:"background"`
}

func registerMarsCLI(r *Registry) error {
	if err := r.Register(
		"mars_cli",
		"Read exhaustive mars CLI reference or execute mars commands with structured argv. This is the mirrored tool for controlling the foundation or deployed harness through the CLI.",
		json.RawMessage(marsCLISchema),
		handleMarsCLI,
	); err != nil {
		return err
	}
	return r.Register(
		"mars_harness_cli",
		"Deprecated compatibility alias for mars_cli. Use mars_cli for new prompts and generated harnesses.",
		json.RawMessage(marsCLISchema),
		handleMarsCLI,
	)
}

func handleMarsCLI(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	args, err := decodeMarsCLIArgs(raw)
	if err != nil {
		return ToolResult{}, fmt.Errorf("mars_cli: parse arguments: %w", err)
	}
	mode := strings.TrimSpace(args.Mode)
	if mode == "" {
		if len(args.Args) == 0 {
			mode = "reference"
		} else {
			mode = "run"
		}
	}
	switch mode {
	case "reference":
		return ToolResult{Output: marsCLIReference()}, nil
	case "run":
		return runMarsCLI(ctx, root, args)
	default:
		return ToolResult{}, fmt.Errorf("mars_cli: unsupported mode %q", args.Mode)
	}
}

func decodeMarsCLIArgs(raw json.RawMessage) (marsCLIArgs, error) {
	var rawArgs rawMarsCLIArgs
	if err := json.Unmarshal(raw, &rawArgs); err != nil {
		return marsCLIArgs{}, err
	}
	args := marsCLIArgs{
		Mode:           rawArgs.Mode,
		Repo:           rawArgs.Repo,
		TimeoutSeconds: rawArgs.TimeoutSeconds,
		Background:     rawArgs.Background,
	}
	if len(rawArgs.Args) == 0 || bytes.Equal(bytes.TrimSpace(rawArgs.Args), []byte("null")) {
		return args, nil
	}
	cliArgs, err := decodeStringSliceArg(rawArgs.Args, "mars_cli.args")
	if err != nil {
		return marsCLIArgs{}, err
	}
	args.Args = normalizeShellExecArgv(cliArgs)
	return args, nil
}

func runMarsCLI(ctx context.Context, root Root, args marsCLIArgs) (ToolResult, error) {
	cliArgs, err := normalizeMarsArgs(root, args)
	if err != nil {
		return ToolResult{}, err
	}
	argv, err := marsCommandArgv(root, cliArgs)
	if err != nil {
		return ToolResult{}, err
	}
	if args.Background {
		return execBackground(root, shellExecArgs{Argv: argv, Background: true})
	}

	timeoutSeconds := args.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	if timeoutSeconds > 600 {
		timeoutSeconds = 600
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...)
	cmd.Dir = root.Abs()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return ToolResult{
				Output:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: -1,
			}, fmt.Errorf("mars_cli: command timed out after %ds; use background:true for serve/start", timeoutSeconds)
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			return ToolResult{Output: stdout.String(), Stderr: stderr.String()}, fmt.Errorf("mars_cli: %w", err)
		}
	}

	rawStdout := stdout.String()
	rawStderr := decorateMarsCLIStderr(stderr.String(), argv, cliArgs, exitCode)
	out, truncOut := capString(rawStdout, DefaultMaxToolOutputBytes/2)
	errOut, truncErr := capString(rawStderr, DefaultMaxToolOutputBytes/2)
	return ToolResult{
		Output:    out,
		Stderr:    errOut,
		ExitCode:  exitCode,
		Truncated: truncOut || truncErr,
	}, nil
}

func normalizeMarsArgs(root Root, args marsCLIArgs) ([]string, error) {
	if len(args.Args) == 0 {
		return nil, fmt.Errorf("mars_cli: args are required in run mode")
	}
	cliArgs := make([]string, len(args.Args))
	for i, arg := range args.Args {
		if arg == "" {
			return nil, fmt.Errorf("mars_cli: args[%d] must be non-empty", i)
		}
		cliArgs[i] = arg
	}
	repo := strings.TrimSpace(args.Repo)
	if repo == "" {
		return cliArgs, nil
	}
	if hasFlag(cliArgs, "--repo") {
		return cliArgs, nil
	}
	if !marsCommandSupportsRepo(cliArgs) {
		return nil, fmt.Errorf("mars_cli: repo was provided but command %q does not support --repo", strings.Join(cliArgs, " "))
	}
	resolved, err := root.ResolvePath(repo)
	if err != nil {
		return nil, fmt.Errorf("mars_cli: resolve repo: %w", err)
	}
	return appendRepoFlag(cliArgs, resolved), nil
}

func appendRepoFlag(args []string, repo string) []string {
	for i, arg := range args {
		if arg == "--" {
			out := make([]string, 0, len(args)+2)
			out = append(out, args[:i]...)
			out = append(out, "--repo", repo)
			out = append(out, args[i:]...)
			return out
		}
	}
	return append(args, "--repo", repo)
}

func marsCommandArgv(root Root, args []string) ([]string, error) {
	currentExe, currentErr := os.Executable()
	return marsCommandArgvWithExecutable(root, args, currentExe, currentErr)
}

func marsCommandArgvWithExecutable(root Root, args []string, currentExe string, currentErr error) ([]string, error) {
	if bin := strings.TrimSpace(os.Getenv("MARS_CLI_BIN")); bin != "" {
		return append([]string{bin}, args...), nil
	}
	if bin := strings.TrimSpace(os.Getenv("MARS_HARNESS_CLI_BIN")); bin != "" {
		return append([]string{bin}, args...), nil
	}
	if currentErr == nil && isLikelyMarsExecutable(currentExe) {
		return append([]string{currentExe}, args...), nil
	}
	if looksLikeMarsSource(root) {
		return append([]string{"go", "run", "./cmd/mars"}, args...), nil
	}
	if bin, err := exec.LookPath("mars"); err == nil {
		return append([]string{bin}, args...), nil
	}
	if bin, err := exec.LookPath("mars-harness"); err == nil {
		return append([]string{bin}, args...), nil
	}
	return nil, fmt.Errorf("mars_cli: mars binary not found in PATH; install it or set MARS_CLI_BIN")
}

func isLikelyMarsExecutable(path string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(path)))
	if base == "" || strings.HasSuffix(base, ".test") {
		return false
	}
	return strings.Contains(base, "mars")
}

func decorateMarsCLIStderr(stderr string, argv []string, cliArgs []string, exitCode int) string {
	if exitCode == 0 || len(argv) == 0 || len(cliArgs) == 0 {
		return stderr
	}
	if !strings.Contains(strings.ToLower(stderr), "unknown command") {
		return stderr
	}
	command := cliArgs[0]
	if len(cliArgs) > 1 && !strings.HasPrefix(cliArgs[1], "-") {
		command += " " + cliArgs[1]
	}
	guidance := fmt.Sprintf("mars_cli: resolved binary %q does not support command %q; set MARS_CLI_BIN to the active harness binary or run `mars update tool` before retrying.", argv[0], command)
	if strings.TrimSpace(stderr) == "" {
		return guidance + "\n"
	}
	if strings.HasSuffix(stderr, "\n") {
		return stderr + guidance + "\n"
	}
	return stderr + "\n" + guidance + "\n"
}

func looksLikeMarsSource(root Root) bool {
	for _, rel := range []string{
		filepath.Join("cmd", "mars", "main.go"),
		"go.mod",
	} {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			return false
		}
		if _, err := os.Stat(abs); err != nil {
			return false
		}
	}
	return true
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

func marsCommandSupportsRepo(args []string) bool {
	if len(args) == 0 {
		return false
	}
	command := args[0]
	sub := ""
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		sub = args[1]
	}
	switch command {
	case "init", "eject", "kill-switch", "uninstall", "upgrade", "scan", "doctor", "register", "start", "run":
		return true
	case "code-intel":
		return sub == "metrics" || sub == "benchmark"
	case "mcp":
		return sub == "serve"
	case "tools":
		return sub == "run"
	case "update":
		return sub == "check" || sub == "harness"
	case "release":
		return sub == "notes" || sub == "backfill-notes" || sub == "audit"
	case "checks":
		return sub == "run"
	case "docsync":
		return sub == "audit"
	case "models":
		if sub == "credentials" && len(args) > 2 {
			return args[2] == "write-local-env"
		}
		return sub == "evaluate" || sub == "override"
	case "guardrails":
		return sub == "secret-scan" || sub == "install-hooks"
	case "scores":
		return sub == "" || sub == "export"
	case "telemetry":
		return sub == "status" || sub == "preview" || sub == "export" || sub == "send" || sub == "triage-foundation"
	case "trust":
		return sub == ""
	default:
		return false
	}
}

func marsCLIReference() string {
	return MarsCLIReference()
}

// MarsCLIReference returns the reference text exposed by the mars_cli tool.
func MarsCLIReference() string {
	return strings.TrimSpace(`mars_cli reference

Purpose:
  Use this mirrored tool to let agents in both the foundation harness and deployed
  harness discover and execute the mars CLI without falling back to a
  generic shell. Pass argv exactly as it would appear after the mars
  binary name.

Modes:
  reference
    Return this exhaustive command reference. No subprocess is executed.
  run
    Execute mars with structured argv.
    Use ["tools", "run", <name>, "--args-json", "{...}"] when an LLM or
    operator needs a universal registered tool from outside an active agent run.

Tool arguments:
  mode: "reference" or "run". Omit mode to get reference when args is empty, or run when args is present.
  args: command arguments excluding the binary, e.g. ["doctor", "--repo", ".", "--json"].
  repo: optional repository path under the current workspace. Appends --repo <absolute path> for commands that support it.
  timeout_seconds: 1-600 seconds, default 60.
  background: true for long-running serve/start processes.

Global command surface:
  version
    Print version, OS/architecture, commit, and build date.
    Root shortcuts: ["--version"], ["-v"]
    Example: ["version"]

  setup
    First-time install: config, shell PATH, hardware detection, optional GitHub setup, model/binary download.
    Flags: --inference <local|cloud|defer>, --local-bundle <auto|local-cpu-q3|local-balanced-q4|local-quality-q8>, --download, --yes, --json, --plain, --skip-download, --github, --test-mode, --dry-run, --install-dir <dir>
    Example: ["setup", "--inference", "local", "--local-bundle", "auto", "--download", "--yes", "--json"]

  init
    Scaffold a deployed .harness/ bundle and starter docs in a target repository.
    Flags: --repo <path>, --force, --model-routing <local|cloud|defer>, --local-bundle <auto|local-cpu-q3|local-balanced-q4|local-quality-q8>, --cloud-provider <provider>, --cloud-model <model>, --cloud-endpoint <url>, --api-key-env <ENV>, --yes, --json, --plain
    Example: ["init", "--repo", ".", "--model-routing", "local", "--local-bundle", "auto", "--yes", "--json"]

  upgrade
    Fill missing generated target harness defaults while preserving target-owned files.
    Flags: --repo <path>
    Example: ["upgrade", "--repo", "."]

  eject
    Dry-run or apply the kill switch for a target repo: remove .harness/,
    generated harness docs, tickets, feature contracts, AGENTS.md, VERSION,
    CHANGELOG.md, and the associated per-repo SQLite database. Aliases:
    kill-switch, uninstall. Does not rewrite git history.
    Flags: --repo <path>, --db <path>, --apply, --confirm <repo-name>, --keep-db, --delete-shared-db
    Example: ["eject", "--repo", "."]
    Destructive example: ["eject", "--repo", ".", "--apply", "--confirm", "my-repo"]

  start
    Auto-init/register, reconcile existing lifecycle state, and run the autonomous orchestrator for one target repo. It resumes active jobs, stale recoverable jobs, in-progress/rework tickets, or recent deterministic dispositions before seeding CEO.
    Flags: --repo <path>, --remote <owner/repo>, --branch <branch>, --webhook-actor-id <numeric-id> (repeatable), --concurrency <n>, --db <path>, --force, --new-lifecycle, --debug, --log-file <path>, --code-intel <true|false>, --model-endpoint <real-url>, --addr <host:port>, --dashboard-addr <host:port> (both hosts must be literal loopback), --dashboard-trusted-origin <exact-https-origin>
    Use --new-lifecycle only when intentionally reseeding CEO over existing lifecycle state.
    --model-endpoint is only for real OpenAI-compatible endpoints; fake or scripted endpoints are test fixtures, not live validation evidence. Control/dashboard listeners accept only literal loopback addresses. Default scoped starts fall back to ephemeral local control/dashboard ports on conflict; the fallback host remains 127.0.0.1. Anonymous loopback dashboard access is limited to page/login shells, embedded assets, and path-free minimal status; rich reads, SSE, and mutation require environment-only MARS_DASHBOARD_CONTROL_SECRET (>=32 bytes) plus login, with exact Host/Origin and session CSRF for mutation. --dashboard-trusted-origin overrides MARS_DASHBOARD_TRUSTED_ORIGIN and permits authenticated remote browser access through that exact HTTPS reverse-proxy origin while MARS stays on loopback. GitHub webhook dispatch also requires a >=32-byte secret (MARS_WEBHOOK_SECRET first, then owner-only 0600 GitHub App credentials fallback), trusted numeric actor IDs from CLI/env/YAML, and an exact registered remote/branch; absent policy leaves local operation healthy with webhook ingress disabled.
    Long-running; use background:true when starting it from an agent.
    Example: ["start", "--repo", ".", "--concurrency", "1"]

  serve
    Run multi-repo orchestrator, dashboard, webhooks, scheduler, and workers.
    Flags: --addr <host:port> (literal loopback), --webhook-actor-id <numeric-id> (repeatable), --dashboard-trusted-origin <exact-https-origin>, --concurrency <n>, --db <path>, --debug, --log-file <path>, --code-intel <true|false>
	Control/dashboard listeners are loopback-only. Anonymous dashboard access is only page/login shells, assets, and minimal path-free status; rich reads, SSE, and mutation require environment-only MARS_DASHBOARD_CONTROL_SECRET (>=32 bytes) plus login, and mutation also requires exact Host/Origin and session CSRF. An exact HTTPS proxy origin may be selected by --dashboard-trusted-origin over MARS_DASHBOARD_TRUSTED_ORIGIN. GitHub webhook policy uses MARS_WEBHOOK_SECRET or the owner-only 0600 setup fallback plus actor IDs from CLI, MARS_WEBHOOK_ALLOWED_ACTOR_IDS, or webhook_allowed_actor_ids YAML; CLI actor policy overrides env, which overrides YAML.
    Long-running; use background:true when starting it from an agent.
    Example: ["serve", "--addr", "127.0.0.1:9091", "--concurrency", "2"]

  register
    Register a repository for autonomous management.
    Flags: --repo <path>, --remote <owner/repo>, --branch <branch>, --db <path>
    Example: ["register", "--repo", ".", "--remote", "owner/repo"]

  run <role>
    Manually execute one role against a target repository.
    Flags: --repo <path>, --model-endpoint <url>, --debug, --log-file <path>, --trace, --dry-run, --no-init, --code-intel <true|false>, --budget <tokens>, --max-turns <n>
    Default TTY output is a full-screen dashboard; --debug streams verbose trace/log output inline. --trace is kept as a run-only compatibility alias for debug-style trace detail. Use --dry-run --no-init for observer-safe inspection of uninitialized targets without scaffolding .harness/. Source work may run foundation-maintainer from the mars source repo to preview the source-only foundation operating context without creating a source manifest.
    Example: ["run", "engineer", "--repo", ".", "--dry-run"]

  scan
    Scan a repository for starter findings.
    Flags: --repo <path>, --tickets
    Example: ["scan", "--repo", ".", "--tickets"]

  code-intel metrics
    Summarize persisted local trace evidence for automatic code graph assistance:
    graph enabled/disabled/unavailable jobs, code-intel tool calls, broad
    exploration calls, context bytes, refreshes, LLM calls, tool invocations,
    token estimates, and mode sources. Reads the per-repo Mars SQLite DB and
    does not contact a model endpoint.
    Flags: --repo <path>, --db <path>, --window-days <n>, --json
    Example: ["code-intel", "metrics", "--repo", ".", "--window-days", "30"]

  code-intel benchmark
    Run a local no-model control/treatment measurement. Control disables graph
    context; treatment builds the graph context and optionally evaluates
    explicit changed-path fixtures against expected files, tests, and docs.
    Writes only the Mars SQLite DB and an optional report path outside the repo.
    Flags: --repo <path>, --db <path>, --case <name>, --trials <n>, --changed-paths <csv>, --expected-files <csv>, --expected-tests <csv>, --expected-docs <csv>, --report <path>, --json
    Example: ["code-intel", "benchmark", "--repo", ".", "--trials", "1", "--changed-paths", "internal/app/app.go"]

  doctor
    Diagnose config, models, database, private release auth, repo, and operating-model health.
    Flags: --config <path>, --db <path>, --repo <path>, --skip-remote, --json
    Example: ["doctor", "--repo", ".", "--json"]

  auth github check
    Check whether private MARS GitHub Release auth is ready without
    printing token values.
    Flags: --config <path>, --json
    Example: ["auth", "github", "check", "--json"]

  auth github setup
    Prepare private release auth for update tool. Prefer gh auth login, then
    run this command; setup saves a verified GitHub CLI fallback under
    ~/.mars. Headless installs may pass --token.
    Flags: --config <path>, --token <token>, --json
    Example: ["auth", "github", "setup"]

  tools list
    List every registered built-in tool. This is the universal tool catalog for
    both foundation and deployed harness operators.
    Flags: --json
    Example: ["tools", "list", "--json"]

  tools run <name>
    Execute one registered built-in tool through the same executor, allowlist,
    trust policy, repo root, and JSON argument path used by agent runs.
    Flags: --repo <path>, --args-json <json>, --allowlist <csv>, --role <name>, --trust <observer|contributor|autonomous>, --max-output-bytes <n>, --json
    Example: ["tools", "run", "tool_create", "--repo", ".", "--args-json", "{\"name\":\"cli_reference\",\"description\":\"Read CLI docs.\",\"fields\":[]}"]

  mcp serve
    Expose all registered MARS tools as a stdio MCP server for any
    MCP-compatible client or local harness agent.
    Flags: --repo <path>, --allowlist <csv>, --role <name>, --trust <observer|contributor|autonomous>, --max-output-bytes <n>
    Example client command: mars mcp serve --repo /path/to/repo --trust contributor

  update check
    Check installed CLI and deployed harness version drift.
    Flags: --repo <path>, --latest-release-url <url>, --skip-remote, --json
    Example: ["update", "check", "--repo", ".", "--json"]

  update tool
    Reinstall or upgrade the installed mars binary. Private release
    auth is checked with GH_TOKEN, GITHUB_TOKEN, GitHub CLI auth, then local
    config; run auth github setup when access is missing.
    Flags: --version <latest|tag|branch>, --install-dir <dir>, --source, --dry-run, --json
    Example: ["update", "tool", "--dry-run", "--json"]

  update harness
    Refresh missing generated target harness files.
    Flags: --repo <path>
    Example: ["update", "harness", "--repo", "."]

  path setup
    Configure shell PATH for the installed mars directory.
    Flags: --install-dir <dir>, --shell <name>, --dry-run, --json
    Example: ["path", "setup", "--dry-run", "--json"]

  release
    Show the release subcommands. Unexpected positional commands fail closed.
    Example: ["release"]

  release notes
    Generate semantic version patch notes, update VERSION, CHANGELOG.md, and buildinfo.
    Flags: --repo <path>, --bump <auto|major|minor|patch>, --dry-run
    Example: ["release", "notes", "--repo", ".", "--bump", "auto"]

  release backfill-notes
    Backfill historical CHANGELOG.md entries to the current Impact, Why, and What Changed format.
    Flags: --repo <path>, --min-version <X.Y.Z>, --max-version <X.Y.Z>, --dry-run, --check
    Example: ["release", "backfill-notes", "--repo", ".", "--max-version", "0.26.2", "--dry-run"]

  checks run
    Run a local check command and record checks_passed/checks_failed in the
    repo database so Mars can route failed checks to pipeline-fixer.
    Flags: --repo <path>, --db <path>, --name <check-name>, --role <role>
    Example: ["checks", "run", "--repo", ".", "--name", "unit", "--", "go", "test", "./..."]

  docsync audit
    Audit source-file MarsDocSync metadata and associated documentation pointers.
    Flags: --repo <path>, --json
    Example: ["docsync", "audit", "--repo", "."]

  validation agent-smoke
    Run compartmentalised role smoke tests against fresh ephemeral targets
    generated through foundation tooling, executing selected roles through the
    server job path in parallel. Successful runs are discarded by default;
    failed runs are retained unless --discard-failed is set. --model-endpoint
    is only for a real OpenAI-compatible model endpoint; fake or scripted
    endpoints are deterministic test plumbing, not validation evidence. Local
    runs default to --single-server so parallel cases share one llama-server.
    Each target receives docs/validation/agent-smoke/current-case.md as the
    target-local case contract. Default --max-turns is 32 for live execution.
    Flags: --role <role>, --case <id>, --project-type <type>, --suite <fast|default|full|held-out>, --parallel <n>, --cycle <key>, --max-turns <n>, --timeout <duration>, --model-endpoint <real-url>, --single-server, --single-server-tier <coding|reasoning|fast>, --fixture-only, --json, --report <path>, --keep-runs, --cleanup-only, --discard-failed, --root <path>
    Example: ["validation", "agent-smoke", "--suite", "fast", "--parallel", "2", "--json"]

  scores
    Show trunk-native accuracy scores.
    Flags: --repo <path>, --db <path>
    Example: ["scores", "--repo", "."]

  scores export
    Export repo quality score from telemetry/scoring evidence. Ticket materialization is opt-in.
    Flags: --repo <path>, --db <path>, --window-days <n>, --create-intervention-debt
    Example: ["scores", "export", "--repo", ".", "--window-days", "30"]

  telemetry status
    Show anonymous foundation telemetry reporting state and local outbox counts.
    Flags: --repo <path>, --db <path>
    Example: ["telemetry", "status", "--repo", "."]

  telemetry preview
    Print the exact sanitized anonymous aggregate payload without sending it.
    Flags: --repo <path>, --db <path>
    Example: ["telemetry", "preview", "--repo", "."]

  telemetry export
    Enqueue a sanitized anonymous aggregate payload in the local outbox.
    Flags: --repo <path>, --db <path>, --anonymous
    Example: ["telemetry", "export", "--repo", ".", "--anonymous"]

  telemetry send
    Send pending anonymous telemetry reports only when config telemetry.reporting is anonymous.
    Flags: --repo <path>, --db <path>
    Example: ["telemetry", "send", "--repo", "."]

  telemetry collect
    Run a local foundation telemetry collector.
    Flags: --addr <addr>, --storage sqlite, --db <path>
    Example: ["telemetry", "collect", "--addr", ":9092", "--storage", "sqlite", "--db", "~/.mars/db/foundation-telemetry/intake.db"]

  telemetry triage-foundation
    Create MARS source tickets from repeated anonymous collector patterns.
    Flags: --db <path>, --repo <path>, --window-days <n>
    Example: ["telemetry", "triage-foundation", "--db", "~/.mars/db/foundation-telemetry/intake.db", "--repo", "."]

  trust
    Show progressive autonomy trust levels.
    Flags: --repo <path>, --db <path>
    Example: ["trust", "--repo", "."]

  trust set <role> <repo> <observer|contributor|autonomous>
    Override a role trust level for a repo.
    Flags: --reason <text>, --db <path>
    Example: ["trust", "set", "engineer", "repo-id", "contributor", "--reason", "human approved"]

  models evaluate
    Print model evaluation plan or run live benchmark against an OpenAI-compatible/Ollama endpoint.
    Flags: --endpoint <url>, --model <name>, --provider <provider>, --repo <path>, --report-dir <path>, --save-report, --api-key-env <ENV>, --timeout <duration>, --json
    Example: ["models", "evaluate", "--json"]

  models eligible
    Show local model bundles eligible for detected hardware.
    Flags: --json
    Example: ["models", "eligible", "--json"]

  models list
    List registry defaults, local eligibility, or locally installed Ollama models.
    Flags: --provider <registry|ollama>, --eligible, --json
    Example: ["models", "list", "--eligible", "--json"]

  models override
    Set a repo-owned tier or role model override in .harness/model-overrides.yaml.
    Flags: --repo <path>, --tier <fast|reasoning|coding>, --role <name>, --routing <local|cloud|defer>, --local-bundle <bundle>, --provider <provider>, --model <name>, --endpoint <url>, --api-key-env <ENV>, --reason <text>, --json
    Example: ["models", "override", "--repo", ".", "--tier", "coding", "--provider", "ollama", "--model", "qwen3.6:27b"]

  models credentials write-local-env
    Read a provider key from the named process environment variable and write ignored .harness/.env.local with 0600 permissions; committed .harness/.env.example receives env names only.
    Flags: --repo <path>, --api-key-env <ENV>, --yes, --json
    Example: ["models", "credentials", "write-local-env", "--repo", ".", "--api-key-env", "ANTHROPIC_API_KEY", "--yes", "--json"]

  guardrails secret-scan
    Scan repository files or staged changes for common secret patterns with redacted output.
    Flags: --repo <path>, --staged, --json
    Example: ["guardrails", "secret-scan", "--repo", ".", "--staged", "--json"]

  guardrails install-hooks
    Install or update an optional managed pre-commit hook that runs guardrails secret-scan --staged.
    Flags: --repo <path>, --json
    Example: ["guardrails", "install-hooks", "--repo", ".", "--json"]

Operational guidance:
  Prefer --json when available for machine-readable output.
  Use repo:"." as shorthand for commands that operate on the current workspace.
  Use --dry-run before mutating setup/update/release operations when planning.
  Use background:true only for serve/start or deliberate long-running processes.
  Binary resolution prefers MARS_CLI_BIN, then the active running
  harness executable, then PATH, then source-checkout go run fallback.
  If a resolved binary rejects a known command, update the installed tool or set
  MARS_CLI_BIN to the active binary before retrying.
  This tool is mutating because many mars commands can write files,
  update trust, start workers, or change release state; observer trust blocks it.`)
}

// MarsCommandSupportsRepo reports whether the mars_cli repo shortcut
// can append a workspace --repo path for the provided mars argv.
func MarsCommandSupportsRepo(args []string) bool {
	return marsCommandSupportsRepo(args)
}
