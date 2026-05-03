# Tickets

Work items live as markdown files in this directory. The repo is the source of truth.

## Directory Structure

```
docs/tickets/
  backlog/       Tickets waiting to be picked up
  in-progress/   Tickets actively being worked on
  done/          Completed tickets committed directly to main
```

## Ticket Format

```markdown
---
id: MH-001
title: Short description
priority: high | medium | low
complexity: small | medium | large
kind: standard | intervention-debt
work_type: feature | enabler | research | docs | intervention-debt
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required | not_applicable
evidence_links: []
verified_by: TBD
owner: TBD
last_attempt: TBD
blocker: none
blocked_by: []
trace_id: TBD
next_action: TBD
dedupe_key: optional-machine-key
source: delivery-schedule M1.3.1
created: 2026-04-11
---

# MH-001: Short description

## Context
[Link to the delivery schedule milestone and task]

## Requirements
[Specific implementation requirements]

## Affected Files
[File paths or packages]

## BDD Evidence
- Scenario IDs: F-001-S001
- Evidence links: test command, report path, trace, dogfood result, or explicit proof
- Verified by: engineer | qa | dogfood | command | human

## Acceptance Criteria

### Functional (happy path)
- [ ] Primary behaviour works as specified

### Edge cases and negative paths
- [ ] Each known failure mode has an explicit line
- [ ] Invalid input, error paths, boundary conditions covered

### Non-goals
- What this ticket does NOT do

### Observability, docs, and regressions
- [ ] Docs updated if behaviour changed
- [ ] Tests cover the new functionality

## Notes
[Implementation notes, discoveries]
```

`kind` defaults to `standard` when omitted. `intervention-debt` is reserved for harness self-improvement work created from telemetry, non-success terminal agent results, guardrail or tool-policy blocks, repeated tool loops, manual stops, timeouts, score regressions, stale ticket state, dogfood failures, human follow-up, or reverted agent commits.

`work_type` drives completion truth:

- `feature` means the ticket implements a BDD scenario or scenario group.
- `enabler` means the ticket improves infrastructure, docs, tests, scaffolding, or process but does not itself ship a feature.
- `research` records evidence needed before feature work.
- `docs` updates documentation without claiming shipped feature behavior.
- `intervention-debt` fixes the harness process that produced bad work.

Feature tickets require non-empty `bdd_scenarios`, `end_to_end_evidence: required`,
non-empty `evidence_links`, and a real `verified_by` value before moving to
`done/`. Enabler, research, docs, and intervention-debt tickets normally use
`end_to_end_evidence: not_applicable` and must not claim a shipped feature.

Intervention-debt tickets must include role, repo, target, category, severity, confidence, evidence, and origin metadata when generated mechanically. Origin metadata should link trace IDs, score snapshots, commits, outcomes, tools, jobs, telemetry events, and source messages when available locally; missing optional GitHub metadata must not block local ticket creation. They are deduped by repo, role, target, category, and evidence window.

## Drain Metadata

The ticket drain gate uses these fields:

- `owner`: role or human currently responsible for the ticket.
- `last_attempt`: ISO date or timestamp for the latest meaningful attempt.
- `blocker`: concrete blocker note; use `none` or `TBD` when unblocked.
- `blocked_by`: dependency ticket IDs that must land before this resumes.
- `trace_id`: trace for the latest relevant run when available.
- `next_action`: concrete resume or unblock instruction.

Eligible in-progress tickets are `docs/tickets/in-progress/` files without a
meaningful `blocker` or `blocked_by`. Eligible in-progress work is always ahead
of backlog work. Blocked in-progress tickets do not cause infinite retries, but
they must point to a dependency ticket or carry a blocker note clear enough for
Janitor, Doctor, and the next Engineer run to recover state.

## Naming Convention

`MH-NNN-short-description.md` where NNN is a zero-padded sequential number.

## Lifecycle

1. Ticket created in `backlog/`
2. Implementation starts: move to `in-progress/`
3. An unfinished in-progress ticket must end as one of:
   - completed and moved to `done/`
   - returned to `backlog/` with `blocker` and `next_action`
   - left in `in-progress/` with `blocked_by` pointing at a dependency ticket
   - guardrail-blocked with `blocked_by` pointing at intervention debt
4. Implementation completes: move to `done/` in a direct semantic commit on `main`
5. Push `main` after the commit so the repo remains the system of record

## Priority Rules

Eligible in-progress tickets are always the front of the queue. If multiple tickets are already in progress, drain the lowest-numbered eligible in-progress ticket first and fix blockers proactively in the same run.

Engineer runs cannot create ordinary backlog tickets while eligible in-progress tickets remain. Dependency tickets are allowed only when deduped and linked back to the blocked ticket through metadata such as `metadata.blocks`. Dogfood ticket creation is capped per run by total count, severity, group, and repeated dedupe key.

Intervention-debt tickets are prioritised ahead of ordinary backlog work because they represent a failure in the harness process, prompts, skills, guardrails, context routing, inference setup, or tool policy. Existing matching intervention-debt tickets are updated rather than duplicated.
