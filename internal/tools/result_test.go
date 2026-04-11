package tools

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestTruncateUTF8_noTruncation(t *testing.T) {
	t.Parallel()
	s := "hello 世界"
	out, trunc := TruncateUTF8(s, 100)
	require.False(t, trunc)
	require.Equal(t, s, out)
}

func TestTruncateUTF8_splitsSafely(t *testing.T) {
	t.Parallel()
	s := "a世界b"
	// Cut in middle of 世界 (3 bytes per char)
	out, trunc := TruncateUTF8(s, 4)
	require.True(t, trunc)
	require.True(t, utf8.ValidString(out))
	require.LessOrEqual(t, len(out), 4)
}

func TestIsProbablyBinary_nul(t *testing.T) {
	t.Parallel()
	require.True(t, IsProbablyBinary([]byte("a\x00b")))
}

func TestIsProbablyBinary_utf8Text(t *testing.T) {
	t.Parallel()
	require.False(t, IsProbablyBinary([]byte("plain text\nline2")))
}

func TestToolResult_FormatForModel_error(t *testing.T) {
	t.Parallel()
	r := ToolResult{Err: errSample{}}
	require.Contains(t, r.FormatForModel(), "boom")
}

func TestToolResult_FormatForModel_combined(t *testing.T) {
	t.Parallel()
	r := ToolResult{
		Output:    "out",
		Stderr:    "err",
		ExitCode:  2,
		Truncated: true,
	}
	got := r.FormatForModel()
	require.Contains(t, got, "stderr:")
	require.Contains(t, got, "exit_code: 2")
	require.Contains(t, got, "truncated")
	require.Contains(t, got, "out")
}

type errSample struct{}

func (errSample) Error() string { return "boom" }
