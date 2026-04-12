package serve

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/dashboard"
	gh "github.com/greaveselliott/mars-harness/internal/github"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/power"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/safety"
	"github.com/greaveselliott/mars-harness/internal/scheduler"
)

// Config controls the serve command.
type Config struct {
	WebhookAddr   string
	WebhookSecret string
	DBPath        string
	Concurrency   int
	ModelsDir     string
	BinDir        string
	DashboardAddr string
}

func (c Config) concurrency() int {
	if c.Concurrency > 0 {
		return c.Concurrency
	}
	return 2
}

// Server composes all subsystems for the autonomous pipeline.
type Server struct {
	cfg    Config
	http   *http.Server
	mux    *http.ServeMux
	estop  *safety.EmergencyStop
	health atomic.Bool

	db        *sql.DB
	repos     *RepoRegistry
	triggers  *TriggerRouter
	queue     *queue.Queue
	workers   *queue.WorkerPool
	scheduler *scheduler.Scheduler
	router    *inference.Router
	executor  *Executor
	dash      *dashboard.Dashboard
	dashHTTP  *http.Server

	cancelSleep func()

	mu      sync.Mutex
	started bool
}

// New creates a Server wired with all subsystems.
func New(cfg Config) (*Server, error) {
	if cfg.WebhookAddr == "" {
		return nil, fmt.Errorf("serve: WebhookAddr is required — set it to e.g. \":9091\"")
	}
	if cfg.DBPath == "" {
		return nil, fmt.Errorf("serve: DBPath is required — set it to e.g. \"~/.mars-harness/db/mars.db\"")
	}

	db, err := sql.Open("sqlite", cfg.DBPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("serve: open database at %s: %w — check path and permissions", cfg.DBPath, err)
	}

	repos, err := NewRepoRegistry(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	jobQueue, err := queue.Open(cfg.DBPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("serve: open job queue: %w", err)
	}

	hw := hardware.Detect()
	modelSet := hardware.DefaultModels(hw.Profile)

	roleMapping := map[string]hardware.Tier{
		"engineer":       hardware.TierCoding,
		"pipeline-fixer": hardware.TierCoding,
		"reviewer":       hardware.TierReasoning,
		"code-reviewer":  hardware.TierReasoning,
		"qa":             hardware.TierCoding,
		"documenter":     hardware.TierFast,
		"docs-writer":    hardware.TierFast,
		"release":        hardware.TierFast,
		"release-manager": hardware.TierFast,
		"triager":        hardware.TierFast,
		"onboarder":      hardware.TierFast,
		"auditor":        hardware.TierReasoning,
		"security-auditor": hardware.TierReasoning,
		"backlog":        hardware.TierFast,
		"evolution":      hardware.TierReasoning,
		"dependency-updater": hardware.TierFast,
		"performance-optimizer": hardware.TierCoding,
		"refactorer":     hardware.TierCoding,
		"incident-responder": hardware.TierCoding,
	}

	binaryPath := filepath.Join(cfg.BinDir, "llama-server")
	router := inference.NewRouter(inference.RouterConfig{
		BinaryPath:  binaryPath,
		Models:      modelSet,
		RoleMapping: roleMapping,
		ModelsDir:   cfg.ModelsDir,
	})

	triggerRouter := NewTriggerRouter()

	repoLookup := func(ctx context.Context, repoID string) (string, error) {
		rec, err := repos.FindByID(ctx, repoID)
		if err != nil {
			return "", err
		}
		if rec == nil {
			return "", fmt.Errorf("repo %s not found in registry", repoID)
		}
		return rec.Path, nil
	}

	executor := NewExecutor(repoLookup, router, nil)

	sched := scheduler.New(jobQueue)

	s := &Server{
		cfg:       cfg,
		mux:       http.NewServeMux(),
		estop:     safety.NewEmergencyStop(),
		db:        db,
		repos:     repos,
		triggers:  triggerRouter,
		queue:     jobQueue,
		scheduler: sched,
		router:    router,
		executor:  executor,
	}

	s.workers = queue.NewWorkerPool(jobQueue, queue.WorkerConfig{
		Concurrency: cfg.concurrency(),
		OnJob:       executor.Execute,
		OnComplete:  s.handleJobComplete,
		OnFail:      s.handleJobFailed,
	})

	dashAddr := cfg.DashboardAddr
	if dashAddr == "" {
		dashAddr = ":9090"
	}
	dash, err := dashboard.New(dashboard.Config{
		Addr:          dashAddr,
		EmergencyStop: func() []error { return s.estop.Execute(context.Background()) },
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("serve: init dashboard: %w", err)
	}
	s.dash = dash
	s.dashHTTP = &http.Server{
		Addr:              dashAddr,
		Handler:           dash.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	executor.SetDashboard(dash)

	webhookHandler := gh.WebhookHandler(
		gh.WebhookConfig{Secret: cfg.WebhookSecret},
		s.handleEvent,
	)
	s.mux.Handle("/webhook", webhookHandler)
	s.mux.Handle("/healthz", s.HealthHandler())

	s.http = &http.Server{
		Addr:              cfg.WebhookAddr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.estop.Register(func(ctx context.Context) error {
		return s.Stop(ctx)
	})

	return s, nil
}

// Start begins all subsystems. Blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("serve: server already started")
	}
	s.started = true
	s.mu.Unlock()

	if cancelSleep, err := power.PreventSleep(); err != nil {
		slog.Warn("serve: could not prevent system sleep — long jobs may be interrupted if the machine idles", "err", err)
	} else {
		s.cancelSleep = cancelSleep
	}

	if n, err := s.queue.ResetOrphans(ctx); err != nil {
		slog.Warn("serve: failed to reset orphaned jobs", "err", err)
	} else if n > 0 {
		slog.Info("serve: reset orphaned jobs from previous run", "count", n)
	}

	repos, err := s.repos.List(ctx)
	if err != nil {
		return fmt.Errorf("serve: load repos: %w", err)
	}
	if err := s.triggers.Rebuild(repos); err != nil {
		slog.Warn("serve: trigger index rebuild had errors", "err", err)
	}
	slog.Info("serve: trigger index built", "repos", len(repos), "entries", s.triggers.Len())

	s.registerCronSchedules(repos)

	s.workers.Start(ctx)
	s.scheduler.Start(ctx)

	power.StartWatchdog(ctx, s.handleWake)

	ln, err := net.Listen("tcp", s.cfg.WebhookAddr)
	if err != nil {
		return fmt.Errorf("serve: failed to bind %s — check if the port is already in use: %w",
			s.cfg.WebhookAddr, err)
	}

	dashLn, err := net.Listen("tcp", s.dashHTTP.Addr)
	if err != nil {
		return fmt.Errorf("serve: failed to bind dashboard %s — check if the port is already in use: %w",
			s.dashHTTP.Addr, err)
	}

	s.health.Store(true)
	slog.Info("serve: orchestrator ready",
		"addr", ln.Addr().String(),
		"dashboard", "http://localhost"+s.dashHTTP.Addr,
		"concurrency", s.cfg.concurrency(),
		"repos", len(repos),
	)

	errCh := make(chan error, 2)
	go func() {
		if serveErr := s.http.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()
	go func() {
		if serveErr := s.dashHTTP.Serve(dashLn); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("serve: context cancelled, shutting down")
		return s.Stop(context.Background())
	case err := <-errCh:
		s.health.Store(false)
		return err
	}
}

// Stop gracefully shuts down all subsystems.
func (s *Server) Stop(ctx context.Context) error {
	s.health.Store(false)
	slog.Info("serve: stopping orchestrator")

	s.workers.Stop()
	s.scheduler.Stop()
	s.router.StopAll()

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var firstErr error
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		firstErr = fmt.Errorf("serve: HTTP shutdown: %w", err)
	}

	if s.dashHTTP != nil {
		if err := s.dashHTTP.Shutdown(shutdownCtx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("serve: dashboard shutdown: %w", err)
		}
	}

	if err := s.queue.Close(); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("serve: queue close: %w", err)
	}

	if err := s.db.Close(); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("serve: db close: %w", err)
	}

	if s.cancelSleep != nil {
		s.cancelSleep()
	}

	slog.Info("serve: orchestrator stopped")
	return firstErr
}

// handleWake is called by the sleep watchdog when the machine resumes
// from suspension. It restarts inference servers (stale connections) and
// resets any jobs stuck in running state so they get retried.
func (s *Server) handleWake(gap time.Duration) {
	slog.Warn("serve: recovering from system sleep", "gap", gap.Round(time.Second))

	s.router.RestartAll()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if n, err := s.queue.ResetOrphans(ctx); err != nil {
		slog.Error("serve: wake recovery — failed to reset orphaned jobs", "err", err)
	} else if n > 0 {
		slog.Info("serve: wake recovery — reset stuck jobs", "count", n)
	}

	if s.dash != nil {
		s.dash.BroadcastEvent("wake_recovery", fmt.Sprintf("resumed after %s sleep", gap.Round(time.Second)))
	}
}

// Healthy reports whether the server is accepting traffic.
func (s *Server) Healthy() bool {
	return s.health.Load()
}

// HealthHandler returns an http.Handler for /healthz.
func (s *Server) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Healthy() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"healthy"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"status":"unhealthy"}`)
	})
}

// Repos returns the registry for external use (e.g. CLI register command).
func (s *Server) Repos() *RepoRegistry { return s.repos }

// SeedJob enqueues a job directly. Used by the `start` command to inject the
// first agent (typically CEO) before the server loop begins.
func (s *Server) SeedJob(ctx context.Context, repoID, role, trigger string) (string, error) {
	idempotencyKey := fmt.Sprintf("seed:%s:%s:%d", repoID, role, time.Now().UnixNano())
	job := queue.Job{
		RepoID:         repoID,
		Role:           role,
		Trigger:        trigger,
		IdempotencyKey: idempotencyKey,
	}
	return s.queue.Enqueue(ctx, job)
}

// handleEvent is invoked by the webhook handler for each valid GitHub event.
// It matches the event against registered triggers and enqueues jobs.
func (s *Server) handleEvent(event gh.Event) {
	slog.Info("serve: received event",
		"type", event.Type,
		"action", event.Action,
		"repo", event.Repo,
		"delivery_id", event.ID)

	matches := s.triggers.Match(event)
	if len(matches) == 0 {
		slog.Debug("serve: no trigger matches for event", "type", event.Type, "action", event.Action)
		return
	}

	triggerJSON, _ := json.Marshal(map[string]string{
		"event_id": event.ID,
		"type":     event.Type,
		"action":   event.Action,
		"repo":     event.Repo,
	})

	for _, m := range matches {
		idempotencyKey := fmt.Sprintf("webhook:%s:%s:%s", event.ID, m.RepoID, m.Role)
		job := queue.Job{
			RepoID:         m.RepoID,
			Role:           m.Role,
			Trigger:        string(triggerJSON),
			IdempotencyKey: idempotencyKey,
		}

		jobID, err := s.queue.Enqueue(context.Background(), job)
		if err != nil {
			slog.Error("serve: failed to enqueue job",
				"role", m.Role,
				"repo", m.RepoID,
				"trigger", m.Trigger,
				"err", err,
			)
			continue
		}
		slog.Info("serve: job enqueued",
			"job_id", jobID,
			"role", m.Role,
			"repo_id", m.RepoID,
			"trigger", m.Trigger,
		)
	}
}

// handleJobComplete is the OnComplete callback for the worker pool.
// It resolves the `then` field from the completed role's manifest and
// enqueues follow-up jobs for each chained role. If a self-chain is
// skipped (role finished too fast, meaning no work was found), the
// `idle_then` roles are triggered instead — this is the bridge between
// the delivery loop and the strategy chain.
func (s *Server) handleJobComplete(ctx context.Context, job *queue.Job) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)

	rec, err := s.repos.FindByID(ctx, job.RepoID)
	if err != nil || rec == nil {
		log.Warn("serve: chain lookup failed — repo not found", "err", err)
		return
	}

	manifest, err := bundle.Load(rec.Path)
	if err != nil {
		log.Warn("serve: chain lookup failed — manifest load error", "err", err)
		return
	}

	role, ok := manifest.Roles[job.Role]
	if !ok || (len(role.Then) == 0 && len(role.IdleThen) == 0) {
		return
	}

	runDuration := time.Since(job.UpdatedAt)
	selfChainMinDuration := 60 * time.Second
	idle := false

	chainJSON, _ := json.Marshal(map[string]string{
		"type":        "chain",
		"source_role": job.Role,
		"source_job":  job.ID,
	})

	for _, target := range role.Then {
		if target == job.Role && runDuration < selfChainMinDuration {
			log.Info("serve: skipping self-chain — run too short, likely no work available",
				"duration", runDuration.Round(time.Second),
				"threshold", selfChainMinDuration,
			)
			idle = true
			continue
		}

		idempotencyKey := fmt.Sprintf("chain:%s:%s:%s", job.ID, job.RepoID, target)
		chainJob := queue.Job{
			RepoID:         job.RepoID,
			Role:           target,
			Trigger:        string(chainJSON),
			IdempotencyKey: idempotencyKey,
		}

		jobID, enqErr := s.queue.Enqueue(ctx, chainJob)
		if enqErr != nil {
			log.Error("serve: failed to enqueue chained job",
				"target_role", target,
				"err", enqErr,
			)
			continue
		}
		log.Info("serve: chained job enqueued",
			"target_role", target,
			"chained_job_id", jobID,
		)
	}

	if !idle || len(role.IdleThen) == 0 {
		return
	}

	idleJSON, _ := json.Marshal(map[string]string{
		"type":        "idle_trigger",
		"source_role": job.Role,
		"source_job":  job.ID,
		"reason":      "self-chain skipped — no work found in backlog",
	})

	log.Info("serve: role idle — triggering strategy chain", "idle_then", role.IdleThen)
	if s.dash != nil {
		s.dash.BroadcastEvent("idle_trigger", fmt.Sprintf("%s idle — seeding %v", job.Role, role.IdleThen))
	}

	for _, target := range role.IdleThen {
		idempotencyKey := fmt.Sprintf("idle:%s:%s:%s", job.RepoID, target, job.ID)
		idleJob := queue.Job{
			RepoID:         job.RepoID,
			Role:           target,
			Trigger:        string(idleJSON),
			IdempotencyKey: idempotencyKey,
		}

		jobID, enqErr := s.queue.Enqueue(ctx, idleJob)
		if enqErr != nil {
			log.Error("serve: failed to enqueue idle_then job",
				"target_role", target,
				"err", enqErr,
			)
			continue
		}
		log.Info("serve: idle_then job enqueued",
			"target_role", target,
			"idle_job_id", jobID,
		)
	}
}

// handleJobFailed is the OnFail callback for the worker pool.
// For self-chaining roles it re-enqueues the role so the pipeline recovers
// automatically from transient failures (e.g. inference timeout, OOM).
func (s *Server) handleJobFailed(ctx context.Context, job *queue.Job, jobErr error) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)

	rec, err := s.repos.FindByID(ctx, job.RepoID)
	if err != nil || rec == nil {
		return
	}

	manifest, err := bundle.Load(rec.Path)
	if err != nil {
		return
	}

	role, ok := manifest.Roles[job.Role]
	if !ok {
		return
	}

	selfChains := false
	for _, target := range role.Then {
		if target == job.Role {
			selfChains = true
			break
		}
	}

	if !selfChains {
		return
	}

	log.Info("serve: auto-recovering self-chaining role after failure",
		"error", jobErr,
	)

	retryJSON, _ := json.Marshal(map[string]string{
		"type":       "auto_recover",
		"source_job": job.ID,
		"reason":     jobErr.Error(),
	})

	idempotencyKey := fmt.Sprintf("recover:%s:%s:%d", job.RepoID, job.Role, time.Now().UnixNano())
	recoverJob := queue.Job{
		RepoID:         job.RepoID,
		Role:           job.Role,
		Trigger:        string(retryJSON),
		IdempotencyKey: idempotencyKey,
	}

	jobID, enqErr := s.queue.Enqueue(ctx, recoverJob)
	if enqErr != nil {
		log.Error("serve: failed to enqueue recovery job", "err", enqErr)
		return
	}
	log.Info("serve: recovery job enqueued", "recovery_job_id", jobID)
}

func (s *Server) registerCronSchedules(repos []RepoRecord) {
	for _, repo := range repos {
		manifest, err := bundle.Load(repo.Path)
		if err != nil {
			slog.Warn("serve: skipping schedules for repo — manifest load failed",
				"repo_id", repo.ID, "err", err)
			continue
		}

		for roleName, roleCfg := range manifest.Roles {
			cron := resolveSchedule(roleCfg.Schedule)

			if cron == "" {
				for _, trig := range roleCfg.Triggers {
					if c := cronFromTrigger(trig); c != "" {
						cron = c
						break
					}
				}
			}

			if cron == "" {
				continue
			}

			sched := scheduler.Schedule{
				Name:     fmt.Sprintf("%s:%s", repo.ID, roleName),
				RepoID:   repo.ID,
				Role:     roleName,
				Cron:     cron,
				Timezone: "UTC",
				Trigger:  fmt.Sprintf(`{"type":"schedule","role":"%s"}`, roleName),
			}
			if err := s.scheduler.Register(sched); err != nil {
				slog.Warn("serve: failed to register cron schedule",
					"repo", repo.ID,
					"role", roleName,
					"cron", cron,
					"err", err,
				)
			} else {
				slog.Info("serve: cron schedule registered",
					"repo", repo.ID,
					"role", roleName,
					"cron", cron,
				)
			}
		}
	}
}

var presetCron = map[string]string{
	"hourly":  "0 * * * *",
	"daily":   "0 6 * * *",
	"weekly":  "0 6 * * 1",
	"monthly": "0 6 1 * *",
}

func resolveSchedule(schedule string) string {
	s := strings.TrimSpace(schedule)
	if s == "" {
		return ""
	}
	if c, ok := presetCron[s]; ok {
		return c
	}
	return s
}

func cronFromTrigger(trigger string) string {
	if c, ok := presetCron[strings.TrimPrefix(trigger, "schedule.")]; ok && strings.HasPrefix(trigger, "schedule.") {
		return c
	}
	return ""
}
