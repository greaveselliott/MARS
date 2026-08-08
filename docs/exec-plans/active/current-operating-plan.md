# Active P0 Exec Plan: Launch MARS As A Supported Open-Source Project

**Status:** Active
**Priority:** P0
**Depends On:** T-070 and F-018-S001 through F-018-S003 complete
**Blocks:** public visibility, supported v0.69.0/v0.69.1 releases, announcement, and G-OSS-001 completion
**Related Tickets:** T-058 and T-071 through T-081
**Current Ticket:** none — T-071 complete; create T-072 only through `ticket_create`
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-017-open-source-publication.md
**Related Feature Contracts:** F-001, F-010, F-018
**Hypothesis:** Sequential closure of hosted surfaces, publication rights, reachable runtime risks, anonymous signed distribution, fork-safe governance, and a public canary will produce a safe supported open-source launch.
**Success Evidence:** F-017-S001 through F-017-S005 pass and the Primary Pass Gate below is durably evidenced.
**Falsification Evidence:** Any unresolved launch no-go reaches visibility, a supported release, or announcement.
**Scenario Schedule:** T-071; T-072; T-073; T-074; T-075; T-076; resumed T-058; T-077; T-078; T-079; T-080; T-081.
**Current Failing Scenario:** F-017-S001 — retained GitHub-hosted publication surfaces are not yet frozen, collected, scanned, and reconciled.
**Walking Skeleton Slice:** Freeze hosted writes, collect every applicable GitHub surface with standard tools, scan offline, and commit only redacted counts, digests, classes, and exact cleanup manifests.
**Learning Or MVP Outcome:** Establish whether the retained hosting-side publication surface is technically eligible for rights and owner disposition.
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, Engineer, QA, Security, Dogfood, Release Manager, and repository owner

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** The repository is public; signed `v0.69.1` is the supported release with signed `v0.69.0` retained only as its rollback bridge; F-017-S001 through F-017-S005 pass; logged-out macOS/Linux clone, build, bootstrap, setup, update, and rollback pass; fork-contribution controls pass; GitHub security and community surfaces are active; a 48-hour public canary is clean; and the launch announcement is posted.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** T-071 restored the green vulnerability baseline at `59ab946` and run `31278506189`. T-072 must still collect and reconcile every retained GitHub-hosted publication surface with zero scanner error, skip, or unresolved finding.
- **Next Primary Action:** Create T-072 through `ticket_create`; do not start T-073 or mutate legacy assets before T-072 passes.

## Starting Baseline

- Clean private `main` and `origin/main` were equal at `d2db7c522795fa2698421434f1d9d4ebd2ec3f02` when this plan activated.
- `VERSION=0.68.49`; source fallback is `0.69.0-dev`.
- T-070 passed the advertised-Git audit with zero unresolved findings. That evidence does not cover retained GitHub-hosted content or complete F-017-S001.
- F-018-S001 through F-018-S003 passed as private producer, consumer, and rehearsal evidence only.
- Recorded GitHub baseline: 302 tags, 56 Releases, 500 legacy assets, 392 workflow runs, 77 deployments, zero packages, one collaborator, and no current Actions artifacts or caches. T-072 must freeze and independently reconcile these counts before relying on them.
- Rulesets, branch protection, CodeQL, Dependabot alerts, secret scanning, and push protection are not yet enabled.

## Assumption Confidence Matrix

| Assumption | Evidence | Confidence | Validation Required |
|---|---|---:|---|
| The starting source is clean and synchronized at `d2db7c…` | Local HEAD and `origin/main` matched; worktree was clean | 1.0 | Re-fetch before the T-071 source commit |
| T-070 clears the advertised Git surface | 12,002 reachable objects, four scanner lanes, zero errors, skips, or unresolved findings | 1.0 | T-072 rechecks publication-ref drift |
| gRPC v1.82.1 closes the called advisory | Source selects v1.82.1; local gates and exact run `31278506189` report zero called vulnerabilities | 1.0 | Complete — T-071 |
| Current hosted-surface counts remain stable | Recorded private GitHub inventory | 0.90 | T-072 freeze and exact reconciliation |
| Elliott's authority can be converted into a complete publication attestation | Owner stated authority; formal disposition is absent | 0.80 | T-073 committed owner attestation and provenance review |
| No reachable runtime P0/P1 remains after scheduled hardening | Existing webhook/dashboard evidence is partial | 0.60 | T-074 through T-076 and resumed T-058 |
| Exact-tag Go/SumDB bootstrap is non-circular on clean macOS/Linux | Design is selected; fresh packaged lifecycle proof is absent | 0.65 | T-077 clean-HOME macOS/Linux lifecycle |
| GoReleaser v2.17.1, Syft v1.50.0, and Cosign v3.0.6 can satisfy the producer gate | Official versions are selected; exact admission is pending | 0.80 | T-078 provenance, binary scan, and two-root proof |
| Required GitHub controls can be configured with owner authority | Owner has repository administration; controls were disabled on 2026-08-08 | 0.80 | T-079 disposable rehearsal and private configuration |
| `v0.69.0` and `v0.69.1` remain available for immutable release commits | As of 2026-08-08, no launch tags used these names | 0.95 | T-080 exact pre-tag remote recheck |
| The public canary will remain incident-free | Future operational result | 0.50 | T-081 continuous 48-hour observation and final replay |

## Launch Version Freeze

**Launch transition exception:** T-071 through T-079, including any semantic
correction required by resumed T-058, retain `VERSION=0.68.49` and
`DefaultVersion=0.69.0-dev`. Validated checkpoints are committed and pushed
without release-note generation, tag creation, signing, upload, publication,
or announcement. T-080 alone ends the pre-release freeze by generating
`0.69.0`, publishing its signed rollback bridge, then generating and publishing
signed `0.69.1` as the supported release. T-081 settings and evidence-only canary closeout
commits retain `0.69.1` without creating `0.69.2`; any product, runtime,
security, or public-contract correction discovered during canary requires
immutable `v0.69.2` and a repeated lifecycle and canary.

## Sequential Ticket Schedule

Only one implementation ticket is current. Each new ticket is created through
`ticket_create` only after its predecessor closes; T-058 is resumed rather
than recreated.

| Order | Ticket | Scenario ownership | Dependency |
|---:|---|---|---|
| 1 | T-071 — restore green vulnerability baseline | F-017-S002 prerequisite | T-070 |
| 2 | T-072 — audit GitHub-hosted publication surfaces | F-017-S001 | T-071 |
| 3 | T-073 — rights, provenance, notices, and owner disposition | F-017-S001 completion | T-072 |
| 4 | T-074 — close remaining network entry points | F-017-S002 | T-073 |
| 5 | T-075 — descriptor-safe repository filesystem and secret scanning | F-017-S002 | T-074 |
| 6 | T-076 — execution profiles, environment, state, and traces | F-017-S002 | T-075 |
| 7 | Resume T-058 — current-candidate browser proof | F-017-S002 and F-010-S024 | T-076 |
| 8 | T-077 — anonymous access, trusted bootstrap, and setup | F-017-S003 | T-058 |
| 9 | T-078 — production signing and legacy-asset sanitation | F-017-S003 and F-018-S004 | T-077 |
| 10 | T-079 — community, fork-safe CI, Pages, and GitHub controls | F-017-S004 | T-078 |
| 11 | T-080 — final private bridge and launch releases | F-017-S003 and F-017-S005 prerequisite | T-079 |
| 12 | T-081 — public cutover and 48-hour canary | F-017-S005 | T-080 and explicit owner visibility approval |

## Per-Ticket Operating Loop

1. COO confirms plan truth, current scenario, and Primary Status.
2. CTO-weekly freezes ticket scope, interfaces, blast radius, DocSync routes, and tests.
3. Engineer implements independently green semantic checkpoints.
4. QA and Security review each frozen diff concurrently.
5. Engineer resolves accepted findings within the same ticket.
6. Dogfood runs installed-current-candidate validation for runtime behavior changes.
7. Release Manager commits and pushes each accepted checkpoint immediately while enforcing the launch freeze.
8. Orchestrator updates ticket, BDD, plan, and goal evidence before advancing.

No custom audit runtime, VM laboratory, exhaustive kernel fixture, or parallel
source-ticket implementation is authorized.

## Completed Walking Skeleton — T-071

Commit `59ab946` upgraded only the selected gRPC dependency. Focused normal and
race suites, the complete local gate, DocSync, four cross-builds, zero called
vulnerabilities, and exact run `31278506189` all pass. No public interface,
version, release, or visibility state changed. F-017-S002 remains pending.

## Next Walking Skeleton — T-072

Freeze repository-hosted writes, collect all applicable Release, Actions,
deployment, Pages/Wiki, issue/discussion, attachment, access, integration,
package, and security metadata surfaces with standard tools, scan content
offline, and reduce only redacted coverage and exact immutable-ID cleanup
manifests. Inaccessible is unresolved; a plausible credential stops the ticket.

## Completion Gates By Ticket

- **T-072:** Every retained GitHub-hosted surface is collected, confirmed empty, or not applicable; scanners have zero errors, skips, and unresolved findings; exact cleanup IDs are frozen.
- **T-073:** Authority, predecessor/AI/media/model/llama.cpp/dependency provenance, name searches, notices, product claims, and `preserve_audited_history` disposition are complete with no deferred finding.
- **T-074:** GitHub App callback and telemetry collection are literal-loopback, bounded, replay-safe, and fail closed.
- **T-075:** All model/agent-controlled repository paths use one descriptor-relative no-follow interface; index-only and force-added secrets are scanned without reproducing values.
- **T-076:** Observer is non-mutating by default; host requires acknowledgement; isolated is unavailable; child environments, owned processes, state permissions, centralized redaction, export, purge, and retention pass.
- **T-058:** A real browser proves hostile DOM/SSE strings inert, browser controls protected, and embedded assets fully offline against an installed current candidate.
- **T-077:** Anonymous-first access, clear-local, exact-tag bootstrap, license-aware setup, and clean macOS/Linux operation without GitHub credentials pass.
- **T-078:** The protected exact-ten signing/publishing workflow passes; all 500 exact legacy assets and exact obsolete hosted objects are removed from active surfaces; immutable releases are enabled only after reconciliation.
- **T-079:** Community files, fork-safe CI, CodeQL/dependency/security controls, rulesets, Pages, Issues, Discussions, and hostile-fork rehearsal pass.
- **T-080:** Signed private `v0.69.0` and `v0.69.1` exact-ten releases pass macOS/Linux update and rollback; all final manifests and role sign-offs are complete; expected convergence is 304 tags, 58 Release objects, and 20 assets belonging only to the two launch releases.
- **T-081:** After separate owner confirmation, logged-out public lifecycle and hostile-fork smoke pass twice around a clean 48-hour canary; an announcement is posted and G-OSS-001/F-017/F-018 close.

## Global No-Go And Rollback Rules

Visibility cannot change while any secret, privacy, ownership, trademark,
provenance, license, called-vulnerability, reachable runtime P0/P1, unsigned
legacy asset, anonymous-access, fork-authority, settings, lifecycle, or sign-off
finding remains. Before visibility changes, stop and repair the owning ticket.
After visibility changes, confidentiality cannot be restored: supersede a bad
release with a new immutable patch tag and never move or replace a public tag.
A canary code or public-contract correction requires `v0.69.2` and repetition
of the complete anonymous lifecycle and 48-hour canary.
