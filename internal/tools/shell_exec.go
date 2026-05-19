/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const shellExecSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "argv": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Program path and arguments (no shell); mutually exclusive with shell_command"
    },
    "shell_command": {
      "type": "string",
      "description": "Single command string run via /bin/sh -c when argv is not used"
    },
    "timeout_seconds": {
      "type": "integer",
      "description": "Per-invocation timeout in seconds (1–300, default 30)"
    },
    "background": {
      "type": "boolean",
      "description": "Start as a background process. Returns immediately with the PID and first 2 seconds of output. Use for dev servers and watchers."
    }
  }
}`

type shellExecArgs struct {
	Argv           []string `json:"argv"`
	ShellCommand   string   `json:"shell_command"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Background     bool     `json:"background"`
}

// bgProcs tracks background processes started by shell_exec so they can
// be killed when the agent job finishes.
var (
	bgMu    sync.Mutex
	bgProcs = map[int]*exec.Cmd{}
)

// KillBackgroundProcs terminates all tracked background processes. Called
// by the executor when a job ends to prevent orphan dev servers.
func KillBackgroundProcs() {
	bgMu.Lock()
	defer bgMu.Unlock()
	for pid, cmd := range bgProcs {
		slog.Info("shell_exec: killing background process", "pid", pid)
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = cmd.Wait()
		delete(bgProcs, pid)
	}
}

func registerShellExec(r *Registry) error {
	return r.Register("shell_exec", "Run a subprocess with repository root as working directory. Prefer argv; use shell_command only when shell features are required. Set background:true for long-running processes like dev servers.", json.RawMessage(shellExecSchema), handleShellExec)
}

func handleShellExec(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args shellExecArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: parse arguments: %w", err)
	}
	hasArgv := len(args.Argv) > 0
	hasShell := strings.TrimSpace(args.ShellCommand) != ""
	if hasArgv == hasShell {
		return ToolResult{}, fmt.Errorf("shell_exec: provide exactly one of argv (non-empty) or shell_command")
	}
	if hasArgv && args.Argv[0] == "" {
		return ToolResult{}, fmt.Errorf("shell_exec: argv[0] must be non-empty")
	}
	if hasArgv {
		if err := validateShellExecArgv(args.Argv); err != nil {
			return ToolResult{}, err
		}
	}

	if args.Background {
		return execBackground(root, args)
	}
	return execForeground(ctx, root, args)
}

func validateShellExecArgv(argv []string) error {
	if len(argv) == 0 {
		return nil
	}
	program := strings.Trim(filepathBase(strings.TrimSpace(argv[0])), `"'`)
	if shellArgvBuiltin(program) {
		return shellArgvSyntaxError(argv[0])
	}
	for _, arg := range argv {
		token := strings.Trim(strings.TrimSpace(arg), `"'`)
		if token == "" {
			continue
		}
		if shellArgvControlToken(token) || shellArgvLooksLikeRedirection(token) || strings.Contains(token, "$(") || strings.Contains(token, "`") || strings.Contains(token, "\n") {
			return shellArgvSyntaxError(arg)
		}
	}
	return nil
}

func shellArgvBuiltin(program string) bool {
	switch strings.ToLower(program) {
	case ":", ".", "cd", "source", "alias", "export", "unset", "set", "ulimit", "jobs", "fg", "bg", "dirs", "pushd", "popd":
		return true
	default:
		return false
	}
}

func shellArgvControlToken(token string) bool {
	switch token {
	case "|", "||", "&&", ";", "&", "(", ")":
		return true
	default:
		return false
	}
}

func shellArgvLooksLikeRedirection(token string) bool {
	token = strings.TrimLeft(token, "0123456789")
	return strings.HasPrefix(token, ">") || strings.HasPrefix(token, "<")
}

func shellArgvSyntaxError(token string) error {
	return fmt.Errorf("shell_exec: argv mode cannot run shell syntax token %q; argv runs one executable without shell parsing, so redirection, pipes, control operators, and shell builtins will not work. Use shell_command when shell syntax is required, or use file_write/file_read for file content changes", token)
}

func execForeground(ctx context.Context, root Root, args shellExecArgs) (ToolResult, error) {
	ts := args.TimeoutSeconds
	if ts <= 0 {
		ts = 30
	}
	if ts > 300 {
		ts = 300
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(ts)*time.Second)
	defer cancel()

	cmd := buildCmd(runCtx, root, args)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exit := 0
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return ToolResult{
				Stderr:   stderr.String(),
				Output:   stdout.String(),
				ExitCode: -1,
			}, fmt.Errorf("shell_exec: command timed out after %ds — use background:true for long-running processes", ts)
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			return ToolResult{
				Stderr: stderr.String(),
				Output: stdout.String(),
			}, fmt.Errorf("shell_exec: %w", err)
		}
	}

	outStr, truncOut := capString(stdout.String(), DefaultMaxToolOutputBytes/2)
	errStr, truncErr := capString(stderr.String(), DefaultMaxToolOutputBytes/2)
	return ToolResult{
		Output:    outStr,
		Stderr:    errStr,
		ExitCode:  exit,
		Truncated: truncOut || truncErr,
	}, nil
}

const bgCaptureWindow = 2 * time.Second

func execBackground(root Root, args shellExecArgs) (ToolResult, error) {
	cmd := buildCmd(context.Background(), root, args)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: start background: %w", err)
	}
	pid := cmd.Process.Pid

	bgMu.Lock()
	bgProcs[pid] = cmd
	bgMu.Unlock()

	// Reap in background and unregister on exit.
	go func() {
		_ = cmd.Wait()
		bgMu.Lock()
		delete(bgProcs, pid)
		bgMu.Unlock()
		slog.Debug("shell_exec: background process exited", "pid", pid)
	}()

	// Capture initial output for the capture window so the agent sees
	// startup messages (e.g. "ready on http://localhost:3000").
	var stdoutBuf, stderrBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutPipe)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(&stderrBuf, stderrPipe)
		done <- struct{}{}
	}()

	timer := time.NewTimer(bgCaptureWindow)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-done:
	}

	initial := strings.TrimSpace(stdoutBuf.String() + "\n" + stderrBuf.String())
	outStr, _ := capString(initial, DefaultMaxToolOutputBytes/2)

	return ToolResult{
		Output: fmt.Sprintf("Started in background (PID %d)\n%s", pid, outStr),
	}, nil
}

func buildCmd(ctx context.Context, root Root, args shellExecArgs) *exec.Cmd {
	var cmd *exec.Cmd
	if len(args.Argv) > 0 {
		cmd = exec.CommandContext(ctx, args.Argv[0], args.Argv[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", args.ShellCommand)
	}
	cmd.Dir = root.Abs()
	return cmd
}

func capString(s string, max int) (string, bool) {
	if max <= 0 {
		max = DefaultMaxToolOutputBytes
	}
	if len(s) <= max {
		return s, false
	}
	out, _ := TruncateUTF8(s, max)
	return out, true
}
