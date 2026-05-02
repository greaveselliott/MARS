package telemetry

import (
	"sync"
	"testing"

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
