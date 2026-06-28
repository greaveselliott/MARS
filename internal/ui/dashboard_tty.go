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
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DashboardOptions controls the full-screen terminal dashboard.
type DashboardOptions struct {
	Title           string
	Command         string
	RepoPath        string
	DashboardURL    string
	LogPath         string
	Controls        string
	RefreshInterval time.Duration

	// Force enables dashboard control sequences for tests with non-TTY writers.
	Force bool
}

// TerminalDashboard renders a compact live dashboard in the terminal using
// ANSI alternate-screen control sequences.
type TerminalDashboard struct {
	w         io.Writer
	provider  StatusProvider
	opts      DashboardOptions
	startedAt time.Time
	active    bool
	started   bool
	stop      chan struct{}
	done      chan struct{}

	mu        sync.Mutex
	flash     string
	flashExp  time.Time
	events    []dashboardEvent
	warnings  []dashboardEvent
	jobs      map[string]*dashboardJobState
	activeJob string
}

type dashboardEvent struct {
	At      time.Time
	Kind    string
	Message string
}

type dashboardJobState struct {
	Meta        JobViewMeta
	StartedAt   time.Time
	EndedAt     time.Time
	LastPhaseAt time.Time
	Ready       bool
	Phase       string
	Turn        int
	MaxTurns    int
	ToolCalls   int
	LastTool    string
	LastError   string
	LastOutput  string
	Summary     string
	Outcome     string
}

// NewTerminalDashboard creates a full-screen terminal dashboard when the writer
// is a TTY. Non-TTY writers are left inactive unless Force is set.
func NewTerminalDashboard(w io.Writer, provider StatusProvider, opts DashboardOptions) *TerminalDashboard {
	if opts.RefreshInterval <= 0 {
		opts.RefreshInterval = time.Second
	}
	if opts.Title == "" {
		opts.Title = "MARS"
	}
	return &TerminalDashboard{
		w:         w,
		provider:  provider,
		opts:      opts,
		startedAt: time.Now(),
		active:    opts.Force || isTTY(w),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		jobs:      make(map[string]*dashboardJobState),
	}
}

// Active reports whether the dashboard owns the terminal.
func (d *TerminalDashboard) Active() bool {
	return d != nil && d.active
}

// SetStatusProvider updates the source used for running/paused/stopped state.
func (d *TerminalDashboard) SetStatusProvider(provider StatusProvider) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.provider = provider
	d.mu.Unlock()
	d.Redraw()
}

// Start enters alternate-screen mode and begins redrawing.
func (d *TerminalDashboard) Start() {
	if d == nil || !d.active {
		return
	}
	d.mu.Lock()
	if d.started {
		d.mu.Unlock()
		return
	}
	d.started = true
	d.mu.Unlock()
	fmt.Fprint(d.w, "\033[?1049h\033[?25l")
	go func() {
		ticker := time.NewTicker(d.opts.RefreshInterval)
		defer ticker.Stop()
		defer close(d.done)
		d.Redraw()
		for {
			select {
			case <-d.stop:
				return
			case <-ticker.C:
				d.Redraw()
			}
		}
	}()
}

// Stop exits alternate-screen mode.
func (d *TerminalDashboard) Stop() {
	if d == nil || !d.active {
		return
	}
	d.mu.Lock()
	if !d.started {
		d.mu.Unlock()
		return
	}
	d.started = false
	d.mu.Unlock()
	select {
	case d.stop <- struct{}{}:
	default:
	}
	select {
	case <-d.done:
	case <-time.After(250 * time.Millisecond):
	}
	fmt.Fprint(d.w, "\033[?25h\033[?1049l")
}

// NewJobView creates a dashboard-backed job view.
func (d *TerminalDashboard) NewJobView(meta JobViewMeta) JobView {
	if d == nil || !d.active {
		return NewPlainJobViewFactory(io.Discard).NewJobView(meta)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if meta.JobID == "" {
		meta.JobID = fmt.Sprintf("%s-%d", meta.Role, time.Now().UnixNano())
	}
	now := time.Now()
	d.jobs[meta.JobID] = &dashboardJobState{Meta: meta, StartedAt: now, LastPhaseAt: now, Phase: "starting"}
	d.activeJob = meta.JobID
	d.appendEventLocked("job", fmt.Sprintf("%s starting", meta.Role))
	d.redrawLocked()
	return &dashboardJobView{dash: d, jobID: meta.JobID}
}

// AddEvent appends a visible lifecycle event.
func (d *TerminalDashboard) AddEvent(kind, message string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.appendEventLocked(kind, message)
	d.redrawLocked()
}

// AddWarning appends a visible warning without flooding the main event list.
func (d *TerminalDashboard) AddWarning(message string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	ev := dashboardEvent{At: time.Now(), Kind: "warn", Message: message}
	d.warnings = appendBounded(d.warnings, ev, 5)
	d.appendEventLocked("warn", message)
	d.redrawLocked()
}

// Flash shows a transient operator message.
func (d *TerminalDashboard) Flash(msg string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.flash = msg
	d.flashExp = time.Now().Add(4 * time.Second)
	d.mu.Unlock()
	d.Redraw()
}

// PrintAbove records a message as an event in full-screen mode.
func (d *TerminalDashboard) PrintAbove(msg string) {
	d.AddEvent("info", msg)
}

// Redraw renders the current dashboard frame.
func (d *TerminalDashboard) Redraw() {
	if d == nil || !d.active {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started {
		return
	}
	d.redrawLocked()
}

func (d *TerminalDashboard) appendEventLocked(kind, message string) {
	ev := dashboardEvent{At: time.Now(), Kind: kind, Message: message}
	d.events = appendBounded(d.events, ev, 12)
}

func appendBounded(in []dashboardEvent, ev dashboardEvent, max int) []dashboardEvent {
	in = append(in, ev)
	if len(in) > max {
		in = in[len(in)-max:]
	}
	return in
}

func (d *TerminalDashboard) redrawLocked() {
	if !d.active {
		return
	}
	if !d.started {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\033[H\033[2J")

	state := "RUNNING"
	if d.provider != nil && d.provider.IsPaused() {
		state = "PAUSED"
	} else if d.provider != nil && !d.provider.Healthy() {
		state = "STOPPED"
	}
	activeJobs := d.activeJobsLocked()
	fmt.Fprintf(&b, "%s%s%s  %s | up %s | active jobs %d\n",
		ansiBold, d.opts.Title, ansiReset, state, time.Since(d.startedAt).Truncate(time.Second), activeJobs)
	fmt.Fprintf(&b, "command: %-12s repo: %s\n", emptyDash(d.opts.Command), displayRepo(d.opts.RepoPath))
	fmt.Fprintf(&b, "web: %-31s logs: %s\n", emptyDash(d.opts.DashboardURL), emptyDash(d.opts.LogPath))
	if d.opts.Controls != "" {
		fmt.Fprintf(&b, "controls: %s\n", d.opts.Controls)
	}
	if d.flash != "" && time.Now().Before(d.flashExp) {
		fmt.Fprintf(&b, "%s\n", truncate("notice: "+d.flash, 120))
	}

	fmt.Fprintf(&b, "\n%sCurrent Work%s\n", ansiBold, ansiReset)
	if job := d.currentJobLocked(); job != nil {
		elapsed := time.Since(job.StartedAt).Truncate(time.Second)
		if !job.EndedAt.IsZero() {
			elapsed = job.EndedAt.Sub(job.StartedAt).Truncate(time.Second)
		}
		turn := "-"
		if job.MaxTurns > 0 {
			turn = fmt.Sprintf("%d/%d", job.Turn, job.MaxTurns)
		}
		status := jobStatusLocked(job)
		if job.Outcome != "" {
			status = job.Outcome
		}
		fmt.Fprintf(&b, "role: %-16s model: %-16s status: %s\n", job.Meta.Role, job.Meta.Model, status)
		fmt.Fprintf(&b, "job: %-36s turn: %-8s tools: %-4d elapsed: %s\n",
			job.Meta.JobID, turn, job.ToolCalls, elapsed)
		if job.LastTool != "" {
			fmt.Fprintf(&b, "last tool: %s\n", truncate(job.LastTool, 96))
		}
		if job.LastError != "" {
			fmt.Fprintf(&b, "blocker: %s\n", truncate(job.LastError, 110))
		}
		if job.Summary != "" {
			fmt.Fprintf(&b, "summary: %s\n", truncate(job.Summary, 110))
		}
	} else {
		fmt.Fprintf(&b, "No active job yet.\n")
	}

	if len(d.warnings) > 0 {
		fmt.Fprintf(&b, "\n%sWarnings%s\n", ansiBold, ansiReset)
		for _, ev := range d.warnings {
			fmt.Fprintf(&b, "%s  %s\n", ev.At.Format("15:04:05"), truncate(ev.Message, 110))
		}
	}

	fmt.Fprintf(&b, "\n%sRecent Events%s\n", ansiBold, ansiReset)
	if len(d.events) == 0 {
		fmt.Fprintf(&b, "No events yet.\n")
	} else {
		for i := len(d.events) - 1; i >= 0; i-- {
			ev := d.events[i]
			fmt.Fprintf(&b, "%s  %-6s %s\n", ev.At.Format("15:04:05"), ev.Kind, truncate(ev.Message, 104))
		}
	}

	fmt.Fprint(d.w, b.String())
}

func jobStatusLocked(job *dashboardJobState) string {
	if job == nil {
		return "running"
	}
	status := "running"
	if job.Ready {
		status = "inference ready"
	}
	if strings.TrimSpace(job.Phase) != "" {
		status = job.Phase
	}
	if job.EndedAt.IsZero() && !job.LastPhaseAt.IsZero() {
		status = fmt.Sprintf("%s (%s)", status, time.Since(job.LastPhaseAt).Truncate(time.Second))
	}
	return status
}

func (d *TerminalDashboard) activeJobsLocked() int {
	n := 0
	for _, job := range d.jobs {
		if job.EndedAt.IsZero() {
			n++
		}
	}
	return n
}

func (d *TerminalDashboard) currentJobLocked() *dashboardJobState {
	if d.activeJob != "" {
		if job := d.jobs[d.activeJob]; job != nil {
			return job
		}
	}
	var newest *dashboardJobState
	for _, job := range d.jobs {
		if newest == nil || job.StartedAt.After(newest.StartedAt) {
			newest = job
		}
	}
	return newest
}

func displayRepo(path string) string {
	if strings.TrimSpace(path) == "" {
		return "-"
	}
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) {
		return path
	}
	return base
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

type dashboardJobView struct {
	dash  *TerminalDashboard
	jobID string
}

func (v *dashboardJobView) withJob(fn func(*dashboardJobState)) {
	v.dash.mu.Lock()
	defer v.dash.mu.Unlock()
	job := v.dash.jobs[v.jobID]
	if job == nil {
		return
	}
	fn(job)
	v.dash.redrawLocked()
}

func (v *dashboardJobView) WriteHeader(role, model string, tools []string, handoff []string) {
	v.withJob(func(job *dashboardJobState) {
		job.Meta.Role = role
		job.Meta.Model = model
		job.Meta.Tools = tools
		job.Meta.Handoff = handoff
		job.Phase = "loaded tools"
		job.LastPhaseAt = time.Now()
		v.dash.appendEventLocked("job", fmt.Sprintf("%s loaded (%d tools)", role, len(tools)))
	})
}

func (v *dashboardJobView) WriteReady() {
	v.withJob(func(job *dashboardJobState) {
		job.Ready = true
		job.Phase = "inference ready"
		job.LastPhaseAt = time.Now()
		v.dash.appendEventLocked("model", fmt.Sprintf("%s inference ready", job.Meta.Role))
	})
}

func (v *dashboardJobView) WriteToolCall(name, args string) {
	v.withJob(func(job *dashboardJobState) {
		job.ToolCalls++
		job.LastTool = name
		job.Phase = "executing tool"
		job.LastPhaseAt = time.Now()
		v.dash.appendEventLocked("tool", fmt.Sprintf("%s called %s", job.Meta.Role, name))
	})
}

func (v *dashboardJobView) WriteToolResult(name, output string) {
	v.withJob(func(job *dashboardJobState) {
		job.LastOutput = truncate(output, 120)
		job.Phase = "tool result ready"
		job.LastPhaseAt = time.Now()
	})
}

func (v *dashboardJobView) WriteAssistant(content string) {
	v.withJob(func(job *dashboardJobState) {
		job.Summary = truncate(content, 140)
		job.Phase = "assistant replied"
		job.LastPhaseAt = time.Now()
		v.dash.appendEventLocked("reply", fmt.Sprintf("%s replied", job.Meta.Role))
	})
}

func (v *dashboardJobView) WriteTurn(turn, maxTurns int) {
	v.withJob(func(job *dashboardJobState) {
		job.Turn = turn
		job.MaxTurns = maxTurns
		job.Phase = "waiting for model response"
		job.LastPhaseAt = time.Now()
		v.dash.appendEventLocked("model", fmt.Sprintf("%s waiting for model response (turn %d/%d)", job.Meta.Role, turn, maxTurns))
	})
}

func (v *dashboardJobView) WriteError(msg string) {
	v.withJob(func(job *dashboardJobState) {
		job.LastError = msg
		job.Outcome = "blocked"
		job.Phase = "blocked"
		job.LastPhaseAt = time.Now()
		if job.EndedAt.IsZero() {
			job.EndedAt = time.Now()
		}
		v.dash.warnings = appendBounded(v.dash.warnings, dashboardEvent{At: time.Now(), Kind: "error", Message: msg}, 5)
		v.dash.appendEventLocked("error", fmt.Sprintf("%s: %s", job.Meta.Role, msg))
	})
}

func (v *dashboardJobView) WriteSummary(role, reason string, llmCalls, toolCalls int, duration time.Duration, tokens int) {
	v.withJob(func(job *dashboardJobState) {
		job.EndedAt = time.Now()
		job.Outcome = reason
		job.Phase = reason
		job.LastPhaseAt = job.EndedAt
		job.Summary = fmt.Sprintf("%s finished: %s, turns=%d, tools=%d, time=%s", role, reason, llmCalls, toolCalls, duration.Round(time.Millisecond))
		v.dash.appendEventLocked("done", job.Summary)
	})
}

func (v *dashboardJobView) WriteHandoff(from string, targets []string) {
	if len(targets) == 0 {
		return
	}
	v.withJob(func(job *dashboardJobState) {
		v.dash.appendEventLocked("handoff", fmt.Sprintf("%s -> %s", from, joinTargets(targets)))
	})
}
