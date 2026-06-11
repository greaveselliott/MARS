---
id: T-025
title: Add govulncheck and fuzz targets for hostile model output parsers
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - "AD-281 in docs/design-docs/source-quality-gates.md, indexed in docs/design-docs/index.md"
  - "30s fuzz runs per target on 2026-06-11: ~1.0-1.2M execs each, clean after the array-path parser fix"
  - "Fuzz-found bug: '[{}]' produced an unnamed tool call; fix in internal/agent/parser.go, regression seed committed in internal/agent/testdata/fuzz/"
  - "govulncheck 2026-06-11: 6 reachable stdlib vulns at go1.26.2, 0 reachable after pinning toolchain go1.26.4"
  - "make vuln and make fuzz-smoke pass locally and run inside make check"
verified_by: "command"
owner: "foundation-maintainer"
last_attempt: "2026-06-11"
blocker: "none"
blocked_by: []
trace_id: "none"
next_action: "Done; fuzz targets and govulncheck wired into the local delivery gate."
source: Foundation improvement plan Phase 2 WS-C (provisional T-025)
created: 2026-06-11
depends_on: []
---

# T-025: Add govulncheck and fuzz targets for hostile model output parsers

## Context

The 2026-06-11 foundation review found no vulnerability scanning and no fuzzing despite two surfaces that parse hostile model output: the tool-call parser in internal/agent/parser.go (structured fields, function tags, inline tool_call tags, embedded JSON) and the T-015 list-string/argv normalizers in internal/tools (decodeStringSliceArg, parsePythonStyleStringList, normalizeShellExecArgv). Local models emit malformed payloads routinely; these parsers must never panic.

Ownership classification: foundation-owned CI/tooling enabler. The parser fix is a behavior change but only rejects structurally invalid (unnamed) tool calls that would have failed downstream; covered by the package test suite, no canary replay required.

**Scope adaptation:** drafted against GitHub Actions CI, but trunk retired workflows for local delivery gates; govulncheck and the fuzz smoke run land in the Makefile gates instead.

## Requirements

- Go fuzz targets: FuzzToolCallsFromAssistantMessage (internal/agent), FuzzDecodeStringSliceArg, FuzzParsePythonStyleStringList, FuzzNormalizeShellExecArgv (internal/tools).
- Seed corpora from internal/agent/testdata recordings and representative list-string shapes.
- govulncheck wired into the local delivery gate with graceful degradation when the tool is unavailable (clear install message).
- Short fuzz smoke run (FUZZTIME, default 10s per target) wired into `make check`; full fuzzing stays local/manual.

## Affected Files

- internal/agent/parser_fuzz_test.go (new)
- internal/agent/parser.go (fuzz-found fix: array path rejects unnamed tool calls)
- internal/agent/testdata/fuzz/ (regression seed)
- internal/tools/string_slice_args_fuzz_test.go (new)
- Makefile (vuln + fuzz-smoke targets, check integration)
- go.mod (toolchain go1.26.4 pin for stdlib vulnerability fixes)
- docs/design-docs/source-quality-gates.md (AD-281)
- docs/design-docs/index.md

## BDD Evidence

- Scenario IDs: none (enabler)
- Evidence links: see frontmatter
- Verified by: command

## Acceptance Criteria

### Functional (happy path)
- [x] Fuzz targets run under go test (seed corpus) and under -fuzz with bounded fuzztime.
- [x] govulncheck runs in the delivery gate and reports current module status (0 reachable after toolchain pin).

### Edge cases and negative paths
- [x] Fuzz harnesses assert no panics and structural invariants (non-empty names, valid JSON arguments, no invalid UTF-8 from valid input, no silently dropped argv).
- [x] Missing govulncheck binary degrades with an actionable install message instead of a cryptic failure.

### Non-goals
- Long-running continuous fuzzing infrastructure.
- Fixing pre-existing vulnerabilities beyond reporting them (the stdlib set was bounded: a toolchain pin resolved all reachable findings in this slice).

### Observability, docs, and regressions
- [x] AD-281 recorded in source-quality-gates.md and indexed; fuzz-found bug and govulncheck findings recorded as discoveries.
- [x] New test files carry MarsDocSync metadata; docsync audit clean.
