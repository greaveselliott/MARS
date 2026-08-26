# F-009: Release And Update Lifecycle

- Feature ID: F-009
- Goals: G-002, G-004
- Status: partially-passing
- Owner: Release Manager

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-009-S001 - Semantic release notes update `VERSION` and `CHANGELOG.md` from commits.
2. F-009-S002 - Release-note commits are ignored as the base for the next generated entry.
3. F-009-S003 - Done-ticket metadata classifies shipped BDD scenarios separately from enablers.
4. F-009-S004 - `update tool` authenticates signed release archives and durably replaces or restores the installed binary.
5. F-009-S005 - `update harness` refreshes deployed harness defaults through the unified update verb.
6. F-009-S006 - The standalone `release verify-assets` command is retired; MARS source uses F-018 and targets use repository-owned gates.
7. F-009-S007 - Source and generated targets inherit the same versioning and release-note discipline.
8. F-009-S008 - Generated release notes explain impact, why, and what changed before commit buckets.
9. F-009-S009 - Historical release entries are backfilled to the current narrative standard from marker ranges.
10. F-009-S010 - The installed CLI reports its version through the explicit command and root-level version flags.
11. F-009-S011 - Official release access is anonymous-first with optional private-fork auth.
12. F-009-S012 - Approved product validation enters release review automatically in generated target lifecycles.
13. F-009-S013 - Repositories own their producer and remote publication never replaces artifact verification.
14. F-009-S014 - Version tags can only be created at the release-note commit.
15. F-009-S015 - Release Manager uses the structured MARS CLI tool instead of stale PATH binaries.
16. F-009-S016 - Source checkout onboarding and update are the primary path for repo cloners.
17. F-009-S017 - The standalone `release audit` command is retired; F-018-S004 owns fail-closed remote convergence.
18. F-009-S018 - The bespoke GitHub mirror publisher is retired; its fail-closed invariant moves to F-018-S004.
19. F-009-S019 - Every non-release semantic commit completes as an immutable, verified release or records a release blocker.

## Scenarios

### F-009-S001: Generated Version And Changelog

Given semantic commits exist after the current release marker
When `mars release notes --repo <path> --bump auto` runs
Then `VERSION` is bumped and `CHANGELOG.md` receives a generated entry

### F-009-S002: Release Commit Exemption

Given the most recent commit is `release: notes X.Y.Z`
When the next release-note generation runs
Then that release commit is ignored so it does not create a recursive version entry

### F-009-S019: Semantic Commit Release Completion

Given a validated non-release semantic commit has reached `main`
When its generated release-note commit is pushed
Then an immutable `vX.Y.Z` tag is created at that exact release-note commit
And the repository-owned producer publishes and independently verifies the
configured release assets, attestations, and supported consumer path before the
change is called complete
And a missing producer, remote, credentials, publication result, or verification
gate records `release_blocked` rather than treating the pushed commit as released
And initialized target harnesses inherit the same completion rule through their
own configured producer

### F-009-S003: BDD Scenario Release Classification

Given done tickets include `work_type`, `bdd_scenarios`, and evidence metadata
When release notes are generated
Then shipped feature scenarios are named separately from enabler work

### F-009-S004: Signed Archive Tool Update

Given an approved release exposes the canonical archive, exact eight-entry
checksum file, and offline Sigstore bundle for the requested platform
When `mars update tool` installs the tool
Then the consumer verifies the signature, workflow identity, immutable tag and
full commit, platform/build metadata, archive checksum, digest, and bounded
structure before replacement
And a pre-commit failure preserves the prior fixed binary, while a failed
post-commit compensation returns recovery-required guidance, preserves the
transaction evidence, and remains blocked until trusted-source repair
And private releases are authenticated through the Getting Started auth resolver in this order: `GH_TOKEN`, `GITHUB_TOKEN`, GitHub CLI auth, then optional local config token
And `mars auth github setup` saves a verified GitHub CLI token as the owner-only local fallback without printing the token
And private release assets are downloaded through GitHub asset API URLs when release metadata provides them
And missing or invalid auth points to `mars auth github setup`

Given a fresh machine has stable Go 1.25.13 or newer, one existing
owner-controlled install directory, and an independently reviewed checkout at
the same exact release tag
When `./scripts/install.sh` is executed directly and receives one exact stable
`vMAJOR.MINOR.PATCH` tag
Then it builds only `github.com/greaveselliott/mars/cmd/mars@<exact-tag>` in
owner-only temporary staging through `https://proxy.golang.org` and
`sum.golang.org`, with workspace, replacement, direct, private-module, and
no-sum bypasses disabled, and no network-fetched script may be piped into a shell
And the privileged Bash shebang suppresses inherited functions and `BASH_ENV`,
an explicit shell-interpreter invocation fails closed, and the real body starts
with only inherited `PATH`, `HOME`, and `TMPDIR`; optional GitHub tokens cross
over dedicated descriptors, remain absent from Go, and are exported only to the
staged signed updater
And the script resolves one absolute Go executable, disables Go auth and CGO,
neutralizes inherited compiler/tool controls, applies `-modcacherw`,
validates the resolved temporary-root ancestry as private current-user or safe
root-owned directories, and uses a private staging `TMPDIR`/`GOTMPDIR`
And the staged command's running `runtime/debug.BuildInfo` must name the
canonical command and module, exact requested version, canonical SHA-256 `h1`
module sum, and no replacements before bootstrap admission
And the staged command invokes `mars update tool` with the same exact tag and
selected final install directory, leaving archive/signature verification and
durable replacement solely to the signed updater and skipping ordinary shell
PATH mutation during the handoff
And pre-commit rejection leaves an existing final binary unchanged, while a
recovery-required result preserves transaction evidence and requires
trusted-source repair before retry
And successful script exit requires verified private-staging removal; an
ordinary failure retains its original error plus a fixed path-free cleanup
warning when removal is incomplete, while a post-install cleanup failure says
the binary was installed but staging cleanup remains incomplete
And later packaged operation does not require Go.

### F-009-S005: Unified Harness Update

Given a target repo has generated harness metadata
When `mars update harness --repo <path>` runs
Then the update path refreshes missing target harness defaults without overwriting user-owned configuration

### F-009-S006: Retired Standalone Asset Verification

Given T-066 D1 has retired the weaker standalone consumer
When `mars release verify-assets` is invoked
Then the CLI reports an unknown command and performs no verification
And MARS source uses the F-018 signed archive contract while target repositories
use their repository-owned artifact-verification gate
And missing, inaccessible, partial, or unverifiable state remains blocked rather
than being classified as clean

### F-009-S007: Mirrored Release Discipline

Given the source harness requires versioning after every non-release semantic commit
When target harness docs are generated
Then they include the same version, changelog, release-note, and release guidance unless explicitly marked source-only

### F-009-S008: Detailed Release Narrative

Given semantic commits exist after the current release marker
When release notes are generated
Then the changelog entry includes complete `Impact`, `Why`, and `What Changed` narrative before semantic commit buckets, using commit-body narrative fields when available and conservative generated prose otherwise
And structural delivery changes such as operating-model, structured dispatch, persona, documentation-sync, and CLI/tool-sync work use topic-aware fallback prose instead of merely restating the commit subject

### F-009-S009: Historical Release Narrative Backfill

Given `CHANGELOG.md` contains older marker-backed release entries
When `mars release backfill-notes --repo <path>` runs
Then each selected entry derives its non-release commits from adjacent release markers or, for non-linear old history, from existing semantic-bucket commit references, replaces legacy narrative sections with `Impact`, `Why`, and `What Changed`, preserves semantic commit buckets and delivery evidence, and reports missing markers or empty release ranges instead of inventing history
And entries that already contain complete current `Impact`, `Why`, and `What Changed` sections are preserved rather than flattened into generic regenerated prose

### F-009-S010: Version Shortcut Parity

Given a user wants to confirm the installed MARS binary version
When `mars version`, `mars --version`, or `mars -v` runs
Then each entrypoint prints the same version, OS/architecture, commit, and build date line

### F-009-S011: Anonymous-First Release Access

Given official MARS release metadata is public
When a user runs ordinary `mars setup` or checks release access
Then setup completes without a private-release-auth step or GitHub credentials
And `mars auth github check` makes one exact, no-redirect anonymous request to the official `api.github.com` release-metadata endpoint
And only an exact `401`, `403`, or `404` may resolve optional credentials and make one authenticated retry to the same origin and path
And redirects, transport failures, unexpected statuses, and custom origins never receive credentials
And the check reports access as `anonymous`, `authenticated`, or `unavailable` without printing token or credential-derived values
And `mars auth github setup` verifies access and stores a local fallback only when auth comes from GitHub CLI or an explicit `--token`
And `mars auth github clear-local` idempotently removes only the stored config `github_token` while preserving all other config fields and leaving environment variables, GitHub CLI and GitHub App credentials, repositories, and remote state unchanged
And `mars doctor` reports anonymous, authenticated, or unavailable release access with a concrete fix
And agents can use the read-only `github_auth_check` tool before update, release verification, install repair, or version-drift remediation

### F-009-S012: Dispatch-To-Release Review

Given a generated target lifecycle has completed product planning, implementation, QA, security, and dogfood validation
When Dogfood records an approved or completed disposition after product or ticket commits
Then dispatch routes to Release Manager before stopping so generated target `VERSION` and `CHANGELOG.md` are updated from unreleased semantic commits
And Release Manager runs `mars release backfill-notes --repo . --check` so legacy release entries are found deliberately instead of being missed during routine versioning
And a Release Manager `release_blocked` publication disposition stops dispatch as operator-visible release evidence instead of routing back to Dogfood or another already-completed product validation role

### F-009-S015: Release Review Uses Structured CLI Resolution

Given a deployed target has an older `mars` binary earlier on `PATH`
When Release Manager needs release notes or backfill
Then the role uses `mars_cli` with structured args instead of `shell_exec mars ...`
And `shell_exec` blocks direct `mars` binary invocations with a correction that names the equivalent `mars_cli` args

### F-009-S016: Source Checkout Onboarding

Given a user clones the MARS source repo
When they follow the README quick start
Then the primary path installs with `make install`, runs `mars setup`, verifies with `mars doctor`, initializes a target repo, and previews an agent run with `--dry-run`
And the README describes system requirements, GPU expectations, model downloads, disk/network prerequisites, and anonymous-first official release access before optional private-fork auth
And setup displays and requires acknowledgement of the exact third-party download plan before local artifact requests or writes; non-interactive/JSON examples use `--download --yes`
And source checkout updates use `make update-tool` as the recommended command for safely fast-forwarding and reinstalling from the clone
And release review cannot fail solely because a stale installed binary lacks a newer command surface

### F-009-S013: Repository-Owned Release Producer

Given a source or target repository needs release artifacts
When Release Manager prepares the release
Then the repository's approved producer and artifact contract are used
And generated target repositories do not inherit MARS's source-specific implementation
And remote publication never replaces local artifact verification

### F-009-S014: Release Tag Commit Invariant

Given generated release notes have updated `VERSION` to `X.Y.Z` and changed `CHANGELOG.md`
When Release Manager attempts to create or update `vX.Y.Z`
Then tag creation is blocked until the release-note files are committed as `release: notes X.Y.Z`
And the tag is blocked when its explicit target resolves to any commit other than the current release-note `HEAD`
And `git_release_guard` fails when `vX.Y.Z` already exists but points at a pre-release-note commit

### F-009-S017: Retired Standalone Release Audit

Given T-066 D1 has retired the standalone audit
When `mars release audit --repo .` runs
Then the CLI reports an unknown command and performs no audit
And repository-owned remote verification plus F-018-S004 must prove exact signed
artifact identity and inventory convergence
And unavailable or unverifiable remote evidence is blocked, never clean

### F-009-S018: Retired Bespoke GitHub Mirror Publisher

Given source production uses the AD-315 conventional F-018 contract and the
older bespoke publisher remains retired
When the MARS release command tree and source packages are inspected
Then `release publish-assets` is unknown and no bespoke GitHub create, upload,
clobber, or convergence implementation is reachable
And exact remote convergence remains a required future invariant under
F-018-S004 rather than a current publication capability.

## Out of Scope

- Treating tags as the only release-note state.
- Publishing GitHub Releases when authentication or remote capability is unavailable.
- Runtime dependencies on npm, Postgres, Redis, or Grafana.
- Requiring a cloud LLM to write release prose.

## Descoped Scenarios

None.

## Evidence

- F-009-S001: `go test ./internal/release -run TestPrepareGeneratesVersionAndChangelog`
- F-009-S002: `go test ./internal/release -run TestPrepareUsesChangelogMarkerAsBase`
- F-009-S003: `go test ./internal/release -run TestPrepareClassifiesDeliveryEvidenceFromDoneTickets`
- F-009-S004: `go test ./internal/selfupdate -run 'Test(InstallScript|RunRelease|VerifyMARSReleaseArchive|VerifySigstoreChecksumsEvidenceRealOfflineFixture|MARSSigstorePolicyBindsExactWorkflowAndCommit|FetchVerifiedMARSRelease|ReplaceVerifiedMARSRelease)'`
- F-009-S005: `go test ./internal/scanner -run TestUpgrade_preservesUserConfiguredManifestAndPrompts`
- F-009-S006: `go test ./cmd/mars -run TestReleaseLegacyConsumerCommandsAreRetired`
- F-009-S007: `go test ./internal/scanner -run TestInit_success` and docs-consistency checks for release guidance
- F-009-S008: `go test ./internal/release -run 'TestRenderReleaseNarrative(UsesImpactWhyAndWhat|ProfilesStructuredDispatch)'`
- F-009-S009: `go test ./internal/release -run TestBackfillNotes` and `go test ./cmd/mars -run TestReleaseBackfillNotesCommandChecksAndWrites`
- F-009-S010: `go test ./cmd/mars -run TestVersionEntrypointsPrintSameVersionLine`
- F-009-S012: `go test ./internal/orchestration -run 'TestDecide_(dogfoodApprovalRoutesDirectlyToReleaseManager|releaseManagerReleaseBlockedStopsDispatch|orchestratorCannotRouteReleaseBlockedBackToDogfood)'`, `go test ./internal/serve -run TestHandleJobComplete_releaseBlockedStopsWithoutDogfoodLoop`, and `go test ./internal/scanner -run TestInit_success`
- F-009-S013: `go test ./internal/scanner -run TestInit_success` and `go test ./internal/tools -run TestReleaseWorkflowsUseRepositoryOwnedProducer`
- F-009-S014: `go test ./internal/tools -run 'TestShellExecPolicyBlocksReleaseTag|TestGitReleaseGuardReportsStaleReleaseTag'`
- F-009-S015: `go test ./internal/tools -run TestShellExecPolicyBlocksMarsBinary` and `go test ./internal/scanner -run TestInit_success`
- F-009-S017: `go test ./cmd/mars -run TestReleaseLegacyConsumerCommandsAreRetired`
- F-009-S018: `go test ./cmd/mars -run TestReleasePublishAssetsCommandIsRetired` plus the source negative-oracle checks recorded by T-065
