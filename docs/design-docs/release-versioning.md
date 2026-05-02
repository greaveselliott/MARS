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

## Implementation Requirements

- Add `mars-harness release notes --repo <path> --bump auto|major|minor|patch [--dry-run]`.
- Infer `auto` bumps from semantic commits:
  - breaking changes -> major
  - `feat:` -> minor
  - other documented changes -> patch
- Preserve strict trunk: generated version and patch-note changes are committed directly to `main` and pushed after verification.
- Ignore release-note commits when generating later patch notes.
- Update the source harness fallback version from a repo-owned constant.
- Generate the same VERSION/CHANGELOG/release guidance in target repos.
- Treat source-repo versioning as part of done for every non-release semantic commit.

## Consequences

- Version and patch-note state is visible to agents and humans.
- Target projects get release discipline without extra configuration.
- Release Manager work becomes deterministic before it becomes judgment work.
- Source-repo work cannot silently land without an accompanying semantic version and patch-note entry.
- Future work can add tag creation, release publishing, and doctor checks for stale patch notes.
