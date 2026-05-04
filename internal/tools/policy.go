package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars-harness/internal/safety"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
)

var mutatingTools = map[string]bool{
	"file_write":          true,
	"shell_exec":          true,
	"mars_harness_cli":    true,
	"git_commit":          true,
	"git_push":            true,
	"record_decision":     true,
	"ticket_create":       true,
	"tool_create":         true,
	"release_orchestrate": true,
}

const (
	dogfoodTicketCreateLimitTotal       = 5
	dogfoodTicketCreateLimitPerSeverity = 3
	dogfoodTicketCreateLimitPerGroup    = 2
)

func preToolPolicy(ctx context.Context, root Root, name string, raw json.RawMessage) error {
	session, hasSession := SessionFromContext(ctx)
	if hasSession {
		if err := enforceTrust(session, name); err != nil {
			return err
		}
	}

	switch name {
	case "file_write":
		return checkFileWritePolicy(root, session, hasSession, raw)
	case "ticket_create":
		return checkTicketCreatePolicy(root, session, hasSession, raw)
	case "git_commit":
		return validateRepoDiff(ctx, root, session)
	case "git_push":
		return checkGitPushPolicy(ctx, root, raw)
	case "shell_exec":
		if err := checkShellPolicy(raw); err != nil {
			return err
		}
		if !shellExecReadOnly(raw) {
			if err := validateRepoDiff(ctx, root, session); err != nil {
				return fmt.Errorf("policy: shell_exec command may mutate while repository is already outside blast-radius limits: %w", err)
			}
		}
		return nil
	default:
		return nil
	}
}

func postToolPolicy(ctx context.Context, root Root, name string, raw json.RawMessage) error {
	if !mutatingTools[name] {
		return nil
	}
	session, _ := SessionFromContext(ctx)
	switch name {
	case "git_commit", "git_push":
		return nil
	case "shell_exec":
		if shellExecReadOnly(raw) {
			return nil
		}
		return validateRepoDiff(ctx, root, session)
	default:
		return validateRepoDiff(ctx, root, session)
	}
}

func enforceTrust(session Session, name string) error {
	level := strings.TrimSpace(strings.ToLower(session.TrustLevel))
	if level == "" {
		return nil
	}
	if level == "observer" && mutatingTools[name] {
		return fmt.Errorf("policy: trust level observer cannot run mutating tool %q", name)
	}
	return nil
}

func checkTicketCreatePolicy(root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession {
		return nil
	}
	var args ticketCreateArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	switch role {
	case "engineer":
		return checkEngineerTicketCreatePolicy(root, args)
	case "dogfood":
		return checkDogfoodTicketCreatePolicy(session, args)
	default:
		return nil
	}
}

func checkEngineerTicketCreatePolicy(root Root, args ticketCreateArgs) error {
	eligible, err := ticketstate.EligibleInProgress(root.Abs())
	if err != nil {
		return fmt.Errorf("policy: inspect in-progress tickets before ticket_create: %w", err)
	}
	if len(eligible) == 0 {
		return nil
	}
	if isInterventionDebtTicket(args) || isDependencyTicketForEligibleWork(args, eligible) {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer cannot create ordinary backlog tickets while eligible in-progress tickets remain: %s. Complete the ticket or create a linked dependency/intervention-debt ticket with metadata.blocks.",
		joinTicketNames(eligible),
	)
}

func isInterventionDebtTicket(args ticketCreateArgs) bool {
	return strings.TrimSpace(args.Kind) == "intervention-debt" || strings.TrimSpace(args.WorkType) == "intervention-debt"
}

func isDependencyTicketForEligibleWork(args ticketCreateArgs, eligible []ticketstate.Ticket) bool {
	if strings.TrimSpace(args.DedupeKey) == "" {
		return false
	}
	values := []string{
		args.Metadata["blocks"],
		args.Metadata["blocked_ticket"],
		args.Metadata["blocked_by_target"],
	}
	for _, value := range values {
		for _, t := range eligible {
			if ticketMetadataMentions(value, t) {
				return true
			}
		}
	}
	return false
}

func ticketMetadataMentions(value string, ticket ticketstate.Ticket) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, strings.ToLower(ticket.ID)) ||
		strings.Contains(value, strings.ToLower(ticket.Name))
}

func checkDogfoodTicketCreatePolicy(session Session, args ticketCreateArgs) error {
	if session.ToolCounts == nil {
		return nil
	}
	severity := strings.ToLower(strings.TrimSpace(args.Priority))
	if severity == "" {
		severity = "medium"
	}
	group := dogfoodTicketGroup(args)
	dedupe := strings.TrimSpace(args.DedupeKey)
	totalKey := "ticket_create:dogfood:total"
	severityKey := "ticket_create:dogfood:severity:" + severity
	groupKey := "ticket_create:dogfood:group:" + group
	dedupeKey := "ticket_create:dogfood:dedupe:" + dedupe
	if dedupe != "" && session.ToolCounts[dedupeKey] > 0 {
		return fmt.Errorf("policy: dogfood ticket_create repeated dedupe key %q in one run; update the existing ticket instead", dedupe)
	}
	if session.ToolCounts[totalKey] >= dogfoodTicketCreateLimitTotal {
		return fmt.Errorf("policy: dogfood ticket_create capped at %d tickets per run; group remaining findings behind the highest-severity dedupe keys", dogfoodTicketCreateLimitTotal)
	}
	if session.ToolCounts[severityKey] >= dogfoodTicketCreateLimitPerSeverity {
		return fmt.Errorf("policy: dogfood ticket_create capped at %d %s-severity tickets per run", dogfoodTicketCreateLimitPerSeverity, severity)
	}
	if session.ToolCounts[groupKey] >= dogfoodTicketCreateLimitPerGroup {
		return fmt.Errorf("policy: dogfood ticket_create capped at %d tickets for group %q per run", dogfoodTicketCreateLimitPerGroup, group)
	}
	session.ToolCounts[totalKey]++
	session.ToolCounts[severityKey]++
	session.ToolCounts[groupKey]++
	if dedupe != "" {
		session.ToolCounts[dedupeKey]++
	}
	return nil
}

func dogfoodTicketGroup(args ticketCreateArgs) string {
	if dedupe := strings.TrimSpace(args.DedupeKey); dedupe != "" {
		parts := strings.Split(dedupe, ":")
		if len(parts) >= 5 {
			return strings.Join(parts[:5], ":")
		}
		return dedupe
	}
	if args.Metadata != nil {
		if category := strings.TrimSpace(args.Metadata["category"]); category != "" {
			return category
		}
	}
	if title := strings.TrimSpace(args.Title); title != "" {
		return slugify(title)
	}
	return "unknown"
}

func joinTicketNames(tickets []ticketstate.Ticket) string {
	names := make([]string, 0, len(tickets))
	for _, t := range tickets {
		names = append(names, t.Name)
	}
	return strings.Join(names, ", ")
}

func checkFileWritePolicy(root Root, session Session, hasSession bool, raw json.RawMessage) error {
	var args fileWriteArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	if err := checkTicketFileWritePolicy(root, args.Path); err != nil {
		return err
	}
	if !hasSession {
		return nil
	}
	if session.Guardrails != nil {
		if err := session.Guardrails.CheckFile(session.Role, args.Path, args.Content); err != nil {
			return err
		}
	}
	if hits := safety.ScanForSecrets(args.Path, args.Content); len(hits) > 0 {
		return fmt.Errorf("policy: secret scanner blocked %s:%d (%s)", hits[0].File, hits[0].Line, hits[0].Pattern)
	}
	return nil
}

func checkTicketFileWritePolicy(root Root, rel string) error {
	rel = cleanRepoPath(rel)
	lowerRel := strings.ToLower(rel)
	if rel == "" || lowerRel == "docs/tickets/readme.md" {
		return nil
	}
	if !strings.HasPrefix(lowerRel, "docs/tickets/") || !strings.HasSuffix(lowerRel, ".md") {
		return nil
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 4 || !isTicketLifecycleDir(parts[2]) {
		return fmt.Errorf("policy: ticket markdown must live under docs/tickets/backlog, docs/tickets/in-progress, docs/tickets/in-review, or docs/tickets/done; use ticket_create for new tickets instead of file_write to %s", rel)
	}
	abs, err := root.ResolvePath(rel)
	if err != nil {
		return nil
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return fmt.Errorf("policy: new ticket files must be created with ticket_create so numbering, backlog placement, and dedupe are enforced; attempted file_write to %s", rel)
	}
	return nil
}

func cleanRepoPath(rel string) string {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(rel))
}

func isTicketLifecycleDir(dir string) bool {
	switch dir {
	case "backlog", "in-progress", "in-review", "done":
		return true
	default:
		return false
	}
}

func checkGitPushPolicy(ctx context.Context, root Root, raw json.RawMessage) error {
	var args gitPushArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	branch := strings.TrimSpace(args.Branch)
	if branch == "" {
		out, err := runGit(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}
		if out.ExitCode != 0 {
			return fmt.Errorf("policy: determine branch before push: %s", strings.TrimSpace(out.Stderr))
		}
		branch = strings.TrimSpace(out.Output)
	}
	if branch != "main" {
		return fmt.Errorf("policy: strict trunk only allows pushing main, got %q", branch)
	}
	return nil
}

func checkShellPolicy(raw json.RawMessage) error {
	var args shellExecArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	cmd := strings.Join(args.Argv, " ")
	if strings.TrimSpace(args.ShellCommand) != "" {
		cmd = args.ShellCommand
	}
	if operation, ok := forbiddenShellOperation(cmd); ok {
		return fmt.Errorf("policy: shell_exec command contains forbidden operation %q", operation)
	}
	return nil
}

func shellExecReadOnly(raw json.RawMessage) bool {
	var args shellExecArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return false
	}
	if args.Background {
		return false
	}
	if len(args.Argv) > 0 {
		return shellTokensReadOnly(args.Argv)
	}
	cmd := strings.TrimSpace(args.ShellCommand)
	if cmd == "" || shellCommandHasControlSyntax(cmd) {
		return false
	}
	return shellTokensReadOnly(shellFields(cmd))
}

func shellCommandHasControlSyntax(cmd string) bool {
	for _, token := range []string{"|", "&&", "||", ";", ">", "<", "`", "$(", "\n"} {
		if strings.Contains(cmd, token) {
			return true
		}
	}
	return false
}

func shellTokensReadOnly(raw []string) bool {
	if len(raw) == 0 {
		return false
	}
	fields := make([]string, 0, len(raw))
	for _, field := range raw {
		field = strings.TrimSpace(strings.ToLower(field))
		field = strings.Trim(field, `"'`)
		if field != "" {
			fields = append(fields, field)
		}
	}
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "ls", "pwd", "cat", "head", "tail", "wc", "test", "grep", "rg":
		return true
	case "sed":
		return sedReadOnly(fields[1:])
	case "find":
		return findReadOnly(fields[1:])
	case "git":
		return gitShellReadOnly(fields)
	default:
		return false
	}
}

func filepathBase(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func sedReadOnly(args []string) bool {
	hasNoPrint := false
	for _, arg := range args {
		switch {
		case arg == "-n":
			hasNoPrint = true
		case strings.HasPrefix(arg, "-") && strings.Contains(arg, "n"):
			hasNoPrint = true
		case arg == "-i" || arg == "--in-place" || strings.HasPrefix(arg, "-i"):
			return false
		}
	}
	return hasNoPrint
}

func findReadOnly(args []string) bool {
	for _, arg := range args {
		switch {
		case arg == "-delete", arg == "-exec", arg == "-execdir", arg == "-ok", arg == "-okdir":
			return false
		case strings.HasPrefix(arg, "-fprint"):
			return false
		}
	}
	return true
}

func gitShellReadOnly(fields []string) bool {
	subcommand, args := gitShellSubcommand(fields)
	switch subcommand {
	case "status", "diff", "log", "show", "rev-parse", "ls-files":
		return !hasGitShellOutputFlag(args)
	case "branch":
		return len(args) == 1 && args[0] == "--show-current"
	default:
		return false
	}
}

func gitShellSubcommand(fields []string) (string, []string) {
	if len(fields) == 0 || filepathBase(fields[0]) != "git" {
		return "", nil
	}
	for i := 1; i < len(fields); i++ {
		token := fields[i]
		switch {
		case token == "-c" || token == "-C":
			i++
			continue
		case strings.HasPrefix(token, "--git-dir") || strings.HasPrefix(token, "--work-tree"):
			continue
		case strings.HasPrefix(token, "-"):
			continue
		default:
			return token, fields[i+1:]
		}
	}
	return "", nil
}

func hasGitShellOutputFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--output" || strings.HasPrefix(arg, "--output=") {
			return true
		}
	}
	return false
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

func shellFields(cmd string) []string {
	raw := strings.Fields(strings.ToLower(cmd))
	fields := make([]string, 0, len(raw))
	for _, field := range raw {
		field = strings.Trim(field, `"'`)
		field = strings.TrimRight(field, ";")
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
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

func validateRepoDiff(ctx context.Context, root Root, session Session) error {
	if err := checkDiffForSecrets(ctx, root); err != nil {
		return err
	}
	limits := session.SafetyLimits
	if limits == (safety.Limits{}) {
		limits = safety.DefaultLimits()
	}
	stats, err := diffStats(ctx, root)
	if err != nil {
		return err
	}
	return safety.Check(stats, limits)
}

// ValidateRepoDiff checks the current repository diff against the same safety
// limits enforced after mutating tool calls.
func ValidateRepoDiff(ctx context.Context, root Root, session Session) error {
	return validateRepoDiff(ctx, root, session)
}

func checkDiffForSecrets(ctx context.Context, root Root) error {
	files, err := changedFiles(ctx, root)
	if err != nil {
		return err
	}
	for _, rel := range files {
		abs, err := root.ResolvePath(rel)
		if err != nil {
			return err
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if hits := safety.ScanForSecrets(rel, string(b)); len(hits) > 0 {
			return fmt.Errorf("policy: secret scanner blocked %s:%d (%s)", hits[0].File, hits[0].Line, hits[0].Pattern)
		}
	}
	return nil
}

func changedFiles(ctx context.Context, root Root) ([]string, error) {
	seen := map[string]bool{}
	tr, err := runGit(ctx, root, "diff", "--name-only", "HEAD", "--")
	if err != nil {
		return nil, err
	}
	var files []string
	if tr.ExitCode == 0 {
		for _, line := range strings.Split(tr.Output, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !seen[line] {
				seen[line] = true
				files = append(files, line)
			}
		}
	}
	untracked, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	if untracked.ExitCode == 0 {
		for _, line := range strings.Split(untracked.Output, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !seen[line] {
				seen[line] = true
				files = append(files, line)
			}
		}
	}
	return files, nil
}

func diffStats(ctx context.Context, root Root) (safety.DiffStats, error) {
	stats := safety.DiffStats{LinesPerFile: map[string]int{}}
	numstat, err := runGit(ctx, root, "diff", "--numstat", "HEAD", "--")
	if err != nil {
		return stats, err
	}
	if numstat.ExitCode != 0 {
		return stats, nil
	}
	for _, line := range strings.Split(numstat.Output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		added := atoiDiffField(fields[0])
		deleted := atoiDiffField(fields[1])
		path := strings.Join(fields[2:], " ")
		lines := added + deleted
		stats.FilesChanged++
		stats.LinesPerFile[path] = lines
		stats.TotalLines += lines
	}
	status, err := runGit(ctx, root, "diff", "--name-status", "HEAD", "--")
	if err != nil {
		return stats, err
	}
	if status.ExitCode != 0 {
		return stats, nil
	}
	for _, line := range strings.Split(status.Output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "D") {
			stats.Deletions++
		}
	}
	untracked, err := runGit(ctx, root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return stats, err
	}
	if untracked.ExitCode == 0 {
		for _, rel := range strings.Split(untracked.Output, "\n") {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}
			abs, err := root.ResolvePath(rel)
			if err != nil {
				return stats, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			lines := strings.Count(string(b), "\n")
			if len(b) > 0 && !strings.HasSuffix(string(b), "\n") {
				lines++
			}
			stats.FilesChanged++
			stats.LinesPerFile[rel] = lines
			stats.TotalLines += lines
		}
	}
	return stats, nil
}

func atoiDiffField(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
