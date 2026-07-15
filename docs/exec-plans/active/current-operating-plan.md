# Active P0 Exec Plan: Private Baseline Reconstruction

**Status:** Active
**Priority:** P0
**Depends On:** Operator-approved private rewrite authority and an unchanged live remote lease manifest
**Blocks:** Standard publication scans, remaining runtime/release/contribution gates, private rehearsal, and public cutover
**Related Tickets:** T-063
**Current Ticket:** T-063
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-017-open-source-publication.md
**Related Feature Contracts:** F-001, F-009, F-017
**Hypothesis:** A small exact reconstruction from the reviewed baseline plus fail-closed release mirroring will produce a safer and more auditable publication surface than retaining the abandoned embedded audit implementation.
**Success Evidence:** The replacement tree, private history/ref transaction, protected work, private `v0.93.0` assets, and fresh-clone manifest all pass exact mechanical and independent review gates.
**Falsification Evidence:** Any unapproved path or audit machinery survives, protected work is lost, remote leases drift, retired refs/releases remain exposed, tests fail, or the replacement release cannot prove exact 9/9 convergence.
**Scenario Schedule:** F-017-S001 private reconstruction, standard surface review and owner disposition; F-017-S002 runtime gates; F-017-S003 public release contract; F-017-S004 contribution controls; F-017-S005 rehearsal, cutover, and canary
**Current Failing Scenario:** F-017-S001 reconstruction slice: T-063 has not completed the private rewrite and replacement-release evidence
**Walking Skeleton Slice:** Rebuild private `main` from `v0.68.49`, retain only reviewed T-061 behavior and lean control docs, retire the captured audit-era refs/releases, and verify private `v0.93.0`
**Learning Or MVP Outcome:** A compact private publication candidate exists with exact history and release truth, without claiming that later publication scenarios pass
**Created:** 2026-07-16
**Owner:** foundation-maintainer as Orchestrator using COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager packets
**Source:** Operator-approved complete private rewrite.

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** F-017-S001 through F-017-S005 pass after the private reconstruction, standard scans and owner dispositions, secure runtime and release gates, fork-safe contribution controls, logged-out cutover smoke, and a clean 48-hour canary.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** T-063 has not completed the private history/ref/release reconstruction and verified replacement release. Publication remains prohibited after T-063 until the remaining F-017 scenarios pass.
- **Next Primary Action:** Finish T-063 in isolation, obtain independent QA/Security review, validate the replacement tree and private release, then execute the exact leased private rewrite.

## Approved Reconstruction Contract

1. Use `v0.68.49` commit `96548d7a8409206bc18ec5169b643a28f7687074` and tree `c00b7387cc1cfa3bdc6d5c70d301a1817c037f80` as the source baseline.
2. Apply only the reviewed T-061 release-mirror delta from `51de9c0627e57dd85c87a0b25aa1edcc6c72b044..6a84f81a0d76b583b7c6315aa96060ab21817ea8` plus the manual AD-312 index entry.
3. Permit only the T-061 source/docs allowlist, the two neutral personal-path corrections, this goal/plan/feature set, the generated T-063 ticket, and version/release-note files.
4. Exclude the abandoned embedded publication-audit implementation and its planning or validation artifacts.
5. Keep `CHANGELOG.md` at the `v0.68.49` history during reconstruction, reserve `VERSION` and buildinfo at `0.92.2`, and create one new `0.93.0` release entry after the semantic reconstruction commit.
6. Retire exactly the 69 private tags and GitHub Releases after `v0.68.49` through `v0.92.2`; preserve `v0.68.49` and every earlier release.
7. Preserve protected stash content by verified re-anchoring on the rewritten history; exact old stash object IDs are not a completion requirement because they retain retired ancestry.
8. Keep repository visibility private throughout. No announcement, public license change, or public cutover is part of T-063.

## Execution Sequence

| Order | Outcome | Gate |
| ---: | --- | --- |
| 1 | Build the replacement history in an isolated no-local clone. | Exact baseline and T-061 patch hashes; no shared-worktree or remote mutation. |
| 2 | Apply only approved lean docs, privacy corrections, ticket, and temporary version floor. | Changed-path oracle and intermediate tree oracle pass. |
| 3 | Run source, race, vulnerability, DocSync, cross-build, installed-binary, prospective release, and forbidden-path gates; then create the semantic reconstruction commit. | Engineer, QA, and Security agree on the exact candidate commit and asset hashes. |
| 4 | Freeze repository writes and force-push only reconstructed `main` with its exact captured lease. | The live `main` lease matches `3a847ad427e58d52e52a6addcbae63375976fd5e`; no tag or Release mutation occurs in this step. |
| 5 | Wait for Pages to build from reconstructed `main`, then delete the 69 captured GitHub Release objects by immutable ID. | Rebuilt Pages succeeds; every expected Release is independently absent; unexpected drift stops the operation. |
| 6 | Delete only the 69 captured remote tags in one atomic push with per-tag OID leases, then remove eligible discarded-history Actions runs, artifacts, caches, and deployments while preserving the rebuilt Pages run/deployment. | Exact leases and immutable IDs match; no broad or unleased fallback is allowed. |
| 7 | Record actual remote reconciliation evidence in T-063 and this plan, create and fast-forward the evidence commit, then generate and fast-forward the explicit `0.93.0` release-note commit. | Main advances only from the reconstructed commit through the evidence and release-note commits; the changelog retains `0.68.49` then `0.93.0`. |
| 8 | Create immutable `v0.93.0`, publish and verify exact 9/9 private assets, and verify a fresh authenticated private clone. | Remote main/tag/release/ref manifests and freshly downloaded bytes match. |
| 9 | Re-anchor protected stashes, prune retired local refs/reflogs, run local Git GC, and remove verified MARS-only temporary artifacts and audit volumes. | Both protected payloads materialize; shared Go caches and unrelated owner state remain untouched; allocated-byte reclamation reconciles. |

## Changed-Path And History Gates

- The intermediate baseline-plus-T-061 tree must equal `ca39e5a348fd299ca7e0fd9060f2fd6a08eaa8d9` before lean edits.
- The 16-path T-061 patch must hash to `baaecbc27c14831768e17a19ce6f91405e0959e72dba0780f9fc5ecb486d5d3a`.
- Every changed path must appear in the frozen reconstruction allowlist; any missing, extra, conflicted, or regenerated path is a no-go.
- No publication-audit source file, tool registration, ticket, report, or role surface may survive in the replacement tree.
- `CHANGELOG.md` must contain the retained history through `0.68.49` followed only by the new `0.93.0` entry; retired private release entries must not be copied.
- A fresh clone must expose only retained tags through `v0.68.49` plus `v0.93.0`; inaccessible GitHub object-storage garbage collection is not claimed.

## Validation Gates

- `git diff --check`
- focused T-061 normal and race tests
- `go test ./...` and the full all-package race suite
- `go vet ./...`, coverage ratchets, fuzz smoke, and fail-closed `govulncheck`
- `mars docsync audit --repo .` and docs-consistency tests
- CGO-disabled linux/darwin amd64/arm64 builds
- installed-binary clean-project validation
- exact-nine local, remote, fresh-download, checksum, alias, version, and commit verification
- private visibility, sole `main`, absent retired refs/releases, and fresh-clone history checks
- protected-stash materialization and unrelated-work preservation

## No-Go Rules

Stop before remote mutation if live `main`, any captured tag lease, release count, branch set, ruleset state, visibility, candidate hash, protected payload, or required gate differs from the frozen manifest. Do not substitute a broad force push, combine the main rewrite with tag deletion, partially delete tags, create a history backup ref, or accept best-effort asset upload.

Publication remains blocked after a successful rewrite. The next plan slice uses conventional, operator-visible tools for history/secret/privacy/IP review, followed by the remaining runtime, public release, contribution, rehearsal, cutover, and canary scenarios. No embedded audit runtime is reintroduced.
