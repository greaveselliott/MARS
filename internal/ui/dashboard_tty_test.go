/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/dashboard.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-005-agent-execution-runtime.md
*/
package ui

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTerminalDashboardRendersJobStateAndSuppressesRawToolOutput(t *testing.T) {
	var buf bytes.Buffer
	dash := NewTerminalDashboard(&buf, &fakeStatusProvider{healthy: true}, DashboardOptions{
		Command:  "run",
		RepoPath: "/tmp/example",
		LogPath:  "/tmp/mars.log",
		Controls: "Ctrl+C cancel",
		Force:    true,
	})
	dash.Start()

	view := dash.NewJobView(JobViewMeta{JobID: "job-1", Role: "engineer", Model: "coding"})
	view.WriteHeader("engineer", "coding", []string{"shell_exec"}, nil)
	view.WriteReady()
	view.WriteTurn(3, 50)
	view.WriteToolCall("shell_exec", `{"argv":["npm","test"]}`)
	view.WriteToolResult("shell_exec", strings.Repeat("raw dependency output ", 20))
	view.WriteError("workspace hygiene blocked")

	dash.Stop()
	out := buf.String()
	require.Contains(t, out, "\033[?1049h")
	require.Contains(t, out, "engineer")
	require.Contains(t, out, "turn: 3/50")
	require.Contains(t, out, "shell_exec")
	require.Contains(t, out, "workspace hygiene blocked")
	require.NotContains(t, out, "raw dependency output")
}

func TestPlainJobViewSuppressesToolResults(t *testing.T) {
	var buf bytes.Buffer
	view := NewPlainJobViewFactory(&buf).NewJobView(JobViewMeta{Role: "engineer"})

	view.WriteHeader("engineer", "coding", []string{"shell_exec"}, nil)
	view.WriteToolCall("shell_exec", "{}")
	view.WriteToolResult("shell_exec", "very noisy output")
	view.WriteSummary("engineer", "completed", 2, 1, time.Second, 0)

	out := buf.String()
	require.Contains(t, out, "engineer starting")
	require.Contains(t, out, "finished")
	require.NotContains(t, out, "very noisy output")
}

func TestInstallCommandLoggerWritesFileAndDashboardWarnings(t *testing.T) {
	var buf bytes.Buffer
	logPath := filepath.Join(t.TempDir(), "mars.log")
	dash := NewTerminalDashboard(&buf, &fakeStatusProvider{healthy: true}, DashboardOptions{Force: true})
	dash.Start()

	logger, err := InstallCommandLogger(LoggingConfig{
		Command:   "serve",
		LogPath:   logPath,
		Dashboard: dash,
	})
	require.NoError(t, err)
	defer logger.Close()

	slog.Warn("operator-visible warning", "role", "engineer")
	dash.Stop()

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "operator-visible warning")
	require.Contains(t, string(data), "role=\"engineer\"")
	require.Contains(t, buf.String(), "operator-visible warning")
}

func TestInstallCommandLoggerDebugStreamsInline(t *testing.T) {
	var inline bytes.Buffer
	logPath := filepath.Join(t.TempDir(), "mars.log")
	logger, err := InstallCommandLogger(LoggingConfig{
		Command: "run",
		LogPath: logPath,
		Debug:   true,
		Inline:  &inline,
	})
	require.NoError(t, err)
	defer logger.Close()

	slog.Info("debug detail", "tool", "grep")

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "debug detail")
	require.Contains(t, inline.String(), "debug detail")
}
