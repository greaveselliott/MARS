---
id: MH-062
title: Recover CEO bootstrap planning handoff after policy blocks
priority: high
complexity: small
work_type: intervention-debt
bdd_scenarios: ["F-006-S005"]
end_to_end_evidence: docs/validation/reports/2026-08-26-ceo-bootstrap-recovery.md
evidence_links: ["make check", "go test -count=1 ./internal/tools ./internal/personas ./internal/scanner ./internal/orchestration", "go test ./internal/docsconsistency/...", "mars docsync audit --repo ."]
verified_by: "foundation-maintainer"
owner: "foundation-maintainer"
last_attempt: "2026-08-26"
blocker: "none"
blocked_by: []
trace_id: "tr-1787709582203156000"
next_action: "Complete: validated CEO completed exec_plan disposition routed to and claimed by COO."
kind: intervention-debt
dedupe_key: "foundation:ceo-bootstrap-planning-policy-recovery"
source: 2026-08-26 live noughts-and-crosses factory replay
created: 2026-08-26
depends_on: []
---

# MH-062: Recover CEO bootstrap planning handoff after policy blocks

## Context

A clean full-factory noughts-and-crosses replay recorded a foundation-owned CEO
bootstrap wedge: the CEO wrote its allowed goal, then repeatedly attempted
COO-owned feature/exec-plan writes. Tool policy correctly blocked the writes,
but the run never recorded the required CEO → COO disposition.

## Requirements

- Preserve the CEO restriction against exec-plan and feature-contract writes.
- After a forbidden CEO planning write, give one explicit, terminal recovery
  path: commit allowed strategy work if dirty, then record completed
  `exec_plan` for COO.
- Ensure later non-terminal tools are blocked until that recovery disposition.
- Cover policy guidance, generated doctrine, dispatch routing, and executor
  recovery.

## Affected Files

- internal/tools policy and executor feedback
- generated CEO doctrine
- queue/orchestration BDD contract and focused tests

## Acceptance Criteria

- [x] CEO remains unable to write COO-owned planning paths.
- [x] A blocked CEO planning write gives actionable COO terminal-handoff guidance.
- [x] A fresh full factory run reaches COO without repeated CEO planning attempts.
- [x] No CEO authority is expanded.

## Completion Evidence

The fresh local-model replay completed CEO job `463ff263-f3ae-4286-ae75-700e96c9fee8`
with `next_need=exec_plan`, `suggested_role=coo`, and trace
`tr-1787709582203156000`; it then dispatched and claimed COO job
`d0380da8-4f28-4e33-bbc4-aba47ff112a5`. The complete matrix report is
`docs/validation/reports/2026-08-26-ceo-bootstrap-recovery.md`.
