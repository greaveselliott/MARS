---
id: T-068
title: Rehearse the GoReleaser pipeline privately without publication
priority: high
complexity: large
work_type: feature
bdd_scenarios: ["F-018-S003"]
end_to_end_evidence: required
evidence_links: ["docs/features/F-018-goreleaser-distribution.md#f-018-s003-private-rehearsal-without-publication", "docs/exec-plans/active/current-operating-plan.md"]
verified_by: "TBD"
owner: "engineer"
last_attempt: "2026-07-22: COO, CTO-weekly, and Security froze checkpoints A/B/C; the four-surface planning handoff is pushed at 90c4217"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Implement only checkpoint A in release-snapshot.yml and its existing contract test, push the bounded seam, then dispatch that exact SHA with no publication or upload authority."
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

## Acceptance criteria

- [ ] Two independent clean builds pass the exact producer contract and archive reproducibility standard.
- [ ] Isolated supported source setup and separate unsigned native snapshot execution pass on macOS; the exact-SHA ephemeral Ubuntu run passes the same source/native-target split plus focused offline consumer/update/rollback fixtures. These are distinct from packaged bootstrap and a real signed update.
- [ ] The snapshot workflow succeeds with `contents: read`, non-persisted checkout authority, exact skip flags, and no later build/test credentials, signing/OIDC, upload, draft, tag, or Release authority.
- [ ] Release-note dry-run selects exactly 0.69.0 and leaves the repository byte-identical.
- [ ] Temporary roots, ticket-specific caches, generated artifacts, and credentials are reconciled or removed without touching shared caches or unrelated user work.
- [ ] QA, Security, Dogfood, Release Manager, and Orchestrator sign off; F-018-S003 is marked passed only with exact evidence and Primary Status remains primary_blocked.
