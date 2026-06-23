/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/design-docs/local-inference.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/board-driven-integrations.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/convergence-state-machine.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-013-board-driven-integrations.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-012-self-improvement-loop.md
*/
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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/greaveselliott/mars-harness/internal/bundle"
	"github.com/greaveselliott/mars-harness/internal/codeintel"
	"github.com/greaveselliott/mars-harness/internal/dashboard"
	"github.com/greaveselliott/mars-harness/internal/evolution"
	gh "github.com/greaveselliott/mars-harness/internal/github"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/inference"
	"github.com/greaveselliott/mars-harness/internal/integrations"
	"github.com/greaveselliott/mars-harness/internal/orchestration"
	"github.com/greaveselliott/mars-harness/internal/orgstate"
	"github.com/greaveselliott/mars-harness/internal/power"
	"github.com/greaveselliott/mars-harness/internal/queue"
	"github.com/greaveselliott/mars-harness/internal/remediation"
	"github.com/greaveselliott/mars-harness/internal/safety"
	"github.com/greaveselliott/mars-harness/internal/scanner"
	"github.com/greaveselliott/mars-harness/internal/scheduler"
	"github.com/greaveselliott/mars-harness/internal/scoring"
	"github.com/greaveselliott/mars-harness/internal/telemetry"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
	"github.com/greaveselliott/mars-harness/internal/trace"
	"github.com/greaveselliott/mars-harness/internal/trust"
	"github.com/greaveselliott/mars-harness/internal/ui"
)

const (
	queueSelfHealInterval = time.Minute
	recoveryJobStaleAfter = 10 * time.Minute
)

// Config controls the serve command.
type Config struct {
	WebhookAddr           string
	WebhookSecret         string
	DBPath                string
	Concurrency           int
	ModelsDir             string
	BinDir                string
	DashboardAddr         string
	RepoScope             string // if set, only operate on repos whose path matches this absolute path
	PerformanceProfile    string
	InferenceTuning       inference.ServerTuning
	ModelEndpoint         string
	RequireModelPreflight bool
	EphemeralHTTPFallback bool
	JobViews              ui.JobViewFactory
	CodeIntelDisabled     bool
	CodeIntelSource       string
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
	orgStore    *orgstate.Store
	remediators remediation.Registry

	mu        sync.Mutex
	started   bool
	startedAt time.Time
	startCtx  context.Context
	stopReq   chan struct{}
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

	modelEndpoint := strings.TrimSpace(cfg.ModelEndpoint)
	hw := hardware.Detect()
	modelSet := hardware.DefaultModelsForHardware(hw, cfg.PerformanceProfile)

	modelsDir := strings.TrimSpace(cfg.ModelsDir)
	if modelsDir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			db.Close()
			return nil, fmt.Errorf("serve: resolve models directory: %w", homeErr)
		}
		modelsDir = filepath.Join(home, ".mars-harness", "models")
	}
	if modelEndpoint != "" {
		modelSet = map[hardware.Tier]hardware.ModelSpec{}
	} else if cfg.RequireModelPreflight {
		if missing, err := hardware.MissingRequiredModelFiles(modelsDir, cfg.PerformanceProfile); err != nil {
			db.Close()
			return nil, fmt.Errorf("serve: verify profile model files: %w", err)
		} else if err := hardware.ProfileModelPreflightError(cfg.PerformanceProfile, missing); err != nil {
			db.Close()
			return nil, fmt.Errorf("serve: %w", err)
		}
	}

	roleMapping := inference.DefaultRoleTierMapping()

	binaryPath := filepath.Join(cfg.BinDir, "llama-server")
	router := inference.NewRouter(inference.RouterConfig{
		BinaryPath:  binaryPath,
		Models:      modelSet,
		RoleMapping: roleMapping,
		ModelsDir:   cfg.ModelsDir,
		FallbackURL: modelEndpoint,
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

	orgStore, err := orgstate.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: orgstate store unavailable — dispatch-mode orchestration disabled", "err", err)
	}

	executor := NewExecutor(repoLookup, router, cfg.DBPath, traceStore, trustStore)
	executor.SetCodeIntel(codeintel.NewRuntime(!cfg.CodeIntelDisabled, cfg.CodeIntelSource))
	if cfg.JobViews != nil {
		executor.SetJobViewFactory(cfg.JobViews)
	}

	sched := scheduler.New(jobQueue)

	telemStore, err := telemetry.OpenStore(cfg.DBPath)
	if err != nil {
		slog.Warn("serve: telemetry store unavailable — events will not persist", "err", err)
	}

	telem := telemetry.NewCollector(nil, telemStore)

	s := &Server{
		cfg:         cfg,
		mux:         http.NewServeMux(),
		estop:       safety.NewEmergencyStop(),
		db:          db,
		repos:       repos,
		triggers:    triggerRouter,
		queue:       jobQueue,
		scheduler:   sched,
		router:      router,
		executor:    executor,
		telemetry:   telem,
		telemStore:  telemStore,
		traceStore:  traceStore,
		scoreStore:  scoreStore,
		evoStore:    evoStore,
		trustStore:  trustStore,
		orgStore:    orgStore,
		stopReq:     make(chan struct{}, 1),
		remediators: remediation.DefaultRegistry(),
	}

	telem.SetRemediator(s.handleRemediation)
	executor.SetInterventionSignalHandler(s.recordInterventionDebtSignal)
	executor.SetOrgState(orgStore)

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
		Addr:             dashAddr,
		EmergencyStop:    func() []error { return s.estop.Execute(context.Background()) },
		PipelineProvider: s.buildPipelineView,
		Controls: dashboard.ControlCallbacks{
			Pause:    func() { s.Pause() },
			Resume:   func() { s.Resume() },
			Restart:  s.Restart,
			Stop:     s.RequestStop,
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
	dash.HandleFunc("/api/quality-score", s.handleQualityScoreAPI)
	dash.HandleFunc("/api/throughput", s.handleThroughputAPI)
	dash.HandleFunc("/api/orchestration", s.handleOrchestrationAPI)
	dash.HandleFunc("/api/orchestration/decisions", s.handleOrchestrationDecisionsAPI)

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
	s.selfHealRecoveryQueue(ctx, "startup")

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

	ln, err := s.listenControlSocket("webhook", s.cfg.WebhookAddr)
	if err != nil {
		return fmt.Errorf("serve: failed to bind %s — check if the port is already in use: %w",
			s.cfg.WebhookAddr, err)
	}

	dashLn, err := s.listenControlSocket("dashboard", s.dashHTTP.Addr)
	if err != nil {
		_ = ln.Close()
		return fmt.Errorf("serve: failed to bind dashboard %s — check if the port is already in use: %w",
			s.dashHTTP.Addr, err)
	}
	s.http.Addr = ln.Addr().String()
	s.dashHTTP.Addr = dashLn.Addr().String()

	s.workers.Start(ctx)
	s.scheduler.Start(ctx)
	go s.runQueueSelfHeal(ctx)
	go s.runOrchestratorSurvey(ctx)

	power.StartWatchdog(ctx, s.handleWake)

	s.health.Store(true)
	slog.Info("serve: orchestrator ready",
		"addr", ln.Addr().String(),
		"dashboard", httpURLForListener(dashLn.Addr()),
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
	case <-s.stopReq:
		slog.Info("serve: dashboard stop requested, shutting down")
		return s.Stop(context.Background())
	case err := <-errCh:
		s.health.Store(false)
		return err
	}
}

func (s *Server) listenControlSocket(label, addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		return ln, nil
	}
	if !s.cfg.EphemeralHTTPFallback || !isAddrInUse(err) {
		return nil, err
	}
	fallbackAddr := "127.0.0.1:0"
	fallbackLn, fallbackErr := net.Listen("tcp", fallbackAddr)
	if fallbackErr != nil {
		return nil, fmt.Errorf("%w; fallback %s bind failed: %v", err, fallbackAddr, fallbackErr)
	}
	slog.Warn("serve: control address in use, using ephemeral scoped lifecycle address",
		"listener", label,
		"requested", addr,
		"actual", fallbackLn.Addr().String(),
	)
	return fallbackLn, nil
}

func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "address already in use")
}

func httpURLForListener(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String()
	}
	switch host {
	case "", "::", "0.0.0.0", "[::]":
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
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

	if n, err := s.queue.PreemptPending(shutdownCtx, "orchestrator stopped with pending work — resume with mars-harness start"); err != nil {
		slog.Warn("serve: failed to preempt pending jobs on stop", "err", err)
	} else if n > 0 {
		slog.Warn("serve: preempted pending jobs on stop", "count", n)
	}

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

// RequestStop asks the server's main loop to run the normal shutdown path.
// Dashboard handlers use this instead of calling Stop directly so the dashboard
// HTTP server is not asked to shut down while it is still serving the request.
func (s *Server) RequestStop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case s.stopReq <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("serve: request stop: %w", ctx.Err())
	default:
		return nil
	}
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
	s.selfHealRecoveryQueue(ctx, "wake")

	if s.dash != nil {
		s.dash.BroadcastEvent("wake_recovery", fmt.Sprintf("resumed after %s sleep", gap.Round(time.Second)))
	}
}

func (s *Server) runQueueSelfHeal(ctx context.Context) {
	ticker := time.NewTicker(queueSelfHealInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			s.selfHealRecoveryQueue(healCtx, "watchdog")
			cancel()
		}
	}
}

func (s *Server) selfHealRecoveryQueue(ctx context.Context, source string) {
	report, err := s.queue.RepairActiveRecoveryJobs(ctx, recoveryJobStaleAfter)
	if err != nil {
		slog.Warn("serve: recovery queue self-heal failed", "source", source, "err", err)
		return
	}
	if report.Total() == 0 {
		return
	}

	slog.Warn("serve: self-healed recovery queue",
		"source", source,
		"stale_failed", report.StaleFailed,
		"duplicates_cancelled", report.DuplicatesCancelled,
		"duplicates_failed", report.DuplicatesFailed,
		"active_groups", report.ActiveGroups,
	)
	if s.dash != nil {
		msg := fmt.Sprintf("repaired recovery queue: stale_failed=%d duplicates_cancelled=%d duplicates_failed=%d",
			report.StaleFailed, report.DuplicatesCancelled, report.DuplicatesFailed)
		s.dash.BroadcastEvent("queue_self_heal", msg)
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
	ID          string `json:"id"`
	Path        string `json:"path"`
	FlowProfile string `json:"flow_profile"`
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
	s.selfHealRecoveryQueue(ctx, "restart")

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
	if err := s.enqueueStaleTicketHygiene(ctx, rec, result.Findings); err != nil {
		return fmt.Errorf("scan: enqueue stale ticket hygiene: %w", err)
	}

	slog.Info("serve: scan complete", "repo", repoID, "findings", len(result.Findings))
	if s.dash != nil {
		s.dash.BroadcastEvent("scan_complete", fmt.Sprintf(
			`{"repo_id":"%s","findings":%d}`, repoID, len(result.Findings)))
	}
	return nil
}

func (s *Server) enqueueStaleTicketHygiene(ctx context.Context, rec *RepoRecord, findings []scanner.Finding) error {
	if rec == nil {
		return nil
	}
	var paths []string
	for _, finding := range findings {
		if finding.Type == "stale_in_progress_ticket" {
			paths = append(paths, finding.Path)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	manifest, err := bundle.Load(rec.Path)
	if err != nil {
		slog.Warn("serve: stale ticket hygiene skipped; manifest load failed", "repo_id", rec.ID, "err", err)
		return nil
	}
	if _, ok := manifest.Roles["janitor"]; !ok {
		slog.Warn("serve: stale ticket hygiene skipped; janitor role is not configured", "repo_id", rec.ID)
		return nil
	}
	trigger, _ := json.Marshal(map[string]any{
		"type":    "ticket.stale_in_progress",
		"source":  "scanner",
		"count":   len(paths),
		"tickets": paths,
	})
	jobID, err := s.queue.Enqueue(ctx, queue.Job{
		RepoID:         rec.ID,
		Role:           "janitor",
		Trigger:        string(trigger),
		IdempotencyKey: fmt.Sprintf("ticket:stale-in-progress:%s", rec.ID),
	})
	if err != nil {
		return err
	}
	slog.Info("serve: stale ticket hygiene enqueued", "repo_id", rec.ID, "job_id", jobID, "tickets", len(paths))
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
		cfg := s.integrationConfigForRepo(r)
		repoInfos[i] = RepoInfo{ID: r.ID, Path: r.Path, FlowProfile: cfg.FlowProfile}
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

func (s *Server) handleQualityScoreAPI(w http.ResponseWriter, r *http.Request) {
	repoPath := strings.TrimSpace(s.cfg.RepoScope)
	if repoPath == "" && s.repos != nil {
		repos, err := s.repos.List(r.Context())
		if err == nil && len(repos) > 0 {
			repoPath = repos[0].Path
		}
	}
	if repoPath == "" {
		http.Error(w, "quality score unavailable: no repo is registered", http.StatusNotFound)
		return
	}
	scorePath := filepath.Join(repoPath, "docs", "QUALITY_SCORE.md")
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	http.ServeFile(w, r, scorePath)
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

func (s *Server) handleOrchestrationAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	type roleSummary struct {
		Name     string   `json:"name"`
		Domain   string   `json:"domain"`
		Mode     string   `json:"mode"`
		Model    string   `json:"model"`
		Schedule string   `json:"schedule"`
		Tools    []string `json:"tools"`
	}
	type repoSummary struct {
		ID                string        `json:"id"`
		Path              string        `json:"path"`
		OrchestrationMode string        `json:"orchestration_mode"`
		Roles             []roleSummary `json:"roles"`
	}
	type apiResponse struct {
		Repos        []repoSummary          `json:"repos"`
		Dispositions []orgstate.Disposition `json:"dispositions"`
		Decisions    []orgstate.Decision    `json:"decisions"`
	}
	var resp apiResponse
	if s.repos != nil {
		repos, err := s.repos.List(ctx)
		if err == nil {
			for _, repo := range repos {
				mode := "legacy"
				var roles []roleSummary
				if manifest, mErr := bundle.Load(repo.Path); mErr == nil {
					if manifest.DispatchMode() {
						mode = "dispatch"
					}
					for _, node := range dashboardRoleNodes(manifest, false) {
						role := manifest.Roles[node.Name]
						roles = append(roles, roleSummary{
							Name:     node.Name,
							Domain:   role.Domain,
							Mode:     role.Mode,
							Model:    role.Model,
							Schedule: role.Schedule,
							Tools:    role.Tools,
						})
					}
				}
				resp.Repos = append(resp.Repos, repoSummary{ID: repo.ID, Path: repo.Path, OrchestrationMode: mode, Roles: roles})
			}
		}
	}
	if s.orgStore != nil {
		dispositions, err := s.orgStore.RecentDispositions(ctx, "", 25)
		if err == nil {
			resp.Dispositions = dispositions
		}
		decisions, err := s.orgStore.RecentDecisions(ctx, "", 25)
		if err == nil {
			resp.Decisions = decisions
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("serve: orchestration API encode error", "err", err)
	}
}

func (s *Server) handleOrchestrationDecisionsAPI(w http.ResponseWriter, r *http.Request) {
	repoID := strings.TrimSpace(r.URL.Query().Get("repo_id"))
	var decisions []orgstate.Decision
	if s.orgStore != nil {
		if rows, err := s.orgStore.RecentDecisions(r.Context(), repoID, 50); err == nil {
			decisions = rows
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]orgstate.Decision{"decisions": decisions}); err != nil {
		slog.Error("serve: orchestration decisions API encode error", "err", err)
	}
}

// Repos returns the registry for external use (e.g. CLI register command).
func (s *Server) Repos() *RepoRegistry { return s.repos }

// SeedJob enqueues a job directly. Used by the `start` command to inject the
// first agent (typically CEO) before the server loop begins.
func (s *Server) SeedJob(ctx context.Context, repoID, role, trigger string) (string, error) {
	idempotencyKey := fmt.Sprintf("seed:%s:%s:%d", repoID, role, time.Now().UnixNano())
	return s.seedJob(ctx, repoID, role, trigger, idempotencyKey)
}

// SeedBootstrapJob enqueues the first bootstrap job with a stable key so
// restarting `mars-harness start` cannot duplicate an already-active pipeline.
func (s *Server) SeedBootstrapJob(ctx context.Context, repoID, role, trigger string) (string, error) {
	idempotencyKey := fmt.Sprintf("seed:%s:%s:bootstrap", repoID, role)
	return s.seedJob(ctx, repoID, role, trigger, idempotencyKey)
}

func (s *Server) seedJob(ctx context.Context, repoID, role, trigger, idempotencyKey string) (string, error) {
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
// In dispatch mode it records a decision from the terminal disposition,
// routing deterministic handoffs directly and using Orchestrator as the
// ambiguous fallback. Legacy manifests still resolve `then` and `idle_then`
// chains so existing deployed harnesses do not break.
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
		if manifest.DispatchMode() {
			s.handleDispatchComplete(ctx, job, rec, manifest)
		}
		return
	}

	if manifest.DispatchMode() {
		s.handleDispatchComplete(ctx, job, rec, manifest)
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
			if s.scoreStore != nil {
				_ = s.scoreStore.RecordOutcome(ctx, scoring.Outcome{
					JobID:   job.ID,
					RepoID:  job.RepoID,
					Role:    job.Role,
					Type:    scoring.OutcomeNoop,
					Details: "self-chain skipped because run was too short; orchestrator survey will route follow-up if needed",
				})
			}
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

func (s *Server) handleDispatchComplete(ctx context.Context, job *queue.Job, rec *RepoRecord, manifest *bundle.Manifest) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	if s.orgStore == nil {
		log.Warn("serve: dispatch mode unavailable — orgstate store is nil")
		return
	}
	if job.Role == "engineer" {
		s.cancelStaleTicketOwnerSurveyJobs(ctx, *rec)
	}

	disposition, err := s.orgStore.GetDisposition(ctx, job.ID)
	if err != nil {
		log.Warn("serve: dispatch disposition load failed", "err", err)
		return
	}
	if disposition == nil {
		log.Warn("serve: dispatch mode job completed without disposition")
		return
	}

	snap, snapErr := snapshotTickets(rec.Path)
	ticketHash := ""
	if snapErr == nil {
		ticketHash = snap.routingHash()
	} else {
		log.Warn("serve: dispatch ticket snapshot failed", "err", snapErr)
	}

	recent, err := s.orgStore.RecentDecisions(ctx, job.RepoID, 20)
	if err != nil {
		log.Warn("serve: recent decisions unavailable", "err", err)
	}

	var source *orgstate.Disposition
	if sourceDisposition, ok := sourceDispositionFromDispatchTrigger(job.RepoID, job.Trigger); ok {
		source = &sourceDisposition
	}
	sourceForGate := source
	if sourceForGate == nil {
		sourceForGate = disposition
	}

	decision, err := orchestration.Decide(orchestration.Input{
		Disposition:       *disposition,
		Manifest:          manifest,
		RecentDecisions:   recent,
		TicketStateHash:   ticketHash,
		SourceDisposition: source,
	})
	if err != nil {
		log.Warn("serve: dispatch decision failed", "err", err)
		return
	}
	if snapErr == nil {
		decision = enforceEngineerTicketPrerequisite(decision, snap, manifest, sourceForGate)
		decision = enforceReleaseRequiresCompletedFeatureScenarios(decision, snap, manifest, rec.Path)
	}
	decision, err = s.orgStore.RecordDecision(ctx, decision)
	if err != nil {
		log.Warn("serve: record dispatch decision failed", "err", err)
		return
	}

	if s.dash != nil {
		dispPayload, _ := json.Marshal(disposition)
		s.dash.BroadcastEvent("job_disposition", string(dispPayload))
		decisionPayload, _ := json.Marshal(decision)
		s.dash.BroadcastEvent("orchestration_decision", string(decisionPayload))
	}

	if strings.TrimSpace(decision.NextRole) == "" {
		log.Info("serve: dispatch stopped", "reason", decision.StopReason)
		return
	}

	triggerPayload := newDispatchTriggerPayload(job, decision, *disposition)
	if source != nil && isPinnedEngineerReworkSource(*source) && strings.EqualFold(decision.NextRole, "engineer") {
		triggerPayload = newDispatchTriggerPayloadForSource(source.Role, source.JobID, decision, *source)
	}
	triggerJSON, err := json.Marshal(triggerPayload)
	if err != nil {
		log.Error("serve: failed to marshal dispatch trigger", "target_role", decision.NextRole, "err", err)
		return
	}
	dispatchJob := queue.Job{
		RepoID:         job.RepoID,
		Role:           decision.NextRole,
		Trigger:        string(triggerJSON),
		IdempotencyKey: fmt.Sprintf("dispatch:%s:%s:%s", job.ID, job.RepoID, decision.NextRole),
	}
	jobID, err := s.queue.Enqueue(ctx, dispatchJob)
	if err != nil {
		log.Error("serve: failed to enqueue dispatch job", "target_role", decision.NextRole, "err", err)
		return
	}
	log.Info("serve: dispatch job enqueued", "target_role", decision.NextRole, "dispatch_job_id", jobID)
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"decision_id": decision.ID,
			"job_id":      jobID,
			"role":        decision.NextRole,
			"repo":        job.RepoID,
			"reason":      decision.Reason,
		})
		s.dash.BroadcastEvent("dispatch_enqueued", string(payload))
	}
}

// handleJobFailed is the OnFail callback for the worker pool.
// It records a telemetry event (which triggers classification and
// remediation), then falls back to the existing self-chain recovery
// for roles that don't match a specific remediation action.
func (s *Server) handleJobFailed(ctx context.Context, job *queue.Job, jobErr error) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)

	cat := telemetry.Classify(jobErr.Error())
	outcomeType := scoring.OutcomeFailed
	switch cat {
	case telemetry.CategoryToolTimeout, telemetry.CategoryContextOverflow:
		outcomeType = scoring.OutcomeTimeout
	case telemetry.CategoryGuardrailBlock, telemetry.CategoryWorkspaceHygiene:
		outcomeType = scoring.OutcomeGuardrailBlocked
	}
	traceID := s.latestTraceID(ctx, job.ID)
	plan := s.planJobFailureRemediation(ctx, job, cat, jobErr.Error(), traceID, true)
	executions := s.executeReadyRemediation(ctx, plan)
	if s.scoreStore != nil {
		_ = s.scoreStore.RecordOutcome(ctx, scoring.Outcome{
			JobID:   job.ID,
			RepoID:  job.RepoID,
			Role:    job.Role,
			Type:    outcomeType,
			Details: remediationOutcomeDetails(jobErr.Error(), traceID, plan, executions),
		})
	}

	suppressSecondaryTicketGate := cat == telemetry.CategoryTicketGate && s.jobHadPolicyBlock(job.ID)
	evt := s.telemetry.Record(job.ID, job.RepoID, job.Role, jobErr.Error())
	if suppressSecondaryTicketGate {
		log.Info("serve: suppressing secondary ticket-gate intervention debt after policy block")
	} else {
		s.recordInterventionDebtSignal(ctx, interventionDebtSignal{
			Kind:           interventionDebtSignalKindForCategory(cat),
			RepoID:         job.RepoID,
			Role:           job.Role,
			JobID:          job.ID,
			Category:       cat,
			EvidenceWindow: "24h",
			Event:          &evt,
			TraceID:        traceID,
			Outcome:        string(outcomeType),
			Message:        jobErr.Error(),
		})
	}

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

	if manifest.DispatchMode() {
		if s.orgStore != nil {
			_ = s.orgStore.RecordDisposition(ctx, orgstate.Disposition{
				JobID:   job.ID,
				RepoID:  job.RepoID,
				Role:    job.Role,
				Status:  "failed",
				Reason:  jobErr.Error(),
				TraceID: traceID,
			})
			if cat == telemetry.CategoryWorkspaceHygiene {
				log.Warn("serve: not dispatching workspace hygiene failure; deterministic recipe must be applied before retry",
					"error", jobErr,
				)
				return
			}
			if deterministicContainmentFailure(cat, jobErr.Error()) {
				log.Warn("serve: not dispatching deterministic containment failure; operator must restore workspace below blast-radius limits before retry",
					"error", jobErr,
				)
				return
			}
			if cat == telemetry.CategoryDispatchProtocol {
				log.Warn("serve: not dispatching protocol failure through Orchestrator; role prompt or tool usage must be corrected before retry",
					"error", jobErr,
				)
				return
			}
			if cat == telemetry.CategoryTicketGate {
				if job.Role == "engineer" && !isTicketGateRepairTrigger(job.Trigger) {
					s.enqueueTicketGateRepair(ctx, job, jobErr)
				} else {
					log.Warn("serve: not dispatching ticket-gate failure through Orchestrator",
						"error", jobErr,
					)
				}
				return
			}
			if evt, ok := s.latestUnremediatedGuardrailLoop(job.ID); ok {
				s.recordGuardrailLoopEscalation(ctx, job, evt, jobErr, traceID)
				return
			}
			if cat == telemetry.CategoryMaxTurns && job.Role == "engineer" && !isTicketGateRepairTrigger(job.Trigger) && s.jobHadTicketLifecyclePolicyBlock(job.ID) {
				log.Warn("serve: enqueueing bounded ticket-gate repair after max turns followed ticket lifecycle policy blocks",
					"error", jobErr,
				)
				s.enqueueTicketGateRepair(ctx, job, jobErr)
				return
			}
			if (cat == telemetry.CategoryMaxTurns || cat == telemetry.CategoryCircleDetected) && job.Role == "engineer" && !isTicketGateRepairTrigger(job.Trigger) && !isProductContinuationTrigger(job.Trigger) && engineerHasContinuableProductTicket(rec.Path) {
				log.Warn("serve: enqueueing bounded product continuation after engineer loop boundary with an active product ticket",
					"category", cat,
					"error", jobErr,
				)
				s.enqueueProductContinuation(ctx, job, jobErr)
				return
			}
			if job.Role != "orchestrator" && isConvergenceRuntimeFailure(cat) {
				s.routeConvergenceFailure(ctx, job, cat, jobErr, traceID)
				return
			}
			if job.Role != "orchestrator" && dispatchRuntimeFailureStops(cat) {
				log.Warn("serve: not dispatching runtime failure through Orchestrator; foundation telemetry or operator retry must resolve it first",
					"category", cat,
					"error", jobErr,
				)
				return
			}
			if job.Role == "orchestrator" {
				if !s.dispatchFallbackAfterOrchestratorFailure(ctx, job, rec, manifest, jobErr) {
					log.Warn("serve: not dispatching failed Orchestrator without deterministic source handoff",
						"error", jobErr,
					)
				}
				return
			}
			s.handleDispatchComplete(ctx, job, rec, manifest)
		}
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

	if cat == telemetry.CategoryTicketGate {
		log.Warn("serve: not auto-recovering ticket-gate failure",
			"error", jobErr,
		)
		go s.checkEvolution(context.Background(), job.Role, job.RepoID)
		return
	}

	if isAutoRecoverTrigger(job.Trigger) {
		log.Warn("serve: not auto-recovering failed recovery job",
			"error", jobErr,
		)
		go s.checkEvolution(context.Background(), job.Role, job.RepoID)
		return
	}

	if !cat.Retryable() {
		log.Warn("serve: not auto-recovering deterministic failure",
			"category", cat,
			"error", jobErr,
		)
		go s.checkEvolution(context.Background(), job.Role, job.RepoID)
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

	idempotencyKey := fmt.Sprintf("recover:%s:%s", job.RepoID, job.Role)
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

func (s *Server) enqueueTicketGateRepair(ctx context.Context, job *queue.Job, jobErr error) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	triggerJSON, _ := json.Marshal(map[string]string{
		"type":         "ticket_gate_repair",
		"source_job":   job.ID,
		"reason":       jobErr.Error(),
		"repair_scope": "ticket_lifecycle_and_evidence_only",
		"ask":          "Repair only the failed ticket lifecycle or evidence condition. If evidence is missing, run bounded validation that exercises the named BDD scenario, update ticket evidence, move the ticket to done, commit the lifecycle correction, then record job_disposition_record. For static HTML/CSS/JS, run node --check main.js when JavaScript exists, then python3 -m http.server 18081 --bind 127.0.0.1 with background:true, curl -fsS http://127.0.0.1:18081/, stop the tracked PID, and retry once on 18082 only if the port is occupied or curl returns an empty reply. Do not restart broad implementation unless validation proves the code state is invalid.",
	})
	repairJob := queue.Job{
		RepoID:         job.RepoID,
		Role:           job.Role,
		Trigger:        string(triggerJSON),
		IdempotencyKey: fmt.Sprintf("ticket-gate:%s:%s:%s", job.ID, job.RepoID, job.Role),
	}
	jobID, err := s.queue.Enqueue(ctx, repairJob)
	if err != nil {
		log.Error("serve: failed to enqueue ticket-gate repair job", "err", err)
		return
	}
	log.Info("serve: ticket-gate repair job enqueued", "repair_job_id", jobID)
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"job_id":     jobID,
			"role":       job.Role,
			"repo":       job.RepoID,
			"source_job": job.ID,
			"reason":     jobErr.Error(),
		})
		s.dash.BroadcastEvent("ticket_gate_repair_enqueued", string(payload))
	}
}

func (s *Server) enqueueProductContinuation(ctx context.Context, job *queue.Job, jobErr error) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	triggerJSON, _ := json.Marshal(map[string]string{
		"type":         "product_continuation",
		"source_job":   job.ID,
		"reason":       jobErr.Error(),
		"repair_scope": "continue_active_product_ticket",
		"ask":          "Continue the active product ticket from its current repository state. Inspect the in-progress ticket, latest commits, and dirty files first. Fix only the remaining product, build, validation, or ticket-lifecycle gaps. For browser-framework work, replace placeholder build scripts with a command that can fail on broken source, fix module-loading/import errors, run the build and a product smoke that proves real UI state, update ticket evidence, move the ticket to done, commit, push if a remote exists, then record job_disposition_record with next_need qa_review. Do not create intervention-debt tickets or restart planning.",
	})
	continuationJob := queue.Job{
		RepoID:         job.RepoID,
		Role:           job.Role,
		Trigger:        string(triggerJSON),
		IdempotencyKey: fmt.Sprintf("product-continuation:%s:%s:%s", job.ID, job.RepoID, job.Role),
	}
	jobID, err := s.queue.Enqueue(ctx, continuationJob)
	if err != nil {
		log.Error("serve: failed to enqueue product continuation job", "err", err)
		return
	}
	log.Info("serve: product continuation job enqueued", "continuation_job_id", jobID)
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"job_id":     jobID,
			"role":       job.Role,
			"repo":       job.RepoID,
			"source_job": job.ID,
		})
		s.dash.BroadcastEvent("product_continuation_enqueued", string(payload))
	}
}

// convergenceRetryFingerprintWindow bounds the automatic operator-retry
// routing edge (AD-289): once an automatic convergence retry for a failure
// fingerprint has itself failed inside this window, further failures with the
// same fingerprint escalate to the operator instead of retrying again.
const convergenceRetryFingerprintWindow = 24 * time.Hour

// isConvergenceRuntimeFailure reports whether a runtime failure category is a
// convergence failure: the session exhausted its loop budget (max_turns) or
// repeated itself (circle_detected), where a fresh bounded session against the
// same repository state is a legitimate recovery — the manual operator
// run-role retries observed in live replays were exactly this action.
// Environment failures (model_unavailable, context_overflow, llm_unreachable,
// inference_crash, ...) are excluded because redispatching the same state
// reproduces the failure deterministically.
func isConvergenceRuntimeFailure(cat telemetry.FailureCategory) bool {
	return cat == telemetry.CategoryMaxTurns || cat == telemetry.CategoryCircleDetected
}

// isAutomaticRecoveryTrigger reports whether the job was itself dispatched by
// a bounded automatic recovery edge, in which case another automatic retry is
// out of budget and the failure must escalate to the operator.
func isAutomaticRecoveryTrigger(raw string) bool {
	var trigger map[string]string
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		return false
	}
	switch trigger["type"] {
	case "convergence_retry", "product_continuation", "ticket_gate_repair", "auto_recover":
		return true
	default:
		return false
	}
}

func convergenceFailureFingerprint(repoID, role string, cat telemetry.FailureCategory) string {
	return fmt.Sprintf("%s:%s:%s", repoID, role, cat)
}

// routeConvergenceFailure implements the AD-286 operator-retry-routing
// transition (AD-289): a runtime convergence failure gets at most one
// automatic same-role retry per failure fingerprint; past that budget the
// failure escalates to the operator with a recorded disposition naming the
// exact retry command instead of halting silently.
func (s *Server) routeConvergenceFailure(ctx context.Context, job *queue.Job, cat telemetry.FailureCategory, jobErr error, traceID string) {
	fingerprint := convergenceFailureFingerprint(job.RepoID, job.Role, cat)
	if isAutomaticRecoveryTrigger(job.Trigger) {
		s.recordConvergenceEscalation(ctx, job, cat, fingerprint, jobErr, traceID,
			"the failed job was already a bounded automatic recovery job")
		return
	}
	if s.convergenceRetryAlreadyFailed(ctx, job.RepoID, job.Role, fingerprint) {
		s.recordConvergenceEscalation(ctx, job, cat, fingerprint, jobErr, traceID,
			"an automatic convergence retry for this failure fingerprint already failed inside the retry window")
		return
	}
	s.enqueueConvergenceRetry(ctx, job, cat, fingerprint, jobErr)
}

// convergenceRetryAlreadyFailed reports whether an automatic convergence
// retry carrying the same failure fingerprint already failed inside the
// fingerprint window, which exhausts the automatic budget for that
// fingerprint. A successful retry leaves no failed retry row, so distinct
// later failures of the same role still earn one fresh automatic retry.
func (s *Server) convergenceRetryAlreadyFailed(ctx context.Context, repoID, role, fingerprint string) bool {
	if s == nil || s.db == nil {
		return false
	}
	cutoff := time.Now().UTC().Add(-convergenceRetryFingerprintWindow).Unix()
	var count int
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM jobs
WHERE repo_id = ?
  AND role = ?
  AND status = 'failed'
  AND trigger_payload LIKE '%"type":"convergence_retry"%'
  AND trigger_payload LIKE ?
  AND completed_at >= ?`,
		repoID, role, `%"fingerprint":"`+fingerprint+`"%`, cutoff).Scan(&count)
	if err != nil {
		slog.Warn("serve: convergence retry fingerprint lookup failed", "repo_id", repoID, "role", role, "err", err)
		return false
	}
	return count > 0
}

func convergenceRetryAsk(role string) string {
	switch role {
	case "qa", "security", "dogfood":
		return "Resume the interrupted review from the current repository state after a convergence failure. Inspect the dispatch-named work and the validation evidence already recorded first, gather only the bounded validation evidence that is still missing, then record job_disposition_record with an honest terminal verdict. If review evidence is already clean, record the terminal disposition immediately instead of re-reading evidence."
	default:
		return "Resume the interrupted work from the current repository state after a convergence failure. Inspect existing commits, tickets, and recorded evidence first, complete only the remaining lifecycle steps, then record job_disposition_record. Do not restart completed work or create intervention-debt tickets."
	}
}

func (s *Server) enqueueConvergenceRetry(ctx context.Context, job *queue.Job, cat telemetry.FailureCategory, fingerprint string, jobErr error) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	triggerJSON, _ := json.Marshal(map[string]string{
		"type":             "convergence_retry",
		"source_job":       job.ID,
		"reason":           jobErr.Error(),
		"failure_category": string(cat),
		"fingerprint":      fingerprint,
		"repair_scope":     "resume_role_from_current_state",
		"ask":              convergenceRetryAsk(job.Role),
	})
	retryJob := queue.Job{
		RepoID:         job.RepoID,
		Role:           job.Role,
		Trigger:        string(triggerJSON),
		IdempotencyKey: fmt.Sprintf("convergence-retry:%s:%s:%s", job.ID, job.RepoID, job.Role),
	}
	jobID, err := s.queue.Enqueue(ctx, retryJob)
	if err != nil {
		log.Error("serve: failed to enqueue convergence retry job", "err", err)
		return
	}
	log.Info("serve: convergence retry job enqueued after runtime convergence failure",
		"retry_job_id", jobID,
		"category", string(cat),
		"fingerprint", fingerprint,
	)
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"job_id":      jobID,
			"role":        job.Role,
			"repo":        job.RepoID,
			"source_job":  job.ID,
			"category":    string(cat),
			"fingerprint": fingerprint,
		})
		s.dash.BroadcastEvent("convergence_retry_enqueued", string(payload))
	}
}

// recordConvergenceEscalation replaces the plain failed disposition with a
// blocked/operator_retry disposition naming the exhausted automatic budget
// and the exact operator retry command, so the halt is an operator-visible
// recorded blocker rather than a silent log line.
func (s *Server) recordConvergenceEscalation(ctx context.Context, job *queue.Job, cat telemetry.FailureCategory, fingerprint string, jobErr error, traceID, why string) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	reason := fmt.Sprintf(
		"convergence failure %s exhausted its automatic retry budget (%s); operator retry required: POST /api/run-role {\"repo_id\":%q,\"role\":%q} or mars-harness run %s --repo <repo-path>; fingerprint=%s; last error: %s",
		cat, why, job.RepoID, job.Role, job.Role, fingerprint, jobErr.Error(),
	)
	log.Warn("serve: convergence failure escalated to operator after automatic retry budget",
		"category", string(cat),
		"fingerprint", fingerprint,
		"why", why,
	)
	if s.orgStore != nil {
		if err := s.orgStore.RecordDisposition(ctx, orgstate.Disposition{
			JobID:    job.ID,
			RepoID:   job.RepoID,
			Role:     job.Role,
			Status:   "blocked",
			NextNeed: "operator_retry",
			Reason:   reason,
			TraceID:  traceID,
		}); err != nil {
			log.Error("serve: failed to record convergence escalation disposition", "err", err)
		}
	}
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"job_id":      job.ID,
			"role":        job.Role,
			"repo":        job.RepoID,
			"category":    string(cat),
			"fingerprint": fingerprint,
			"reason":      reason,
		})
		s.dash.BroadcastEvent("convergence_escalated", string(payload))
	}
}

func (s *Server) recordGuardrailLoopEscalation(ctx context.Context, job *queue.Job, evt telemetry.Event, jobErr error, traceID string) {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	reason := fmt.Sprintf(
		"guardrail_loop reached threshold without same-job remediation; policy evidence: %s; last error: %s; operator retry required after correcting the blocked action or role guidance",
		evt.Message, jobErr.Error(),
	)
	log.Warn("serve: guardrail loop escalated to blocked operator retry",
		"event_id", evt.ID,
		"category", string(evt.Category),
	)
	if s.orgStore != nil {
		if err := s.orgStore.RecordDisposition(ctx, orgstate.Disposition{
			JobID:    job.ID,
			RepoID:   job.RepoID,
			Role:     job.Role,
			Status:   "blocked",
			NextNeed: "operator_retry",
			Reason:   reason,
			TraceID:  traceID,
		}); err != nil {
			log.Error("serve: failed to record guardrail loop escalation disposition", "err", err)
		}
	}
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"job_id":   job.ID,
			"role":     job.Role,
			"repo":     job.RepoID,
			"event_id": evt.ID,
			"reason":   reason,
		})
		s.dash.BroadcastEvent("guardrail_loop_escalated", string(payload))
	}
}

func (s *Server) dispatchFallbackAfterOrchestratorFailure(ctx context.Context, job *queue.Job, rec *RepoRecord, manifest *bundle.Manifest, jobErr error) bool {
	log := slog.With("job_id", job.ID, "role", job.Role, "repo_id", job.RepoID)
	var trigger dispatchTriggerPayload
	if err := json.Unmarshal([]byte(job.Trigger), &trigger); err != nil {
		log.Warn("serve: failed Orchestrator trigger is not dispatch JSON; stopping dispatch recovery", "err", err)
		s.recordStoppedOrchestratorFallback(ctx, job, rec, orgstate.Disposition{}, "failed Orchestrator trigger was not usable dispatch JSON", jobErr)
		return true
	}
	if trigger.Type != "dispatch" ||
		strings.TrimSpace(trigger.SourceRole) == "" ||
		strings.TrimSpace(trigger.SourceJob) == "" ||
		strings.EqualFold(strings.TrimSpace(trigger.SourceRole), "orchestrator") {
		log.Warn("serve: failed Orchestrator trigger lacks a usable source handoff; stopping dispatch recovery",
			"source_role", trigger.SourceRole,
			"source_job", trigger.SourceJob,
		)
		s.recordStoppedOrchestratorFallback(ctx, job, rec, orgstate.Disposition{}, "failed Orchestrator trigger lacked a non-Orchestrator source handoff", jobErr)
		return true
	}

	sourceDisposition := orgstate.Disposition{
		JobID:         strings.TrimSpace(trigger.SourceJob),
		RepoID:        job.RepoID,
		Role:          strings.TrimSpace(trigger.SourceRole),
		Status:        strings.TrimSpace(trigger.SourceDisposition.Status),
		NextNeed:      strings.TrimSpace(trigger.SourceDisposition.NextNeed),
		SuggestedRole: strings.TrimSpace(trigger.SourceDisposition.SuggestedRole),
		TicketID:      strings.TrimSpace(trigger.SourceDisposition.TicketID),
		Reason:        strings.TrimSpace(trigger.SourceDisposition.Reason),
		EvidenceLinks: append([]string{}, trigger.SourceDisposition.EvidenceLinks...),
		TraceID:       strings.TrimSpace(trigger.SourceDisposition.TraceID),
		Handoff:       trigger.SourceDisposition.Handoff,
		Feedback:      trigger.SourceDisposition.Feedback,
	}
	if sourceDisposition.Status == "" {
		sourceDisposition.Status = "completed"
	}

	hasRoutingSignal := sourceDisposition.NextNeed != "" ||
		sourceDisposition.SuggestedRole != "" ||
		strings.TrimSpace(sourceDisposition.Handoff.TargetRole) != "" ||
		strings.TrimSpace(sourceDisposition.Feedback.ForRole) != ""
	fallbackStatus := "completed"
	if !hasRoutingSignal {
		switch sourceDisposition.Status {
		case "changes_requested", "in_review":
			fallbackStatus = sourceDisposition.Status
		default:
			s.recordStoppedOrchestratorFallback(ctx, job, rec, sourceDisposition, "failed Orchestrator source handoff had no deterministic routing signal", jobErr)
			return true
		}
	}

	fallbackDisposition := sourceDisposition
	fallbackDisposition.JobID = job.ID
	fallbackDisposition.Role = "orchestrator"
	fallbackDisposition.Status = fallbackStatus
	fallbackDisposition.Reason = strings.TrimSpace(fmt.Sprintf("deterministic fallback after Orchestrator failed: %s; source role %s said: %s", jobErr.Error(), sourceDisposition.Role, sourceDisposition.Reason))
	fallbackDisposition.TraceID = s.latestTraceID(ctx, job.ID)
	if fallbackDisposition.TraceID == "" {
		fallbackDisposition.TraceID = sourceDisposition.TraceID
	}

	snap, snapErr := snapshotTickets(rec.Path)
	ticketHash := ""
	if snapErr == nil {
		ticketHash = snap.routingHash()
	} else {
		log.Warn("serve: dispatch fallback ticket snapshot failed", "err", snapErr)
	}

	recent, err := s.orgStore.RecentDecisions(ctx, job.RepoID, 20)
	if err != nil {
		log.Warn("serve: dispatch fallback recent decisions unavailable", "err", err)
	}

	decision, err := orchestration.Decide(orchestration.Input{
		Disposition:       fallbackDisposition,
		Manifest:          manifest,
		RecentDecisions:   recent,
		TicketStateHash:   ticketHash,
		SourceDisposition: &sourceDisposition,
	})
	if err != nil {
		log.Warn("serve: dispatch fallback decision failed", "err", err)
		return false
	}
	decision.Reason = strings.TrimSpace(decision.Reason + "; deterministic fallback after failed Orchestrator")
	if snapErr == nil {
		decision = enforceEngineerTicketPrerequisite(decision, snap, manifest, &sourceDisposition)
	}
	if strings.EqualFold(strings.TrimSpace(decision.NextRole), "orchestrator") {
		log.Warn("serve: dispatch fallback refused recursive Orchestrator route",
			"reason", decision.Reason,
		)
		decision.NextRole = ""
		decision.DecisionKind = "deterministic_fallback"
		decision.StopReason = "failed Orchestrator recovery stopped without recursive dispatch"
		decision.Reason += "; refused recursive Orchestrator route"
	}
	decision, err = s.orgStore.RecordDecision(ctx, decision)
	if err != nil {
		log.Warn("serve: record dispatch fallback decision failed", "err", err)
		return false
	}

	if s.dash != nil {
		decisionPayload, _ := json.Marshal(decision)
		s.dash.BroadcastEvent("orchestration_decision", string(decisionPayload))
	}

	if strings.TrimSpace(decision.NextRole) == "" {
		log.Info("serve: dispatch fallback stopped", "reason", decision.StopReason)
		return true
	}

	triggerJSON, err := json.Marshal(newDispatchTriggerPayloadForSource(trigger.SourceRole, trigger.SourceJob, decision, sourceDisposition))
	if err != nil {
		log.Error("serve: failed to marshal dispatch fallback trigger", "target_role", decision.NextRole, "err", err)
		return false
	}
	dispatchJob := queue.Job{
		RepoID:         job.RepoID,
		Role:           decision.NextRole,
		Trigger:        string(triggerJSON),
		IdempotencyKey: fmt.Sprintf("dispatch:%s:%s:%s:fallback", job.ID, job.RepoID, decision.NextRole),
	}
	jobID, err := s.queue.Enqueue(ctx, dispatchJob)
	if err != nil {
		log.Error("serve: failed to enqueue dispatch fallback job", "target_role", decision.NextRole, "err", err)
		return false
	}
	log.Info("serve: dispatch fallback job enqueued", "target_role", decision.NextRole, "dispatch_job_id", jobID)
	if s.dash != nil {
		payload, _ := json.Marshal(map[string]string{
			"decision_id": decision.ID,
			"job_id":      jobID,
			"role":        decision.NextRole,
			"repo":        job.RepoID,
			"reason":      decision.Reason,
		})
		s.dash.BroadcastEvent("dispatch_enqueued", string(payload))
	}
	return true
}

func (s *Server) recordStoppedOrchestratorFallback(ctx context.Context, job *queue.Job, rec *RepoRecord, source orgstate.Disposition, reason string, jobErr error) {
	if s == nil || s.orgStore == nil {
		return
	}
	ticketHash := ""
	if rec != nil {
		if snap, err := snapshotTickets(rec.Path); err == nil {
			ticketHash = snap.routingHash()
		}
	}
	decision := orgstate.Decision{
		JobID:           job.ID,
		RepoID:          job.RepoID,
		SourceRole:      "orchestrator",
		TicketID:        source.TicketID,
		NextNeed:        source.NextNeed,
		DecisionKind:    "deterministic_fallback",
		Reason:          strings.TrimSpace(fmt.Sprintf("%s after error: %s", reason, jobErr.Error())),
		StopReason:      "failed Orchestrator recovery stopped without recursive dispatch",
		TicketStateHash: ticketHash,
	}
	if _, err := s.orgStore.RecordDecision(ctx, decision); err != nil {
		slog.Warn("serve: record stopped Orchestrator fallback decision failed",
			"job_id", job.ID,
			"repo_id", job.RepoID,
			"err", err,
		)
	}
}

func deterministicContainmentFailure(cat telemetry.FailureCategory, msg string) bool {
	if cat == telemetry.CategoryWorkspaceHygiene {
		return true
	}
	if cat != telemetry.CategoryGuardrailBlock {
		return false
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "dirty worktree containment") ||
		strings.Contains(lower, "blast radius exceeded")
}

func dispatchRuntimeFailureStops(cat telemetry.FailureCategory) bool {
	switch cat {
	case telemetry.CategoryContextOverflow,
		telemetry.CategoryLLMUnreachable,
		telemetry.CategoryInferenceCrash,
		telemetry.CategoryInferencePortConflict,
		telemetry.CategoryModelUnavailable,
		telemetry.CategoryToolTimeout,
		telemetry.CategoryCircleDetected,
		telemetry.CategoryMaxTurns,
		telemetry.CategoryBudgetExceeded,
		telemetry.CategoryManifestError,
		telemetry.CategoryGuardrailBlock,
		telemetry.CategoryDispatchProtocol,
		telemetry.CategoryManualStop,
		telemetry.CategoryUnknown:
		return true
	default:
		return false
	}
}

func isTicketGateRepairTrigger(raw string) bool {
	var trigger map[string]string
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		return false
	}
	return trigger["type"] == "ticket_gate_repair"
}

func isProductContinuationTrigger(raw string) bool {
	var trigger map[string]string
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		return false
	}
	return trigger["type"] == "product_continuation"
}

func engineerHasContinuableProductTicket(repoPath string) bool {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return false
	}
	tickets, err := ticketstate.ListStatus(repoPath, ticketstate.StatusInProgress)
	if err != nil {
		return false
	}
	for _, ticket := range tickets {
		kind := strings.ToLower(strings.TrimSpace(ticket.Kind))
		workType := strings.ToLower(strings.TrimSpace(ticket.WorkType))
		if kind == "intervention-debt" || workType == "intervention-debt" {
			continue
		}
		return true
	}
	return false
}

func (s *Server) jobHadPolicyBlock(jobID string) bool {
	if s == nil || s.telemetry == nil || strings.TrimSpace(jobID) == "" {
		return false
	}
	for _, evt := range s.telemetry.Events() {
		if evt.JobID != jobID {
			continue
		}
		if evt.Category == telemetry.CategoryGuardrailBlock {
			return true
		}
		msg := strings.ToLower(evt.Message)
		if strings.Contains(msg, "tool policy blocked") || strings.Contains(msg, "blast radius exceeded") {
			return true
		}
	}
	return false
}

func (s *Server) jobHadTicketLifecyclePolicyBlock(jobID string) bool {
	if s == nil || s.telemetry == nil || strings.TrimSpace(jobID) == "" {
		return false
	}
	for _, evt := range s.telemetry.Events() {
		if evt.JobID != jobID {
			continue
		}
		msg := strings.ToLower(evt.Message)
		if !strings.Contains(msg, "policy:") {
			continue
		}
		if strings.Contains(msg, "cannot record a successful disposition") ||
			strings.Contains(msg, "cannot populate ticket evidence_links") ||
			strings.Contains(msg, "cannot move to docs/tickets/done") ||
			strings.Contains(msg, "cannot move a product ticket to docs/tickets/done") ||
			(strings.Contains(msg, "feature ticket") && strings.Contains(msg, "evidence_links")) {
			return true
		}
	}
	return false
}

func (s *Server) latestUnremediatedGuardrailLoop(jobID string) (telemetry.Event, bool) {
	if s == nil || s.telemetry == nil || strings.TrimSpace(jobID) == "" {
		return telemetry.Event{}, false
	}
	events := s.telemetry.Events()
	for i := len(events) - 1; i >= 0; i-- {
		evt := events[i]
		if evt.JobID == jobID && evt.Category == telemetry.CategoryGuardrailLoop && !evt.Remedied {
			return evt, true
		}
	}
	return telemetry.Event{}, false
}

func (s *Server) latestTraceID(ctx context.Context, jobID string) string {
	if s.traceStore == nil || strings.TrimSpace(jobID) == "" {
		return ""
	}
	rec, err := s.traceStore.GetLatestByJobID(ctx, jobID)
	if err != nil {
		slog.Warn("serve: trace lookup failed for intervention-debt evidence", "job_id", jobID, "err", err)
		return ""
	}
	if rec == nil {
		return ""
	}
	return rec.TraceID
}

func interventionDebtSignalKindForCategory(category telemetry.FailureCategory) string {
	return interventionDebtSignalKind(interventionDebtSignal{Category: category})
}

func isAutoRecoverTrigger(trigger string) bool {
	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(trigger), &payload); err != nil {
		return false
	}
	return payload.Type == "auto_recover"
}

// handleRemediation is the telemetry remediation callback. It executes
// the auto-fix action determined by the classifier.
func (s *Server) handleRemediation(evt telemetry.Event) {
	log := slog.With("event_id", evt.ID, "job_id", evt.JobID, "role", evt.Role, "action", evt.Action)
	traceID := s.latestTraceID(context.Background(), evt.JobID)
	plan := s.planEventRemediation(context.Background(), evt, traceID, false)
	if remediationPlanHasExecutableReadyAttempt(plan) {
		log.Info("telemetry: generic remediation deferred because executable deterministic recipe is ready",
			"attempts", len(plan.Attempts),
		)
		return
	}

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

func (s *Server) buildPipelineView() dashboard.PipelineView {
	repos, err := s.repos.List(context.Background())
	if err != nil || len(repos) == 0 {
		return dashboard.PipelineView{Mode: "none", Description: "No repositories registered."}
	}

	manifest, err := bundle.Load(repos[0].Path)
	if err != nil {
		return dashboard.PipelineView{Mode: "unknown", Description: "Unable to load manifest."}
	}

	if manifest.DispatchMode() {
		nodes := dashboardRoleNodes(manifest, true)
		return dashboard.PipelineView{
			Mode:        "dispatch",
			Description: "Each role records a disposition; deterministic handoffs route directly and Orchestrator handles ambiguous follow-up.",
			Nodes:       nodes,
		}
	}

	nodes := s.buildPipelineChain()
	return dashboard.PipelineView{
		Mode:        "legacy",
		Description: "Legacy mode follows manifest then/idle_then chains.",
		Nodes:       nodes,
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

func dashboardRoleNodes(manifest *bundle.Manifest, excludeOrchestrator bool) []dashboard.ChainNode {
	if manifest == nil {
		return nil
	}
	names := make([]string, 0, len(manifest.Roles))
	for name := range manifest.Roles {
		if excludeOrchestrator && name == "orchestrator" {
			continue
		}
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		left := roleDashboardOrder(names[i])
		right := roleDashboardOrder(names[j])
		if left == right {
			return names[i] < names[j]
		}
		return left < right
	})
	nodes := make([]dashboard.ChainNode, 0, len(names))
	for _, name := range names {
		role := manifest.Roles[name]
		nodes = append(nodes, dashboard.ChainNode{Name: name, Domain: role.Domain, Mode: role.Mode})
	}
	return nodes
}

func roleDashboardOrder(role string) int {
	order := map[string]int{
		"ceo":                10,
		"cto-weekly":         20,
		"coo":                30,
		"engineer":           40,
		"qa":                 50,
		"security":           60,
		"dependency-manager": 70,
		"release-manager":    80,
		"dogfood":            90,
		"pipeline-fixer":     100,
		"janitor":            110,
		"orchestrator":       120,
	}
	if n, ok := order[role]; ok {
		return n
	}
	return 1000
}

func (s *Server) registerCronSchedules(repos []RepoRecord) {
	var schedules []scheduler.Schedule
	for _, repo := range repos {
		manifest, err := bundle.Load(repo.Path)
		if err != nil {
			slog.Warn("serve: skipping schedules for repo — manifest load failed",
				"repo_id", repo.ID, "err", err)
			continue
		}
		integrationCfg := s.integrationConfigForRepo(repo)
		slog.Info("serve: integrations profile loaded",
			"repo", repo.ID,
			"flow_profile", integrationCfg.FlowProfile,
		)

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
			if integrationCfg.SuppressesSchedule(roleName) {
				slog.Info("serve: cron schedule suppressed by integrations profile",
					"repo", repo.ID,
					"role", roleName,
					"cron", cron,
					"flow_profile", integrationCfg.FlowProfile,
				)
				continue
			}

			sched := scheduler.Schedule{
				Name:             fmt.Sprintf("%s:%s", repo.ID, roleName),
				RepoID:           repo.ID,
				Role:             roleName,
				Cron:             cron,
				Timezone:         "UTC",
				Trigger:          fmt.Sprintf(`{"type":"schedule","role":"%s"}`, roleName),
				PayloadMode:      "schedule",
				ConcurrencyGroup: fmt.Sprintf("schedule:%s:%s", repo.ID, roleName),
				DailyCap:         scheduleDailyCap(cron),
			}
			schedules = append(schedules, sched)
			slog.Info("serve: cron schedule prepared",
				"repo", repo.ID,
				"role", roleName,
				"cron", cron,
			)
		}
	}
	if err := s.scheduler.ReplaceSchedules(schedules); err != nil {
		slog.Warn("serve: failed to replace cron schedules", "err", err)
	}
}

func (s *Server) integrationConfigForRepo(repo RepoRecord) integrations.Config {
	cfg, err := integrations.Load(repo.Path)
	if err != nil {
		slog.Warn("serve: integrations config unavailable; using ceo-led defaults",
			"repo", repo.ID,
			"path", repo.Path,
			"err", err,
		)
		return integrations.Defaults()
	}
	return cfg
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

func scheduleDailyCap(cron string) int {
	if strings.TrimSpace(cron) == "" {
		return 0
	}
	return 24
}

// jobCount tracks how many jobs have completed since the last evolution check.
var jobCount int32

// checkEvolution runs pattern detection and triggers evolution reviews
// when recurring failures are detected or after enough jobs accumulate.
func (s *Server) checkEvolution(ctx context.Context, role, repoID string) {
	if s.telemetry == nil {
		return
	}

	atomic.AddInt32(&jobCount, 1)

	patterns := s.telemetry.DetectPatternsFromStore()
	proposals := telemetry.TriagePatterns(patterns)

	for i, p := range patterns {
		if !matchesTriageScope(p.RepoID, p.Role, repoID, role) {
			continue
		}
		proposal := proposals[i]
		if proposal.RepoID == "" {
			proposal.RepoID = repoID
		}
		window := p.Window
		if window == "" {
			window = "24h"
		}
		origin := interventionDebtOrigin{
			Kind:           "telemetry_pattern",
			EvidenceWindow: window,
		}
		if s.telemStore != nil {
			if evt, err := s.telemStore.LatestByRoleCategory(proposal.RepoID, proposal.Role, proposal.Category, time.Now().UTC().Add(-telemetry.PatternWindow)); err == nil {
				origin.Event = evt
			} else {
				slog.Warn("serve: latest telemetry evidence lookup failed", "role", proposal.Role, "category", proposal.Category, "err", err)
			}
		}
		s.recordInterventionDebtTicket(ctx, repoID, proposal, origin)

		if s.dash != nil {
			data, _ := json.Marshal(p)
			s.dash.BroadcastEvent("telemetry_pattern", string(data))
			proposalData, _ := json.Marshal(proposal)
			s.dash.BroadcastEvent("telemetry_triage", string(proposalData))
		}

		s.offerInterventionDebtEvolution(ctx, repoID, proposal, fmt.Sprintf("telemetry_%s_%s", proposal.Target, proposal.Category))
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
	if s.scoreStore == nil {
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

	s.recordInterventionDebtTicket(ctx, repoID, proposal, interventionDebtOrigin{
		Kind:           "score_snapshot",
		EvidenceWindow: fmt.Sprintf("%dd", sc.WindowDays),
		Score: &telemetry.ScoreSnapshot{
			Role:       role,
			RepoID:     repoID,
			Value:      sc.Value,
			SampleSize: sc.SampleSize,
			WindowDays: sc.WindowDays,
		},
	})

	if s.dash != nil {
		data, _ := json.Marshal(proposal)
		s.dash.BroadcastEvent("score_triage", string(data))
	}

	s.offerInterventionDebtEvolution(ctx, repoID, proposal, fmt.Sprintf("score_%s", proposal.Target))
}

func (s *Server) offerInterventionDebtEvolution(ctx context.Context, repoID string, proposal telemetry.ImprovementProposal, classification string) {
	if s.evoStore == nil {
		return
	}
	files := boundedEvolutionFiles(proposal.CandidateFiles)
	if proposal.Confidence < 0.65 || len(files) == 0 || proposal.Target == telemetry.TargetUnknown {
		slog.Info("serve: evolution review skipped (ticket only)",
			"role", proposal.Role,
			"target", proposal.Target,
			"confidence", proposal.Confidence,
			"candidate_files", len(files),
		)
		return
	}
	ok, reason := evolution.CanReview(s.evoStore, proposal.Role, evolution.DefaultReviewerConfig())
	if !ok {
		slog.Info("serve: evolution review skipped", "role", proposal.Role, "reason", reason)
		return
	}

	result := evolution.ReviewResult{
		Classification: classification,
		Suggestion:     proposal.Suggestion,
		FilesToModify:  files,
		Confidence:     proposal.Confidence,
	}
	if err := evolution.RecordEvolution(ctx, s.evoStore, proposal.Role, repoID, result); err != nil {
		slog.Error("serve: failed to record evolution", "role", proposal.Role, "err", err)
	}
}

func boundedEvolutionFiles(files []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, file := range files {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(file)))
		if clean == "." || clean == "" {
			continue
		}
		allowed := clean == ".harness/manifest.yaml" ||
			clean == ".harness/knowledge-routes.yaml" ||
			strings.HasPrefix(clean, ".harness/roles/") ||
			strings.HasPrefix(clean, ".harness/guardrails/") ||
			strings.HasPrefix(clean, ".harness/knowledge/")
		if !allowed {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func matchesTriageScope(patternRepoID, patternRole, repoID, role string) bool {
	if patternRepoID != "" && repoID != "" && patternRepoID != repoID {
		return false
	}
	if patternRole != "" && role != "" && patternRole != role {
		return false
	}
	return true
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
