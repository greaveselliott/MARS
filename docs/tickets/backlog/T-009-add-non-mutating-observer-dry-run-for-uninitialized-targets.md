---
id: T-009
title: Add non-mutating observer dry-run for uninitialized targets
priority: medium
complexity: small
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: []
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "TBD"
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "observer_mode"
  severity: "medium"
source: MH-049 mars observer validation 2026-05-19
created: 2026-05-19
depends_on: []
---

# T-009: Add non-mutating observer dry-run for uninitialized targets

## Context
The first Mars observer trial found that `mars-harness run engineer --repo <target> --dry-run --trace` auto-initializes `.harness/` when the target has no manifest. That makes the profile command unsafe against a real observer target such as `/path/to/local-redacted`, where the validation contract forbids writes. The trial used a temporary clone for this command and left the real Mars checkout clean.

## Requirements
- Add a non-mutating way to preview role context for an uninitialized target, or add an explicit no-init/observer-safe mode for dry-run.
- Ensure observer validation docs and CLI help make the write boundary clear.
- Preserve the existing auto-init behavior for normal `run` workflows unless an observer-safe flag is selected.

## Acceptance criteria
- [ ] A command can assemble or explain role context for an uninitialized target without writing `.harness/` or docs into the target.
- [ ] Tests prove the observer-safe path leaves the target worktree clean.
- [ ] Mars observer profile no longer needs a temporary clone for the dry-run context check.
