---
id: MH-022
title: mars setup orchestrating detect, models, GitHub App, bundle, verify, serve
priority: high
complexity: medium
source: delivery-schedule M8
created: 2026-04-11
---

# MH-022: `mars setup` — idempotent end-to-end bootstrap wizard

## Context

First-run friction blocks adoption. M8 needs one guided command that sequences hardware detection, model acquisition, GitHub App creation, bundle initialization, verification, and optional `serve` start—with safe replays.

## Requirements

- Subcommand `setup` with `--dry-run` and per-step flags to skip (`--skip-github`, etc.); `--test-mode` is an alias for `--skip-download --skip-github`
- Ordered steps: hardware detect → ensure model artifacts → GitHub App manifest flow (MH-011) → `init` bundle scaffold (MH-023 subset or call) → verify (inference ping, GitHub ping, DB migrate) → optional `serve` launch
- Idempotency: persisted state file records completed steps with hashes; re-run skips unless `--force-step`
- Clear TTY UX: progress, estimated time, copy-paste URLs; non-TTY JSON summary mode

## Acceptance Criteria

### Functional (happy path)
- [x] `--test-mode` works as alias for `--skip-download --skip-github`
- [x] Fresh machine completes setup to “ready” with all checkmarks on supported OS
- [x] Second run completes in under N seconds with “skipped” lines for each satisfied step
- [x] `--dry-run` prints intended mutations without writing secrets

### Edge cases and negative paths
- [x] Interrupted mid-step resumes from last incomplete step without corruption
- [x] Partial GitHub success (pem written, webhook secret missing) is detected and repaired on re-run
- [x] Verify failures print remediation commands (link to MH-028 when present)

### Non-goals
- Remote cluster provisioning
- Installing system GPU drivers automatically without user consent prompt

### Observability, docs, and regressions
- [x] Integration test with fakes for network-heavy steps
- [x] Machine-readable log artifact path documented for CI
- [x] README quickstart replaced or linked to `setup` as primary path
