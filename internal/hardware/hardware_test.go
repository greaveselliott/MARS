/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package hardware

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetect_returnsValidSummary(t *testing.T) {
	s := Detect()
	require.NotEmpty(t, s.OS)
	require.NotEmpty(t, s.Arch)
	require.Greater(t, s.CPUCores, 0)
	require.Greater(t, s.RAMMiB, 0)
	require.NotEmpty(t, string(s.Profile))
}

func TestSelectProfile_thresholds(t *testing.T) {
	tests := []struct {
		vram int
		want Profile
	}{
		{0, ProfileCPU},
		{4096, ProfileLow},
		{8192, ProfileMedium},
		{16384, ProfileHigh},
		{49152, ProfileHigh},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.vram)+"_"+string(tt.want), func(t *testing.T) {
			var gpus []GPU
			if tt.vram > 0 || tt.want != ProfileCPU {
				gpus = []GPU{{Index: 0, Name: "test", VRAMMiB: tt.vram, Driver: "test"}}
			}
			got := selectProfile(gpus)
			require.Equal(t, tt.want, got, "vram=%d", tt.vram)
		})
	}
}

func TestSelectProfile_multiGPU(t *testing.T) {
	gpus := []GPU{
		{Index: 0, VRAMMiB: 4096},
		{Index: 1, VRAMMiB: 4096},
	}
	require.Equal(t, ProfileMulti, selectProfile(gpus))
}

func TestDefaultModels_allTiersCovered(t *testing.T) {
	for _, p := range []Profile{ProfileCPU, ProfileLow, ProfileMedium, ProfileHigh, ProfileMulti} {
		m := DefaultModels(p)
		require.Contains(t, m, TierCoding)
		require.Contains(t, m, TierReasoning)
		require.Contains(t, m, TierFast)
		for tier, spec := range m {
			require.NotEmpty(t, spec.Name, "profile=%s tier=%s", p, tier)
			require.NotEmpty(t, spec.File, "profile=%s tier=%s", p, tier)
			require.NotEmpty(t, spec.Revision, "profile=%s tier=%s", p, tier)
			require.NotEmpty(t, spec.SHA256, "profile=%s tier=%s", p, tier)
			require.NotContains(t, spec.DownloadURL(), "/resolve/main/")
		}
	}
}

func TestDefaultModelsForPerformance_capsHighProfiles(t *testing.T) {
	t.Parallel()

	quality := DefaultModelsForPerformance(ProfileHigh, PerformanceQuality)
	require.Equal(t, "Q8_0", quality[TierCoding].Quant)

	balanced := DefaultModelsForPerformance(ProfileHigh, PerformanceBalanced)
	require.Equal(t, "Q4_K_M", balanced[TierCoding].Quant)
	require.Equal(t, "Q5_K_M", balanced[TierFast].Quant)

	speed := DefaultModelsForPerformance(ProfileHigh, PerformanceSpeed)
	require.Equal(t, "Q3_K_L", speed[TierCoding].Quant)
	require.Equal(t, "Q4_K_M", speed[TierFast].Quant)
}

func TestDefaultModelsForHardware_autoBalancesAppleSilicon(t *testing.T) {
	t.Parallel()

	hw := Summary{
		Profile: ProfileHigh,
		RAMMiB:  64 * 1024,
		GPUs:    []GPU{{Name: "Apple Silicon", Driver: "Metal", VRAMMiB: 64 * 1024}},
	}
	models := DefaultModelsForHardware(hw, PerformanceAuto)
	require.Equal(t, "Q4_K_M", models[TierCoding].Quant)
	require.Equal(t, PerformanceBalanced, EffectivePerformanceProfile(hw, PerformanceAuto))
}

func TestDefaultModelsForHardware_autoKeepsQualityForLargeDedicatedGPU(t *testing.T) {
	t.Parallel()

	hw := Summary{
		Profile: ProfileHigh,
		RAMMiB:  128 * 1024,
		GPUs:    []GPU{{Name: "RTX 6000", Driver: "CUDA", VRAMMiB: 48 * 1024}},
	}
	models := DefaultModelsForHardware(hw, PerformanceAuto)
	require.Equal(t, "Q8_0", models[TierCoding].Quant)
	require.Equal(t, PerformanceQuality, EffectivePerformanceProfile(hw, PerformanceAuto))
}

func TestNormalizePerformanceProfile(t *testing.T) {
	t.Parallel()

	require.Equal(t, PerformanceAuto, NormalizePerformanceProfile(" auto "))
	require.Equal(t, PerformanceBalanced, NormalizePerformanceProfile(" balanced "))
	require.Equal(t, PerformanceSpeed, NormalizePerformanceProfile("SPEED"))
	require.Equal(t, PerformanceAuto, NormalizePerformanceProfile("unknown"))
	require.Equal(t, PerformanceAuto, NormalizePerformanceProfile(""))
}
