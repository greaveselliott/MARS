package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/agent"
	"github.com/greaveselliott/mars-harness/internal/bundle"
	harctx "github.com/greaveselliott/mars-harness/internal/context"
	"github.com/greaveselliott/mars-harness/internal/dashboard"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
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
	dash       *dashboard.Dashboard
}

// NewExecutor creates an executor bound to a repo lookup function and inference router.
// traceStore is optional; pass nil to disable trace persistence.
func NewExecutor(lookupRepo RepoLookup, router *inference.Router, traceStore *trace.Store) *Executor {
	return &Executor{
		lookupRepo: lookupRepo,
		router:     router,
		traceStore: traceStore,
	}
}

// SetDashboard wires the dashboard for SSE event broadcasting.
func (e *Executor) SetDashboard(d *dashboard.Dashboard) {
	e.dash = d
}

// Execute is the OnJob callback for the worker pool.
// It loads the bundle, assembles context, starts inference, and runs the agent loop.
func (e *Executor) Execute(ctx context.Context, job *queue.Job) error {
	defer tools.KillBackgroundProcs()

	log := slog.With("job_id", job.ID, "repo_id", job.RepoID, "role", job.Role)
	tw := ui.NewTraceWriter(os.Stdout, false, false)

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

	tw.WriteHeader(job.Role, role.Model, role.Tools, role.Then)

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

	var skills []harctx.Skill
	if skillDefs, sErr := bundle.LoadSkills(repoPath, job.Role); sErr != nil {
		log.Warn("executor: failed to load skills, continuing without", "err", sErr)
	} else {
		for _, sd := range skillDefs {
			skills = append(skills, harctx.Skill{Name: sd.Name, Scope: sd.Scope, Body: sd.Body})
		}
	}

	system, stats, err := harctx.Assemble(harctx.Input{
		RoleScope:  job.Role,
		RolePrompt: rolePrompt,
		Skills:     skills,
		Trigger:    job.Trigger,
	})
	if err != nil {
		tw.WriteError(fmt.Sprintf("context assembly: %v", err))
		return fmt.Errorf("executor: assemble context: %w", err)
	}
	for _, s := range stats {
		log.Debug("executor: context section", "section", s.Name, "tokens", s.Tokens)
	}

	endpoint, err := e.router.ServerForRole(ctx, job.Role)
	if err != nil {
		tw.WriteError(fmt.Sprintf("inference for %q: %v", job.Role, err))
		return fmt.Errorf("executor: get inference endpoint for role %q: %w — check GPU availability or configure a remote fallback", job.Role, err)
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

	reg, err := tools.DefaultRegistry()
	if err != nil {
		tw.WriteError(fmt.Sprintf("tool registry: %v", err))
		return fmt.Errorf("executor: init tool registry: %w", err)
	}

	toolExec := tools.NewExecutor(reg)

	root, err := tools.NewRoot(repoPath)
	if err != nil {
		tw.WriteError(fmt.Sprintf("sandbox root: %v", err))
		return fmt.Errorf("executor: create sandbox root for %q: %w", repoPath, err)
	}

	allowlist := role.Tools
	if len(allowlist) == 0 {
		allowlist = reg.Names()
	}

	rec := trace.NewRecorder(nil)

	params := agent.Params{
		Completer:    client,
		Registry:     reg,
		Executor:     toolExec,
		Root:         root,
		Allowlist:    allowlist,
		SystemPrompt: system,
		UserMessage:  userMessage,
		Config: agent.LoopConfig{
			Model:    role.Model,
			MaxTurns: role.MaxTurns,
		},
		JobID:      job.ID,
		Trace:      rec,
		TraceStore: e.traceStore,
		UI:         tw,
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

	e.broadcastEvent("job_complete", map[string]string{
		"job_id":   job.ID,
		"role":     job.Role,
		"repo":     job.RepoID,
		"outcome":  outcome,
		"duration": time.Since(started).Round(time.Millisecond).String(),
	})

	if res.Err != nil {
		return fmt.Errorf("executor: agent loop error (%s): %w", res.EndReason, res.Err)
	}

	if len(role.Then) > 0 {
		e.broadcastEvent("chain", map[string]string{
			"from": job.Role,
			"to":   strings.Join(role.Then, ","),
			"repo": job.RepoID,
		})
	}

	tw.WriteHandoff(job.Role, role.Then)

	return nil
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
