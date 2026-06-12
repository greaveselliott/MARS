---
id: T-033
title: Verify profile-required model files at start so a profile swap cannot fail jobs with model_unavailable
priority: high
complexity: small
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md"]
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Decide the check surface (start/serve preflight reusing the AD-032 setup verification, plus a doctor row) and add a regression test for the profile-swap-after-setup path."
kind: intervention-debt
source: 2026-06-12 independent replay-monitor observation (demo-12 run 1)
created: 2026-06-12
depends_on: []
---

# T-033: Verify profile-required model files at start so a profile swap cannot fail jobs with model_unavailable

## Context

During the 2026-06-12 demo-12 package-managed-frontend session, the independent read-only replay monitor observed the first CEO job fail in ~2s with `model_unavailable`: the operator had swapped `performance_profile` to `balanced`, which resolves the reasoning/coding tiers to `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf`, but only the quality-profile `Q8_0` weights were installed at that point. The run proceeded only after the operator downloaded the Q4_K_M artifact. Classification: foundation-owned (setup/profile mismatch).

AD-032 already makes `mars-harness setup` verify the model files required by the active profile before accepting the download marker as complete, but a profile change made after setup leaves the marker stale: `start`/`serve` dispatch jobs against a profile whose model files were never verified, and the failure surfaces as a per-job runtime failure instead of an actionable preflight error.

## Requirements

- `start`/`serve` (and `run`) preflight the model files required by the effective performance profile before dispatching jobs, reusing the AD-032 verification.
- On mismatch, fail fast with an actionable error naming the missing artifact and the remediation command (`mars-harness setup`), per the errors-must-be-actionable constraint.
- `doctor` reports the same mismatch as a failed check.

## Affected Files

- internal/inference or internal/models (profile-to-artifact resolution and verification)
- cmd/mars-harness (start/serve/doctor preflight wiring)
- docs/design-docs/local-inference.md (AD update or Discoveries entry)

## Acceptance Criteria

- A profile swap after setup without the required weights produces a preflight error naming the missing file, not a ~2s model_unavailable job failure.
- doctor flags the mismatch.
- Regression test covers the profile-swap-after-setup path.
- Model/provider change class: one archetype replay end-to-end on the affected model path per AD-284, or the recorded blocker.
