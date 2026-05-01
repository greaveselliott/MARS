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
