package serve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	gh "github.com/greaveselliott/mars-harness/internal/github"
	"github.com/greaveselliott/mars-harness/internal/safety"
)

// Config controls the serve command.
type Config struct {
	WebhookAddr   string
	WebhookSecret string
	DBPath        string
	Concurrency   int
}

func (c Config) concurrency() int {
	if c.Concurrency > 0 {
		return c.Concurrency
	}
	return 2
}

// Server composes all subsystems for the serve command.
type Server struct {
	cfg    Config
	http   *http.Server
	mux    *http.ServeMux
	estop  *safety.EmergencyStop
	health atomic.Bool

	mu      sync.Mutex
	started bool
}

// New creates a Server wired with webhook, health, and safety subsystems.
func New(cfg Config) (*Server, error) {
	if cfg.WebhookAddr == "" {
		return nil, fmt.Errorf("serve: WebhookAddr is required — set it to e.g. \":8080\"")
	}

	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		estop: safety.NewEmergencyStop(),
	}

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

// Start begins serving. Blocks until the context is cancelled or the server fails.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("serve: server already started")
	}
	s.started = true
	s.mu.Unlock()

	ln, err := net.Listen("tcp", s.cfg.WebhookAddr)
	if err != nil {
		return fmt.Errorf("serve: failed to bind %s — check if the port is already in use: %w",
			s.cfg.WebhookAddr, err)
	}

	s.health.Store(true)
	slog.Info("serve: listening",
		"addr", ln.Addr().String(),
		"concurrency", s.cfg.concurrency())

	errCh := make(chan error, 1)
	go func() {
		if serveErr := s.http.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
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

// Stop gracefully shuts down the server with a 10-second deadline.
func (s *Server) Stop(ctx context.Context) error {
	s.health.Store(false)

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	slog.Info("serve: shutting down HTTP server")
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("serve: shutdown error — %w", err)
	}
	return nil
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

// handleEvent is invoked by the webhook handler for each valid GitHub event.
// Currently logs the event; queue integration will be wired in later milestones.
func (s *Server) handleEvent(event gh.Event) {
	slog.Info("serve: received event",
		"type", event.Type,
		"action", event.Action,
		"repo", event.Repo,
		"delivery_id", event.ID)
}
