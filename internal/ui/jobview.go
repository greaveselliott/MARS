/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/dashboard.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-010-dashboard-control-plane.md
*/
package ui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// JobViewMeta describes one role execution for a terminal view.
type JobViewMeta struct {
	JobID    string
	RepoID   string
	RepoPath string
	Role     string
	Model    string
	Tools    []string
	Handoff  []string
}

// JobView receives role-run lifecycle and loop progress updates.
type JobView interface {
	WriteHeader(role, model string, tools []string, handoff []string)
	WriteReady()
	WriteToolCall(name, args string)
	WriteToolResult(name, output string)
	WriteAssistant(content string)
	WriteTurn(turn, maxTurns int)
	WriteError(msg string)
	WriteSummary(role, reason string, llmCalls, toolCalls int, duration time.Duration, tokens int)
	WriteHandoff(from string, targets []string)
}

// JobViewFactory creates per-job views. Implementations must be safe for
// concurrent jobs when used by serve workers.
type JobViewFactory interface {
	NewJobView(meta JobViewMeta) JobView
}

// DebugJobViewFactory preserves the existing verbose trace stream. Writes are
// serialized so concurrent workers cannot interleave individual trace lines.
type DebugJobViewFactory struct {
	w       io.Writer
	quiet   bool
	noColor bool
	mu      sync.Mutex
}

// NewDebugJobViewFactory creates verbose trace views.
func NewDebugJobViewFactory(w io.Writer, quiet, noColor bool) *DebugJobViewFactory {
	return &DebugJobViewFactory{w: w, quiet: quiet, noColor: noColor}
}

// NewJobView returns a locked TraceWriter.
func (f *DebugJobViewFactory) NewJobView(meta JobViewMeta) JobView {
	return &lockedJobView{
		mu:    &f.mu,
		inner: NewTraceWriter(f.w, f.quiet, f.noColor),
	}
}

type lockedJobView struct {
	mu    *sync.Mutex
	inner JobView
}

func (v *lockedJobView) withLock(fn func()) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fn()
}

func (v *lockedJobView) WriteHeader(role, model string, tools []string, handoff []string) {
	v.withLock(func() { v.inner.WriteHeader(role, model, tools, handoff) })
}

func (v *lockedJobView) WriteReady() {
	v.withLock(func() { v.inner.WriteReady() })
}

func (v *lockedJobView) WriteToolCall(name, args string) {
	v.withLock(func() { v.inner.WriteToolCall(name, args) })
}

func (v *lockedJobView) WriteToolResult(name, output string) {
	v.withLock(func() { v.inner.WriteToolResult(name, output) })
}

func (v *lockedJobView) WriteAssistant(content string) {
	v.withLock(func() { v.inner.WriteAssistant(content) })
}

func (v *lockedJobView) WriteTurn(turn, maxTurns int) {
	v.withLock(func() { v.inner.WriteTurn(turn, maxTurns) })
}

func (v *lockedJobView) WriteError(msg string) {
	v.withLock(func() { v.inner.WriteError(msg) })
}

func (v *lockedJobView) WriteSummary(role, reason string, llmCalls, toolCalls int, duration time.Duration, tokens int) {
	v.withLock(func() { v.inner.WriteSummary(role, reason, llmCalls, toolCalls, duration, tokens) })
}

func (v *lockedJobView) WriteHandoff(from string, targets []string) {
	v.withLock(func() { v.inner.WriteHandoff(from, targets) })
}

// PlainJobViewFactory emits concise, scrollback-friendly job summaries for
// non-TTY output.
type PlainJobViewFactory struct {
	w  io.Writer
	mu sync.Mutex
}

// NewPlainJobViewFactory creates concise non-interactive views.
func NewPlainJobViewFactory(w io.Writer) *PlainJobViewFactory {
	return &PlainJobViewFactory{w: w}
}

// NewJobView returns a concise view.
func (f *PlainJobViewFactory) NewJobView(meta JobViewMeta) JobView {
	return &plainJobView{w: f.w, mu: &f.mu, meta: meta}
}

type plainJobView struct {
	w    io.Writer
	mu   *sync.Mutex
	meta JobViewMeta
}

func (v *plainJobView) printf(format string, args ...any) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fmt.Fprintf(v.w, format, args...)
}

func (v *plainJobView) WriteHeader(role, model string, tools []string, handoff []string) {
	if role == "" {
		role = v.meta.Role
	}
	if model == "" {
		model = v.meta.Model
	}
	v.printf("mars: %s starting (model=%s, tools=%d)\n", role, model, len(tools))
}

func (v *plainJobView) WriteReady() {
	v.printf("mars: %s inference ready\n", v.meta.Role)
}

func (v *plainJobView) WriteToolCall(name, args string) {}

func (v *plainJobView) WriteToolResult(name, output string) {}

func (v *plainJobView) WriteAssistant(content string) {
	content = truncate(content, 160)
	if content != "" {
		v.printf("mars: %s replied: %s\n", v.meta.Role, content)
	}
}

func (v *plainJobView) WriteTurn(turn, maxTurns int) {}

func (v *plainJobView) WriteError(msg string) {
	v.printf("mars: %s error: %s\n", v.meta.Role, msg)
}

func (v *plainJobView) WriteSummary(role, reason string, llmCalls, toolCalls int, duration time.Duration, tokens int) {
	if role == "" {
		role = v.meta.Role
	}
	v.printf("mars: %s finished reason=%s turns=%d tools=%d time=%s\n",
		role, reason, llmCalls, toolCalls, duration.Round(time.Millisecond))
}

func (v *plainJobView) WriteHandoff(from string, targets []string) {
	if len(targets) == 0 {
		return
	}
	v.printf("mars: %s handoff to %s\n", from, joinTargets(targets))
}

func joinTargets(targets []string) string {
	out := ""
	for i, t := range targets {
		if i > 0 {
			out += ", "
		}
		out += t
	}
	return out
}
