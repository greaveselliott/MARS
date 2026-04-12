package serve

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/greaveselliott/mars-harness/internal/agent"
	"github.com/greaveselliott/mars-harness/internal/bundle"
	harctx "github.com/greaveselliott/mars-harness/internal/context"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/trace"
)

const userMessage = "A trigger event has fired. Inspect the repository and execute your role. Trigger context is in the system prompt."

// RepoLookup resolves a repo ID to its local filesystem path.
type RepoLookup func(ctx context.Context, repoID string) (string, error)

// Executor runs agent jobs claimed from the queue.
type Executor struct {
	lookupRepo RepoLookup
	router     *inference.Router
	traceStore *trace.Store
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

// Execute is the OnJob callback for the worker pool.
// It loads the bundle, assembles context, starts inference, and runs the agent loop.
func (e *Executor) Execute(ctx context.Context, job *queue.Job) error {
	log := slog.With("job_id", job.ID, "repo_id", job.RepoID, "role", job.Role)
	log.Info("executor: starting job")

	repoPath, err := e.lookupRepo(ctx, job.RepoID)
	if err != nil {
		return fmt.Errorf("executor: resolve repo %q: %w — ensure the repo is registered via `mars-harness repo add`", job.RepoID, err)
	}

	manifest, err := bundle.Load(repoPath)
	if err != nil {
		return fmt.Errorf("executor: load bundle for repo %q: %w", job.RepoID, err)
	}

	role, ok := manifest.Roles[job.Role]
	if !ok {
		available := make([]string, 0, len(manifest.Roles))
		for name := range manifest.Roles {
			available = append(available, name)
		}
		return fmt.Errorf("executor: role %q not found in manifest for repo %q; available: %v", job.Role, job.RepoID, available)
	}

	rolePrompt, err := manifest.RolePrompt(repoPath, job.Role)
	if err != nil {
		return fmt.Errorf("executor: load role prompt: %w", err)
	}

	system, stats, err := harctx.Assemble(harctx.Input{
		RoleScope:  job.Role,
		RolePrompt: rolePrompt,
		Trigger:    job.Trigger,
	})
	if err != nil {
		return fmt.Errorf("executor: assemble context: %w", err)
	}
	for _, s := range stats {
		log.Debug("executor: context section", "section", s.Name, "tokens", s.Tokens)
	}

	endpoint, err := e.router.ServerForRole(ctx, job.Role)
	if err != nil {
		return fmt.Errorf("executor: get inference endpoint for role %q: %w — check GPU availability or configure a remote fallback", job.Role, err)
	}
	log.Info("executor: inference endpoint resolved", "endpoint", endpoint, "model", role.Model)

	client, err := llm.NewClient(llm.Config{
		BaseURL: endpoint,
		Model:   role.Model,
	})
	if err != nil {
		return fmt.Errorf("executor: create LLM client: %w", err)
	}

	reg, err := tools.DefaultRegistry()
	if err != nil {
		return fmt.Errorf("executor: init tool registry: %w", err)
	}

	toolExec := tools.NewExecutor(reg)

	root, err := tools.NewRoot(repoPath)
	if err != nil {
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
			Model: role.Model,
		},
		JobID:      job.ID,
		Trace:      rec,
		TraceStore: e.traceStore,
	}

	res, err := agent.Run(ctx, params)
	if err != nil {
		return fmt.Errorf("executor: agent run failed: %w", err)
	}

	log.Info("executor: job finished",
		"end_reason", string(res.EndReason),
		"llm_calls", res.LLMCalls,
		"tool_invocations", res.ToolInvocations,
		"tokens", res.TokenEstimate,
		"wall_ms", res.WallTime.Milliseconds(),
	)

	if res.Err != nil {
		return fmt.Errorf("executor: agent loop error (%s): %w", res.EndReason, res.Err)
	}

	return nil
}
