/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-003-local-inference-lifecycle.md
*/
package hardware

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	PerformanceAuto     = "auto"
	PerformanceQuality  = "quality"
	PerformanceBalanced = "balanced"
	PerformanceSpeed    = "speed"

	ModelBaseRevisionNotPublished = "not_published"
	ModelBaseRevisionPublished    = "published"
	ModelDistributionDownloadOnly = "download_only"
)

// ModelProvenance records the publisher evidence and terms for a downloaded
// GGUF without claiming facts the publisher did not record.
type ModelProvenance struct {
	Publisher                  string
	EvidenceRevision           string
	DeclaredBaseRepo           string
	BaseRevision               string
	BaseRevisionStatus         string
	LicenseID                  string
	LicenseURL                 string
	TermsURL                   string
	QuantizedBy                string
	QuantizationToolRepo       string
	QuantizationToolRevision   string
	QuantizationToolLicenseURL string
	Distribution               string
}

// ModelSpec describes one downloadable model.
type ModelSpec struct {
	Name       string
	Repo       string // HuggingFace repo ID (e.g. "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF")
	Revision   string // immutable HuggingFace commit or tag
	File       string // GGUF filename within repo
	Params     string // e.g. "30B-A3B"
	Quant      string // e.g. "Q4_K_M"
	RAMMinMiB  int    // minimum RAM/VRAM to load
	ContextLen int    // default context length
	SHA256     string // expected checksum (empty = skip verification, computed on first download)
	SizeBytes  int64  // exact publisher-recorded artifact size
	Provenance ModelProvenance
}

var (
	qwenCoderProvenance = ModelProvenance{
		Publisher:                  "lmstudio-community",
		EvidenceRevision:           "1f4ceb1041258b3fbfe59e1175d1321c6b41863b",
		DeclaredBaseRepo:           "Qwen/Qwen3-Coder-30B-A3B-Instruct",
		BaseRevisionStatus:         ModelBaseRevisionNotPublished,
		LicenseID:                  "Apache-2.0",
		LicenseURL:                 "https://huggingface.co/Qwen/Qwen3-Coder-30B-A3B-Instruct/blob/b2cff646eb4bb1d68355c01b18ae02e7cf42d120/LICENSE",
		TermsURL:                   "https://huggingface.co/Qwen/Qwen3-Coder-30B-A3B-Instruct/blob/b2cff646eb4bb1d68355c01b18ae02e7cf42d120/LICENSE",
		QuantizedBy:                "bartowski",
		QuantizationToolRepo:       "ggml-org/llama.cpp",
		QuantizationToolRevision:   "00fa15fedc79263fa0285e6a3bbb0cfb3e3878a2",
		QuantizationToolLicenseURL: "https://github.com/ggml-org/llama.cpp/blob/00fa15fedc79263fa0285e6a3bbb0cfb3e3878a2/LICENSE",
		Distribution:               ModelDistributionDownloadOnly,
	}
	gemmaProvenance = ModelProvenance{
		Publisher:                  "bartowski",
		EvidenceRevision:           "029e94146666900b08caf49a3b47b413dfa8ec66",
		DeclaredBaseRepo:           "google/gemma-4-E4B-it",
		BaseRevisionStatus:         ModelBaseRevisionNotPublished,
		LicenseID:                  "Apache-2.0",
		LicenseURL:                 "https://huggingface.co/bartowski/google_gemma-4-E4B-it-GGUF/blob/029e94146666900b08caf49a3b47b413dfa8ec66/README.md",
		TermsURL:                   "https://ai.google.dev/gemma/docs/gemma_4_license",
		QuantizedBy:                "bartowski",
		QuantizationToolRepo:       "ggml-org/llama.cpp",
		QuantizationToolRevision:   "0893f50f2dc14fcc046e10d4f76a1ac7a62c0490",
		QuantizationToolLicenseURL: "https://github.com/ggml-org/llama.cpp/blob/0893f50f2dc14fcc046e10d4f76a1ac7a62c0490/LICENSE",
		Distribution:               ModelDistributionDownloadOnly,
	}
)

// ValidateProvenance rejects incomplete or overclaimed default-model records.
func (s ModelSpec) ValidateProvenance() error {
	if strings.TrimSpace(s.Repo) == "" || strings.TrimSpace(s.File) == "" {
		return fmt.Errorf("model provenance requires an artifact repository and filename")
	}
	if !isLowerHex(s.Revision, 40) {
		return fmt.Errorf("model provenance requires a full 40-character artifact revision")
	}
	if !isLowerHex(s.SHA256, 64) || s.SizeBytes <= 0 {
		return fmt.Errorf("model provenance requires an exact SHA256 and positive artifact size")
	}
	p := s.Provenance
	if strings.TrimSpace(p.Publisher) == "" || !strings.HasPrefix(s.Repo, p.Publisher+"/") || !isLowerHex(p.EvidenceRevision, 40) || strings.TrimSpace(p.DeclaredBaseRepo) == "" {
		return fmt.Errorf("model provenance requires publisher, evidence revision, and declared base model")
	}
	if strings.TrimSpace(p.LicenseID) == "" || !isHTTPS(p.LicenseURL) || !isHTTPS(p.TermsURL) {
		return fmt.Errorf("model provenance requires applicable license and terms")
	}
	if strings.TrimSpace(p.QuantizedBy) == "" || strings.TrimSpace(p.QuantizationToolRepo) == "" || !isLowerHex(p.QuantizationToolRevision, 40) || !isHTTPS(p.QuantizationToolLicenseURL) {
		return fmt.Errorf("model provenance requires quantizer and conversion-tool evidence")
	}
	switch p.BaseRevisionStatus {
	case ModelBaseRevisionNotPublished:
		if strings.TrimSpace(p.BaseRevision) != "" {
			return fmt.Errorf("model provenance must not infer an unpublished base revision")
		}
	case ModelBaseRevisionPublished:
		if !isLowerHex(p.BaseRevision, 40) {
			return fmt.Errorf("model provenance requires a full published base revision")
		}
	default:
		return fmt.Errorf("model provenance requires an explicit base revision status")
	}
	if p.Distribution != ModelDistributionDownloadOnly {
		return fmt.Errorf("model provenance requires the download-only distribution boundary")
	}
	return nil
}

func isLowerHex(value string, length int) bool {
	if len(value) != length || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func isHTTPS(value string) bool {
	return strings.HasPrefix(value, "https://") && len(strings.TrimPrefix(value, "https://")) > 0
}

// DownloadURL returns the HuggingFace direct download URL for this model.
func (s ModelSpec) DownloadURL() string {
	if s.Repo == "" || s.File == "" || s.Revision == "" {
		return ""
	}
	return fmt.Sprintf("https://huggingface.co/%s/resolve/%s/%s", s.Repo, s.Revision, s.File)
}

// Tier maps roles to model weight classes.
type Tier string

const (
	TierCoding    Tier = "coding"
	TierReasoning Tier = "reasoning"
	TierFast      Tier = "fast"
)

// DefaultModels returns the default model set for a given hardware profile.
// All models are single-file GGUFs from verified HuggingFace repos.
func DefaultModels(p Profile) map[Tier]ModelSpec {
	switch p {
	case ProfileCPU, ProfileLow:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				Revision:   "b48fadd07cca9112bc27123e669b8bf55823013c",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q3_K_L.gguf",
				Params:     "30B-A3B",
				Quant:      "Q3_K_L",
				RAMMinMiB:  8192,
				ContextLen: 32768,
				SHA256:     "ddad34d487a85c5a5872b422a15b1f3db196c7912ecd939e7e1ef373cbc7ef29",
				SizeBytes:  14583005504,
				Provenance: qwenCoderProvenance,
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				Revision:   "b48fadd07cca9112bc27123e669b8bf55823013c",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q3_K_L.gguf",
				Params:     "30B-A3B",
				Quant:      "Q3_K_L",
				RAMMinMiB:  8192,
				ContextLen: 65536,
				SHA256:     "ddad34d487a85c5a5872b422a15b1f3db196c7912ecd939e7e1ef373cbc7ef29",
				SizeBytes:  14583005504,
				Provenance: qwenCoderProvenance,
			},
			TierFast: {
				Name:       "Gemma 4 E4B",
				Repo:       "bartowski/google_gemma-4-E4B-it-GGUF",
				Revision:   "ada4143251234f041e9577f8415eb21c9b620885",
				File:       "google_gemma-4-E4B-it-Q4_K_M.gguf",
				Params:     "4B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  3072,
				ContextLen: 16384,
				SHA256:     "b937a48e96379116137c50acbe39fd1b46eb101d2df4e560f47f5e2171b6451e",
				SizeBytes:  5405167904,
				Provenance: gemmaProvenance,
			},
		}
	case ProfileMedium:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				Revision:   "a000510ef6de0a66dafa731c2d8d712a96fa7009",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf",
				Params:     "30B-A3B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  12288,
				ContextLen: 32768,
				SHA256:     "79ad15a5ee3caddc3f4ff0db33a14454a5a3eb503d7fa1c1e35feafc579de486",
				SizeBytes:  18632186176,
				Provenance: qwenCoderProvenance,
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				Revision:   "a000510ef6de0a66dafa731c2d8d712a96fa7009",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf",
				Params:     "30B-A3B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  12288,
				ContextLen: 131072,
				SHA256:     "79ad15a5ee3caddc3f4ff0db33a14454a5a3eb503d7fa1c1e35feafc579de486",
				SizeBytes:  18632186176,
				Provenance: qwenCoderProvenance,
			},
			TierFast: {
				Name:       "Gemma 4 E4B",
				Repo:       "bartowski/google_gemma-4-E4B-it-GGUF",
				Revision:   "e4aa9542a0831b455713909211f97454c5812c5d",
				File:       "google_gemma-4-E4B-it-Q5_K_M.gguf",
				Params:     "4B",
				Quant:      "Q5_K_M",
				RAMMinMiB:  4096,
				ContextLen: 16384,
				SHA256:     "8c2686257c840a1dcd4e6a3794a7e25c335cc5490a188d7f222b792bb5e82b4d",
				SizeBytes:  5820881184,
				Provenance: gemmaProvenance,
			},
		}
	case ProfileHigh, ProfileMulti:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				Revision:   "e9eb3e611bdcd5842e021c014b392c70746da574",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf",
				Params:     "30B-A3B",
				Quant:      "Q8_0",
				RAMMinMiB:  24576,
				ContextLen: 32768,
				SHA256:     "a4a0207f4653bfece73d9818c83acf714f5593525fe3aab7026347fd73090fcc",
				SizeBytes:  32483934528,
				Provenance: qwenCoderProvenance,
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				Revision:   "e9eb3e611bdcd5842e021c014b392c70746da574",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf",
				Params:     "30B-A3B",
				Quant:      "Q8_0",
				RAMMinMiB:  24576,
				ContextLen: 131072,
				SHA256:     "a4a0207f4653bfece73d9818c83acf714f5593525fe3aab7026347fd73090fcc",
				SizeBytes:  32483934528,
				Provenance: qwenCoderProvenance,
			},
			TierFast: {
				Name:       "Gemma 4 E4B",
				Repo:       "bartowski/google_gemma-4-E4B-it-GGUF",
				Revision:   "62c51d90ba0d5499436edbf24b5247bf3aa9d509",
				File:       "google_gemma-4-E4B-it-Q8_0.gguf",
				Params:     "4B",
				Quant:      "Q8_0",
				RAMMinMiB:  8192,
				ContextLen: 16384,
				SHA256:     "9c536ba17e55f3cf4d45aaa985bea7637f7b9034240b1377aca88d873aa6cb5c",
				SizeBytes:  8031240480,
				Provenance: gemmaProvenance,
			},
		}
	default:
		return DefaultModels(ProfileCPU)
	}
}

// DefaultModelsForPerformance returns models for the detected hardware profile
// adjusted by an operator-selected performance profile.
func DefaultModelsForPerformance(p Profile, performance string) map[Tier]ModelSpec {
	return DefaultModels(EffectiveModelProfile(p, performance))
}

// DefaultModelsForHardware returns the default model set after applying the
// operator's performance preference to the detected hardware.
func DefaultModelsForHardware(s Summary, performance string) map[Tier]ModelSpec {
	return DefaultModels(EffectiveModelProfile(s.Profile, EffectivePerformanceProfile(s, performance)))
}

// EffectivePerformanceProfile resolves "auto" to the profile the harness
// should use for this machine without requiring operator tuning.
func EffectivePerformanceProfile(s Summary, requested string) string {
	normalized := NormalizePerformanceProfile(requested)
	if normalized != PerformanceAuto {
		return normalized
	}
	return RecommendedPerformanceProfile(s)
}

// RecommendedPerformanceProfile chooses a safe default for local inference.
func RecommendedPerformanceProfile(s Summary) string {
	if s.Profile == ProfileHigh || s.Profile == ProfileMulti {
		if usesUnifiedMetalMemory(s) {
			if s.RAMMiB > 0 && s.RAMMiB < 96*1024 {
				return PerformanceBalanced
			}
			return PerformanceQuality
		}
		if largestGPUMiB(s.GPUs) < 48*1024 {
			return PerformanceBalanced
		}
	}
	return PerformanceQuality
}

// EffectiveModelProfile maps quality/speed preferences to an existing model
// registry profile without changing the detected hardware record.
func EffectiveModelProfile(p Profile, performance string) Profile {
	switch NormalizePerformanceProfile(performance) {
	case PerformanceAuto:
		return p
	case PerformanceSpeed:
		if p == ProfileCPU {
			return ProfileCPU
		}
		return ProfileLow
	case PerformanceBalanced:
		if p == ProfileHigh || p == ProfileMulti {
			return ProfileMedium
		}
		return p
	default:
		return p
	}
}

// NormalizePerformanceProfile returns the supported performance profile name.
func NormalizePerformanceProfile(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case PerformanceAuto, PerformanceBalanced, PerformanceSpeed, PerformanceQuality:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return PerformanceAuto
	}
}

func usesUnifiedMetalMemory(s Summary) bool {
	for _, gpu := range s.GPUs {
		name := strings.ToLower(gpu.Name)
		driver := strings.ToLower(gpu.Driver)
		if strings.Contains(driver, "metal") || strings.Contains(name, "apple silicon") {
			return true
		}
	}
	return false
}

func largestGPUMiB(gpus []GPU) int {
	var largest int
	for _, gpu := range gpus {
		if gpu.VRAMMiB > largest {
			largest = gpu.VRAMMiB
		}
	}
	return largest
}

// UniqueModels deduplicates the model set (coding and reasoning often share the same file).
func UniqueModels(models map[Tier]ModelSpec) []ModelSpec {
	seen := make(map[string]bool)
	var unique []ModelSpec
	for _, spec := range models {
		key := spec.Repo + "@" + spec.Revision + "/" + spec.File
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, spec)
	}
	return unique
}
