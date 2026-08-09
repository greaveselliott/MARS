/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
*/
package setup

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDownloadPlanIsStableUniqueCompleteAndReadOnly(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	cfg := Config{LocalBundle: models.LocalBundleCPU}
	hw := hardware.Summary{Profile: hardware.ProfileCPU, RAMMiB: 16_384, OS: "darwin", Arch: "arm64"}

	first, err := resolveDownloadPlan(baseDir, cfg, hw, "macos-arm64")
	require.NoError(t, err)
	second, err := resolveDownloadPlan(baseDir, cfg, hw, "macos-arm64")
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, models.LocalBundleCPU, first.LocalBundle)

	bundle, _, err := models.ResolveLocalBundle(hw, models.LocalBundleCPU)
	require.NoError(t, err)
	require.Len(t, first.Artifacts, len(hardware.UniqueModels(bundle.Models))+1)

	identities := make([]string, 0, len(first.Artifacts))
	var total int64
	for _, artifact := range first.Artifacts {
		require.NoError(t, validatePlannedDownload(artifact))
		if artifact.Kind == downloadArtifactModel {
			assert.GreaterOrEqual(t, len(artifact.TermsOrNoticeURLs), 2)
		}
		identities = append(identities, artifact.Identity)
		total += artifact.SizeBytes
	}
	assert.True(t, sort.StringsAreSorted(identities))
	assert.Equal(t, total, first.TotalBytes)
	_, statErr := os.Stat(baseDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "planning must not create setup storage")
}

func TestResolvedDownloadPlanPinsConcreteBundleAcrossHardwareChanges(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	firstHardware := hardware.Summary{Profile: hardware.ProfileCPU, RAMMiB: 16_384, OS: "darwin", Arch: "arm64"}
	plan, err := resolveDownloadPlan(baseDir, Config{LocalBundle: models.LocalBundleAuto}, firstHardware, "macos-arm64")
	require.NoError(t, err)
	require.Equal(t, models.LocalBundleCPU, plan.LocalBundle)

	changedHardware := hardware.Summary{
		Profile: hardware.ProfileHigh,
		GPUs:    []hardware.GPU{{Name: "Apple Silicon", Driver: "Metal", VRAMMiB: 131_072}},
		RAMMiB:  131_072,
		OS:      "darwin",
		Arch:    "arm64",
	}
	_, automatic, err := models.ResolveLocalBundle(changedHardware, models.LocalBundleAuto)
	require.NoError(t, err)
	require.NotEqual(t, plan.LocalBundle, automatic.SelectedBundle)

	executionConfig := bindResolvedDownloadPlan(Config{LocalBundle: models.LocalBundleAuto}, plan)
	resolved, _, err := models.ResolveLocalBundle(changedHardware, executionConfig.LocalBundle)
	require.NoError(t, err)
	require.Equal(t, plan.LocalBundle, resolved.ID)
}

func TestResolveDownloadPlanIncludesOnlyMissingArtifacts(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	binDir := filepath.Join(baseDir, "bin")
	modelsDir := filepath.Join(baseDir, "models")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.MkdirAll(modelsDir, 0o755))
	binaryPath := filepath.Join(binDir, "llama-server")
	require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/sh\n: > \"$0.ran\"\necho b8833\n"), 0o755))

	hw := hardware.Summary{Profile: hardware.ProfileCPU, RAMMiB: 16_384, OS: "darwin", Arch: "arm64"}
	bundle, _, err := models.ResolveLocalBundle(hw, models.LocalBundleCPU)
	require.NoError(t, err)
	unique := hardware.UniqueModels(bundle.Models)
	require.NotEmpty(t, unique)
	require.NoError(t, os.WriteFile(filepath.Join(modelsDir, unique[0].File), []byte("already present"), 0o644))

	plan, err := resolveDownloadPlan(baseDir, Config{LocalBundle: models.LocalBundleCPU}, hw, "macos-arm64")
	require.NoError(t, err)
	_, ranErr := os.Stat(binaryPath + ".ran")
	assert.ErrorIs(t, ranErr, os.ErrNotExist, "download planning must not execute an existing llama-server")
	require.Len(t, plan.Artifacts, len(unique)-1)
	for _, artifact := range plan.Artifacts {
		assert.Equal(t, downloadArtifactModel, artifact.Kind)
		assert.NotContains(t, artifact.Identity, unique[0].File)
	}
	for _, spec := range unique[1:] {
		require.NoError(t, os.WriteFile(filepath.Join(modelsDir, spec.File), []byte("already present"), 0o644))
	}
	complete, err := resolveDownloadPlan(baseDir, Config{LocalBundle: models.LocalBundleCPU}, hw, "macos-arm64")
	require.NoError(t, err)
	assert.True(t, complete.Empty())
	assert.Equal(t, models.LocalBundleCPU, complete.LocalBundle)
}

func TestPendingModelDisappearanceCannotAddUnplannedDownload(t *testing.T) {
	modelsDir := t.TempDir()
	spec := hardware.DefaultModels(hardware.ProfileMedium)[hardware.TierCoding]
	modelPath := filepath.Join(modelsDir, spec.File)
	require.NoError(t, os.WriteFile(modelPath, []byte("present during planning"), 0o644))

	planned, err := plannedModelDownloads(modelsDir, []hardware.ModelSpec{spec})
	require.NoError(t, err)
	require.Empty(t, planned)
	require.NoError(t, os.Remove(modelPath))

	pending, err := pendingPlannedModels(modelsDir, []hardware.ModelSpec{spec}, DownloadPlan{Artifacts: planned})
	require.Error(t, err)
	assert.Empty(t, pending)
	assert.ErrorContains(t, err, "changed after acknowledgement")
	_, statErr := os.Stat(modelPath)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestDownloadPlanRejectsIncompleteProvenanceBeforeMutation(t *testing.T) {
	modelsDir := filepath.Join(t.TempDir(), "missing", "models")
	invalid := hardware.DefaultModels(hardware.ProfileMedium)[hardware.TierCoding]
	invalid.Provenance.TermsURL = ""

	_, err := plannedModelDownloads(modelsDir, []hardware.ModelSpec{invalid})
	require.Error(t, err)
	assert.ErrorContains(t, err, "incomplete provenance")
	_, statErr := os.Stat(filepath.Dir(modelsDir))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestDownloadPlanRejectsIncompleteLlamaRecordBeforeMutation(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	previous := pinnedLlamaCppRelease
	invalid := previous
	invalid.Platforms = make(map[string]llamaCppPlatformArtifact, len(previous.Platforms))
	for key, artifact := range previous.Platforms {
		invalid.Platforms[key] = artifact
	}
	artifact := invalid.Platforms["macos-arm64"]
	artifact.SHA256 = ""
	invalid.Platforms["macos-arm64"] = artifact
	pinnedLlamaCppRelease = invalid
	t.Cleanup(func() { pinnedLlamaCppRelease = previous })
	hw := hardware.Summary{Profile: hardware.ProfileCPU, RAMMiB: 16_384, OS: "darwin", Arch: "arm64"}

	_, err := resolveDownloadPlan(baseDir, Config{LocalBundle: models.LocalBundleCPU}, hw, "macos-arm64")
	require.Error(t, err)
	assert.ErrorContains(t, err, "incomplete provenance")
	_, statErr := os.Stat(baseDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestDownloadPlanNoDownloadModesNeedNoAcknowledgement(t *testing.T) {
	tests := []Config{
		{SkipDownload: true},
		{TestMode: true},
		{Inference: models.RoutingDefer},
		{Inference: models.RoutingCloud},
	}
	for _, cfg := range tests {
		plan, err := resolveDownloadPlan(filepath.Join(t.TempDir(), ".mars"), cfg, hardware.Summary{}, "ubuntu-x64")
		require.NoError(t, err)
		assert.True(t, plan.Empty())
	}
}

func TestRunResolvedSetupRejectsUnacknowledgedPlanBeforeWrites(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	plan := DownloadPlan{Artifacts: []DownloadArtifact{{Identity: "pending"}}}

	result, err := runResolvedSetup(baseDir, Config{}, plan)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "--download --yes")
	_, statErr := os.Stat(baseDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRunResolvedSetupRejectsChangedApprovedPlanBeforeWrites(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	approved := DownloadPlan{Artifacts: []DownloadArtifact{{Identity: "approved"}}}
	current := DownloadPlan{Artifacts: []DownloadArtifact{{Identity: "changed"}}}

	result, err := runResolvedSetup(baseDir, Config{ApprovedDownloadPlan: &approved}, current)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "plan changed")
	_, statErr := os.Stat(baseDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestResolveDownloadPlanKeepsLinuxLlamaAcquisitionDisabled(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mars")
	hw := hardware.Summary{Profile: hardware.ProfileCPU, RAMMiB: 16_384, OS: "linux", Arch: "amd64"}

	_, err := resolveDownloadPlan(baseDir, Config{LocalBundle: models.LocalBundleCPU}, hw, "ubuntu-x64")
	require.Error(t, err)
	assert.ErrorContains(t, err, "installation is unavailable")
	_, statErr := os.Stat(baseDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}
