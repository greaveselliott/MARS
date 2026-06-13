# Foundation Operating Model

**Status:** Accepted
**Date:** 2026-06-13
**Author:** foundation-maintainer
**Scope:** Source-only (`mars-harness` software factory). Not mirrored into generated target harnesses.

## Context

Mars Harness is a software factory: agents, queue, orchestration, tool policy,
inference, and generated target defaults. Unit tests and `make check` catch
regressions in isolated packages, but they cannot prove that the autonomous
lifecycle still advances from brief to ticket to implementation to review on a
real target shape.

Prior doctrine for this lived across `AGENTS.md` Working Discipline,
[delivery-operating-model.md](delivery-operating-model.md) AD-138, and
[validation-matrix-gating.md](validation-matrix-gating.md) AD-284. The 2026-06-13
demo-14 invalid canary (checkpoint git with all product tickets done + fresh
queue DB → unbounded CTO loop) showed agents still treating `go test` passes or
mis-seeded replays as lifecycle validation. This document is the canonical
**foundation** operating model: how source changes are validated before they
count as done.

## Relationship To Other Doctrine

| Topic | Canonical home |
| --- | --- |
| Six operating domains, role modes, dispatch handoffs | [harness-operating-model.md](harness-operating-model.md) |
| BDD-led target delivery loop (generated + deployed) | [delivery-operating-model.md](delivery-operating-model.md) |
| Live demo improvement loop steps (AD-138) | [delivery-operating-model.md](delivery-operating-model.md) § AD-138 |
| Source-change class → minimum archetype replays | [validation-matrix-gating.md](validation-matrix-gating.md) AD-284 |
| Validation report recording contract | [validation-matrix-gating.md](validation-matrix-gating.md) AD-285 |
| Foundation maintainer role contract | [docs/roles/personas/foundation-maintainer.md](../roles/personas/foundation-maintainer.md) |
| Validation artifacts directory | [docs/validation/README.md](../validation/README.md) |

## Key Design Decisions

### AD-291: Foundation Harness Changes Require Clean-Project Harness Validation

Every change to **foundation-owned runtime behavior** must be validated by
running the **installed** `mars-harness` binary against a **clean validation
project** before the change is treated as fixed or complete.

**Foundation-owned runtime behavior** includes any change that can alter what
happens when `mars-harness start`, `serve`, or `run <role>` executes against a
target: orchestration and ticket gates (`internal/orchestration`,
`internal/serve`, `internal/orgstate`), queue and scheduler behavior, agent loop
and parser, tool policy and guards (`internal/tools`), inference routing,
scanner/init generated defaults that change role or bootstrap behavior, scoring,
trust, guardrails, dashboard control plane, release/update paths, and
intervention-debt routing.

**Clean validation project** means all of:

1. **Clean git state for the validation intent** — either a fresh
   `mars-harness init` target for greenfield lifecycle work, or a documented
   matrix profile whose ticket tree and scenario schedule match the replay
   purpose. Do **not** pair a finished checkpoint repo (all product tickets in
   `done/`) with a wiped per-repo DB unless the test explicitly targets
   post-lifecycle review/resume behavior.
2. **Isolated per-repo database** — default `~/.mars-harness/db/{repo-name}/mars.db`
   or an explicit `--db` path reserved for the run. Wipe or use a fresh path
   when bootstrap intent must match git state.
3. **Built binary under test** — `make install` (or an explicit release binary)
   before the replay when validating source edits; do not rely on a stale
   source-root binary from a failed `go build`.
4. **Recorded evidence** — a report under `docs/validation/reports/` per AD-285,
   or an append to an existing dated report, with exact command, target path,
   source ref/binary, model identity, DB/log paths, job sequence, product
   progress reached, intervention-debt count, and stop reason.

**Minimum replay scope** follows AD-284: classify the change, run the union of
required archetype replays, or record the blocker and exact replay command with
the claim left **unconfirmed**.

**When a clean-project replay is not required** (mechanical gates still apply):

| Change class | Minimum validation |
| --- | --- |
| Docs/doctrine only (no runtime behavior) | `make check` docs gates |
| Quality export / telemetry rendering only | Package tests; next scheduled baseline replay |
| Pure refactor with line-multiset / policy oracle proof | `go test` for affected packages + `make check` |

Even exempt classes do **not** replace clean-project validation for the next
behavior-affecting change in the same area.

### AD-292: The Foundation Improvement Loop

Foundation stabilization uses the AD-138 loop with foundation-specific stop
rules:

1. Fetch remote trunk or record the blocker.
2. **`make check`** (and affected package tests) on the source change.
3. **Install** the candidate binary (`make install`).
4. **Run** `mars-harness start --repo <clean-target>` (or a scoped matrix
   command documented in the validation report) until the failure signature
   under test is exercised or the expected lifecycle stage is reached.
5. **Review** queue health, role mix, ticket tree deltas, and traces — stop
   immediately when the run is wedged (same role repeating with no product
   progress); do not wait on drain monitors or rolling ETAs after the failure
   mode is identified.
6. Implement one or two bounded source fixes; return to step 2.
7. **Rerun** on a clean seed; claim improvement only from rerun evidence.
8. Commit, release-note, push trunk, publish/verify release assets per
   `AGENTS.md`.

Batch related slices (for example WS-D policy migrations) into **one**
post-change replay where AD-284 allows, instead of one multi-hour run per
commit.

## Validation Matrix (Summary)

Use the full table in [validation-matrix-gating.md](validation-matrix-gating.md).
Archetypes (AD-138 portfolio):

| Archetype | Typical use |
| --- | --- |
| Static browser app | Bootstrap, planning, ticketing, static smoke |
| Package-managed frontend | Build/dev scripts, dependency hygiene |
| API or service | Non-frontend planning, tests, health endpoints |
| CLI/tooling | Command contracts, release notes, docsync |
| Existing-repo maintenance | Resume, dirty-tree safety, backward-compatible edits |

## Anti-Patterns

| Anti-pattern | Why it fails |
| --- | --- |
| `go test` only for orchestration/tool-policy fixes | Proves point rules, not dispatch and role pacing |
| Checkpoint git + fresh DB for delivery validation | Ticket tree and bootstrap intent diverge (demo-14 CTO loop) |
| Monitoring a known-wedge canary until timeout | Wastes GPU; record invalid run and fix seed or source |
| Claiming validation from a run that never reached the affected role | Report must name product progress or explicit blocked stage |
| Branch-only green replay | Trunk push is part of done per AD-138 |

## Discoveries

- **2026-06-13 — demo-14 invalid canary:** WS-D slices 4–5 were validated by
  policy unit tests and an earlier orchestration replay; the post-change canary
  used demo-14 at a convergence checkpoint with a wiped DB. See
  [2026-06-13-demo-14-wsd-slice4-canary-invalid.md](../validation/reports/2026-06-13-demo-14-wsd-slice4-canary-invalid.md).
  AD-290 (CTO ticket-gate loop break) landed from that evidence.

## Consequences

- Foundation work is **not done** at `make check` alone when runtime behavior
  changed.
- Agents and operators have one doc to cite for “how do we prove this fix?”
- Invalid or blocked replays stay **unconfirmed** with a recorded replay command
  rather than silent waiver.
- Deployed target operating model stays separate; targets inherit the generic
  product evidence loop, not foundation matrix shortcuts or demo names.
