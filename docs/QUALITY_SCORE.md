# Quality Score

**Status:** Seeded audit
**Updated:** 2026-05-02
**Owner:** Mars Harness maintainers
**Lineage:** Adapted from `../mars/docs/QUALITY_SCORE.md` and the Mars parity supersession plan.

## Purpose

This file makes Mars Harness quality visible in the repo, not only in SQLite,
the dashboard, or chat. It grades the harness as a product and as an autonomous
operating system, then names the improvement targets that should drive tickets,
plans, and bounded self-improvement.

This is the first seeded scorecard. The target state is an automated
`mars-harness scores export --repo <path>` command that refreshes this file from
role scores, traces, ticket state, checks, dogfood results, guardrail blocks,
human follow-up, and telemetry triage.

## Grading Scale

| Grade | Meaning |
| --- | --- |
| A | Complete, tested, documented, and meeting the harness quality bar. |
| B | Functional with minor gaps or hardening work still open. |
| C | Partially functional; significant implementation or proof work remains. |
| D | Scaffolded or documented, but not meaningfully implemented. |
| F | Missing entirely. |

Use plus or minus only when the direction matters. A seeded manual grade must
include evidence and the next action needed to improve it.

## Overall Roll-Up

| Area | Grade | Evidence | Next Action |
| --- | --- | --- | --- |
| Harness product foundation | B- | CLI, setup, init, run, start, serve, doctor, trust, scores, release notes, generated target harness, guardrails, and queue recovery all exist with tests. | Close release-asset, model-refresh, coverage, and live-dogfood hardening gaps. |
| Autonomous operating loop | C+ | Telemetry triage, intervention-debt tickets, ticket-drain gates, scoring, trust, and recovery hooks exist, but orchestration surveys and quality regression triggers are not complete. | Add quality export, regression detection, and Orchestrator survey triggers. |
| Mars supersession readiness | C | Mars parity has a strategic plan and several imported doctrines, but role registry, dogfood matrix, deterministic remediation, and quality export automation remain open. | Materialize parity workstreams as tickets and execute them in priority order. |
| Generated target harness | B- | Init deploys AGENTS.md, roles, skills, guardrails, knowledge routes, tickets, exec-plan docs, design docs, references, versioning, and this scorecard seed. | Add generated score export and richer target-specific readiness checks. |

**Current overall grade: C+.** The foundation is real, but the system is not yet
allowed to claim Mars supersession until the self-grading loop is generated,
regression-aware, and tied to autonomous work creation.

## Feature And Functionality Scorecard

| Domain | Grade | What Works | Gap |
| --- | --- | --- | --- |
| Strict trunk workflow | B | Source and generated guidance use direct semantic commits to `main`, push after each completed step, and generated release notes. | More workflow drift checks are still planned for stale docs and target updates. |
| Ticket workflow | B- | Canonical `docs/tickets/backlog`, `in-progress`, and `done` paths; ticket creation dedupe; in-progress drain gates. | Stale in-progress scanner and dogfood ticket caps are still open. |
| Agent runtime | B | Terminal result truth, role manifests, tool registry, tracing, and budget handling are implemented. | More loop and live model failure cases need dogfood coverage. |
| Tool policy and safety | B- | Session-aware policy, guardrails, secret scan, destructive-operation checks, and blast-radius gates exist. | Hard guardrail coverage and target-specific policy tests should expand. |
| Trust and scoring | C+ | `scores`, `trust`, and trunk-native outcome recording exist. | Quality export, regression detector, and score-to-work triggers remain open. |
| Self-reflective telemetry | C+ | Recurring failures and low scores become typed improvement proposals and intervention-debt tickets. | Orchestrator surveys and richer dogfood/ticket-state signals are pending. |
| Local inference and model routing | C+ | Pinned defaults, hardware profile logic, manifest-tier routing, and clearer missing-model errors exist. | Benchmark-backed model refresh and Ollama swap workflow are still early. |
| Setup and doctor | B- | Local-first setup, `doctor --json`, model checks, guardrail/workflow checks, and version drift warnings exist. | Zero-config throughput tuning still needs live measurement and remediation. |
| Update and release | B- | Unified `update` verb, target harness metadata, semantic patch notes, and GitHub release publication are documented and used. | Checksum-verified binary release assets are not published yet. |
| Generated target harness | B- | Target repos receive compact AGENTS.md, glossary routing, release guidance, skills, references, and quality score seed. | Generated docs must become evidence-backed rather than static seeds. |
| Dashboard and observability | B- | Dashboard, event stream, controls, traces, and health state exist. | Dashboard must link to the same quality export data without becoming the source of truth. |
| Dogfood and CI | C | Deterministic tests cover many packages and the harness has dogfood roles/design. | Coverage is below 70% in several packages, live dogfood matrix is incomplete, and `golangci-lint` availability is not guaranteed locally. |
| Distribution | C | Source install and `update tool` support the current developer workflow. | Signed/checksum release assets and installer upgrade flow are still tracked separately. |

## Top Improvement Targets

1. Implement `mars-harness scores export --repo <path>` and make it refresh this file deterministically.
2. Add quality regression detection that creates or updates intervention-debt tickets.
3. Materialize the Mars parity workstreams as normal backlog tickets so agents do not execute a giant plan directly.
4. Publish checksum-verified release assets so `mars-harness update tool` can upgrade from a released binary.
5. Add benchmark-backed model refresh and explicit Ollama model swap support without weakening zero-config defaults.
6. Close coverage gaps or document explicit exceptions for platform-specific hardware and UI paths.

## Update Rules

- Update this file whenever a material feature, architecture decision, quality gate, or parity claim changes the grade.
- Do not count passive documentation as implementation unless the behavior is wired and tested.
- Do not mark Mars supersession above C until the quality score is generated from live harness evidence.
- Once `scores export` exists, manual edits should be limited to explanatory notes that the generator preserves.
