/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/agent-smoke-validation.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/dashboard.md
- docs/design-docs/guardrails.md
- docs/design-docs/harness-glossary.md
- docs/design-docs/harness-operating-model.md
- docs/design-docs/local-inference.md
- docs/design-docs/release-versioning.md
- docs/design-docs/self-reflective-telemetry.md
- docs/validation/README.md
- docs/validation/agent-smoke/README.md
- docs/product-specs/product-surface.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-007-guardrails-and-safety.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-012-self-improvement-loop.md
- docs/roles/ROLES.md
*/
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGuardrailsSecretScanRedactsAndSkipsLocalEnv(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".harness"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte(`api_`+`key = "sk_`+`live_abcdefghijklmnop"`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".harness", ".env.local"), []byte(`ANTHROPIC_API_`+`KEY=sk_`+`live_should_not_report_from_local_env`), 0o600))

	findings, err := runCLISecretScan(repo, false)
	require.NoError(t, err)
	require.Len(t, findings, 1)
	require.Equal(t, "README.md", findings[0].File)
	require.Equal(t, "[REDACTED]", findings[0].Match)
}

func TestGuardrailsInstallHooksIsIdempotent(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o755))

	path, changed, err := installSecretScanHook(repo)
	require.NoError(t, err)
	require.True(t, changed)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "BEGIN MARS SECRET SCAN")
	require.Contains(t, string(data), "guardrails secret-scan")

	_, changed, err = installSecretScanHook(repo)
	require.NoError(t, err)
	require.False(t, changed)
}
