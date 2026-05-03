---
id: MH-047
title: Add native Orchestrator survey loop
priority: high
complexity: large
kind: intervention-debt
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["go test ./internal/queue ./internal/context ./internal/scheduler", "go test ./internal/serve -run 'TestOrchestratorSurvey|TestHandleJobComplete|TestScanRepoEnqueuesJanitor|TestSelfHealRecoveryQueue|TestHandleJobFailed|TestBuildTicketIndex'", "go test ./... (initial failure: active-plan hygiene before ticket closure)"]
verified_by: command
owner: engineer
last_attempt: 2026-05-03T19:20:00Z
blocker: none
blocked_by: []
trace_id: TBD
next_action: Closed; rerun go test ./... after active-plan ticket state is reconciled.
dedupe_key: "public-example"
source: Mars parity workstream G
created: 2026-05-03
metadata:
  role: "planner"
  repo_id: "mars-harness"
  target: "orchestrator"
  category: "unattended_failure_detection"
  severity: "high"
  confidence: "high"
---

# MH-047: Add native Orchestrator survey loop

## Context

Mars uses scheduled automation and watchdog behavior to detect unattended
failure states. Harness has native queue, scheduler, trace, scoring, guardrail,
and trust primitives, but it needs an Orchestrator loop that surveys those
signals and creates prioritized jobs before work silently stalls.

## Requirements

- Add an Orchestrator loop that surveys queue, tickets, scores, guardrails,
  traces, doctor checks, and recent run outcomes.
- Translate the "eligible task has an agent/workspace" rule into queue ownership
  for in-progress tickets, blocked tickets, and scheduled retries.
- Run self-reflective telemetry triage during surveys.
- Add event routing for failed checks, dogfood failures, stale in-progress
  tickets, quality regression, intervention debt, dependency alerts, and release
  readiness.
- Add concurrency groups and daily caps to queue scheduling.
- Add payload-mode support to jobs and role prompts.
- Add watchdog detection for stuck jobs and silent no-op runs.

## Affected Files

- `internal/serve/`
- `internal/scheduler/`
- `internal/queue/`
- `internal/trace/`
- `internal/scoring/`
- `internal/trust/`
- `internal/telemetry/`
- `docs/design-docs/trigger-orchestration.md`
- `docs/design-docs/self-reflective-telemetry.md`

## Acceptance Criteria

### Functional

- [x] Orchestrator surveys can create or update prioritized jobs for unattended
      failure states.
- [x] Queue ownership prevents duplicate work on the same eligible ticket.
- [x] Telemetry triage runs during surveys, not only after job completion.
- [x] Payload modes are represented in jobs and prompts.
- [x] Concurrency groups and daily caps constrain repeated retries.

### Edge cases and negative paths

- [x] Stuck-job detection does not interrupt healthy long-running jobs.
- [x] No-op runs are detected without relying on GitHub or PR state.
- [x] Optional GitHub events enrich routing but are not required for local
      orchestration.

### Observability, docs, and regressions

- [x] Integration tests cover survey-to-job routing for stale tickets and failed
      checks.
- [x] Trace and score records show why Orchestrator created each job.
- [x] Dashboard and doctor surfaces point to the same source signals.

## Completion Notes

Completed: 2026-05-03 — Native Orchestrator survey loop now runs from `serve`
startup/watchdog, self-heals recovery jobs, fails only long-stuck running jobs,
routes eligible/stale/blocked tickets, failed checks, dogfood failures, no-op
outcomes, telemetry patterns, and low scores, and adds queue payload mode,
concurrency group, and daily cap metadata. Verification:

- `go test ./internal/queue ./internal/context ./internal/scheduler`
- `go test ./internal/serve -run 'TestOrchestratorSurvey|TestHandleJobComplete|TestScanRepoEnqueuesJanitor|TestSelfHealRecoveryQueue|TestHandleJobFailed|TestBuildTicketIndex'`
- `go test ./...` was run before the final active-plan reconciliation and
  failed only because this ticket had moved from backlog to in-progress while
  the active plan still listed it as backlog.
