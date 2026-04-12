package tools

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// DefaultMaxToolOutputBytes is the maximum combined output kept for shell-like tools (MH-002).
// 32KB keeps a single tool result under ~8K tokens, preventing context window blowouts.
const DefaultMaxToolOutputBytes = 32 * 1024

// ToolResult is the structured outcome of a tool invocation for the LLM and future tracing.
type ToolResult struct {
	Output    string        // primary text shown to the model
	Stderr    string        // captured stderr when applicable
	ExitCode  int           // subprocess exit code; 0 if not applicable
	Err       error         // execution error (timeout, validation); not serialized to model text by default
	Truncated bool          // true if output was truncated to respect byte limits
	IsBinary  bool          // true if binary content was detected and redacted
	Duration  time.Duration // wall time spent in the tool handler
}

// FormatForModel returns a single string suitable for a tool_result message.
func (r ToolResult) FormatForModel() string {
	if r.Err != nil {
		return fmt.Sprintf("error: %v", r.Err)
	}
	var b strings.Builder
	if r.IsBinary {
		b.WriteString("(binary content omitted)\n")
	}
	if r.Stderr != "" {
		b.WriteString("stderr:\n")
		b.WriteString(r.Stderr)
		b.WriteString("\n")
	}
	if r.ExitCode != 0 {
		fmt.Fprintf(&b, "exit_code: %d\n", r.ExitCode)
	}
	if r.Truncated {
		b.WriteString("(output truncated)\n")
	}
	b.WriteString(r.Output)
	return strings.TrimSpace(b.String())
}

// TruncateUTF8 truncates s to at most maxBytes, measured in UTF-8 bytes, without splitting code points.
// If truncated, ok is true.
func TruncateUTF8(s string, maxBytes int) (out string, truncated bool) {
	if maxBytes <= 0 {
		return "", len(s) > 0
	}
	if len(s) <= maxBytes {
		return s, false
	}
	out = s[:maxBytes]
	for !utf8.ValidString(out) {
		out = out[:len(out)-1]
		if len(out) == 0 {
			return "", true
		}
	}
	return out, true
}

// IsProbablyBinary reports whether buf looks like non-text (NUL or non-UTF8), or DetectContentType suggests binary media.
func IsProbablyBinary(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}
	if !utf8.Valid(buf) {
		return true
	}
	ct := http.DetectContentType(buf)
	switch {
	case strings.HasPrefix(ct, "text/"):
		return false
	case strings.Contains(ct, "json"), strings.Contains(ct, "xml"):
		return false
	case ct == "application/octet-stream":
		// Common for UTF-8 text without a BOM; we already validated UTF-8 above.
		return false
	case strings.HasPrefix(ct, "image/"), strings.HasPrefix(ct, "video/"), strings.HasPrefix(ct, "audio/"):
		return true
	default:
		return false
	}
}
