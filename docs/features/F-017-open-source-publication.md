# F-017: Open-Source Publication

- Feature ID: F-017
- Goals: G-OSS-001, G-001, G-002, G-003, G-004
- Status: active
- Owner: foundation-maintainer

## Business Logic

MARS remains private until publication authority and every technical cutover gate are durable. The operator states ownership of the first-party project and name and has authorized a complete private rewrite; that authorization permits the T-063 reconstruction but does not itself complete privacy, IP, runtime, release, contribution, or cutover validation.

The selected history strategy is a clean private reconstruction from `v0.68.49`, retaining only reviewed release-mirror convergence and lean publication planning. Private audit-era refs and releases are retired before publication. The rewrite removes them from accessible repository refs and releases, but does not claim that unreachable hosting-provider storage, external clones, or caches are physically recalled.

Supporting private evidence cannot authorize visibility conversion. Visibility is a separately approved owner action only after F-017-S001 through F-017-S005 pass.

## Step-By-Step Behavior

1. Keep the repository private and freeze writes before destructive ref or release operations.
2. Reconstruct and validate the approved source tree in isolation through T-063.
3. Use exact leases and an atomic remote ref transaction; unexpected remote drift is a no-go.
4. Preserve unrelated owner work and retain no rollback ref, tag, branch, or bundle after successful verification.
5. Run conventional history, secret, privacy, IP, provenance, and name checks against the replacement publication surface; inaccessible evidence remains unresolved rather than clean.
6. Close reachable runtime and local-state P0 findings without reintroducing an embedded publication-audit runtime.
7. Make official source and supported release access anonymous, signed, immutable, and attributable.
8. Require fork-safe contribution controls and private vulnerability reporting.
9. Rehearse the cutover privately and change visibility only through a separate approved owner action.
10. Run logged-out smoke immediately after cutover and wait for a clean 48-hour canary before announcement.

## Scenario Schedule

1. F-017-S001 - Complete T-063 private reconstruction, conventional publication-surface review, and owner disposition.
2. F-017-S002 - Close secure runtime, filesystem, execution, credential, logging, and update gates.
3. F-017-S003 - Verify anonymous signed install, update, rollback, and exact release publication.
4. F-017-S004 - Verify fork-safe contribution, governance, support, and private reporting.
5. F-017-S005 - Rehearse privately, perform separately approved cutover smoke, and complete the 48-hour canary.

## Scenarios

### F-017-S001: Publication Surface, Authority, Privacy, And Provenance

Given the replacement private history, repository configuration, releases, source, documentation, prompts, dependencies, models, and assets are the proposed publication surface
When standard pinned secret/history scans, manual privacy/IP/provenance review, name checks, and owner disposition complete
Then every surface is collected, confirmed empty, not applicable, or explicitly accepted as unrecoverable
And every finding is resolved without raw candidate values entering repository evidence
And the owner attestation records authority, third-party obligations, media/model provenance, accepted evidence gaps, and the clean-history strategy
And the scenario remains blocked if any surface, right, provenance chain, or finding is unresolved.

### F-017-S002: Secure Runtime Defaults

Given a new user runs MARS against an untrusted repository
When network listeners, browser controls, webhooks, filesystem tools, command execution, credentials, logs, traces, and updates operate
Then control listeners default to loopback and remotely exposed controls require explicit authentication and request protections
And untrusted events cannot enqueue autonomous mutation
And repository filesystem operations cannot escape through links or races
And observer execution is the new-install default while host authority requires explicit acknowledgement
And child environments, state permissions, logs, traces, and vulnerability gates fail closed without overstating containment.

### F-017-S003: Anonymous, Immutable, Verifiable Release Lifecycle

Given a supported release is built from an immutable release-note commit
When local assets are produced or a GitHub mirror is attempted
Then the local release contains the exact required platform assets and checksums
And already-matching remote assets are preserved
And the remote tag resolves to the exact local release-note commit before mutation and before success
And success requires the exact unique remote asset names, uploaded states, byte sizes, and SHA-256 digests
And missing, extra, duplicate, pending, mismatched, unverifiable, canceled, or non-converging state returns non-zero as `mirror_incomplete`
And upload-process exit zero alone can never become completion evidence
And the future supported public archive, signature, notices, SBOM, provenance, anonymous download, update, and rollback contract passes independently.

### F-017-S004: Fork-Safe Contribution And Governance

Given an untrusted contributor opens a fork pull request
When repository automation runs
Then it receives no secrets, write token, release key, or self-hosted privileged runner
And it cannot mutate protected refs or publish releases
And required CI, DCO, review, governance, support, conduct, and private vulnerability-reporting policies apply.

### F-017-S005: Controlled Cutover And Canary

Given F-017-S001 through F-017-S004 pass with required owner, Security, QA, Dogfood, Release Manager, and Orchestrator sign-offs
When the owner separately approves public visibility
Then final refs, settings, artifacts, recovery procedures, and credentials are verified
And logged-out clone, build, install, update, vulnerability reporting, and fork-PR smoke pass
And integrations are re-enabled least-to-most privileged
And announcement waits until the 48-hour canary remains clean.

## Evidence

- **F-017-S001:** T-063 is reconstructing the proposed private publication surface. Conventional scans, manual review, and committed owner disposition remain pending.
- **F-017-S002:** The retained baseline includes existing vulnerability, loopback, webhook, and dashboard hardening, but the complete runtime/filesystem/execution gate has not passed.
- **F-017-S003:** T-061 exact GitHub mirror convergence is retained and mechanically testable. Signed public archives, anonymous install/update, negative verification, and rollback remain pending.
- **F-017-S004:** Pending fork-safe CI, governance, reporting, and disposable hostile-fork rehearsal.
- **F-017-S005:** Pending private rehearsal, separate visibility approval, logged-out smoke, recovery proof, and 48-hour canary.

No scenario is complete. Primary Status remains `primary_blocked`, repository visibility remains private, and no publication or announcement is authorized.

## Out of Scope

- Reintroducing a repository-embedded publication audit laboratory.
- Treating the private rewrite as secret, privacy, IP, runtime, or cutover clearance.
- Claiming physical deletion from hosting-provider storage or third-party caches.
- Changing visibility, announcing the project, or finalizing unsupported historical releases during T-063.

## Descoped Scenarios

None.
