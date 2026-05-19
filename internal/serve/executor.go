/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/dogfood-and-decisions.md
- docs/design-docs/guardrails.md
- docs/design-docs/dashboard.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-007-guardrails-and-safety.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/agent"
	"github.com/greaveselliott/mars-harness/internal/bundle"
	harctx "github.com/greaveselliott/mars-harness/internal/context"
	"github.com/greaveselliott/mars-harness/internal/dashboard"
	"github.com/greaveselliott/mars-harness/internal/guardrails"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/learnings"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/safety"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/greaveselliott/mars-harness/internal/ui"
)

const userMessage = "A trigger event has fired. Inspect the repository and execute your role. Trigger context is in the system prompt."

// RepoLookup resolves a repo ID to its local filesystem path.
type RepoLookup func(ctx context.Context, repoID string) (string, error)

// Executor runs agent jobs claimed from the queue.
type Executor struct {
	lookupRepo RepoLookup
	router     *inference.Router
	traceStore *trace.Store
	trustStore *trust.Store
	orgStore   *orgstate.Store
	dash       *dashboard.Dashboard
	onSignal   func(context.Context, interventionDebtSignal)
	jobViews   ui.JobViewFactory
}

// NewExecutor creates an executor bound to a repo lookup function and inference router.
// traceStore is optional; pass nil to disable trace persistence.
func NewExecutor(lookupRepo RepoLookup, router *inference.Router, traceStore *trace.Store, trustStore *trust.Store) *Executor {
	return &Executor{
		lookupRepo: lookupRepo,
		router:     router,
		traceStore: traceStore,
		trustStore: trustStore,
		jobViews:   ui.NewDebugJobViewFactory(os.Stdout, false, false),
	}
}

// SetDashboard wires the dashboard for SSE event broadcasting.
func (e *Executor) SetDashboard(d *dashboard.Dashboard) {
	e.dash = d
}

// SetOrgState wires the operational orchestration state store.
func (e *Executor) SetOrgState(store *orgstate.Store) {
	e.orgStore = store
}

// SetInterventionSignalHandler wires intervention-debt signal ingestion for
// tool-policy blocks observed inside an otherwise-running agent loop.
func (e *Executor) SetInterventionSignalHandler(handler func(context.Context, interventionDebtSignal)) {
	e.onSignal = handler
}

// SetJobViewFactory wires terminal job views for role execution.
func (e *Executor) SetJobViewFactory(factory ui.JobViewFactory) {
	if factory != nil {
		e.jobViews = factory
	}
}

func roleDefaultTrustLevel(role bundle.RoleConfig) trust.Level {
	if level, ok := trust.ParseLevel(role.TrustLevel); ok {
		return level
	}
	return trust.LevelObserver
}

// Execute is the OnJob callback for the worker pool.
// It loads the bundle, assembles context, starts inference, and runs the agent loop.
func (e *Executor) Execute(ctx context.Context, job *queue.Job) error {
	defer tools.KillBackgroundProcs()
	if job.Role == "dogfood" {
		defer cleanupDogfoodContainers(filepath.Base(job.RepoID))
	}

	log := slog.With("job_id", job.ID, "repo_id", job.RepoID, "role", job.Role)
	tw := e.jobViews.NewJobView(ui.JobViewMeta{
		JobID:  job.ID,
		RepoID: job.RepoID,
		Role:   job.Role,
	})

	repoPath, err := e.lookupRepo(ctx, job.RepoID)
	if err != nil {
		tw.WriteError(fmt.Sprintf("resolve repo %q: %v", job.RepoID, err))
		return fmt.Errorf("executor: resolve repo %q: %w — ensure the repo is registered via `mars-harness register`", job.RepoID, err)
	}

	manifest, err := bundle.Load(repoPath)
	if err != nil {
		tw.WriteError(fmt.Sprintf("load bundle: %v", err))
		return fmt.Errorf("executor: load bundle for repo %q: %w", job.RepoID, err)
	}

	role, ok := manifest.Roles[job.Role]
	if !ok {
		available := make([]string, 0, len(manifest.Roles))
		for name := range manifest.Roles {
			available = append(available, name)
		}
		tw.WriteError(fmt.Sprintf("role %q not found; available: %v", job.Role, available))
		return fmt.Errorf("executor: role %q not found in manifest for repo %q; available: %v", job.Role, job.RepoID, available)
	}

	handoff := manifest.DisplayHandoff(job.Role)
	tw.WriteHeader(job.Role, role.Model, role.Tools, handoff)

	started := time.Now()
	e.broadcastEvent("job_start", map[string]string{
		"job_id": job.ID,
		"role":   job.Role,
		"repo":   job.RepoID,
	})

	rolePrompt, err := manifest.RolePrompt(repoPath, job.Role)
	if err != nil {
		tw.WriteError(fmt.Sprintf("load role prompt: %v", err))
		return fmt.Errorf("executor: load role prompt: %w", err)
	}
	if manifest.DispatchMode() {
		rolePrompt = appendDispatchCompletionInstruction(rolePrompt)
	}

	guardRules, err := manifest.LoadGuardrails(repoPath, job.Role)
	if err != nil {
		tw.WriteError(fmt.Sprintf("load guardrails: %v", err))
		return fmt.Errorf("executor: load guardrails: %w", err)
	}
	guardEngine, err := guardrails.New(guardRules)
	if err != nil {
		tw.WriteError(fmt.Sprintf("compile guardrails: %v", err))
		return fmt.Errorf("executor: compile guardrails: %w", err)
	}
	knowledgeDefs, err := manifest.LoadKnowledgeRoutes(repoPath, job.Role)
	if err != nil {
		tw.WriteError(fmt.Sprintf("load knowledge routes: %v", err))
		return fmt.Errorf("executor: load knowledge routes: %w", err)
	}
	var knowledgeRoutes []harctx.KnowledgeRoute
	for _, kr := range knowledgeDefs {
		knowledgeRoutes = append(knowledgeRoutes, harctx.KnowledgeRoute{When: kr.When, Paths: kr.Paths})
	}
	var promptGuardrails []harctx.Guardrail
	for _, r := range guardRules {
		body := r.Message
		if body == "" {
			body = r.Pattern
		}
		promptGuardrails = append(promptGuardrails, harctx.Guardrail{Scope: r.Scope, Title: r.Name, Body: body})
	}

	var skills []harctx.Skill
	if skillDefs, sErr := bundle.LoadSkills(repoPath, job.Role); sErr != nil {
		log.Warn("executor: failed to load skills, continuing without", "err", sErr)
	} else {
		for _, sd := range skillDefs {
			skills = append(skills, harctx.Skill{Name: sd.Name, Scope: sd.Scope, Body: sd.Body})
		}
	}

	learnStore := learnings.NewStore(repoPath)
	learnData, lErr := learnStore.Load()
	if lErr != nil {
		log.Warn("executor: failed to load learnings, continuing without", "err", lErr)
		learnData = &learnings.Learnings{}
	}

	if learnData.Conventions.PackageManager == "" {
		conv := learnings.DetectConventions(repoPath)
		learnData.Conventions = conv
		if len(learnData.Excludes) == 0 {
			learnData.Excludes = learnings.DetectExcludes(repoPath)
		}
		if err := learnStore.Save(learnData); err != nil {
			log.Warn("executor: failed to save detected conventions", "err", err)
		}
	}

	var ticketIndex string
	switch job.Role {
	case "coo", "engineer", "janitor", "qa", "dogfood":
		ticketIndex = BuildTicketIndex(repoPath)
	}

	system, stats, err := harctx.Assemble(harctx.Input{
		RoleScope:       job.Role,
		RolePrompt:      rolePrompt,
		Guardrails:      promptGuardrails,
		KnowledgeRoutes: knowledgeRoutes,
		Skills:          skills,
		Trigger:         job.Trigger,
		PayloadMode:     job.PayloadMode,
		Learnings:       learnData.FormatForContext(),
		TicketIndex:     ticketIndex,
	})
	if err != nil {
		tw.WriteError(fmt.Sprintf("context assembly: %v", err))
		return fmt.Errorf("executor: assemble context: %w", err)
	}
	for _, s := range stats {
		log.Debug("executor: context section", "section", s.Name, "tokens", s.Tokens)
	}

	reg, err := tools.DefaultRegistry()
	if err != nil {
		tw.WriteError(fmt.Sprintf("tool registry: %v", err))
		return fmt.Errorf("executor: init tool registry: %w", err)
	}

	tools.RecordDecisionRole = job.Role

	trustLevel := roleDefaultTrustLevel(role)
	if e.trustStore != nil {
		entry, tErr := e.trustStore.Get(ctx, job.Role, job.RepoID)
		if tErr != nil {
			return fmt.Errorf("executor: load trust for %s/%s: %w", job.Role, job.RepoID, tErr)
		}
		if entry != nil {
			trustLevel = entry.Level
		}
	}

	rec := trace.NewRecorder(nil)
	toolExec := tools.NewExecutor(reg)
	terminalDispositionRecorded := false
	toolExec.StopAfterTool = func() bool {
		return terminalDispositionRecorded
	}
	toolExec.Session = &tools.Session{
		Role:         job.Role,
		JobID:        job.ID,
		RepoID:       job.RepoID,
		TrustLevel:   string(trustLevel),
		Guardrails:   guardEngine,
		SafetyLimits: safety.DefaultLimits(),
		PolicyRecorder: func(evt tools.PolicyEvent) {
			if e.onSignal == nil {
				return
			}
			category := telemetry.Classify(evt.Message)
			if category == telemetry.CategoryUnknown {
				category = telemetry.CategoryGuardrailBlock
			}
			e.onSignal(context.Background(), interventionDebtSignal{
				Kind:           interventionDebtSignalKindForCategory(category),
				RepoID:         job.RepoID,
				Role:           job.Role,
				JobID:          job.ID,
				Category:       category,
				EvidenceWindow: "24h",
				TraceID:        rec.TraceID(),
				ToolName:       evt.ToolName,
				Message:        fmt.Sprintf("%s tool policy blocked %s: %s", evt.Stage, evt.ToolName, evt.Message),
			})
		},
		DispositionRecorder: func(ctx context.Context, raw json.RawMessage) error {
			if e.orgStore == nil {
				return fmt.Errorf("orgstate store unavailable")
			}
			var d orgstate.Disposition
			if err := json.Unmarshal(raw, &d); err != nil {
				return fmt.Errorf("parse disposition: %w", err)
			}
			d.JobID = job.ID
			d.RepoID = job.RepoID
			d.Role = job.Role
			if d.TraceID == "" {
				d.TraceID = rec.TraceID()
			}
			if err := e.orgStore.RecordDisposition(ctx, d); err != nil {
				return err
			}
			terminalDispositionRecorded = true
			return nil
		},
	}

	root, err := tools.NewRoot(repoPath)
	if err != nil {
		tw.WriteError(fmt.Sprintf("sandbox root: %v", err))
		return fmt.Errorf("executor: create sandbox root for %q: %w", repoPath, err)
	}

	repair, err := tools.RepairWorkspaceHygieneIgnorePolicy(ctx, root)
	if err != nil {
		log.Warn("executor: workspace hygiene ignore auto-repair failed", "err", err)
	} else if repair.Committed {
		log.Info("executor: workspace hygiene ignore auto-repaired", "commit", repair.Commit, "missing_ignores", repair.MissingIgnores)
	}

	hygiene, err := tools.AuditWorkspaceHygiene(ctx, root, tools.WorkspaceHygieneOptions{Mode: "pre_job"})
	if err != nil {
		tw.WriteError(fmt.Sprintf("workspace hygiene: %v", err))
		return fmt.Errorf("executor: workspace hygiene pre-job check failed before role %q run: %w", job.Role, err)
	}
	if hygiene.Blocking {
		tw.WriteError(fmt.Sprintf("workspace hygiene: %s", hygiene.Message))
		return fmt.Errorf("executor: workspace_hygiene_blocked before role %q run: %s — next action: %s", job.Role, hygiene.Message, hygiene.NextAction)
	}

	if err := tools.ValidateRepoDiff(ctx, root, tools.Session{
		Role:         job.Role,
		JobID:        job.ID,
		RepoID:       job.RepoID,
		TrustLevel:   string(trustLevel),
		SafetyLimits: safety.DefaultLimits(),
	}); err != nil {
		tw.WriteError(fmt.Sprintf("dirty worktree containment: %v", err))
		return fmt.Errorf("executor: dirty worktree containment before role %q run: %w", job.Role, err)
	}

	endpoint, err := e.router.ServerForRoleModel(ctx, job.Role, role.Model)
	if err != nil {
		tw.WriteError(fmt.Sprintf("inference for %q: %v", job.Role, err))
		return fmt.Errorf("executor: get inference endpoint for role %q: %w", job.Role, err)
	}
	tw.WriteReady()

	client, err := llm.NewClient(llm.Config{
		BaseURL: endpoint,
		Model:   role.Model,
	})
	if err != nil {
		tw.WriteError(fmt.Sprintf("LLM client: %v", err))
		return fmt.Errorf("executor: create LLM client: %w", err)
	}

	allowlist := role.Tools
	if len(allowlist) == 0 {
		return fmt.Errorf("executor: role %q has no tools configured; strict trunk requires an explicit tools allowlist in .harness/manifest.yaml", job.Role)
	}

	var beforeTickets ticketSnapshot
	if job.Role == "engineer" {
		beforeTickets, err = snapshotTickets(repoPath)
		if err != nil {
			tw.WriteError(fmt.Sprintf("snapshot tickets before run: %v", err))
			return fmt.Errorf("executor: snapshot tickets before engineer run: %w", err)
		}
	}

	params := agent.Params{
		Completer:    client,
		Registry:     reg,
		Executor:     toolExec,
		Root:         root,
		Allowlist:    allowlist,
		SystemPrompt: system,
		UserMessage:  userMessage,
		Config: agent.LoopConfig{
			Model:       role.Model,
			MaxTurns:    role.MaxTurns,
			ContextSize: role.ContextSize,
		},
		JobID:      job.ID,
		Trace:      rec,
		TraceStore: e.traceStore,
		UI:         tw,
	}
	if manifest.DispatchMode() {
		params.RequiredTerminalTool = "job_disposition_record"
	}

	res, err := agent.Run(ctx, params)
	if err != nil {
		tw.WriteError(fmt.Sprintf("agent run: %v", err))
		return fmt.Errorf("executor: agent run failed: %w", err)
	}

	tw.WriteSummary(
		job.Role,
		string(res.EndReason),
		res.LLMCalls,
		res.ToolInvocations,
		res.WallTime,
		res.TokenEstimate,
	)

	outcome := "success"
	if res.Err != nil {
		outcome = "error"
		tw.WriteError(fmt.Sprintf("agent loop error (%s): %v", res.EndReason, res.Err))
	}
	if !agent.SuccessfulEnd(res.EndReason) {
		outcome = "error"
		tw.WriteError(fmt.Sprintf("agent loop ended without success: %s", res.EndReason))
	}

	e.broadcastEvent("job_complete", map[string]string{
		"job_id":   job.ID,
		"role":     job.Role,
		"repo":     job.RepoID,
		"outcome":  outcome,
		"duration": time.Since(started).Round(time.Millisecond).String(),
	})

	if res.Err != nil {
		learnings.RecordJobLessons(learnStore, job.Role, res.Err.Error(), "", nil)
		if _, _, commitErr := commitRuntimeLearningsIfOnlyDirty(ctx, repoPath, job.Role); commitErr != nil {
			log.Warn("executor: runtime learnings auto-commit failed", "err", commitErr)
		}
		return fmt.Errorf("executor: agent loop error (%s): %w", res.EndReason, res.Err)
	}
	if err := agent.NonSuccessError(res); err != nil {
		learnings.RecordJobLessons(learnStore, job.Role, err.Error(), "", nil)
		if _, _, commitErr := commitRuntimeLearningsIfOnlyDirty(ctx, repoPath, job.Role); commitErr != nil {
			log.Warn("executor: runtime learnings auto-commit failed", "err", commitErr)
		}
		return fmt.Errorf("executor: %w", err)
	}

	if job.Role == "engineer" {
		afterTickets, sErr := snapshotTickets(repoPath)
		if sErr != nil {
			tw.WriteError(fmt.Sprintf("snapshot tickets after run: %v", sErr))
			return fmt.Errorf("executor: snapshot tickets after engineer run: %w", sErr)
		}
		if gateErr := validateEngineerTicketGateWithEvidence(repoPath, beforeTickets, afterTickets); gateErr != nil {
			tw.WriteError(gateErr.Error())
			learnings.RecordJobLessons(learnStore, job.Role, gateErr.Error(), "", nil)
			if _, _, commitErr := commitRuntimeLearningsIfOnlyDirty(ctx, repoPath, job.Role); commitErr != nil {
				log.Warn("executor: runtime learnings auto-commit failed", "err", commitErr)
			}
			return fmt.Errorf("executor: %w", gateErr)
		}
	}

	if manifest.DispatchMode() {
		if e.orgStore == nil {
			return fmt.Errorf("executor: dispatch mode requires orgstate store")
		}
		disposition, dErr := e.orgStore.GetDisposition(ctx, job.ID)
		if dErr != nil {
			return fmt.Errorf("executor: load dispatch disposition: %w", dErr)
		}
		if disposition == nil {
			return fmt.Errorf("executor: dispatch mode requires %s to call job_disposition_record before completing", job.Role)
		}
	}

	learnings.RecordJobLessons(learnStore, job.Role, "", "", nil)
	if committed, commit, commitErr := commitRuntimeLearningsIfOnlyDirty(ctx, repoPath, job.Role); commitErr != nil {
		log.Warn("executor: runtime learnings auto-commit failed", "err", commitErr)
	} else if committed {
		log.Info("executor: committed runtime learnings", "commit", commit, "role", job.Role)
	}

	if manifest.DispatchMode() && job.Role != "orchestrator" {
		e.broadcastEvent("dispatch_return", map[string]string{
			"from": job.Role,
			"to":   "orchestrator",
			"repo": job.RepoID,
		})
	} else if len(role.Then) > 0 {
		e.broadcastEvent("chain", map[string]string{
			"from": job.Role,
			"to":   strings.Join(role.Then, ","),
			"repo": job.RepoID,
		})
	}

	tw.WriteHandoff(job.Role, handoff)

	return nil
}

func appendDispatchCompletionInstruction(rolePrompt string) string {
	return strings.TrimSpace(rolePrompt) + `

## Dispatch Completion

Before finishing this autonomous server job, call job_disposition_record exactly
once. Set status, next_need, suggested_role when you have a concrete suggestion,
ticket_id when applicable, reason, and evidence_links. Include handoff when
another role should continue with a concrete ask, and feedback when a previous
role must correct or clarify work. The Orchestrator receives this structured
disposition as source_disposition, translates it into the next role handoff,
and decides the next best role; do not assume a fixed linear handoff.`
}

func (e *Executor) broadcastEvent(eventType string, payload map[string]string) {
	if e.dash == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("executor: marshal SSE payload", "error", err)
		return
	}
	e.dash.BroadcastEvent(eventType, string(data))
}

func commitRuntimeLearningsIfOnlyDirty(ctx context.Context, repoPath, role string) (bool, string, error) {
	const learningsPath = ".harness/learnings.yaml"
	status, err := runGitCommand(ctx, repoPath, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return false, "", fmt.Errorf("status: %w", err)
	}
	lines := nonEmptyLines(status)
	if len(lines) == 0 {
		return false, "", nil
	}
	for _, line := range lines {
		path := porcelainPath(line)
		if path != learningsPath {
			return false, "", nil
		}
	}
	if _, err := runGitCommand(ctx, repoPath, "add", learningsPath); err != nil {
		return false, "", fmt.Errorf("stage %s: %w", learningsPath, err)
	}
	if _, err := runGitCommand(ctx, repoPath, "diff", "--cached", "--quiet", "--", learningsPath); err == nil {
		return false, "", nil
	}
	message := fmt.Sprintf("chore(learnings): update runtime learnings for %s", strings.TrimSpace(role))
	if strings.TrimSpace(role) == "" {
		message = "chore(learnings): update runtime learnings"
	}
	if _, err := runGitCommand(ctx, repoPath, "commit", "-m", message, "--", learningsPath); err != nil {
		return false, "", fmt.Errorf("commit %s: %w", learningsPath, err)
	}
	commit, err := runGitCommand(ctx, repoPath, "rev-parse", "--short", "HEAD")
	if err != nil {
		return true, "", nil
	}
	return true, strings.TrimSpace(commit), nil
}

func runGitCommand(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%v: %w\n%s", args, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func porcelainPath(line string) string {
	if len(line) < 4 {
		return strings.TrimSpace(line)
	}
	path := strings.TrimSpace(line[3:])
	if idx := strings.Index(path, " -> "); idx >= 0 {
		path = strings.TrimSpace(path[idx+4:])
	}
	return path
}

// BuildTicketIndex scans docs/tickets/ and returns a compact inventory for context injection.
func BuildTicketIndex(repoPath string) string {
	all, err := ticketstate.List(repoPath)
	if err != nil || len(all) == 0 {
		return "No existing tickets found in docs/tickets/."
	}
	var inProgressInterventionEligible []string
	var inProgressEligible []string
	var inProgressBlocked []string
	var inReview []string
	var backlogInterventionPreemptive []string
	var deferredInterventionCount int
	var backlog []string
	var done []string
	for _, t := range all {
		line := ticketIndexLine(t)
		switch t.Status {
		case ticketstate.StatusInProgress:
			if t.Blocked() {
				inProgressBlocked = append(inProgressBlocked, line)
			} else if t.Kind == "intervention-debt" {
				inProgressInterventionEligible = append(inProgressInterventionEligible, line)
			} else {
				inProgressEligible = append(inProgressEligible, line)
			}
		case ticketstate.StatusBacklog:
			if t.Kind == "intervention-debt" {
				if interventionDebtPreemptsBacklog(t) {
					backlogInterventionPreemptive = append(backlogInterventionPreemptive, line)
				} else {
					deferredInterventionCount++
				}
			} else {
				backlog = append(backlog, line)
			}
		case ticketstate.StatusInReview:
			inReview = append(inReview, line)
		case ticketstate.StatusDone:
			done = append(done, line)
		}
	}
	var lines []string
	header := fmt.Sprintf("Existing tickets (%d total). Eligible product in-progress tickets are the Engineer front of queue; intervention-debt tickets stay visible in quality/status evidence but do not preempt ordinary product backlog unless an active product ticket names them in blocked_by (%d hidden). Complete the lowest-numbered eligible product ticket before claiming backlog work. Blocked in-progress tickets must name blocker, blocked_by, trace_id, and next_action metadata and do not block backlog work.\n", len(all), deferredInterventionCount)
	lines = append(lines, inProgressEligible...)
	lines = append(lines, backlog...)
	lines = append(lines, inReview...)
	lines = append(lines, inProgressBlocked...)
	lines = append(lines, inProgressInterventionEligible...)
	lines = append(lines, backlogInterventionPreemptive...)
	lines = append(lines, done...)
	return header + strings.Join(lines, "\n")
}

func interventionDebtPreemptsBacklog(t ticketstate.Ticket) bool {
	return false
}

func ticketIndexLine(t ticketstate.Ticket) string {
	labels := []string{t.Status}
	if t.Kind == "intervention-debt" {
		labels = append(labels, "intervention-debt")
	}
	if t.Blocked() {
		labels = append(labels, "blocked")
	}
	line := fmt.Sprintf("- [%s] %s", strings.Join(labels, "]["), t.Name)
	if rel := strings.TrimSpace(t.RelPath); rel != "" {
		line += fmt.Sprintf(" (path: %s)", rel)
	}
	if t.Blocked() && strings.TrimSpace(t.NextAction) != "" && !strings.EqualFold(strings.TrimSpace(t.NextAction), "TBD") {
		line += fmt.Sprintf(" — next: %s", t.NextAction)
	}
	return line
}

// cleanupDogfoodContainers removes any orphaned Podman containers from dogfood runs.
func cleanupDogfoodContainers(projectName string) {
	name := "dogfood-" + projectName
	if _, err := exec.LookPath("podman"); err != nil {
		return
	}
	out, err := exec.Command("podman", "rm", "-f", name).CombinedOutput()
	if err != nil {
		slog.Debug("executor: dogfood container cleanup (may not exist)", "name", name, "output", string(out))
		return
	}
	slog.Info("executor: cleaned up dogfood container", "name", name)
}
