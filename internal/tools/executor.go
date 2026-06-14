/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Executor dispatches tool calls against a registry with allowlist enforcement.
type Executor struct {
	Registry   *Registry
	MaxOutput  int // defaults to DefaultMaxToolOutputBytes when zero
	DefaultTTL time.Duration
	Session    *Session

	// StopAfterTool lets callers make specific tool calls terminal for an agent
	// loop. It is checked after a successful tool result has been traced.
	StopAfterTool func() bool
}

// NewExecutor returns an executor backed by reg.
func NewExecutor(reg *Registry) *Executor {
	if reg == nil {
		panic("tools: NewExecutor registry is nil")
	}
	return &Executor{
		Registry:   reg,
		DefaultTTL: 2 * time.Minute,
	}
}

func (e *Executor) maxOut() int {
	if e.MaxOutput > 0 {
		return e.MaxOutput
	}
	return DefaultMaxToolOutputBytes
}

// Execute runs a single tool if it is allowlisted.
func (e *Executor) Execute(ctx context.Context, root Root, allowlist []string, name, argsJSON string) (ToolResult, error) {
	start := time.Now()
	name = strings.TrimSpace(name)
	if name == "" {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: tool name is empty")
	}
	if len(allowlist) == 0 {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: no tools are allowed for this role; configure an explicit tools list in .harness/manifest.yaml")
	}
	if !Allowlisted(name, allowlist) {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: tool %q is not allowed for this role; allowlist: %s", name, strings.Join(allowlist, ", "))
	}
	h, _, ok := e.Registry.Lookup(name)
	if !ok {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: tool %q is not registered", name)
	}
	ttl := e.DefaultTTL
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, ttl)
	defer cancel()

	raw := json.RawMessage(strings.TrimSpace(argsJSON))
	if len(strings.TrimSpace(argsJSON)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return ToolResult{Duration: time.Since(start)}, fmt.Errorf("tools: arguments for %q are not valid JSON", name)
	}

	if e.Session != nil {
		if e.Session.ToolCounts == nil {
			e.Session.ToolCounts = make(map[string]int)
		}
		runCtx = WithSession(runCtx, *e.Session)
	}
	if err := preToolPolicy(runCtx, root, name, raw); err != nil {
		recordPolicyEvent(runCtx, "pre", name, err)
		count := recordSessionToolPolicyFailure(e.Session, name, raw, err)
		err = withPolicyFailureRepairFeedback(root, e.Session, "pre", name, err, count)
		return ToolResult{Duration: time.Since(start)}, err
	}
	res, err := executeHandlerWithTimeout(runCtx, start, ttl, name, root, raw, h)
	recordSessionToolOutcome(e.Session, root, name, raw, res, err)
	if err != nil {
		return res, err
	}
	if res.Output != "" || res.Stderr != "" {
		combined := len(res.Output) + len(res.Stderr)
		if combined > e.maxOut() {
			trunc := e.maxOut() / 2
			out, _ := TruncateUTF8(res.Output, trunc)
			errOut, _ := TruncateUTF8(res.Stderr, trunc)
			res.Output = out
			res.Stderr = errOut
			res.Truncated = true
		}
	}
	return res, nil
}

func recordSessionToolOutcome(session *Session, root Root, name string, raw json.RawMessage, res ToolResult, err error) {
	if session == nil {
		return
	}
	if session.ToolCounts == nil {
		session.ToolCounts = make(map[string]int)
	}
	if session.ToolState == nil {
		session.ToolState = make(map[string]string)
	}
	if err == nil {
		session.ToolCounts["tool:"+name+":success"]++
	} else {
		session.ToolCounts["tool:"+name+":failure"]++
	}
	recordCodeIntelEfficiencyOutcome(session, name, raw, res, err)
	recordTicketCreationOutcome(session, name, raw, err)
	if name == "file_write" && err == nil && strings.ToLower(strings.TrimSpace(session.Role)) == "engineer" {
		recordTestBuildRepairWritePath(session, raw)
	}
	if name == "file_write" && err == nil && session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] > 0 {
		session.ToolCounts[runtimeValidationEditAfterFailureKey]++
	}
	if name == "file_write" && err == nil && session.ToolCounts[testBuildValidationOutstandingKey] > 0 {
		session.ToolCounts[testBuildValidationEditAfterFailureKey]++
	}
	if name == "dependency_sync" && err == nil && strings.ToLower(strings.TrimSpace(session.Role)) == "engineer" && session.ToolCounts[testBuildValidationOutstandingKey] > 0 {
		session.ToolCounts[testBuildValidationEditAfterFailureKey]++
	}
	if name != "shell_exec" {
		return
	}
	if movedID, ok := shellExecInProgressToDoneTicketID(raw); err == nil && res.ExitCode == 0 && ok {
		session.ToolCounts[ticketDoneMoveSuccessKey]++
		session.ToolState[ticketDoneMoveLastIDKey] = movedID
	}
	args, decodeErr := decodeShellExecArgs(raw)
	if decodeErr != nil {
		return
	}
	if shellExecNoop(args) && err != nil {
		session.ToolCounts[shellNoopFailureKey]++
		return
	}
	if err == nil && res.ExitCode == 0 {
		recordSuccessfulValidationArtifactBuild(session, root, args)
	}
	if !shellExecRunsValidationCommandForSession(session, root, args) {
		return
	}
	runtimeValidation := shellExecRunsRuntimeOrArtifactValidationCommandForSession(session, root, args)
	runtimeStderrFailure := err == nil && res.ExitCode == 0 && runtimeValidation && runtimeValidationStderrLooksFailure(res.Stderr)
	session.ToolCounts[validationCommandAttemptKey]++
	if err == nil && res.ExitCode != 0 && validationProcedureFailure(session, root, args, res) {
		session.ToolCounts[validationProcedureFailureKey]++
		session.ToolState[validationProcedureFailureCommandKey] = shellExecCommandForPrompt(args, nil)
		return
	}
	if err == nil && res.ExitCode == 0 && !runtimeStderrFailure {
		session.ToolCounts[validationCommandSuccessKey]++
		if shellExecRunsBrowserProductSmokeCommand(args) {
			session.ToolCounts[browserProductSmokeSuccessKey]++
		}
		if runtimeValidation {
			if recordSuccessfulRuntimeValidationRepair(session, args) && session.ToolCounts[validationCommandFailureKey] > 0 {
				session.ToolCounts[validationCommandFailureKey]--
			}
		}
		if shellExecRunsTestCommand(args) {
			session.ToolCounts[testCommandSuccessKey]++
		}
		if shellExecRunsBuildCommand(args) {
			session.ToolCounts[buildCommandSuccessKey]++
		}
		recordSuccessfulTestBuildValidationRepair(session, args)
		return
	}
	if err == nil && args.ExpectedExitCode != nil && *args.ExpectedExitCode != 0 && runtimeValidation && shellExecLooksLikeMissingArgumentRuntimeProbe(args) {
		recordMissingArgumentCorrectionAttempt(session, args, *args.ExpectedExitCode)
	}
	if err == nil && args.ExpectedExitCode != nil && *args.ExpectedExitCode != 0 && res.ExitCode == *args.ExpectedExitCode && runtimeValidation {
		session.ToolCounts[expectedRuntimeFailureSuccessKey]++
		if recordExpectedRuntimeValidationCorrection(session, args, res.ExitCode) && session.ToolCounts[validationCommandFailureKey] > 0 {
			session.ToolCounts[validationCommandFailureKey]--
		}
		return
	}
	if err == nil && args.ExpectedExitCode == nil && runtimeValidation && runtimeValidationLooksExpectedInputValidationFailure(args, res) {
		session.ToolCounts[expectedRuntimeFailureSuccessKey]++
		return
	}
	session.ToolCounts[validationCommandFailureKey]++
	if err == nil && (res.ExitCode != 0 || runtimeStderrFailure) && runtimeValidation {
		exitCode := res.ExitCode
		session.ToolCounts[unexpectedRuntimeValidationFailureKey(args, exitCode)]++
		session.ToolCounts[unexpectedRuntimeValidationFailureFingerprintKey(args)]++
		session.ToolCounts[runtimeValidationFailureEditWatermarkKey(args)] = session.ToolCounts[runtimeValidationEditAfterFailureKey]
		session.ToolCounts[unexpectedRuntimeValidationOutstandingKey]++
		recordUnexpectedRuntimeValidationState(session, args)
	}
	if shellExecRunsTestCommand(args) {
		session.ToolCounts[testCommandFailureKey]++
	}
	if shellExecRunsBuildCommand(args) {
		session.ToolCounts[buildCommandFailureKey]++
	}
	recordFailedTestBuildValidation(session, args, res)
}

const (
	codeIntelToolCallsKey      = "codeintel:tool_calls"
	codeIntelToolSuccessKey    = "codeintel:tool_success"
	codeIntelOutputBytesKey    = "codeintel:output_bytes"
	broadRepoToolCallsKey      = "repo_exploration:broad_tool_calls"
	broadRepoOutputBytesKey    = "repo_exploration:broad_output_bytes"
	bulkFileReadCallsKey       = "repo_exploration:bulk_file_read_calls"
	bulkFileReadOutputBytesKey = "repo_exploration:bulk_file_read_output_bytes"
	broadShellSearchCallsKey   = "repo_exploration:broad_shell_search_calls"
)

func recordCodeIntelEfficiencyOutcome(session *Session, name string, raw json.RawMessage, res ToolResult, err error) {
	outputBytes := len(res.Output) + len(res.Stderr)
	if isCodeIntelTool(name) {
		session.ToolCounts[codeIntelToolCallsKey]++
		session.ToolCounts[codeIntelOutputBytesKey] += outputBytes
		if err == nil {
			session.ToolCounts[codeIntelToolSuccessKey]++
		}
		return
	}
	if !isBroadRepoExplorationTool(name, raw, res) {
		return
	}
	session.ToolCounts[broadRepoToolCallsKey]++
	session.ToolCounts[broadRepoOutputBytesKey] += outputBytes
	if name == "file_read" {
		session.ToolCounts[bulkFileReadCallsKey]++
		session.ToolCounts[bulkFileReadOutputBytesKey] += outputBytes
	}
	if name == "shell_exec" {
		session.ToolCounts[broadShellSearchCallsKey]++
	}
}

func isCodeIntelTool(name string) bool {
	switch strings.TrimSpace(name) {
	case "code_index", "code_search", "code_snippet", "code_trace", "code_impact":
		return true
	default:
		return false
	}
}

func isBroadRepoExplorationTool(name string, raw json.RawMessage, res ToolResult) bool {
	switch strings.TrimSpace(name) {
	case "grep", "file_search":
		return true
	case "file_read":
		return fileReadLooksBulk(raw, res)
	case "shell_exec":
		return shellExecLooksBroadRepoSearch(raw)
	default:
		return false
	}
}

func fileReadLooksBulk(raw json.RawMessage, res ToolResult) bool {
	var args fileReadArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return false
	}
	if args.StartLine != nil || args.EndLine != nil {
		return false
	}
	return strings.TrimSpace(args.Path) != "" && len(res.Output)+len(res.Stderr) > 0
}

func shellExecLooksBroadRepoSearch(raw json.RawMessage) bool {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return false
	}
	if len(args.Argv) > 0 {
		switch strings.TrimSpace(filepathBase(args.Argv[0])) {
		case "rg", "grep", "find":
			return true
		}
	}
	cmd := strings.TrimSpace(args.ShellCommand)
	return strings.HasPrefix(cmd, "rg ") || strings.HasPrefix(cmd, "grep ") || strings.HasPrefix(cmd, "find ")
}

func recordTestBuildRepairWritePath(session *Session, raw json.RawMessage) {
	if session == nil {
		return
	}
	if session.ToolState == nil {
		session.ToolState = make(map[string]string)
	}
	args, err := decodeFileWriteArgs(raw)
	if err != nil {
		return
	}
	rel := cleanRepoPath(args.Path)
	if rel == "" {
		return
	}
	session.ToolState[testBuildRepairWritePathKey(rel)] = "true"
}

func recordSessionToolPolicyFailure(session *Session, name string, raw json.RawMessage, err error) int {
	if session == nil || err == nil {
		return 0
	}
	if session.ToolCounts == nil {
		session.ToolCounts = make(map[string]int)
	}
	if session.ToolState == nil {
		session.ToolState = make(map[string]string)
	}
	session.ToolCounts["tool:"+name+":failure"]++
	count := recordRepeatedPolicyFailure(session, "pre", name, err)
	recordTicketCreationOutcome(session, name, raw, err)
	if name == "shell_exec" {
		args, decodeErr := decodeShellExecArgs(raw)
		if decodeErr == nil && shellExecNoop(args) {
			session.ToolCounts[shellNoopFailureKey]++
		}
	}
	return count
}

func recordTicketCreationOutcome(session *Session, name string, raw json.RawMessage, err error) {
	if session == nil {
		return
	}
	if session.ToolCounts == nil {
		session.ToolCounts = make(map[string]int)
	}
	switch name {
	case "ticket_create":
		if err == nil {
			session.ToolCounts[ticketCreationOutstandingFailureKey] = 0
			return
		}
		session.ToolCounts[ticketCreationOutstandingFailureKey]++
	case "file_write":
		if err != nil && fileWriteTargetsTicketCreationPath(session.Role, raw) {
			session.ToolCounts[ticketCreationOutstandingFailureKey]++
		}
	}
}

func fileWriteTargetsTicketCreationPath(role string, raw json.RawMessage) bool {
	if strings.ToLower(strings.TrimSpace(role)) == "engineer" {
		return false
	}
	return fileWriteTargetsTicketPath(raw)
}

func fileWriteTargetsTicketPath(raw json.RawMessage) bool {
	args, err := decodeFileWriteArgs(raw)
	if err != nil {
		return false
	}
	rel := cleanRepoPath(args.Path)
	return rel == "docs/tickets" || strings.HasPrefix(rel, "docs/tickets/")
}

func recordUnexpectedRuntimeValidationState(session *Session, args shellExecArgs) {
	if session == nil {
		return
	}
	if session.ToolState == nil {
		session.ToolState = make(map[string]string)
	}
	session.ToolState[unexpectedRuntimeValidationCommandKey] = shellExecCommandForPrompt(args, nil)
	if shellExecLooksLikeMissingArgumentRuntimeProbe(args) {
		exitCode := 1
		correction := shellExecCommandForPrompt(args, &exitCode)
		if session.ToolState[unexpectedRuntimeValidationCorrectionKey] != correction {
			delete(session.ToolState, unexpectedRuntimeValidationAttemptedKey)
		}
		session.ToolState[unexpectedRuntimeValidationMissingArgKey] = "true"
		session.ToolState[unexpectedRuntimeValidationCorrectionKey] = correction
		return
	}
	delete(session.ToolState, unexpectedRuntimeValidationMissingArgKey)
	delete(session.ToolState, unexpectedRuntimeValidationCorrectionKey)
	delete(session.ToolState, unexpectedRuntimeValidationAttemptedKey)
}

func recordFailedTestBuildValidation(session *Session, args shellExecArgs, res ToolResult) {
	if session == nil || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return
	}
	if !shellExecRunsTestCommand(args) && !shellExecRunsBuildCommand(args) {
		return
	}
	if session.ToolCounts == nil {
		session.ToolCounts = make(map[string]int)
	}
	if session.ToolState == nil {
		session.ToolState = make(map[string]string)
	}
	session.ToolCounts[testBuildValidationFailureFingerprintKey(args)]++
	session.ToolCounts[testBuildValidationFailureEditWatermarkKey(args)] = session.ToolCounts[testBuildValidationEditAfterFailureKey]
	session.ToolCounts[testBuildValidationLastFailureEditKey] = session.ToolCounts[testBuildValidationEditAfterFailureKey]
	session.ToolCounts[testBuildValidationOutstandingKey]++
	session.ToolState[testBuildValidationCommandKey] = shellExecCommandForPrompt(args, nil)
	if output := summarizeTestBuildFailureOutput(res); output != "" {
		session.ToolState[testBuildValidationOutputKey] = output
	} else {
		delete(session.ToolState, testBuildValidationOutputKey)
	}
	if scopes := testBuildValidationRepairScopes(args); len(scopes) > 0 {
		session.ToolState[testBuildValidationScopeKey] = strings.Join(scopes, "\n")
	} else {
		delete(session.ToolState, testBuildValidationScopeKey)
	}
}

func summarizeTestBuildFailureOutput(res ToolResult) string {
	var parts []string
	if strings.TrimSpace(res.Output) != "" {
		parts = append(parts, strings.TrimSpace(res.Output))
	}
	if strings.TrimSpace(res.Stderr) != "" {
		parts = append(parts, strings.TrimSpace(res.Stderr))
	}
	if len(parts) == 0 && res.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("exit code %d", res.ExitCode))
	}
	out := strings.Join(parts, "\n")
	out = strings.Join(strings.Fields(out), " ")
	out, _ = TruncateUTF8(out, 900)
	return out
}

func validationProcedureFailure(session *Session, root Root, args shellExecArgs, res ToolResult) bool {
	if session == nil || !roleAllowsValidationProcedureFailure(session.Role) {
		return false
	}
	if shellExecNodeCheckHTML(args) {
		return true
	}
	if nodeCheckMissingFileProcedureFailure(root, args, res) {
		return true
	}
	if nodeEvalBrowserFrameworkProcedureFailure(root, args, res) {
		return true
	}
	if shellExecRunsHTTPProbe(args) && httpProbeFailedBeforeServerStart(res) {
		return true
	}
	if !shellExecRunsTestCommand(args) && !shellExecRunsBuildCommand(args) {
		return false
	}
	if goCommandMissingRelativePackagePrefix(args, res) {
		return true
	}
	if goCommandTargetsRootWithoutGoFiles(root, args, res) {
		return true
	}
	return false
}

func nodeEvalBrowserFrameworkProcedureFailure(root Root, args shellExecArgs, res ToolResult) bool {
	code, ok := shellExecNodeEvalCode(args)
	if !ok || strings.TrimSpace(code) == "" {
		return false
	}
	output := strings.ToLower(strings.TrimSpace(res.Stderr + "\n" + res.Output))
	if !browserGlobalMissingInNode(output) {
		return false
	}
	if !nodeEvalLoadsProjectOrBrowserFramework(code) && !browserFrameworkPackageInStack(output) {
		return false
	}
	info := repoBrowserFrameworkInfo(root)
	return info.UsesFramework || browserFrameworkPackageInStack(output)
}

func shellExecNodeEvalCode(args shellExecArgs) (string, bool) {
	fields := normalizedShellExecFields(args)
	for i := 0; i < len(fields); i++ {
		if filepathBase(fields[i]) != "node" {
			continue
		}
		for j := i + 1; j < len(fields); j++ {
			flag := fields[j]
			if flag == "--" {
				break
			}
			if flag == "-e" || flag == "--eval" {
				if j+1 < len(fields) {
					return fields[j+1], true
				}
				return "", false
			}
			if strings.HasPrefix(flag, "--eval=") {
				return strings.TrimPrefix(flag, "--eval="), true
			}
		}
	}
	return "", false
}

func browserGlobalMissingInNode(output string) bool {
	for _, marker := range []string{
		"referenceerror: window is not defined",
		"referenceerror: document is not defined",
		"referenceerror: navigator is not defined",
		"referenceerror: self is not defined",
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func nodeEvalLoadsProjectOrBrowserFramework(code string) bool {
	lower := strings.ToLower(code)
	for _, marker := range []string{
		"require('./",
		`require("./`,
		"require('../",
		`require("../`,
		"import('./",
		`import("./`,
		"import('../",
		`import("../`,
		" from './",
		` from "./`,
		" from '../",
		` from "../`,
		"require('phaser')",
		`require("phaser")`,
		"import('phaser')",
		`import("phaser")`,
		" from 'phaser'",
		` from "phaser"`,
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func browserFrameworkPackageInStack(output string) bool {
	for _, marker := range []string{
		"node_modules/phaser",
		`node_modules\phaser`,
		"phaser/src/",
		`phaser\src\`,
		"node_modules/pixi.js",
		`node_modules\pixi.js`,
		"node_modules/@pixi",
		`node_modules\@pixi`,
		"node_modules/konva",
		`node_modules\konva`,
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func nodeCheckMissingFileProcedureFailure(root Root, args shellExecArgs, res ToolResult) bool {
	target, ok := shellExecNodeCheckTarget(args)
	if !ok || target == "" {
		return false
	}
	abs, err := root.ResolvePath(target)
	if err != nil {
		return false
	}
	if _, err := os.Stat(abs); err == nil {
		return false
	}
	output := strings.ToLower(strings.TrimSpace(res.Stderr + "\n" + res.Output))
	for _, marker := range []string{
		"cannot find module",
		"module_not_found",
		"no such file",
		"enoent",
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func shellExecNodeCheckTarget(args shellExecArgs) (string, bool) {
	fields := normalizedShellExecFields(args)
	for i := 0; i < len(fields)-2; i++ {
		if filepathBase(fields[i]) != "node" {
			continue
		}
		flag := fields[i+1]
		if flag != "--check" && flag != "-c" {
			continue
		}
		return cleanShellPathToken(fields[i+2]), true
	}
	return "", false
}

func httpProbeFailedBeforeServerStart(res ToolResult) bool {
	if res.ExitCode == 0 {
		return false
	}
	output := strings.ToLower(strings.TrimSpace(res.Stderr + "\n" + res.Output))
	if output == "" {
		return false
	}
	for _, marker := range []string{
		"curl: (7)",
		"failed to connect",
		"could not connect",
		"couldn't connect",
		"connection refused",
		"connection failure",
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func roleAllowsValidationProcedureFailure(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "engineer":
		return true
	default:
		return reviewRoleRequiresValidationEvidence(role)
	}
}

func goCommandMissingRelativePackagePrefix(args shellExecArgs, res ToolResult) bool {
	output := strings.ToLower(res.Stderr + "\n" + res.Output)
	if !strings.Contains(output, " is not in std ") {
		return false
	}
	fields := goTestOrBuildCommandFields(args)
	for _, field := range fields[2:] {
		token := cleanShellPathToken(field)
		if token == "" || shellControlToken(token) {
			break
		}
		if strings.HasPrefix(token, "-") {
			continue
		}
		if strings.HasPrefix(token, "cmd/") || strings.HasPrefix(token, "internal/") || strings.HasPrefix(token, "pkg/") {
			return true
		}
	}
	return false
}

func goCommandTargetsRootWithoutGoFiles(root Root, args shellExecArgs, res ToolResult) bool {
	output := strings.ToLower(res.Stderr + "\n" + res.Output)
	if !strings.Contains(output, "no go files in") {
		return false
	}
	if _, ok := firstCmdMain(root); !ok {
		return false
	}
	fields := goTestOrBuildCommandFields(args)
	if len(fields) < 2 {
		return false
	}
	hasTarget := false
	for i := 2; i < len(fields); i++ {
		token := cleanShellPathToken(fields[i])
		if token == "" || shellControlToken(token) {
			break
		}
		if token == "-o" {
			i++
			continue
		}
		if strings.HasPrefix(token, "-o=") || strings.HasPrefix(token, "-") {
			continue
		}
		hasTarget = true
		if token == "." {
			return true
		}
	}
	return !hasTarget
}

func goTestOrBuildCommandFields(args shellExecArgs) []string {
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellCommandFields(args.ShellCommand)
	}
	for i := 0; i < len(fields)-1; i++ {
		if filepathBase(cleanShellPathToken(fields[i])) != "go" {
			continue
		}
		action := cleanShellPathToken(fields[i+1])
		if action == "test" || action == "build" {
			return fields[i:]
		}
	}
	return nil
}

func recordSuccessfulTestBuildValidationRepair(session *Session, args shellExecArgs) bool {
	if session == nil || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return false
	}
	if !shellExecRunsTestCommand(args) && !shellExecRunsBuildCommand(args) {
		return false
	}
	if session.ToolCounts == nil || session.ToolCounts[testBuildValidationOutstandingKey] <= 0 {
		return false
	}
	repaired := 0
	if shellExecRunsTestCommand(args) {
		repaired += session.ToolCounts[testCommandFailureKey]
		session.ToolCounts[testCommandFailureKey] = 0
	}
	if shellExecRunsBuildCommand(args) {
		repaired += session.ToolCounts[buildCommandFailureKey]
		session.ToolCounts[buildCommandFailureKey] = 0
	}
	if repaired <= 0 {
		repaired = 1
	}
	session.ToolCounts[testBuildValidationRepairKey(args)] += repaired
	decrementOutstandingTestBuildFailures(session, repaired)
	if session.ToolCounts[validationCommandFailureKey] <= repaired {
		session.ToolCounts[validationCommandFailureKey] = 0
	} else {
		session.ToolCounts[validationCommandFailureKey] -= repaired
	}
	if session.ToolCounts[testBuildValidationOutstandingKey] == 0 && session.ToolState != nil {
		delete(session.ToolState, testBuildValidationCommandKey)
		delete(session.ToolState, testBuildValidationOutputKey)
		delete(session.ToolState, testBuildValidationScopeKey)
	}
	return true
}

func decrementOutstandingTestBuildFailures(session *Session, n int) {
	if session == nil || session.ToolCounts == nil || n <= 0 {
		return
	}
	outstanding := session.ToolCounts[testBuildValidationOutstandingKey]
	if outstanding <= n {
		session.ToolCounts[testBuildValidationOutstandingKey] = 0
		return
	}
	session.ToolCounts[testBuildValidationOutstandingKey] = outstanding - n
}

func clearUnexpectedRuntimeValidationState(session *Session) {
	if session == nil || session.ToolState == nil {
		return
	}
	delete(session.ToolState, unexpectedRuntimeValidationCommandKey)
	delete(session.ToolState, unexpectedRuntimeValidationCorrectionKey)
	delete(session.ToolState, unexpectedRuntimeValidationMissingArgKey)
	delete(session.ToolState, unexpectedRuntimeValidationAttemptedKey)
}

func shellExecCommandForPrompt(args shellExecArgs, expectedExitCode *int) string {
	payload := map[string]any{}
	if len(args.Argv) > 0 {
		payload["argv"] = args.Argv
	} else {
		payload["shell_command"] = args.ShellCommand
	}
	if expectedExitCode != nil {
		payload["expected_exit_code"] = *expectedExitCode
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "shell_exec with the exact failing command"
	}
	return "shell_exec " + string(encoded)
}

func runtimeValidationStderrLooksFailure(stderr string) bool {
	s := strings.ToLower(strings.TrimSpace(stderr))
	if s == "" {
		return false
	}
	for _, marker := range []string{"error:", "usage of ", "panic:", "traceback", "exception"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

func runtimeValidationLooksExpectedInputValidationFailure(args shellExecArgs, res ToolResult) bool {
	if res.ExitCode == 0 {
		return false
	}
	output := strings.ToLower(strings.TrimSpace(res.Stderr + "\n" + res.Output))
	if output == "" {
		return false
	}
	if runtimeValidationOutputHasCrashMarker(output) {
		return false
	}
	if shellExecLooksLikeInputValidationRuntimeProbe(args) && runtimeValidationOutputHasInputValidationMarker(output) {
		return true
	}
	return shellExecLooksLikeSurplusArgumentRuntimeProbe(args) && runtimeValidationOutputHasSurplusArgumentMarker(output)
}

func runtimeValidationOutputHasCrashMarker(output string) bool {
	for _, marker := range []string{"panic:", "traceback", "exception", "runtime error", "segmentation fault"} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func runtimeValidationOutputHasInputValidationMarker(output string) bool {
	for _, marker := range []string{
		"required",
		"missing",
		"usage:",
		"usage of ",
		"requires",
		"must provide",
		"must be",
		"expected",
		"invalid",
		"invalid input",
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func runtimeValidationOutputHasSurplusArgumentMarker(output string) bool {
	for _, marker := range []string{
		"too many argument",
		"too many args",
		"too many values",
		"at most one",
		"only one",
		"single argument",
		"exactly one argument",
	} {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func shellExecLooksLikeInputValidationRuntimeProbe(args shellExecArgs) bool {
	if shellExecLooksLikeMissingArgumentRuntimeProbe(args) {
		return true
	}
	for _, field := range shellExecRuntimeProductArgs(args) {
		token := strings.ToLower(strings.Trim(strings.TrimSpace(field), `"'`))
		switch token {
		case "invalid", "bad", "abc", "not-a-number", "not_a_number":
			return true
		}
		if strings.Contains(token, "invalid") || strings.Contains(token, "not-a-number") || strings.Contains(token, "not_a_number") {
			return true
		}
	}
	return false
}

func shellExecLooksLikeSurplusArgumentRuntimeProbe(args shellExecArgs) bool {
	return len(shellExecRuntimeProductArgs(args)) > 1
}

func shellExecRuntimeProductArgs(args shellExecArgs) []string {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return nil
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "go", "cargo", "dotnet":
		if len(fields) >= 3 && fields[1] == "run" {
			return fields[3:]
		}
	case "python", "python3", "node", "deno", "ruby":
		if len(fields) >= 2 {
			return fields[2:]
		}
	case "java":
		if len(fields) >= 3 && fields[1] == "-jar" {
			return fields[3:]
		}
	case "npm", "pnpm", "yarn", "bun":
		if len(fields) >= 3 && fields[1] == "run" {
			return fields[3:]
		}
		if len(fields) >= 2 {
			return fields[2:]
		}
	default:
		return fields[1:]
	}
	return nil
}

func shellExecRunsRuntimeOrArtifactValidationCommandForSession(session *Session, root Root, args shellExecArgs) bool {
	if shellExecRunsTestCommand(args) || shellExecRunsBuildCommand(args) {
		return false
	}
	if shellExecRunsRuntimeValidationCommand(args) {
		return true
	}
	return shellExecRunsRecordedValidationArtifact(*session, root, args)
}

func recordExpectedRuntimeValidationCorrection(session *Session, args shellExecArgs, exitCode int) bool {
	if !roleCanCorrectUnexpectedRuntimeValidation(session.Role, args) {
		return false
	}
	failureKey := unexpectedRuntimeValidationFailureKey(args, exitCode)
	correctionKey := expectedRuntimeValidationCorrectionKey(args, exitCode)
	unrepaired := session.ToolCounts[failureKey] - session.ToolCounts[correctionKey]
	if unrepaired <= 0 {
		return false
	}
	session.ToolCounts[correctionKey] += unrepaired
	session.ToolCounts[runtimeValidationRepairKey(args)] += unrepaired
	decrementOutstandingRuntimeFailures(session, unrepaired)
	if session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] == 0 {
		clearUnexpectedRuntimeValidationState(session)
	}
	return true
}

func recordMissingArgumentCorrectionAttempt(session *Session, args shellExecArgs, exitCode int) {
	if session == nil || session.ToolState == nil || exitCode == 0 {
		return
	}
	session.ToolState[unexpectedRuntimeValidationAttemptedKey] = shellExecCommandForPrompt(args, &exitCode)
}

func roleCanCorrectUnexpectedRuntimeValidation(role string, args shellExecArgs) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "qa", "security":
		return true
	case "engineer":
		return shellExecLooksLikeMissingArgumentRuntimeProbe(args)
	default:
		return false
	}
}

func recordSuccessfulRuntimeValidationRepair(session *Session, args shellExecArgs) bool {
	failureKey := unexpectedRuntimeValidationFailureFingerprintKey(args)
	repairKey := runtimeValidationRepairKey(args)
	unrepaired := session.ToolCounts[failureKey] - session.ToolCounts[repairKey]
	if unrepaired <= 0 {
		return false
	}
	session.ToolCounts[repairKey] += unrepaired
	decrementOutstandingRuntimeFailures(session, unrepaired)
	if session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] == 0 {
		clearUnexpectedRuntimeValidationState(session)
	}
	return true
}

func decrementOutstandingRuntimeFailures(session *Session, n int) {
	if session == nil || session.ToolCounts == nil || n <= 0 {
		return
	}
	outstanding := session.ToolCounts[unexpectedRuntimeValidationOutstandingKey]
	if outstanding <= n {
		session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] = 0
		return
	}
	session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] = outstanding - n
}

type handlerResult struct {
	result ToolResult
	err    error
}

func executeHandlerWithTimeout(ctx context.Context, start time.Time, ttl time.Duration, name string, root Root, raw json.RawMessage, h Handler) (ToolResult, error) {
	done := make(chan handlerResult, 1)
	slog.Debug("tools: executing tool", "tool", name, "ttl", ttl)
	go func() {
		res, err := h(ctx, root, raw)
		res.Duration = time.Since(start)
		if err == nil {
			if postErr := postToolPolicy(ctx, root, name, raw); postErr != nil {
				recordPolicyEvent(ctx, "post", name, postErr)
				err = postErr
			}
		}
		done <- handlerResult{result: res, err: err}
	}()

	select {
	case out := <-done:
		slog.Debug("tools: tool finished", "tool", name, "duration", out.result.Duration, "err", out.err != nil)
		return out.result, out.err
	case <-ctx.Done():
		duration := time.Since(start)
		slog.Warn("tools: tool timed out", "tool", name, "duration", duration, "ttl", ttl, "err", ctx.Err())
		return ToolResult{
			Duration: duration,
			ExitCode: -1,
		}, fmt.Errorf("tools: tool %q timed out after %s; the harness stopped waiting so the agent can record a blocker instead of hanging", name, ttl.Round(time.Second))
	}
}

func recordPolicyEvent(ctx context.Context, stage, toolName string, err error) {
	if err == nil {
		return
	}
	session, ok := SessionFromContext(ctx)
	if !ok || session.PolicyRecorder == nil {
		return
	}
	session.PolicyRecorder(PolicyEvent{
		Stage:    stage,
		ToolName: toolName,
		Message:  err.Error(),
	})
}
