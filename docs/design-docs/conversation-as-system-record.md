# AD-086: Conversation As System Record

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** MARS maintainers

## Context

MARS already treats the repository as the system of record. The gap is
precision: significant agent conversations can still leave durable decisions,
investigations, quality findings, or completion claims only in chat. Future
agents cannot safely use that context unless it is converted into repo-owned
artifacts that strict-trunk delivery can commit directly to `main`.

The rule must not create documentation churn for trivial replies. A short
command answer, status check, or throwaway experiment can stay in chat when it
does not change product intent, architecture, quality state, ticket state, or
future execution.

## Decision

Significant conversations are inputs, not durable records. When a conversation
changes how the harness should operate, what should be built, why a trade-off
was chosen, what an investigation discovered, whether quality is acceptable, or
what work is complete, the agent must update the owning repo artifact in the
same direct commit to `main`.

Chat summaries can help humans catch up, but they cannot replace required
artifacts. The durable record is the checked-in file, generated release note,
trace, ticket, plan, quality note, design decision, product spec, or executable
test evidence.

## Artifact Routing

| Conversation signal | Required durable artifact |
| --- | --- |
| Goal, priority, scope, or scenario direction changes | `docs/goals/`, `docs/features/`, and the active exec plan as applicable. |
| Ticket creation, claim, blocker, dependency, or completion state changes | `docs/tickets/` plus the active plan ticket-state section when it names ticket locations. |
| Architecture, workflow, guardrail, tool-policy, or non-obvious trade-off decisions | `docs/design-docs/`, `docs/design-docs/index.md`, and `record_decision` when the tool is available. |
| Product-facing behavior or user-visible capability changes | Owning product spec or design doc with the reason why. |
| Investigation findings that future agents may need | Owning design doc Discoveries section, `docs/references/`, `docs/reports/`, or a focused ticket. |
| Quality findings, regressions, verification evidence, or readiness claims | Ticket evidence fields, `docs/QUALITY_SCORE.md`, test names, traces, or reproducible report paths. |
| Completed work | Ticket moved to `done/`, active plan refreshed when it names the ticket, semantic commit, generated release notes, and release evidence when configured. |

## Negative Cases

The following conversations do not require repo artifact churn by themselves:

- A simple command answer that does not change state.
- A one-off explanation of existing code with no new decision.
- A local experiment explicitly marked throwaway and not used to justify a
  future action.
- A status update that only repeats already-checked-in facts.

If the conversation later becomes evidence for a decision, plan, investigation,
or completion claim, the relevant artifact must be updated before work is
claimed complete.

## Enforcement Evidence

The active-plan hygiene checker delivered by `MH-034` is the enforcement point
for plan and ticket-state drift. It is exposed through `mars doctor
--repo .` and the docs-consistency tests, and it reports multiple active plans,
stale or misleading ticket-location claims, unresolved `TBD` placeholders,
relative status language without dates, and stale verification notes.

This decision does not duplicate that checker. It defines when conversation
content must become artifacts; the existing checker verifies that a key subset
of those artifacts stays coherent.

## Mirrored Target Guidance

Generated target harnesses receive the same generic doctrine through
`AGENTS.md` and seeded design guidance. The target wording stays project
neutral: it describes plans, tickets, design decisions, investigations, quality
evidence, completed work, and direct commits to `main` without importing
Mars-specific product constraints.

Existing initialized targets remain user-owned. Upgrade writes only missing
defaults and reports drift instead of overwriting local policy.

## Consequences

This raises the bar for completion claims. Agents must leave the repo in a
state where the next run can recover intent, evidence, and ticket state without
reading chat history. The tradeoff is a small amount of extra documentation for
meaningful conversations, bounded by the explicit negative cases above.
