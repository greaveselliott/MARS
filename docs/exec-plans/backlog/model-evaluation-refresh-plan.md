# Model Evaluation Refresh Plan

**Status:** Backlog (initial MH-030 slice complete; future expansion needs a new ticket)
**Priority:** P4
**Depends On:** Higher-priority release/quality work for any future promotion ticket
**Blocks:** Default model registry promotion until live benchmark evidence exists
**Related Tickets:** MH-030
**Goals:** G-003
**BDD Feature:** F-001
**Hypothesis:** Model changes should be promoted only when benchmark and integration evidence proves they improve scenario delivery without breaking tool calls, speed, or memory constraints.
**Success Evidence:** Candidate models are evaluated against repo-backed scenarios, hardware metrics, tool-call reliability, and pinned artifact requirements before default promotion.
**Falsification Evidence:** A model is promoted from novelty or catalog freshness without scenario evidence, checksum metadata, or throughput/memory proof.
**Scenario Schedule:** Future model-evaluation feature contract; temporarily linked to F-001-S005 because quality/release evidence must distinguish enabler benchmark work from shipped product behavior.
**Current Failing Scenario:** Waiting for active-plan promotion after higher-priority operating-model and release-asset work.
**Walking Skeleton Slice:** Run one candidate through the smallest benchmark suite that exercises tool-call JSON, strict JSON output, and one repo-backed edit/test loop.
**Learning Or MVP Outcome:** Learn whether a candidate model should become an explicit user override, a pinned default, or rejected evidence.
**Updated:** 2026-05-02
**Owner:** MARS maintainers
**Sources:** [May 2026 model landscape](../../references/model-landscape-may-2026.md), [local inference design](../../design-docs/local-inference.md)

## Goal

Make "best model" an evidence-backed, repeatable harness process instead of another hardcoded snapshot.

The initial product slice shipped through `MH-030`: model evaluation now supports
Ollama catalog access, explicit tier/role overrides, persisted reports,
repo-backed ticket-completion JSON, and promotion blocking. Future model-refresh
work should use a new ticket and promote only the specific benchmark or default
promotion slice into `../active/current-operating-plan.md`.

## Current Slice

- [x] Capture May 2026 model shortlist.
- [x] Add `mars models evaluate` planning output.
- [x] Add initial mechanical benchmark cases for tool-call JSON and strict JSON output.
- [x] Document benchmark-backed promotion as an architecture decision.
- [x] Add Ollama provider/catalog support for local model listing and evaluation.
- [x] Add explicit tier/role override support through `.harness/model-overrides.yaml`.
- [x] Persist live evaluation reports under `docs/generated/model-evaluations/`.
- [x] Add a repo-backed ticket-completion JSON benchmark case.
- [x] Add a promotion report that blocks unpinned or cloud-only defaults.

## Next Slices

- [ ] Add override diagnostics:
  - [ ] surface missing explicit overrides through doctor/setup
- [ ] Add deeper repo-backed benchmark cases using temporary target repos:
  - [ ] brownfield bug diagnosis
  - [ ] small code edit plus test run
  - [ ] multiple in-progress ticket prioritization
  - [ ] guardrail violation refusal
  - [ ] context-glossary retrieval
- [ ] Add hardware metrics:
  - [ ] wall time
  - [ ] tokens/sec
  - [ ] peak RSS where available
  - [ ] model file size
  - [ ] effective context length
- [ ] Add a promotion report that compares candidates against the current registry defaults.
- [ ] Add `mars models promote --candidate <name>` only after pinned revision/SHA256 metadata exists.
- [ ] Update default registry entries only after benchmark evidence supports the change.

## Candidate Run Order

1. Current Qwen3-Coder default.
2. Qwen3.6 35B-A3B coding variant.
3. Qwen3.6 27B variant.
4. Laguna XS.2.
5. Optional remote/cloud upper bounds: GLM-5.1, Mistral Medium 3.5, Kimi K2.6, DeepSeek V4.

## Quality Gate

A model can become a default only when:

- benchmark pass rate is at least as good as the current default for the relevant tier
- mutating-role tool-call JSON is reliable
- structured triage output is reliable
- speed and memory fit the target hardware profile
- artifact source revision and SHA256 are pinned
- ad-hoc Ollama selections are promoted only after becoming pinned default artifacts or explicit operator-owned overrides
- rationale is recorded in `docs/design-docs/local-inference.md` and product specs
