# Self-Improvement Loop

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

Closed-loop evolution: detect when humans or the platform had to compensate, classify root cause, propose safe changes (often as PRs), and measure before/after impact.

## Context

Self-improvement is a core tenet but must not destabilize repos or the harness itself. Signals include interventions, failed jobs, and score regressions. The **Reviewer** meta-role consumes traces; evolution must respect **safety rails** and acknowledge limits of models reviewing models.

Evolution outputs are **suggestions** until policy allows auto-merge; human review remains the default for high-blast-radius repos.

## Key Design Decisions

_(AD IDs to be assigned when the evolution pipeline is locked.)_

### Design anchors

- **Intervention detection:** classify events as **clear interventions** (unambiguous human override), **ambiguous** (could be normal workflow), or **non-interventions** to reduce false-positive evolution; store classification rationale in the trace.
- **Reviewer meta-role:** dedicated analysis pass over traces and diffs; separate prompts/policies from worker roles where practical; same inference stack as workers—see circular trust below.
- **Root cause classification:** buckets aligned with tenets (prompt, guardrail, trigger, policy, context, model limitation)—each maps to a concrete evolution target file or setting.
- **Evolution PR creation:** concrete file edits (e.g. `.harness/roles/`, guardrails, manifest) with human-reviewable diffs; include summary comment linking to originating job and score snapshot.
- **Before/after tracking:** link evolution commits to subsequent score distributions and intervention rates; automatic rollback proposal if metrics violate guard thresholds.
- **Safety rails (non-exhaustive):** cannot modify **own meta-prompts** arbitrarily; **rate limits** (e.g. max one evolution PR per day per scope); **auto-disable** evolution if scores worsen beyond a threshold after a change lands.
- **Circular trust problem:** Reviewer uses the **same model stack** as workers—mitigate with deterministic checks, hard guardrails, mandatory human merge for high-risk paths, and logging of reviewer conclusions for audit.

### Non-goals (v1)

Fully autonomous prompt rewriting without human visibility, and cross-repo learning without explicit consent and scoping.

### Telemetry

Every evolution candidate should record **inputs hash** (manifest + role versions) so duplicate proposals can be deduplicated and A/B comparisons stay reproducible.

Store the **parent job id** on each evolution PR comment for traceability across merges and reverts.

## Discoveries

_(None yet.)_
