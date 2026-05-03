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
)

var mutatingTools = map[string]bool{
	"file_write":      true,
	"shell_exec":      true,
	"git_commit":      true,
	"git_push":        true,
	"record_decision": true,
	"ticket_create":   true,
	"tool_create":     true,
}

func preToolPolicy(ctx context.Context, root Root, name string, raw json.RawMessage) error {
	session, hasSession := SessionFromContext(ctx)
	if hasSession {
		if err := enforceTrust(session, name); err != nil {
			return err
		}
	}

	switch name {
	case "file_write":
		return checkFileWritePolicy(session, hasSession, raw)
	case "git_commit":
		return validateRepoDiff(ctx, root, session)
	case "git_push":
		return checkGitPushPolicy(ctx, root, raw)
	case "shell_exec":
		return checkShellPolicy(raw)
	default:
		return nil
	}
}

func postToolPolicy(ctx context.Context, root Root, name string) error {
	if !mutatingTools[name] {
		return nil
	}
	session, _ := SessionFromContext(ctx)
	switch name {
	case "git_commit", "git_push":
		return nil
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

func checkFileWritePolicy(session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession {
		return nil
	}
	var args fileWriteArgs
	if err := json.Unmarshal(raw, &args); err != nil {
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
	lower := strings.ToLower(cmd)
	blocked := []string{
		"git push --force",
		"git push -f",
		"git reset --hard",
		"git clean -fd",
		"git checkout -b",
		"git branch -d",
		"git branch -D",
		"rm -rf /",
	}
	for _, phrase := range blocked {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			return fmt.Errorf("policy: shell_exec command contains forbidden operation %q", phrase)
		}
	}
	return nil
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

func repoRel(root Root, abs string) string {
	rel, err := filepath.Rel(root.Abs(), abs)
	if err != nil {
		return abs
	}
	return rel
}
