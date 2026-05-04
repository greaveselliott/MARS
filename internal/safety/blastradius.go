/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/features/F-007-guardrails-and-safety.md
*/
package safety

import (
	"fmt"
	"log/slog"
)

// Limits defines blast radius constraints.
type Limits struct {
	MaxFilesPerJob    int  // max files changed per job
	MaxLinesPerFile   int  // max lines changed per single file
	MaxTotalLines     int  // max total lines changed per job
	MaxOpenPRsPerRepo int  // max concurrent open PRs per repo
	ForbidDelete      bool // prevent file deletions
}

// DefaultLimits returns conservative defaults. File-count caps are opt-in
// because file count is a weak proxy for risk; line volume and deletions are
// better default blast-radius signals.
func DefaultLimits() Limits {
	return Limits{
		MaxFilesPerJob:    0,
		MaxLinesPerFile:   500,
		MaxTotalLines:     2000,
		MaxOpenPRsPerRepo: 3,
		ForbidDelete:      true,
	}
}

// DiffStats tracks changes made during a job.
type DiffStats struct {
	FilesChanged int
	LinesPerFile map[string]int
	TotalLines   int
	Deletions    int
}

// Check validates current diff stats against limits.
// Returns nil if within limits, or a descriptive error.
func Check(stats DiffStats, limits Limits) error {
	if limits.MaxFilesPerJob > 0 && stats.FilesChanged > limits.MaxFilesPerJob {
		slog.Warn("blast radius: file count exceeded",
			"files", stats.FilesChanged, "limit", limits.MaxFilesPerJob)
		return fmt.Errorf("blast radius exceeded: %d files changed (limit %d). "+
			"Split this job into smaller changes or raise MaxFilesPerJob",
			stats.FilesChanged, limits.MaxFilesPerJob)
	}

	if limits.MaxLinesPerFile > 0 {
		for file, lines := range stats.LinesPerFile {
			if lines > limits.MaxLinesPerFile {
				slog.Warn("blast radius: per-file line count exceeded",
					"file", file, "lines", lines, "limit", limits.MaxLinesPerFile)
				return fmt.Errorf("blast radius exceeded: %d lines changed in %s (limit %d). "+
					"Break the file changes into smaller increments or raise MaxLinesPerFile",
					lines, file, limits.MaxLinesPerFile)
			}
		}
	}

	if limits.MaxTotalLines > 0 && stats.TotalLines > limits.MaxTotalLines {
		slog.Warn("blast radius: total line count exceeded",
			"total", stats.TotalLines, "limit", limits.MaxTotalLines)
		return fmt.Errorf("blast radius exceeded: %d total lines changed (limit %d). "+
			"Split this job into smaller changes or raise MaxTotalLines",
			stats.TotalLines, limits.MaxTotalLines)
	}

	if limits.ForbidDelete && stats.Deletions > 0 {
		slog.Warn("blast radius: file deletions forbidden",
			"deletions", stats.Deletions)
		return fmt.Errorf("blast radius exceeded: %d file deletions attempted but ForbidDelete is enabled. "+
			"Remove deletion operations or disable ForbidDelete",
			stats.Deletions)
	}

	return nil
}
