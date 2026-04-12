package inference

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	healthPollInterval = 500 * time.Millisecond
	healthWaitTimeout  = 60 * time.Second
	termWait           = 10 * time.Second
	maxRestartAttempts = 5
)

// ServerConfig configures a single llama-server instance.
type ServerConfig struct {
	BinaryPath    string
	ModelPath     string
	Port          int
	ContextLength int
	GPULayers     int // -1 = auto, 0 = CPU only
	Threads       int // 0 = auto (omit -t)
}

// Server manages a single llama-server subprocess.
type Server struct {
	cfg    ServerConfig
	cmd    *exec.Cmd
	mu     sync.Mutex
	state  ServerState
	cancel context.CancelFunc // cancels health wait and supervisor backoff

	startMu sync.Mutex

	intentionalStop bool
	restarts        int

	supervisorWG sync.WaitGroup
}

type ServerState string

const (
	StateStopped  ServerState = "stopped"
	StateStarting ServerState = "starting"
	StateHealthy  ServerState = "healthy"
	StateFailed   ServerState = "failed"
)

// NewServer creates a server manager (does not start it).
func NewServer(cfg ServerConfig) *Server {
	return &Server{cfg: cfg, state: StateStopped}
}

// Start launches the subprocess and waits for health check.
// Returns error if server fails to become healthy within 60 seconds.
func (s *Server) Start(ctx context.Context) error {
	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	switch s.state {
	case StateHealthy:
		s.mu.Unlock()
		return nil
	case StateFailed:
		// allow retry after explicit failure
	}
	s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.intentionalStop = false
	s.restarts = 0
	if s.cancel != nil {
		s.cancel()
	}
	superCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.state = StateStarting
	s.mu.Unlock()

	slog.Info("inference server starting",
		"model", s.cfg.ModelPath,
		"port", s.cfg.Port,
		"ctx_size", effectiveContextLength(s.cfg.ContextLength),
		"gpu_layers", s.cfg.GPULayers,
	)

	cmd, err := s.startCmdLocked()
	if err != nil {
		cancel()
		s.mu.Lock()
		s.state = StateFailed
		s.mu.Unlock()
		slog.Error("inference server failed to launch process", "err", err)
		return err
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, healthWaitTimeout)
	defer waitCancel()
	if err := s.pollHealth(waitCtx, s.healthURL()); err != nil {
		_ = s.terminateProcessLocked(cmd)
		cancel()
		s.mu.Lock()
		s.state = StateFailed
		s.mu.Unlock()
		slog.Error("inference server failed health check", "err", err, "port", s.cfg.Port)
		return fmt.Errorf("inference: health check: %w", err)
	}

	s.mu.Lock()
	s.state = StateHealthy
	active := cmd
	s.mu.Unlock()

	slog.Info("inference server healthy", "port", s.cfg.Port)

	s.supervisorWG.Add(1)
	go func(cmdRef *exec.Cmd) {
		defer s.supervisorWG.Done()
		s.supervise(superCtx, cmdRef)
	}(active)

	return nil
}

func (s *Server) supervise(superCtx context.Context, initial *exec.Cmd) {
	cmd := initial
	for {
		err := cmd.Wait()
		if superCtx.Err() != nil {
			slog.Info("inference server supervisor stopped", "port", s.cfg.Port)
			return
		}

		s.mu.Lock()
		stopping := s.intentionalStop
		s.mu.Unlock()
		if stopping {
			slog.Info("inference server process exited after stop", "port", s.cfg.Port, "wait_err", err)
			return
		}

		s.mu.Lock()
		if s.restarts >= maxRestartAttempts {
			s.state = StateFailed
			s.mu.Unlock()
			slog.Error("inference server exceeded restart limit", "port", s.cfg.Port, "attempts", maxRestartAttempts)
			return
		}
		s.restarts++
		attempt := s.restarts
		s.state = StateStarting
		s.mu.Unlock()

		backoff := restartBackoff(attempt)
		slog.Warn("inference server process exited unexpectedly; restarting",
			"port", s.cfg.Port,
			"attempt", attempt,
			"backoff", backoff,
			"wait_err", err,
		)

		select {
		case <-superCtx.Done():
			return
		case <-time.After(backoff):
		}

		select {
		case <-superCtx.Done():
			return
		default:
		}
		s.mu.Lock()
		if s.intentionalStop {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		next, startErr := s.startCmdLocked()
		if startErr != nil {
			s.mu.Lock()
			s.state = StateFailed
			s.mu.Unlock()
			slog.Error("inference server restart failed to launch", "err", startErr, "port", s.cfg.Port)
			return
		}

		waitCtx, waitCancel := context.WithTimeout(superCtx, healthWaitTimeout)
		hErr := s.pollHealth(waitCtx, s.healthURL())
		waitCancel()
		if hErr != nil {
			_ = s.terminateProcessLocked(next)
			s.mu.Lock()
			s.state = StateFailed
			s.mu.Unlock()
			slog.Error("inference server restart failed health check", "err", hErr, "port", s.cfg.Port)
			return
		}

		s.mu.Lock()
		s.state = StateHealthy
		s.mu.Unlock()
		slog.Info("inference server healthy after restart", "port", s.cfg.Port, "attempt", attempt)

		cmd = next
	}
}

func restartBackoff(attempt int) time.Duration {
	// attempt is 1-based restart count: 1s, 2s, 4s, ... capped at 30s.
	d := time.Second
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= 30*time.Second {
			return 30 * time.Second
		}
	}
	if d > 30*time.Second {
		return 30 * time.Second
	}
	return d
}

// Stop gracefully shuts down: SIGTERM → wait 10s → SIGKILL.
func (s *Server) Stop() error {
	s.startMu.Lock()
	defer s.startMu.Unlock()

	s.mu.Lock()
	s.intentionalStop = true
	if s.cancel != nil {
		s.cancel()
	}
	cmd := s.cmd
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		s.supervisorWG.Wait()
		s.mu.Lock()
		s.state = StateStopped
		s.mu.Unlock()
		slog.Info("inference server stop noop (no process)", "port", s.cfg.Port)
		return nil
	}

	slog.Info("inference server stopping", "port", s.cfg.Port)

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		slog.Debug("inference server SIGTERM failed", "err", err, "port", s.cfg.Port)
	}

	deadline := time.After(termWait)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				slog.Debug("inference server SIGKILL failed", "err", err, "port", s.cfg.Port)
			} else {
				slog.Warn("inference server sent SIGKILL after grace period", "port", s.cfg.Port)
			}
			s.supervisorWG.Wait()
			s.mu.Lock()
			s.state = StateStopped
			s.restarts = 0
			s.cmd = nil
			s.mu.Unlock()
			slog.Info("inference server stopped", "port", s.cfg.Port)
			return nil
		case <-ticker.C:
			if !processAlive(cmd) {
				s.supervisorWG.Wait()
				s.mu.Lock()
				s.state = StateStopped
				s.restarts = 0
				s.cmd = nil
				s.mu.Unlock()
				slog.Info("inference server stopped", "port", s.cfg.Port)
				return nil
			}
		}
	}
}

func processAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func (s *Server) terminateProcessLocked(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	deadline := time.After(termWait)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			return cmd.Process.Kill()
		case <-ticker.C:
			if !processAlive(cmd) {
				return nil
			}
		}
	}
}

// State returns the current server state.
func (s *Server) State() ServerState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// BaseURL returns the OpenAI-compatible API base URL.
func (s *Server) BaseURL() string {
	return fmt.Sprintf("http://localhost:%d", s.cfg.Port)
}

func (s *Server) healthURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/health", s.cfg.Port)
}

// Healthy returns true if the server's /health endpoint returns 200.
func (s *Server) Healthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.healthURL(), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *Server) pollHealth(ctx context.Context, url string) error {
	t := time.NewTicker(healthPollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				slog.Debug("inference server health poll failed", "err", err, "url", url)
				continue
			}
			code := resp.StatusCode
			_ = resp.Body.Close()
			if code == http.StatusOK {
				return nil
			}
			slog.Debug("inference server health not ready", "status", code, "url", url)
		}
	}
}

func (s *Server) startCmdLocked() (*exec.Cmd, error) {
	args := llamaServerArgs(s.cfg)
	cmd := exec.Command(s.cfg.BinaryPath, args...)

	logPath := s.logFilePath()
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
			f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err == nil {
				cmd.Stdout = f
				cmd.Stderr = f
				slog.Debug("inference server logs", "path", logPath)
			}
		}
	}
	if cmd.Stdout == nil {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	if err := cmd.Start(); err != nil {
		s.mu.Lock()
		s.cmd = nil
		s.mu.Unlock()
		return nil, err
	}
	return cmd, nil
}

func (s *Server) logFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mars-harness", "logs",
		fmt.Sprintf("llama-server-%d.log", s.cfg.Port))
}

func effectiveContextLength(n int) int {
	if n <= 0 {
		return 4096
	}
	return n
}

func llamaServerArgs(cfg ServerConfig) []string {
	ctxLen := effectiveContextLength(cfg.ContextLength)
	args := []string{
		"--model", cfg.ModelPath,
		"--port", strconv.Itoa(cfg.Port),
		"--ctx-size", strconv.Itoa(ctxLen),
		"--n-gpu-layers", strconv.Itoa(cfg.GPULayers),
	}
	if cfg.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(cfg.Threads))
	}
	return args
}
