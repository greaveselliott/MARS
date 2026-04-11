---
id: MH-015
title: Process sandbox, blast-radius limits, and emergency stop with GitHub cleanup
priority: high
complexity: large
source: delivery-schedule M5b
created: 2026-04-11
---

# MH-015: Process sandbox (Linux namespaces, macOS fallback), blast radius, emergency stop

## Context

Autonomous roles mutate repos and open PRs. M5b must limit filesystem and process damage on developer machines and CI-like Linux hosts, and provide a trustworthy “stop everything” path that also unwinds remote artifacts.

## Requirements

- Linux: optional namespace isolation for child processes (mount/pid/user where permitted); macOS documented fallback (cwd + ulimit-style limits only)
- Blast radius caps: max files touched, max lines changed, max open PRs per repo; default deny `git rm` / destructive deletes unless explicitly allowed by manifest flag
- Secret scanner pre-commit hook equivalent: block known high-entropy patterns and `.env` keys before push stage (configurable severity)
- Emergency stop: halt workers, cancel in-flight jobs cooperatively, then GitHub cleanup sequence: revert draft PRs to closed or mark draft, delete harness-owned branches matching prefix, cancel outstanding check runs created by harness

## Acceptance Criteria

### Functional (happy path)
- [ ] Tooling subprocess runs under sandbox profile on Linux CI image used by project
- [ ] Caps trigger clean abort with user-visible summary (which cap, current vs limit)
- [ ] Emergency stop completes remote cleanup in fixture org using test tokens (recorded steps)

### Edge cases and negative paths
- [ ] Insufficient privileges for namespaces → degrade with loud warning and stricter caps
- [ ] Partial GitHub cleanup failure → aggregated error report with per-resource status; idempotent retry
- [ ] Secret scanner false positive → override token in manifest with audit log entry

### Non-goals
- [ ] eBPF syscall filtering
- [ ] Full code review of third-party dependency source

### Observability, docs, and regressions
- [ ] Integration tests on Linux runner; macOS unit tests for fallback path only
- [ ] Audit log table entries for cap hits, overrides, emergency stop invocations
- [ ] Operator runbook: when to hit emergency stop, expected GitHub end state
