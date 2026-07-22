---
id: T-067
title: Raise the MARS source Go floor to 1.25.12 without imposing Go at runtime
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-018-S002"]
end_to_end_evidence: required
evidence_links: ["go test ./... -count=1 under exact Go 1.25.12", "make check composite under exact Go 1.26.5", "docs/exec-plans/active/current-operating-plan.md#ticket-sequence"]
verified_by: "QA, Security, foundation-maintainer as Dogfood, Release Manager, and Orchestrator"
owner: "engineer"
last_attempt: "2026-07-22: exact minimum/release/negative lanes, source checks, cross-builds, and installed clean-target smoke passed"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "None; T-067 is complete. Resume T-066 checkpoint A1 with exact sigstore-go v1.2.2."
dedupe_key: "release:source-go-1.25.12-compatibility"
metadata:
  blocks: "T-066"
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  scope: "source-only"
  supports: "F-018-S002"
source: current-operating-plan.md — T-067 source compatibility prerequisite for F-018-S002, owner-approved 2026-07-22
created: 2026-07-22
depends_on: [T-065]
---

# T-067: Raise the MARS source Go floor to 1.25.12 without imposing Go at runtime

## Context

T-066 requires official sigstore-go v1.2.2, whose secure exact path is admissible only after the MARS source floor moves to Go 1.25.12. The owner approved this bounded compatibility prerequisite on 2026-07-22.

## Requirements

Set go.mod to exact go 1.25.12 while retaining toolchain go1.26.5. Update the current source Go-version check to compare major, minor, and patch and enforce the floor only for an explicit canonical MARS source checkout; packaged/default and ordinary target operation must not require an externally installed Go toolchain. Add exact minimum and release toolchain CI lanes with GOTOOLCHAIN=local, and prove 1.25.11 fails specifically at the module floor without auto-download. Update only source requirements and their design decision. Do not add Sigstore dependencies, trust roots, signing, publication, release notes, tags, Releases, visibility changes, or generated-target floor changes.

## Acceptance criteria

Exact Go 1.25.12 and release Go 1.26.5 source lanes pass tidy, build, tests, vet, vulnerability, and documentation gates; exact Go 1.25.11 with GOTOOLCHAIN=local is rejected for the module floor; doctor fails closed for missing, malformed, and older Go only in an explicit MARS source checkout and does not invoke Go for packaged/default or ordinary target operation; four CGO-disabled cross-builds pass; an installed commit-bound binary and fresh target smoke pass without an external Go requirement or source-floor injection; dependency graph, VERSION, CHANGELOG, build fallback, tags, Releases, Pages, visibility, and publication authority remain unchanged. Completion unblocks but does not complete F-018-S002 or T-066.

## Completion evidence

Exact Go 1.25.11 with `GOTOOLCHAIN=local` rejected the module with `go.mod requires go >= 1.25.12`. Exact Go 1.25.12 passed tidy with no `go.sum` drift, full uncached tests, vet, and `govulncheck` with zero called vulnerabilities. Exact Go 1.26.5 passed build, race/coverage at 73.9% with every ratchet met, zero-called vulnerability scanning, four fuzz-smoke lanes, and the vet fallback. Darwin/Linux AMD64/ARM64 CGO-disabled builds passed.

An installed CGO-disabled binary ran with a scrubbed environment and no Go executable in `PATH`, initialized and committed a fresh target, and reported the target `go-version` check as not required without injecting the MARS source floor. QA and Security independently approved the frozen diff. The dependency graph, `VERSION`, changelog, source fallback, tags, Releases, Pages, visibility, and publication authority did not change; F-018-S002 remains open for T-066.
