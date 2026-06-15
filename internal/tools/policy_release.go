/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-009-release-update-lifecycle.md
*/
package tools

import (
	"context"
	"fmt"
	"strings"
)

func checkShellReleaseTagPolicy(ctx context.Context, root Root, args shellExecArgs) error {
	tag, target, ok := shellExecReleaseTagMutation(args)
	if !ok {
		return nil
	}
	version := strings.TrimSpace(readOptional(root, "VERSION"))
	if version == "" {
		return fmt.Errorf("policy: release tag %s cannot be created before VERSION exists; generate release notes, commit them, then tag the release-note commit", tag)
	}
	expectedTag := "v" + version
	if tag != strings.ToLower(expectedTag) {
		return fmt.Errorf("policy: release tag %s does not match VERSION %s; create or update %s only after the release-note commit is HEAD", tag, version, expectedTag)
	}
	files, err := changedFiles(ctx, root)
	if err != nil {
		return err
	}
	files = dispositionBlockingFiles(files)
	if len(files) > 0 {
		return fmt.Errorf("policy: release tag %s must be created after VERSION and CHANGELOG.md are committed; uncommitted changes remain: %s. Commit them with git_commit message %q, then tag that release-note commit", expectedTag, summarizeChangedFiles(files), "release: notes "+version)
	}
	headSubject := strings.TrimSpace(gitOutput(ctx, root, "log", "-1", "--pretty=%s"))
	if !strings.HasPrefix(strings.ToLower(headSubject), "release: notes ") {
		return fmt.Errorf("policy: release tag %s must point at a release-note commit, but HEAD subject is %q. Commit VERSION and CHANGELOG.md as %q before tagging", expectedTag, headSubject, "release: notes "+version)
	}
	if target == "" {
		return nil
	}
	head := strings.TrimSpace(gitOutput(ctx, root, "rev-parse", "HEAD"))
	resolved, err := runGit(ctx, root, "rev-parse", "--verify", target+"^{commit}")
	if err != nil {
		return err
	}
	if resolved.ExitCode != 0 {
		return fmt.Errorf("policy: release tag %s target %q is not a commit; tag the current release-note HEAD instead", expectedTag, target)
	}
	targetSHA := strings.TrimSpace(resolved.Output)
	if head == "" || targetSHA != head {
		return fmt.Errorf("policy: release tag %s target %q resolves to %s, not current release-note HEAD %s; create or update the tag at HEAD after committing release notes", expectedTag, target, targetSHA, head)
	}
	return nil
}

func shellExecReleaseTagMutation(args shellExecArgs) (tag string, target string, ok bool) {
	fields := normalizedShellExecFields(args)
	subcommand, subArgs := gitShellSubcommand(fields)
	if subcommand != "tag" {
		return "", "", false
	}
	if len(subArgs) == 0 || gitTagArgsListOnly(subArgs) {
		return "", "", false
	}
	if hasToken(subArgs, "-d") || hasToken(subArgs, "--delete") {
		return "", "", false
	}
	for i := 0; i < len(subArgs); i++ {
		arg := strings.TrimSpace(subArgs[i])
		if arg == "" {
			continue
		}
		if gitTagFlagConsumesNext(arg) {
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if !strings.HasPrefix(arg, "v") {
			return "", "", false
		}
		tag = arg
		if i+1 < len(subArgs) {
			for j := i + 1; j < len(subArgs); j++ {
				candidate := strings.TrimSpace(subArgs[j])
				if gitTagFlagConsumesNext(candidate) {
					j++
					continue
				}
				candidate = strings.TrimSpace(candidate)
				if candidate == "" || strings.HasPrefix(candidate, "-") {
					continue
				}
				target = candidate
				break
			}
		}
		return tag, target, true
	}
	return "", "", false
}

func gitTagArgsListOnly(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		switch {
		case arg == "-l", arg == "--list", strings.HasPrefix(arg, "--list="):
			return true
		case strings.HasPrefix(arg, "-"):
			continue
		default:
			return false
		}
	}
	return true
}

func gitTagFlagConsumesNext(arg string) bool {
	switch arg {
	case "-m", "-F", "-u", "--message", "--file", "--local-user":
		return true
	default:
		return false
	}
}
