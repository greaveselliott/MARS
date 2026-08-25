# Active P0 Exec Plan: Launch MARS As A Supported Open-Source Project

**Status:** Active
**Priority:** P0
**Depends On:** T-070 and F-018-S001 through F-018-S003 complete
**Blocks:** public visibility, supported v0.69.0/v0.69.1 releases, announcement, and G-OSS-001 completion
**Related Tickets:** T-058 and T-071 through T-081
**Current Ticket:** T-079 — final contribution-control evidence and hosted verification
**Goals:** G-OSS-001, G-001, G-002, G-003, G-004
**BDD Feature:** F-017-open-source-publication.md
**Related Feature Contracts:** F-001, F-010, F-018
**Hypothesis:** A conventional least-privilege producer and GitHub's standard attestation path can complete the launch faster and more safely than the preserved bespoke T-078 platform, provided every hosted mutation and visibility change remains separately approved.
**Success Evidence:** F-017-S001 through F-017-S005 pass and the Primary Pass Gate below is durably evidenced.
**Falsification Evidence:** Any unresolved launch no-go reaches visibility, a supported release, or announcement.
**Scenario Schedule:** T-071 through T-077 complete; T-073 owner holds resolved by recorded disposition; T-078; T-079; T-080 public cutover and attested launch releases; T-081 canary and announcement.
**Current Failing Scenario:** F-017-S004 — T-079's private source and hosted settings are installed: community files, DCO/CODEOWNERS, fork-safe CI, Dependabot, Discussions, and exact ruleset `21491158` pass their private boundary without changing visibility or enabling public-only controls. Final evidence-source hosted verification is pending. The public-only security bundle and genuine hostile-fork smoke remain T-080 work after visibility, while T-078's exact cleanup/future-only immutable-Release transaction remains separately approval-gated.
**Walking Skeleton Slice:** T-078 implements only the standard Go/archive/Syft/GitHub-attestation path, proves the exact-ten contract without publication, refreshes the hosted-state seal, and prepares or performs only separately approved exact sanitation. T-079 closes private contribution controls. T-080 changes visibility under separate approval before creating the two public attested launch releases; T-081 owns the canary and announcement.
**Learning Or MVP Outcome:** Keep build, attestation, publication, hosted sanitation, visibility, and announcement as distinct authority boundaries while avoiding bespoke infrastructure that is not required for launch.
**Owner:** foundation-maintainer as Orchestrator with COO, CTO-weekly, Engineer, QA, Security, Dogfood, Release Manager, and repository owner

## Primary Outcome Contract

- **Primary Outcome:** Publish MARS as a supported open-source project without exposing confidential material, weakening controls, or distributing unsafe or unverifiable binaries.
- **Primary Pass Gate:** The repository is public; attested `v0.69.1` is the supported release with attested `v0.69.0` retained only as its rollback bridge; F-017-S001 through F-017-S005 pass; logged-out macOS/Linux clone, build, bootstrap, setup, update, and rollback pass; fork-contribution controls pass; GitHub security and community surfaces are active; a 48-hour public canary is clean; and the launch announcement is posted.
- **Primary Status:** `primary_blocked`
- **Current Primary Blocker:** T-079's final evidence-source hosted run is pending. The separately approval-gated T-078 cleanup remains parked; public-only controls, visibility, launch attestations, releases, and announcement remain later gates.
- **Next Primary Action:** Push and verify T-079's exact live-contract/evidence closure, then close T-079. Return to the separately approval-gated T-078 hosted cleanup before T-080; do not change visibility, Releases, tags, Pages, Apps, or publication state.

## Starting Baseline

- Clean private `main` and `origin/main` were equal at `d2db7c522795fa2698421434f1d9d4ebd2ec3f02` when this plan activated.
- `VERSION=0.68.49`; source fallback is `0.69.0-dev`.
- T-070 passed the advertised-Git audit with zero unresolved findings. That evidence does not cover retained GitHub-hosted content or complete F-017-S001.
- F-018-S001 through F-018-S003 passed as private producer, consumer, and rehearsal evidence only.
- Reconciled T-072 closure: 301 tags, 57 Release objects, 500 legacy assets, 401 then-completed workflow runs, 77 deployments, one collaborator, zero packages, zero linked projects, an uninitialized zero-page Wiki, and no current Actions artifacts or caches. T-078's current read-only acquisition on 2026-08-24 now freezes 473 completed and zero active runs: the oldest 401 and next 65 exactly reproduce their prior seals, while the seven-run delta is acquired and scan-clean with zero authoritative scanner errors or unresolved candidates. Current state is 301 tags, the intended 56 Releases, 500 assets, 77 deployments, and zero Actions artifacts/caches; exact deletion/setting manifests are separately approval-gated. Account-wide GitHub App administration is out of launch scope unless a MARS workflow actually depends on an App.
- Rulesets, branch protection, CodeQL, Dependabot alerts, secret scanning, and push protection are not yet enabled.

## Assumption Confidence Matrix

| Assumption | Evidence | Confidence | Validation Required |
|---|---|---:|---|
| The launch plan started from clean synchronized source at `d2db7c…` | Local HEAD and `origin/main` matched; worktree was clean | 1.0 | Complete — historical activation fact |
| T-070 clears the advertised Git surface | 12,002 reachable objects, four scanner lanes, zero errors, skips, or unresolved findings | 1.0 | Complete — T-072 reconciled publication refs |
| gRPC v1.82.1 closes the called advisory | Source selects v1.82.1; local gates and exact run `31278506189` report zero called vulnerabilities | 1.0 | Complete — T-071 |
| Current hosted-surface counts remain stable | T-072 acquisition, UI confirmation, and post-evidence run delta reconcile with zero scan error or unresolved secret candidate | 1.0 | Complete — T-072 |
| Elliott's authority can be converted into a complete publication attestation | Exact owner statement is recorded in the 2026-08-24 disposition report | 1.0 | Complete — owner authority disposition |
| The unresolved `MARS` name conflict can be accepted for launch | Owner explicitly accepted the recorded risk and declined registration/counsel clearance | 1.0 | Complete as an owner risk disposition, not a clearance finding |
| No reachable runtime P0/P1 remains after scheduled hardening | T-074 through T-076 and the exact-current T-058 browser replay pass | 1.0 | Complete — F-017-S002 |
| Exact-tag Go/SumDB bootstrap is non-circular on clean macOS/Linux | T-077 implementation, hostile fixtures, four exact builds, and clean credential-free macOS/Linux source/setup pass through `56b8de3` | 1.0 | Complete for private source/setup; real signed official tags remain T-080/T-081 |
| AD-315 can produce and attest the exact ten-asset release without bespoke infrastructure | Exact source `d411cbe` passed two clean offline producer runs, independent nine-file verification, normalized SBOM comparison, and the standard public attestation fixture; a real public OIDC attestation remains future evidence | 0.90 | T-078 hosted CI and T-080 public attestation/publication |
| Required GitHub controls can be configured with owner authority | Owner has repository administration; controls were disabled on 2026-08-08 | 0.80 | T-079 disposable rehearsal and private configuration |
| `v0.69.0` and `v0.69.1` remain available for immutable release commits | As of 2026-08-08, no launch tags used these names | 0.95 | T-080 exact pre-tag remote recheck |
| The public canary will remain incident-free | Future operational result | 0.50 | T-081 continuous 48-hour observation and final replay |

## Launch Version Freeze

**Launch transition exception:** T-071 through T-079, including any semantic
correction required by resumed T-058, retain `VERSION=0.68.49` and
`DefaultVersion=0.69.0-dev`. Validated checkpoints are committed and pushed
without release-note generation, tag creation, signing, upload, publication,
or announcement. After the separately approved public-visibility change,
T-080 ends the pre-release freeze by generating `0.69.0`, publishing its
attested rollback bridge, then generating and publishing attested `0.69.1` as
the supported release. T-081 evidence-only canary closeout
commits retain `0.69.1` without creating `0.69.2`; any product, runtime,
security, or public-contract correction discovered during canary requires
immutable `v0.69.2` and a repeated lifecycle and canary.

## Sequential Ticket Schedule

Only one implementation ticket is current. T-073's machine checkpoints were
already complete and the owner resolved its name-risk and publication-authority
holds on 2026-08-24, so the ticket is closed without presenting the risk
acceptance as trademark clearance. T-078 through T-081 proceed sequentially,
one current ticket at a time. T-058 was resumed rather than recreated.

| Order | Ticket | Scenario ownership | Dependency |
|---:|---|---|---|
| 1 | T-071 — restore green vulnerability baseline | F-017-S002 prerequisite | T-070 |
| 2 | T-072 — audit GitHub-hosted publication surfaces | F-017-S001 | T-071 |
| 3 | T-073 — owner disposition and machine evidence complete | F-017-S001 | T-072 |
| 4 | T-074 — close remaining network entry points | F-017-S002 | T-072 and T-073 machine checkpoints |
| 5 | T-075 — descriptor-safe repository filesystem and secret scanning | F-017-S002 | T-074 |
| 6 | T-076 — execution profiles, environment, state, and traces | F-017-S002 | T-075 |
| 7 | Resume T-058 — passed 2026-08-09 | F-017-S002 and F-010-S024 | T-076 |
| 8 | T-077 — anonymous access, trusted bootstrap, and setup | F-017-S003 | T-058 |
| 9 | T-078 — standard release path and legacy-asset sanitation | F-017-S003 and F-018-S004 | T-077 |
| 10 | T-079 — community, fork-safe CI, Pages, and GitHub controls | F-017-S004 | T-078 source/rehearsal/hosted-CI checkpoints; separately approved T-078 sanitation may remain parked |
| 11 | T-080 — approved public cutover and attested launch releases | F-017-S003 through F-017-S005 prerequisite | T-079 and explicit owner visibility approval |
| 12 | T-081 — 48-hour canary and announcement | F-017-S005 | T-080 |

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

## Completed Walking Skeleton — T-072

T-072 froze repository-hosted writes, collected every applicable Release,
Actions, deployment, Pages/Wiki, issue/discussion, attachment, access,
integration, package, and security metadata surface with standard tools,
scanned content offline, and reduced only redacted coverage and exact
immutable-ID cleanup manifests.

Snapshot `T072-723d689d-3194-40c8-9d92-322b921149a3` completed the immutable
inventory, payload acquisition, offline scan, finding classification, and exact
cleanup-ID digest checkpoints for every API-accessible surface. The Release
discrepancy reconciles to 57 objects and 500 assets. Scanner errors, skips, and
unresolved candidates are zero. On 2026-08-08, closure required one consolidated
owner-authenticated read-only confirmation of Apps, Packages, Projects v2, and
Wiki state. That confirmation found two write-capable all-repository Apps, zero
packages, zero projects, and a zero-page Wiki. The evidence commit's one new
successful workflow run was acquired and rescanned with zero findings or
errors, producing a final exact 401-run cleanup set. T-072 passes; the later
owner correction removes account-wide App scope from the MARS launch, and no
hosted mutation was allowed.

## Completed Walking Skeleton — T-073

T-073 was created through `ticket_create` after bounded COO, CTO-weekly, and
Security scope review. Complete its four independently reviewed checkpoints:
authority/name/predecessor/AI/media evidence; machine-checked model and
llama.cpp provenance; deterministic dependency notices and accurate product
claims; then the owner-attested `preserve_audited_history` disposition with no
deferred rights finding.

Checkpoint A's machine inventory is recorded at publication root `1b870f8`.
The unsupported symbolic prompt-source headers were replaced at `a8d448f`
with verified introduction/comparison facts and explicit pending owner
disposition; the current PNG and both live references were replaced at
`12faa47` with browser-verified semantic HTML/CSS. These current-tree fixes do
not establish rights over the retained prompts, AI/automation contributions,
predecessor material, or historical PNG blob. The materially overlapping live
U.S. `MARS` registration also remains. Checkpoint C's dependency notices and
product claims are complete through `dc0dbe0` and exact run `31288019067`;
checkpoint B is complete. Commits `cf95b39` and `b8d9349` bind all six unique
default GGUFs to exact artifact commits, filenames, sizes, hashes, declared
publisher/base/license/terms/quantizer/tool facts, and reject incomplete
records before model-download-step mutation or a model-download network
request. Unpublished conversion-input
revisions are recorded as unavailable and never inferred. Exact GitHub run
`31289522986` passes both supported Go lanes, the below-minimum rejection, and
the dependency-notice job at `b8d9349`; no model bytes or routing behavior
changed. Those historical facts remained visible for the owner's final
disposition.

All machine-verifiable T-073 checkpoints are complete at `5068334`. T-073 is
complete after the 2026-08-24 owner disposition accepted the unresolved name
risk without registration/counsel clearance and attested publication authority
over the current repository and retained material. The exact statement is in
`docs/validation/reports/2026-08-24-owner-launch-dispositions.md`. This closes
F-017-S001 as an owner disposition, not as a trademark-clearance finding, and
grants no tag, attestation, upload, Release, visibility, or announcement
authority.

## Completed Walking Skeleton — T-074

T-074 was created through `ticket_create` after bounded COO, CTO-weekly, and
Security review. Checkpoint A passed at exact commit `596524e`: the directly
reachable telemetry collector is literal-loopback-only, rejects invalid binds
before database creation, bounds request admission, and passed focused
normal/race, four-platform build, and clean commit-bound installed smoke gates.
Checkpoint B passed at exact commit `f77fac6`: the source-only GitHub manifest
flow, which has no production callsite, uses literal-loopback binding,
cryptographic one-time state, single-use callback admission, bounded exchange
and server behavior, owner-only credential persistence, and fixed redacted
results without wiring a new CLI command. Focused normal/race tests, vet,
docs-consistency, DocSync, and four CGO-disabled builds passed. QA, Security,
Release Manager, and Orchestrator approved the frozen checkpoint.

Remote telemetry, a generalized HTTP framework, dashboard/webhook changes, and
all T-075 through T-077 surfaces remained out of scope. T-074 is complete but
does not close F-017-S002 by itself, and it grants no release, settings,
visibility, or publication authority.

## Completed Walking Skeleton — T-075

T-075 was created through `ticket_create` after bounded COO, CTO-weekly, QA,
and Security scope review. Two reachable gaps define the ticket: universal
file operations use lexical path checks before ordinary filesystem calls, so a
hostile repository symlink can redirect reads or writes outside the selected
repo; staged secret scanning enumerates index names but reads worktree bytes,
so partially staged, index-only, renamed, tracked, or force-added secret bytes
can pass falsely.

Checkpoint A uses Go's standard-library `os.Root` as the descriptor-bound
containment primitive and migrates only the direct file tools. Checkpoint B
scans exact stage-0 Git index blobs and omits `.harness/.env.local` only when it
is genuinely untracked and ignored. Checkpoint C mechanically migrates the
finite named ticket/tool/persona, target lifecycle, credential/model, release,
and Jira mutation/writer surfaces in independently green commits. Checkpoint D
inventories and migrates the deferred bundle/context, tools-policy, scanner,
release, and Jira general read-side surfaces, then records proportionate tests,
four-platform builds, installed clean-target smoke, and bounded role sign-off.

Checkpoint A passed at exact commit
`f9993b5941e2fcd4f8e77866526f8a9b81946d3f`. The implementation retains the
admitted descriptor even when the original repository pathname is replaced,
rejects observed symlink parents/leaves, installs `file_write` replacements
atomically, and routes `file_read`, `file_write`, `file_search`, and `grep`
through that boundary. Focused normal/race tests, vet, the 70% coverage floor,
documentation consistency, and DocSync pass; QA, Security, Release Manager,
and Orchestrator returned GO.

Checkpoint B passed at exact commit
`b3b5b9808e001491da793e515cb71444655dbf22`. One shared scanner reads raw
stage-0 object bytes with replace objects disabled, reconciles rename and
deletion states, scans tracked/force-added local credentials and Git-hidden
worktree entries, and rejects a nested worktree root. Focused normal/race,
vet, documentation-consistency, DocSync, staged, and real full-repository
scans pass with candidate- and OID-free results; QA, Security, Release Manager,
and Orchestrator returned GO.

Checkpoint C sub-checkpoint 1 passed through exact commits
`88f7737bf9be3b804483f676507a193f68ffa7d4` and
`e30f207c9edc00a42a28b4d31b9ea5e52dba8a08`. At those commits, the finite
ticket, tool, persona, workspace-hygiene, record-decision, and learnings writer family
uses the admitted descriptor, exclusive creation, atomic mode-preserving
replacement, and pre/post Git repository-identity checks. Focused normal/race
tests, package vet, the exact 26-case Git-admission fixture regression,
formatting, and diff checks pass; QA, Security, Release Manager, and
Orchestrator returned GO.

Checkpoint C sub-checkpoint 2 passed through exact commits
`66d7e412f0c0dade49c037752c8fa3f0000ee94e` and
`c8c28cbcc709e12554236e92b7c2e7ba19006784`. Init, upgrade, generated-target
mutation, metadata, workspace-ignore, and eject paths retain the admitted
descriptor. Git initialization verifies repository identity before and after
execution; eject preflights every target before descriptor-relative removal
and rejects symlink parents/leaves without touching outside sentinels. Focused
normal/race tests, scanner vet, formatting, and diff checks pass; QA, Security,
Release Manager, and Orchestrator returned GO. This was sub-checkpoint 2
evidence only.

Checkpoint C sub-checkpoint 3 passed through exact commits
`d67b04278db608c5fb39d61d3fa0b54c4909cbed`,
`f99964e79047b3e71d3076d1a05c75b3df9c4e95`, and
`e08deb4bd118ff025abf131e7db8cf4eeb4cf333`. Model overrides and local
credential fallback are descriptor-bound, distinguish read failure from
missing credentials, enforce owner-only local credential mode, and scrub
example values. Release notes/backfill retain one descriptor for controlled
release files with mode-preserving atomic replacement. Jira ticket creation
is exclusive and reconciliation is atomic and mode-preserving. Focused
normal/race tests, affected caller tests, package vet, formatting, and diff
checks pass; QA, Security, Release Manager, and Orchestrator returned GO.
Checkpoint C's named mutation/writer portion is complete. Checkpoint D's
deferred read-side inventory passed through exact commits
`228d859511fb2f7c93e0162424c2e6dc95107e44`,
`7578549bdd0dde90857f9652e651832d484abdb2`,
`ff69aaa1bab3680d169e8889866ba73cccb397c9`,
`16b5527bbe48e8afea82bb70127d383d6f280ed7`, and
`c18030edb44d1b869a03d30e4339ff457641c6e4`; final DocSync containment passed
at `9ba8156942a584f301888a3675942923739993d6`. Focused normal/race tests,
affected-package vet, full DocSync/docs-consistency, four CGO-disabled builds,
and installed Dogfood passed. The four build SHA-256 values are
`dedcecb5e05416fdb6614e7c9d8010f446a51ff3e4fbafe2429435fc46bf4ed0`,
`c1137731531fded59e600e36ba8f77cd7ef1d6759262ddc223a3b7235831a28f`,
`5113f1b119a35c46a90fbb93a28877c61213527ee163319960961717ac7d290c`, and
`844cd84754cc55ffd426a657549b104a89ce2b02a8dbab5ab47b739379bace06`.
T-075 passes with QA, Security, Dogfood, Release Manager, and Orchestrator GO;
T-076 later passed as recorded below, and resumed T-058 subsequently passed at
`57d7851`, so F-017-S002 is complete.

No custom `openat` framework, scanner runtime, VM/container, kernel/race lab,
arbitrary shell containment, global-state permission work, version, Release,
settings, visibility, or publication mutation is authorized. T-075 cannot
close F-017-S002 by itself.

## Completed Walking Skeleton — T-076 And Resumed T-058

T-076 was created through `ticket_create` after bounded CTO-weekly, QA, and
Security review at clean synchronized `c04a172`. Four conventional checkpoints
own the remaining execution boundary: explicit execution-profile admission;
sanitized child environments and job-owned process cleanup; owner-only state
and centralized display/persistence/export redaction; then redacted trace
export, dry-run-first body purge, a hard 30-day full-body maximum, indefinite
summary retention, and closure evidence.

Checkpoint A is complete at exact pushed commit
`9191182601d79b996f1848a1e867e50b7d6eaf1c`. `run`, `start`, `serve`,
`tools run`, and `mcp serve` default to an observer ceiling that manifest and
stored progressive trust cannot bypass. Host authority requires both an
explicit host profile and acknowledgement, without upgrading requested role
trust; unsupported `isolated` execution fails before state or subprocess work.
Observer suppresses direct target-writing scan, Jira, remediation,
intervention-debt, convention, hygiene, and learning paths while retaining
owner-local bookkeeping. QA caught one final manual-run path that hard-coded
contributor trust under acknowledged host; it was corrected before push to
parse the manifest role trust with observer fallback, and a live-command
regression proves the observer mutator rejection reaches the model without
creating the requested file.

The exact focused gates were
`go test ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars ./internal/scanner ./internal/docsconsistency ./internal/docsync`,
`go test -race ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars`,
`go vet ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars ./internal/scanner`,
`go run ./cmd/mars docsync audit --repo .` with 362 files checked and zero
findings, and `git diff --check`. The caught trust delta then passed
`go test ./cmd/mars -run '^TestRunHostAcknowledgementPreservesObserverRoleTrust$' -count=1`,
full `go test ./cmd/mars`, `go test -race ./cmd/mars`, `go vet ./cmd/mars`, and
`git diff --check`. QA, Security, Release Manager, and Orchestrator returned GO.

Checkpoint B1 is complete at exact pushed commit
`5c23f536fadd9ab18694e3f46ed9b10ca96594da`. One thin child-environment seam
preserves ordinary non-sensitive PATH, HOME, temporary-directory, locale, and
toolchain/cache variables while filtering credential-like, MARS, GitHub,
cloud/provider, delimiter-bounded auth, SSH, token, secret, password, API-key,
private-key, and credential names. Parent-only
`MARS_CHILD_ENV_ALLOWLIST` can restore only exact names and never reaches a
child. Every named shell, MARS CLI, dependency, Git, code-intelligence,
managed-inference, MCP stdio, Jira, GitHub-auth, and source-update child
receives an explicit environment. Jira removes repository-requested names from
the sanitized base and requires the independent owner allowlist before
restoring a value. This is name-based exposure reduction, not containment.

The exact B1 gates were
`go test ./internal/childenv ./internal/tools ./internal/inference ./internal/serve ./internal/codeintel ./internal/githubauth ./internal/selfupdate ./internal/mcpclient ./internal/jira ./cmd/mars -count=1`,
`GOCACHE=/private/tmp/mars-t076b1-root-race-cache go test -race ./internal/childenv ./internal/tools ./internal/inference ./internal/jira -run 'Test(Filter|ApplyWith|ShellExecSanitizes|DependencySyncChild|ServerManagedChild|ProxyEnvironment)' -count=1`,
`GOCACHE=/private/tmp/mars-t076b1-root-cache go vet ./internal/childenv ./internal/tools ./internal/inference ./internal/serve ./internal/codeintel ./internal/githubauth ./internal/selfupdate ./internal/mcpclient ./internal/jira ./cmd/mars`,
`go test ./internal/docsconsistency ./internal/docsync -count=1`,
`go run ./cmd/mars docsync audit --repo .` with 364 files checked and zero
findings, and `git diff --check`. Real-child coverage includes foreground and
background shell, dependency PATH/cache preservation, managed inference, and
the Jira repository-request plus owner-allowlist intersection. QA, Security,
Release Manager, and Orchestrator returned GO.

Checkpoint B2 is complete at exact pushed commit
`473b829efe865630f4942b55af9e0108d7529d0c`. Background process records require
a job ID; listing, direct helper policy, targeted kill, and lifecycle cleanup
are scoped to that job. Unowned and cross-job targets fail closed without an
OS-wide fallback. Cleanup sends TERM to the recorded process group, waits at
most two seconds, and sends KILL if that group remains. Server jobs, manual
run, one-shot tools run, and MCP EOF/error clean only their owned groups. Serve
no longer performs blind lsof/pgrep cleanup, so a foreign listener and an
unrelated llama-server-named process survive.

The exact B2 gates were
`go test ./internal/tools ./internal/mcpstdio ./internal/serve ./internal/scanner ./cmd/mars`,
`go test -race ./internal/tools ./internal/mcpstdio ./internal/serve ./cmd/mars`,
`go vet ./internal/tools ./internal/mcpstdio ./internal/serve ./internal/scanner ./cmd/mars`,
`go test ./internal/docsync ./internal/docsconsistency`,
`go run ./cmd/mars docsync audit --repo .` with 364 files checked and zero
findings, and `git diff --check`. QA, Security, and Orchestrator returned GO.

The final launch-scope config correction is complete at exact pushed commit
`9eb3f96d1de9f91ba54ee4f2dd70d0cdf98b8708`: setup and config persistence
create or tighten regular config leaves to `0600`, reject links/non-regular
entries, and preserve custom parent modes. Focused config/setup normal, race,
vet, formatting, and diff gates pass. An installed clean Go 1.26.5 candidate
with SHA-256 `dc8033d3024624ae182175fec80362a87e8585048f5cf9d17cb319f0a0420dbe`
completed setup under `umask 000` and created `~/.mars/config.yaml` at `0600`.
Commit `31b00b1ce01cce10df81fc0769f6bdbbc94ff1b5` then corrected every live shared
serve database path claim to the runtime's actual legacy
`~/.mars-harness/db/mars.db` without migrating state.

T-076 is complete under the owner's risk-proportionate launch governor.
Broader database/log permission normalization, centralized redaction, trace
export, purge, and retention are deferred post-launch product hardening, not
publication gates. T-058 then passed at exact pushed commit
`57d7851d9c82975256761b0134d20e91382e9bcd`: the installed in-app browser
confirmed hostile runtime data as inert text and eight observed assets from the
exact loopback origin with zero external origins or console errors. The replay
found and corrected one cached-DOM logout disclosure; the corrected logout
returned to an anonymous page with no privileged rows or target data.
F-010-S024 and F-017-S002 pass. These
changes did not alter version, legal/rights or installed-App disposition,
Release, settings, visibility, signing, publication, or announcement state.
The repository remains private at `VERSION=0.68.49` and Primary Status remains
`primary_blocked`.

## Completed Walking Skeleton — T-077

Checkpoint A is complete at exact pushed commit
`10b62f7d59620022b2e1030c5f33856d0c16e70f`. The bounded 33-path packet makes
ordinary setup independent of GitHub auth; makes the exact no-redirect official
release-metadata request anonymously first; permits optional credential
resolution and one same-origin/path retry only after an exact `401`, `403`, or
`404`; adds idempotent config-only `auth github clear-local`; and preserves
doctor's selected custom `ConfigPath`. The existing signed release consumer was
not redesigned.

Affected-package normal tests, focused race tests, affected-package vet,
documentation consistency, DocSync across 364 files with zero findings,
formatting, and diff checks passed. QA, Security, and Orchestrator returned GO.

Checkpoint B is complete at exact pushed commit
`04d6ba6844126dc84eb6bedc13c78bd31f8d371d`. The bounded 21-file packet
stable-sorts one unique pending-artifact plan, fixes the concrete local bundle,
and shows immutable identities, exact byte sizes, license IDs/URLs, and
applicable terms/notice URLs. Interactive setup confirms once; non-TTY and JSON
use require exact `--download --yes`, with a complete JSON preflight event on
stderr before requests and the same plan in final stdout. The accepted plan is
compared again and both downloaders admit only listed identities. Decline,
missing acknowledgement, incomplete provenance, or a changed plan causes zero
download requests and zero download-artifact writes. Skip, test, deferred, and
cloud paths need no acknowledgement, no legal attestation is persisted, and
automatic Linux llama.cpp acquisition remains disabled.

Affected-package normal and race tests, affected-package vet, documentation
consistency and DocSync tests, DocSync across 366 files with zero findings,
formatting, and diff checks passed. QA, Security, and Orchestrator returned GO.

Checkpoint C implementation is complete at exact pushed commit
`85c689c70ef801a2747acabf537739c9ebad3c12`. The bounded 13-file packet
executes only through a privileged Bash startup boundary, carries optional
GitHub credentials outside `env(1)` arguments and Go, closes their transport
descriptors before external commands, constrains the exact Go build to the
public proxy and SumDB in trusted private staging, and verifies canonical
runtime command/module/tag/`h1` identity with no replacements before the
existing signed updater may acquire or replace anything. The hidden bootstrap
admission cannot weaken the normal signed-update lane; the handoff skips only
post-replacement shell-profile mutation. Pre-commit failures preserve the
prior binary, while recovery-required results preserve transaction evidence and
report truthful recovery state. Verified cleanup uses `-modcacherw` and reports
any residue without exposing paths.

Normal and race tests for `internal/selfupdate` and `cmd/mars`, affected vet,
the hostile installer suite, Bash 3.2 syntax and native descriptor-close proof,
CLI/help mirror checks, documentation consistency, DocSync across 366 files
with zero findings, and diff checks passed. QA, Security, Release Manager, and
Orchestrator returned GO on frozen installer hashes `87c2bc1d…` and
`d055e830…`. Exact source `56b8de3` then passed four Go 1.26.5 CGO-disabled
Darwin/Linux AMD64/ARM64 builds and independently reviewed clean-HOME macOS
arm64 plus native Linux arm64 non-root source/setup lanes. The Linux installer
suite passed under GNU `stat`; both platforms ran deferred setup twice at
`4/0` then `0/4` steps with no download artifacts, GitHub tokens, local
fallback, llama-server, or model. The T-075 test-fixture drift was corrected
test-only at the same source without weakening fail-closed scanning. QA and
Security recomputed the retained evidence, and all four temporary roots were
removed and verified absent. The exact report is
`docs/validation/reports/2026-08-24-t077-bootstrap-setup-closure.md`.

T-077 passes. A real official-tag signed lifecycle remains T-080/T-081, so
F-017-S003 remains incomplete. The repository remains private at
`VERSION=0.68.49` with Primary Status `primary_blocked`; the later owner
dispositions close the T-073, account-wide App, and billing planning holds,
while hosted CI still needs a green rerun. No release, settings,
visibility, signing, publication, or announcement authority changed.

## Active Walking Skeleton — T-078

T-078 was created through `ticket_create` after T-077's reviewed closure. Its
first GoReleaser/Cosign selection failed before execution and the replacement
route expanded into a bespoke secure-build platform. The owner stopped that
scope, preserved the exact source/evidence as a non-authorizing checkpoint, and
approved AD-315: conventional Go builds and archives, upstream Syft, and
GitHub's standard `actions/attest` under split least privilege.

The independent read-only hosted slice reconstructed T-072's exact 401-run
digest, froze 466 completed and zero active runs, and acquired the exact 65-run
delta. Both admitted offline scanners returned zero findings, errors, skips,
timeouts, rejected inputs, or unresolved candidates. This does not authorize
deletion: the report commit and every later run remain in scope.

T-078 authorizes repository source/tests/docs and read-only hosted acquisition.
It authorizes no launch tag, real attestation, upload, supported Release,
visibility change, App mutation, hosted setting change, or announcement.
Deleting 500 legacy assets, 77 deployments, one obsolete Release object, or
refreshed workflow runs, or enabling future-only immutable Releases, requires
separate exact owner approval naming that mutation plus immediate live-state
revalidation.
The owner reported GitHub funding restored on 2026-08-24, so hosted CI must be rerun rather
than carried as a planning blocker. The expected post-sanitation surface is 301
tags, 56 historical Release objects, zero assets, and zero deployments. After
separately approved public visibility, T-080 later converges on 303 tags, 58
Release objects, and 20 uploaded Release assets.

## Completion Gates By Ticket

- **T-072 — passed 2026-08-08:** Every then-retained GitHub-hosted surface was collected, confirmed empty, or not applicable; scanners had zero errors, skips, and unresolved secret candidates; exact cleanup IDs were frozen. Account-wide App scope is not a MARS launch gate.
- **T-073 — passed 2026-08-24:** Machine checkpoints passed; the owner accepted the recorded name risk and attested publication authority. This is a complete owner disposition, not a trademark-clearance finding.
- **T-074:** GitHub App callback and telemetry collection are literal-loopback, bounded, replay-safe, and fail closed.
- **T-075 — passed 2026-08-09:** All named model/agent-controlled repository paths use the descriptor-relative no-follow interface; index-only and force-added secrets are scanned without reproducing values; four builds and installed Dogfood pass.
- **T-076 — passed 2026-08-09:** Observer is non-mutating by default; host requires acknowledgement; isolated is unavailable; child environments, job-owned processes, persisted-config credential modes, and shared-database path truth pass. Broader local-state, redaction, export, purge, and retention work is deferred.
- **T-058 — passed 2026-08-09:** A real browser proved hostile runtime strings inert, browser controls protected, all observed assets same-origin, and logout free of cached privileged DOM against exact current commit `57d7851`.
- **T-077 — passed 2026-08-24:** Anonymous-first access and config-only `clear-local` pass at `10b62f7`; the stable third-party download plan and explicit acknowledgement boundary pass at `04d6ba6`; the exact-version Go/SumDB bootstrap passes at `85c689c7`; and exact source `56b8de3` passes four builds plus independently reviewed clean macOS/Linux source/setup and verified evidence cleanup.
- **T-078 — parked on separate approval:** AD-315's dormant standard workflow, compatible GitHub-attestation consumer, and two-root no-publish rehearsal pass through exact source `d411cbe`. Deployment state drifted to zero before the approved sanitation transaction, so no mutation occurred and the revised 500-asset/474-run cleanup plus future-only immutable Releases remains separately approval-gated. Exact pushed sources `a4bbf81` and `2dd73fc` plus hosted runs `32848968969` and `32849610523` pass Go 1.25.13, Go 1.27.0, dependency notices, the intentional Go 1.25.12 rejection, and zero called application vulnerabilities without an exception.
- **T-079 — final verification:** Source checkpoint `b807afa` and hosted run `32901437495` pass. GitHub now enforces GitHub-owned full-SHA Actions, approval-gated read-only/no-secret private-fork workflows, Dependabot alerts/security updates, Discussions, and exact active ruleset `21491158`; visibility remains private and Releases/tags/Pages/Apps are unchanged. The exact receipt is `docs/validation/reports/2026-08-25-t079-private-contribution-controls.md`. Public-only CodeQL, secret scanning/push protection, private vulnerability reporting, Pages, and the live hostile-fork smoke remain T-080 work after visibility.
- **T-080:** After separate visibility confirmation, public-only controls are enabled and verified; attested public `v0.69.0` and `v0.69.1` exact-ten releases pass macOS/Linux update and rollback; expected convergence is 303 tags, 58 Release objects, and 20 uploaded Release assets belonging only to the two launch releases.
- **T-081:** Logged-out public lifecycle and hostile-fork smoke pass twice around a clean 48-hour canary; an announcement is posted and G-OSS-001/F-017/F-018 close.

## Global No-Go And Rollback Rules

Visibility cannot change while any secret, privacy, ownership, undisposed
name-risk, provenance, license, called-vulnerability, reachable runtime P0/P1, unsigned
legacy asset, anonymous-access, fork-authority, settings, lifecycle, or sign-off
finding remains. Before visibility changes, stop and repair the owning ticket.
After visibility changes, confidentiality cannot be restored: supersede a bad
release with a new immutable patch tag and never move or replace a public tag.
A canary code or public-contract correction requires `v0.69.2` and repetition
of the complete anonymous lifecycle and 48-hour canary.
