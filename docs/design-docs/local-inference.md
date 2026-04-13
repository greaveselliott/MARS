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

### AD-031: Inference resilience — timeouts, context headroom, and health verification

Three failure modes observed in production pipeline runs (crowd-runner, April 2026):

**1. HTTP client timeout (60s → 5 min).** A 30B Q8 model on Apple Silicon routinely exceeds 60 seconds per completion on complex multi-turn prompts. Local inference has zero per-second cost, so generous timeouts are correct. Default changed from 60s to 5 minutes. Retry count increased from 3 to 5 with backoff ceiling raised from 5s to 15s. Agent loop `chatWithRetries` backoff also increased from 100ms–2s to 2s–15s.

**2. Fast-tier context headroom (8192 → 16384).** Gemma 4 E4B was configured with 8192 context across all hardware profiles. Assembled role prompts with ticket indices exceeded this (8442 tokens observed). Gemma 4 natively supports 128k. Increased all fast-tier profiles to 16384 — sufficient for any role prompt while keeping memory modest. Also added explicit non-retryable error detection for context-exceeded responses (HTTP 400 with "exceed" + "context" in body) so the client doesn't waste retries on prompts that will never fit.

**3. Stale health state (connection refused).** `ServerForRole` returned immediately when a server was previously marked `StateHealthy` without verifying it was still alive. If the server crashed between jobs, the next job got `connection refused`. Added an active `/health` spot-check after `Start` returns. If the server fails the check, it's torn down and restarted before the endpoint is returned. This closes the race window between the supervisor detecting a crash and the next job claiming the "healthy" server.

### Open topics (M2 and beyond)

- **Hardware detection:** CPU vs GPU paths, memory ceilings, and safe default model bundles; degrade gracefully when VRAM is insufficient.
- **Model registry:** naming, versioning, compatibility with server flags, deprecation notices in CLI output.
- **Download with resume:** partial files, checksum retry, bandwidth-friendly defaults; mirror URLs as optional fallback.
- **Server lifecycle:** start/stop, backoff on crash, upgrade without orphan processes; pidfile or equivalent for operator tooling.
- **Multi-model serving:** concurrent endpoints vs serial reuse; resource isolation when two roles need different sizes.

## Discoveries

- **Local inference timeout math:** On Apple M1 Max (64GB), Qwen3-Coder-30B-A3B Q8_0 with 32k context can take 2–4 minutes per completion when generating long multi-tool responses. The 60s default was set assuming cloud API speeds. Any timeout below 3 minutes will produce false-positive timeouts on complex engineer turns.
- **Fast-tier context floor:** Role prompts with ticket indices typically assemble to 5000–9000 tokens. Any context window below 12k risks overflow on mature projects with many tickets. 16k provides comfortable headroom.
- **Health check race window:** The supervisor restart loop (exponential backoff 1s→30s) can leave a 1–30 second gap where `State()` returns `StateHealthy` but the process is dead. Active verification on every `ServerForRole` call is cheap (2s timeout HTTP GET) and closes this gap completely.
