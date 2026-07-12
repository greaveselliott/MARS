# Active P0 Exec Plan: Safe Open-Source Publication

**Status:** Active
**Priority:** P0
**Depends On:** Legal ownership and licensing clearance for publication; technical work has no legal dependency
**Blocks:** Public repository visibility, supported public release, historical asset withdrawal, and public announcement
**Related Tickets:** T-055
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-017-open-source-publication.md
**Related Feature Contracts:** F-001, F-005, F-007, F-009, F-010, F-011, F-017
**Hypothesis:** Separating reversible technical readiness from irreversible publication authority, and proving each gate through BDD evidence, will permit a supported open-source launch without confidential exposure, unsafe defaults, unverifiable binaries, or privileged fork automation.
**Success Evidence:** F-017-S001 through F-017-S005 pass with legal attestation, redacted audit evidence, secure runtime defaults, anonymous signed release verification, fork-safe CI/governance, private rehearsal, logged-out cutover smoke, and a clean 48-hour canary.
**Falsification Evidence:** Any unresolved secret/privacy/IP finding; absent legal authority; open runtime P0s; failed anonymous release access or signature verification; privileged fork CI; unsafe historical binaries; failed logged-out smoke; or visibility changing before all no-go gates pass.
**Scenario Schedule:** F-017-S002 technical hardening; F-017-S003 public release contract; F-017-S004 contribution and governance controls; F-017-S001 audit and legal clearance; F-017-S005 rehearsal, cutover, and canary
**Current Failing Scenario:** F-017-S002, first bounded slice: vulnerability and baseline gate
**Walking Skeleton Slice:** Pin a non-vulnerable release toolchain, make `govulncheck` mandatory rather than silently skipped, resolve or disposition imported-module findings, and prove the source baseline with uncached tests, race, vet, fuzz smoke, and vulnerability scanning.
**Learning Or MVP Outcome:** The program gets a fail-closed security baseline without claiming that publication is legally authorized or complete.
**Created:** 2026-07-12
**Owner:** foundation-maintainer as Orchestrator using COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager packets
**Source:** Operator request to implement the MARS Open-Source Delivery Program.

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** A logged-out user can clone, build, install, update, report a vulnerability, and submit an externally reviewed contribution through the source-only public contribution path; exposed history is approved; runtime P0 findings are closed; public artifacts are licensed, signed, attributable, and tied to an immutable source commit; cutover smoke and the 48-hour canary pass.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** Publication authority is not established. Owner/legal review must confirm the right to license the source, predecessor material, prompts, docs, and assets, and trademark/name review must have no blocking result.
- **Next Primary Action:** Record the publication-authority decision while the separately authorized reversible technical lane begins F-017-S002.
- **Supporting Evidence:** On 2026-07-12 local `main` was clean and matched freshly fetched `origin/main` at `53bfd95`; the operator authorized technical hardening while reserving irreversible publication actions; `go.mod` still pinned Go 1.26.4 and `make vuln` still skipped missing `govulncheck`, confirming the first bounded slice.

## Publication Authority Boundary

Technical hardening, CI, tests, threat modelling, audits, documentation drafts,
release tooling, and private rehearsals may proceed.

Until legal clearance is durable, do not:

- change repository visibility or announce public availability;
- change copyright ownership or finalize ownership-dependent notice text;
- delete or publicly republish historical release assets;
- claim predecessor material, images, prompts, trademarks, or naming are cleared;
- publish the final supported open-source release.

Agent output may record evidence and risks but cannot substitute for an
owner/legal decision.

## Evidence And Classification

| Finding | Classification | Disposition |
| --- | --- | --- |
| Runtime, release/update, dashboard, webhook, filesystem, execution, CI, and documentation work belongs in MARS source. | foundation-owned | Deliver through F-017 ticket slices. |
| Secret handling, generated target ignores, execution disclosure, release-access guidance, and CLI/skill synchronization affect source and deployed defaults. | mirrored doctrine | Update source and generated surfaces together. |
| Scans, backups, inventories, validation reports, and release verification are proof rather than general doctrine. | evidence-only | Store redacted durable reports; keep raw sensitive material outside the repo. |
| Rights, predecessor provenance, historical personal-data acceptance, and trademark clearance cannot be inferred from code. | mixed/unclear | Require explicit owner/legal disposition before F-017-S001 passes. |
| No target application is being repaired. | deployed-owned: none | Do not create target-product tickets. |

## Scenario Schedule

| Order | Scenario | Outcome | Status |
| ---: | --- | --- | --- |
| 1 | F-017-S002 | Runtime control, filesystem, execution, secret, and baseline security gates are secure by default or explicitly gated. | Active; current slice is vulnerability baseline |
| 2 | F-017-S003 | Anonymous signed release/install/update contract is verified. | Planned |
| 3 | F-017-S004 | Fork-safe contribution, CI, rulesets, and governance controls are verified. | Planned |
| 4 | F-017-S001 | Ownership, history, privacy, provenance, and GitHub surfaces are approved. | Technical audit permitted; final pass blocked by legal clearance |
| 5 | F-017-S005 | Private rehearsal, logged-out cutover, rollback readiness, and 48-hour canary pass. | Blocked by F-017-S001 through F-017-S004 |

## Ticket Progress Ledger

Only the current scenario slice receives a ticket. Later rows are scheduled,
not yet ticket IDs.

| Order | Planned Slice | Scenario | State | Exit Evidence |
| ---: | --- | --- | --- | --- |
| 1 | Vulnerability and baseline gate | F-017-S002 | T-055 created through `ticket_create`; backlog | Non-vulnerable toolchain; fail-closed scan; test/race/vet/fuzz/vuln evidence |
| 2 | Release artifact integrity | F-017-S003 | Planned | Deterministic signed archive and negative verification tests |
| 3 | Dashboard, webhook, and HTTP security | F-017-S002 | Planned | Auth/origin/CSRF/XSS/HMAC/allowlist/loopback tests |
| 4 | Filesystem and secret containment | F-017-S002 | Planned | Descriptor-safe hostile-repository and staged-secret tests |
| 5 | Execution and local-state safety | F-017-S002 | Planned | Execution-profile, environment, PID, permissions, and redaction tests |
| 6 | Public-native release access | F-017-S003 | Planned | Anonymous setup/update and private fallback evidence |
| 7 | Threat model, claims, and provenance | F-017-S001, F-017-S002 | Planned | Threat model, corrected claims, immutable dependency/model provenance |
| 8 | Community and CI | F-017-S004 | Planned | Fork-safe CI, community files, DCO, ruleset rehearsal |
| 9 | Full-history and GitHub-surface audit | F-017-S001 | Planned | Redacted scan/inventory report and explicit dispositions |
| 10 | Private rehearsal and cutover | F-017-S005 | Public actions legally blocked | Matrix report, signed release, logged-out smoke, rollback, canary |

## Per-Ticket Role Loop

1. COO confirms Primary Outcome status and the current scenario.
2. CTO-weekly creates or refines one bounded ticket through `ticket_create`.
3. Engineer moves only that ticket to `in-progress/` and implements tests, docs, and `MarsDocSync`.
4. QA and Security review the completed diff concurrently.
5. Engineer addresses accepted findings.
6. Dogfood records installed-binary clean-project evidence or an exact replay blocker.
7. Release Manager commits, versions, backfill-checks, tags, publishes/verifies legally allowed private assets, and pushes, or records the blocker.
8. Orchestrator updates ticket, feature, plan, confidence, and Primary Status before scheduling the next slice.

## Assumption Confidence Matrix

| Assumption | Evidence | Confidence | Validation Required |
| --- | --- | ---: | --- |
| Reversible technical work is authorized while publication remains blocked. | Explicit operator instruction. | 1.0 | Reconfirm only for irreversible public action. |
| The operator has authority to license all publishable material. | Legal review was explicitly requested. | 0.2 | Owner/legal attestation covering source, predecessor material, prompts, docs, and assets. |
| Audited Git history can be preserved. | No completed all-ref/GitHub-surface audit in this program. | 0.4 | Pinned all-ref scans, manual IP/privacy review, and dispositions. |
| Runtime findings are reusable foundation concerns. | They affect MARS source/runtime rather than a target product. | 0.9 | Security review and hostile-path tests for each ticket. |
| Signed anonymous releases can replace private-auth-first distribution compatibly. | Existing release/update packages and compatibility aliases exist. | 0.8 | Bridge-release and anonymous clean-room install/update/rollback. |
| Fork-safe contribution can coexist with direct-trunk automation. | GitHub rulesets support narrow bypasses. | 0.8 | Disposable public hostile-fork rehearsal. |
| The MARS name is safe to promote. | No recorded clearance. | 0.3 | Trademark/name review and explicit owner acceptance. |

## Validation Gates

### Planning chain

- `git diff --check`
- `mars docsync audit --repo .`
- `go test ./internal/docsconsistency ./internal/docsync`
- `go test ./...`
- `mars doctor --repo .`
- `mars run foundation-maintainer --repo . --dry-run --no-init`
- Exactly one active exec plan and only the current-slice ticket.

### Ticket gate

Every ticket names associated docs, updates `MarsDocSync`, runs affected
unit/integration/security checks, and satisfies the AD-284 matrix union.
Runtime/release/update work requires installed-binary clean-project evidence or
a report with the exact blocker and rerun command.

### Publication gate

F-017-S001 through F-017-S005, independent Security/QA/Dogfood/Release Manager
evidence, legal clearance, logged-out smoke, rollback rehearsal, and the
48-hour canary must all pass. Supporting technical success cannot change the
Primary Status by itself.

## Current Evidence And Residual Risks

- F-016 and T-054 are complete and archived as planning-doctrine evidence.
- No eligible ticket was in progress when this plan began.
- Go 1.26.4 has a reachable GO-2026-5856 finding; Go 1.26.5 is the fixed baseline.
- On 2026-07-12, `make vuln` skipped when `govulncheck` was missing.
- Legal ownership and trademark clearance remain unrecorded.
- Legal review may require removing material or publishing a clean snapshot.
- Public cutover is irreversible in confidentiality terms.

## Planning Chain Evidence

- PASS 2026-07-12: `git diff --check`.
- PASS 2026-07-12: `mars docsync audit --repo .` checked 332 files with zero findings.
- PASS 2026-07-12: `go test ./internal/docsconsistency ./internal/docsync`.
- PASS 2026-07-12: `go test ./...`.
- PASS 2026-07-12: `mars run foundation-maintainer --repo . --dry-run --no-init` consumed AD-304 and AD-308.
- PASS 2026-07-12: active-plan hygiene is clean, exactly one active plan exists, and no eligible in-progress ticket exists.
- PASS 2026-07-12: T-055 was created through `ticket_create` for only the current F-017-S002 slice.
- EXPECTED WARN 2026-07-12: `mars doctor --repo . --skip-remote --json` reports missing target manifest/role registry because this is the source-only foundation repo; plan hygiene, ticket drain, and workspace hygiene are healthy.
