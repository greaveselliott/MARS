package hardware

import (
	"crypto/sha256"
	"encoding/hex"
)

// ModelSpec describes one downloadable model.
type ModelSpec struct {
	Name       string
	Repo       string // HuggingFace repo ID
	File       string // GGUF filename within repo
	Params     string // e.g. "27B"
	Quant      string // e.g. "Q4_K_M"
	RAMMinMiB  int    // minimum RAM to load
	ContextLen int    // default context length
	SHA256     string // expected checksum
}

// Tier maps roles to model weight classes.
type Tier string

const (
	TierCoding    Tier = "coding"    // Primary coding model
	TierReasoning Tier = "reasoning" // Deep reasoning
	TierFast      Tier = "fast"      // Quick tasks
)

func specSHA(label string) string {
	sum := sha256.Sum256([]byte("mars-harness:placeholder:" + label))
	return hex.EncodeToString(sum[:])
}

// DefaultModels returns the default model set for a given hardware profile.
func DefaultModels(p Profile) map[Tier]ModelSpec {
	switch p {
	case ProfileCPU, ProfileLow:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-Next",
				Repo:       "Qwen/Qwen3-Coder-Next-GGUF",
				File:       "qwen3-coder-next-30b-q4_k_s.gguf",
				Params:     "30B",
				Quant:      "Q4_K_S",
				RAMMinMiB:  20000,
				ContextLen: 32768,
				SHA256:     specSHA("coding-cpu-low-q4ks"),
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-Next",
				Repo:       "Qwen/Qwen3-Coder-Next-GGUF",
				File:       "qwen3-coder-next-30b-q4_k_s.gguf",
				Params:     "30B",
				Quant:      "Q4_K_S",
				RAMMinMiB:  20000,
				ContextLen: 65536,
				SHA256:     specSHA("reasoning-cpu-low-q4ks"),
			},
			TierFast: {
				Name:       "Gemma 4 Mini",
				Repo:       "google/gemma-4-mini-GGUF",
				File:       "gemma-4-mini-q4_k_s.gguf",
				Params:     "4B",
				Quant:      "Q4_K_S",
				RAMMinMiB:  4096,
				ContextLen: 8192,
				SHA256:     specSHA("fast-cpu-low-q4ks"),
			},
		}
	case ProfileMedium:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-Next",
				Repo:       "Qwen/Qwen3-Coder-Next-GGUF",
				File:       "qwen3-coder-next-30b-q4_k_m.gguf",
				Params:     "30B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  22000,
				ContextLen: 32768,
				SHA256:     specSHA("coding-medium-q4km"),
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-Next",
				Repo:       "Qwen/Qwen3-Coder-Next-GGUF",
				File:       "qwen3-coder-next-30b-q4_k_m.gguf",
				Params:     "30B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  22000,
				ContextLen: 131072,
				SHA256:     specSHA("reasoning-medium-q4km"),
			},
			TierFast: {
				Name:       "Gemma 4 Mini",
				Repo:       "google/gemma-4-mini-GGUF",
				File:       "gemma-4-mini-q4_k_m.gguf",
				Params:     "4B",
				Quant:      "Q4_K_M",
				RAMMinMiB:  5120,
				ContextLen: 8192,
				SHA256:     specSHA("fast-medium-q4km"),
			},
		}
	case ProfileHigh, ProfileMulti:
		return map[Tier]ModelSpec{
			TierCoding: {
				Name:       "Qwen3-Coder-Next",
				Repo:       "Qwen/Qwen3-Coder-Next-GGUF",
				File:       "qwen3-coder-next-30b-q5_k_m.gguf",
				Params:     "30B",
				Quant:      "Q5_K_M",
				RAMMinMiB:  26000,
				ContextLen: 32768,
				SHA256:     specSHA("coding-high-q5km"),
			},
			TierReasoning: {
				Name:       "Qwen3-Coder-Next",
				Repo:       "Qwen/Qwen3-Coder-Next-GGUF",
				File:       "qwen3-coder-next-30b-q5_k_m.gguf",
				Params:     "30B",
				Quant:      "Q5_K_M",
				RAMMinMiB:  26000,
				ContextLen: 131072,
				SHA256:     specSHA("reasoning-high-q5km"),
			},
			TierFast: {
				Name:       "Gemma 4 Mini",
				Repo:       "google/gemma-4-mini-GGUF",
				File:       "gemma-4-mini-q5_k_m.gguf",
				Params:     "4B",
				Quant:      "Q5_K_M",
				RAMMinMiB:  6144,
				ContextLen: 8192,
				SHA256:     specSHA("fast-high-q5km"),
			},
		}
	default:
		return DefaultModels(ProfileCPU)
	}
}
