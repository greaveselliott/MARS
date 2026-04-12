package hardware

import "fmt"

// ModelSpec describes one downloadable model.
type ModelSpec struct {
	Name       string
	Repo       string // HuggingFace repo ID (e.g. "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF")
	File       string // GGUF filename within repo
	Params     string // e.g. "30B-A3B"
	Quant      string // e.g. "Q4_K_M"
	RAMMinMiB  int    // minimum RAM/VRAM to load
	ContextLen int    // default context length
	SHA256     string // expected checksum (empty = skip verification, computed on first download)
}

// DownloadURL returns the HuggingFace direct download URL for this model.
func (s ModelSpec) DownloadURL() string {
	if s.Repo == "" || s.File == "" {
		return ""
	}
	return fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", s.Repo, s.File)
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
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q3_K_L.gguf",
				Params:     "30B-A3B",
				Quant:      "Q3_K_L",
				RAMMinMiB:  8192,
				ContextLen: 32768,
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q3_K_L.gguf",
				Params:     "30B-A3B",
				Quant:      "Q3_K_L",
				RAMMinMiB:  8192,
				ContextLen: 65536,
			},
			TierFast: {
				Name:       "Gemma 4 E4B",
				Repo:       "bartowski/google_gemma-4-E4B-it-GGUF",
				File:       "google_gemma-4-E4B-it-Q4_K_M.gguf",
				Params:     "4B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  3072,
				ContextLen: 8192,
			},
		}
	case ProfileMedium:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf",
				Params:     "30B-A3B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  12288,
				ContextLen: 32768,
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf",
				Params:     "30B-A3B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  12288,
				ContextLen: 131072,
			},
			TierFast: {
				Name:       "Gemma 4 E4B",
				Repo:       "bartowski/google_gemma-4-E4B-it-GGUF",
				File:       "google_gemma-4-E4B-it-Q5_K_M.gguf",
				Params:     "4B",
				Quant:      "Q5_K_M",
				RAMMinMiB:  4096,
				ContextLen: 8192,
			},
		}
	case ProfileHigh, ProfileMulti:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf",
				Params:     "30B-A3B",
				Quant:      "Q8_0",
				RAMMinMiB:  24576,
				ContextLen: 32768,
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-30B-A3B-Instruct",
				Repo:       "lmstudio-community/Qwen3-Coder-30B-A3B-Instruct-GGUF",
				File:       "Qwen3-Coder-30B-A3B-Instruct-Q8_0.gguf",
				Params:     "30B-A3B",
				Quant:      "Q8_0",
				RAMMinMiB:  24576,
				ContextLen: 131072,
			},
			TierFast: {
				Name:       "Gemma 4 E4B",
				Repo:       "bartowski/google_gemma-4-E4B-it-GGUF",
				File:       "google_gemma-4-E4B-it-Q8_0.gguf",
				Params:     "4B",
				Quant:      "Q8_0",
				RAMMinMiB:  8192,
				ContextLen: 8192,
			},
		}
	default:
		return DefaultModels(ProfileCPU)
	}
}

// UniqueModels deduplicates the model set (coding and reasoning often share the same file).
func UniqueModels(models map[Tier]ModelSpec) []ModelSpec {
	seen := make(map[string]bool)
	var unique []ModelSpec
	for _, spec := range models {
		key := spec.Repo + "/" + spec.File
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, spec)
	}
	return unique
}
