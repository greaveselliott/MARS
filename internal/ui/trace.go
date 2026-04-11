package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiDim    = "\033[2m"
)

// TraceWriter outputs agent loop progress to the terminal with colour coding.
type TraceWriter struct {
	w       io.Writer
	quiet   bool
	noColor bool
}

// NewTraceWriter creates a writer that renders agent progress.
// When quiet is true only the final summary is printed.
// Colour is auto-disabled when w is not a TTY, or forced off with noColor.
func NewTraceWriter(w io.Writer, quiet, noColor bool) *TraceWriter {
	if !noColor {
		noColor = !isTTY(w)
	}
	return &TraceWriter{w: w, quiet: quiet, noColor: noColor}
}

// WriteToolCall prints a tool invocation line.
func (tw *TraceWriter) WriteToolCall(name, args string) {
	if tw.quiet {
		return
	}
	args = truncate(args, 120)
	tw.printf("%s▶ %s%s %s%s\n",
		tw.color(ansiCyan), name, tw.color(ansiReset),
		tw.color(ansiDim), args,
	)
}

// WriteToolResult prints a tool result summary.
func (tw *TraceWriter) WriteToolResult(name, truncatedOutput string) {
	if tw.quiet {
		return
	}
	truncatedOutput = truncate(truncatedOutput, 200)
	tw.printf("%s◀ %s%s %s\n",
		tw.color(ansiCyan), name, tw.color(ansiReset),
		truncatedOutput,
	)
}

// WriteAssistant prints assistant text.
func (tw *TraceWriter) WriteAssistant(content string) {
	if tw.quiet {
		return
	}
	tw.printf("%s%s%s\n", tw.color(ansiGreen), content, tw.color(ansiReset))
}

// WriteError prints an error message.
func (tw *TraceWriter) WriteError(msg string) {
	tw.printf("%s✖ %s%s\n", tw.color(ansiRed), msg, tw.color(ansiReset))
}

// WriteSummary prints the final run summary.
func (tw *TraceWriter) WriteSummary(reason string, llmCalls, toolCalls int, duration time.Duration, tokens int) {
	tw.printf("\n%s%s── run complete ──%s\n", tw.color(ansiYellow), tw.color(ansiBold), tw.color(ansiReset))
	tw.printf("%s  outcome:    %s%s\n", tw.color(ansiYellow), reason, tw.color(ansiReset))
	tw.printf("%s  llm calls:  %d%s\n", tw.color(ansiYellow), llmCalls, tw.color(ansiReset))
	tw.printf("%s  tool calls: %d%s\n", tw.color(ansiYellow), toolCalls, tw.color(ansiReset))
	tw.printf("%s  duration:   %s%s\n", tw.color(ansiYellow), duration.Round(time.Millisecond), tw.color(ansiReset))
	if tokens > 0 {
		tw.printf("%s  tokens:     ~%d%s\n", tw.color(ansiYellow), tokens, tw.color(ansiReset))
	}
}

func (tw *TraceWriter) printf(format string, args ...any) {
	fmt.Fprintf(tw.w, format, args...)
}

func (tw *TraceWriter) color(code string) string {
	if tw.noColor {
		return ""
	}
	return code
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TIOCGETA)
	return err == nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
