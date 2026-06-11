# Source Quality Gates

**Status:** Accepted
**Date:** 2026-06-11
**Author:** foundation-maintainer

## Context

Delivery gates for this repo run locally (`make check`) since the GitHub
Actions workflows were retired in favor of local asset publication (AD-059,
AD-078). Until T-024, the gate ran tests with coverage instrumentation but
enforced nothing: any package's coverage could silently regress, which is
unacceptable ahead of the WS-D/WS-E behavior-preserving refactor tracks where
the test suite is the primary preservation oracle.

## Key Design Decisions

### AD-280: Per-Package Coverage Floors Are A Ratchet Enforced By The Local Gate

`scripts/check-coverage.sh` compares every package's `go test -cover` result
against `scripts/coverage-floors.txt` and fails on:

- any package below its floor,
- any measured package missing a floor entry (new packages must opt in),
- any stale floor entry naming a removed package,
- any floored package that suddenly reports `[no test files]` or
  `[no statements]`.

Floors were seeded 2026-06-11 from verified actuals: floor = actual rounded
down to the nearest whole point; packages at or above 70% get floor 70 (the
repo's documented minimum target); actuals landing exactly on a whole point
were seeded one point lower to absorb run-to-run variance. Special floors
`notest` (currently `internal/buildinfo`) and `nostmt`
(`internal/docsconsistency`) make untestable packages explicit instead of
silently skipped.

**Ratchet-raise procedure:** floors only move up. When a commit durably
improves a package's coverage, raise that package's floor to the new actual
(rounded down) in the same commit. Never lower a floor; a regression is fixed
by restoring tests, not by editing the floors file. Packages below 70 are
improvement targets — each floor raise should move them toward the 70 minimum.

`make check` feeds its captured per-package coverage output into the gate via
`--input`, so the gate adds no second test run. `make coverage-check` runs the
gate standalone. `internal/docsconsistency` tests exercise the script's
pass/regression/missing-floor/stale-floor/deleted-tests paths against fixture
output and keep the floors file synchronized with `go list ./...`.

## Discoveries

- **2026-06-11 — Seeding floors from one run is boundary-fragile:** two
  coverage runs on the same tree differed by up to 3 points for
  `cmd/mars-harness` (test additions between runs) and packages landing on
  exact whole points (65.0, 62.0) would fail on any negative jitter; the seed
  rule therefore backs off one point at exact boundaries and the ratchet
  procedure handles the rest.
