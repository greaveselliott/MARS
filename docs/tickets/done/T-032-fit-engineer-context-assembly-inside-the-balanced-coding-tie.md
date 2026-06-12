---
id: T-032
title: Fit engineer context assembly inside the balanced coding-tier window for package-managed frontend targets
priority: P1
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md#run-2-v05011-context-fix", "docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md#run-2-v05011-context-fix", "docs/validation/baselines/2026-06-12-factory-pace-baseline.md", "docs/design-docs/context-efficiency.md"]
verified_by: "Live demo-12 and demo-13 lifecycle replays on mars-harness 0.50.11 (fix commit bee4f5b, tag v0.50.11), same balanced model identity as the wedged baselines; zero context_overflow events across 9 + 8 jobs with 2 + 12 budget-pruner engagements at the exact former wedge states; lifecycle reach exceeded both wedged baselines"
owner: "foundation-maintainer"
last_attempt: >-
  2026-06-12: fixed as AD-288 (one slice, two bounded changes). Root cause:
  llm.EstimateTokens used ~4 chars/token while the Qwen3-Coder tokenizer
  measures ~3.15 chars/token on package-managed JS content (demo-12 job
  0b93881f trace: estimate 26,188 vs 33,281 served), so the agent-loop
  pruner never fired before llama.cpp rejected the request; the loop also
  budgeted from a hardcoded default window instead of the served tier
  window. Fix: (1) calibrate the estimator to 3 chars/token with the wedge
  measurement as a regression-floor test; (2) wire the router's per-tier
  served context window into the loop budget, return a typed
  ContextSizeError carrying n_prompt_tokens/n_ctx, never retry it verbatim,
  and clamp-prune-retry inside the loop so overflow cannot fail a job while
  prunable history remains. Coding-tier ctx deliberately not raised (would
  have changed AD-285 model identity mid-validation and masked the budget
  fix). Replays per AD-284: demo-12 reached T-001 done + T-002 feature
  commit (was: T-001 wedged in progress); demo-13 reached multiple product
  commits + T-001 evidence cycles toward done (was: one commit then both
  engineer jobs overflowed). Zero context_overflow in both; residual
  max_turns/guardrail churn recorded as evidence for T-031/AD-286 scope.
blocker: "none"
blocked_by: []
trace_id: "tr-1781225306000294000"
kind: intervention-debt
source: weekly-priorities.md
created: 2026-06-12
depends_on: []
---

# T-032: Fit engineer context assembly inside the balanced coding-tier window for package-managed frontend targets

During the 2026-06-12 demo-12 package-managed-frontend baseline (docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md, finding F1), engineer prompts crossed the balanced coding tier 32768-token window (33281 and 32883 prompt tokens) once the Vite/React scaffold existed, and llama.cpp rejected the requests as non-retryable exceed_context_size_error. The wedge reproduced across an operator retry and blocks the archetype past initial scaffolding. Options: trim context assembly to fit the configured tier window (Tenet 9), route oversized engineer turns to the 131k reasoning endpoint, or raise the balanced coding ctx. The heavy-model probe of the same archetype wedged engineer differently (dependency_sync ENOENT against a nested project dir) — both belong to this archetype gap.

2026-06-12 update: the demo-13 existing-repo-maintenance baseline
(docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md, finding
F1) reproduced the identical overflow (32,923 prompt tokens vs 32,768 ctx)
on both engineer jobs against an existing Phaser/Tetris project. The wedge
generalizes to any package-managed JS target with a non-trivial file tree,
so the fix must replay both archetypes per AD-284 before claiming
improvement.

## Resolution (2026-06-12, AD-288, v0.50.11)

Chosen option: trim context assembly to fit the served tier window
(Tenet 9), made impossible-by-construction with a server-reported clamp;
the 131k rerouting and ctx-raise options were rejected for this slice (see
AD-288 in docs/design-docs/context-efficiency.md for rationale). Both
AD-284-required archetype replays passed with zero context_overflow and
lifecycle reach beyond the wedged baselines; the replay runs are the new
archetype pace baselines. Foundation-owned; no mirrored-doctrine change
(runtime behavior only — generated target guidance does not name token
estimation or window wiring).
