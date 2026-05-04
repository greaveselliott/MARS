/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
*/
package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobDispositionRecordRequiresSession(t *testing.T) {
	t.Parallel()

	_, err := handleJobDispositionRecord(context.Background(), Root{}, json.RawMessage(`{"status":"completed"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no tool session")
}

func TestJobDispositionRecordUsesSessionRecorder(t *testing.T) {
	t.Parallel()

	var recorded json.RawMessage
	ctx := WithSession(context.Background(), Session{
		DispositionRecorder: func(_ context.Context, raw json.RawMessage) error {
			recorded = append(recorded[:0], raw...)
			return nil
		},
	})

	result, err := handleJobDispositionRecord(ctx, Root{}, json.RawMessage(`{"status":"completed"}`))
	require.NoError(t, err)
	require.Equal(t, "job disposition recorded", result.Output)
	require.JSONEq(t, `{"status":"completed"}`, string(recorded))
}
