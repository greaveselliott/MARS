---
id: T-063
title: Reconstruct private baseline and retire embedded publication audit lab
priority: high
complexity: large
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["git show --stat 6d326ffa82e236570509b3783711b87911e5500d", "docs/tickets/done/T-063-reconstruct-private-baseline-and-retire-embedded-publication.md#evidence"]
verified_by: "qa, security, dogfood, release-manager, foundation-maintainer (reconstruction and retirement only)"
owner: "none"
last_attempt: "2026-07-21: original exact-nine release outcome superseded by owner-approved v0.93 retirement and GoReleaser migration"
blocker: "superseded_by_T-064"
blocked_by: []
trace_id: "TBD"
next_action: "No resume; preserve reconstruction evidence and continue through T-064."
dedupe_key: "oss:private-rewrite-v06849"
metadata:
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  supports: "private-reconstruction-evidence"
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
The replacement tree passes the explicit allowlist oracle and full gates; 69 audit-era refs/releases and eligible GitHub artifacts are reconciled; protected work is preserved; Primary Status remains primary_blocked. The proposed v0.93 exact-nine release was explicitly superseded and is not completion evidence.

## Evidence

The reviewed semantic tree became private `main`, Pages built from it, all 69 captured Release IDs and tag refs were removed, and the discarded-history Actions/deployment cleanup is reconciled in the active plan. GitHub-only SHA metadata that has no deletion API is classified explicitly instead of being called erased. On 2026-07-21 the owner superseded the remaining exact-nine release outcome with T-064 and the GoReleaser transition; this blocked ticket is retained only as historical reconstruction evidence and must not be resumed or counted as release-lifecycle completion.
