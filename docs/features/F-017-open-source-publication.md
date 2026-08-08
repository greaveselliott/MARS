# F-017: Open-Source Publication

- Feature ID: F-017
- Goals: G-OSS-001, G-001, G-002, G-003, G-004
- Status: active
- Owner: foundation-maintainer

## Business Logic

MARS remains private until publication authority and every technical cutover gate are durable. The operator states ownership of the first-party project and name and has authorized the private rewrite and GoReleaser transition; that authorization does not itself complete privacy, IP, runtime, release, contribution, or cutover validation.

The selected history strategy is a clean private reconstruction from `v0.68.49`. The private v0.93 experiment was retired through T-064 and replaced with a source-only GoReleaser archive/checksum/SBOM/signature pipeline through T-068. T-070 passed only the advertised-Git slice of F-017-S001. T-071 through T-081 now own the remaining vulnerability, GitHub-hosted, rights/provenance, runtime, anonymous distribution, contribution, private-release, cutover, and canary gates. Private audit-era refs and releases are retired before publication, without claiming that hosting-provider storage, external clones, or caches are physically recalled.

Supporting private evidence cannot authorize visibility conversion. Visibility is a separately approved owner action only after F-017-S001 through F-017-S005 pass.

## Step-By-Step Behavior

1. Keep the repository private and freeze writes before destructive ref or release operations.
2. Reconstruct and validate the approved source tree in isolation through T-063.
3. Use exact leases and an atomic remote ref transaction; unexpected remote drift is a no-go.
4. Preserve unrelated owner work and retain no rollback ref, tag, branch, or bundle after successful verification.
5. Restore a green called-vulnerability baseline through T-071, then freeze and audit every retained GitHub-hosted publication surface through T-072.
6. Complete authority, rights, privacy, IP, model/media/dependency provenance, name review, final notices, and owner disposition through T-073.
7. Close network, filesystem, execution, credential, state, log, and trace P0/P1 findings through T-074 through T-076; `isolated` remains unavailable and cannot claim containment without an enforceable adapter.
8. Resume T-058 only against the installed current candidate and close its real-browser/offline-network lane.
9. Make official source, setup, and release access anonymous; complete exact-tag bootstrap and signed immutable distribution through T-077 and T-078.
10. Configure fork-safe contribution, governance, Pages, Issues, Discussions, private reporting, rulesets, and GitHub security controls through T-079.
11. Privately publish and prove signed `v0.69.0` as a rollback bridge and signed `v0.69.1` as latest through T-080.
12. Change visibility only through a separate approved owner action, run immediate logged-out/fork smoke, and wait for a clean 48-hour canary before the T-081 announcement.

## Scenario Schedule

1. F-017-S001 - Complete the private reconstruction, conventional publication-surface review, and owner disposition.
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

Given supported rollback-bridge and latest releases are built from their immutable release-note commits
When pinned GoReleaser builds Darwin/Linux AMD64/ARM64 archives into fresh output directories
Then each archive contains the binary, license, notices, third-party notices, and deterministic source metadata
And SPDX SBOMs, SHA-256 checksums, and a keyless signature bind the published artifacts to the exact tag and commit
And an unpublished draft is fresh-downloaded and compared before publication
And missing, extra, duplicate, pending, mismatched, unsigned, unverifiable, canceled, or non-converging state never becomes a supported release
And signed `v0.69.0` is retained only as the rollback bridge while signed `v0.69.1` is latest
And anonymous install, update, rollback, immutable-release verification, and provenance verification pass independently across both releases.

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

## Launch Public Interfaces

- `setup` and official release access are anonymous-first; authentication is optional for rate limits, private forks, and custom repositories.
- `auth github check` reports anonymous, authenticated, or unavailable without exposing credentials; `auth github clear-local` removes only MARS's legacy stored fallback.
- `run`, `start`, and `serve` default to observer execution. Host execution requires explicit acknowledgement; `isolated` fails closed as unavailable until an enforceable adapter exists.
- `shell_exec` is host-only, receives a sanitized environment, and reports the operating-system authority accurately.
- Repository filesystem tools are descriptor-relative and no-follow; trace export is owner-only and redacted, while purge is dry-run-first.
- Telemetry collection and GitHub App setup bind only to literal loopback with the documented authentication, state, request, and timeout boundaries.
- The installer uses exact-tag Go/SumDB bootstrap into the signed updater; packaged operation remains Go-free.
- Each supported release has exactly four archives, four SPDX SBOMs, `checksums.txt`, and `checksums.txt.sigstore.json`.
- Contributions use fork PRs with DCO, CODEOWNERS, required fork-safe CI, and maintainer-only release authority.

## Evidence

- **F-017-S001:** The proposed private publication surface is reconstructed. On 2026-08-08, T-070 froze 302 canonical publication refs at `d04f642` and scanned all 12,002 reachable objects with both exact pinned tools in Git-history and raw-object lanes without accepted-scan errors or skips. Gitleaks group `a2a292e31d652f22` and URI group `e32927624f4a2cac` are resolved as synthetic test stubs. Direct Git-plumbing reconciliation proved all 36 occurrences in `f3dc0e336620abc6` are SHA-1 child blob IDs emitted by materialized Git tree objects, not credentials; every record matched its source tree, reachable child blob, and raw scanner field with zero mismatches. T-070 passes with zero unresolved findings. GitHub-hosted content review, manual privacy/IP/provenance/name review, and committed owner disposition remain blocked subsequent slices. Local stashes, administrative refs, unreachable objects, and the unrelated Codex worktree are preserved non-publication exceptions rather than inputs to T-070.
- **F-017-S002:** T-071 passed at exact commit `59ab946` and GitHub run `31278506189`: both supported Go lanes and the expected below-minimum rejection pass, while `govulncheck v1.6.0` reports zero called application vulnerabilities and GO-2026-6061 is absent. Existing loopback, webhook, and dashboard hardening remains supporting evidence; T-074 through T-076 and resumed T-058 still own the network/filesystem/execution/browser completion gate.
- **F-017-S003:** T-064 through T-068 completed the private v0.93 retirement, GoReleaser producer/consumer contract, source compatibility, and unsigned rehearsal. T-077, T-078, and T-080 must complete anonymous bootstrap, final notices and producer disposition, exact-ten signed `v0.69.0`/`v0.69.1` assets, publication, update, and rollback.
- **F-017-S004:** T-079 is pending fork-safe CI, DCO/CODEOWNERS, governance/support/reporting, GitHub security/settings, Pages/community surfaces, and disposable hostile-fork rehearsal.
- **F-017-S005:** T-080 and T-081 are pending final private two-release rehearsal, role sign-offs, separate visibility approval, logged-out lifecycle/fork smoke, recovery proof, 48-hour canary, and announcement.

No scenario is complete. Primary Status remains `primary_blocked`, repository visibility remains private, and no publication or announcement is authorized.

## Out of Scope

- Reintroducing a repository-embedded publication audit laboratory.
- Treating the private rewrite as secret, privacy, IP, runtime, or cutover clearance.
- Claiming physical deletion from hosting-provider storage or third-party caches.
- Changing visibility or announcing the project before T-081's separately approved cutover and clean canary.
- Publishing the launch releases before T-080 or treating `v0.69.0` as latest after `v0.69.1` exists.

## Descoped Scenarios

None.
