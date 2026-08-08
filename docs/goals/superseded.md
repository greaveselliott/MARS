# Superseded Goals

Paused, validated, invalidated, merged, split, or replaced goals move here with
the evidence and date that closed them.

## G-DOCSYNC-TS-001: Govern TypeScript Monorepos With MarsDocSync

- ID: G-DOCSYNC-TS-001
- Status: validated
- Category: operational
- Priority: P0
- Confidence: high
- Source: user_chat
- Dedupe Key: docsync:typescript-monorepo-coverage
- Hypothesis: Target-owned DocSync roots, extensions, and exclusions plus safe TypeScript defaults will make the no-stale-documentation contract work for modern web/mobile monorepos without target-specific audit code.
- Success Evidence: F-019-S001 passed on 2026-08-08 through focused/full/race tests, source DocSync, and the clean generated-target CLI/tool validation at `docs/validation/reports/2026-08-08-typescript-docsync-live-target.md`.
- Falsification Evidence: TypeScript code escapes audit, dependency/build output is audited, target configuration can escape the repository, or existing target behavior regresses.
- Competes With: hard-coded framework-specific scanners, auditing every repository file indiscriminately
- Supports: G-001, G-002, G-003
- Last Reviewed: 2026-08-08
- Review Trigger: DocSync root/extension changes, generated manifest changes, TypeScript target bootstrap, or stale-doc findings in monorepos.
- Owner: foundation-maintainer with CTO-weekly, Engineer, QA, and Dogfood
- Closure: Validated by T-071 and F-019-S001 on 2026-08-08.
