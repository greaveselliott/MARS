---
id: T-063
title: Reconstruct private baseline and retire embedded publication audit lab
priority: high
complexity: high
work_type: enabler
bdd_scenarios: ["F-017-S001", "F-017-S003"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md#private-rewrite-evidence--2026-07-16"]
verified_by: "qa, security, dogfood, release-manager, foundation-maintainer (rewrite and retirement); final release verification pending"
owner: "engineer"
last_attempt: "2026-07-16: reviewed reconstruction force-leased to private main; 69 Releases, 69 tags, 130 Actions runs, and 129 deployments retired exactly"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Commit rewrite evidence, generate explicit 0.93.0 notes, publish and verify exact 9/9 private assets, then re-anchor protected stashes and clean MARS-only local artifacts"
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

## Evidence

The reviewed semantic tree is now private `main`, Pages built from it, all 69 captured Release IDs and tag refs are absent, and the discarded-history Actions/deployment cleanup is reconciled in the active plan. GitHub-only SHA metadata that has no deletion API is classified explicitly instead of being called erased. The ticket remains in progress until the exact-nine `v0.93.0` release and protected-work/local cleanup gates pass.
