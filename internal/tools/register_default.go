package tools

import (
	"fmt"
	"sync"
)

// RegisterBuiltinTools registers all built-in tools on r.
func RegisterBuiltinTools(r *Registry) error {
	if r == nil {
		return fmt.Errorf("tools: RegisterBuiltinTools registry is nil")
	}
	registrations := []func(*Registry) error{
		registerFileRead,
		registerFileWrite,
		registerFileSearch,
		registerGrep,
		registerShellExec,
		registerMarsHarnessCLI,
		registerGitTools,
		registerRecordDecision,
		registerTicketCreate,
		registerToolCreate,
	}
	for _, fn := range registrations {
		if err := fn(r); err != nil {
			return err
		}
	}
	return nil
}

var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
	defaultRegistryErr  error
)

// DefaultRegistry returns the process-wide registry with built-in tools (lazy init).
func DefaultRegistry() (*Registry, error) {
	defaultRegistryOnce.Do(func() {
		r := NewRegistry()
		defaultRegistryErr = RegisterBuiltinTools(r)
		if defaultRegistryErr != nil {
			return
		}
		defaultRegistry = r
	})
	return defaultRegistry, defaultRegistryErr
}
