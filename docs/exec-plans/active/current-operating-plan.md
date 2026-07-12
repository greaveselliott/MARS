# Active P0 Exec Plan: Safe Open-Source Publication

**Status:** Active
**Priority:** P0
**Depends On:** Legal ownership and licensing clearance for publication; technical work has no legal dependency
**Blocks:** Public repository visibility, supported public release, historical asset withdrawal, and public announcement
**Related Tickets:** T-055, T-056, T-057, T-058
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-017-open-source-publication.md
**Related Feature Contracts:** F-001, F-005, F-007, F-009, F-010, F-011, F-017
**Hypothesis:** Separating reversible technical readiness from irreversible publication authority, and proving each gate through BDD evidence, will permit a supported open-source launch without confidential exposure, unsafe defaults, unverifiable binaries, or privileged fork automation.
**Success Evidence:** F-017-S001 through F-017-S005 pass with legal attestation, redacted audit evidence, secure runtime defaults, anonymous signed release verification, fork-safe CI/governance, private rehearsal, logged-out cutover smoke, and a clean 48-hour canary.
**Falsification Evidence:** Any unresolved secret/privacy/IP finding; absent legal authority; open runtime P0s; failed anonymous release access or signature verification; privileged fork CI; unsafe historical binaries; failed logged-out smoke; or visibility changing before all no-go gates pass.
**Scenario Schedule:** F-017-S001 read-only publication-surface inventory; F-017-S002 runtime hardening; F-017-S003 public release contract; F-017-S004 contribution and governance controls; F-017-S001 final legal disposition; F-017-S005 rehearsal, cutover, and canary
**Current Failing Scenario:** F-017-S002/F-010-S024 bounded runtime P0 slice T-058: embedded dashboard browser and authenticated control security
**Walking Skeleton Slice:** Require authenticated, origin-bound, CSRF-protected control mutations while keeping a loopback read-only dashboard usable with vendored assets and safe DOM construction when outbound networking is unavailable.
**Learning Or MVP Outcome:** Browser-delivered content and cross-origin requests cannot drive the local control plane, and the dashboard does not depend on runtime CDNs.
**Created:** 2026-07-12
**Owner:** foundation-maintainer as Orchestrator using COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager packets
**Source:** Operator request to implement the MARS Open-Source Delivery Program.

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** A logged-out user can clone, build, install, update, report a vulnerability, and submit an externally reviewed contribution through the source-only public contribution path; exposed history is approved; runtime P0 findings are closed; public artifacts are licensed, signed, attributable, and tied to an immutable source commit; cutover smoke and the 48-hour canary pass.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** Publication authority is not established. Owner/legal review must confirm the right to license the source, predecessor material, prompts, docs, and assets, and trademark/name review must have no blocking result.
- **Next Primary Action:** The owner provisions the restricted audit and legal/name-clearance work. The technical lane opens an in-app browser and runs T-058's remaining hostile-DOM and outbound-disabled browser-network cases against installed private v0.68.48; technical completion cannot change Primary Status.
- **Supporting Evidence:** T-055 closed the vulnerability baseline in private v0.68.42. T-056 recorded the redacted publication-surface inventory and isolated restricted-audit contract in private v0.68.44. T-057 semantic/release commits `23efb13`/`f7b9814`, private tag `v0.68.46`, all nine local and remote assets, and the rolling ten-release audit passed. T-058 source implementation, QA, Security, full normal/race/vet/fuzz/vulnerability/DocSync/docs gates, private v0.68.48 publication, and installed clean-target HTTP/SQLite security matrix pass; only real-browser DOM/network evidence remains blocked because the in-app browser inventory is empty. The operator authorized technical work while reserving irreversible publication actions.

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
| 1 | F-017-S001 initial inventory | Publication surfaces and restricted-audit inputs are enumerated without legal conclusions. | T-056 complete in private v0.68.44; restricted audit/final legal disposition blocked |
| 2 | F-017-S002 | Runtime control, filesystem, execution, secret, and baseline security gates are secure by default or explicitly gated. | Active; T-055 and T-057 complete; T-058 in progress with installed HTTP/SQLite PASS and mandatory real-browser DOM/network BLOCKED; later filesystem/execution runtime slices pending |
| 3 | F-017-S003 | Anonymous signed release/install/update contract is verified. | Planned |
| 4 | F-017-S004 | Fork-safe contribution, CI, rulesets, and governance controls are verified. | Planned |
| 5 | F-017-S001 final disposition | Restricted scans, privacy/IP/provenance review, authority, name clearance, and history decision are approved. | Owner/Security/legal work blocked or pending |
| 6 | F-017-S005 | Private rehearsal, logged-out cutover, rollback readiness, and 48-hour canary pass. | Blocked by F-017-S001 through F-017-S004 |

## Ticket Progress Ledger

Only the current scenario slice receives a ticket. Later rows are scheduled,
not yet ticket IDs.

| Order | Planned Slice | Scenario | State | Exit Evidence |
| ---: | --- | --- | --- | --- |
| 1 | Vulnerability and baseline gate | F-017-S002 | T-055 complete in private v0.68.42 | Non-vulnerable toolchain; fail-closed scan; test/race/vet/fuzz/vuln evidence |
| 2 | Publication history and GitHub-surface inventory | F-017-S001 | T-056 complete in private v0.68.44; downstream restricted audit/legal disposition blocked | Redacted inventory/access-gap report and offline scan contract; full scan/disposition remains pending |
| 3 | Loopback and GitHub webhook ingress | F-017-S002 | T-057 complete in private v0.68.46 | Loopback, HMAC, numeric actor, exact repo/branch/fork policy, disabled comments, body/replay/idempotency tests plus installed two-archetype evidence |
| 4 | Embedded dashboard browser and control security | F-017-S002, F-010-S024 | T-058 in progress; source/QA/Security and installed HTTP/SQLite lane pass, real-browser DOM/network lane blocked by unavailable browser | Session/Host/Origin/CSRF/method/body/rate/redaction/XSS/CSP/offline-asset and installed browser tests |
| 5 | Filesystem and secret containment | F-017-S002 | Planned | Descriptor-safe hostile-repository and staged-secret tests |
| 6 | Execution and local-state safety | F-017-S002 | Planned | Execution-profile, environment, PID, permissions, and redaction tests |
| 7 | Release artifact integrity | F-017-S003 | Planned | Deterministic signed archive and negative verification tests |
| 8 | Public-native release access | F-017-S003 | Planned | Anonymous setup/update and private fallback evidence |
| 9 | Threat model, claims, provenance, community, and CI | F-017-S001, F-017-S002, F-017-S004 | Planned | Threat model, corrected claims, provenance, fork-safe CI, governance |
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
- On 2026-07-12, T-055 declared Go 1.26.5, updated `x/sys` to the newest release compatible
  with the Go 1.22.4 minimum, records the unsupported-Windows disposition, and
  makes the `govulncheck` gate fail closed with pinned v1.6.0 installation
  remediation. Engineer, QA, and Security recorded a v1.6.0 scan with zero
  reachable findings; QA and Security passed the corrected diff; full uncached
  tests, race, vet, focused fuzz/DocSync checks, and Dogfood candidate smoke
  passed. AD-284 clean-project replay is not applicable to this build-gate-only
  ticket. Semantic commit `9c7db7d`, release commit/tag `51b9550`/`v0.68.42`,
  all nine local and remote assets, and the ten-release audit passed.
- T-056 was the bounded F-017-S001 publication-surface inventory slice.
  Raw GitHub/scanner evidence must remain outside the repo and agent transcripts;
  only redacted counts, gaps, classifications, and opaque evidence identifiers
  may become durable repository evidence.
- T-056 Engineer evidence inventories matching advertised/local publishable refs,
  local-only and unreachable review categories, and redacted GitHub aggregate
  surfaces. Package, security, App, environment, account/org, and retained
  content gaps remain unknown or routed to the owner-controlled restricted audit.
  The technical history recommendation remains `undecided`; no scan or legal
  conclusion was made. QA and Security passed the corrected restricted-audit,
  provenance, snapshot, gap-binding, and no-go contract; Dogfood confirmed
  AD-284/installed-binary replay is not applicable to the docs-only research
  diff. The owner-controlled restricted audit and legal disposition remain open.
- Legal ownership and trademark clearance remain unrecorded.
- Legal review may require removing material or publishing a clean snapshot.
- Public cutover is irreversible in confidentiality terms.

## Planning Chain Evidence

- PASS 2026-07-12: `git diff --check`.
- PASS 2026-07-12: `mars docsync audit --repo .` checked 332 files with zero findings.
- PASS 2026-07-12: `go test ./internal/docsconsistency ./internal/docsync`.
- PASS 2026-07-12: `go test ./...`.
- PASS 2026-07-12: `mars run foundation-maintainer --repo . --dry-run --no-init` consumed AD-304 and AD-308.
- PASS 2026-07-12: at program initialization, active-plan hygiene was clean, exactly one active plan existed, and no eligible in-progress ticket existed.
- PASS 2026-07-12: T-055 was created through `ticket_create` for only the current F-017-S002 slice.
- PASS 2026-07-12: T-055 and private v0.68.42 completed with local/remote asset verification and a clean release audit.
- PASS 2026-07-12: T-056 was created through `ticket_create` for only the current F-017-S001 research slice.
- PASS 2026-07-12: T-056 passed QA, Security, Dogfood, full-suite, DocSync, docs-consistency, forbidden-content, and diff validation; F-017-S001 remains blocked by the restricted audit and owner/legal disposition.
- PASS 2026-07-12: T-056 semantic/release commits `28254c9`/`4eda99c`, private tag `v0.68.44`, local/remote assets, and the rolling release audit passed.
- PASS 2026-07-12: T-057 was created through `ticket_create` for only the current F-017-S002 OSS-02 ingress slice.
- PASS 2026-07-12: T-057 Engineer, QA, and Security evidence closes the bounded OSS-02 source contract after accepted corrections for authentic check-suite shape, complete event metadata, direct actor-policy validation, hardened branch identifiers, App subscription, and owner-only atomic credential persistence.
- PASS 2026-07-12: the exact full repository race suite, uncached tests, vet, fuzz smoke, DocSync, docs consistency, and diff gates pass. The dependency-unchanged ticket relies on T-055's pinned clean vulnerability result because the restricted sandbox cannot resolve `vuln.go.dev`; the fail-closed gate correctly reports that network blocker.
- PASS 2026-07-12: installed clean static-browser and API/service targets prove loopback-only sockets, healthy optional-GitHub operation, zero mutation for rejected traffic, one authorized job, durable single-shot replay across completion/restart, and real local-model worker completion. F-017-S002 remains incomplete and Primary Status remains `primary_blocked`.
- PASS 2026-07-12: T-057 semantic/release commits `23efb13`/`f7b9814`, private tag `v0.68.46`, all nine local and remote assets, and the rolling ten-release audit passed; repository visibility remained PRIVATE and no publication action occurred.
- PLANNED 2026-07-12: COO, CTO-weekly, and Security packets classify the next bounded slice as the shipped embedded dashboard's browser/control boundary under new F-010-S024. GitHub setup callback and remote telemetry intake remain separate later tickets, and F-010-S012/MH-053 remains the future TanStack-specific gateway contract.
- PASS 2026-07-12: T-058 was created through `ticket_create` with dedupe key `open-source:dashboard-browser-control-security` for F-017-S002/F-010-S024 only; no second implementation ticket is current.
- PASS 2026-07-12: T-058 planning commit `ab70dad`, release commit/tag `f06fdc9`/`v0.68.47`, all nine local and remote assets, and the rolling ten-release audit passed; T-058 is now the sole in-progress implementation ticket.
- SUPPORTING 2026-07-12: T-058 source, QA, and Security correction loops pass. Full uncached tests, the exact all-package race suite, vet, fuzz smoke, the pinned fail-closed vulnerability gate with zero called vulnerabilities, DocSync 341/0, docs consistency, JavaScript syntax, asset hashes/licenses, forbidden application-owned sink/CDN, and diff gates pass.
- PASS 2026-07-12: T-058 semantic/release commits `0bd72b1`/`db35cc7` and private tag `v0.68.48` are pushed; repository visibility remained PRIVATE; all nine local and remote assets verify, and the rolling release audit passed. This is a supporting private release and does not complete T-058 or authorize publication.
- SUPPORTING 2026-07-12: installed private v0.68.48 passes the clean AD-284 static-browser HTTP/SQLite matrix for loopback sockets, bounded anonymous observation, authenticated reads, Host/Origin/session/CSRF/method/type/body/rate rejection with zero jobs, exactly one authorized mutation, SSE/session invalidation, security headers, and vendored asset hashes. No foundation-owned product failure surfaced.
- BLOCKED 2026-07-12: the required in-app browser inventory is empty after initialization and troubleshooting, so hostile strings have not been observed as inert text in a real DOM and outbound-disabled rendering has not been observed with zero external requests. T-058 remains the sole current ticket; F-010-S024 and F-017-S002 remain incomplete, and no next implementation ticket may start.
- EXPECTED WARN 2026-07-12: `mars doctor --repo . --skip-remote --json` reports missing target manifest/role registry because this is the source-only foundation repo; plan hygiene, ticket drain, and workspace hygiene are healthy.
