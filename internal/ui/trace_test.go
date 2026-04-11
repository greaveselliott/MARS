package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWriteToolCall_PrintsNameAndArgs(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteToolCall("file_read", `{"path":"main.go"}`)

	out := buf.String()
	assert.Contains(t, out, "file_read")
	assert.Contains(t, out, "main.go")
}

func TestWriteToolCall_Quiet(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, true, true)
	tw.WriteToolCall("file_read", `{"path":"main.go"}`)

	assert.Empty(t, buf.String())
}

func TestWriteToolResult_PrintsOutput(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteToolResult("shell_exec", "exit code 0")

	out := buf.String()
	assert.Contains(t, out, "shell_exec")
	assert.Contains(t, out, "exit code 0")
}

func TestWriteAssistant_PrintsContent(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteAssistant("I found the bug in line 42.")

	assert.Contains(t, buf.String(), "I found the bug in line 42.")
}

func TestWriteAssistant_Quiet(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, true, true)
	tw.WriteAssistant("This should not appear")

	assert.Empty(t, buf.String())
}

func TestWriteError_PrintsMessage(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteError("connection refused")

	out := buf.String()
	assert.Contains(t, out, "connection refused")
}

func TestWriteError_NotSuppressedByQuiet(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, true, true)
	tw.WriteError("errors always show")

	assert.Contains(t, buf.String(), "errors always show")
}

func TestWriteSummary_IncludesAllFields(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteSummary("completed", 5, 12, 3*time.Second, 8000)

	out := buf.String()
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "5")
	assert.Contains(t, out, "12")
	assert.Contains(t, out, "3s")
	assert.Contains(t, out, "8000")
}

func TestWriteSummary_HidesTokensWhenZero(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteSummary("completed", 1, 1, time.Second, 0)

	assert.NotContains(t, buf.String(), "tokens")
}

func TestWriteSummary_VisibleInQuietMode(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, true, true)
	tw.WriteSummary("completed", 1, 1, time.Second, 100)

	assert.Contains(t, buf.String(), "completed")
}

func TestNoColor_StripsANSI(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	tw.WriteAssistant("hello")

	assert.NotContains(t, buf.String(), "\033[")
}

func TestColor_IncludesANSI(t *testing.T) {
	var buf bytes.Buffer
	tw := &TraceWriter{w: &buf, noColor: false}
	tw.WriteAssistant("hello")

	assert.Contains(t, buf.String(), "\033[")
}

func TestTruncate_LongArgs(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, true)
	longArg := strings.Repeat("x", 200)
	tw.WriteToolCall("test", longArg)

	out := buf.String()
	assert.Contains(t, out, "...")
	assert.Less(t, len(out), 300)
}

func TestNonTTYWriter_DisablesColor(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTraceWriter(&buf, false, false)
	assert.True(t, tw.noColor, "non-TTY writer should auto-disable colour")
}
