/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package telemetry

import (
	"log/slog"
	"time"
)

// Pattern represents a recurring failure for a specific role+category.
type Pattern struct {
	RepoID    string          `json:"repo_id,omitempty"`
	Role      string          `json:"role"`
	Category  FailureCategory `json:"category"`
	Count     int             `json:"count"`
	Window    string          `json:"window"`
	FirstSeen time.Time       `json:"first_seen"`
	LastSeen  time.Time       `json:"last_seen"`
}

// PatternThreshold is the number of same role+category events within the
// detection window that triggers a pattern alert.
const PatternThreshold = 3

// PatternWindow is the time window for recurring pattern detection.
const PatternWindow = 24 * time.Hour

// DetectPatterns scans the in-memory event buffer for recurring failures.
// A pattern is flagged when the same (role, category) appears >= threshold
// times within the window.
func (c *Collector) DetectPatterns() []Pattern {
	c.mu.RLock()
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	cutoff := time.Now().UTC().Add(-PatternWindow)

	type key struct{ repoID, role, cat string }
	buckets := map[key][]Event{}

	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		k := key{e.RepoID, e.Role, string(e.Category)}
		buckets[k] = append(buckets[k], e)
	}

	var patterns []Pattern
	for k, evts := range buckets {
		if len(evts) < PatternThreshold {
			continue
		}
		p := Pattern{
			RepoID:    k.repoID,
			Role:      k.role,
			Category:  FailureCategory(k.cat),
			Count:     len(evts),
			Window:    "24h",
			FirstSeen: evts[0].Timestamp,
			LastSeen:  evts[len(evts)-1].Timestamp,
		}
		patterns = append(patterns, p)
		slog.Warn("telemetry: recurring failure pattern detected",
			"role", p.Role,
			"category", string(p.Category),
			"count", p.Count,
		)
	}
	return patterns
}

// DetectPatternsFromStore uses the persistent store for pattern detection.
// Falls back to in-memory if the store is nil.
func (c *Collector) DetectPatternsFromStore() []Pattern {
	c.mu.RLock()
	store := c.store
	c.mu.RUnlock()

	if store == nil {
		return c.DetectPatterns()
	}

	since := time.Now().UTC().Add(-PatternWindow)
	counts, err := store.RoleCategoryCountsSince(since)
	if err != nil {
		slog.Warn("telemetry: pattern detection query failed", "err", err)
		return c.DetectPatterns()
	}
	var patterns []Pattern

	for _, rc := range counts {
		count := rc.DistinctJobs
		if count <= 0 {
			count = rc.Count
		}
		if count >= PatternThreshold {
			patterns = append(patterns, Pattern{
				RepoID:    rc.RepoID,
				Role:      rc.Role,
				Category:  rc.Category,
				Count:     count,
				Window:    "24h",
				FirstSeen: rc.FirstSeen,
				LastSeen:  rc.LastSeen,
			})
			slog.Warn("telemetry: recurring failure pattern detected (store)",
				"repo_id", rc.RepoID,
				"role", rc.Role,
				"category", string(rc.Category),
				"count", count,
			)
		}
	}
	return patterns
}
