# F-017: Open-Source Publication

- Feature ID: F-017
- Goals: G-OSS-001, G-001, G-002, G-003, G-004
- Status: active
- Owner: foundation-maintainer

## Business Logic

MARS remains private until publication authority and every private cutover gate
are durable. The owner has attested authority to publish the current source,
documentation, release artifacts, retained history, first-party material, and
Cursor/automation-assisted material. The owner also accepts the recorded name
risk without trademark registration or counsel clearance. Those dispositions
complete the owner-only F-017-S001 holds but do not themselves complete
privacy, runtime, release, contribution, hosted-state, or cutover validation.

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
9. Make official source, setup, and release access anonymous; complete exact-tag
   bootstrap and the conventional AD-315 dormant producer/attestation contract
   through T-077 and T-078.
10. Through T-079, commit and configure private-safe contribution, governance,
    Issues, Discussions, Dependabot, and ruleset controls. Account-wide GitHub
    App administration is not a launch gate unless a MARS workflow actually
    depends on that App.
11. Complete hosted sanitation, future-only immutable Releases, and the
    no-publish workflow rehearsal while private. Then change visibility only
    through a separate approved owner action and immediately enable and verify
    the public-only security bundle.
12. Publish and prove attested `v0.69.0` as the rollback bridge and attested
    `v0.69.1` as latest through T-080. Run logged-out/fork smoke and wait for a
    clean 48-hour canary before the T-081 announcement.

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
And the named reachable child-environment and persisted-config credential exposures fail closed without overstating same-user containment; broader local-state, log, trace-export, and retention hardening is deferred post-launch.

### F-017-S003: Anonymous, Immutable, Verifiable Release Lifecycle

Given supported rollback-bridge and latest releases are built from their immutable release-note commits
When the pinned conventional workflow builds Darwin/Linux AMD64/ARM64 archives into fresh output directories
Then each archive contains the binary, license, notices, third-party notices, and deterministic source metadata
And SPDX SBOMs, SHA-256 checksums, and GitHub/Sigstore keyless provenance bind the published artifacts to the exact tag and commit
And an unpublished draft is fresh-downloaded and compared before publication
And missing, extra, duplicate, pending, mismatched, unsigned, unverifiable, canceled, or non-converging state never becomes a supported release
And signed `v0.69.0` is retained only as the rollback bridge while signed `v0.69.1` is latest
And anonymous install, update, rollback, immutable-release verification, and provenance verification pass independently across both releases.

### F-017-S004: Fork-Safe Contribution And Governance

Given an untrusted contributor opens a fork pull request
When repository automation runs
Then it receives no secrets, write token, release key, or self-hosted privileged runner
And it cannot mutate protected refs or publish releases
And required CI, DCO, review, governance, support, and conduct policies apply
And the public-repository-only CodeQL, secret-scanning, push-protection, private vulnerability-reporting, and Pages bundle passes a disposable public rehearsal before cutover and is enabled on the production repository immediately after separately approved visibility change.

### F-017-S005: Controlled Cutover And Canary

Given F-017-S001 through F-017-S004 pass with required owner, Security, QA, Dogfood, Release Manager, and Orchestrator sign-offs
When the owner separately approves public visibility
Then final refs, settings, artifacts, recovery procedures, and credentials are verified
And public-only CodeQL, secret scanning, push protection, private vulnerability reporting, and Pages are enabled and verified immediately
And logged-out clone, build, install, update, vulnerability reporting, and fork-PR smoke pass
And integrations are re-enabled least-to-most privileged
And announcement waits until the 48-hour canary remains clean.

## Launch Public Interfaces

- `setup` and official release access are anonymous-first; authentication is optional for rate limits, private forks, and custom repositories. Pending third-party downloads require one stable license/terms plan and explicit acknowledgement before requests or artifact writes.
- `auth github check` reports anonymous, authenticated, or unavailable without exposing credentials; `auth github clear-local` removes only MARS's legacy stored fallback.
- `run`, `start`, and `serve` default to observer execution. Host execution requires explicit acknowledgement; `isolated` fails closed as unavailable until an enforceable adapter exists.
- `shell_exec` is host-only, receives a sanitized environment, and reports the operating-system authority accurately.
- The repository filesystem paths named in T-075 are descriptor-relative and no-follow; broader trace export, purge, and retention work remains deferred.
- Telemetry collection and GitHub App setup bind only to literal loopback with the documented authentication, state, request, and timeout boundaries.
- The installer uses exact-tag Go/SumDB bootstrap into the signed updater; packaged operation remains Go-free.
- Each supported release has exactly four archives, four SPDX SBOMs, `checksums.txt`, and `checksums.txt.sigstore.json`.
- Contributions use fork PRs with DCO, CODEOWNERS, required fork-safe CI, and maintainer-only release authority.

## Evidence

### Owner And Scope Reconciliation — 2026-08-24

The owner accepted the unresolved `MARS` name risk without registration or
counsel clearance, supplied the publication-authority attestation recorded in
`docs/validation/reports/2026-08-24-owner-launch-dispositions.md`, funded the
GitHub account, and corrected account-wide GitHub App administration out of the
MARS launch scope. These decisions close the owner-only name/authority/funding
holds; they do not waive the remaining technical, hosted-state, release,
visibility, or canary gates. The owner also stopped the bespoke T-078 security
platform and approved AD-315's conventional release path.

- **F-017-S001:** The proposed private publication surface is reconstructed. On 2026-08-08, T-070 froze 302 canonical publication refs at `d04f642` and scanned all 12,002 reachable objects with both exact pinned tools in Git-history and raw-object lanes without accepted-scan errors or skips. Gitleaks group `a2a292e31d652f22` and URI group `e32927624f4a2cac` are resolved as synthetic test stubs. Direct Git-plumbing reconciliation proved all 36 occurrences in `f3dc0e336620abc6` are SHA-1 child blob IDs emitted by materialized Git tree objects, not credentials; every record matched its source tree, reachable child blob, and raw scanner field with zero mismatches. T-070 passes with zero unresolved findings. T-072 then reconciled 305 hosted refs, 57 Release objects, 500 assets, five addressable workflows, 401 completed runs, 77 deployments, and every access/content/settings surface. Packages, linked projects, Actions artifacts/caches, and Wiki pages are confirmed empty; Pages and Discussions are not applicable. Exact payloads and logs were acquired, admitted offline scanners completed with zero error, skip, or unresolved secret candidate, and exact cleanup ID sets are sealed. Owner-authenticated UI evidence found two all-repository write-capable Apps; opaque finding `T072-6a87eaed-e746-41c1-bf0e-1e519fc66705` is an explicit T-079/T-080 pre-cutover scope-reduction/removal no-go. T-072 passes without hosted mutation. T-073 checkpoint A froze the exact prompt, tool/automation-attribution, and media inventories. The eleven unsupported symbolic prompt headers are corrected at `a8d448f` with verified introduction/comparison facts and explicit pending disposition; current `main` removes the sole PNG and both live references at `12faa47`, while retained-history rights remain open. Browser-asset and llama.cpp provenance are bound at `c7168c5` and `f3df0a5`; product claims are corrected at `2ffde82`; deterministic final dependency notices are enforced at `dc0dbe0` with SHA-256 `d18a021e0d32c342d733f1c3e59ad72da8893bbbc41ae5dda6dbcca980631739` and green run `31288019067`. Commits `cf95b39` and `b8d9349` bind all six unique default GGUFs to exact artifact identities and truthful publisher/base/license/terms/quantizer/tool facts, and reject incomplete records before model-download-step mutation or a model-download network request. Unpublished conversion-input revisions are recorded as unavailable rather than inferred because MARS links to rather than redistributes the weights. Exact run `31289522986` passes every source-compatibility lane at `b8d9349` without changing model bytes or routing behavior. The official-register screen found live U.S. registration 8092258 directly overlapping AI-agent/process-automation services under the exact `MARS` mark. On 2026-08-09 the owner directed that `MARS` be retained; name clearance remains a no-go without qualified trademark counsel's written disposition for the intended use and launch territories. Owner authority over retained first-party, Cursor/automation, predecessor, and historical PNG material also remains unsigned. Local stashes, administrative refs, unreachable objects, and the unrelated Codex worktree remain preserved non-publication exceptions.
- **F-017-S002 — passed 2026-08-09:** T-071 passed at exact commit `59ab946` and GitHub run `31278506189`: both supported Go lanes and the expected below-minimum rejection pass, while `govulncheck v1.6.0` reports zero called application vulnerabilities and GO-2026-6061 is absent. T-074 passed through exact commits `596524e` and `f77fac6`; T-075 passed through the exact A-D chain recorded below; and T-076 passed through exact commits `9191182`, `5c23f53`, `473b829`, `9eb3f96`, and `31b00b1` under the narrowed owner governor. Resumed T-058 then passed at exact pushed commit `57d7851d9c82975256761b0134d20e91382e9bcd`: the installed in-app browser rendered hostile runtime data only as text, observed eight assets from the exact loopback origin with zero external origins or console warnings/errors, and cleared privileged DOM on corrected logout. The exact candidate SHA-256 was `ef5258e3b135c1e03a53635655b565c0e069fd16a3ca67af7631c76f9fa9e2bc`. F-010-S024 also passes. Broader local-state, log, trace-export, purge, and retention hardening remains deferred and is not claimed by this scenario.
  T-076 Checkpoint A passed at exact pushed commit `9191182601d79b996f1848a1e867e50b7d6eaf1c`: `run`, `start`, `serve`, `tools run`, and `mcp serve` default to an independent observer ceiling; acknowledged host authority does not upgrade configured role trust; unsupported isolation fails before effects; and observer suppresses the reachable direct target-writer paths while retaining owner-local bookkeeping. QA caught a final manual-run hard-coded-contributor regression, which was corrected before push by parsing manifest role trust with an observer fallback; the live-command regression proves the observer mutator rejection reaches the model and no requested target file is created. Exact gates passed: `go test ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars ./internal/scanner ./internal/docsconsistency ./internal/docsync`; `go test -race ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars`; `go vet ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars ./internal/scanner`; `go run ./cmd/mars docsync audit --repo .` with 362 files and zero findings; and `git diff --check`. The final trust delta also passed its exact targeted test plus full `go test ./cmd/mars`, `go test -race ./cmd/mars`, `go vet ./cmd/mars`, and `git diff --check`. QA, Security, Release Manager, and Orchestrator returned GO.
  T-076 Checkpoint B1 passed at exact pushed commit `5c23f536fadd9ab18694e3f46ed9b10ca96594da`: one thin child-environment seam preserves ordinary non-sensitive PATH, HOME, temporary-directory, locale, and toolchain/cache state while filtering credential-like, MARS, GitHub, cloud/provider, delimiter-bounded auth, SSH, token, secret, password, API-key, private-key, and credential names. Parent-only `MARS_CHILD_ENV_ALLOWLIST` restores exact names but never reaches a child or becomes repository/model configuration. Every named shell, MARS CLI, dependency, Git, code-intelligence, managed-inference, MCP stdio, Jira, GitHub-auth, and source-update subprocess receives an explicit environment; Jira requires both the repository passthrough request and owner allowlist. Exact gates passed: `go test ./internal/childenv ./internal/tools ./internal/inference ./internal/serve ./internal/codeintel ./internal/githubauth ./internal/selfupdate ./internal/mcpclient ./internal/jira ./cmd/mars -count=1`; `GOCACHE=/private/tmp/mars-t076b1-root-race-cache go test -race ./internal/childenv ./internal/tools ./internal/inference ./internal/jira -run 'Test(Filter|ApplyWith|ShellExecSanitizes|DependencySyncChild|ServerManagedChild|ProxyEnvironment)' -count=1`; `GOCACHE=/private/tmp/mars-t076b1-root-cache go vet ./internal/childenv ./internal/tools ./internal/inference ./internal/serve ./internal/codeintel ./internal/githubauth ./internal/selfupdate ./internal/mcpclient ./internal/jira ./cmd/mars`; `go test ./internal/docsconsistency ./internal/docsync -count=1`; `go run ./cmd/mars docsync audit --repo .` with 364 files and zero findings; and `git diff --check`. Real-child regressions cover foreground/background shell, dependency PATH/cache preservation, managed inference, and the Jira dual gate. QA, Security, Release Manager, and Orchestrator returned GO. This is name-based exposure reduction, not containment.
  T-076 Checkpoint B2 passed at exact pushed commit `473b829efe865630f4942b55af9e0108d7529d0c`: background records require a job ID; listing, direct helper policy, kill, and cleanup are scoped by that ID; unowned and cross-job targets fail closed; and cleanup sends TERM to the owned process group, waits at most two seconds, then sends KILL if the group remains. Server jobs, manual run, one-shot tools run, and MCP EOF/error clean only their groups. Serve no longer performs blind lsof/pgrep cleanup, and representative regressions prove a foreign listener and unrelated llama-server-named process survive. Exact gates passed: `go test ./internal/tools ./internal/mcpstdio ./internal/serve ./internal/scanner ./cmd/mars`; `go test -race ./internal/tools ./internal/mcpstdio ./internal/serve ./cmd/mars`; `go vet ./internal/tools ./internal/mcpstdio ./internal/serve ./internal/scanner ./cmd/mars`; `go test ./internal/docsync ./internal/docsconsistency`; `go run ./cmd/mars docsync audit --repo .` with 364 files and zero findings; and `git diff --check`. QA, Security, and Orchestrator returned GO. The narrowed config-secret correction passed at 9eb3f96 and the live shared-database path truth correction passed at 31b00b1, so T-076 is complete under the owner governor; broader local-state hardening is deferred. Resumed T-058 passed at `57d7851`, completing F-017-S002. The repository remains private at `VERSION=0.68.49` with Primary Status `primary_blocked`; legal/rights and installed-App no-gos remain, and no Release, settings, visibility, signing, publication, or announcement authority changed.
- **F-017-S003:** T-064 through T-068 completed the private v0.93 retirement, GoReleaser producer/consumer contract, source compatibility, and unsigned rehearsal. T-077 Checkpoint A passed at exact pushed commit `10b62f7d59620022b2e1030c5f33856d0c16e70f` in a bounded 33-path packet: ordinary setup performs no auth step; the exact no-redirect official metadata probe is anonymous first; only an exact `401`, `403`, or `404` can resolve optional credentials for one same-origin/path retry; `auth github clear-local` removes only the stored config fallback; and doctor preserves its selected custom `ConfigPath`. Affected normal/race tests, vet, docs consistency, DocSync across 364 files with zero findings, formatting, and diff checks passed with QA, Security, and Orchestrator GO. Checkpoint B then passed at exact pushed commit `04d6ba6844126dc84eb6bedc13c78bd31f8d371d` in a bounded 21-file packet: setup stable-sorts one unique pending-artifact plan with the concrete bundle, immutable identities, exact sizes, license IDs/URLs, and applicable terms/notice URLs; interactive setup confirms once, while non-TTY/JSON requires exact `--download --yes` and emits the complete JSON plan before requests. The accepted plan and downloader identities are rechecked so decline, missing acknowledgement, incomplete provenance, or changed state causes zero download requests and zero download-artifact writes. Skip, test, deferred, and cloud paths need no acknowledgement; no attestation is persisted; automatic Linux llama.cpp acquisition stays disabled. Affected normal/race tests, vet, docs consistency, DocSync across 366 files with zero findings, formatting, and diff checks passed with QA, Security, and Orchestrator GO. Checkpoint C implementation passed at exact pushed commit `85c689c70ef801a2747acabf537739c9ebad3c12` in a bounded 13-file packet: direct privileged Bash startup rejects inherited functions and `BASH_ENV`; optional GitHub tokens stay outside `env(1)` arguments and Go and are scoped only to the staged updater after their transport descriptors close; the exact Go build uses the public proxy/SumDB, private trusted staging, and `-modcacherw`; and runtime `BuildInfo` must prove the canonical command/module, exact tag, canonical SHA-256 `h1`, and no replacements before the existing signed updater may acquire or replace anything. Pre-commit failures preserve the prior binary; recovery-required results retain transaction evidence and report truthful remediation. Normal/race, vet, hostile installer, Bash 3.2 descriptor-close, CLI/help mirror, docs, DocSync 366/0, and diff gates passed with QA, Security, Release Manager, and Orchestrator GO. Exact clean pushed source `56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b` then passed four Go 1.26.5 CGO-disabled builds and clean credential-free macOS arm64 plus native Linux arm64 non-root source/setup lanes. The Linux installer suite passed under GNU `stat`; setup ran twice at `4/0` then `0/4` steps with empty download plans and truthful disabled Linux acquisition. QA and Security independently recomputed the retained binary, BuildInfo, JSON, config, and postconditions; all four temporary roots were removed and verified absent. The exact evidence is `docs/validation/reports/2026-08-24-t077-bootstrap-setup-closure.md`. T-077 passes its private boundary. T-078 then stopped before producer execution: exact SumDB GoReleaser v2.17.1 built with Go 1.26.5 reported 12 called vulnerability IDs and 104 terminal called symbols. Separately, T-072's 401-run seal was reproduced and the exact 65-run delta frozen at 466 completed runs was acquired and scan-clean, but every later run remains in scope. The exact blocked-admission evidence is `docs/validation/reports/2026-08-24-t078-release-production-admission-blocked.md`. Real official tags, Releases, signing, logged-out downloads, update, and rollback remain T-080/T-081 work. Hosted workflow proof is externally blocked on GitHub Billing & plans, while deletion remains separately approval-gated. The repository remains private at `VERSION=0.68.49` with Primary Status `primary_blocked`; T-073 legal/rights disposition and the installed-App findings remain launch no-gos, and no Release, settings, visibility, signing, publication, or announcement authority changed.
- **F-017-S004:** T-079 is pending community files, fork-safe CI, DCO/CODEOWNERS, governance/support, private-safe GitHub settings, rulesets, Issues/Discussions, and a disposable public hostile-fork rehearsal. Account-wide GitHub App administration is excluded unless a MARS workflow has an actual dependency. Because this private personal repository cannot production-enable CodeQL/code scanning, secret scanning/push protection, private vulnerability reporting, or Pages without unavailability or premature publication, T-079 stages that bundle for immediate post-visibility enablement and verification.
- **F-017-S005:** T-080 and T-081 are pending final private two-release rehearsal, role sign-offs, separate visibility approval, logged-out lifecycle/fork smoke, recovery proof, 48-hour canary, and announcement.

F-017-S001 and F-017-S002 are complete. F-017-S003 through F-017-S005 remain incomplete. Primary Status remains `primary_blocked`, repository visibility remains private, and no publication or announcement is authorized.

## Out of Scope

- Reintroducing a repository-embedded publication audit laboratory.
- Treating the private rewrite as secret, privacy, IP, runtime, or cutover clearance.
- Claiming physical deletion from hosting-provider storage or third-party caches.
- Changing visibility or announcing the project before T-081's separately approved cutover and clean canary.
- Publishing the launch releases before T-080 or treating `v0.69.0` as latest after `v0.69.1` exists.

## Descoped Scenarios

None.
