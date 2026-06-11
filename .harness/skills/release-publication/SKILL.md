# Release Publication

Use this in the foundation harness when a non-release semantic commit has
landed and Release Manager or Codex must publish the matching source release.
This is source-only procedural guidance; target repos keep their generated
release docs unless they explicitly define their own binary publication flow.

## Workflow

1. Confirm the semantic commit is coherent and verified.
2. When running as a harness agent, use `mars_harness_cli` with args
   `["release","notes","--repo",".","--bump","auto"]`; when operating from a
   trusted terminal, run the equivalent `mars-harness release notes --repo .
   --bump auto`.
3. Review `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go`.
4. Run `go test ./internal/docsconsistency ./internal/docsync`, plus any
   package tests touched by the semantic commit.
5. Commit the generated files as `release: notes X.Y.Z`.
6. Push the release-note commit to `origin main` and the active working branch.
7. Create or update tag `vX.Y.Z` at the release-note commit and push it. Do
   not tag while `VERSION` or `CHANGELOG.md` are dirty, and do not target a
   pre-release-note commit.
8. As a harness agent, use `mars_harness_cli` with args
   `["release","publish-assets","--repo",".","--version","vX.Y.Z","--upload","auto"]`;
   from a trusted terminal, run the equivalent `mars-harness release
   publish-assets --repo . --version vX.Y.Z --upload auto`.
9. Verify local assets with `mars_harness_cli` args
   `["release","verify-assets","--dist","dist/releases","--version","vX.Y.Z"]`
   or `mars-harness release verify-assets --dist dist/releases --version
   vX.Y.Z`.
10. If GitHub mirroring was enabled, verify `gh release view vX.Y.Z --repo
    greaveselliott/mars-harness` and then run `mars-harness release
    verify-assets --version vX.Y.Z`.
11. Audit the mirror for drift across recent versions, not just the one you
    published: `mars_harness_cli` args `["release","audit","--repo","."]` or
    `mars-harness release audit --repo .`. Each finding names a notes-only or
    missing release with the exact `publish-assets` backfill command; run the
    backfill or record the blocker.
12. If local assets, the optional mirror, or the audit findings are
    unresolved, record the blocker in the active plan before moving to
    unrelated work.

## Token Safety

- Use `mars-harness auth github check` or `gh auth status` when optional
  GitHub mirroring needs credentials.
- Never paste token values into chat, docs, commits, traces, tickets, logs, or
  tool output.
- Prefer GitHub CLI auth, `GH_TOKEN`, or `GITHUB_TOKEN`; local stored tokens
  stay under `~/.mars-harness/`.

## Stop Conditions

- Stop and record a blocker if `git push`, tag push, local asset publication,
  local asset verification, GitHub release creation, or GitHub mirror
  verification fails.
- A notes-only GitHub Release satisfies the optional mirror object gate only.
  The local dist remains the source of truth; the mirror stays incomplete until
  GitHub `verify-assets` passes.
- Do not start another semantic change while the release-note commit, pushed
  tag, release object, or missing-asset blocker is unrecorded.

## Evidence

Record the release version, pushed commit, pushed tag, local dist path, local
`verify-assets` result, optional `gh release view` result, optional GitHub
`verify-assets` result, and any local build or GitHub API blocker in the active
plan or owning ticket.
