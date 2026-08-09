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
			require.NoError(t, spec.ValidateProvenance(), "profile=%s tier=%s", p, tier)
		}
	}
}

func TestDefaultModels_exactArtifactProvenance(t *testing.T) {
	t.Parallel()

	want := map[string]struct {
		revision string
		size     int64
		sha256   string
	}{
		"Qwen3-Coder-30B-A3B-Instruct-Q3_K_L.gguf": {"b48fadd07cca9112bc27123e669b8bf55823013c", 14583005504, "ddad34d487a85c5a5872b422a15b1f3db196c7912ecd939e7e1ef373cbc7ef29"},
		"Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf": {"a000510ef6de0a66dafa731c2d8d712a96fa7009", 18632186176, "79ad15a5ee3caddc3f4ff0db33a14454a5a3eb503d7fa1c1e35feafc579de486"},
		"Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf":   {"e9eb3e611bdcd5842e021c014b392c70746da574", 32483934528, "a4a0207f4653bfece73d9818c83acf714f5593525fe3aab7026347fd73090fcc"},
		"google_gemma-4-E4B-it-Q4_K_M.gguf":        {"ada4143251234f041e9577f8415eb21c9b620885", 5405167904, "b937a48e96379116137c50acbe39fd1b46eb101d2df4e560f47f5e2171b6451e"},
		"google_gemma-4-E4B-it-Q5_K_M.gguf":        {"e4aa9542a0831b455713909211f97454c5812c5d", 5820881184, "8c2686257c840a1dcd4e6a3794a7e25c335cc5490a188d7f222b792bb5e82b4d"},
		"google_gemma-4-E4B-it-Q8_0.gguf":          {"62c51d90ba0d5499436edbf24b5247bf3aa9d509", 8031240480, "9c536ba17e55f3cf4d45aaa985bea7637f7b9034240b1377aca88d873aa6cb5c"},
	}

	got := make(map[string]ModelSpec)
	for _, profile := range []Profile{ProfileCPU, ProfileMedium, ProfileHigh} {
		for _, spec := range DefaultModels(profile) {
			if previous, ok := got[spec.File]; ok {
				require.Equal(t, previous.Provenance, spec.Provenance, "file=%s", spec.File)
				continue
			}
			got[spec.File] = spec
		}
	}
	require.Len(t, got, 6)
	for file, expected := range want {
		spec, ok := got[file]
		require.True(t, ok, "missing artifact %s", file)
		require.Equal(t, expected.revision, spec.Revision)
		require.Equal(t, expected.size, spec.SizeBytes)
		require.Equal(t, expected.sha256, spec.SHA256)
		require.NoError(t, spec.ValidateProvenance())
		require.Equal(t, ModelBaseRevisionNotPublished, spec.Provenance.BaseRevisionStatus)
		require.Empty(t, spec.Provenance.BaseRevision)
		require.Equal(t, ModelDistributionDownloadOnly, spec.Provenance.Distribution)
	}
}

func TestModelSpecValidateProvenance_rejectsIncompleteOrOverclaimedRecords(t *testing.T) {
	t.Parallel()

	valid := DefaultModels(ProfileMedium)[TierCoding]
	tests := map[string]func(*ModelSpec){
		"abbreviated artifact revision": func(spec *ModelSpec) { spec.Revision = "abc1234" },
		"missing size":                  func(spec *ModelSpec) { spec.SizeBytes = 0 },
		"invalid checksum":              func(spec *ModelSpec) { spec.SHA256 = "not-a-checksum" },
		"abbreviated evidence revision": func(spec *ModelSpec) {
			spec.Provenance.EvidenceRevision = "abc1234"
		},
		"missing base model": func(spec *ModelSpec) { spec.Provenance.DeclaredBaseRepo = "" },
		"insecure terms URL": func(spec *ModelSpec) { spec.Provenance.TermsURL = "http://example.invalid" },
		"abbreviated tool revision": func(spec *ModelSpec) {
			spec.Provenance.QuantizationToolRevision = "abc1234"
		},
		"inferred unpublished base revision": func(spec *ModelSpec) {
			spec.Provenance.BaseRevision = "b2cff646eb4bb1d68355c01b18ae02e7cf42d120"
		},
		"missing distribution boundary": func(spec *ModelSpec) { spec.Provenance.Distribution = "" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := valid
			mutate(&spec)
			require.Error(t, spec.ValidateProvenance())
		})
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
