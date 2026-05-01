---
id: MH-006
title: Implement GPU hardware detection and model registry
priority: high
complexity: medium
source: delivery-schedule M2.1, M2.2
created: 2026-04-11
---

# MH-006: Implement GPU hardware detection and model registry

## Context

The harness must auto-detect GPU hardware and select appropriate models without user configuration (tenet 1). This covers hardware profiling and the model catalog.

Reference: [docs/design-docs/local-inference.md](../../design-docs/local-inference.md) (AD-007, AD-008), [docs/references/model-landscape-april-2026.md](../../references/model-landscape-april-2026.md)

## Requirements

### Hardware detection (`internal/hardware/detect.go`)
- Detect NVIDIA GPU via `nvidia-smi` output parsing (GPU count, VRAM per GPU, driver version)
- Detect AMD via `rocm-smi` (GPU count, VRAM)
- Detect Apple Silicon via `sysctl` (unified memory)
- Select hardware profile: cpu / light / standard / full / multi

### Model registry (`internal/models/registry.go`)
- Catalog of supported models: name, size, quantization variants, VRAM required, HuggingFace URL, SHA256
- Profile-to-model mapping: each hardware profile maps to specific model(s)
- Model storage path: `~/.mars-harness/models/`

## Acceptance Criteria

### Functional
- [x] NVIDIA GPU correctly detected with VRAM on test machine (or reports "no GPU" in CI)
- [x] Hardware profile selected correctly based on VRAM thresholds
- [x] Registry returns correct models for each profile
- [x] Model storage path resolves correctly

### Edge cases
- [x] `nvidia-smi` not found → falls back to cpu profile (not crash)
- [x] `nvidia-smi` returns unexpected output format → logs warning, falls back to cpu
- [x] Multiple GPUs detected → multi profile with per-GPU VRAM

### Non-goals
- Model download (MH-007)
- llama.cpp server management (MH-008)

## Notes

Test hardware detection with mock `nvidia-smi` output fixtures (capture output from real machines, replay in tests).
