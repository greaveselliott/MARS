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

### AD-281: Hostile Model Output Parsers Are Fuzzed And The Module Is Vulnerability-Scanned

Two source surfaces parse hostile model output: the tool-call parser in
`internal/agent/parser.go` (structured fields, function tags, inline
`tool_call` tags, embedded JSON) and the T-015 list-string/argv normalizers in
`internal/tools` (`decodeStringSliceArg`, `parsePythonStyleStringList`,
`normalizeShellExecArgv`). Local models emit malformed payloads routinely, so
these parsers must never panic and must never produce structurally invalid
tool calls.

- Go fuzz targets cover both surfaces:
  `FuzzToolCallsFromAssistantMessage` (seeded from `internal/agent/testdata`
  recordings) asserts parsed calls always carry a non-empty name and valid
  JSON arguments; `FuzzDecodeStringSliceArg`,
  `FuzzParsePythonStyleStringList`, and `FuzzNormalizeShellExecArgv` assert
  the normalizers never panic, never produce invalid UTF-8 from valid input,
  and never silently drop non-empty argv.
- `make fuzz-smoke` runs each target for a bounded `FUZZTIME` (default 10s)
  and is part of `make check`. Full fuzzing stays local and manual
  (`go test <pkg> -fuzz <target> -fuzztime 5m`). Crash corpus entries are
  committed under `testdata/fuzz/` as permanent regression seeds.
- `make vuln` runs `govulncheck ./...` and is part of `make check`. A missing
  `govulncheck` binary degrades with the exact install command instead of
  failing cryptically, so offline or fresh environments still pass the rest of
  the gate.

## Discoveries

- **2026-06-11 — First fuzz run found a real parser bug in seconds:**
  `FuzzToolCallsFromAssistantMessage` immediately produced `[{}]`, which the
  array path of `parseToolCallsFromText` converted into a tool call with an
  empty name (the single-object path already rejected that). The fix rejects
  unnamed tool calls in the array path; the crashing input is committed in
  `internal/agent/testdata/fuzz/` as a regression seed.
- **2026-06-11 — govulncheck found 6 reachable stdlib vulnerabilities:** all
  were fixed upstream in go1.26.3/go1.26.4, so the bounded fix was pinning
  `toolchain go1.26.4` in `go.mod`; after the bump the module scans clean (one
  unreachable vulnerability remains in a required module).
- **2026-06-11 — Seeding floors from one run is boundary-fragile:** two
  coverage runs on the same tree differed by up to 3 points for
  `cmd/mars-harness` (test additions between runs) and packages landing on
  exact whole points (65.0, 62.0) would fail on any negative jitter; the seed
  rule therefore backs off one point at exact boundaries and the ratchet
  procedure handles the rest.
