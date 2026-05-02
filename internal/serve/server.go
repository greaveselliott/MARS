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
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/scheduler"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/trust"
)

// Config controls the serve command.
type Config struct {
	WebhookAddr        string
	WebhookSecret      string
	DBPath             string
	Concurrency        int
	ModelsDir          string
	BinDir             string
	DashboardAddr      string
	RepoScope          string // if set, only operate on repos whose path matches this absolute path
	PerformanceProfile string
	InferenceTuning    inference.ServerTuning
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
	trustStore  *trust.Store

	mu        sync.Mutex
	started   bool
	startedAt time.Time
	startCtx  context.Context
	stopFunc  context.CancelFunc
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
	modelSet := hardware.DefaultModelsForHardware(hw, cfg.PerformanceProfile)

	roleMapping := map[string]hardware.Tier{
		"engineer":              hardware.TierCoding,
		"pipeline-fixer":        hardware.TierCoding,
		"reviewer":              hardware.TierReasoning,
		"code-reviewer":         hardware.TierReasoning,
		"qa":                    hardware.TierCoding,
		"documenter":            hardware.TierFast,
		"docs-writer":           hardware.TierFast,
		"release":               hardware.TierFast,
		"release-manager":       hardware.TierFast,
		"triager":               hardware.TierFast,
		"onboarder":             hardware.TierFast,
		"auditor":               hardware.TierReasoning,
		"security-auditor":      hardware.TierReasoning,
		"backlog":               hardware.TierFast,
		"janitor":               hardware.TierFast,
		"evolution":             hardware.TierReasoning,
		"dependency-updater":    hardware.TierFast,
		"performance-optimizer": hardware.TierCoding,
		"refactorer":            hardware.TierCoding,
		"incident-responder":    hardware.TierCoding,
	}

	binaryPath := filepath.Join(cfg.BinDir, "llama-server")
	router := inference.NewRouter(inference.RouterConfig{
		BinaryPath:  binaryPath,
		Models:      modelSet,
		RoleMapping: roleMapping,
		ModelsDir:   cfg.ModelsDir,
		Tuning:      cfg.InferenceTuning,
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

	trustStore, err := trust.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: trust store unavailable — mutating tools default to observer restrictions", "err", err)
	}

	executor := NewExecutor(repoLookup, router, traceStore, trustStore)

	sched := scheduler.New(jobQueue)

	telemStore, err := telemetry.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: telemetry store unavailable — events will not persist", "err", err)
	}

	telem := telemetry.NewCollector(nil, telemStore)

	s := &Server{
		cfg:        cfg,
		mux:        http.NewServeMux(),
		estop:      safety.NewEmergencyStop(),
		db:         db,
		repos:      repos,
		triggers:   triggerRouter,
		queue:      jobQueue,
		scheduler:  sched,
		router:     router,
		executor:   executor,
		telemetry:  telem,
		telemStore: telemStore,
		traceStore: traceStore,
		scoreStore: scoreStore,
		evoStore:   evoStore,
		trustStore: trustStore,
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
		ChainProvider: s.buildPipelineChain,
		Controls: dashboard.ControlCallbacks{
			Pause:    func() { s.Pause() },
			Resume:   func() { s.Resume() },
			Restart:  s.Restart,
			Stop:     s.Stop,
			Scan:     s.ScanRepo,
			RunRole:  s.RunRole,
			Status:   func() interface{} { return s.Status() },
			IsPaused: s.IsPaused,
			ListRepos: func() []dashboard.RepoInfoDTO {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				repos, _ := s.repos.List(ctx)
				out := make([]dashboard.RepoInfoDTO, len(repos))
				for i, r := range repos {
					out[i] = dashboard.RepoInfoDTO{ID: r.ID, Path: r.Path}
				}
				return out
			},
			ListRoles: s.ListRoles,
		},
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
	dash.HandleFunc("/api/throughput", s.handleThroughputAPI)

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
	s.startedAt = time.Now()
	s.startCtx = ctx
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
	if s.cfg.RepoScope != "" {
		repos = filterReposByPath(repos, s.cfg.RepoScope)
		slog.Info("serve: scoped to repo", "path", s.cfg.RepoScope, "matched", len(repos))
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
	if s.trustStore != nil {
		_ = s.trustStore.Close()
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

// --- Control surface methods (used by CLI key-listener and dashboard API) ---

// StatusResponse is the JSON shape returned by the status endpoint and
// consumed by the CLI status bar.
type StatusResponse struct {
	Healthy    bool       `json:"healthy"`
	Paused     bool       `json:"paused"`
	ActiveJobs int        `json:"active_jobs"`
	UptimeSecs float64    `json:"uptime_secs"`
	Repos      []RepoInfo `json:"repos"`
}

// RepoInfo is a lightweight repo descriptor for control-surface responses.
type RepoInfo struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// Pause stops the worker pool from claiming new jobs.
func (s *Server) Pause() {
	s.workers.Pause()
	if s.dash != nil {
		s.dash.BroadcastEvent("status_change", `{"state":"paused"}`)
	}
}

// Resume allows the worker pool to claim jobs again after a Pause.
func (s *Server) Resume() {
	s.workers.Resume()
	if s.dash != nil {
		s.dash.BroadcastEvent("status_change", `{"state":"running"}`)
	}
}

// IsPaused reports whether the worker pool is paused.
func (s *Server) IsPaused() bool {
	return s.workers.IsPaused()
}

// Restart performs a warm restart: stops workers, reloads manifests and
// triggers, then starts workers again. HTTP servers stay up.
func (s *Server) Restart(ctx context.Context) error {
	slog.Info("serve: warm restart initiated")
	if s.dash != nil {
		s.dash.BroadcastEvent("status_change", `{"state":"restarting"}`)
	}

	s.workers.Stop()
	s.scheduler.Stop()

	s.router.RestartAll()

	repos, err := s.repos.List(ctx)
	if err != nil {
		return fmt.Errorf("serve: restart — reload repos: %w", err)
	}
	if err := s.triggers.Rebuild(repos); err != nil {
		slog.Warn("serve: restart — trigger rebuild had errors", "err", err)
	}

	s.registerCronSchedules(repos)

	s.workers = queue.NewWorkerPool(s.queue, queue.WorkerConfig{
		Concurrency: s.cfg.concurrency(),
		OnJob:       s.executor.Execute,
		OnComplete:  s.handleJobComplete,
		OnFail:      s.handleJobFailed,
	})
	s.workers.Start(s.startCtx)
	s.scheduler.Start(s.startCtx)

	if s.dash != nil {
		s.dash.BroadcastEvent("status_change", `{"state":"running"}`)
	}
	slog.Info("serve: warm restart complete")
	return nil
}

// ScanRepo runs the scanner against a registered repo and enqueues
// findings as tickets.
func (s *Server) ScanRepo(ctx context.Context, repoID string) error {
	rec, err := s.repos.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("scan: lookup repo %s: %w", repoID, err)
	}
	if rec == nil {
		return fmt.Errorf("scan: repo %s not found in registry", repoID)
	}

	result, err := scanner.Scan(ctx, scanner.Config{RepoRoot: rec.Path})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	if err := scanner.GenerateTickets(result.Findings, rec.Path); err != nil {
		return fmt.Errorf("scan: generate tickets: %w", err)
	}

	slog.Info("serve: scan complete", "repo", repoID, "findings", len(result.Findings))
	if s.dash != nil {
		s.dash.BroadcastEvent("scan_complete", fmt.Sprintf(
			`{"repo_id":"%s","findings":%d}`, repoID, len(result.Findings)))
	}
	return nil
}

// ScanAllRepos runs the scanner on every registered repo sequentially.
func (s *Server) ScanAllRepos(ctx context.Context) error {
	repos, err := s.repos.List(ctx)
	if err != nil {
		return fmt.Errorf("scan-all: list repos: %w", err)
	}
	var firstErr error
	for _, repo := range repos {
		if err := s.ScanRepo(ctx, repo.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// RunRole validates that a role exists in the manifest and enqueues it.
func (s *Server) RunRole(ctx context.Context, repoID, role string) error {
	manifest, err := s.loadManifest(ctx, repoID)
	if err != nil {
		return err
	}
	if _, ok := manifest.Roles[role]; !ok {
		available := make([]string, 0, len(manifest.Roles))
		for name := range manifest.Roles {
			available = append(available, name)
		}
		return fmt.Errorf("run-role: role %q not found in manifest — available: %v", role, available)
	}

	trigger := fmt.Sprintf(`{"type":"manual","source":"control-surface","role":"%s"}`, role)
	jobID, err := s.SeedJob(ctx, repoID, role, trigger)
	if err != nil {
		return fmt.Errorf("run-role: enqueue %s: %w", role, err)
	}
	slog.Info("serve: manual role run enqueued", "role", role, "job", jobID)
	return nil
}

// Status returns a snapshot of the orchestrator's state.
func (s *Server) Status() StatusResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	repos, _ := s.repos.List(ctx)
	repoInfos := make([]RepoInfo, len(repos))
	for i, r := range repos {
		repoInfos[i] = RepoInfo{ID: r.ID, Path: r.Path}
	}

	active, _ := s.queue.CountByStatus(ctx, "running")

	return StatusResponse{
		Healthy:    s.health.Load(),
		Paused:     s.workers.IsPaused(),
		ActiveJobs: active,
		UptimeSecs: time.Since(s.startedAt).Seconds(),
		Repos:      repoInfos,
	}
}

// ListRoles returns role names from the manifest of a registered repo.
func (s *Server) ListRoles(repoID string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	manifest, err := s.loadManifest(ctx, repoID)
	if err != nil {
		return nil
	}
	roles := make([]string, 0, len(manifest.Roles))
	for name := range manifest.Roles {
		roles = append(roles, name)
	}
	return roles
}

// loadManifest fetches the manifest for a repo by ID.
func (s *Server) loadManifest(ctx context.Context, repoID string) (*bundle.Manifest, error) {
	rec, err := s.repos.FindByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("load manifest: lookup repo %s: %w", repoID, err)
	}
	if rec == nil {
		return nil, fmt.Errorf("load manifest: repo %s not found", repoID)
	}
	manifest, err := bundle.Load(rec.Path)
	if err != nil {
		return nil, fmt.Errorf("load manifest: parse %s: %w", rec.Path, err)
	}
	return manifest, nil
}

// handleTelemetryAPI serves the telemetry event history as JSON.
func (s *Server) handleTelemetryAPI(w http.ResponseWriter, r *http.Request) {
	type apiResponse struct {
		Events []telemetry.Event                 `json:"events"`
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

	roles := []string{"ceo", "cto-weekly", "coo", "engineer", "qa",
		"security", "dependency-manager", "release-manager",
		"dogfood", "pipeline-fixer", "janitor"}

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

func (s *Server) handleThroughputAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type jobEntry struct {
		ID        string  `json:"id"`
		Role      string  `json:"role"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
		Duration  *string `json:"duration,omitempty"`
		Error     string  `json:"error,omitempty"`
	}

	type apiResponse struct {
		Hourly     []queue.HourlyCount `json:"hourly"`
		RecentJobs []jobEntry          `json:"recent_jobs"`
		Summary    struct {
			Total     int `json:"total"`
			Completed int `json:"completed"`
			Failed    int `json:"failed"`
			Running   int `json:"running"`
			Pending   int `json:"pending"`
		} `json:"summary"`
	}

	var resp apiResponse

	hourly, err := s.queue.JobCountsByHour(ctx, 48)
	if err != nil {
		slog.Error("serve: throughput hourly query error", "err", err)
	}
	resp.Hourly = hourly

	recent, err := s.queue.RecentJobs(ctx, 50)
	if err != nil {
		slog.Error("serve: throughput recent jobs error", "err", err)
	}
	for _, j := range recent {
		entry := jobEntry{
			ID:        j.ID[:8],
			Role:      j.Role,
			Status:    string(j.Status),
			CreatedAt: j.CreatedAt.Format(time.RFC3339),
			Error:     j.Error,
		}
		if j.CompletedAt != nil {
			d := j.CompletedAt.Sub(j.CreatedAt).Round(time.Second).String()
			entry.Duration = &d
		}
		resp.RecentJobs = append(resp.RecentJobs, entry)

		resp.Summary.Total++
		switch j.Status {
		case queue.StatusCompleted:
			resp.Summary.Completed++
		case queue.StatusFailed:
			resp.Summary.Failed++
		case queue.StatusRunning:
			resp.Summary.Running++
		case queue.StatusPending, queue.StatusClaimed:
			resp.Summary.Pending++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("serve: throughput API encode error", "err", err)
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

// buildPipelineChain walks the manifest's `then` chains starting from CEO
// and returns an ordered list of chain nodes for the dashboard.
func (s *Server) buildPipelineChain() []dashboard.ChainNode {
	repos, err := s.repos.List(context.Background())
	if err != nil || len(repos) == 0 {
		return nil
	}

	manifest, err := bundle.Load(repos[0].Path)
	if err != nil {
		return nil
	}

	var nodes []dashboard.ChainNode
	visited := map[string]bool{}

	// Walk the chain starting from "ceo" (the pipeline entry point).
	current := "ceo"
	for current != "" && !visited[current] {
		visited[current] = true
		role, ok := manifest.Roles[current]
		if !ok {
			break
		}

		nodes = append(nodes, dashboard.ChainNode{Name: current})

		next := ""
		for _, t := range role.Then {
			if !visited[t] {
				next = t
				break
			}
		}
		current = next
	}

	// Append idle_then targets from the last node that has them.
	for i := len(nodes) - 1; i >= 0; i-- {
		role, ok := manifest.Roles[nodes[i].Name]
		if !ok {
			continue
		}
		for _, t := range role.IdleThen {
			if !visited[t] {
				nodes = append(nodes, dashboard.ChainNode{Name: t})
				visited[t] = true
			}
		}
		if len(role.IdleThen) > 0 {
			break
		}
	}

	// Add any roles with `then` branches not yet visited (parallel chains like dogfood).
	for _, node := range nodes {
		role, ok := manifest.Roles[node.Name]
		if !ok {
			continue
		}
		for _, t := range role.Then {
			if !visited[t] {
				nodes = append(nodes, dashboard.ChainNode{Name: t})
				visited[t] = true
			}
		}
	}

	return nodes
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
	proposals := telemetry.TriagePatterns(patterns)

	for i, p := range patterns {
		proposal := proposals[i]
		if s.dash != nil {
			data, _ := json.Marshal(p)
			s.dash.BroadcastEvent("telemetry_pattern", string(data))
			proposalData, _ := json.Marshal(proposal)
			s.dash.BroadcastEvent("telemetry_triage", string(proposalData))
		}

		ok, reason := evolution.CanReview(s.evoStore, proposal.Role, evolution.DefaultReviewerConfig())
		if !ok {
			slog.Info("serve: evolution review skipped", "role", proposal.Role, "reason", reason)
			continue
		}

		result := evolution.ReviewResult{
			Classification: fmt.Sprintf("telemetry_%s_%s", proposal.Target, proposal.Category),
			Suggestion:     proposal.Suggestion,
			FilesToModify:  proposal.CandidateFiles,
			Confidence:     proposal.Confidence,
		}
		if err := evolution.RecordEvolution(ctx, s.evoStore, proposal.Role, repoID, result); err != nil {
			slog.Error("serve: failed to record evolution", "role", proposal.Role, "err", err)
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

	proposal, ok := telemetry.TriageScore(telemetry.ScoreSnapshot{
		Role:       role,
		RepoID:     repoID,
		Value:      sc.Value,
		SampleSize: sc.SampleSize,
		WindowDays: sc.WindowDays,
	})
	if !ok {
		return
	}

	if s.dash != nil {
		data, _ := json.Marshal(proposal)
		s.dash.BroadcastEvent("score_triage", string(data))
	}

	result := evolution.ReviewResult{
		Classification: fmt.Sprintf("score_%s", proposal.Target),
		Suggestion:     proposal.Suggestion,
		FilesToModify:  proposal.CandidateFiles,
		Confidence:     proposal.Confidence,
	}
	if err := evolution.RecordEvolution(ctx, s.evoStore, role, repoID, result); err != nil {
		slog.Error("serve: failed to record score-drop evolution", "role", role, "err", err)
	}
}

func filterReposByPath(repos []RepoRecord, scopePath string) []RepoRecord {
	var filtered []RepoRecord
	for _, r := range repos {
		if r.Path == scopePath {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
