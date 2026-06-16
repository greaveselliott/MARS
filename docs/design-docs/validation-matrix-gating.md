# Validation Matrix Gating

**Status:** Accepted
**Date:** 2026-06-11
**Author:** foundation-maintainer

## Context

AD-138 defines the live demo improvement loop and a five-archetype validation
portfolio (static browser app, package-managed frontend, API or service,
CLI/tooling project, existing-repo maintenance), with informal selection
guidance: narrow fixes may replay one archetype, generic claims should run at
least two. The 2026-06-11 foundation review found that the informal rule lets
agents under-select replays for exactly the change classes most likely to
regress across project shapes, and that replay evidence is recorded
inconsistently, which makes "the rerun confirmed the fix" claims hard to
audit. T-028 (provisional T-040 in the foundation improvement plan) makes the
mapping from source-change class to minimum archetype replays mechanical
doctrine, and fixes the evidence-recording contract.

This is foundation validation doctrine. It governs how `mars-harness` source
changes are validated before lifecycle claims; it is **source-only** and is
not mirrored into generated target guidance (AGENTS.md rule 13 classification:
no generated-doctrine change required, recorded in T-028).

## Key Design Decisions

### AD-284: Source-Change Classes Gate Minimum Archetype Replays

Before claiming a source change improved or preserved factory lifecycle
behavior, classify the change and run at least the minimum archetype replays
from this table, or record the exact blocker and replay command. The
archetypes are AD-138's portfolio.

| Source-change class | Typical surfaces | Minimum archetype replays |
| --- | --- | --- |
| Tool policy | `internal/tools` policy and guards: shell/file/ticket/disposition rules, argv normalization, claim and convergence gates | Two archetypes: static browser app plus one of CLI/tooling or API/service (policy fires across shapes) |
| Role guidance | Generated role prompts, personas, bootstrap guidance in `internal/scanner` and `internal/personas` | The archetype whose guidance changed; cross-role or generic guidance changes need two archetypes |
| Orchestration | `internal/orchestration`, `internal/serve`, `internal/queue`, `internal/scheduler` dispatch, routing, convergence, survey behavior | Two archetypes: static browser app plus API/service or CLI/tooling (routing must not overfit one shape) |
| Release flow | `internal/release`, `internal/selfupdate`, release guards, release-note generation, publish/verify/audit | One CLI/tooling archetype replayed through its local release stage, plus the source repo's own release-note flow as live evidence |
| Scanner/generated doctrine | `mars-harness init`/`upgrade` defaults, mirrored docs, tools glossary, skills, target `AGENTS.md` | One fresh `init` replay of an affected archetype; upgrade-path changes add the existing-repo maintenance archetype |
| Model/provider behavior | `internal/llm` routing, model registry defaults, inference management | One archetype end-to-end on the affected model path, or the recorded blocker when hardware is unavailable |
| Quality export and telemetry rendering | `internal/qualityscore`, `internal/scoring` export/render surfaces with no runtime recording change | None beyond package tests; the next scheduled baseline replay doubles as live validation and the ticket must say so |
| Docs/doctrine only | Design docs, ADs, tickets, plans, glossary entries with no behavior change | None; documentation gates (`docsync`, docsconsistency) suffice |

Selection rules:

- A change spanning multiple classes takes the union of their minimums.
- "Replay" means a clean representative target run per AD-138 steps 2–8, not
  a unit-test pass.
- When a required replay cannot run (no GPU, no clean target, budget), the
  claim stays unconfirmed: record the blocker and the exact replay command in
  the validation report or the active plan, and do not mark the improvement
  as validated.

### AD-285: Validation Evidence Has A Fixed Recording Contract

Replay evidence is durable, discoverable, and uniform:

- **Report path:** `docs/validation/reports/YYYY-MM-DD-<target>-<purpose>.md`
  (the date is the date the report file is created; later runs append dated,
  anchor-linkable sections to the same file, as
  `2026-05-19-demo-123-live-lifecycle.md` already does).
- **Matrix run report:** whenever a validation matrix is run or attempted, the
  report names the selected matrix/suite, selected cases or archetypes, exact
  command, source ref or binary, model identity, target/run paths,
  DB/log/trace paths, per-case status, failure class, cleanup status, and the
  exact blocker or rerun command. Setup failures are still reportable matrix
  outcomes; do not leave them only in chat.
- **Primary outcome contract:** every report created on or after 2026-06-16
  names `Primary Outcome`, `Primary Pass Gate`, `Primary Status`,
  `Current Primary Blocker`, `Next Primary Action`, and `Supporting Evidence`
  before summary/result language. `Primary Status` must be one of
  `primary_passed`, `primary_failed`, `primary_blocked`, or `supporting_only`.
  If the primary pass gate is unmet, the report leads with
  `primary_failed`/`primary_blocked`; support evidence cannot be framed as the
  validation pass.
- **Required fields per run section** (from AD-138 step 3 and
  `docs/validation/README.md`): exact command, target path or remote, source
  ref or binary, **model identity (model name, quantization, context size,
  and the performance profile that resolved it) per inference tier used**,
  database/log paths, job sequence, target commits/tickets/docs produced,
  telemetry highlights, product progress reached, target intervention-debt
  count, runtime artifacts, and stop reason.
- **Model identity is part of the measurement contract:** pace baselines and
  pace-delta comparisons are only valid between runs on the same model
  identity. A model or quantization change invalidates prior pace baselines
  for Phase-style before/after claims; the affected baselines are
  reclassified evidence-only and re-captured on the new model (discovered
  2026-06-12 when the quality-profile Q8_0 weights maxed unified memory and
  the harness moved to the balanced model mid-measurement).
- **Fake endpoints are not validation evidence:** fake, stub, mock, canned, or
  scripted LLM endpoints may be used only for deterministic automated tests.
  Matrix reports backed by those endpoints are evidence-only plumbing and must
  not be counted as live role, model, or lifecycle validation.
- **What counts as a pass:** the rerun reaches at least the lifecycle stage
  of the prior baseline for that archetype; the failure signature the change
  claimed to fix does not reappear; no new foundation-owned failure class
  appears; the target intervention-debt count does not increase; and any
  remaining stop is an operator-visible recorded blocker rather than a silent
  loop.
- **Cross-references:** the owning ticket links the report anchor in
  `evidence_links`, and grade-affecting runs are followed by a quality-score
  export per AD-278.

### AD-294: Agent Smoke Is A Complementary Fast Lane

`mars-harness validation agent-smoke` is a source-only validation lane for
role-local smoke coverage. It generates ephemeral targets through foundation
tools, executes selected roles through the server job path, cycles cases across
API, web, game, CLI, library, docs-site, and maintenance project shapes, and
records per-role pass/fail results with terminal dispositions and failure
classes. Local-model runs default to one shared local inference server tier,
and selected cases can run in parallel because each case owns its target repo,
DB, logs, and trace state.

Agent smoke can support a source-change claim by showing that affected roles
still behave at their lifecycle checkpoints. It does not satisfy an AD-284
minimum archetype replay by itself unless the validation report explicitly says
the required full replay was unavailable and leaves the lifecycle claim
unconfirmed.

## Consequences

- Lifecycle claims name their change class and replay evidence, so reviewers
  can mechanically check whether the minimum matrix subset ran.
- Under-replayed claims become visible: a tool-policy fix validated only on
  the static canary is explicitly unconfirmed for the matrix, not implicitly
  done.
- Report naming stays consistent with the existing
  `docs/validation/reports/` files, so no migration is needed.
- This doctrine is source-only; generated targets keep inheriting the generic
  product evidence loop from AD-138 without this gating table.

## Discoveries

- **2026-06-12 — Model identity added to the AD-285 contract:** the first
  T-011 baseline attempt ran on the heavy quality-profile model while the
  operator was swapping to the balanced model for memory and speed; the run
  was reclassified evidence-only because Phase 3 replays will run on the
  balanced model and pace deltas would be confounded. AD-285 now requires
  model identity per tier in every run section and ties baseline validity to
  model identity.
- **2026-06-11 — Phase 2 WS-C slices are the first consumers:** T-024/T-025
  (local quality gates) and T-026/T-027 (release flow audit, export rendering)
  were classified under this table while it was being written. The quality
  export slice (T-027) is the worked example of the "none beyond package
  tests; next baseline replay doubles as live validation" row, and its ticket
  records that flag.
