# Self-Improvement Loop

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

Closed-loop evolution: detect when humans or the platform had to compensate, classify root cause, propose or commit safe bounded changes, and measure before/after impact.

## Context

Self-improvement is a core tenet but must not destabilize repos or the harness itself. Signals include interventions, failed jobs, and score regressions. The **Reviewer** meta-role consumes traces; evolution must respect **safety rails** and acknowledge limits of models reviewing models.

Evolution outputs are **suggestions** until an autonomous trust policy allows direct trunk evolution. Human-triggered contributor runs remain the default for high-blast-radius repos.

## Key Design Decisions

_(AD IDs to be assigned when the evolution pipeline is locked.)_

### Design anchors

- **Intervention detection:** classify events as **clear interventions** (unambiguous human override), **ambiguous** (could be normal workflow), or **non-interventions** to reduce false-positive evolution; store classification rationale in the trace.
- **Reviewer meta-role:** dedicated analysis pass over traces and diffs; separate prompts/policies from worker roles where practical; same inference stack as workers—see circular trust below.
- **Root cause classification:** buckets aligned with tenets (prompt, skill, guardrail, trigger, policy, context, model limitation)—each maps to a concrete evolution target file or setting.
- **Telemetry triage:** recurring failure patterns and low scores become typed improvement proposals before any prompt, guardrail, context, tool, manifest, process, or inference change is attempted.
- **Evolution commit creation:** concrete file edits (e.g. `.harness/roles/`, `.harness/skills/`, guardrails, manifest) with trace-linked diffs; include commit text linking to originating job and score snapshot.
- **Before/after tracking:** link evolution commits to subsequent score distributions and intervention rates; automatic rollback proposal if metrics violate guard thresholds.
- **Safety rails (non-exhaustive):** cannot modify **own meta-prompts** arbitrarily; **rate limits** (e.g. max one evolution commit per role and scope per day); **auto-disable** evolution if scores worsen beyond a threshold after a change lands.
- **Circular trust problem:** Reviewer uses the **same model stack** as workers—mitigate with deterministic checks, hard guardrails, human-triggered contributor runs for high-risk paths, and logging of reviewer conclusions for audit.

### Non-goals (v1)

Unbounded prompt rewriting without trace visibility, and cross-repo learning without explicit consent and scoping.

### Telemetry

Every evolution candidate should record **inputs hash** (manifest + role versions) so duplicate proposals can be deduplicated and A/B comparisons stay reproducible.

Store the **parent job id** in each evolution commit message and trace record for traceability across follow-up commits and reverts.

## Discoveries

- **2026-05-02 — Self-reflective telemetry triage:** Recurring telemetry patterns and low score snapshots now map to explicit improvement targets in `internal/telemetry`. The serve loop records bounded evolution reviews from those proposals instead of generic "investigate the prompt" notes.
- **2026-05-02 — Skill evolution target:** Repeated workflow confusion, max-turn loops, and human recovery procedures should create or update compact scoped skills instead of bloating role prompts.
