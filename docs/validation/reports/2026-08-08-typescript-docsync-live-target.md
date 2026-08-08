# TypeScript Monorepo DocSync Live-Target Validation

**Date:** 2026-08-08
**Source Branch:** `codex/typescript-docsync`
**Source Baseline:** `d2db7c522795fa2698421434f1d9d4ebd2ec3f02`
**Feature:** F-019-S001
**Ticket:** T-071
**Failure Ownership:** foundation-owned generated-default and DocSync audit behavior

## Primary Outcome Contract

**Primary Outcome:** Publish MARS as a supported open-source project without
exposing confidential material, weakening controls, or distributing unsafe or
unverifiable binaries.

**Primary Pass Gate:** F-017-S001 through F-017-S005 pass, including anonymous
signed install/update, fork-safe contribution, logged-out cutover smoke, and a
clean 48-hour canary.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** GitHub-hosted surface review, manual owner
disposition, runtime, contribution, cutover, and canary gates remain incomplete.

**Next Primary Action:** Create the next bounded F-017-S001 ticket for retained
GitHub-hosted surfaces without changing repository visibility.

**Supporting Evidence:** This report proves a bounded foundation compatibility
prerequisite for a TypeScript target; it does not claim progress through the
publication pass gate.

## Target And Commands

A clean ephemeral target was initialized at
`/private/tmp/mars-docsync-live-target-20260808` from the branch source using:

```text
go run ./cmd/mars init --repo /private/tmp/mars-docsync-live-target-20260808
```

The fixture then added one authored `apps/web/src/page.tsx` with valid
MarsDocSync metadata and one metadata-free generated
`apps/web/dist/page.js`. Validation ran:

```text
go run ./cmd/mars docsync audit --repo /private/tmp/mars-docsync-live-target-20260808
go run ./cmd/mars docsync audit --repo /private/tmp/mars-docsync-live-target-20260808 --json
go run ./cmd/mars tools run docsync_audit --repo /private/tmp/mars-docsync-live-target-20260808 --trust observer --args-json '{}'
```

## Result

- Generated `.harness/manifest.yaml` contained the TypeScript-monorepo roots,
  extensions, and safe exclusion defaults.
- CLI text output reported `checked 1 files, findings 0` and `Status: ok`.
- CLI JSON selected only `apps/web/src/page.tsx` with its target-owned F-001
  documentation pointer.
- The mirrored tool reported the same one-file, zero-finding result.
- `apps/web/dist/page.js` was excluded before metadata validation.
- No runtime, model, network, database, or production resource was required.

## Remaining Scope

This proves the clean generated-target DocSync path, not autonomous role
execution or a production target upgrade. Existing target manifests remain
user-owned and are not overwritten by `mars upgrade`; operators adopt the new
selection fields deliberately when they want non-default coverage.
