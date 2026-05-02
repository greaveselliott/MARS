# AD-074: BDD-Led Goal-Driven Walking-Skeleton Operating Model

**Status:** Accepted
**Date:** 2026-05-02
**Owner:** Mars Harness maintainers

## Context

Mars Harness had the right instincts: tickets, strict trunk, dogfood, telemetry,
quality score, role scoring, and self-improvement. The gap was delivery truth.
Agents could move tickets, add docs, or create enabler code while still leaving
the intended feature half-built. The user-visible complaint was concrete:
dogfood creates tickets, engineers start many of them, and handoff happens
without enough proof that the work is actually complete.

We need an operating model that makes feature completeness observable before
implementation starts, then lets autonomous agents take the smallest useful
path to pass evidence. Lean and walking skeleton are good delivery strategies,
but they become vague without a feature contract and falsification loop. BDD
provides the contract; walking skeleton provides the implementation strategy.

## Decision

Mars Harness uses **BDD-Led Goal-Driven Walking-Skeleton Delivery** as the
canonical operating model for the source harness and generated target
harnesses.

The closed loop is:

1. Goals define outcomes and competing priorities.
2. BDD feature contracts define the full intended capability before
   implementation.
3. Exec plans rank failing BDD scenarios and state the hypothesis for why this
   work advances active goals.
4. Tickets implement the next highest-value failing scenario or scenario group.
5. Walking skeleton delivery makes scenarios pass through the thinnest real
   end-to-end path.
6. Evidence from BDD E2E/integration, dogfood, telemetry, quality score, and
   feedback updates goals and plans.

Core rule:

> BDD defines the full feature. Walking skeleton is the implementation
> strategy. The schedule is the ordered list of failing BDD scenarios.

No feature is shipped until its in-scope BDD scenarios pass or the CEO
explicitly descopes, supersedes, or invalidates them. Enabler work may complete
without shipping a feature, but it must be labelled as enabler work and must
not be represented as shipped feature value.

## Artifact Ownership

Goals live in `docs/goals/`. Active goals may come from user chat, product
requirements, telemetry, quality scores, dogfood, GitHub issues, feedback
forms, or manual docs. Weak/noisy evidence becomes an observation until it is
actionable.

BDD feature contracts live in `docs/features/`. They use Markdown
Given/When/Then in v1. We deliberately do not add a custom Gherkin parser yet;
Go integration/E2E tests and explicit evidence commands execute behavior.

Exec plans live in `docs/exec-plans/` with exactly one active plan. Active and
backlog plans must name goals, BDD feature contracts, hypothesis, success and
falsification evidence, scenario schedule, current failing scenario, walking
skeleton slice, and learning or MVP outcome.

Tickets live in `docs/tickets/` and carry `work_type`, `bdd_scenarios`,
`end_to_end_evidence`, `evidence_links`, and `verified_by`. Feature tickets
require scenario evidence before done. Enabler tickets can complete without
feature evidence, but cannot claim shipped feature value.

## Role Responsibilities

The CEO owns goals, BDD feature contracts, scenario schedule, tradeoffs, and the
single active exec plan.

The CTO validates the plan hypothesis, architecture fit, and whether the
walking skeleton is a real end-to-end path rather than scaffold-only work.

The COO creates tickets only from the current failing scenario or scenario
group.

The Engineer implements one ticket and provides scenario evidence before done.

QA reviews behavior against BDD scenarios and ticket evidence.

Dogfood validates the real build, run, user, or agent path.

The Janitor detects stale goals, stale scenarios, false done, missing evidence,
duplicate work, and misleading in-progress state.

The Release Manager separates shipped scenarios from enablers in patch notes
and release notes.

## Failure Modes And Mitigations

| Failure Mode | Why It Happens | Mitigation |
| --- | --- | --- |
| BDD becomes decorative prose | Scenarios are written but not executed or checked | Every feature needs at least one integration/E2E test or command mapped to scenario IDs. |
| Walking skeleton becomes scaffold theater | Thin slice is mistaken for placeholder architecture | Slice must pass through a real user, CLI, agent, tool, ticket, docs, or evidence path as applicable. |
| Half-features are marked done | Ticket AC passes locally while the feature contract still fails | Feature truth lives in BDD scenario state, not ticket count. |
| Enabler work is misrepresented as shipped value | Release notes infer feature status from commit text | Release notes and quality score classify by `work_type` and scenario evidence. |
| Autonomous goals create thrash | Feedback or telemetry is noisy | Require source, confidence, dedupe key, evidence, and review trigger; weak signals go to observations. |
| CEO ranks by vibes | Goals are ambiguous or incomparable | CEO ranks by value, urgency, confidence, unblock potential, and falsification risk. |
| Competing goals blur strategy | Multiple goals pull in different directions | Goals declare `Competes With`; active plan records tradeoffs and deferred goals. |
| Source and target harness diverge | Init gets new doctrine but old targets keep stale docs | `update check` and `doctor --repo` report drift; `update harness` writes missing defaults only; stale user-owned files become migration tickets. |
| Unit tests give false confidence | Operating-model bugs appear only in full loops | BDD E2E/integration is default acceptance; unit and docs tests support deterministic helpers. |
| Lean becomes endless learning | Hypotheses never close | Exec plans require success and falsification evidence; inconclusive plans are revised, superseded, or split. |

## Mirrored Application

This operating model applies to Mars Harness source and generated target
harnesses unless a rule is explicitly marked source-only. Source changes update
`AGENTS.md`, design docs, product specs, goals, features, exec plans, ticket
docs, quality score, scanner defaults, role prompts, knowledge routes, and
docs-consistency tests in the same task.

Generated targets receive goal docs, feature-contract docs, this design
decision seed, updated exec-plan and ticket templates, updated role prompts,
updated knowledge routes, and target drift checks. Existing target adoption is
non-destructive: missing defaults are written, stale user-owned files are
reported, and conflicts become intervention-debt migration tickets instead of
silent overwrites.

## Consequences

This makes the harness more demanding. A ticket can be done only when the
evidence is visible. That is intentional: the system should optimise for
truthful completion, not autonomous motion. The tradeoff is more upfront
structure in goals and feature contracts, but less downstream ambiguity, fewer
half-shipped features, and clearer evidence for self-improvement.
