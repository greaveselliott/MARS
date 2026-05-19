# Release Publication

Use this in the foundation harness when a non-release semantic commit has
landed and Release Manager or Codex must publish the matching source release.
This is source-only procedural guidance; target repos keep their generated
release docs unless they explicitly define their own binary publication flow.

## Workflow

1. Confirm the semantic commit is coherent and verified.
2. Run `mars-harness release notes --repo . --bump auto`.
3. Review `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go`.
4. Run `go test ./internal/docsconsistency ./internal/docsync`, plus any
   package tests touched by the semantic commit.
5. Commit the generated files as `release: notes X.Y.Z`.
6. Push the release-note commit to `origin main` and the active working branch.
7. Create or update tag `vX.Y.Z` at the release-note commit and push it.
8. Verify `gh release view vX.Y.Z --repo greaveselliott/mars-harness`.
9. If the release object is missing but the tag exists, create a notes-only
   GitHub Release from the generated `CHANGELOG.md` entry.
10. Run `mars-harness release verify-assets --version vX.Y.Z`.
11. If assets are missing, record the blocker in the active plan before moving
    to unrelated work.

## Token Safety

- Use `mars-harness auth github check` or `gh auth status` when release or
  asset verification needs GitHub credentials.
- Never paste token values into chat, docs, commits, traces, tickets, logs, or
  tool output.
- Prefer GitHub CLI auth, `GH_TOKEN`, or `GITHUB_TOKEN`; local stored tokens
  stay under `~/.mars-harness/`.

## Stop Conditions

- Stop and record a blocker if `git push`, tag push, GitHub release creation,
  or asset verification fails.
- A notes-only GitHub Release satisfies the release-object gate only. It is not
  a complete binary release until `verify-assets` passes.
- Do not start another semantic change while the release-note commit, pushed
  tag, release object, or missing-asset blocker is unrecorded.

## Evidence

Record the release version, pushed commit, pushed tag, `gh release view`
result, `verify-assets` result, and any workflow run or GitHub API blocker in
the active plan or owning ticket.
