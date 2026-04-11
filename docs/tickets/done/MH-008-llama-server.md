---
id: MH-008
title: Implement llama.cpp server lifecycle management
priority: high
complexity: large
source: delivery-schedule M2.4, M2.5
created: 2026-04-11
---

# MH-008: Implement llama.cpp server lifecycle management and LLM router

## Context

The harness manages llama.cpp as a subprocess (AD-007). Users never interact with the inference server directly. The router maps roles to model endpoints.

Reference: [docs/design-docs/local-inference.md](../../design-docs/local-inference.md) (AD-007)

## Requirements

### Server management (`internal/inference/server.go`)
- Download llama-server binary for detected OS/arch during setup
- Start with correct model path, port, GPU layers, context size
- Health check: poll `/health` endpoint until ready (with timeout)
- Restart: detect process exit, restart with exponential backoff
- Shutdown: graceful SIGTERM, then SIGKILL after timeout
- Multi-model: start multiple llama-server instances on different ports for different model tiers
- Log capture: forward llama-server stdout/stderr to structured log

### LLM Router (`internal/llm/router.go`)
- Route LLM requests to correct endpoint based on role → model tier → endpoint mapping from manifest
- Fallback: if local endpoint is unhealthy and cloud endpoint is configured, route to cloud
- Health awareness: check endpoint health before routing, skip unhealthy endpoints

## Acceptance Criteria

### Functional
- [ ] llama-server binary downloaded for correct OS/arch
- [ ] Server starts and `/health` returns OK
- [ ] Agent runtime from M1 works end-to-end with real local model
- [ ] Router maps roles to correct endpoints based on manifest
- [ ] Multi-model: two servers on different ports serving different models

### Edge cases
- [ ] Server crashes → restarted automatically with backoff
- [ ] Server fails to start (wrong model path) → descriptive error with fix suggestion
- [ ] Health check timeout → server reported as unhealthy, fallback to cloud if configured
- [ ] Shutdown while inference is in progress → waits for current request to complete

### Non-goals
- [ ] Model download (MH-007 handles that)
- [ ] Hardware detection (MH-006 handles that)

## Notes

Integration test: hardware detect → model download → server start → agent loop with real model → file created. This is the end-to-end proof that local inference works.
