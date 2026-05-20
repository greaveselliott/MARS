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
	"strconv"
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
      "description": "Start as a background process. Returns after a short startup window with the PID and initial output. Use for dev servers and watchers; startup exits are reported as errors."
    }
  }
}`

type shellExecArgs struct {
	Argv           []string `json:"argv"`
	ShellCommand   string   `json:"shell_command"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Background     bool     `json:"background"`
}

type rawShellExecArgs struct {
	Argv           json.RawMessage `json:"argv"`
	ShellCommand   string          `json:"shell_command"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	Background     bool            `json:"background"`
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
	procs := make(map[int]*exec.Cmd, len(bgProcs))
	for pid, cmd := range bgProcs {
		procs[pid] = cmd
		delete(bgProcs, pid)
	}
	bgMu.Unlock()

	for pid, cmd := range procs {
		slog.Info("shell_exec: killing background process", "pid", pid)
		killBackgroundProcessTree(pid)
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}

func maybeHandleTrackedBackgroundKill(args shellExecArgs) (bool, ToolResult) {
	if len(args.Argv) < 2 || strings.TrimSpace(args.ShellCommand) != "" {
		return false, ToolResult{}
	}
	if filepathBase(args.Argv[0]) != "kill" {
		return false, ToolResult{}
	}
	var killed []string
	for _, arg := range args.Argv[1:] {
		arg = strings.TrimSpace(strings.Trim(arg, `"'`))
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		pid, err := strconv.Atoi(arg)
		if err != nil {
			continue
		}
		if killTrackedBackgroundPID(pid) {
			killed = append(killed, strconv.Itoa(pid))
		}
	}
	if len(killed) == 0 {
		return false, ToolResult{}
	}
	return true, ToolResult{Output: fmt.Sprintf("Killed background process tree for PID(s): %s", strings.Join(killed, ", "))}
}

func killTrackedBackgroundPID(pid int) bool {
	bgMu.Lock()
	cmd, ok := bgProcs[pid]
	if ok {
		delete(bgProcs, pid)
	}
	bgMu.Unlock()
	if !ok {
		return false
	}
	slog.Info("shell_exec: killing tracked background process tree", "pid", pid)
	killBackgroundProcessTree(pid)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return true
}

func killBackgroundProcessTree(pid int) {
	descendants := processDescendants(pid)
	for i := len(descendants) - 1; i >= 0; i-- {
		killProcessGroupOrProcess(descendants[i])
	}
	killProcessGroupOrProcess(pid)
}

func killProcessGroupOrProcess(pid int) {
	if pid <= 0 {
		return
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err == nil {
		return
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

func processDescendants(pid int) []int {
	out, err := exec.Command("ps", "-eo", "pid=,ppid=").Output()
	if err != nil {
		return nil
	}
	children := map[int][]int{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		child, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		parent, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		children[parent] = append(children[parent], child)
	}
	var descendants []int
	var walk func(int)
	walk = func(parent int) {
		for _, child := range children[parent] {
			descendants = append(descendants, child)
			walk(child)
		}
	}
	walk(pid)
	return descendants
}

func registerShellExec(r *Registry) error {
	return r.Register("shell_exec", "Run a subprocess with repository root as working directory. Prefer argv; use shell_command only when shell features are required. Set background:true for long-running processes like dev servers.", json.RawMessage(shellExecSchema), handleShellExec)
}

func handleShellExec(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
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
		if err := validateShellExecGitRemoteMutation(args.Argv, "argv"); err != nil {
			return ToolResult{}, err
		}
	} else {
		if err := validateShellExecShellCommand(args.ShellCommand); err != nil {
			return ToolResult{}, err
		}
		if err := validateShellExecGitRemoteMutation(strings.Fields(args.ShellCommand), "shell_command"); err != nil {
			return ToolResult{}, err
		}
	}

	if args.Background {
		return execBackground(root, args)
	}
	if handled, res := maybeHandleTrackedBackgroundKill(args); handled {
		return res, nil
	}
	return execForeground(ctx, root, args)
}

func decodeShellExecArgs(raw json.RawMessage) (shellExecArgs, error) {
	var rawArgs rawShellExecArgs
	if err := json.Unmarshal(raw, &rawArgs); err != nil {
		return shellExecArgs{}, err
	}
	args := shellExecArgs{
		ShellCommand:   rawArgs.ShellCommand,
		TimeoutSeconds: rawArgs.TimeoutSeconds,
		Background:     rawArgs.Background,
	}
	if len(rawArgs.Argv) == 0 || bytes.Equal(bytes.TrimSpace(rawArgs.Argv), []byte("null")) {
		return args, nil
	}
	argv, err := decodeStringSliceArg(rawArgs.Argv, "shell_exec.argv")
	if err != nil {
		return shellExecArgs{}, err
	}
	args.Argv = argv
	args.Argv = normalizeShellExecArgv(args.Argv)
	return args, nil
}

func normalizeShellExecArgv(argv []string) []string {
	if len(argv) != 1 {
		return argv
	}
	only := strings.TrimSpace(argv[0])
	if !simpleSingleArgvCommand(only) {
		return argv
	}
	return strings.Fields(only)
}

func simpleSingleArgvCommand(s string) bool {
	if !strings.ContainsAny(s, " \t") {
		return false
	}
	if strings.ContainsAny(s, "\n\"'`$;&|<>") {
		return false
	}
	fields := strings.Fields(s)
	return len(fields) > 1
}

func validateShellExecArgv(argv []string) error {
	if len(argv) == 0 {
		return nil
	}
	program := strings.Trim(filepathBase(strings.TrimSpace(argv[0])), `"'`)
	if shellExecExternalTimeoutCommand(program) {
		return shellExecExternalTimeoutError(program)
	}
	if shellExecBarePortToken(program) {
		return shellExecBarePortError(program)
	}
	if shellArgvBuiltin(program) {
		return shellArgvSyntaxError(argv[0])
	}
	for _, arg := range argv {
		token := strings.Trim(strings.TrimSpace(arg), `"'`)
		if token == "" {
			continue
		}
		if shellArgvControlToken(token) || shellArgvLooksLikeRedirection(token) || shellArgvContainsControlSyntax(token) || strings.Contains(token, "$(") || strings.Contains(token, "`") || strings.Contains(token, "\n") {
			return shellArgvSyntaxError(arg)
		}
	}
	return nil
}

func validateShellExecGitRemoteMutation(argv []string, mode string) error {
	if len(argv) < 3 {
		return nil
	}
	if argv[0] != "git" || argv[1] != "remote" {
		return nil
	}
	switch argv[2] {
	case "add", "set-url", "remove", "rm", "rename":
		return fmt.Errorf("shell_exec: git remote %s is blocked in %s mode; agents must not invent or rewrite repository remotes. Configure remotes outside the harness, or record a release blocker when no remote is available", argv[2], mode)
	default:
		return nil
	}
}

func validateShellExecShellCommand(cmd string) error {
	if shellCommandHasBackgroundOperator(cmd) {
		return fmt.Errorf("shell_exec: shell_command cannot use the shell background operator & because it can leak child processes after timeouts. Start the long-running command with background:true instead, then run a separate probe such as curl, and rely on harness cleanup or a targeted kill after validation")
	}
	fields := strings.Fields(strings.TrimSpace(cmd))
	if len(fields) > 0 {
		program := strings.Trim(filepathBase(strings.TrimSpace(fields[0])), `"'`)
		if shellExecExternalTimeoutCommand(program) {
			return shellExecExternalTimeoutError(program)
		}
	}
	if len(fields) == 1 {
		token := strings.Trim(fields[0], `"'`)
		if shellExecBarePortToken(token) {
			return shellExecBarePortError(token)
		}
	}
	return nil
}

func shellExecExternalTimeoutCommand(program string) bool {
	switch strings.ToLower(strings.TrimSpace(program)) {
	case "timeout", "gtimeout":
		return true
	default:
		return false
	}
}

func shellExecExternalTimeoutError(program string) error {
	return fmt.Errorf("shell_exec: external timeout command %q is not portable inside harness-managed validation. Use shell_exec timeout_seconds for bounded foreground commands, or start long-running servers with background:true and probe them separately", program)
}

func shellExecBarePortToken(token string) bool {
	if len(token) < 2 || token[0] != ':' {
		return false
	}
	for _, r := range token[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shellExecBarePortError(token string) error {
	return fmt.Errorf("shell_exec: %q is a port, not an executable command. Start the app with its real server command using background:true, then probe it separately with curl http://localhost%s/health or the target route", token, token)
}

func shellCommandHasBackgroundOperator(cmd string) bool {
	var inSingle, inDouble, escaped bool
	for i, r := range cmd {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '&':
			if inSingle || inDouble {
				continue
			}
			prev, next := byte(0), byte(0)
			if i > 0 {
				prev = cmd[i-1]
			}
			if i+1 < len(cmd) {
				next = cmd[i+1]
			}
			if prev == '&' || next == '&' || prev == '>' || next == '>' {
				continue
			}
			return true
		}
	}
	return false
}

func shellArgvBuiltin(program string) bool {
	switch strings.ToLower(program) {
	case ":", ".", "cd", "source", "alias", "export", "unset", "set", "ulimit", "jobs", "fg", "bg", "dirs", "pushd", "popd":
		return true
	default:
		return false
	}
}

func shellArgvContainsControlSyntax(token string) bool {
	for _, syntax := range []string{"&&", "||", "|", ";", ">", "<"} {
		if strings.Contains(token, syntax) {
			return true
		}
	}
	return false
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
	cmd.WaitDelay = 2 * time.Second
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
	exitCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		bgMu.Lock()
		delete(bgProcs, pid)
		bgMu.Unlock()
		slog.Debug("shell_exec: background process exited", "pid", pid)
		exitCh <- err
	}()

	// Capture initial output for the capture window so the agent sees
	// startup messages (e.g. "ready on http://localhost:3000").
	var stdoutBuf, stderrBuf bytes.Buffer
	done := make(chan struct{}, 2)
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

	var exitErr error
	exited := false
	select {
	case <-timer.C:
	case exitErr = <-exitCh:
		exited = true
	}

	if exited {
		for i := 0; i < 2; i++ {
			select {
			case <-done:
			case <-time.After(100 * time.Millisecond):
				i = 2
			}
		}
	}

	outStr, truncOut := capString(stdoutBuf.String(), DefaultMaxToolOutputBytes/2)
	errStr, truncErr := capString(stderrBuf.String(), DefaultMaxToolOutputBytes/2)
	if exited {
		exitCode := 0
		if exitErr != nil {
			exitCode = -1
			var ee *exec.ExitError
			if errors.As(exitErr, &ee) {
				exitCode = ee.ExitCode()
			}
		}
		return ToolResult{
			Output:    strings.TrimSpace(fmt.Sprintf("Background process (PID %d) exited during startup\n%s", pid, strings.TrimSpace(outStr))),
			Stderr:    strings.TrimSpace(errStr),
			ExitCode:  exitCode,
			Truncated: truncOut || truncErr,
		}, fmt.Errorf("shell_exec: background process exited during startup with exit code %d; inspect output, fix the command, or run a long-running server with background:true and probe it separately", exitCode)
	}

	initial := strings.TrimSpace(outStr + "\n" + errStr)
	return ToolResult{
		Output:    fmt.Sprintf("Started in background (PID %d)\n%s", pid, initial),
		Truncated: truncOut || truncErr,
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
