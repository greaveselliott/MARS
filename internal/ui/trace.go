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
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiItalic    = "\033[3m"
	ansiRed       = "\033[31m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiBlue      = "\033[34m"
	ansiMagenta   = "\033[35m"
	ansiCyan      = "\033[36m"
	ansiWhite     = "\033[37m"
	ansiBgBlue    = "\033[44m"
	ansiBgMagenta = "\033[45m"
)

// TraceWriter outputs agent loop progress to the terminal.
type TraceWriter struct {
	w       io.Writer
	quiet   bool
	noColor bool
}

// NewTraceWriter creates a writer that renders agent progress.
func NewTraceWriter(w io.Writer, quiet, noColor bool) *TraceWriter {
	if !noColor {
		noColor = !isTTY(w)
	}
	return &TraceWriter{w: w, quiet: quiet, noColor: noColor}
}

// WriteHeader prints the initial agent banner with role, model, and handoff info.
func (tw *TraceWriter) WriteHeader(role, model string, tools []string, handoff []string) {
	tw.printf("\n")
	tw.printf("%s%s %s %s\n",
		tw.color(ansiBold+ansiWhite+ansiBgBlue), " MARS HARNESS ", tw.color(ansiReset),
		tw.color(ansiDim)+"autonomous agent"+tw.color(ansiReset))
	tw.printf("\n")
	tw.printf("  %s%srole%s      %s%s%s\n",
		tw.color(ansiDim), tw.color(ansiBold), tw.color(ansiReset),
		tw.color(ansiCyan+ansiBold), role, tw.color(ansiReset))
	tw.printf("  %s%smodel%s     %s\n",
		tw.color(ansiDim), tw.color(ansiBold), tw.color(ansiReset), model)
	tw.printf("  %s%stools%s     %s\n",
		tw.color(ansiDim), tw.color(ansiBold), tw.color(ansiReset),
		strings.Join(tools, ", "))
	if len(handoff) > 0 {
		tw.printf("  %s%shandoff%s   %s%s%s\n",
			tw.color(ansiDim), tw.color(ansiBold), tw.color(ansiReset),
			tw.color(ansiMagenta), strings.Join(handoff, " → "), tw.color(ansiReset))
	}
	tw.printf("\n")
	tw.printf("  %sloading model...%s\n", tw.color(ansiDim), tw.color(ansiReset))
}

// WriteReady prints that inference is ready and the agent loop is starting.
func (tw *TraceWriter) WriteReady() {
	tw.printf("  %s%s✓ inference ready%s\n\n",
		tw.color(ansiGreen), tw.color(ansiBold), tw.color(ansiReset))
}

// WriteTurn prints the current turn separator.
func (tw *TraceWriter) WriteTurn(turn, maxTurns int) {
	if tw.quiet {
		return
	}
	tw.printf("%s─── turn %d/%d %s\n",
		tw.color(ansiDim), turn, maxTurns,
		tw.color(ansiReset))
}

// WriteToolCall prints a tool invocation.
func (tw *TraceWriter) WriteToolCall(name, args string) {
	if tw.quiet {
		return
	}
	args = truncate(args, 120)
	tw.printf("  %s▶%s %s%s%s %s%s%s\n",
		tw.color(ansiBlue), tw.color(ansiReset),
		tw.color(ansiCyan+ansiBold), name, tw.color(ansiReset),
		tw.color(ansiDim), args, tw.color(ansiReset))
}

// WriteToolResult prints a tool result.
func (tw *TraceWriter) WriteToolResult(name, output string) {
	if tw.quiet {
		return
	}
	output = truncate(output, 200)
	tw.printf("  %s◀%s %s%s%s %s%s%s\n",
		tw.color(ansiBlue), tw.color(ansiReset),
		tw.color(ansiCyan), name, tw.color(ansiReset),
		tw.color(ansiDim), output, tw.color(ansiReset))
}

// WriteAssistant prints the agent's text response.
func (tw *TraceWriter) WriteAssistant(content string) {
	if tw.quiet {
		return
	}
	tw.printf("\n  %s%s%s\n\n",
		tw.color(ansiGreen), content, tw.color(ansiReset))
}

// WriteError prints an error.
func (tw *TraceWriter) WriteError(msg string) {
	tw.printf("  %s%s✖ %s%s\n",
		tw.color(ansiRed), tw.color(ansiBold), msg, tw.color(ansiReset))
}

// WriteSummary prints the final run summary.
func (tw *TraceWriter) WriteSummary(role, reason string, llmCalls, toolCalls int, duration time.Duration, tokens int) {
	outcome := reason
	outcomeColor := ansiGreen
	switch reason {
	case "completed":
		outcome = "completed successfully"
	case "max_turns":
		outcome = "hit turn limit"
		outcomeColor = ansiYellow
	case "llm_unreachable":
		outcome = "inference error"
		outcomeColor = ansiRed
	case "circle_detected":
		outcome = "stuck in loop"
		outcomeColor = ansiRed
	case "budget_exceeded":
		outcome = "token budget exceeded"
		outcomeColor = ansiYellow
	case "timeout":
		outcome = "wall time exceeded"
		outcomeColor = ansiYellow
	}

	tw.printf("\n%s%s%s ── %s complete ── %s%s\n",
		tw.color(ansiBold), tw.color(outcomeColor),
		strings.ToUpper(string(outcome[0]))+outcome[1:],
		role, tw.color(ansiReset), tw.color(ansiReset))
	tw.printf("  %sturns%s  %d    %stools%s  %d    %stime%s  %s",
		tw.color(ansiDim+ansiBold), tw.color(ansiReset), llmCalls,
		tw.color(ansiDim+ansiBold), tw.color(ansiReset), toolCalls,
		tw.color(ansiDim+ansiBold), tw.color(ansiReset), duration.Round(time.Millisecond))
	if tokens > 0 {
		tw.printf("    %stokens%s  ~%d",
			tw.color(ansiDim+ansiBold), tw.color(ansiReset), tokens)
	}
	tw.printf("\n\n")
}

// WriteHandoff prints that the agent is handing off to the next role.
func (tw *TraceWriter) WriteHandoff(from string, targets []string) {
	if len(targets) == 0 {
		return
	}
	tw.printf("  %s%s→ handing off to: %s%s\n\n",
		tw.color(ansiMagenta), tw.color(ansiBold),
		strings.Join(targets, ", "), tw.color(ansiReset))
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
	_, err := unix.IoctlGetTermios(int(f.Fd()), ioctlGetTermios)
	return err == nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
