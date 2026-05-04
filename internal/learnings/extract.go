/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dogfood-and-decisions.md
- docs/features/F-012-self-improvement-loop.md
*/
package learnings

import (
	"fmt"
	"log/slog"
	"strings"
)

// ExtractFailureLesson derives a lesson from a failed job. Returns empty
// string if no actionable lesson can be extracted.
func ExtractFailureLesson(role, errMsg string, lastToolOutput string) string {
	lower := strings.ToLower(errMsg)

	switch {
	case strings.Contains(lower, "eresolve") && strings.Contains(lower, "npm"):
		return "npm install fails with ERESOLVE — use --legacy-peer-deps"
	case strings.Contains(lower, "eresolve") && strings.Contains(lower, "peer dep"):
		return "npm install fails with peer dependency conflicts — use --legacy-peer-deps or --force"

	case strings.Contains(lower, "command not found"):
		cmd := extractQuoted(errMsg)
		if cmd != "" {
			return fmt.Sprintf("Command %q not found — may need to install it or use full path", cmd)
		}

	case strings.Contains(lower, "permission denied"):
		return "Permission denied on shell command — check file permissions or use sudo if appropriate"

	case strings.Contains(lower, "enospc"):
		return "Disk space exhausted — clean build artifacts or increase disk"

	case strings.Contains(lower, "context size") || strings.Contains(lower, "context_overflow"):
		return "Context window overflow — use targeted file reads instead of broad find/grep commands"

	case strings.Contains(lower, "timed out") || strings.Contains(lower, "context deadline exceeded"):
		if strings.Contains(lower, "next dev") || strings.Contains(lower, "npm start") || strings.Contains(lower, "dev server") {
			return "Long-running dev server timed out — use shell_exec with background:true"
		}
	}

	if lastToolOutput != "" {
		return extractFromToolOutput(role, lastToolOutput)
	}

	return ""
}

// ExtractSuccessLesson checks for non-obvious patterns in successful runs.
func ExtractSuccessLesson(role string, toolCalls []string) string {
	for _, call := range toolCalls {
		lower := strings.ToLower(call)
		if strings.Contains(lower, "--legacy-peer-deps") {
			return "This repo requires --legacy-peer-deps for npm install"
		}
		if strings.Contains(lower, "--force") && strings.Contains(lower, "install") {
			return "This repo requires --force flag for package installation"
		}
	}
	return ""
}

// RecordJobLessons extracts and saves lessons from a completed or failed job.
func RecordJobLessons(store *Store, role, errMsg, lastToolOutput string, toolCalls []string) {
	if store == nil {
		return
	}

	if errMsg != "" {
		lesson := ExtractFailureLesson(role, errMsg, lastToolOutput)
		if lesson != "" {
			added, err := store.AddLesson(role, "failure_avoidance", lesson)
			if err != nil {
				slog.Warn("learnings: failed to save failure lesson", "err", err)
			} else if added {
				slog.Info("learnings: recorded failure lesson", "role", role, "lesson", lesson)
			}
		}
	} else {
		lesson := ExtractSuccessLesson(role, toolCalls)
		if lesson != "" {
			added, err := store.AddLesson(role, "convention", lesson)
			if err != nil {
				slog.Warn("learnings: failed to save success lesson", "err", err)
			} else if added {
				slog.Info("learnings: recorded success lesson", "role", role, "lesson", lesson)
			}
		}
	}
}

func extractQuoted(s string) string {
	start := strings.Index(s, "'")
	if start < 0 {
		start = strings.Index(s, "\"")
	}
	if start < 0 {
		return ""
	}
	quote := s[start : start+1]
	end := strings.Index(s[start+1:], quote)
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

func extractFromToolOutput(role, output string) string {
	lower := strings.ToLower(output)

	if strings.Contains(lower, "module not found") {
		idx := strings.Index(lower, "module not found")
		snippet := output[max(0, idx-20):min(len(output), idx+60)]
		return fmt.Sprintf("Module not found error — check imports: %s", strings.TrimSpace(snippet))
	}

	if strings.Contains(lower, "cannot find module") {
		return "Missing module — run package install before build"
	}

	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
