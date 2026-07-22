---
id: T-068
title: Rehearse the GoReleaser pipeline privately without publication
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-018-S003"]
end_to_end_evidence: required
evidence_links: ["docs/features/F-018-goreleaser-distribution.md#f-018-s003-private-rehearsal-without-publication", "docs/exec-plans/active/current-operating-plan.md"]
verified_by: "QA, Security, Dogfood, Release Manager, and foundation-maintainer as Orchestrator"
owner: "engineer"
last_attempt: "2026-07-22: exact-head run 29899168382 and final private-state reconciliation passed; checkpoint C closed"
blocker: "none"
blocked_by: []
trace_id: "github-actions:29899168382"
next_action: "Complete; retain publication denial and continue the independent F-017 gates through a separately created ticket."
dedupe_key: "release:goreleaser-private-rehearsal"
metadata:
  classification: "foundation-owned"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  supports: "F-018-S003"
source: current-operating-plan.md — T-068 / F-018-S003
created: 2026-07-22
depends_on: [T-066]
---

# T-068: Rehearse the GoReleaser pipeline privately without publication

## Context

T-065/F-018-S001 proved the publication-disabled producer contract and T-066/F-018-S002 proved the fail-closed consumer contract. Before any real MARS signing, tag, Release, upload, visibility change, or support claim, F-018-S003 requires one bounded private rehearsal of the combined pipeline. The repository stays private at VERSION 0.68.49 and untagged source reports 0.69.0-dev.

## Requirements

1. Freeze one immutable clean source commit and use only the exact pinned Go 1.26.5, GoReleaser, and Syft producer path already admitted by T-065.
2. Build in two clean isolated roots. Require byte-identical archives and archive checksum records, exact per-run binding of all four SBOM bytes, normalized SPDX semantic equality, exact nine-file publishable allowlists, and the existing artifact-contract checker.
3. Exercise the fork-safe snapshot workflow successfully with read-only authority and prove statically and at runtime that tag, sign, attest, announce, publish, draft, upload, and Release authority is absent. The workflow has no draft/release job; do not invoke a dangerous live release path merely to watch it fail.
4. Keep three evidence classes separate: supported clean source setup, direct execution of a contract-verified but unsigned native snapshot archive, and offline synthetic/preverified consumer update plus rollback. Run the native snapshot binary on isolated macOS and ephemeral Ubuntu environments. Fresh packaged bootstrap remains fail-closed; never describe an unsigned snapshot as a supported packaged install or update.
5. Run release-note generation in dry-run mode and require exactly 0.69.0 without changing VERSION, CHANGELOG, source fallback, refs, Releases, Pages, visibility, or announcements.
6. Record redacted commands, hashes, environment identities, outcomes, cleanup, and persona sign-offs in the ticket, F-018, active plan, and goal. Do not commit raw credentials, signatures, temporary paths, scanner output, or synthetic secret values.

## Checkpoint sequence

A. Add only the smallest fork-safe Ubuntu workflow/test seam needed to create two distinct clean clones and dist roots, preflight the existing config/workflow/Syft contract tests, run the exact publication-disabled producer twice, compare both outputs, then execute the contract-verified native Linux archive, fresh source/archive target fixtures, fail-closed installer, and focused offline T-066 consumer/rollback tests. Touch only `.github/workflows/release-snapshot.yml` and its existing contract test unless a concrete failure proves otherwise. Commit/push the seam, dispatch it at that exact SHA, upload nothing, and commit redacted run/state evidence before macOS work.

B. On the owner Darwin/arm64 host, use a fresh clean clone plus isolated HOME/GOBIN/TMP/GOCACHE to run one exact pinned snapshot, the contract verifier and native archive, supported source setup, separate fresh source/archive targets, authority-free update dry-run, fail-closed installer, and the same offline consumer/rollback fixtures. Use the immutable checkpoint-A source commit and call the unsigned archive execution fixture evidence, not a release installation.

C. Require release-note dry-run to select exactly `0.69.0`, reconcile refs/Releases/visibility/worktrees and the frozen state manifest, delete only manifest-listed T-068 temporary storage, collect QA/Security/Dogfood/Release Manager sign-offs, and close T-068/F-018-S003 in a separate evidence commit.

Each checkpoint is independently green, committed, and pushed before the next. No new VM, container, trust system, signer, generalized verifier, or publication framework belongs in this ticket.

## Checkpoint A Evidence — 2026-07-22

- Commit `aa4a16bb5d26bcb766851dec375149b906fa6ce8` is pushed and exact-head GitHub Actions [run 29894376197](https://github.com/greaveselliott/MARS/actions/runs/29894376197) passed on Ubuntu 24.04. Job `88841325303` ran from `2026-07-22T05:38:59Z` through `2026-07-22T05:55:48Z`; preflight, pinned-tool construction, the combined two-root rehearsal, cleanup, and post-job steps all passed.
- Exact Go `1.26.5`, GoReleaser `v2.17.0`, and Syft `v1.49.0` produced two clean snapshot roots. The committed verifier accepted the archive/checksum/SBOM reproducibility contract; supported source installation and direct execution of the contract-verified unsigned native Linux snapshot each initialized a clean equivalent target and passed DocSync; the authority-free update dry-run and five focused offline consumer/install/update/rollback tests passed.
- Earlier runs were rejected diagnostic evidence only: `29891275429`/`7904b43` failed closed on missing clone identity plus read-only-cache cleanup; `29892244613`/`6437f81` failed closed on target HOME/TMP fixture collision; and `29893274768`/`b8be85f` failed closed because different fixture leaf names produced repository-bound metadata differences. Commits `6437f81`, `b8be85f`, and `aa4a16b` corrected those harness defects without accepting a partial run. Each rejected run uploaded zero Actions artifacts.
- The accepted run uploaded zero Actions artifacts. Reconciliation found `origin/main` exactly at `aa4a16b`, repository visibility `PRIVATE`, and no `v0.69.0` tag or Release. `VERSION` remains `0.68.49` and the source fallback remains `0.69.0-dev`.
- This accepts checkpoint A only at this evidence boundary. Checkpoint B is accepted separately below; release-note/state/cleanup checkpoint C, final persona sign-offs, F-018-S003, and every public-cutover gate remain pending.

## Checkpoint B Evidence — 2026-07-22

- A fresh owner-host rehearsal at immutable commit `aa4a16bb5d26bcb766851dec375149b906fa6ce8` passed on macOS `26.3.1` (Darwin `25.3.0`)/arm64 with exact Go `1.26.5` and snapshot version `0.69.0-dev.aa4a16b`. The pinned producer, committed artifact verifier, supported source install, direct unsigned native archive execution, separate equivalent source/archive targets, authority-free update dry-run, and the exact five offline consumer/install/update/rollback tests all passed.
- The accepted artifact identities were archive SHA-256 `b9f73f666d66ed7100a6c126b677541f5fea33ac8e7ffa36edcfe2b42a9ad460`, native binary SHA-256 `8530158a8f9df3799627efee3df51dafb3e71c578737cef13f24f572ada34bf9`, source binary SHA-256 `ba227b4d4eb98d6e190ab43fb57b1a42f3b8380db065d5a8762bc6714ce054e7`, and equivalent target tree `e3f3e0ee21b751bfa7a183717be48a76a9134a4b`.
- Two earlier owner-host attempts remained rejected harness evidence: the first compared the verifier's UTC build time with a local-offset timestamp, and the second placed replacement fixtures beneath a world-writable temporary ancestor that production correctly rejected. Neither produced accepted evidence or repository mutation.
- The final functional run reached and emitted `checkpoint_b=pass`, then its aggregate shell exited `1` because a best-effort recursive `chmod` warned on read-only module-cache entries during cleanup. An independent postcondition proved the exact rehearsal root absent and the host repository clean. Engineer, QA, and Security classify this as a cleanup-status false negative rather than a product or residue failure and require no producer rerun; this record does not claim that the aggregate wrapper exited zero.
- Checkpoint B is accepted only as private source and unsigned-snapshot fixture evidence. Checkpoint C, fresh packaged bootstrap, final sign-offs, F-018-S003, and all public-cutover gates remain pending; no version, tag, Release, signature, upload, visibility, or support claim changed.

## Checkpoint C Evidence — 2026-07-22

- Exact Go `1.26.5` ran `mars release notes --repo . --bump auto --dry-run` and selected `0.68.49 -> 0.69.0` (minor) without writing the repository. HEAD and `origin/main` remained `742b133e64493d6350779fb74ef0409283a2a5cc`, the tree remained `b8def771989cb8f1a081f26847dc80a38ef6b110`, and `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go` retained their pre-run hashes.
- The exact 22 manifest-listed T-068 temporary roots were owner-owned, non-symlinked, free of open descriptors and special nodes, and then removed. Their manifest digest was `bd4600f84310cc2a4e1bdfdc4aeff668054d911b920b5f66370f3943d3fb5e25`; allocated removal was 4,322,880 KiB (4.123 GiB), and manifest-scoped verification found all 22 absent. Shared caches, unrelated temporary data, both protected stashes, and the unrelated Codex worktree were preserved.
- Checkpoint C then failed closed on exact-head source-compatibility run `29896978096`: both supported Linux lanes failed while the below-minimum lane passed. T-069 corrected only those portability failures; run `29898672813` passed Go `1.25.12`, Go `1.26.5`, and the expected Go `1.25.11` rejection at `03008f74477ba1b5fa8ea85c3af55e448ea037ac`.
- At final evidence head `2ef9d277f60baca2123ecea7c460ffdb49d018eb`, exact-head run `29899168382` passed the same three jobs and retained zero Actions artifacts. Exact Go `1.26.5` again selected `0.68.49 -> 0.69.0` (minor, 48 commits) in dry-run mode; HEAD/origin remained exact, tree `7f63e4c901a117a7adc08e85e031c8ed392904cd` and the three release-state file hashes were unchanged.
- Final state reconciliation found one private branch `main`, 301 retained tags, no `v0.69.0` tag or Release, disabled Pages, `VERSION=0.68.49`, source fallback `0.69.0-dev`, both protected stashes intact, and the unrelated Codex worktree untouched. The REST collection enumerates 56 Releases; direct lookup and the GraphQL/`gh` inventory additionally expose intentionally retained, non-draft, non-prerelease, zero-asset `v0.65.7` Release ID `344415010`, so 57 retained Release objects are addressable. This endpoint discrepancy is not an unknown object or deletion target.
- Final Security discovery found two additional T-068 reviewer roots totaling 426,200 KiB—one omitted pre-existing root and one later dry-run cache—plus ten ticket-specific T-069 Go caches. All 12 owner-created roots were removed by exact path; final bounded discovery found no T-068/T-069 rehearsal residue while shared caches and unrelated data remained untouched.
- QA, Security, Dogfood, Release Manager, and Orchestrator are GO for T-068/F-018-S003 only. Fresh packaged bootstrap, notices, recorded producer vulnerabilities, the independent history/privacy/IP audit, runtime/filesystem/execution gates, signed anonymous publication, contribution governance, visibility approval, logged-out cutover smoke, recovery, and the 48-hour canary remain blocked. No version, tag, Release, signature, upload, visibility, announcement, or supported-release authority was exercised.

## Acceptance criteria

- [x] Two independent clean builds pass the exact producer contract and archive reproducibility standard.
- [x] Isolated supported source setup and separate unsigned native snapshot execution pass on macOS; the exact-SHA ephemeral Ubuntu run passes the same source/native-target split plus focused offline consumer/update/rollback fixtures. These are distinct from packaged bootstrap and a real signed update.
- [x] The snapshot workflow succeeds with `contents: read`, non-persisted checkout authority, exact skip flags, and no later build/test credentials, signing/OIDC, upload, draft, tag, or Release authority.
- [x] Release-note dry-run selects exactly 0.69.0 and leaves the repository byte-identical.
- [x] Temporary roots, ticket-specific caches, generated artifacts, and credentials are reconciled or removed without touching shared caches or unrelated user work.
- [x] QA, Security, Dogfood, Release Manager, and Orchestrator sign off; F-018-S003 is marked passed only with exact evidence and Primary Status remains primary_blocked.
