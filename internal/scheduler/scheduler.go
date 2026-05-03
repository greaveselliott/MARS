package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/greaveselliott/mars-harness/internal/queue"
)

// Schedule represents a periodic job trigger.
type Schedule struct {
	Name             string
	RepoID           string
	Role             string
	Cron             string // standard 5-field cron expression
	Timezone         string // IANA timezone name
	Trigger          string // JSON payload template
	PayloadMode      string
	ConcurrencyGroup string
	DailyCap         int
}

// Scheduler evaluates cron schedules and enqueues jobs.
type Scheduler struct {
	q         *queue.Queue
	mu        sync.Mutex
	schedules []Schedule
	lastFired map[string]time.Time // keyed by schedule name
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New creates a scheduler backed by the given queue.
func New(q *queue.Queue) *Scheduler {
	return &Scheduler{
		q:         q,
		lastFired: make(map[string]time.Time),
	}
}

// Register adds a schedule. Must be called before Start.
func (s *Scheduler) Register(sched Schedule) error {
	if _, err := parseCron(sched.Cron); err != nil {
		return fmt.Errorf("scheduler: invalid cron %q: %w", sched.Cron, err)
	}
	if sched.Timezone != "" {
		if _, err := time.LoadLocation(sched.Timezone); err != nil {
			return fmt.Errorf("scheduler: invalid timezone %q: %w", sched.Timezone, err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schedules = append(s.schedules, sched)
	slog.Info("scheduler: registered", "name", sched.Name, "cron", sched.Cron)
	return nil
}

// Start begins a tick loop that evaluates schedules every 60 seconds.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	s.wg.Add(1)
	go s.loop(ctx)
	slog.Info("scheduler: started")
}

// Stop cancels the tick loop and waits for it to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	slog.Info("scheduler: stopped")
}

func (s *Scheduler) loop(ctx context.Context) {
	defer s.wg.Done()

	// Evaluate immediately on start for catch-up.
	s.tick(ctx)

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	s.mu.Lock()
	schedules := make([]Schedule, len(s.schedules))
	copy(schedules, s.schedules)
	s.mu.Unlock()

	for _, sched := range schedules {
		s.evaluate(ctx, sched)
	}
}

func (s *Scheduler) evaluate(ctx context.Context, sched Schedule) {
	loc := time.UTC
	if sched.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(sched.Timezone)
		if err != nil {
			slog.Error("scheduler: bad timezone", "name", sched.Name, "tz", sched.Timezone, "error", err)
			return
		}
	}

	now := time.Now().In(loc)
	// Truncate to minute boundary for matching.
	nowMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, loc)

	expr, err := parseCron(sched.Cron)
	if err != nil {
		return
	}

	if !expr.matches(nowMinute) {
		return
	}

	// fire_once: don't fire again if already fired in this same minute window.
	s.mu.Lock()
	last, seen := s.lastFired[sched.Name]
	if seen && !last.Before(nowMinute) {
		s.mu.Unlock()
		return
	}
	s.lastFired[sched.Name] = nowMinute
	s.mu.Unlock()

	idemKey := fmt.Sprintf("sched:%s:%d", sched.Name, nowMinute.Unix())

	_, err = s.q.Enqueue(ctx, queue.Job{
		RepoID:           sched.RepoID,
		Role:             sched.Role,
		Trigger:          sched.Trigger,
		PayloadMode:      sched.PayloadMode,
		ConcurrencyGroup: sched.ConcurrencyGroup,
		DailyCap:         sched.DailyCap,
		IdempotencyKey:   idemKey,
	})
	if err != nil {
		slog.Error("scheduler: enqueue failed", "name", sched.Name, "error", err)
		return
	}
	slog.Info("scheduler: fired", "name", sched.Name, "minute", nowMinute.Format(time.RFC3339))
}

// cronExpr holds parsed 5-field cron data. Each field is a set of valid values.
type cronExpr struct {
	minutes     map[int]bool
	hours       map[int]bool
	daysOfMonth map[int]bool
	months      map[int]bool
	daysOfWeek  map[int]bool
}

func (e *cronExpr) matches(t time.Time) bool {
	return e.minutes[t.Minute()] &&
		e.hours[t.Hour()] &&
		e.daysOfMonth[t.Day()] &&
		e.months[int(t.Month())] &&
		e.daysOfWeek[int(t.Weekday())]
}

// parseCron parses a standard 5-field cron expression.
// Fields: minute(0-59) hour(0-23) day-of-month(1-31) month(1-12) day-of-week(0-6, 0=Sunday)
func parseCron(expr string) (*cronExpr, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	minutes, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hours, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	dom, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day-of-month: %w", err)
	}
	months, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	dow, err := parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("day-of-week: %w", err)
	}

	return &cronExpr{
		minutes:     minutes,
		hours:       hours,
		daysOfMonth: dom,
		months:      months,
		daysOfWeek:  dow,
	}, nil
}

// parseField handles *, N, N-M, N-M/S, */S, and comma-separated lists.
func parseField(field string, min, max int) (map[int]bool, error) {
	result := make(map[int]bool)

	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty field segment")
		}

		if err := parsePart(part, min, max, result); err != nil {
			return nil, err
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("field resolves to empty set")
	}
	return result, nil
}

func parsePart(part string, min, max int, result map[int]bool) error {
	step := 1

	if idx := strings.Index(part, "/"); idx >= 0 {
		var err error
		step, err = strconv.Atoi(part[idx+1:])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step %q", part[idx+1:])
		}
		part = part[:idx]
	}

	var rangeMin, rangeMax int

	switch {
	case part == "*":
		rangeMin, rangeMax = min, max
	case strings.Contains(part, "-"):
		bounds := strings.SplitN(part, "-", 2)
		var err error
		rangeMin, err = strconv.Atoi(bounds[0])
		if err != nil {
			return fmt.Errorf("invalid range start %q", bounds[0])
		}
		rangeMax, err = strconv.Atoi(bounds[1])
		if err != nil {
			return fmt.Errorf("invalid range end %q", bounds[1])
		}
	default:
		val, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid value %q", part)
		}
		rangeMin, rangeMax = val, val
	}

	if rangeMin < min || rangeMax > max || rangeMin > rangeMax {
		return fmt.Errorf("range %d-%d out of bounds [%d,%d]", rangeMin, rangeMax, min, max)
	}

	for v := rangeMin; v <= rangeMax; v += step {
		result[v] = true
	}
	return nil
}
