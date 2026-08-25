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
- `make vuln` runs `govulncheck ./...` and is part of `make check`. The scanner
  path is injectable through `GOVULNCHECK`, while the default resolves to the
  configured Go install bin (`GOBIN`, or `GOPATH/bin` when unset), matching
  where `go install` writes the tool. A missing scanner fails closed with the
  pinned remediation
  `go install golang.org/x/vuln/cmd/govulncheck@v1.7.0`. Package-load,
  vulnerability-database, and reachable-vulnerability failures propagate the
  scanner's non-zero result instead of being treated as optional evidence.

### AD-314: Source Builds Require Go 1.25.13 While Packaged Operation Does Not Require An External Go Toolchain

The canonical MARS source module declares `go 1.25.13` and retains
`toolchain go1.27.0` for release builds. Read-only CI exercises
the exact 1.25.13 and 1.27.0 toolchains with `GOTOOLCHAIN=local`, while an
exact 1.25.12 lane must fail specifically at the module floor without
auto-downloading a newer compiler.

Only `mars doctor --repo <mars-source>` enforces this source prerequisite.
Packaged/default operation and ordinary target repositories do not invoke or
require an externally installed Go toolchain. Generated target repositories
do not inherit the MARS source floor and choose their own toolchain.

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
- **2026-07-12 — publication readiness requires a fail-closed scanner:** the
  release toolchain moved to Go 1.26.5 for GO-2026-5856. `x/sys` v0.44.0
  requires Go 1.25 and would break MARS's Go 1.22.4 minimum, so the dependency
  moved to v0.30.0, the newest release whose module still supports Go 1.22.
  The remaining published `x/sys` finding affects Windows, while MARS supports
  macOS and Linux; it remains explicitly dispositioned rather than weakening
  the minimum-version contract. T-055 also made a missing or failing
  `govulncheck` invocation block `make vuln` with a pinned recovery command.
  Offline environments must install the pinned scanner before making a passing
  source-security claim.
- **2026-07-22 — the source compatibility floor moved without changing target or packaged requirements:**
  AD-314 supersedes the current source-floor portion of the 2026-07-12
  discovery with exact Go 1.25.12 compatibility and retained release
  toolchain Go 1.26.5. The historical dependency disposition above remains
  unchanged evidence; generated targets still choose their own toolchain.
- **2026-08-08 — exact gRPC v1.82.1 restored the called-vulnerability gate:**
  T-071 changed only `go.mod` and `go.sum`, retaining the same gRPC module
  requirements while closing GO-2026-6061. Exact run `31278506189` passed
  Go 1.25.12, Go 1.26.5, and the intentional Go 1.25.11 rejection at commit
  `59ab946`. Local and remote `govulncheck v1.6.0` report zero called
  application vulnerabilities; two uncalled findings remain visible and are
  not described as zero advisories.
- **2026-08-24 — launch production moved to exact Go 1.27 and govulncheck v1.7:**
  AD-315's conventional source producer uses Go 1.27.0, and source CI retains
  Go 1.25.12 compatibility while adding the exact Go 1.27.0 lane. The
  vulnerability remediation command moved to `govulncheck v1.7.0`. Its first
  exact-toolchain scan found called GO-2026-5970 in `x/text v0.38.0`; upgrading
  to `x/text v0.39.0` restored zero called vulnerabilities. Earlier Go
  1.26.5/v1.6 evidence remains historical rather than current guidance.
- **2026-08-25 — the supported source patch floor moved to Go 1.25.13:**
  Hosted Go 1.25.12 source compatibility reported six called standard-library
  vulnerabilities fixed in Go 1.25.13. The bounded remediation raises only
  the MARS source/bootstrap floor and intentional below-minimum lane, retains
  exact Go 1.27.0 release production, and adds no vulnerability exception.
- **2026-06-11 — Seeding floors from one run is boundary-fragile:** two
  coverage runs on the same tree differed by up to 3 points for
  `cmd/mars` (test additions between runs) and packages landing on
  exact whole points (65.0, 62.0) would fail on any negative jitter; the seed
  rule therefore backs off one point at exact boundaries and the ratchet
  procedure handles the rest.
