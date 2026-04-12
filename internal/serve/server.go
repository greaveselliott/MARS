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
	"github.com/greaveselliott/mars-harness/internal/evolution"
	gh "github.com/greaveselliott/mars-harness/internal/github"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/power"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/safety"
	"github.com/greaveselliott/mars-harness/internal/scheduler"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/greaveselliott/mars-harness/internal/trace"
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
	telemetry   *telemetry.Collector
	telemStore  *telemetry.Store
	traceStore  *trace.Store
	scoreStore  *scoring.Store
	evoStore    *evolution.Store

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
		"janitor":        hardware.TierFast,
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

	traceStore, err := trace.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: trace store unavailable — traces will not be persisted", "err", err)
	}

	scoreStore, err := scoring.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: scoring store unavailable — outcomes will not be recorded", "err", err)
	}

	evoStore, err := evolution.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: evolution store unavailable — evolution tracking disabled", "err", err)
	}

	executor := NewExecutor(repoLookup, router, traceStore)

	sched := scheduler.New(jobQueue)

	telemStore, err := telemetry.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: telemetry store unavailable — events will not persist", "err", err)
	}

	telem := telemetry.NewCollector(nil, telemStore)

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
		executor:   executor,
		telemetry:  telem,
		telemStore: telemStore,
		traceStore: traceStore,
		scoreStore: scoreStore,
		evoStore:   evoStore,
	}

	telem.SetRemediator(s.handleRemediation)

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
	telem.SetDashboard(dash)

	dash.HandleFunc("/api/telemetry", s.handleTelemetryAPI)
	dash.HandleFunc("/api/evolution", s.handleEvolutionAPI)
	dash.HandleFunc("/api/roles", s.handleRolesAPI)

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

	if s.telemStore != nil {
		_ = s.telemStore.Close()
	}
	if s.traceStore != nil {
		_ = s.traceStore.Close()
	}
	if s.scoreStore != nil {
		_ = s.scoreStore.Close()
	}
	if s.evoStore != nil {
		_ = s.evoStore.Close()
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

// handleTelemetryAPI serves the telemetry event history as JSON.
func (s *Server) handleTelemetryAPI(w http.ResponseWriter, r *http.Request) {
	type apiResponse struct {
		Events []telemetry.Event          `json:"events"`
		Stats  map[telemetry.FailureCategory]int `json:"stats"`
	}
	resp := apiResponse{
		Events: s.telemetry.Events(),
		Stats:  s.telemetry.Stats(),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("serve: telemetry API encode error", "err", err)
	}
}

func (s *Server) handleEvolutionAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type evoEvent struct {
		ID             string  `json:"id"`
		Role           string  `json:"role"`
		RepoID         string  `json:"repo_id"`
		Classification string  `json:"classification"`
		Suggestion     string  `json:"suggestion"`
		ScoreBefore    float64 `json:"score_before"`
		ScoreAfter     float64 `json:"score_after"`
		CreatedAt      string  `json:"created_at"`
	}

	type apiResponse struct {
		Evolutions []evoEvent          `json:"evolutions"`
		Telemetry  []telemetry.Event   `json:"telemetry"`
		Patterns   []telemetry.Pattern `json:"patterns"`
	}

	resp := apiResponse{
		Telemetry: s.telemetry.Events(),
		Patterns:  s.telemetry.DetectPatterns(),
	}

	if s.evoStore != nil {
		roles := []string{"ceo", "cto-weekly", "coo", "engineer", "qa", "janitor"}
		for _, role := range roles {
			evos, err := s.evoStore.GetEvolutions(ctx, role, 10)
			if err != nil {
				continue
			}
			for _, ev := range evos {
				var result evolution.ReviewResult
				_ = json.Unmarshal([]byte(ev.Result), &result)
				resp.Evolutions = append(resp.Evolutions, evoEvent{
					ID:             ev.ID,
					Role:           ev.Role,
					RepoID:         ev.RepoID,
					Classification: result.Classification,
					Suggestion:     result.Suggestion,
					ScoreBefore:    ev.ScoreBefore,
					ScoreAfter:     ev.ScoreAfter,
					CreatedAt:      ev.CreatedAt.Format(time.RFC3339),
				})
			}
		}
	}

	if len(resp.Telemetry) > 50 {
		resp.Telemetry = resp.Telemetry[len(resp.Telemetry)-50:]
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("serve: evolution API encode error", "err", err)
	}
}

func (s *Server) handleRolesAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type roleInfo struct {
		Role         string  `json:"role"`
		Score        float64 `json:"score"`
		SampleSize   int     `json:"sample_size"`
		SuccessCount int     `json:"success_count"`
		FailCount    int     `json:"fail_count"`
	}

	type apiResponse struct {
		Roles []roleInfo `json:"roles"`
	}

	roles := []string{"ceo", "cto-weekly", "cto-pr-merge", "coo", "engineer", "qa",
		"security-pr", "security-weekly", "dependency-manager", "release-pr",
		"release-weekly", "dogfood", "pipeline-fixer", "pr-comment-fixer", "janitor"}

	var resp apiResponse

	for _, role := range roles {
		info := roleInfo{Role: role}

		if s.scoreStore != nil {
			sc, err := s.scoreStore.GetScore(ctx, role, "")
			if err == nil && sc != nil {
				info.Score = sc.Value
				info.SampleSize = sc.SampleSize
			}
		}

		telemEvents := s.telemetry.Events()
		for _, evt := range telemEvents {
			if evt.Role == role {
				info.FailCount++
			}
		}
		info.SuccessCount = info.SampleSize - info.FailCount
		if info.SuccessCount < 0 {
			info.SuccessCount = 0
		}

		if info.SampleSize > 0 || info.FailCount > 0 {
			resp.Roles = append(resp.Roles, info)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("serve: roles API encode error", "err", err)
	}
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

	if s.scoreStore != nil {
		_ = s.scoreStore.RecordOutcome(ctx, scoring.Outcome{
			JobID:  job.ID,
			RepoID: job.RepoID,
			Role:   job.Role,
			Type:   scoring.OutcomePassed,
		})
	}

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

	go s.checkEvolution(context.Background(), job.Role, job.RepoID)
}

// handleJobFailed is the OnFail callback for the worker pool.
// It records a telemetry event (which triggers classification and
// remediation), then falls back to the existing self-chain recovery
// for roles that don't match a specific remediation action.
func (s *Server) handleJobFailed(ctx context.Context, job *queue.Job, jobErr error) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)

	if s.scoreStore != nil {
		cat := telemetry.Classify(jobErr.Error())
		outcomeType := scoring.OutcomeFailed
		switch cat {
		case telemetry.CategoryMaxTurns:
			outcomeType = scoring.OutcomeNoop
		case telemetry.CategoryCircleDetected:
			outcomeType = scoring.OutcomeNoop
		case telemetry.CategoryToolTimeout, telemetry.CategoryContextOverflow:
			outcomeType = scoring.OutcomeTimeout
		}
		_ = s.scoreStore.RecordOutcome(ctx, scoring.Outcome{
			JobID:   job.ID,
			RepoID:  job.RepoID,
			Role:    job.Role,
			Type:    outcomeType,
			Details: jobErr.Error(),
		})
	}

	s.telemetry.Record(job.ID, job.RepoID, job.Role, jobErr.Error())

	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"job_id":   job.ID,
			"role":     job.Role,
			"repo":     job.RepoID,
			"error":    jobErr.Error(),
			"category": string(telemetry.Classify(jobErr.Error())),
		})
		s.dash.BroadcastEvent("job_failed", string(payload))
	}

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

	go s.checkEvolution(context.Background(), job.Role, job.RepoID)
}

// handleRemediation is the telemetry remediation callback. It executes
// the auto-fix action determined by the classifier.
func (s *Server) handleRemediation(evt telemetry.Event) {
	log := slog.With("event_id", evt.ID, "job_id", evt.JobID, "role", evt.Role, "action", evt.Action)

	switch telemetry.RemediationAction(evt.Action) {
	case telemetry.ActionRestartInference:
		log.Info("telemetry: restarting inference servers")
		s.router.RestartAll()

	case telemetry.ActionRetryHalfContext:
		log.Info("telemetry: retrying job with halved context")
		s.enqueueRetry(evt, "half_context")

	case telemetry.ActionRetryLonger:
		log.Info("telemetry: retrying job with longer timeout")
		s.enqueueRetry(evt, "longer_timeout")

	case telemetry.ActionRetryPlain:
		log.Info("telemetry: retrying job")
		s.enqueueRetry(evt, "retry")
	}
}

func (s *Server) enqueueRetry(evt telemetry.Event, reason string) {
	triggerJSON, _ := json.Marshal(map[string]string{
		"type":       "telemetry_retry",
		"source_job": evt.JobID,
		"reason":     reason,
		"category":   string(evt.Category),
	})

	idempotencyKey := fmt.Sprintf("telem:%s:%s:%s", evt.RepoID, evt.Role, evt.ID)
	retryJob := queue.Job{
		RepoID:         evt.RepoID,
		Role:           evt.Role,
		Trigger:        string(triggerJSON),
		IdempotencyKey: idempotencyKey,
	}

	ctx := context.Background()
	jobID, err := s.queue.Enqueue(ctx, retryJob)
	if err != nil {
		slog.Error("telemetry: failed to enqueue retry job", "event_id", evt.ID, "err", err)
		return
	}
	slog.Info("telemetry: retry job enqueued", "event_id", evt.ID, "retry_job_id", jobID)

	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"event_id":     evt.ID,
			"retry_job_id": jobID,
			"role":         evt.Role,
			"action":       reason,
		})
		s.dash.BroadcastEvent("telemetry_retry", string(payload))
	}
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

// jobCount tracks how many jobs have completed since the last evolution check.
var jobCount int32

// checkEvolution runs pattern detection and triggers evolution reviews
// when recurring failures are detected or after enough jobs accumulate.
func (s *Server) checkEvolution(ctx context.Context, role, repoID string) {
	if s.evoStore == nil || s.scoreStore == nil {
		return
	}

	atomic.AddInt32(&jobCount, 1)

	patterns := s.telemetry.DetectPatterns()

	for _, p := range patterns {
		if s.dash != nil {
			data, _ := json.Marshal(p)
			s.dash.BroadcastEvent("telemetry_pattern", string(data))
		}

		ok, reason := evolution.CanReview(s.evoStore, p.Role, evolution.DefaultReviewerConfig())
		if !ok {
			slog.Info("serve: evolution review skipped", "role", p.Role, "reason", reason)
			continue
		}

		result := evolution.ReviewResult{
			Classification: fmt.Sprintf("recurring_%s", p.Category),
			Suggestion:     fmt.Sprintf("Role %q has %d %s failures in 24h — investigate prompt or tool configuration", p.Role, p.Count, p.Category),
			Confidence:     0.7,
		}
		if err := evolution.RecordEvolution(ctx, s.evoStore, p.Role, repoID, result); err != nil {
			slog.Error("serve: failed to record evolution", "role", p.Role, "err", err)
		}
	}

	if count := atomic.LoadInt32(&jobCount); count >= 10 {
		atomic.StoreInt32(&jobCount, 0)
		s.runScoreReview(ctx, role, repoID)
	}
}

const scoreDropThreshold = 0.5

// runScoreReview checks if the role's score has dropped below the threshold
// and triggers an evolution review if so.
func (s *Server) runScoreReview(ctx context.Context, role, repoID string) {
	if s.evoStore == nil || s.scoreStore == nil {
		return
	}

	sc, err := s.scoreStore.ComputeScore(ctx, role, repoID, 30)
	if err != nil {
		slog.Warn("serve: score computation failed", "role", role, "err", err)
		return
	}

	if sc.SampleSize < 5 || sc.Value >= scoreDropThreshold {
		return
	}

	ok, reason := evolution.CanReview(s.evoStore, role, evolution.DefaultReviewerConfig())
	if !ok {
		slog.Info("serve: evolution review skipped (score drop)", "role", role, "reason", reason)
		return
	}

	result := evolution.ReviewResult{
		Classification: "score_drop",
		Suggestion:     fmt.Sprintf("Role %q score dropped to %.2f (%d samples) — review prompt effectiveness", role, sc.Value, sc.SampleSize),
		Confidence:     0.8,
	}
	if err := evolution.RecordEvolution(ctx, s.evoStore, role, repoID, result); err != nil {
		slog.Error("serve: failed to record score-drop evolution", "role", role, "err", err)
	}
}
