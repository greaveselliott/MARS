/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/internal/llm"
	"github.com/greaveselliott/mars/internal/qualityscore"
	"github.com/greaveselliott/mars/internal/scoring"
	"github.com/greaveselliott/mars/internal/tools"

	"github.com/stretchr/testify/require"
)

func TestIntegration_mockLLMWritesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	allow := []string{"file_write"}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_write", "w1", `{"path":"created.txt","content":"from-agent"}`),
		textResp("File written."),
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    allow,
		SystemPrompt: "You write files when asked.",
		UserMessage:  "Create created.txt.",
		Config:       LoopConfig{Model: "integration", MaxTurns: 10},
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)

	b, err := os.ReadFile(filepath.Join(dir, "created.txt"))
	require.NoError(t, err)
	require.Equal(t, "from-agent", string(b))
}

func TestIntegration_fakeLLMDogfoodRecordsQualityEvidence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "backlog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "in-progress"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "tickets", "done"), 0o755))

	root, err := tools.NewRoot(dir)
	require.NoError(t, err)
	reg, err := tools.DefaultRegistry()
	require.NoError(t, err)
	ex := tools.NewExecutor(reg)
	ex.Session = &tools.Session{Role: "engineer", RepoID: "repo-1", TrustLevel: "contributor"}

	mock := &seqMock{replies: []llm.ChatCompletionResponse{
		toolResp("file_write", "write-feature", `{"path":"feature.txt","content":"dogfood-ready\n"}`),
		toolResp("shell_exec", "verify-feature", `{"argv":["test","-f","feature.txt"]}`),
		textResp("Implemented and verified."),
	}}

	res, err := Run(context.Background(), Params{
		Completer:    mock,
		Registry:     reg,
		Executor:     ex,
		Root:         root,
		Allowlist:    []string{"file_write", "shell_exec"},
		SystemPrompt: "You are a deterministic dogfood engineer.",
		UserMessage:  "Create and verify a feature artifact.",
		Config:       LoopConfig{Model: "fake-dogfood", MaxTurns: 10},
		JobID:        "fake-dogfood-job",
	})
	require.NoError(t, err)
	require.Equal(t, EndCompleted, res.EndReason)
	require.Equal(t, 2, res.ToolInvocations)

	dbPath := filepath.Join(dir, "mars.db")
	scoreStore, err := scoring.OpenStore(dbPath)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		require.NoError(t, scoreStore.RecordOutcome(context.Background(), scoring.Outcome{
			JobID:  "fake-dogfood-job",
			RepoID: "repo-1",
			Role:   "engineer",
			Type:   OutcomeForEndReason(res.EndReason),
		}))
	}
	_, err = scoreStore.ComputeScore(context.Background(), "engineer", "repo-1", 30)
	require.NoError(t, err)
	require.NoError(t, scoreStore.Close())

	report, err := qualityscore.Export(context.Background(), qualityscore.Options{
		RepoPath:              dir,
		RepoID:                "repo-1",
		DBPath:                dbPath,
		DisableTicketCreation: true,
	})
	require.NoError(t, err)
	require.NotEqual(t, "Insufficient evidence", report.Grade)
	data, err := os.ReadFile(filepath.Join(dir, "docs", "QUALITY_SCORE.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "| Repo | Role | Grade | Score | Samples | Window | Computed |")
	require.Contains(t, string(data), "engineer")
}
