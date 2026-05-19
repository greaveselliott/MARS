---
id: T-009
title: Add non-mutating observer dry-run for uninitialized targets
priority: medium
complexity: small
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: "go run ./cmd/mars-harness run engineer --repo /path/to/local-redacted --dry-run --trace --no-init"
evidence_links: ["docs/validation/reports/2026-05-19-mars-observer-validation.md"]
verified_by: "go test ./cmd/mars-harness -run 'TestRunCommand(NoInit|AutoInit|RejectsRepoLocalLogFile)|TestMarsHarnessCLI'; go run ./cmd/mars-harness run engineer --repo /path/to/local-redacted --dry-run --trace --no-init; git -C /path/to/local-redacted status --short --branch"
owner: "Codex"
last_attempt: "2026-05-19"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Done. Continue Mars observer graduation only after source-side maintainer acceptance."
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
- [x] A command can assemble or explain role context for an uninitialized target without writing `.harness/` or docs into the target.
- [x] Tests prove the observer-safe path leaves the target worktree clean.
- [x] Mars observer profile no longer needs a temporary clone for the dry-run context check.
