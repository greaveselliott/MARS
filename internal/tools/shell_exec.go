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
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/greaveselliott/mars/internal/childenv"
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
    },
    "expected_exit_code": {
      "type": "integer",
      "description": "Expected process exit code for validation probes. Omit for normal success. Use a non-zero value only when intentionally testing an error path."
    }
  }
}`

type shellExecArgs struct {
	Argv             []string `json:"argv"`
	ShellCommand     string   `json:"shell_command"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
	Background       bool     `json:"background"`
	ExpectedExitCode *int     `json:"expected_exit_code"`
}

type rawShellExecArgs struct {
	Argv             json.RawMessage `json:"argv"`
	ShellCommand     string          `json:"shell_command"`
	TimeoutSeconds   int             `json:"timeout_seconds"`
	Background       bool            `json:"background"`
	ExpectedExitCode *int            `json:"expected_exit_code"`
}

// bgProcs tracks background processes started by shell_exec so they can be
// listed, stopped, and cleaned up only by the job that started them.
var (
	bgMu    sync.Mutex
	bgProcs = map[int]*backgroundProcess{}
)

type backgroundProcess struct {
	jobID string
	cmd   *exec.Cmd
	done  chan struct{}
}

type lockedBuffer struct {
	mu  sync.Mutex
	b   bytes.Buffer
	max int
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.max <= 0 {
		_, _ = b.b.Write(p)
		return len(p), nil
	}
	remaining := b.max - b.b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = b.b.Write(p[:remaining])
		} else {
			_, _ = b.b.Write(p)
		}
	}
	return len(p), nil
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// CleanupBackgroundProcesses terminates background processes owned by jobID.
// It never touches another job's process or an untracked host process.
func CleanupBackgroundProcesses(jobID string) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return
	}
	bgMu.Lock()
	procs := make(map[int]*backgroundProcess)
	for pid, proc := range bgProcs {
		if proc.jobID != jobID {
			continue
		}
		procs[pid] = proc
		delete(bgProcs, pid)
	}
	bgMu.Unlock()

	for pid, proc := range procs {
		stopBackgroundProcess(pid, proc)
	}
}

func handleTrackedBackgroundKill(jobID string, args shellExecArgs) (ToolResult, error) {
	pids, err := shellExecKillPIDs(args)
	if err != nil {
		return ToolResult{}, err
	}
	bgMu.Lock()
	procs := make(map[int]*backgroundProcess, len(pids))
	for _, pid := range pids {
		proc, ok := bgProcs[pid]
		if !ok || proc.jobID != jobID {
			bgMu.Unlock()
			return ToolResult{}, fmt.Errorf("shell_exec: refusing to signal PID %d because it is not a background process owned by job %q; stop only a PID returned by this job's background:true call", pid, jobID)
		}
		procs[pid] = proc
	}
	for _, pid := range pids {
		delete(bgProcs, pid)
	}
	bgMu.Unlock()

	for _, pid := range pids {
		stopBackgroundProcess(pid, procs[pid])
	}
	killed := make([]string, 0, len(pids))
	for _, pid := range pids {
		killed = append(killed, strconv.Itoa(pid))
	}
	return ToolResult{Output: fmt.Sprintf("Stopped job-owned background process group(s) for PID(s): %s", strings.Join(killed, ", "))}, nil
}

func stopBackgroundProcess(pid int, proc *backgroundProcess) {
	if pid <= 0 || proc == nil {
		return
	}
	slog.Info("shell_exec: stopping job-owned background process group", "job_id", proc.jobID, "pid", pid)
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && proc.cmd.Process != nil {
		_ = proc.cmd.Process.Signal(syscall.SIGTERM)
	}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for processGroupExists(pid) {
		select {
		case <-ticker.C:
		case <-timer.C:
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && proc.cmd.Process != nil {
				_ = proc.cmd.Process.Kill()
			}
			return
		}
	}
}

func processGroupExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(-pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func registerShellExec(r *Registry) error {
	return r.Register("shell_exec", "Run a subprocess with repository root as working directory. Prefer argv; use shell_command only when shell features are required. Set background:true for long-running processes like dev servers.", json.RawMessage(shellExecSchema), handleShellExec)
}

func handleShellExec(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: parse arguments: %w", err)
	}
	session, hasSession := SessionFromContext(ctx)
	jobID := strings.TrimSpace(session.JobID)
	if shellExecNoop(args) {
		return shellExecNoopResult(jobID), fmt.Errorf("shell_exec: no-op command cannot advance work")
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
	if err := validateShellExecProcessControl(args); err != nil {
		return ToolResult{}, err
	}
	if hasArgv && filepathBase(args.Argv[0]) == "kill" {
		if !hasSession || jobID == "" {
			return ToolResult{}, fmt.Errorf("shell_exec: kill requires a non-empty job session and a PID returned by that job's background:true call")
		}
		return handleTrackedBackgroundKill(jobID, args)
	}

	if args.Background {
		if !hasSession || jobID == "" {
			return ToolResult{}, fmt.Errorf("shell_exec: background:true requires a non-empty Session.JobID so the process can be scoped and cleaned up safely")
		}
		return execBackground(root, args, jobID)
	}
	return execForeground(ctx, root, args)
}

func decodeShellExecArgs(raw json.RawMessage) (shellExecArgs, error) {
	var rawArgs rawShellExecArgs
	if err := json.Unmarshal(raw, &rawArgs); err != nil {
		return shellExecArgs{}, err
	}
	args := shellExecArgs{
		ShellCommand:     rawArgs.ShellCommand,
		TimeoutSeconds:   rawArgs.TimeoutSeconds,
		Background:       rawArgs.Background,
		ExpectedExitCode: rawArgs.ExpectedExitCode,
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
	if strings.TrimSpace(args.ShellCommand) == "" {
		if shellCommand, ok := normalizeShellExecCdValidationArgv(args.Argv); ok {
			args.ShellCommand = shellCommand
			args.Argv = nil
		}
	}
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

func normalizeShellExecCdValidationArgv(argv []string) (string, bool) {
	if len(argv) < 5 || strings.ToLower(strings.TrimSpace(argv[0])) != "cd" || strings.TrimSpace(argv[2]) != "&&" {
		return "", false
	}
	dir := strings.TrimSpace(argv[1])
	if !shellExecSimpleToken(dir) {
		return "", false
	}
	rhs := make([]string, 0, len(argv)-3)
	for _, arg := range argv[3:] {
		arg = strings.TrimSpace(arg)
		if !shellExecSimpleToken(arg) {
			return "", false
		}
		rhs = append(rhs, arg)
	}
	if !shellFieldsRunTestCommand(rhs) && !shellFieldsRunBuildCommand(rhs) {
		return "", false
	}
	return "cd " + dir + " && " + strings.Join(rhs, " "), true
}

func shellExecSimpleToken(token string) bool {
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return false
	}
	if shellArgvControlToken(token) || shellArgvLooksLikeRedirection(token) || shellArgvContainsControlSyntax(token) || strings.Contains(token, "$(") || strings.Contains(token, "`") {
		return false
	}
	return true
}

func shellExecNoop(args shellExecArgs) bool {
	if args.Background {
		return false
	}
	shell := strings.TrimSpace(args.ShellCommand)
	if shell != "" {
		return strings.Trim(shell, `"'`) == ":"
	}
	var nonEmpty []string
	for _, arg := range args.Argv {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			nonEmpty = append(nonEmpty, arg)
		}
	}
	if len(nonEmpty) == 0 {
		return true
	}
	return len(nonEmpty) == 1 && strings.Trim(nonEmpty[0], `"'`) == ":"
}

func shellExecNoopResult(jobID string) ToolResult {
	pids := trackedBackgroundPIDs(jobID)
	var b strings.Builder
	b.WriteString("No command was run. shell_exec no-op calls do not wait for background processes or finish ticket work.")
	if len(pids) > 0 {
		b.WriteString(" Active background PID(s): ")
		for i, pid := range pids {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strconv.Itoa(pid))
		}
		b.WriteString(". Probe the intended route, then stop the tracked PID with shell_exec argv [\"kill\",\"<pid>\"] before committing.")
	}
	b.WriteString(" Do not retry empty argv or ':' calls. If an implementation ticket is claimed but validation has not started, read the in-progress ticket and feature contract, then use file_write to implement or record job_disposition_record with status blocked. If validation is complete or the worktree is dirty, use git_status, update ticket evidence, git_commit the work, move the ticket to done when appropriate, commit the lifecycle move, push, and call job_disposition_record.")
	return ToolResult{Output: b.String()}
}

func trackedBackgroundPIDs(jobID string) []int {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil
	}
	bgMu.Lock()
	defer bgMu.Unlock()
	pids := make([]int, 0, len(bgProcs))
	for pid, proc := range bgProcs {
		if proc.jobID != jobID {
			continue
		}
		pids = append(pids, pid)
	}
	slices.Sort(pids)
	return pids
}

func validateShellExecProcessControl(args shellExecArgs) error {
	if strings.TrimSpace(args.ShellCommand) != "" && shellCommandUsesProcessControl(args.ShellCommand) {
		return fmt.Errorf("shell_exec: shell-form process control is blocked because ownership cannot be validated; use argv [\"kill\",\"<pid>\"] with a PID returned by this job's background:true call")
	}
	if len(args.Argv) == 0 {
		return nil
	}
	switch filepathBase(args.Argv[0]) {
	case "pkill", "killall":
		return fmt.Errorf("shell_exec: %s is blocked because it can select processes outside the current job; stop only an exact PID returned by this job's background:true call with argv [\"kill\",\"<pid>\"]", filepathBase(args.Argv[0]))
	case "kill":
		if args.Background {
			return fmt.Errorf("shell_exec: kill cannot run with background:true; stop the exact job-owned PID in the foreground")
		}
		_, err := shellExecKillPIDs(args)
		return err
	default:
		return nil
	}
}

func shellCommandUsesProcessControl(command string) bool {
	segments := strings.FieldsFunc(command, func(r rune) bool {
		return r == '\n' || r == ';' || r == '&' || r == '|' || r == '(' || r == ')'
	})
	for _, segment := range segments {
		for _, field := range strings.Fields(segment) {
			field = strings.Trim(field, `"'`)
			if field == "" || shellCommandPrefixToken(field) {
				continue
			}
			switch filepathBase(field) {
			case "kill", "pkill", "killall":
				return true
			}
			break
		}
	}
	return false
}

func shellCommandPrefixToken(token string) bool {
	if strings.HasPrefix(token, "-") {
		return true
	}
	if i := strings.IndexByte(token, '='); i > 0 && !strings.ContainsAny(token[:i], "/ ") {
		return true
	}
	switch filepathBase(token) {
	case "command", "builtin", "exec", "env", "nohup", "sudo":
		return true
	default:
		return false
	}
}

func shellExecKillPIDs(args shellExecArgs) ([]int, error) {
	if len(args.Argv) < 2 || filepathBase(args.Argv[0]) != "kill" {
		return nil, fmt.Errorf("shell_exec: kill requires at least one exact job-owned PID")
	}
	seen := map[int]bool{}
	pids := make([]int, 0, len(args.Argv)-1)
	for _, raw := range args.Argv[1:] {
		arg := strings.TrimSpace(strings.Trim(raw, `"'`))
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		pid, err := strconv.Atoi(arg)
		if err != nil || pid <= 0 {
			return nil, fmt.Errorf("shell_exec: kill operand %q is not a positive tracked PID; stop only exact PIDs returned by this job's background:true calls", raw)
		}
		if !seen[pid] {
			seen[pid] = true
			pids = append(pids, pid)
		}
	}
	if len(pids) == 0 {
		return nil, fmt.Errorf("shell_exec: kill requires at least one positive PID returned by this job's background:true call")
	}
	return pids, nil
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
	for i, arg := range argv {
		if shellArgvCodeArgument(program, argv, i) {
			continue
		}
		token := strings.Trim(strings.TrimSpace(arg), `"'`)
		if token == "" {
			continue
		}
		if shellArgvControlToken(token) || shellArgvLooksLikeRedirection(token) || shellArgvContainsControlSyntax(token) || strings.Contains(token, "$(") || strings.Contains(token, "`") {
			return shellArgvSyntaxError(arg)
		}
	}
	return nil
}

func shellArgvCodeArgument(program string, argv []string, index int) bool {
	if index <= 0 || index >= len(argv) {
		return false
	}
	prev := strings.Trim(strings.TrimSpace(argv[index-1]), `"'`)
	switch strings.ToLower(strings.TrimSpace(program)) {
	case "node", "nodejs":
		return prev == "-e" || prev == "--eval" || prev == "-p" || prev == "--print"
	case "python", "python3", "ruby", "perl":
		return prev == "-c" || prev == "-e"
	default:
		return false
	}
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

	cmd, err := buildCmd(runCtx, root, args)
	if err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
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

func execBackground(root Root, args shellExecArgs, jobID string) (ToolResult, error) {
	cmd, err := buildCmd(context.Background(), root, args)
	if err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		return ToolResult{}, fmt.Errorf("shell_exec: stdout pipe: %w", err)
	}
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil {
		stdoutRead.Close()
		stdoutWrite.Close()
		return ToolResult{}, fmt.Errorf("shell_exec: stderr pipe: %w", err)
	}
	cmd.Stdout = stdoutWrite
	cmd.Stderr = stderrWrite

	if err := cmd.Start(); err != nil {
		stdoutRead.Close()
		stdoutWrite.Close()
		stderrRead.Close()
		stderrWrite.Close()
		return ToolResult{}, fmt.Errorf("shell_exec: start background: %w", err)
	}
	stdoutWrite.Close()
	stderrWrite.Close()
	pid := cmd.Process.Pid
	proc := &backgroundProcess{jobID: jobID, cmd: cmd, done: make(chan struct{})}

	bgMu.Lock()
	bgProcs[pid] = proc
	bgMu.Unlock()

	// Reap in background and unregister on exit.
	exitCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		close(proc.done)
		bgMu.Lock()
		if bgProcs[pid] == proc && !processGroupExists(pid) {
			delete(bgProcs, pid)
		}
		bgMu.Unlock()
		slog.Debug("shell_exec: background process exited", "job_id", jobID, "pid", pid)
		exitCh <- err
	}()

	// Capture initial output for the capture window so the agent sees
	// startup messages (e.g. "ready on http://localhost:3000").
	var stdoutBuf, stderrBuf lockedBuffer
	stdoutBuf.max = DefaultMaxToolOutputBytes / 2
	stderrBuf.max = DefaultMaxToolOutputBytes / 2
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(&stdoutBuf, stdoutRead)
		_ = stdoutRead.Close()
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(&stderrBuf, stderrRead)
		_ = stderrRead.Close()
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
		bgMu.Lock()
		if bgProcs[pid] == proc {
			delete(bgProcs, pid)
		}
		bgMu.Unlock()
		stopBackgroundProcess(pid, proc)
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
		Output:    fmt.Sprintf("Started in background (PID %d)\n%s\nAfter probes, stop this tracked PID with shell_exec argv [\"kill\",\"%d\"] or rely on job cleanup. Do not call shell_exec with empty argv or : as a wait command.", pid, initial, pid),
		Truncated: truncOut || truncErr,
	}, nil
}

func buildCmd(ctx context.Context, root Root, args shellExecArgs) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if len(args.Argv) > 0 {
		cmd = exec.CommandContext(ctx, args.Argv[0], args.Argv[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", args.ShellCommand)
	}
	cmd.Dir = root.Abs()
	if err := childenv.Apply(cmd); err != nil {
		return nil, err
	}
	return cmd, nil
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
