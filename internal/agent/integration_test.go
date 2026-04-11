package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/llm"
	"github.com/greaveselliott/mars-harness/internal/tools"

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
