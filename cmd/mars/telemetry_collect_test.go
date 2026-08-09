/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/release-versioning.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-012-self-improvement-loop.md
- docs/product-specs/product-surface.md
*/
package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelemetryCollectRejectsNonLiteralLoopbackBeforeDatabaseCreation(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:9092", "192.168.1.5:9092", "localhost:9092", "mars.local:9092"} {
		t.Run(addr, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "must-not-exist", "intake.db")
			cmd := telemetryCollectCmd()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"--addr", addr, "--db", dbPath})

			err := cmd.Execute()

			require.EqualError(t, err, "telemetry collector must use a literal loopback IP and TCP port such as 127.0.0.1:9092 or [::1]:9092")
			require.NoFileExists(t, dbPath)
			require.NoDirExists(t, filepath.Dir(dbPath))
			require.NotContains(t, output.String(), dbPath)
		})
	}
}

func TestTelemetryCollectDefaultIsLiteralLoopback(t *testing.T) {
	cmd := telemetryCollectCmd()
	require.Equal(t, "127.0.0.1:9092", cmd.Flag("addr").DefValue)
}
