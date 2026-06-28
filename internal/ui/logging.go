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
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LoggingConfig controls command-level slog routing.
type LoggingConfig struct {
	Command   string
	LogPath   string
	Debug     bool
	Inline    io.Writer
	Dashboard *TerminalDashboard
	Now       time.Time
}

// InstalledLogger restores global slog state and closes the log file.
type InstalledLogger struct {
	restore func()
	file    *os.File
	path    string
}

// Path returns the log file path.
func (l *InstalledLogger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Close restores the previous default logger and closes the command log file.
func (l *InstalledLogger) Close() error {
	if l == nil {
		return nil
	}
	if l.restore != nil {
		l.restore()
		l.restore = nil
	}
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// DefaultLogPath returns the default per-command log file location.
func DefaultLogPath(command string, now time.Time) (string, error) {
	if now.IsZero() {
		now = time.Now()
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("log path: determine home directory: %w", err)
	}
	name := strings.TrimSpace(command)
	if name == "" {
		name = "command"
	}
	name = strings.NewReplacer("/", "-", string(filepath.Separator), "-").Replace(name)
	return filepath.Join(home, ".mars", "traces", "logs", now.Format("20060102-150405")+"-"+name+".log"), nil
}

// InstallCommandLogger installs a global slog handler for one CLI command.
func InstallCommandLogger(cfg LoggingConfig) (*InstalledLogger, error) {
	path := strings.TrimSpace(cfg.LogPath)
	if path == "" {
		var err error
		path, err = DefaultLogPath(cfg.Command, cfg.Now)
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("log path: create directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log path: open %s: %w", path, err)
	}
	prev := slog.Default()
	handler := &commandLogHandler{
		file:      file,
		inline:    cfg.Inline,
		debug:     cfg.Debug,
		dashboard: cfg.Dashboard,
		attrs:     nil,
		mu:        &sync.Mutex{},
	}
	slog.SetDefault(slog.New(handler))
	return &InstalledLogger{
		path: path,
		file: file,
		restore: func() {
			slog.SetDefault(prev)
		},
	}, nil
}

type commandLogHandler struct {
	file      io.Writer
	inline    io.Writer
	debug     bool
	dashboard *TerminalDashboard
	attrs     []slog.Attr
	group     string
	mu        *sync.Mutex
}

func (h *commandLogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *commandLogHandler) Handle(_ context.Context, r slog.Record) error {
	line := h.format(r)
	h.mu.Lock()
	_, fileErr := io.WriteString(h.file, line)
	if h.debug && h.inline != nil {
		_, _ = io.WriteString(h.inline, line)
	}
	h.mu.Unlock()
	if !h.debug && h.dashboard != nil && r.Level >= slog.LevelWarn {
		h.dashboard.AddWarning(fmt.Sprintf("%s: %s", r.Level.String(), r.Message))
	}
	return fileErr
}

func (h *commandLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *commandLogHandler) WithGroup(name string) slog.Handler {
	clone := *h
	if clone.group == "" {
		clone.group = name
	} else {
		clone.group += "." + name
	}
	return &clone
}

func (h *commandLogHandler) format(r slog.Record) string {
	var b strings.Builder
	when := r.Time
	if when.IsZero() {
		when = time.Now()
	}
	fmt.Fprintf(&b, "%s %s %s", when.Format(time.RFC3339), r.Level.String(), r.Message)
	for _, attr := range h.attrs {
		appendAttr(&b, h.group, attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		appendAttr(&b, h.group, attr)
		return true
	})
	b.WriteByte('\n')
	return b.String()
}

func appendAttr(b *strings.Builder, group string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Key == "" {
		return
	}
	key := attr.Key
	if group != "" {
		key = group + "." + key
	}
	fmt.Fprintf(b, " %s=%q", key, attr.Value.String())
}
