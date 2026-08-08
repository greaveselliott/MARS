# Completed P0 Exec Plan: Add TypeScript Monorepo DocSync Coverage

**Status:** Completed
**Priority:** P0
**Depends On:** None
**Blocks:** MARS-native bootstrap for the Real-Time Chess TypeScript monorepo
**Related Tickets:** T-071
**Current Ticket:** T-071 complete
**Goals:** G-DOCSYNC-TS-001, G-001, G-002, G-003
**BDD Feature:** F-019-typescript-monorepo-docsync.md
**Related Feature Contracts:** F-001
**Hypothesis:** Configurable source roots, extensions, and exclusions with safe TypeScript monorepo defaults will let generated target harnesses enforce the existing no-stale-documentation contract without auditing generated dependency or build output.
**Success Evidence:** F-019-S001 passes through unit, CLI, mirrored-tool, generated-target, DocSync, and full-suite evidence on a clean target fixture containing TypeScript and TSX workspaces.
**Falsification Evidence:** TypeScript files escape audit, configured exclusions are ignored, unsafe paths escape the repository, CLI and mirrored-tool results diverge, or existing Go/static targets change behavior unexpectedly.
**Scenario Schedule:** T-071 implements and verifies F-019-S001; then restore the restricted-publication plan from `docs/exec-plans/backlog/restricted-publication-audit.md` unless the operator selects another priority.
**Current Failing Scenario:** None — F-019-S001 passed on 2026-08-08.
**Walking Skeleton Slice:** Initialize a target manifest with DocSync defaults, audit a nested `apps/web/src/page.tsx`, exclude generated output, and return the same result through the CLI and mirrored tool.
**Learning Or MVP Outcome:** Prove that MARS can govern a modern TypeScript monorepo through its existing documentation operating model without adding a target-specific auditor.
**Owner:** foundation-maintainer as Orchestrator with CTO-weekly, Engineer, QA, and Dogfood

## Delivery Contract

- Read optional `docsync.include_roots`, `docsync.include_extensions`, and `docsync.exclude_globs` from `.harness/manifest.yaml`.
- Use built-in Go/static plus TypeScript/TSX monorepo defaults when fields are absent; a non-empty field deliberately replaces that field's defaults.
- Reject absolute roots/globs, parent traversal, malformed extensions, and malformed glob patterns with actionable errors.
- Keep CLI, mirrored tool, successful disposition gates, and direct package callers on the same `docsync.Audit` path.
- Mirror discoverable defaults and configuration guidance into newly initialized targets without overwriting existing target manifests during upgrade.
- Preserve the source/target distinction: foundation prefix rules still apply only to MARS source packages, while target monorepo files point to target-owned feature and design docs.

## Completion And Handoff

- T-071 moves through `backlog` → `in-progress` → `done` with F-019-S001 evidence.
- Required checks: focused package tests, CLI tests, scanner/init tests, docs consistency, `mars docsync audit --repo .`, and `go test ./...`.
- This source-runtime/generated-default change also requires a clean generated-target fixture or explicit live-validation blocker.
- The restricted-publication plan is restored as the active plan. This branch does not change repository visibility or publish a release.
