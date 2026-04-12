package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
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
    }
  }
}`

type shellExecArgs struct {
	Argv           []string `json:"argv"`
	ShellCommand   string   `json:"shell_command"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

func registerShellExec(r *Registry) error {
	return r.Register("shell_exec", "Run a subprocess with repository root as working directory. Prefer argv; use shell_command only when shell features are required.", json.RawMessage(shellExecSchema), handleShellExec)
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
	ts := args.TimeoutSeconds
	if ts <= 0 {
		ts = 30
	}
	if ts > 300 {
		ts = 300
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(ts)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if hasArgv {
		cmd = exec.CommandContext(runCtx, args.Argv[0], args.Argv[1:]...)
	} else {
		cmd = exec.CommandContext(runCtx, "/bin/sh", "-c", args.ShellCommand)
	}
	cmd.Dir = root.Abs()
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
			}, fmt.Errorf("shell_exec: command timed out after %ds — long-running servers (e.g. dev servers) should not be started by agents", ts)
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
