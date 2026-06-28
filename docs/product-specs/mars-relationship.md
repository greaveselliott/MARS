# Relationship To Mars

**Status:** Accepted
**Updated:** 2026-05-02
**Owner:** MARS maintainers
**Sources:** [Mars parity supersession plan](../exec-plans/backlog/mars-parity-supersession-plan.md), [Mars audit](../references/mars-meta-harness-relevance-audit.md), [vision](vision.md)

## Position

Mars and MARS are separate repositories. Mars remains the application and the successful prototype of the operating model. MARS is the standalone product that should supersede Mars's meta-harness layer.

Supersession does not mean copying Mars exactly. It means preserving the useful delivery system Mars evolved through real autonomous work, then rebuilding those lessons as first-class MARS primitives.

## What Mars Proved

Mars proved that autonomous delivery works best when the repository is treated as an operating system for agents:

- compact agent entrypoints
- durable tickets and plans
- role-specific prompts
- decision records
- intervention debt
- self-review and quality scoring
- deterministic repair paths before probabilistic work
- dogfood coverage that generates real backlog
- docs and rules that future agents can retrieve

Those ideas are product requirements for MARS, not historical trivia.

## What MARS Keeps

MARS carries forward the meta-harness pieces that are generally useful:

- multi-role delivery loops
- planner, engineer, reviewer, maintainer, tester, orchestrator, and evolution responsibilities
- tickets as work state
- intervention debt as product input
- context routing over context stuffing
- quality and trust signals from real outcomes
- generated target guidance from day one
- dogfood results as actionable work

## What MARS Changes

MARS translates Mars's repo-specific automation into product-native primitives:

| Mars lesson | MARS product form |
| --- | --- |
| Repository docs guide agents | `AGENTS.md`, design docs, exec plans, tickets, references, and generated target docs |
| Role prompts operate the pipeline | `.harness/manifest.yaml`, role prompts, tool allowlists, trust levels, and queue jobs |
| Human fixes reveal automation debt | Telemetry triage, intervention tickets, score penalties, and bounded evolution |
| Agent context must be routed | `.harness/knowledge/` and context glossary files |
| Dogfood failures create work | Scanner and dogfood tickets in `docs/tickets/backlog/` with dedupe |
| Quality must be visible | Scores, traces, dashboard state, and future repo-visible exports |
| Workflow is repo-owned | Source and initialized harnesses mirror the same operating doctrine |

## Supersession Criteria

MARS can be considered ready to supersede Mars's meta-harness when:

- initialized target repos receive guidance at least as useful as the Mars template
- role registry, trust, scoring, tickets, context routes, guardrails, and references are visible and checked
- in-progress work is drained or truthfully blocked instead of left to pile up
- local inference setup is zero-config for normal users and transparent for advanced users
- telemetry produces concrete improvement proposals instead of passive charts
- generated target harness files evolve with the source harness through `upgrade`
- Mars itself can be run as a target repo without relying on Mars-specific automation glue

## Current Status

MARS now has the stronger product foundation: Go binary, local inference, SQLite queue, dashboard, trust store, scoring store, guardrails, scanner, generated target harness, context glossary, and optional remote-code-host integration.

Mars still has richer proven operating habits in some areas: role registry, intervention-debt hygiene, deterministic maintenance scripts, dogfood matrix, and repo-visible quality artifacts. The [Mars parity supersession plan](../exec-plans/backlog/mars-parity-supersession-plan.md) is the P1 backlog plan for that remaining work; the current active plan decides which slice runs next.

## Product Rule

When Mars and MARS disagree, MARS should copy the underlying lesson only if it strengthens the reusable product. Mars-specific application rules stay in Mars. General autonomous-delivery discipline graduates into MARS.
