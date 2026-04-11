# Local Inference

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

How the harness serves large language models locally: process boundaries, weight storage, verification, and operational lifecycle (download, health, upgrades).

## Context

Mars Harness targets **plug-and-play** local inference without requiring users to compile C/C++ or manage fragile native bindings in the main binary. Model artifacts are large and must not bloat the repository; integrity and reproducibility still matter for support and evolution.

The agent runtime ([agent-runtime.md](agent-runtime.md)) assumes a **stable HTTP or stdin/stdout contract** to the inference server; this document owns how that server is provisioned and supervised.

## Key Design Decisions

### AD-007: llama.cpp as subprocess, not embedded via CGO

The inference server runs **llama.cpp in a separate process** managed by the harness. The main Go binary stays **CGO-free**, simplifying distribution and cross-compilation. Trade-off: **two processes** to supervise, IPC, and coordinated shutdown—acceptable for clearer packaging and fewer toolchain failures on end-user machines.

The harness is responsible for **argv, env, working directory**, and capturing stderr for diagnostics when jobs fail for infra reasons.

### AD-008: Model weights outside the repo

Weights live under **`~/.mars-harness/models/`** (not committed). Expected hashes are recorded in **`bundle.lock.json`** (e.g. SHA256 per artifact) so installs and upgrades can verify downloads and detect drift.

Corrupt or partial downloads must never be loaded silently; verification runs **before** binding a model to active traffic.

### Open topics (M2 and beyond)

- **Hardware detection:** CPU vs GPU paths, memory ceilings, and safe default model bundles; degrade gracefully when VRAM is insufficient.
- **Model registry:** naming, versioning, compatibility with server flags, deprecation notices in CLI output.
- **Download with resume:** partial files, checksum retry, bandwidth-friendly defaults; mirror URLs as optional fallback.
- **Server lifecycle:** start/stop, backoff on crash, upgrade without orphan processes; pidfile or equivalent for operator tooling.
- **Multi-model serving:** concurrent endpoints vs serial reuse; resource isolation when two roles need different sizes.
- **Health checks:** readiness for jobs, latency probes, and failure surfacing to the pipeline engine queue.

## Discoveries

_(None yet.)_
