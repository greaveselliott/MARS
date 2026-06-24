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
| AI-client role-subagent work model | This document AD-304 |
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

### Matrix Run Reports

Whenever a foundation maintainer runs or attempts a validation matrix, create
or update a **matrix run report** under `docs/validation/reports/` in the same
work. This applies to full lifecycle matrix replays and scoped lanes such as
`mars-harness validation agent-smoke`.

A matrix run report records the selected matrix or suite, all selected cases or
archetypes, exact command, source ref or installed binary, model identity,
target/run paths, DB/log/trace paths, per-case status, failure class, cleanup
status, and any exact blocker or rerun command. A setup failure is still a
matrix run result: write the report, classify the failure, and keep the
validation claim unconfirmed until a passing rerun exists.

### AD-294: Compartmentalised Agent Smoke Complements Full Sweeps

Mars Harness also provides a fast role-local validation lane:
`mars-harness validation agent-smoke`. This lane generates fresh ephemeral
target repositories for selected roles and project types, seeds each target
through foundation scaffold/tool surfaces, executes the selected role through
the server job path, records generation provenance and terminal disposition,
and discards successful runs by default. Parallel smoke runs use independent
target repos, DBs, logs, and traces.

Agent smoke is appropriate for quickly checking whether a role still behaves
correctly at a realistic lifecycle checkpoint: CEO on an empty brief, COO after
strategy context, CTO after BDD planning, Engineer with an in-progress ticket,
reviewers after implementation evidence, maintainers after validation reports,
and Orchestrator with source disposition context.

Agent smoke suppresses follow-on dispatch after the target role. The report
records the would-be next role so route quality remains inspectable without
letting one compartmentalised case become a full lifecycle sweep.

Agent smoke does **not** replace the clean-project lifecycle replay required
for broad runtime, orchestration, or release claims. It catches role-local
regressions and project-shape overfitting earlier; full `mars-harness start`
sweeps still prove cross-role handoff quality and lifecycle progress.

### AD-298: Confidence-Gated Planning For Foundation Observations

Foundation planning created from live validation, telemetry, operator feedback,
subagent notes, or source investigations must be confidence-gated before it can
drive implementation. A plan is not decision-complete when high-impact
assumptions are hidden in prose, when discoverable repo facts have not been
checked, or when validation evidence is not mapped to the claim it supports.

This rule is source-only first. It applies to `foundation-maintainer` plans for
the Mars Harness software factory and is not mirrored into generated target
harnesses until the pattern proves reusable beyond foundation work.

Every non-trivial foundation plan created from observations uses this template:

1. **Primary Outcome Contract** — `Primary Outcome`, `Primary Pass Gate`,
   `Primary Status`, `Current Primary Blocker`, `Next Primary Action`, and
   `Supporting Evidence`, so the operator's core goal is not replaced by
   ancillary progress.
2. **Summary** — goal, observation source, and intended outcome, stated in a
   way that preserves the primary outcome status.
3. **Evidence And Classification** — source paths, logs, reports, traces, DB
   evidence, and each finding classified as `foundation-owned`,
   `deployed-owned`, `mirrored doctrine`, `evidence-only`, or `mixed/unclear`.
4. **Key Changes** — concrete behavior changes grouped by subsystem; no vague
   "improve" bullets without naming the runtime, document, tool, or validation
   surface being changed.
5. **Assumption Confidence Matrix** — columns exactly named `Assumption`,
   `Evidence`, `Confidence`, and `Validation Required`; confidence is scored
   from `0.0` to `1.0`, and any assumption below `0.9` needs explicit follow-up
   validation before completion can be claimed.
6. **Test And Validation Plan** — unit, integration, live, or docs validation
   that names which assumptions each test validates.
7. **Assumptions And Defaults** — defaults selected where the operator has not
   specified a preference, with unvalidated assumptions kept visible.

Planning starts with repo and system exploration before asking the operator for
preferences. Discoverable facts are inspected directly, not recorded as guesses.
Live validation observations do not become doctrine until the reusable
foundation-owned rule is separated from target-specific subject matter. Fake,
stub, mock, canned, or scripted model endpoints cannot increase confidence for
live behavior claims.

Completion evidence must revisit the plan's confidence matrix and primary
outcome status, then state which assumptions were proven, disproven, or still
below threshold.

### AD-300: Primary Outcome Contract Blocks Support-Only Success Claims

Foundation plans and validation reports must lead with the operator's core
goal before naming supporting work. The required contract fields are:
`Primary Outcome`, `Primary Pass Gate`, `Primary Status`,
`Current Primary Blocker`, `Next Primary Action`, and `Supporting Evidence`.
This rule is source-only foundation doctrine until a separate generated-target
mirroring decision is recorded.

Allowed `Primary Status` values are `primary_passed`, `primary_failed`,
`primary_blocked`, and `supporting_only`.

When the primary pass gate is not met, the artifact must use
`primary_failed` or `primary_blocked` and lead with that status. Passing
supporting work is recorded as `Supporting Evidence`; it cannot be described as
completion of the primary outcome. `supporting_only` is reserved for bounded
checks such as startup smoke, role-local plumbing, or release asset verification
that deliberately support a larger lifecycle claim without satisfying it.

If the primary status is not `primary_passed`, the next plan or completion
report targets `Current Primary Blocker` through `Next Primary Action` unless
the operator explicitly changes the goal. This keeps useful infrastructure work
visible without letting it displace the outcome the operator asked to prove.

### AD-304: External AI Clients Use Role-Assuming Subagents For Foundation Work

When Codex, Cursor, Claude, Gemini, Copilot, or another capable AI coding
client changes the Mars Harness source repo, the primary client instance acts
as `foundation-maintainer` and Orchestrator/integrator. For non-trivial source
work, the primary client uses role-assuming subagents, separate worktree
threads, or explicit role-labelled work packets that assume existing Mars
personas. Vendor-specific client files remain thin adapters; this document and
the `foundation-maintainer` role packet are the canonical doctrine.

A foundation work item is **non-trivial** when it changes durable behavior,
changes generated target doctrine, creates or updates an exec plan, modifies
release/update/validation/orchestration behavior, touches secrets or external
integrations, or needs more than a single mechanical docs correction. Tiny
typos, one-off command answers, and explicitly throwaway experiments may stay
single-agent.

The standard role packet is:

| Role | Foundation responsibility |
| --- | --- |
| COO | Maintain the active exec-plan slice, BDD scenario schedule, Primary Outcome Contract, and plan status. |
| CTO-weekly | Decompose the technical approach, create or refine the first implementation ticket, and name assumptions and blast radius. |
| Engineer | Implement one bounded ticket with tests, docs, and `MarsDocSync` evidence. |
| QA | Review acceptance criteria, rerun relevant tests, inspect docsync evidence, and challenge unsupported completion claims. |
| Security | Review secrets, auth, config containment, trust boundaries, external-service scope, and irreversible-action blast radius. |
| Dogfood | Own installed-binary clean-project validation, matrix reports, replay blockers, and failure ownership classification. |
| Release Manager | Generate release notes, run backfill checks, tag, publish and verify assets, or record release blockers after the primary pass gate is met or explicitly accepted as blocked. |

CEO or Head of Strategy packets are used only when the operator's goal,
product direction, or program shape is unclear. Extra specialist packets are
allowed when they map to a current Mars role, skill, or validation lane, but
they do not create independent doctrine.

The primary `foundation-maintainer` keeps final accountability:

1. Read `AGENTS.md`, the foundation role packet, this operating model, the
   active plan, relevant feature contracts, and docs named by changed-file
   `MarsDocSync` metadata.
2. Assign each subagent a bounded scope, expected artifact, file ownership, and
   validation question. Parallel subagents may run only when write scopes are
   disjoint.
3. Require each subagent output to classify findings as `foundation-owned`,
   `deployed-owned`, `mirrored doctrine`, `evidence-only`, or `mixed/unclear`.
4. Integrate outputs into repo-owned artifacts: exec plans, tickets, feature
   contracts, design docs, validation reports, release notes, or blockers.
   Chat summaries and tool transcripts are not durable system records.
5. Resolve conflicts, run the relevant gates, preserve dirty work it did not
   create, and make the final claim with the Primary Outcome status.

Subagents inherit all foundation constraints. They must not bypass trust
settings, guardrails, release rules, trunk discipline, credential hygiene,
tool allowlists, or the source/deployed ownership boundary. A subagent can
recommend a commit, release, or blocker classification, but the primary
`foundation-maintainer` owns the final decision and must not cut a release for
a failed ticket unless the blocker is explicitly recorded and accepted.

If a client does not support native subagents, the same model is expressed as
role-labelled sections in the main thread or as separate role-specific notes in
the owning artifact. The requirement is collective role coverage and durable
evidence, not a vendor-specific concurrency feature.

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

Compartmentalised agent smoke expands those archetypes with generated API, web,
game, CLI, library, docs-site, and maintenance targets across individual role
checkpoints. The source matrix lives in
[docs/validation/agent-smoke/matrix.yaml](../validation/agent-smoke/matrix.yaml).
Each ephemeral target also receives a generated
`docs/validation/agent-smoke/current-case.md` contract so live roles can read
the bounded case expectation from inside the target repo without depending on
foundation-only files.

## Anti-Patterns

| Anti-pattern | Why it fails |
| --- | --- |
| `go test` only for orchestration/tool-policy fixes | Proves point rules, not dispatch and role pacing |
| Checkpoint git + fresh DB for delivery validation | Ticket tree and bootstrap intent diverge (demo-14 CTO loop) |
| Monitoring a known-wedge canary until timeout | Wastes GPU; record invalid run and fix seed or source |
| Claiming validation from a run that never reached the affected role | Report must name product progress or explicit blocked stage |
| Treating agent-smoke as a full lifecycle sweep | Role-local fixtures can pass while cross-role handoffs still regress |
| Using fake, stub, mock, canned, or scripted model endpoints as live validation evidence | Proves runner plumbing at most and creates false positives for role behavior |
| Expecting target agents to inspect the foundation agent-smoke matrix | Generated targets only contain target-local contracts; agents must read `docs/validation/agent-smoke/current-case.md` |
| Treating assumptions as prose | Plans hide uncertainty, so implementers cannot tell which claims need validation |
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
