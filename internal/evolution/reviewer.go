package evolution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// ReviewerConfig controls the Reviewer meta-role.
type ReviewerConfig struct {
	MaxRunsPerDay    int // default 1 per role
	AutoDisableAfter int // disable after N worsening evolutions (default 3)
}

// DefaultReviewerConfig returns safe defaults for the Reviewer.
func DefaultReviewerConfig() ReviewerConfig {
	return ReviewerConfig{
		MaxRunsPerDay:    1,
		AutoDisableAfter: 3,
	}
}

// ReviewResult is the output of a Reviewer analysis.
type ReviewResult struct {
	Classification string   `json:"classification"`
	Suggestion     string   `json:"suggestion"`
	FilesToModify  []string `json:"files_to_modify"`
	Confidence     float64  `json:"confidence"`
}

// Evolution records a single Reviewer run and its outcome.
type Evolution struct {
	ID          string
	Role        string
	RepoID      string
	Result      string  // JSON ReviewResult
	ScoreBefore float64
	ScoreAfter  float64 // set after outcome tracking
	CreatedAt   time.Time
}

// CanReview checks rate limits and auto-disable status for the Reviewer.
// Returns true if a review is allowed, or false with a reason string.
func CanReview(store *Store, role string, cfg ReviewerConfig) (bool, string) {
	if cfg.MaxRunsPerDay <= 0 {
		cfg.MaxRunsPerDay = 1
	}
	if cfg.AutoDisableAfter <= 0 {
		cfg.AutoDisableAfter = 3
	}

	ctx := context.Background()
	since := time.Now().UTC().Add(-24 * time.Hour)

	count, err := store.CountRecentEvolutions(ctx, role, since)
	if err != nil {
		slog.Error("evolution: failed to count recent evolutions", "role", role, "error", err)
		return false, fmt.Sprintf("failed to check rate limit: %v", err)
	}

	if count >= cfg.MaxRunsPerDay {
		return false, fmt.Sprintf("rate limit: %d/%d runs used in last 24h for role %q", count, cfg.MaxRunsPerDay, role)
	}

	evolutions, err := store.GetEvolutions(ctx, role, cfg.AutoDisableAfter)
	if err != nil {
		slog.Error("evolution: failed to get evolutions for auto-disable check", "role", role, "error", err)
		return false, fmt.Sprintf("failed to check auto-disable: %v", err)
	}

	if len(evolutions) >= cfg.AutoDisableAfter {
		allWorsening := true
		for _, ev := range evolutions[:cfg.AutoDisableAfter] {
			if ev.ScoreAfter >= ev.ScoreBefore {
				allWorsening = false
				break
			}
		}
		if allWorsening {
			return false, fmt.Sprintf("auto-disabled: last %d evolutions for role %q all worsened scores", cfg.AutoDisableAfter, role)
		}
	}

	return true, ""
}

// RecordEvolution stores a Reviewer run result as an Evolution.
func RecordEvolution(ctx context.Context, store *Store, role, repoID string, result ReviewResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("evolution: marshal review result: %w", err)
	}

	ev := Evolution{
		ID:        newUUID(),
		Role:      role,
		RepoID:    repoID,
		Result:    string(resultJSON),
		CreatedAt: time.Now().UTC(),
	}

	if err := store.SaveEvolution(ctx, ev); err != nil {
		return fmt.Errorf("evolution: record evolution: %w", err)
	}

	slog.Info("evolution: recorded evolution",
		"id", ev.ID,
		"role", role,
		"classification", result.Classification,
		"confidence", result.Confidence,
	)
	return nil
}
