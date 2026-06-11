---
id: T-024
title: Add per-package coverage ratchet gate to CI
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - "AD-280 in docs/design-docs/source-quality-gates.md, indexed in docs/design-docs/index.md"
  - "scripts/check-coverage.sh --input <captured go test -cover output> passes on the live tree with seeded floors (run 2026-06-11)"
  - "Simulated regression (inference 43.7% edited to 41.2% against floor 43) fails with the exact-package remediation message (run 2026-06-11)"
  - "go test ./internal/docsconsistency -run TestCheckCoverage covers pass/regression/missing-floor/stale-floor/deleted-tests paths; TestCoverageFloorsFileTracksRealPackages keeps floors synced with go list ./..."
verified_by: "command"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; ratchet gate wired into make check, floors seeded from verified actuals, regression path proven."
source: Foundation improvement plan Phase 2 WS-C (provisional T-024)
created: 2026-06-11
depends_on: []
---

# T-024: Add per-package coverage ratchet gate to CI

## Context

The 2026-06-11 foundation review found the delivery gate runs `go test -race -coverprofile` but enforces nothing on coverage: any package can silently regress. Before the WS-D/WS-E refactor tracks start, coverage regression must be mechanical to detect (the policy test suite is the behavior-preservation oracle for those tracks).

Ownership classification: foundation-owned CI/tooling enabler. No factory runtime behavior changes, so no canary replay is required (evidence class: test suite + docs gates).

**Scope adaptation recorded 2026-06-11:** the ticket was drafted against `.github/workflows/ci.yml`, but main retired all GitHub Actions workflows in `feat(release): move delivery gates local` (v0.44.0 on trunk). The gate therefore lands in the local delivery gate: `make check` feeds its captured per-package coverage output into `scripts/check-coverage.sh`, and `make coverage-check` runs the gate standalone.

## Requirements

- Per-package coverage floors seeded from verified current actuals: floor = actual rounded DOWN to the nearest whole point, ratchet-only. Packages at or above 70% get floor 70. Exact-boundary actuals (config 65.0, ui 62.0) seeded one point lower to absorb run-to-run variance.
- A gate script (`scripts/check-coverage.sh`) that compares each package against its floor and fails with an actionable message on regression below floor, missing floor entries, stale floor entries, and deleted tests.
- Delivery-gate integration: `make check` runs the gate from its captured coverage output (no second test run).
- Documented ratchet-raise procedure (AD-280): raise a floor in the same commit that durably improves coverage; never lower one.
- Packages with no test files (`internal/buildinfo`) or no statements (`internal/docsconsistency`) are listed explicitly so new packages must opt in.

## Affected Files

- scripts/check-coverage.sh (new)
- scripts/coverage-floors.txt (new; seeded floors)
- Makefile (`check` + `coverage-check` targets)
- internal/docsconsistency/check_coverage_script_test.go (new)
- docs/design-docs/source-quality-gates.md (new; AD-280)
- docs/design-docs/index.md

## BDD Evidence

- Scenario IDs: none (enabler)
- Evidence links: see frontmatter
- Verified by: command

## Acceptance Criteria

### Functional (happy path)
- [x] Gate passes on the current tree with seeded floors (verified 2026-06-11 against a fresh `go test -cover ./...` capture).
- [x] Delivery gate runs the coverage check on every `make check` (workflow adaptation: local gates replaced GitHub Actions on trunk).

### Edge cases and negative paths
- [x] A simulated regression (inference below floor) fails the gate with a clear remediation message.
- [x] Packages with `[no test files]` or `[no statements]` are handled explicitly via `notest`/`nostmt` floors; a floored package that loses its tests fails.
- [x] New packages without a floor entry cause a visible failure asking for a floor to be added; stale entries for removed packages also fail.

### Non-goals
- Raising coverage of low-coverage packages (separate work; the ratchet only prevents regression).
- Line-level or diff-based coverage analysis.

### Observability, docs, and regressions
- [x] AD-280 recorded and indexed; ratchet-raise procedure documented in source-quality-gates.md.
- [x] scripts/check-coverage.sh and the script test carry MarsDocSync metadata; docsync audit reports 0 findings.
- [x] Gate logic covered by internal/docsconsistency script tests plus the floors-file/go-list sync test.
