---
id: MH-059
title: Add Available Models and dashboard model override proposals
priority: medium
complexity: large
kind: standard
work_type: feature
bdd_scenarios: ["F-010-S020"]
end_to_end_evidence: required
evidence_links: []
verified_by: "pending implementation evidence"
owner: "Orchestrator backlog"
last_attempt: none
blocker: none
blocked_by: ["MH-053", "MH-054", "MH-055"]
trace_id: none
next_action: "Unify registry, cache, endpoint, Ollama, and configured provider data into an authenticated model catalog contract."
dedupe_key: dashboard-control-plane:available-models-overrides
source: user request 2026-05-20
created: 2026-05-20
depends_on: ["MH-053", "MH-054", "MH-055"]
---

# MH-059: Add Available Models and dashboard model override proposals

## Context

The dashboard needs to list offline and cloud-hosted models currently available
to Mars Harness and provide a safe path to add models or propose overrides.
Visibility is not the same as default promotion; benchmark and validation
evidence still matters.

## BDD Scenario IDs

- F-010-S020

## Affected Docs/Code Areas

- future `web/dashboard/`
- `internal/dashboard/`
- `internal/models/`
- `internal/inference/`
- `internal/llm/`
- `internal/serve/`
- `cmd/mars-harness/`
- `.harness/model-overrides.yaml` handling
- `docs/features/F-010-dashboard-control-plane.md`
- `docs/product-specs/dashboard-control-plane.md`

## Acceptance Criteria

- [ ] Available Models lists pinned registry models, cached model files, reachable llama.cpp endpoints, Ollama models, OpenAI-compatible endpoints, and configured cloud providers when present.
- [ ] Each model shows provider, name, local/cloud kind, endpoint, health, context window when known, role or tier eligibility, benchmark evidence when available, current usage, and unavailable reason.
- [ ] Add-model and routing-change actions create explicit override or registry-change proposals with validation requirements.
- [ ] `serve` and `start` can honor approved model override configuration for dashboard-visible routing.
- [ ] The dashboard never promotes a model default solely because it is reachable.
- [ ] Secret values for provider credentials are never rendered.

## Non-Goals

- Downloading every listed model.
- Making cloud providers mandatory.
- Changing default model registry policy without evaluation evidence.
- Exposing provider secrets.

## Evidence Requirements

- Fixture tests for registry, cache, llama.cpp endpoint, Ollama, OpenAI-compatible endpoint, cloud provider, disabled provider, and missing credential states.
- Override proposal tests with validation-plan output.
- `serve` and `start` override support tests when approved override config exists.
- Browser verification for catalog, filters, health states, and proposal preview.

## Next Action

Inventory existing `models list`, `models override`, registry, cache, Ollama,
and endpoint code paths, then define the dashboard model catalog API.
