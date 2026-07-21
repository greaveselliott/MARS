---
id: T-064
title: Retire v0.93.0 and reset the private release line
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md#t-064-exact-retirement-transaction"]
verified_by: "QA, Security, Dogfood, Release Manager, foundation-maintainer"
owner: "engineer"
last_attempt: "2026-07-21"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Construct the replacement commit from 6d326ff, validate exact leases and protected work, then execute the ID/OID-scoped retirement transaction."
dedupe_key: "oss:retire-v093-reset-release-line"
metadata:
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  supports: "release-transition-enabler"
source: Owner-approved Retire v0.93.0 and Adopt GoReleaser plan
created: 2026-07-21
depends_on: []
---

# T-064: Retire v0.93.0 and reset the private release line

# Context

The private v0.93.0 release exists only to carry the bespoke exact-nine publisher. The owner selected complete retirement and a standard GoReleaser migration.

# Requirements

Rebuild main from retained commit 6d326ff with the release floor in VERSION reset to 0.68.49 and the untagged source fallback set to 0.69.0-dev; remove v0.93 claims; disable the Pages site observed public on 2026-07-21 before the rewrite; force-push only with the exact main lease; delete Release ID 354800199 and tag v0.93.0 by exact OID; preserve private visibility and unrelated work; re-anchor both protected stashes; replace the installed v0.93 binary before deleting its exact local artifacts.

# Interfaces And Blast Radius

Git history, VERSION/buildinfo/changelog truth, Pages/Actions/release metadata, protected stashes, and the installed development binary. T-061 source remains until T-065 replaces it.

# Acceptance Criteria

The rewritten private main is live while Pages remains disabled and unavailable logged out; v0.93 Release/tag/changelog entry are absent; VERSION retains the 0.68.49 release floor while untagged builds report 0.69.0-dev; 301 retained tags and 56 retained Releases remain; no active workflow or unexpected branch exists; both stashes reproduce exactly on rewritten main; the unrelated dashboard worktree is unchanged; Primary Status remains primary_blocked; and residual provider metadata is classified without an erasure claim.
