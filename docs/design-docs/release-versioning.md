# Release Versioning And Patch Notes

**Status:** Accepted
**Date:** 2026-05-02

## Context

Mars Harness had a build-time `version` command and release-manager prompt, but no repo-owned semantic version, no committed patch notes, and no deterministic way for the harness or a target repo to produce them. That made release state live in ad hoc human memory.

The behavior also needs to mirror into initialized target repositories. If the source harness expects semantic versioning and generated patch notes, target harnesses should receive the same version files, release guidance, and release-manager workflow.

## Decisions

### AD-049: VERSION Is The Semantic Version Source

Mars Harness and initialized target repos use a root `VERSION` file containing `MAJOR.MINOR.PATCH`. Release builds may still inject build metadata with linker flags, but the repo-owned version is what agents and release tooling update.

### AD-050: CHANGELOG.md Is Generated Patch Notes

Patch notes live in root `CHANGELOG.md`. Entries are generated from semantic commits using `mars-harness release notes`. Each generated entry includes a release marker with the version and source commit so the next run can find only new commits even when no git tag exists yet.

Generated entries start with plain-English narrative before the commit buckets.
The narrative must include `Impact`, `Why`, and `What Changed` sections so the
changelog explains the operator-facing effect of the release, why the change
was worth shipping, and what concrete behavior or documentation changed. The
semantic commit buckets remain as an audit index, not the only release text.

### AD-099: Release Notes Explain Impact, Why, And What

Release notes are product communication, not only a commit digest. Each
generated release entry must provide complete user-facing text for:

- `Impact`: who or what is affected, including operator, agent, maintainer,
  target-repo, compatibility, reliability, or evidence impact.
- `Why`: the reason the change matters, the failure mode or capability gap it
  closes, or the operating-model value it preserves.
- `What Changed`: the concrete change that landed, with commit references for
  traceability.

Commit bodies may include `Impact:`, `Why:`, and `What:` lines for richer
release text. When those fields are absent, the generator produces conservative
fallback prose from semantic commit type, scope, and message. Structural
delivery changes use stronger topic-aware fallback profiles, so operating-model,
structured dispatch, persona, documentation-sync, and CLI/tool-sync releases
explain the workflow shift instead of repeating a thin commit subject.

### AD-100: Historical Release Notes Are Backfilled Through The Release Tool

Release-note standards apply to the whole changelog, not only the newest entry.
When the narrative standard changes, maintainers use `mars-harness release
backfill-notes` to rewrite existing marker-backed entries from their historical
commit ranges. The command treats each `mars-harness-release` marker as the
authoritative release boundary, ignores `release:` commits, preserves existing
semantic buckets and delivery evidence, and replaces legacy narrative sections
with generated `Impact`, `Why`, and `What Changed` prose. If old git topology is
non-linear, the tool falls back to the commit hashes already present in that
entry's semantic buckets.

The command supports `--dry-run` for review, `--check` for docs-consistency and
CI gates, and `--min-version` / `--max-version` for bounded historical batches.
If a marker commit is missing or no non-release commits can be found for an
entry, the command fails with an actionable error instead of inventing release
history.

### AD-051: Source And Target Release Behavior Mirrors

`mars-harness init` creates `VERSION`, `CHANGELOG.md`, release-versioning design guidance, and release knowledge routes in target repos. The default release-manager prompt uses the same `mars-harness release notes --repo . --bump auto` command that Mars Harness itself uses.

### AD-056: Source Changes Are Automatically Versioned On Commit

Every non-release semantic commit to this source repository must be followed by an automatic release-note generation step before the task is considered done:

1. commit the coherent source/doc/test change
2. run `mars-harness release notes --repo . --bump auto`
3. verify the generated `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go` changes
4. commit them as `release: notes X.Y.Z`
5. push `main`

The `release: notes X.Y.Z` commit itself is exempt. The release generator ignores release-note commits so the workflow does not create an infinite version loop.

### AD-057: Target Harnesses Inherit Automatic Versioning

Initialized target repositories use the same operating rule. `mars-harness init` writes target `AGENTS.md`, `docs/design-docs/release-versioning.md`, and the release-manager prompt so every non-release semantic commit in the target repo is followed by:

1. `mars-harness release notes --repo . --bump auto`
2. verification of generated `VERSION` and `CHANGELOG.md`
3. a `release: notes X.Y.Z` commit
4. push to `main`

Target repos do not have `internal/buildinfo/version.go` unless their own project defines one. The mirrored rule is the workflow contract, not a requirement for target repos to copy Mars Harness internals.

### AD-059: Versioned Releases Are Published To GitHub When Configured

A release is not fully complete until the generated version is visible as a GitHub Release when the repository has authenticated GitHub release capability.

After a `release: notes X.Y.Z` commit is pushed to `main`, release work must:

1. create or update tag `vX.Y.Z` at the release-note commit
2. push the tag so the release workflow builds binary assets and publishes or
   updates GitHub Release `vX.Y.Z`
3. use the generated `CHANGELOG.md` entry for `X.Y.Z` as the release notes body
4. verify the release is visible in GitHub and `mars-harness release
   verify-assets --version vX.Y.Z` passes

GitHub remains optional infrastructure. If the repo has no GitHub remote, no authenticated release credentials, or the GitHub API fails, the release manager records the blocker and leaves a follow-up ticket instead of claiming the release is complete.

### AD-068: The Installed Command Can Update Itself

Operators should not need to `cd` into the source repository to upgrade the built binary. The installed `mars-harness` command owns its own update surface through `mars-harness update tool`.

The packaged-user path downloads the matching `mars-harness-{os}-{arch}` release
asset, verifies `checksums.txt`, and atomically replaces the binary in the
directory containing the currently running command. This avoids requiring Go or a
source checkout for ordinary upgrades.

Private release repositories use the same checksum-verified path through the
Getting Started private release auth operating model. Operators run
`mars-harness auth github setup` once, then `mars-harness update tool` resolves
auth in this order: `GH_TOKEN`, `GITHUB_TOKEN`, GitHub CLI auth from
`gh auth token`, then an optional local token stored under `~/.mars-harness/`
with owner-only permissions. Token values are never printed, written to target
repos, or included in traces, telemetry, doctor output, JSON, errors, tickets,
or docs.

When release metadata exposes GitHub asset API URLs, the updater downloads from
those authenticated API URLs instead of browser download URLs so private assets
do not fail behind a misleading 404. Missing or invalid auth points to
`mars-harness auth github setup`; headless installs may use `GH_TOKEN`,
`GITHUB_TOKEN`, or `mars-harness auth github setup --token <token>`.

`mars-harness setup` includes a private-release auth check before ordinary
runtime setup. `--skip-github` and `--test-mode` skip that external auth check.
`mars-harness doctor` reports private-release auth readiness with a concrete
fix, and agents can use the read-only `github_auth_check` tool before update,
release verification, install repair, or version-drift remediation.

Source-development updates remain available through `mars-harness update tool
--source` or `mars-harness update tool --version main`. That path uses `go
install github.com/greaveselliott/mars-harness/cmd/mars-harness@<version>` with
`GOBIN` set to the install directory.

### AD-069: Update Is The Unified Verb For Tool And Deployed Harness Changes

The CLI should use the same language when the goal is the same. "Update" means bring an installed or deployed harness surface up to the current expected version:

- `mars-harness update tool` updates the installed CLI binary.
- `mars-harness update harness --repo <path>` updates the `.harness/` bundle deployed into a target repo.
- `mars-harness upgrade --repo <path>` remains as a compatibility alias for target harness updates while docs migrate to `update harness`.

### AD-070: Update Check Detects Tool And Target Harness Drift

`mars-harness update check --repo <path>` compares both update surfaces before mutating anything:

- the installed CLI version against the latest GitHub release, or another GitHub-compatible latest-release endpoint supplied by the operator
- the target repo's generated `.harness/metadata.yaml` generator version against the installed CLI version

The command emits machine-readable JSON with `--json` and recommends `update tool`, `update harness`, or both. Remote lookup failures are reported as `unknown` for the tool and do not prevent local target-harness checks. `mars-harness doctor --repo <path>` includes the same drift signal as a warning so operators see stale binary or target harness state during health checks.

`mars-harness init` and `mars-harness update harness` write `.harness/metadata.yaml`. This file is generated state owned by the harness updater, unlike role prompts, manifests, guardrails, knowledge routes, tickets, and docs, which remain user-owned after init.

### AD-075: Install And Update Configure User Shell PATH

Operators should not need shell-specific setup knowledge to run the installed
command. Source `make install`, `mars-harness setup`, and `mars-harness update
tool` converge on the same shell-path configurator. It detects Fish, Zsh, Bash,
POSIX sh/Ksh, Csh, and Tcsh; writes an idempotent user-profile snippet; and
reports unsupported shells with the install directory to add manually.

The explicit command is:

```bash
mars-harness path setup --install-dir <dir>
```

`make install` runs that command through the freshly installed binary using its
absolute path, so the first source install fixes Fish/Zsh/Bash/POSIX PATH for
future terminals. `update tool` repeats the same setup after reinstalling the
binary. `setup` includes the same step so binary-release installers can call one
setup flow later.

The command cannot mutate the already-running parent shell process after it
exits. It prints a reload hint for the current session and makes new terminals
work without manual profile editing.

### AD-078: Release Assets Are Built From Tags And Verified

For the source harness, `git tag vX.Y.Z && git push origin vX.Y.Z` is the
authoritative release-publication trigger after the release-note commit is on
`main`. Direct `gh release create` publication can still create a notes-only
release, so it is not the default source harness release path and is incomplete
until release assets are attached and verified.

The Release workflow cross-compiles `linux/darwin` x `amd64/arm64`, writes
`checksums.txt`, verifies all expected assets before publication, and uses the
matching `CHANGELOG.md` entry as the GitHub Release body. It runs on pushed
version tags, on newly published GitHub Releases to recover from notes-only
publication, and through `workflow_dispatch` with a `version` input for manual
backfills. Release managers must run `mars-harness release verify-assets
--version vX.Y.Z` after publication before claiming the installer or self-update
path is shipped.

## Implementation Requirements

- Add `mars-harness release notes --repo <path> --bump auto|major|minor|patch [--dry-run]`.
- Add `mars-harness release backfill-notes --repo <path> [--min-version X.Y.Z] [--max-version X.Y.Z] [--dry-run] [--check]`.
- Add `mars-harness update tool [--version <version>] [--install-dir <path>] [--dry-run]`.
- Add `mars-harness release verify-assets [--version <tag>]`.
- Add `mars-harness update harness --repo <path>`.
- Add `mars-harness update check --repo <path> [--json] [--skip-remote]`.
- Add `mars-harness path setup [--install-dir <path>]`.
- Infer `auto` bumps from semantic commits:
  - breaking changes -> major
  - `feat:` -> minor
  - other documented changes -> patch
- Preserve strict trunk: generated version and patch-note changes are committed directly to `main` and pushed after verification.
- Ignore release-note commits when generating later patch notes.
- Start each generated changelog entry with complete plain-English `Impact`, `Why`, and `What Changed` sections before semantic commit buckets.
- Keep all historical changelog entries on the same narrative standard through `release backfill-notes --check`.
- Update the source harness fallback version from a repo-owned constant.
- Generate the same VERSION/CHANGELOG/release guidance in target repos.
- Treat source-repo versioning as part of done for every non-release semantic commit.
- Treat target-repo versioning as part of done for every non-release semantic commit after `mars-harness init`.
- Publish or update matching GitHub Releases when authenticated GitHub release capability is configured.
- Let the installed binary reinstall itself without requiring a source checkout.
- Verify release assets before announcing installer or self-update availability.
- Use the same update vocabulary for binary and deployed target harness updates.
- Record generated target harness version in `.harness/metadata.yaml`.
- Check version drift without mutating the installed tool or target repo.

## Consequences

- Version and patch-note state is visible to agents and humans.
- Historical changelog entries can be upgraded deterministically instead of hand-edited in bulk.
- Target projects get release discipline without extra configuration.
- Release Manager work becomes deterministic before it becomes judgment work.
- Source-repo work cannot silently land without an accompanying semantic version and patch-note entry.
- Target repos inherit the same release discipline without extra setup.
- GitHub users see versioned release notes in the GitHub Releases UI, while local-only users still have repo-owned `VERSION` and `CHANGELOG.md`.
- Future work can add tag creation, release publishing, release-asset self-update, and doctor checks for stale patch notes.
