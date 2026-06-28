/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package testutil

import (
	"context"
	"testing"

	"github.com/greaveselliott/mars/internal/llm"
	"github.com/stretchr/testify/require"
)

func TestMockLLM_replaysInOrder(t *testing.T) {
	t.Parallel()
	m := &MockLLM{Replies: []llm.ChatCompletionResponse{
		{Choices: []llm.Choice{{Message: llm.Message{Content: "a"}}}},
		{Choices: []llm.Choice{{Message: llm.Message{Content: "b"}}}},
	}}

	r1, err := m.ChatCompletion(context.Background(), llm.ChatCompletionRequest{})
	require.NoError(t, err)
	require.Equal(t, "a", r1.Choices[0].Message.Content)

	r2, err := m.ChatCompletion(context.Background(), llm.ChatCompletionRequest{})
	require.NoError(t, err)
	require.Equal(t, "b", r2.Choices[0].Message.Content)

	_, err = m.ChatCompletion(context.Background(), llm.ChatCompletionRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exhausted")
	require.Equal(t, 2, m.CallCount())
}
