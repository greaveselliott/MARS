package testutil

import (
	"context"
	"fmt"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

// MockLLM replays scripted ChatCompletionResponses in order.
// When all replies are exhausted it returns an error.
type MockLLM struct {
	Replies []llm.ChatCompletionResponse
	idx     int
}

func (m *MockLLM) ChatCompletion(_ context.Context, _ llm.ChatCompletionRequest) (llm.ChatCompletionResponse, error) {
	if m.idx >= len(m.Replies) {
		return llm.ChatCompletionResponse{}, fmt.Errorf("MockLLM: exhausted %d scripted replies", len(m.Replies))
	}
	r := m.Replies[m.idx]
	m.idx++
	return r, nil
}

// CallCount returns how many completions have been served so far.
func (m *MockLLM) CallCount() int { return m.idx }
