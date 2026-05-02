---
id: MH-041
title: Configure shell PATH automatically for installed command
priority: high
complexity: medium
kind: standard
work_type: feature
bdd_scenarios: ["F-002-S001", "F-002-S002", "F-002-S003", "F-002-S004", "F-002-S005"]
end_to_end_evidence: required
evidence_links: ["go test ./internal/shellpath", "go test ./internal/setup", "go test ./internal/selfupdate", "go test ./...", "make install"]
verified_by: "Codex using go test and make install"
source: user feedback: Fish reported Unknown command mars-harness after install
created: 2026-05-02
depends_on: []
---

# MH-041: Configure shell PATH automatically for installed command

## Context

After installing the binary, Fish could not resolve `mars-harness` until the Go
bin directory was added manually. That violates plug-and-play: the harness
should configure supported user shells automatically wherever the command is
installed or updated.

## Requirements

- Add a reusable shell PATH configurator.
- Support Fish, Zsh, Bash, POSIX sh/Ksh, Csh, and Tcsh.
- Make updates idempotent and user-owned.
- Wire the configurator into `make install`, `mars-harness setup`, and
  `mars-harness update tool`.
- Add an explicit repair command for direct use.
- Document the behavior in product and quickstart docs.

## Affected Files

- `Makefile`
- `cmd/mars-harness/main.go`
- `internal/shellpath/`
- `internal/setup/`
- `internal/selfupdate/`
- `docs/design-docs/release-versioning.md`
- `docs/features/F-002-zero-config-shell-path.md`
- `docs/product-specs/product-surface.md`
- `docs/quickstart.md`
- `AGENTS.md`

## BDD Evidence

- Scenario IDs: F-002-S001, F-002-S002, F-002-S003, F-002-S004, F-002-S005
- Evidence links:
  - `go test ./internal/shellpath`
  - `go test ./internal/setup`
  - `go test ./internal/selfupdate`
  - `go test ./...`
  - `make install`
- Verified by: Codex using `go test` and `make install`

## Acceptance Criteria

### Functional

- [x] `make install` invokes shell PATH setup after `go install`.
- [x] `mars-harness setup` includes shell PATH configuration.
- [x] `mars-harness update tool` configures shell PATH after reinstalling the binary.
- [x] `mars-harness path setup` exists for direct repair.
- [x] Fish, Zsh, Bash, POSIX sh/Ksh, Csh, and Tcsh profile snippets are supported.

### Edge cases and negative paths

- [x] Re-running setup does not duplicate managed profile snippets.
- [x] Unsupported shells return a manual remediation message and do not write random files.
- [x] Dry runs do not mutate profile files.

### Non-goals

- Mutating the already-running parent shell process after command exit.
- System-wide `/etc/paths` modification.
- Checksum-verified release-asset installer work; that remains `MH-031`.

### Observability, docs, and regressions

- [x] Product spec and quickstart describe automatic shell PATH setup.
- [x] F-002 feature contract records scenario evidence.
