---
id: T-063
title: Reconstruct private baseline and retire embedded publication audit lab
priority: high
complexity: high
work_type: enabler
bdd_scenarios: ["F-017-S001", "F-017-S003"]
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "engineer"
last_attempt: "2026-07-16: isolated reconstruction candidate and focused source gates passed; independent review pending"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Freeze the exact candidate hash, obtain concurrent QA and Security review, then run the remaining pre-mutation gates"
dedupe_key: "oss:private-rewrite-v06849"
metadata:
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  supports: "F-017-S001,F-017-S003"
source: Operator-approved private rewrite plan
created: 2026-07-16
depends_on: []
---

# T-063: Reconstruct private baseline and retire embedded publication audit lab

## Context
The private audit implementation expanded beyond the publication outcome. Reconstruct main from v0.68.49, retain only fail-closed release mirror convergence, and remove audit runtime and false evidence.

## Requirements
Use an exact leased private-history rewrite, preserve unrelated user work, keep visibility private, and clean only verified MARS-owned artifacts.

## Acceptance criteria
The replacement tree passes the explicit allowlist oracle and full gates; v0.93.0 verifies exact 9/9; 69 retired refs/releases and eligible GitHub artifacts are reconciled; protected work is preserved; Primary Status remains primary_blocked.
