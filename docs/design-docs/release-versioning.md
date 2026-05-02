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
2. publish or update GitHub Release `vX.Y.Z`
3. use the generated `CHANGELOG.md` entry for `X.Y.Z` as the release notes body
4. verify the release is visible in GitHub

GitHub remains optional infrastructure. If the repo has no GitHub remote, no authenticated release credentials, or the GitHub API fails, the release manager records the blocker and leaves a follow-up ticket instead of claiming the release is complete.

### AD-068: The Installed Command Can Update Itself

Operators should not need to `cd` into the source repository to upgrade the built binary. The installed `mars-harness` command owns its own update surface through `mars-harness update tool`.

The first implementation uses `go install github.com/greaveselliott/mars-harness/cmd/mars-harness@<version>` with `GOBIN` set to the directory containing the currently running binary. This supports source-development and Go-installed workflows immediately. It also avoids the stale source-root binary trap where `go build; ./mars-harness ...` can run an old binary after a failed build.

Release-asset self-updates remain the desired packaged-user path: download the matching OS/arch binary, verify `checksums.txt`, and atomically replace the installed executable. That depends on the release asset contract tracked separately by MH-031.

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

## Implementation Requirements

- Add `mars-harness release notes --repo <path> --bump auto|major|minor|patch [--dry-run]`.
- Add `mars-harness update tool [--version <version>] [--install-dir <path>] [--dry-run]`.
- Add `mars-harness update harness --repo <path>`.
- Add `mars-harness update check --repo <path> [--json] [--skip-remote]`.
- Infer `auto` bumps from semantic commits:
  - breaking changes -> major
  - `feat:` -> minor
  - other documented changes -> patch
- Preserve strict trunk: generated version and patch-note changes are committed directly to `main` and pushed after verification.
- Ignore release-note commits when generating later patch notes.
- Update the source harness fallback version from a repo-owned constant.
- Generate the same VERSION/CHANGELOG/release guidance in target repos.
- Treat source-repo versioning as part of done for every non-release semantic commit.
- Treat target-repo versioning as part of done for every non-release semantic commit after `mars-harness init`.
- Publish or update matching GitHub Releases when authenticated GitHub release capability is configured.
- Let the installed binary reinstall itself without requiring a source checkout.
- Use the same update vocabulary for binary and deployed target harness updates.
- Record generated target harness version in `.harness/metadata.yaml`.
- Check version drift without mutating the installed tool or target repo.

## Consequences

- Version and patch-note state is visible to agents and humans.
- Target projects get release discipline without extra configuration.
- Release Manager work becomes deterministic before it becomes judgment work.
- Source-repo work cannot silently land without an accompanying semantic version and patch-note entry.
- Target repos inherit the same release discipline without extra setup.
- GitHub users see versioned release notes in the GitHub Releases UI, while local-only users still have repo-owned `VERSION` and `CHANGELOG.md`.
- Future work can add tag creation, release publishing, release-asset self-update, and doctor checks for stale patch notes.
