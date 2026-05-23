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
4. F-009-S004 - `update tool` installs checksum-verified release assets atomically.
5. F-009-S005 - `update harness` refreshes deployed harness defaults through the unified update verb.
6. F-009-S006 - `release verify-assets` fails when platform binaries or checksums are missing from a GitHub Release.
7. F-009-S007 - Source and generated targets inherit the same versioning and release-note discipline.
8. F-009-S008 - Generated release notes explain impact, why, and what changed before commit buckets.
9. F-009-S009 - Historical release entries are backfilled to the current narrative standard from marker ranges.
10. F-009-S010 - The installed CLI reports its version through the explicit command and root-level version flags.
11. F-009-S011 - Private release auth is a first-class Getting Started operating model.
12. F-009-S012 - Approved product validation enters release review automatically in generated target lifecycles.
13. F-009-S013 - GitHub Release objects are created even when asset workflows are blocked.
14. F-009-S014 - Version tags can only be created at the release-note commit.
15. F-009-S015 - Release Manager uses the structured Mars Harness CLI tool instead of stale PATH binaries.

## Scenarios

### F-009-S001: Generated Version And Changelog

Given semantic commits exist after the current release marker
When `mars-harness release notes --repo <path> --bump auto` runs
Then `VERSION` is bumped and `CHANGELOG.md` receives a generated entry

### F-009-S002: Release Commit Exemption

Given the most recent commit is `release: notes X.Y.Z`
When the next release-note generation runs
Then that release commit is ignored so it does not create a recursive version entry

### F-009-S003: BDD Scenario Release Classification

Given done tickets include `work_type`, `bdd_scenarios`, and evidence metadata
When release notes are generated
Then shipped feature scenarios are named separately from enabler work

### F-009-S004: Checksum-Verified Tool Update

Given a release asset and `checksums.txt` are available
When `mars-harness update tool` installs the tool
Then checksum mismatch prevents replacement and valid assets atomically replace the installed binary
And private releases are authenticated through the Getting Started auth resolver in this order: `GH_TOKEN`, `GITHUB_TOKEN`, GitHub CLI auth, then optional local config token
And `mars-harness auth github setup` saves a verified GitHub CLI token as the owner-only local fallback without printing the token
And private release assets are downloaded through GitHub asset API URLs when release metadata provides them
And missing or invalid auth points to `mars-harness auth github setup`

### F-009-S005: Unified Harness Update

Given a target repo has generated harness metadata
When `mars-harness update harness --repo <path>` runs
Then the update path refreshes missing target harness defaults without overwriting user-owned configuration

### F-009-S006: Release Asset Verification

Given a GitHub Release exists for version `vX.Y.Z`
When `mars-harness release verify-assets --version vX.Y.Z` runs
Then it fails unless all required platform binaries and `checksums.txt` are attached

Given a release-note commit and tag `vX.Y.Z` have been pushed
When `gh release view vX.Y.Z` fails because the tag workflow did not create the release object
Then Release Manager creates or updates a notes-only GitHub Release for the existing tag from the generated `CHANGELOG.md` entry
And the release remains blocked for installer or self-update claims until `mars-harness release verify-assets --version vX.Y.Z` passes

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
When `mars-harness release backfill-notes --repo <path>` runs
Then each selected entry derives its non-release commits from adjacent release markers or, for non-linear old history, from existing semantic-bucket commit references, replaces legacy narrative sections with `Impact`, `Why`, and `What Changed`, preserves semantic commit buckets and delivery evidence, and reports missing markers or empty release ranges instead of inventing history
And entries that already contain complete current `Impact`, `Why`, and `What Changed` sections are preserved rather than flattened into generic regenerated prose

### F-009-S010: Version Shortcut Parity

Given a user wants to confirm the installed Mars Harness binary version
When `mars-harness version`, `mars-harness --version`, or `mars-harness -v` runs
Then each entrypoint prints the same version, OS/architecture, commit, and build date line

### F-009-S011: Private Release Auth Getting Started

Given Mars Harness release assets live in a private GitHub Release repository
When a user follows Getting Started
Then the documented sequence includes `mars-harness auth github setup`, `mars-harness setup`, `mars-harness doctor`, and `mars-harness update tool`
And `mars-harness auth github check` reports `status`, `auth_source`, `repo_access`, `release_access`, `message`, and `next_action` without printing token values
And `mars-harness auth github setup` verifies access and stores a local fallback only when auth comes from GitHub CLI or an explicit `--token`
And `mars-harness setup` checks private-release auth unless `--skip-github` or `--test-mode` is used
And `mars-harness doctor` reports private-release auth readiness with a concrete fix
And agents can use the read-only `github_auth_check` tool before update, release verification, install repair, or version-drift remediation

### F-009-S012: Dispatch-To-Release Review

Given a generated target lifecycle has completed product planning, implementation, QA, security, and dogfood validation
When Dogfood records an approved or completed disposition after product or ticket commits
Then dispatch routes to Release Manager before stopping so generated target `VERSION` and `CHANGELOG.md` are updated from unreleased semantic commits
And Release Manager runs `mars-harness release backfill-notes --repo . --check` so legacy release entries are found deliberately instead of being missed during routine versioning
And a Release Manager `release_blocked` publication disposition stops dispatch as operator-visible release evidence instead of routing back to Dogfood or another already-completed product validation role

### F-009-S015: Release Review Uses Structured CLI Resolution

Given a deployed target has an older `mars-harness` binary earlier on `PATH`
When Release Manager needs release notes, backfill, or asset verification
Then the role uses `mars_harness_cli` with structured args instead of `shell_exec mars-harness ...`
And `shell_exec` blocks direct `mars-harness` binary invocations with a correction that names the equivalent `mars_harness_cli` args
And release review cannot fail solely because a stale installed binary lacks a newer command surface

### F-009-S013: Notes-Only Release Object Fallback

Given GitHub release credentials are configured and the tag `vX.Y.Z` has been pushed
When the Release workflow is blocked by CI, billing, permissions, or another workflow failure before publishing the GitHub Release object
Then the release process must publish a notes-only GitHub Release using the generated changelog entry for `X.Y.Z`
And the operating record must distinguish that release-object publication succeeded from the remaining asset blocker
And asset verification remains failing until the required binaries and checksums are attached

### F-009-S014: Release Tag Commit Invariant

Given generated release notes have updated `VERSION` to `X.Y.Z` and changed `CHANGELOG.md`
When Release Manager attempts to create or update `vX.Y.Z`
Then tag creation is blocked until the release-note files are committed as `release: notes X.Y.Z`
And the tag is blocked when its explicit target resolves to any commit other than the current release-note `HEAD`
And `git_release_guard` fails when `vX.Y.Z` already exists but points at a pre-release-note commit

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
- F-009-S004: `go test ./internal/selfupdate -run TestRunReleaseAssets`
- F-009-S005: `go test ./internal/scanner -run TestUpgrade_preservesUserConfiguredManifestAndPrompts`
- F-009-S006: `go test ./internal/selfupdate -run TestVerifyReleaseAssetsReportsMissingAssets`
- F-009-S007: `go test ./internal/scanner -run TestInit_success` and docs-consistency checks for release guidance
- F-009-S008: `go test ./internal/release -run 'TestRenderReleaseNarrative(UsesImpactWhyAndWhat|ProfilesStructuredDispatch)'`
- F-009-S009: `go test ./internal/release -run TestBackfillNotes` and `go test ./cmd/mars-harness -run TestReleaseBackfillNotesCommandChecksAndWrites`
- F-009-S010: `go test ./cmd/mars-harness -run TestVersionEntrypointsPrintSameVersionLine`
- F-009-S012: `go test ./internal/orchestration -run 'TestDecide_(dogfoodApprovalRoutesDirectlyToReleaseManager|releaseManagerReleaseBlockedStopsDispatch|orchestratorCannotRouteReleaseBlockedBackToDogfood)'`, `go test ./internal/serve -run TestHandleJobComplete_releaseBlockedStopsWithoutDogfoodLoop`, and `go test ./internal/scanner -run TestInit_success`
- F-009-S013: `go test ./internal/docsconsistency -run TestSourceRepoVersioningRuleIsDocumented` and `go test ./internal/scanner -run TestInit_success`
- F-009-S014: `go test ./internal/tools -run 'TestShellExecPolicyBlocksReleaseTag|TestGitReleaseGuardReportsStaleReleaseTag'`
- F-009-S015: `go test ./internal/tools -run TestShellExecPolicyBlocksMarsHarnessBinary` and `go test ./internal/scanner -run TestInit_success`
