package safety

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// EmergencyStop halts all running jobs and performs cleanup.
type EmergencyStop struct {
	mu     sync.Mutex
	onStop []func(ctx context.Context) error
}

// NewEmergencyStop creates an EmergencyStop with no registered handlers.
func NewEmergencyStop() *EmergencyStop {
	return &EmergencyStop{}
}

// Register adds a cleanup function that will be called on Execute.
func (es *EmergencyStop) Register(fn func(ctx context.Context) error) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.onStop = append(es.onStop, fn)
}

// Execute runs all registered stop handlers and collects errors.
// All handlers are called regardless of individual failures.
func (es *EmergencyStop) Execute(ctx context.Context) []error {
	es.mu.Lock()
	handlers := make([]func(ctx context.Context) error, len(es.onStop))
	copy(handlers, es.onStop)
	es.mu.Unlock()

	slog.Warn("emergency stop: executing", "handlers", len(handlers))

	var errs []error
	for i, fn := range handlers {
		if err := fn(ctx); err != nil {
			slog.Error("emergency stop: handler failed",
				"index", i, "error", err)
			errs = append(errs, fmt.Errorf("handler %d: %w", i, err))
		}
	}

	slog.Info("emergency stop: complete",
		"total", len(handlers), "errors", len(errs))
	return errs
}
