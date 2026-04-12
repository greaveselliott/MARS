package telemetry

import (
	"log/slog"
	"time"
)

// Pattern represents a recurring failure for a specific role+category.
type Pattern struct {
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

	type key struct{ role, cat string }
	buckets := map[key][]Event{}

	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		k := key{e.Role, string(e.Category)}
		buckets[k] = append(buckets[k], e)
	}

	var patterns []Pattern
	for k, evts := range buckets {
		if len(evts) < PatternThreshold {
			continue
		}
		p := Pattern{
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

	roles := c.uniqueRoles()
	categories := []FailureCategory{
		CategoryContextOverflow, CategoryLLMUnreachable,
		CategoryInferenceCrash, CategoryToolTimeout,
		CategoryCircleDetected, CategoryMaxTurns,
		CategoryBudgetExceeded, CategoryManifestError,
	}

	since := time.Now().UTC().Add(-PatternWindow)
	var patterns []Pattern

	for _, role := range roles {
		for _, cat := range categories {
			count, err := store.CountByRoleCategory(role, cat, since)
			if err != nil {
				slog.Warn("telemetry: pattern detection query failed", "role", role, "category", cat, "err", err)
				continue
			}
			if count >= PatternThreshold {
				patterns = append(patterns, Pattern{
					Role:     role,
					Category: cat,
					Count:    count,
					Window:   "24h",
				})
				slog.Warn("telemetry: recurring failure pattern detected (store)",
					"role", role,
					"category", string(cat),
					"count", count,
				)
			}
		}
	}
	return patterns
}

// uniqueRoles extracts distinct roles from the in-memory buffer.
func (c *Collector) uniqueRoles() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	seen := map[string]struct{}{}
	for _, e := range c.events {
		seen[e.Role] = struct{}{}
	}
	roles := make([]string, 0, len(seen))
	for r := range seen {
		roles = append(roles, r)
	}
	return roles
}
