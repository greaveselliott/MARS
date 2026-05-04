/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-runtime.md
- docs/features/F-005-agent-execution-runtime.md
*/
package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/greaveselliott/mars-harness/internal/llm"
)

const defaultMaxBodyBytes = 100 * 1024

// Recorder appends JSON Lines (one object per line) and keeps a full copy for persistence (MH-005).
type Recorder struct {
	extra   io.Writer
	copyBuf bytes.Buffer
	w       io.Writer
	enc     *json.Encoder
	maxBody int

	mu            sync.Mutex
	traceID       string
	headerWritten bool
	turns         []Turn
	toolNames     map[string]struct{}
	started       time.Time
}

// NewRecorder writes each line to an internal buffer (for SQLite) and optionally mirrors to extra.
func NewRecorder(extra io.Writer) *Recorder {
	r := &Recorder{
		extra:     extra,
		maxBody:   defaultMaxBodyBytes,
		toolNames: make(map[string]struct{}),
	}
	if extra != nil {
		r.w = io.MultiWriter(&r.copyBuf, extra)
	} else {
		r.w = &r.copyBuf
	}
	r.enc = json.NewEncoder(r.w)
	r.enc.SetEscapeHTML(false)
	return r
}

// SetMaxBody sets the maximum bytes kept per content field (default 100KiB).
func (r *Recorder) SetMaxBody(n int) {
	if n > 0 {
		r.maxBody = n
	}
}

// NewID returns a unique trace identifier (no extra deps).
func NewID() string {
	return fmt.Sprintf("tr-%d", time.Now().UnixNano())
}

// WriteHeader writes the mandatory first line. Call once per job.
func (r *Recorder) WriteHeader(jobID, traceID, model string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.headerWritten {
		return fmt.Errorf("trace: header already written")
	}
	r.traceID = traceID
	r.started = time.Now()
	h := Header{
		Type:    "header",
		TraceID: traceID,
		JobID:   jobID,
		Model:   model,
		Started: r.started.UTC().Format(time.RFC3339Nano),
	}
	if err := r.enc.Encode(h); err != nil {
		return err
	}
	r.headerWritten = true
	return nil
}

// WriteTurn records one message. tokenEstimate is typically cumulative context size for the turn.
func (r *Recorder) WriteTurn(msg llm.Message, tokenEstimate int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.headerWritten {
		return fmt.Errorf("trace: WriteHeader before WriteTurn")
	}
	t := Turn{
		Type:          "turn",
		Role:          msg.Role,
		ToolCalls:     append([]llm.ToolCall(nil), msg.ToolCalls...),
		ToolCallID:    msg.ToolCallID,
		Timestamp:     time.Now().UTC(),
		TokenEstimate: tokenEstimate,
	}
	content := msg.Content
	if len(content) > r.maxBody {
		content = truncateUTF8(content, r.maxBody)
		t.Truncated = true
		content += "\n(trace: content truncated per MH-005)"
	}
	t.Content = content
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "" {
			r.toolNames[tc.Function.Name] = struct{}{}
		}
	}
	if err := r.enc.Encode(t); err != nil {
		return err
	}
	r.turns = append(r.turns, t)
	return nil
}

// JSONL returns everything recorded (including the header line).
func (r *Recorder) JSONL() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.copyBuf.String()
}

// Started reports whether WriteHeader completed successfully.
func (r *Recorder) Started() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.headerWritten
}

// TraceID returns the identifier from WriteHeader.
func (r *Recorder) TraceID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.traceID
}

// Finalize builds a Summary from recorded turns.
func (r *Recorder) Finalize(jobID, outcome string, wall time.Duration, llmCalls, toolInvocations int, runErr error) Summary {
	r.mu.Lock()
	defer r.mu.Unlock()
	sumTok := 0
	for _, t := range r.turns {
		sumTok += t.TokenEstimate
	}
	tools := make([]string, 0, len(r.toolNames))
	for n := range r.toolNames {
		tools = append(tools, n)
	}
	sort.Strings(tools)
	errStr := ""
	if runErr != nil {
		errStr = runErr.Error()
	}
	return Summary{
		TraceID:          r.traceID,
		JobID:            jobID,
		Outcome:          outcome,
		WallMs:           wall.Milliseconds(),
		TotalTokens:      sumTok,
		ToolsCalled:      tools,
		TurnCount:        len(r.turns),
		ToolCallMessages: countToolCalls(r.turns),
		ToolInvocations:  toolInvocations,
		LLMCalls:         llmCalls,
		Error:            errStr,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func countToolCalls(turns []Turn) int {
	n := 0
	for _, t := range turns {
		n += len(t.ToolCalls)
	}
	return n
}

func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	out := s[:maxBytes]
	for !utf8.ValidString(out) && len(out) > 0 {
		out = out[:len(out)-1]
	}
	return out
}
