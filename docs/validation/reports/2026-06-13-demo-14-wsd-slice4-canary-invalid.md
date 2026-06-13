# demo-14 WS-D slice 4–5 canary — invalid (checkpoint seed mismatch)

**Date:** 2026-06-13  
**Target:** `/path/to/local-redacted`  
**Source ref:** v0.54.0 foundation-restart branch  
**Purpose:** Intended AD-284 tool-policy replay for WS-D slices 4–5  

## Outcome

**INVALID — do not use as pass/fail evidence.**

The run wedged in a CTO ticket-gate loop (26+ serial `cto-weekly` completions,
zero `engineer`/`qa`) because git state had all ordinary product tickets in
`docs/tickets/done/` while a fresh DB restarted CEO→COO→CTO bootstrap. CTO kept
handing off `implementation` → Engineer blocked by AD-116 → CTO `ticket_breakdown`
repeat.

Orchestrator stopped by operator after diagnosis (~02:23 local). Orphan running
job reconciled to `failed` with operator-stop reason.

## Fix landed (same session)

`enforceEngineerTicketPrerequisite` now breaks the loop: CTO completion without
an open ticket routes to **QA** when done product tickets exist, or **COO** when
greenfield ticket shaping failed. `handleDispatchComplete` uses the completing
job disposition as the ticket-gate source when the dispatch trigger lacks one.

## Validation status for WS-D 4–5

Slices 4–5 remain validated by:

- `go test ./internal/tools/...` (policy oracle)
- Prior 2026-06-12 demo-14 orchestration replay (`docs/validation/reports/2026-06-12-demo-14-convergence-routing-replay.md`)

Remaining WS-D 6–8 landed 2026-06-13 with the same unit-test oracle; schedule
one batched two-archetype replay before plan closure per AD-284.
