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

const marsHarnessCLISchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "mode": {
      "type": "string",
      "enum": ["reference", "run"],
      "description": "Use reference for exhaustive CLI guidance, or run to execute mars-harness with structured argv"
    },
    "args": {
      "type": "array",
      "items": { "type": "string" },
      "description": "mars-harness arguments excluding the binary name, e.g. [\"doctor\", \"--repo\", \".\", \"--json\"]"
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
      "description": "Start mars-harness in the background. Use for long-running serve/start runs."
    }
  }
}`

type marsHarnessCLIArgs struct {
	Mode           string   `json:"mode"`
	Args           []string `json:"args"`
	Repo           string   `json:"repo"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Background     bool     `json:"background"`
}

func registerMarsHarnessCLI(r *Registry) error {
	return r.Register(
		"mars_harness_cli",
		"Read exhaustive mars-harness CLI reference or execute mars-harness commands with structured argv. This is the mirrored tool for controlling the foundation or deployed harness through the CLI.",
		json.RawMessage(marsHarnessCLISchema),
		handleMarsHarnessCLI,
	)
}

func handleMarsHarnessCLI(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args marsHarnessCLIArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("mars_harness_cli: parse arguments: %w", err)
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
		return ToolResult{Output: marsHarnessCLIReference()}, nil
	case "run":
		return runMarsHarnessCLI(ctx, root, args)
	default:
		return ToolResult{}, fmt.Errorf("mars_harness_cli: unsupported mode %q", args.Mode)
	}
}

func runMarsHarnessCLI(ctx context.Context, root Root, args marsHarnessCLIArgs) (ToolResult, error) {
	cliArgs, err := normalizeMarsHarnessArgs(root, args)
	if err != nil {
		return ToolResult{}, err
	}
	argv, err := marsHarnessCommandArgv(root, cliArgs)
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
			}, fmt.Errorf("mars_harness_cli: command timed out after %ds; use background:true for serve/start", timeoutSeconds)
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			return ToolResult{Output: stdout.String(), Stderr: stderr.String()}, fmt.Errorf("mars_harness_cli: %w", err)
		}
	}

	out, truncOut := capString(stdout.String(), DefaultMaxToolOutputBytes/2)
	errOut, truncErr := capString(stderr.String(), DefaultMaxToolOutputBytes/2)
	return ToolResult{
		Output:    out,
		Stderr:    errOut,
		ExitCode:  exitCode,
		Truncated: truncOut || truncErr,
	}, nil
}

func normalizeMarsHarnessArgs(root Root, args marsHarnessCLIArgs) ([]string, error) {
	if len(args.Args) == 0 {
		return nil, fmt.Errorf("mars_harness_cli: args are required in run mode")
	}
	cliArgs := make([]string, len(args.Args))
	for i, arg := range args.Args {
		if arg == "" {
			return nil, fmt.Errorf("mars_harness_cli: args[%d] must be non-empty", i)
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
	if !marsHarnessCommandSupportsRepo(cliArgs) {
		return nil, fmt.Errorf("mars_harness_cli: repo was provided but command %q does not support --repo", strings.Join(cliArgs, " "))
	}
	resolved, err := root.ResolvePath(repo)
	if err != nil {
		return nil, fmt.Errorf("mars_harness_cli: resolve repo: %w", err)
	}
	return append(cliArgs, "--repo", resolved), nil
}

func marsHarnessCommandArgv(root Root, args []string) ([]string, error) {
	if bin := strings.TrimSpace(os.Getenv("MARS_HARNESS_CLI_BIN")); bin != "" {
		return append([]string{bin}, args...), nil
	}
	if bin, err := exec.LookPath("mars-harness"); err == nil {
		return append([]string{bin}, args...), nil
	}
	if looksLikeMarsHarnessSource(root) {
		return append([]string{"go", "run", "./cmd/mars-harness"}, args...), nil
	}
	return nil, fmt.Errorf("mars_harness_cli: mars-harness binary not found in PATH; install it or set MARS_HARNESS_CLI_BIN")
}

func looksLikeMarsHarnessSource(root Root) bool {
	for _, rel := range []string{
		filepath.Join("cmd", "mars-harness", "main.go"),
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

func marsHarnessCommandSupportsRepo(args []string) bool {
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
	case "mcp":
		return sub == "serve"
	case "tools":
		return sub == "run"
	case "update":
		return sub == "check" || sub == "harness"
	case "release":
		return sub == "notes"
	case "scores":
		return sub == "" || sub == "export"
	default:
		return false
	}
}

func marsHarnessCLIReference() string {
	return strings.TrimSpace(`mars_harness_cli reference

Purpose:
  Use this mirrored tool to let agents in both the foundation harness and deployed
  harness discover and execute the mars-harness CLI without falling back to a
  generic shell. Pass argv exactly as it would appear after the mars-harness
  binary name.

Modes:
  reference
    Return this exhaustive command reference. No subprocess is executed.
  run
    Execute mars-harness with structured argv.
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
    Example: ["version"]

  setup
    First-time install: config, shell PATH, hardware detection, optional GitHub setup, model/binary download.
    Flags: --skip-download, --github, --test-mode, --dry-run, --install-dir <dir>
    Example: ["setup", "--test-mode", "--dry-run"]

  init
    Scaffold a deployed .harness/ bundle and starter docs in a target repository.
    Flags: --repo <path>, --force
    Example: ["init", "--repo", "."]

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
    Auto-init/register and run the autonomous orchestrator for one target repo.
    Flags: --repo <path>, --concurrency <n>, --db <path>, --force
    Long-running; use background:true when starting it from an agent.
    Example: ["start", "--repo", ".", "--concurrency", "1"]

  serve
    Run multi-repo orchestrator, dashboard, webhooks, scheduler, and workers.
    Flags: --addr <host:port>, --concurrency <n>, --db <path>
    Long-running; use background:true when starting it from an agent.
    Example: ["serve", "--addr", ":9091", "--concurrency", "2"]

  register
    Register a repository for autonomous management.
    Flags: --repo <path>, --remote <owner/repo>, --branch <branch>, --db <path>
    Example: ["register", "--repo", ".", "--remote", "owner/repo"]

  run <role>
    Manually execute one role against a target repository.
    Flags: --repo <path>, --model-endpoint <url>, --trace, --dry-run, --budget <tokens>, --max-turns <n>
    Example: ["run", "engineer", "--repo", ".", "--dry-run"]

  scan
    Scan a repository for starter findings.
    Flags: --repo <path>, --tickets
    Example: ["scan", "--repo", ".", "--tickets"]

  doctor
    Diagnose config, models, database, repo, and operating-model health.
    Flags: --config <path>, --db <path>, --repo <path>, --skip-remote, --json
    Example: ["doctor", "--repo", ".", "--json"]

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
    Expose all registered Mars Harness tools as a stdio MCP server for any
    MCP-compatible client or local harness agent.
    Flags: --repo <path>, --allowlist <csv>, --role <name>, --trust <observer|contributor|autonomous>, --max-output-bytes <n>
    Example client command: mars-harness mcp serve --repo /path/to/repo --trust contributor

  update check
    Check installed CLI and deployed harness version drift.
    Flags: --repo <path>, --latest-release-url <url>, --skip-remote, --json
    Example: ["update", "check", "--repo", ".", "--json"]

  update tool
    Reinstall or upgrade the installed mars-harness binary.
    Flags: --version <latest|tag|branch>, --install-dir <dir>, --source, --dry-run, --json
    Example: ["update", "tool", "--dry-run", "--json"]

  update harness
    Refresh missing generated target harness files.
    Flags: --repo <path>
    Example: ["update", "harness", "--repo", "."]

  path setup
    Configure shell PATH for the installed mars-harness directory.
    Flags: --install-dir <dir>, --shell <name>, --dry-run, --json
    Example: ["path", "setup", "--dry-run", "--json"]

  release notes
    Generate semantic version patch notes, update VERSION, CHANGELOG.md, and buildinfo.
    Flags: --repo <path>, --bump <auto|major|minor|patch>, --dry-run
    Example: ["release", "notes", "--repo", ".", "--bump", "auto"]

  release verify-assets
    Verify release metadata/assets for the updater.
    Flags: --repo <owner/name>, --version <latest|tag>, --release-url <url>, --json
    Example: ["release", "verify-assets", "--version", "latest", "--json"]

  scores export
    Export repo quality score from telemetry/scoring evidence.
    Flags: --repo <path>, --db <path>, --window-days <n>, --no-ticket
    Example: ["scores", "export", "--repo", ".", "--window-days", "30"]

  trust set <role> <repo> <observer|contributor|autonomous>
    Override a role trust level for a repo.
    Flags: --reason <text>, --db <path>
    Example: ["trust", "set", "engineer", "repo-id", "contributor", "--reason", "human approved"]

  models evaluate
    Print model evaluation plan or run live benchmark against an OpenAI-compatible/Ollama endpoint.
    Flags: --endpoint <url>, --model <name>, --provider <openai-compatible|ollama>, --repo <path>, --report-dir <path>, --save-report, --api-key <key>, --timeout <duration>, --json
    Example: ["models", "evaluate", "--json"]

  models list
    List registry defaults or locally installed Ollama models.
    Flags: --provider <registry|ollama>, --json
    Example: ["models", "list", "--provider", "ollama", "--json"]

  models override
    Set a repo-owned tier or role model override in .harness/model-overrides.yaml.
    Flags: --repo <path>, --tier <fast|reasoning|coding>, --role <name>, --provider <ollama|openai-compatible>, --model <name>, --endpoint <url>, --reason <text>, --json
    Example: ["models", "override", "--repo", ".", "--tier", "coding", "--provider", "ollama", "--model", "qwen3.6:27b"]

Operational guidance:
  Prefer --json when available for machine-readable output.
  Use repo:"." as shorthand for commands that operate on the current workspace.
  Use --dry-run before mutating setup/update/release operations when planning.
  Use background:true only for serve/start or deliberate long-running processes.
  This tool is mutating because many mars-harness commands can write files,
  update trust, start workers, or change release state; observer trust blocks it.`)
}
