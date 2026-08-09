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

F-017-S001 through F-017-S004 may receive bounded private technical work
independently. An unresolved S001 owner/legal hold never counts as scenario
completion, and T-080/T-081 cannot begin until S001 passes together with the
other required scenarios.

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

- **F-017-S001:** The proposed private publication surface is reconstructed. On 2026-08-08, T-070 froze 302 canonical publication refs at `d04f642` and scanned all 12,002 reachable objects with both exact pinned tools in Git-history and raw-object lanes without accepted-scan errors or skips. Gitleaks group `a2a292e31d652f22` and URI group `e32927624f4a2cac` are resolved as synthetic test stubs. Direct Git-plumbing reconciliation proved all 36 occurrences in `f3dc0e336620abc6` are SHA-1 child blob IDs emitted by materialized Git tree objects, not credentials; every record matched its source tree, reachable child blob, and raw scanner field with zero mismatches. T-070 passes with zero unresolved findings. T-072 then reconciled 305 hosted refs, 57 Release objects, 500 assets, five addressable workflows, 401 completed runs, 77 deployments, and every access/content/settings surface. Packages, linked projects, Actions artifacts/caches, and Wiki pages are confirmed empty; Pages and Discussions are not applicable. Exact payloads and logs were acquired, admitted offline scanners completed with zero error, skip, or unresolved secret candidate, and exact cleanup ID sets are sealed. Owner-authenticated UI evidence found two all-repository write-capable Apps; opaque finding `T072-6a87eaed-e746-41c1-bf0e-1e519fc66705` is an explicit T-079/T-080 pre-cutover scope-reduction/removal no-go. T-072 passes without hosted mutation. T-073 checkpoint A froze the exact prompt, tool/automation-attribution, and media inventories. The eleven unsupported symbolic prompt headers are corrected at `a8d448f` with verified introduction/comparison facts and explicit pending disposition; current `main` removes the sole PNG and both live references at `12faa47`, while retained-history rights remain open. Browser-asset and llama.cpp provenance are bound at `c7168c5` and `f3df0a5`; product claims are corrected at `2ffde82`; deterministic final dependency notices are enforced at `dc0dbe0` with SHA-256 `d18a021e0d32c342d733f1c3e59ad72da8893bbbc41ae5dda6dbcca980631739` and green run `31288019067`. Commits `cf95b39` and `b8d9349` bind all six unique default GGUFs to exact artifact identities and truthful publisher/base/license/terms/quantizer/tool facts, and reject incomplete records before model-download-step mutation or a model-download network request. Unpublished conversion-input revisions are recorded as unavailable rather than inferred because MARS links to rather than redistributes the weights. Exact run `31289522986` passes every source-compatibility lane at `b8d9349` without changing model bytes or routing behavior. The official-register screen found live U.S. registration 8092258 directly overlapping AI-agent/process-automation services under the exact `MARS` mark. On 2026-08-09 the owner directed that `MARS` be retained; name clearance remains a no-go without qualified trademark counsel's written disposition for the intended use and launch territories. Owner authority over retained first-party, Cursor/automation, predecessor, and historical PNG material also remains unsigned. Local stashes, administrative refs, unreachable objects, and the unrelated Codex worktree remain preserved non-publication exceptions.
- **F-017-S002:** T-071 passed at exact commit `59ab946` and GitHub run `31278506189`: both supported Go lanes and the expected below-minimum rejection pass, while `govulncheck v1.6.0` reports zero called application vulnerabilities and GO-2026-6061 is absent. Existing loopback, webhook, and dashboard hardening remains supporting evidence; T-075 is current, while T-076 and resumed T-058 still own the later execution/browser completion gates. T-074 passed through exact commits `596524e` and `f77fac6`: telemetry validates literal loopback before database creation and accepts only bounded fail-closed requests, while the source-only GitHub manifest flow consumes one cryptographic state before one bounded exchange and returns no secret credentials. T-075 Checkpoint A passed at exact commit `f9993b5`: the retained standard-library repository descriptor remains bound across root-path replacement, observed symlink parents/leaves are rejected, direct file-tool writes are atomic, and focused normal/race, vet, coverage, documentation-consistency, and DocSync gates pass with QA/Security/Release Manager GO. Checkpoint B passed at exact commit `b3b5b98`: shared CLI/tool-policy scanning reads raw stage-0 blobs with replacement objects disabled, reconciles rename/deletion states, includes tracked/force-added local credentials and Git-hidden worktree entries, rejects nested worktree roots, and passed focused normal/race, vet, documentation, staged, and real full-repository scans with zero findings and QA/Security/Release Manager GO. Checkpoint C1 passed through exact commits `88f7737` and `e30f207`: the first finite repository-writer family uses exclusive creation, atomic mode-preserving replacement, retained descriptor-backed learnings persistence, and pre/post Git repository-identity checks. Focused normal/race tests, package vet, the exact 26-case Git-admission fixture regression, formatting, and diff checks pass with QA/Security/Release Manager/Orchestrator GO. Target lifecycle, the remaining named repository-path migrations, Checkpoint D, T-075, and F-017-S002 remain incomplete.
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
