# Model Evaluation Refresh Plan

**Status:** Backlog
**Priority:** P4
**Updated:** 2026-05-02
**Owner:** Mars Harness maintainers
**Sources:** [May 2026 model landscape](../../references/model-landscape-may-2026.md), [local inference design](../../design-docs/local-inference.md)

## Goal

Make "best model" an evidence-backed, repeatable harness process instead of another hardcoded snapshot.

This plan is waiting behind higher-priority active-plan hygiene, Mars parity
ticket materialization, quality score export, and release asset work. Promote
the next model-refresh slice into `../active/current-operating-plan.md` before
execution.

## Current Slice

- [x] Capture May 2026 model shortlist.
- [x] Add `mars-harness models evaluate` planning output.
- [x] Add initial mechanical benchmark cases for tool-call JSON and strict JSON output.
- [x] Document benchmark-backed promotion as an architecture decision.

## Next Slices

- [ ] Add Ollama provider/catalog support:
  - [ ] list locally installed Ollama models
  - [ ] accept any Ollama model name as an evaluation candidate
  - [ ] represent provider/model overrides separately from default registry entries
- [ ] Add simple model swap support:
  - [ ] switch `fast`, `reasoning`, or `coding` tier for a repo
  - [ ] switch one role to an explicit provider/model override
  - [ ] surface missing explicit overrides through doctor/setup
- [ ] Add benchmark result persistence under the harness database or `docs/generated/model-evaluations/`.
- [ ] Add repo-backed benchmark cases using temporary target repos:
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
- [ ] Add `mars-harness models promote --candidate <name>` only after pinned revision/SHA256 metadata exists.
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
