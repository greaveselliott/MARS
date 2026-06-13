---
id: T-033
title: Verify profile-required model files at start so a profile swap cannot fail jobs with model_unavailable
priority: high
complexity: small
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md
verified_by: "go test ./internal/hardware -run ProfileModel; serve.New preflight; doctor profile-required-models check"
owner: "foundation-maintainer"
last_attempt: "2026-06-12: MissingRequiredModelFiles + ProfileModelPreflightError in hardware; serve.New fails fast before dispatch; doctor check profile-required-models."
blocker: ""
blocked_by: []
trace_id: ""
next_action: ""
kind: intervention-debt
source: 2026-06-12 independent replay-monitor observation (demo-12 run 1)
created: 2026-06-12
depends_on: []
---

# T-033: Verify profile-required model files at start

Closed 2026-06-12. `serve`/`start` now preflight required GGUF files for the
effective performance profile before dispatch; `doctor` adds
`profile-required-models` check. Actionable error directs to
`mars-harness setup`.

## Acceptance criteria

- [x] Profile swap without weights produces preflight error naming missing file(s).
- [x] doctor flags mismatch via `profile-required-models`.
- [x] Regression tests in `internal/hardware/profile_models_test.go`.
