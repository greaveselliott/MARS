package telemetry

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const maxEvents = 500

// Broadcaster pushes SSE events to connected dashboard clients.
type Broadcaster interface {
	BroadcastEvent(eventType, data string)
}

// Collector records telemetry events, maintains a ring buffer of recent
// history, and broadcasts failures to the dashboard.
type Collector struct {
	mu     sync.RWMutex
	events []Event
	dash   Broadcaster

	onRemediate func(Event) // called when a remediation action is taken
}

// NewCollector creates a collector. dash may be nil (events are still recorded).
func NewCollector(dash Broadcaster) *Collector {
	return &Collector{
		events: make([]Event, 0, maxEvents),
		dash:   dash,
	}
}

// SetDashboard wires the broadcaster after construction (the dashboard
// may not exist at collector creation time).
func (c *Collector) SetDashboard(dash Broadcaster) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dash = dash
}

// SetRemediator registers a callback invoked when a remediation action
// is determined. The server uses this to trigger retries/restarts.
func (c *Collector) SetRemediator(fn func(Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onRemediate = fn
}

// Record classifies an error, stores the event, broadcasts it, and
// triggers remediation if the category is retryable.
func (c *Collector) Record(jobID, repoID, role, errMsg string) Event {
	category := Classify(errMsg)

	evt := Event{
		ID:        newEventID(),
		Timestamp: time.Now().UTC(),
		JobID:     jobID,
		RepoID:    repoID,
		Role:      role,
		Category:  category,
		Message:   errMsg,
	}

	action := Remediate(category)
	if action != ActionNone {
		evt.Remedied = true
		evt.Action = string(action)
	}

	c.mu.Lock()
	if len(c.events) >= maxEvents {
		c.events = c.events[1:]
	}
	c.events = append(c.events, evt)
	dash := c.dash
	onRemediate := c.onRemediate
	c.mu.Unlock()

	slog.Info("telemetry: event recorded",
		"event_id", evt.ID,
		"job_id", jobID,
		"role", role,
		"category", string(category),
		"remedied", evt.Remedied,
		"action", evt.Action,
	)

	if dash != nil {
		data, _ := json.Marshal(evt)
		dash.BroadcastEvent("telemetry", string(data))
	}

	if evt.Remedied && onRemediate != nil {
		onRemediate(evt)
	}

	return evt
}

// Events returns a copy of recent events (newest last).
func (c *Collector) Events() []Event {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Event, len(c.events))
	copy(out, c.events)
	return out
}

// Stats returns per-category failure counts from the event buffer.
func (c *Collector) Stats() map[FailureCategory]int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	counts := map[FailureCategory]int{}
	for _, e := range c.events {
		counts[e.Category]++
	}
	return counts
}

func newEventID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("evt-%x", b)
}
