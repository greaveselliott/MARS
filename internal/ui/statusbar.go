/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// StatusProvider supplies the data the status bar renders.
type StatusProvider interface {
	IsPaused() bool
	Healthy() bool
}

// StatusBar renders a single-line ANSI bar at the bottom of the terminal
// showing the orchestrator state and key hints.
type StatusBar struct {
	w         io.Writer
	provider  StatusProvider
	startedAt time.Time
	flash     string
	flashExp  time.Time
	mu        sync.Mutex
	noColor   bool
	stop      chan struct{}
}

// NewStatusBar creates a status bar writing to w.
func NewStatusBar(w io.Writer, provider StatusProvider) *StatusBar {
	return &StatusBar{
		w:         w,
		provider:  provider,
		startedAt: time.Now(),
		noColor:   !isTTY(w),
		stop:      make(chan struct{}),
	}
}

// Start begins a goroutine that redraws the status bar every 2 seconds.
func (sb *StatusBar) Start() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		sb.Redraw()
		for {
			select {
			case <-sb.stop:
				return
			case <-ticker.C:
				sb.Redraw()
			}
		}
	}()
}

// Stop halts the status bar refresh loop and clears the line.
func (sb *StatusBar) Stop() {
	select {
	case sb.stop <- struct{}{}:
	default:
	}
	fmt.Fprintf(sb.w, "\r\033[K")
}

// Redraw renders the status bar immediately.
func (sb *StatusBar) Redraw() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	state := "RUNNING"
	stateColor := ansiGreen
	if sb.provider.IsPaused() {
		state = "PAUSED"
		stateColor = ansiYellow
	} else if !sb.provider.Healthy() {
		state = "STOPPED"
		stateColor = ansiRed
	}

	uptime := time.Since(sb.startedAt).Truncate(time.Second)

	var line string
	if sb.noColor {
		line = fmt.Sprintf("\r\033[K  %s | up %s | [p]ause [r]estart [s]can [q]uit [h]elp",
			state, uptime)
	} else {
		line = fmt.Sprintf("\r\033[K  %s%s%s%s | up %s | %s[p]%sause %s[r]%sestart %s[s]%scan %s[q]%suit %s[h]%selp",
			ansiBold, stateColor, state, ansiReset,
			uptime,
			ansiBold, ansiReset,
			ansiBold, ansiReset,
			ansiBold, ansiReset,
			ansiBold, ansiReset,
			ansiBold, ansiReset,
		)
	}

	if sb.flash != "" && time.Now().Before(sb.flashExp) {
		line += fmt.Sprintf("  %s→ %s%s", ansiCyan, sb.flash, ansiReset)
	}

	fmt.Fprint(sb.w, line)
}

// Flash shows a temporary message on the status bar for 4 seconds.
func (sb *StatusBar) Flash(msg string) {
	sb.mu.Lock()
	sb.flash = msg
	sb.flashExp = time.Now().Add(4 * time.Second)
	sb.mu.Unlock()
	sb.Redraw()
}

// PrintAbove prints a message above the status bar by clearing the line,
// printing the message, then redrawing the bar.
func (sb *StatusBar) PrintAbove(msg string) {
	sb.mu.Lock()
	fmt.Fprintf(sb.w, "\r\033[K%s\n", msg)
	sb.mu.Unlock()
	sb.Redraw()
}

// Writer returns the underlying writer for other components that need
// to print interleaved output (they should use PrintAbove instead).
func (sb *StatusBar) Writer() io.Writer {
	return os.Stderr
}
