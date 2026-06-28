/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
*/
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
}

type anthropicResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *Client) anthropicChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	payload := anthropicRequest{
		Model:       req.Model,
		Messages:    anthropicMessages(req.Messages),
		Tools:       anthropicTools(req.Tools),
		MaxTokens:   4096,
		Temperature: req.Temperature,
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		payload.MaxTokens = *req.MaxTokens
	}
	payload.System = anthropicSystem(req.Messages)
	if len(payload.Messages) == 0 {
		payload.Messages = []anthropicMessage{{Role: "user", Content: ""}}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("llm: marshal anthropic request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.prefix+"/messages", bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("llm: build anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("x-api-key", c.cfg.APIKey)
	}

	resp, err := c.cfg.HTTPClient.Do(httpReq)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("llm: anthropic request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr != nil {
			return ChatCompletionResponse{}, fmt.Errorf("llm: read anthropic error body: %w", readErr)
		}
		return ChatCompletionResponse{}, fmt.Errorf("llm: anthropic returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var out anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("llm: decode anthropic response: %w", err)
	}
	converted := anthropicToChatCompletion(out)
	if converted.Model == "" {
		converted.Model = req.Model
	}
	if err := validateChatCompletionResponse(converted); err != nil {
		return ChatCompletionResponse{}, err
	}
	return converted, nil
}

func anthropicSystem(messages []Message) string {
	var parts []string
	for _, msg := range messages {
		if msg.Role == "system" && strings.TrimSpace(msg.Content) != "" {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func anthropicMessages(messages []Message) []anthropicMessage {
	var out []anthropicMessage
	for _, msg := range messages {
		role := msg.Role
		content := msg.Content
		switch role {
		case "system":
			continue
		case "assistant":
			role = "assistant"
		case "tool":
			role = "user"
			content = "Tool result"
			if msg.ToolCallID != "" {
				content += " " + msg.ToolCallID
			}
			content += ": " + msg.Content
		default:
			role = "user"
		}
		out = append(out, anthropicMessage{Role: role, Content: content})
	}
	return out
}

func anthropicTools(tools []ToolDefinition) []anthropicTool {
	out := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}
	return out
}

func anthropicToChatCompletion(resp anthropicResponse) ChatCompletionResponse {
	var textParts []string
	var calls []ToolCall
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			args := strings.TrimSpace(string(block.Input))
			if args == "" {
				args = "{}"
			}
			calls = append(calls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: args,
				},
			})
		}
	}
	return ChatCompletionResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:      "assistant",
				Content:   strings.Join(textParts, "\n"),
				ToolCalls: calls,
			},
			FinishReason: resp.StopReason,
		}},
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}
