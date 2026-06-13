/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

func checkEngineerRepeatedNoopPolicy(ctx context.Context, root Root, session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" || !shellExecNoop(args) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[shellNoopFailureKey] == 0 {
		return nil
	}
	tickets, err := ticketstate.ListStatus(root.Abs(), ticketstate.StatusInProgress)
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before repeated shell_exec no-op: %w", err)
	}
	tickets = ordinaryProductTickets(tickets)
	if len(tickets) == 0 {
		return nil
	}
	if !engineerInValidatedPhase(session) {
		return fmt.Errorf(
			"policy: repeated shell_exec no-op before implementation is a loop. Do not call shell_exec again for placeholders or waits. Use file_read on %q and the linked feature contract, then file_write the product implementation or record job_disposition_record with status blocked if the ticket cannot be implemented",
			tickets[0].RelPath,
		)
	}
	files, err := changedFiles(ctx, root)
	if err != nil || len(dispositionBlockingFiles(files)) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: repeated shell_exec no-op after successful validation is a loop. Do not call shell_exec again for placeholders or waits. Run git_status, update evidence for %s if needed, git_commit the dirty implementation/ticket files, move %s to docs/tickets/done/ when acceptance evidence is present, commit that lifecycle move, then record job_disposition_record with next_need qa_review",
		tickets[0].ID,
		tickets[0].ID,
	)
}

func checkForegroundLongRunningShellPolicy(root Root, args shellExecArgs) error {
	if args.Background {
		return nil
	}
	cmd, ok := likelyForegroundLongRunningCommand(root, args)
	if !ok {
		return nil
	}
	return fmt.Errorf("policy: shell_exec command %q is likely a long-running server or watcher; rerun it with background:true, probe readiness with a separate curl or equivalent command, then stop the tracked PID after validation", cmd)
}

func likelyForegroundLongRunningCommand(root Root, args shellExecArgs) (string, bool) {
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return "", false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "npm":
		if len(fields) >= 2 && fields[1] == "start" {
			return "npm start", true
		}
		if len(fields) >= 3 && fields[1] == "run" && serverScriptName(fields[2]) {
			return "npm run " + fields[2], true
		}
	case "pnpm", "yarn", "bun":
		if len(fields) >= 2 && serverScriptName(fields[1]) {
			return cmd + " " + fields[1], true
		}
		if len(fields) >= 3 && fields[1] == "run" && serverScriptName(fields[2]) {
			return cmd + " run " + fields[2], true
		}
	case "python", "python3":
		if len(fields) >= 3 && fields[1] == "-m" && fields[2] == "http.server" {
			return cmd + " -m http.server", true
		}
	case "uvicorn", "gunicorn", "hypercorn", "rails", "vite", "next":
		return cmd, true
	case "go":
		if len(fields) >= 2 && fields[1] == "run" && goRunLikelyStartsServer(root, fields[2:]) {
			return "go run", true
		}
	}
	return "", false
}

func serverScriptName(name string) bool {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "dev", "start", "serve", "server", "preview", "watch":
		return true
	default:
		return false
	}
}

func goRunLikelyStartsServer(root Root, args []string) bool {
	targets := goRunTargets(args)
	if len(targets) == 0 {
		return false
	}
	for _, target := range targets {
		for _, rel := range goRunCandidateFiles(target) {
			if sourceContainsServerMarker(root, rel) {
				return true
			}
		}
	}
	return false
}

func goRunTargets(args []string) []string {
	var targets []string
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		targets = append(targets, arg)
	}
	return targets
}

func goRunCandidateFiles(target string) []string {
	target = strings.TrimPrefix(cleanRepoPath(cleanShellPathToken(target)), "./")
	switch {
	case target == "" || target == ".":
		return []string{"main.go"}
	case strings.HasSuffix(target, ".go"):
		return []string{target}
	default:
		return []string{filepath.ToSlash(filepath.Join(target, "main.go"))}
	}
}

func sourceContainsServerMarker(root Root, rel string) bool {
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	for _, marker := range []string{
		"listenandserve",
		"http.handle",
		"http.newservemux",
		"gin.default",
		"fiber.new",
		"chi.newrouter",
		"echo.new",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func checkShellPolicy(raw json.RawMessage) error {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
	}
	if err := checkShellMarsHarnessCLIPolicy(args); err != nil {
		return err
	}
	cmd := strings.Join(args.Argv, " ")
	if strings.TrimSpace(args.ShellCommand) != "" {
		cmd = args.ShellCommand
	}
	if err := checkShellTicketPathPolicy(cmd); err != nil {
		return err
	}
	if operation, ok := forbiddenShellOperation(cmd); ok {
		return fmt.Errorf("policy: shell_exec command contains forbidden operation %q", operation)
	}
	if operation, ok := dependencyShellOperation(cmd); ok {
		return fmt.Errorf("policy: shell_exec command %q mutates dependency state; use dependency_sync so workspace hygiene preflight and postflight run", operation)
	}
	if operation, ok := broadGeneratedTraversal(cmd); ok {
		return fmt.Errorf("policy: shell_exec command %q may flood context with generated dependency/build output; use file_search, grep, or add explicit generated-directory excludes", operation)
	}
	return nil
}

func checkShellMarsHarnessCLIPolicy(args shellExecArgs) error {
	if cliArgs, ok := shellExecMarsHarnessArgs(args); ok {
		encoded, _ := json.Marshal(cliArgs)
		return fmt.Errorf("policy: run mars-harness commands with mars_harness_cli, not shell_exec, so agents use the active harness executable instead of a stale PATH binary. Retry with mars_harness_cli args %s", encoded)
	}
	return nil
}

func shellExecMarsHarnessArgs(args shellExecArgs) ([]string, bool) {
	if len(args.Argv) > 0 {
		if filepath.Base(strings.TrimSpace(args.Argv[0])) == "mars-harness" {
			return append([]string(nil), args.Argv[1:]...), true
		}
		return nil, false
	}
	fields := strings.Fields(args.ShellCommand)
	for len(fields) > 0 && shellEnvAssignment(fields[0]) {
		fields = fields[1:]
	}
	if len(fields) == 0 || filepath.Base(strings.TrimSpace(fields[0])) != "mars-harness" {
		return nil, false
	}
	return append([]string(nil), fields[1:]...), true
}

func shellEnvAssignment(field string) bool {
	if field == "" || strings.HasPrefix(field, "-") {
		return false
	}
	idx := strings.IndexByte(field, '=')
	if idx <= 0 {
		return false
	}
	name := field[:idx]
	for i, r := range name {
		if r == '_' || ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') || (i > 0 && '0' <= r && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func validateShellExecPolicyArgs(raw json.RawMessage) (shellExecArgs, error) {
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return shellExecArgs{}, fmt.Errorf("shell_exec: parse arguments: %w", err)
	}
	if shellExecNoop(args) {
		return args, nil
	}
	hasArgv := len(args.Argv) > 0
	hasShell := strings.TrimSpace(args.ShellCommand) != ""
	if hasArgv == hasShell {
		return shellExecArgs{}, fmt.Errorf("shell_exec: provide exactly one of argv (non-empty) or shell_command")
	}
	if hasArgv && args.Argv[0] == "" {
		return shellExecArgs{}, fmt.Errorf("shell_exec: argv[0] must be non-empty")
	}
	if hasArgv {
		if err := validateShellExecArgv(args.Argv); err != nil {
			return shellExecArgs{}, err
		}
		if err := validateShellExecGitRemoteMutation(args.Argv, "argv"); err != nil {
			return shellExecArgs{}, err
		}
	} else {
		if err := validateShellExecShellCommand(args.ShellCommand); err != nil {
			return shellExecArgs{}, err
		}
		if err := validateShellExecGitRemoteMutation(strings.Fields(args.ShellCommand), "shell_command"); err != nil {
			return shellExecArgs{}, err
		}
	}
	return args, nil
}

func shellExecGeneratedArtifactCleanup(ctx context.Context, root Root, args shellExecArgs) (bool, error) {
	paths, ok := shellRemovalPathOperands(args)
	if !ok || len(paths) == 0 {
		return false, nil
	}
	for _, rel := range paths {
		if _, ok := validationArtifactPath(root, rel); ok {
			continue
		}
		generated, err := isUntrackedRootBuildArtifact(ctx, root, rel)
		if err != nil {
			return false, err
		}
		if !generated {
			return false, nil
		}
	}
	return true, nil
}

func checkShellBuildOutputPolicy(root Root, args shellExecArgs) error {
	output, implicit, ok := goBuildOutputPath(root, args)
	if !ok || strings.TrimSpace(output) == "" {
		return nil
	}
	if implicit {
		suggestion := validationBinaryOutputSuggestion(output)
		correction := goBuildValidationCorrection(args, suggestion)
		return fmt.Errorf("policy: go build without -o can create the build artifact %q inside the target repo; rerun exactly: %s, or use go test ./... for compile validation so repository diffs stay source-only", output, correction)
	}
	inside, err := pathResolvesInsideRepo(root, output)
	if err != nil {
		return err
	}
	if !inside {
		if _, ok := validationArtifactPath(root, output); ok {
			return nil
		}
		suggestion := validationBinaryOutputSuggestion(output)
		correction := goBuildValidationCorrection(args, suggestion)
		return fmt.Errorf("policy: go build output %q is outside the target repo but is not a tracked validation artifact; rerun exactly: %s so stale temp binaries are blocked and same-session validation can be trusted", output, correction)
	}
	suggestion := validationBinaryOutputSuggestion(output)
	correction := goBuildValidationCorrection(args, suggestion)
	return fmt.Errorf("policy: go build output %q would create a build artifact inside the target repo; rerun exactly: %s, then run or delete that external validation binary and keep repo diffs source-only", output, correction)
}

func goBuildValidationCorrection(args shellExecArgs, suggestion string) string {
	fields := goBuildCommandFields(args)
	if len(fields) < 2 {
		raw, _ := json.Marshal([]string{"go", "build", "-o", suggestion})
		return "shell_exec argv " + string(raw)
	}
	corrected := make([]string, 0, len(fields)+2)
	corrected = append(corrected, cleanShellDisplayToken(fields[0]), cleanShellDisplayToken(fields[1]))
	inserted := false
	for i := 2; i < len(fields); i++ {
		field := strings.TrimSpace(fields[i])
		if field == "" || shellControlToken(field) {
			break
		}
		switch {
		case field == "-o":
			if !inserted {
				corrected = append(corrected, "-o", suggestion)
				inserted = true
			}
			if i+1 < len(fields) {
				i++
			}
		case strings.HasPrefix(field, "-o="):
			if !inserted {
				corrected = append(corrected, "-o", suggestion)
				inserted = true
			}
		default:
			corrected = append(corrected, cleanShellDisplayToken(field))
		}
	}
	if !inserted {
		corrected = append([]string{corrected[0], corrected[1], "-o", suggestion}, corrected[2:]...)
	}
	raw, _ := json.Marshal(corrected)
	return "shell_exec argv " + string(raw)
}

func goBuildCommandFields(args shellExecArgs) []string {
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellCommandFields(args.ShellCommand)
	}
	for i := 0; i < len(fields)-1; i++ {
		if filepathBase(cleanShellPathToken(fields[i])) == "go" && cleanShellPathToken(fields[i+1]) == "build" {
			return fields[i:]
		}
	}
	return nil
}

func validationBinaryOutputSuggestion(output string) string {
	base := filepath.Base(cleanShellPathToken(output))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "app"
	}
	return filepath.ToSlash(filepath.Join("/tmp", base+"-validation"))
}

func cleanShellDisplayToken(field string) string {
	field = strings.TrimSpace(field)
	field = strings.TrimPrefix(field, "1>")
	field = strings.TrimPrefix(field, "2>")
	field = strings.TrimLeft(field, "><")
	return strings.Trim(field, `"'`)
}

func dependencyShellOperation(cmd string) (string, bool) {
	fields := shellFields(cmd)
	if len(fields) == 0 {
		return "", false
	}
	for i, field := range fields {
		switch filepathBase(field) {
		case "npm":
			if nextTokenIs(fields, i, "install") || nextTokenIs(fields, i, "i") || nextTokenIs(fields, i, "ci") {
				return "npm " + fields[i+1], true
			}
		case "pnpm", "yarn", "bun":
			if nextTokenIs(fields, i, "install") || nextTokenIs(fields, i, "i") {
				return filepathBase(field) + " " + fields[i+1], true
			}
		case "go":
			if nextTokenIs(fields, i, "get") {
				return "go get", true
			}
			if i+2 < len(fields) && fields[i+1] == "mod" && fields[i+2] == "download" {
				return "go mod download", true
			}
		case "cargo":
			if nextTokenIs(fields, i, "fetch") {
				return "cargo fetch", true
			}
		case "pip", "pip3":
			if nextTokenIs(fields, i, "install") {
				return filepathBase(field) + " install", true
			}
		case "python", "python3":
			if i+3 < len(fields) && fields[i+1] == "-m" && fields[i+2] == "pip" && fields[i+3] == "install" {
				return filepathBase(field) + " -m pip install", true
			}
		case "bundle":
			if nextTokenIs(fields, i, "install") {
				return "bundle install", true
			}
		case "composer":
			if nextTokenIs(fields, i, "install") {
				return "composer install", true
			}
		}
	}
	return "", false
}

func nextTokenIs(fields []string, index int, token string) bool {
	return index+1 < len(fields) && fields[index+1] == token
}

func broadGeneratedTraversal(cmd string) (string, bool) {
	fields := shellFields(cmd)
	if len(fields) == 0 || hasGeneratedExcludeToken(fields) {
		return "", false
	}
	for i, field := range fields {
		switch filepathBase(field) {
		case "find":
			if i+1 < len(fields) && (fields[i+1] == "." || fields[i+1] == "./") {
				return "find .", true
			}
		case "ls":
			if hasToken(fields[i+1:], "-r") {
				for _, token := range fields[i+1:] {
					if token == "." || token == "./" {
						return "ls recursive/broad root", true
					}
				}
			}
		case "cat":
			for _, token := range fields[i+1:] {
				if token == "." || token == "./" {
					return "cat .", true
				}
			}
		}
	}
	return "", false
}

func hasGeneratedExcludeToken(fields []string) bool {
	for _, field := range fields {
		for _, dir := range generatedWorkspaceDirs {
			if strings.Contains(field, dir) {
				return true
			}
		}
	}
	return false
}

func checkShellTicketPathPolicy(cmd string) error {
	for _, field := range shellFields(cmd) {
		rel := cleanShellPathToken(field)
		if rel == "" {
			continue
		}
		lowerRel := strings.ToLower(cleanRepoPath(rel))
		if lowerRel == "" || lowerRel == "docs/tickets/readme.md" {
			continue
		}
		if !strings.HasPrefix(lowerRel, "docs/tickets/") || !strings.HasSuffix(lowerRel, ".md") {
			continue
		}
		parts := strings.Split(lowerRel, "/")
		if len(parts) < 4 || !isTicketLifecycleDir(parts[2]) {
			return fmt.Errorf("policy: ticket markdown must live under docs/tickets/backlog, docs/tickets/in-progress, docs/tickets/in-review, or docs/tickets/done; use ticket_create for new tickets instead of shell_exec to %s", rel)
		}
	}
	return nil
}

func forbiddenShellOperation(cmd string) (string, bool) {
	fields := shellFields(cmd)
	if len(fields) == 0 {
		return "", false
	}
	if hasGitSubcommand(fields, "push") && hasGitForcePushFlag(fields) {
		return "git push --force", true
	}
	if hasGitSubcommand(fields, "reset") && hasToken(fields, "--hard") {
		return "git reset --hard", true
	}
	if hasGitSubcommand(fields, "clean") && hasGitCleanForceDelete(fields) {
		return "git clean -fd", true
	}
	if hasGitSubcommand(fields, "branch") && hasGitBranchDelete(fields) {
		return "git branch -d", true
	}
	if hasGitSubcommand(fields, "rm") {
		return "git rm", true
	}
	if hasGitSubcommand(fields, "checkout") && hasToken(fields, "-b") {
		return "git checkout -b", true
	}
	if hasRootRemoval(fields) {
		return "rm -rf /", true
	}
	if operation, ok := hasShellRemoval(fields); ok {
		return operation, true
	}
	if hasFindDelete(fields) {
		return "find -delete", true
	}
	return "", false
}

func hasGitSubcommand(fields []string, subcommand string) bool {
	for i := 0; i < len(fields); i++ {
		if fields[i] != "git" && !strings.HasSuffix(fields[i], "/git") {
			continue
		}
		for j := i + 1; j < len(fields); j++ {
			token := fields[j]
			if token == "-c" && j+1 < len(fields) {
				j++
				continue
			}
			if strings.HasPrefix(token, "--git-dir") || strings.HasPrefix(token, "--work-tree") {
				continue
			}
			if token == subcommand {
				return true
			}
			if strings.HasPrefix(token, "-") {
				continue
			}
			break
		}
	}
	return false
}

func hasToken(fields []string, want string) bool {
	for _, field := range fields {
		if field == want {
			return true
		}
	}
	return false
}

func hasGitForcePushFlag(fields []string) bool {
	for _, field := range fields {
		switch {
		case field == "-f", field == "--force", field == "--force-with-lease":
			return true
		case strings.HasPrefix(field, "--force="), strings.HasPrefix(field, "--force-with-lease="):
			return true
		}
	}
	return false
}

func hasGitCleanForceDelete(fields []string) bool {
	force := false
	dirs := false
	for _, field := range fields {
		switch {
		case field == "-f", field == "--force":
			force = true
		case field == "-d":
			dirs = true
		case strings.HasPrefix(field, "-") && !strings.HasPrefix(field, "--"):
			if strings.Contains(field, "f") {
				force = true
			}
			if strings.Contains(field, "d") {
				dirs = true
			}
		}
	}
	return force && dirs
}

func hasGitBranchDelete(fields []string) bool {
	for _, field := range fields {
		switch field {
		case "-d", "--delete", "--force":
			return true
		}
	}
	return false
}

func hasRootRemoval(fields []string) bool {
	for i := 0; i < len(fields); i++ {
		if fields[i] != "rm" {
			continue
		}
		flags := ""
		for j := i + 1; j < len(fields); j++ {
			token := fields[j]
			if token == "--" {
				continue
			}
			if strings.HasPrefix(token, "-") {
				flags += token
				continue
			}
			if token == "/" && strings.Contains(flags, "r") && strings.Contains(flags, "f") {
				return true
			}
		}
	}
	return false
}

func hasShellRemoval(fields []string) (string, bool) {
	for _, field := range fields {
		switch field {
		case "rm", "rmdir", "unlink":
			return field, true
		}
	}
	return "", false
}

func hasFindDelete(fields []string) bool {
	for i, field := range fields {
		if field != "find" && !strings.HasSuffix(field, "/find") {
			continue
		}
		for _, token := range fields[i+1:] {
			if token == "-delete" {
				return true
			}
		}
	}
	return false
}
