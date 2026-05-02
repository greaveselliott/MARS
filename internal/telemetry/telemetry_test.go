package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassify_contextOverflow(t *testing.T) {
	t.Parallel()
	cases := []string{
		`request (134706 tokens) exceeds the available context size (32768 tokens)`,
		`exceed_context_size_error`,
		`context_overflow in agent loop`,
	}
	for _, msg := range cases {
		require.Equal(t, CategoryContextOverflow, Classify(msg), "input: %s", msg)
	}
}

func TestClassify_llmUnreachable(t *testing.T) {
	t.Parallel()
	cases := []string{
		`llm_unreachable: connection reset`,
		`unexpected status 502 Bad Gateway`,
		`dial tcp 127.0.0.1:8080: connection refused`,
	}
	for _, msg := range cases {
		require.Equal(t, CategoryLLMUnreachable, Classify(msg), "input: %s", msg)
	}
}

func TestClassify_inferenceCrash(t *testing.T) {
	t.Parallel()
	cases := []string{
		`inference: health check failed after 30s`,
		`inference server crash detected`,
		`inference: OOM killed`,
		`process exited with signal: killed`,
	}
	for _, msg := range cases {
		require.Equal(t, CategoryInferenceCrash, Classify(msg), "input: %s", msg)
	}
}

func TestClassify_modelUnavailable(t *testing.T) {
	t.Parallel()
	cases := []string{
		`inference: no local model configured for tier "coding" and no remote fallback configured`,
		`inference: local model for tier "fast" is missing at /models/fast.gguf and no remote fallback configured`,
		`executor: get inference endpoint for role "ceo": inference: no local model for tier "coding" and no remote fallback configured`,
	}
	for _, msg := range cases {
		require.Equal(t, CategoryModelUnavailable, Classify(msg), "input: %s", msg)
	}
}

func TestClassify_toolTimeout(t *testing.T) {
	t.Parallel()
	cases := []string{
		`shell_exec: command timed out after 30s`,
		`context deadline exceeded (timeout)`,
	}
	for _, msg := range cases {
		require.Equal(t, CategoryToolTimeout, Classify(msg), "input: %s", msg)
	}
}

func TestClassify_circleDetected(t *testing.T) {
	t.Parallel()
	require.Equal(t, CategoryCircleDetected, Classify("circle_detected: same tool call 3x"))
}

func TestClassify_maxTurns(t *testing.T) {
	t.Parallel()
	require.Equal(t, CategoryMaxTurns, Classify("agent stopped: max_turns reached"))
}

func TestClassify_budgetExceeded(t *testing.T) {
	t.Parallel()
	require.Equal(t, CategoryBudgetExceeded, Classify("budget_exceeded at 50000 tokens"))
}

func TestClassify_manifestError(t *testing.T) {
	t.Parallel()
	require.Equal(t, CategoryManifestError, Classify("bundle: role 'x' not found"))
}

func TestClassify_unknown(t *testing.T) {
	t.Parallel()
	require.Equal(t, CategoryUnknown, Classify("something completely different"))
}

func TestRetryable(t *testing.T) {
	t.Parallel()
	retryable := []FailureCategory{
		CategoryContextOverflow,
		CategoryLLMUnreachable,
		CategoryInferenceCrash,
		CategoryToolTimeout,
	}
	for _, c := range retryable {
		require.True(t, c.Retryable(), "expected retryable: %s", c)
	}
	nonRetryable := []FailureCategory{
		CategoryModelUnavailable,
		CategoryCircleDetected,
		CategoryMaxTurns,
		CategoryBudgetExceeded,
		CategoryManifestError,
		CategoryUnknown,
	}
	for _, c := range nonRetryable {
		require.False(t, c.Retryable(), "expected non-retryable: %s", c)
	}
}

func TestRemediate_actions(t *testing.T) {
	t.Parallel()
	require.Equal(t, ActionRetryHalfContext, Remediate(CategoryContextOverflow))
	require.Equal(t, ActionRestartInference, Remediate(CategoryLLMUnreachable))
	require.Equal(t, ActionRestartInference, Remediate(CategoryInferenceCrash))
	require.Equal(t, ActionNone, Remediate(CategoryModelUnavailable))
	require.Equal(t, ActionRetryLonger, Remediate(CategoryToolTimeout))
	require.Equal(t, ActionNone, Remediate(CategoryCircleDetected))
	require.Equal(t, ActionNone, Remediate(CategoryMaxTurns))
	require.Equal(t, ActionNone, Remediate(CategoryUnknown))
}

type mockBroadcaster struct {
	mu     sync.Mutex
	events []string
}

func (m *mockBroadcaster) BroadcastEvent(eventType, data string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, eventType+":"+data)
}

func TestCollector_RecordAndBroadcast(t *testing.T) {
	t.Parallel()
	dash := &mockBroadcaster{}
	c := NewCollector(dash, nil)

	evt := c.Record("job-1", "repo-1", "cto", "request (134706 tokens) exceeds the available context size (32768 tokens)")

	require.Equal(t, CategoryContextOverflow, evt.Category)
	require.True(t, evt.Remedied)
	require.Equal(t, string(ActionRetryHalfContext), evt.Action)

	require.Len(t, c.Events(), 1)
	require.Len(t, dash.events, 1)

	stats := c.Stats()
	require.Equal(t, 1, stats[CategoryContextOverflow])
}

func TestCollector_RemediationCallback(t *testing.T) {
	t.Parallel()
	c := NewCollector(nil, nil)

	var called Event
	c.SetRemediator(func(evt Event) {
		called = evt
	})

	c.Record("job-2", "repo-1", "engineer", "connection refused")

	require.Equal(t, "job-2", called.JobID)
	require.Equal(t, CategoryLLMUnreachable, called.Category)
	require.Equal(t, string(ActionRestartInference), called.Action)
}

func TestCollector_NonRetryableSkipsRemediation(t *testing.T) {
	t.Parallel()
	c := NewCollector(nil, nil)

	remediated := false
	c.SetRemediator(func(evt Event) {
		remediated = true
	})

	evt := c.Record("job-3", "repo-1", "ceo", "circle_detected: same tool call 3x")

	require.False(t, evt.Remedied)
	require.False(t, remediated)
}

func TestCollector_RingBufferCap(t *testing.T) {
	t.Parallel()
	c := NewCollector(nil, nil)

	for i := 0; i < maxEvents+50; i++ {
		c.Record("job", "repo", "role", "timeout error")
	}

	require.Len(t, c.Events(), maxEvents)
}

func TestDetectPatternsFromStoreGroupsByRepoRoleCategory(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(filepath.Join(t.TempDir(), "telemetry.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	c := NewCollector(nil, store)
	for i := 0; i < PatternThreshold; i++ {
		c.Record("job-a", "repo-1", "engineer", "agent stopped: max_turns reached")
	}
	c.Record("job-b", "repo-2", "engineer", "agent stopped: max_turns reached")

	patterns := c.DetectPatternsFromStore()
	require.Len(t, patterns, 1)
	require.Equal(t, "repo-1", patterns[0].RepoID)
	require.Equal(t, "engineer", patterns[0].Role)
	require.Equal(t, CategoryMaxTurns, patterns[0].Category)
	require.Equal(t, PatternThreshold, patterns[0].Count)
}

func TestStore_LatestByRoleCategory(t *testing.T) {
	t.Parallel()
	store, err := OpenStore(filepath.Join(t.TempDir(), "telemetry.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	old := Event{
		ID:        "evt-old",
		Timestamp: time.Now().UTC().Add(-time.Hour),
		JobID:     "job-old",
		RepoID:    "repo-1",
		Role:      "engineer",
		Category:  CategoryToolTimeout,
		Message:   "old timeout",
	}
	latest := old
	latest.ID = "evt-new"
	latest.JobID = "job-new"
	latest.Timestamp = time.Now().UTC()
	latest.Message = "new timeout"
	require.NoError(t, store.Save(old))
	require.NoError(t, store.Save(latest))

	got, err := store.LatestByRoleCategory("repo-1", "engineer", CategoryToolTimeout, time.Now().UTC().Add(-24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "evt-new", got.ID)
	require.Equal(t, "job-new", got.JobID)
}

func TestTriagePattern_modelUnavailableTargetsInference(t *testing.T) {
	t.Parallel()

	proposal := TriagePattern(Pattern{
		Role:     "ceo",
		Category: CategoryModelUnavailable,
		Count:    3,
	})

	require.Equal(t, TargetInference, proposal.Target)
	require.Equal(t, "Install or route model tier", proposal.Title)
	require.Contains(t, proposal.Suggestion, "setup")
	require.Contains(t, proposal.CandidateFiles, "~/.mars-harness/config.yaml")
}

func TestTriagePattern_contextOverflowTargetsContext(t *testing.T) {
	t.Parallel()

	proposal := TriagePattern(Pattern{
		Role:     "engineer",
		Category: CategoryContextOverflow,
		Count:    3,
		Window:   "24h",
	})

	require.Equal(t, TargetContext, proposal.Target)
	require.Equal(t, "medium", proposal.Severity)
	require.Contains(t, proposal.Suggestion, "knowledge routes")
	require.Contains(t, proposal.CandidateFiles, ".harness/knowledge/context-glossary.yaml")
	require.Contains(t, proposal.CandidateFiles, ".harness/roles/engineer.md")
}

func TestTriagePattern_manifestErrorTargetsManifest(t *testing.T) {
	t.Parallel()

	proposal := TriagePattern(Pattern{
		Role:     "coo",
		Category: CategoryManifestError,
		Count:    6,
	})

	require.Equal(t, TargetManifest, proposal.Target)
	require.Equal(t, "high", proposal.Severity)
	require.Contains(t, proposal.CandidateFiles, ".harness/manifest.yaml")
	require.Greater(t, proposal.Confidence, 0.8)
}

func TestTriagePattern_loopTargetsSkill(t *testing.T) {
	t.Parallel()

	proposal := TriagePattern(Pattern{
		Role:     "engineer",
		Category: CategoryMaxTurns,
		Count:    3,
	})

	require.Equal(t, TargetSkill, proposal.Target)
	require.Contains(t, proposal.Suggestion, "compact scoped skill")
	require.Contains(t, proposal.CandidateFiles, ".harness/skills/engineer-workflow/SKILL.md")
	require.Contains(t, proposal.CandidateFiles, ".harness/roles/engineer.md")
}

func TestTriageScore_lowScoreProducesProcessProposal(t *testing.T) {
	t.Parallel()

	proposal, ok := TriageScore(ScoreSnapshot{
		Role:       "dogfood",
		RepoID:     "repo-1",
		Value:      0.33,
		SampleSize: 9,
		WindowDays: 30,
	})

	require.True(t, ok)
	require.Equal(t, TargetProcess, proposal.Target)
	require.Equal(t, "high", proposal.Severity)
	require.Contains(t, proposal.Suggestion, "intervention debt")
	require.Contains(t, proposal.Suggestion, "reusable skills")
	require.Contains(t, proposal.CandidateFiles, ".harness/roles/dogfood.md")
	require.Contains(t, proposal.CandidateFiles, ".harness/skills/dogfood-workflow/SKILL.md")
}

func TestTriageScore_ignoresHealthyOrSparseScores(t *testing.T) {
	t.Parallel()

	_, ok := TriageScore(ScoreSnapshot{Role: "engineer", Value: 0.4, SampleSize: 4})
	require.False(t, ok)

	_, ok = TriageScore(ScoreSnapshot{Role: "engineer", Value: 0.75, SampleSize: 10})
	require.False(t, ok)
}

func TestRecordGoalFromProposal_createsActiveGoalForActionableTelemetry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	proposal := TriagePattern(Pattern{
		RepoID:   "repo-1",
		Role:     "ceo",
		Category: CategoryModelUnavailable,
		Count:    PatternThreshold * 2,
		Window:   "24h",
	})

	result, err := RecordGoalFromProposal(dir, proposal, time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, "active", result.Status)
	require.Equal(t, "docs/goals/active.md", result.Path)

	data, err := os.ReadFile(filepath.Join(dir, "docs", "goals", "active.md"))
	require.NoError(t, err)
	text := string(data)
	require.Contains(t, text, "Status: active")
	require.Contains(t, text, "Source: telemetry")
	require.Contains(t, text, "Dedupe Key: "+result.DedupeKey)
	require.Contains(t, text, "Review Trigger")
}

func TestRecordGoalFromProposal_createsObservationForWeakTelemetry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	proposal := ImprovementProposal{
		RepoID:     "repo-1",
		Role:       "janitor",
		Category:   CategoryUnknown,
		Target:     TargetUnknown,
		Severity:   "medium",
		Title:      "Noisy signal",
		Suggestion: "Watch this pattern before creating work.",
		Evidence:   "one unknown event",
		Confidence: 0.4,
	}

	result, err := RecordGoalFromProposal(dir, proposal, time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, "observation", result.Status)
	require.Equal(t, "docs/goals/observations.md", result.Path)

	data, err := os.ReadFile(filepath.Join(dir, "docs", "goals", "observations.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "Status: observation")
	require.Contains(t, string(data), "weak/noisy evidence")
}

func TestRecordGoalFromProposal_dedupesByEvidenceKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	proposal := TriagePattern(Pattern{
		RepoID:   "repo-1",
		Role:     "engineer",
		Category: CategoryInferenceCrash,
		Count:    PatternThreshold * 2,
		Window:   "24h",
	})

	_, err := RecordGoalFromProposal(dir, proposal, time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	result, err := RecordGoalFromProposal(dir, proposal, time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, result.Updated)

	data, err := os.ReadFile(filepath.Join(dir, "docs", "goals", "active.md"))
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(data), "## G-001:"), "duplicate telemetry should update the existing goal")
	require.Contains(t, string(data), "Evidence Update - 2026-05-03")
}
