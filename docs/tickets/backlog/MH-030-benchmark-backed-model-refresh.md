---
id: MH-030
title: Benchmark-backed model refresh and promotion
priority: high
complexity: large
source: model-evaluation-refresh-plan.md
created: 2026-05-02
---

# MH-030: Benchmark-backed model refresh and promotion

## Context

The April 2026 model snapshot is stale. Ollama now lists new candidates such as Qwen3.6, Laguna XS.2, GLM-5.1, Kimi K2.6, DeepSeek V4, and Mistral Medium 3.5. Mars Harness must not change autonomous-agent defaults from newest-model claims alone. Defaults need benchmark evidence, hardware fit, and pinned artifacts.

Reference: [model evaluation refresh plan](../../exec-plans/active/model-evaluation-refresh-plan.md), [May 2026 model landscape](../../references/model-landscape-may-2026.md), [local inference AD-063](../../design-docs/local-inference.md).

## Requirements

- Extend `mars-harness models evaluate` beyond mechanical probes into repo-backed benchmark cases.
- Persist evaluation reports with model, endpoint, hardware profile, benchmark results, timing, token counts, and failures.
- Compare candidates against the current pinned defaults per tier.
- Add candidate metadata for Qwen3.6, Laguna XS.2, and optional cloud/remote candidates.
- Add a promotion report that says whether a candidate is safe to promote and why.
- Require immutable revision plus SHA256 before updating default registry entries.

## Affected Files

- `cmd/mars-harness/main.go`
- `internal/models/`
- `internal/hardware/registry.go`
- `docs/references/model-landscape-may-2026.md`
- `docs/design-docs/local-inference.md`
- `docs/product-specs/product-surface.md`
- `docs/exec-plans/active/model-evaluation-refresh-plan.md`

## Acceptance Criteria

### Functional (happy path)

- [ ] `mars-harness models evaluate --endpoint <url> --model <name> --json` writes a parseable report.
- [ ] The benchmark pack includes at least one repo-backed ticket completion task.
- [ ] Reports include pass/fail, latency, token counts, and enough metadata to reproduce the run.
- [ ] Candidate comparison identifies whether Qwen3.6 or Laguna XS.2 should replace any current tier default.
- [ ] Default registry update is blocked unless revision and SHA256 are present.

### Edge cases and negative paths

- [ ] Missing endpoint/model prints the plan and does not pretend evaluation happened.
- [ ] Endpoint failures produce actionable errors.
- [ ] Tool-call JSON failures are visible as failed benchmark cases.
- [ ] Cloud-only candidates cannot be promoted into local defaults.

### Non-goals

- Changing default model registry entries without benchmark evidence.
- Downloading large candidate models automatically during this ticket.

### Observability, docs, and regressions

- [ ] Docs explain why the chosen default changed or why it did not.
- [ ] Tests cover report JSON, failed benchmark cases, and promotion blocking.
- [ ] Product specs and design docs stay in sync with the implemented command behavior.
