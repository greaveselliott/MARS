---
id: T-075
title: Make repository paths descriptor-safe and scan exact Git index blobs
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002"]
end_to_end_evidence: required
evidence_links: ["commit:f9993b5941e2fcd4f8e77866526f8a9b81946d3f", "commit:b3b5b9808e001491da793e515cb71444655dbf22", "commit:88f7737bf9be3b804483f676507a193f68ffa7d4", "commit:e30f207c9edc00a42a28b4d31b9ea5e52dba8a08"]
verified_by: "Checkpoints A, B, and C1: QA, Security, Release Manager, and Orchestrator on 2026-08-09; T-075 closure pending Dogfood and final sign-off"
owner: "foundation-maintainer"
last_attempt: "2026-08-09: Checkpoint C1 passed through fixture repair 88f7737 and repository-writer containment e30f207; Checkpoint C2 is current"
blocker: "none"
blocked_by: []
trace_id: "launch-repository-boundary:2026-08-09"
next_action: "Implement and validate Checkpoint C sub-checkpoint 2 only: migrate target init, upgrade, generated-target writes, and eject through repofs."
dedupe_key: "open-source:repository-path-and-index-secret-boundary"
metadata:
  classification: "foundation-owned"
  mutation_authority: "repository-source-tests-docs-only"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  standard_primitive: "go-os-root"
source: MARS Launch-Complete Open-Source Delivery Plan — T-075
created: 2026-08-09
depends_on: [T-074]
---

# T-075: Make repository paths descriptor-safe and scan exact Git index blobs

## Context

T-074 closed the remaining network-entry-point slice. F-017-S002 remains current because universal file tools and several fixed repository writers use lexical path checks followed by ordinary os operations, so a hostile repository symlink can redirect reads or writes outside the selected repo. The staged secret scanner enumerates Git index path names but reads worktree bytes, allowing partially staged, index-only, force-added, or tracked ignored secrets to pass falsely. The repository remains private and the launch version freeze remains in force.

## Outcome

Use Go's standard-library os.Root as one thin descriptor-bound repository filesystem, reject observed symlink parents and leaves, migrate every named model/agent-controlled repository path, and make CLI and pre-commit secret scanning inspect the exact publishable Git index blobs. Do not build a custom openat framework, scanner runtime, VM, container, generalized policy engine, or race laboratory.

## Checkpoint A — Standard Repository Root And Direct File Tools

- Add a small internal repofs wrapper around os.OpenRoot. It supports read, read inventory, stat/lstat, mkdir, exclusive create, same-directory atomic write, chmod, rename, remove, and the bounded recursive removal required by eject.
- Reject empty, absolute, traversal, NUL, and observed symlink-parent/leaf names. os.Root supplies descriptor containment when a path changes concurrently.
- Atomic replacement uses a cryptographically named same-parent exclusive temporary file, exact mode, file sync/close, descriptor-relative rename, and parent-directory sync; clean the temporary entry on failure.
- Route tools.Root plus file_read, file_write, file_search, and grep through it.
- Prove ordinary operations, traversal rejection, parent/leaf symlink rejection, unchanged external sentinels, and atomic output/mode without an exhaustive kernel/race matrix.
- Commit and push this checkpoint before further migration.

Checkpoint A passed at exact commit `f9993b5941e2fcd4f8e77866526f8a9b81946d3f`.
The retained `os.Root` descriptor stays bound when the admitted repository path
is renamed and replaced, direct file-tool parent/leaf symlink fixtures leave
external sentinels unchanged, focused normal/race tests and vet pass,
`internal/repofs` coverage is 74.3% against a 70% floor, and
documentation-consistency/DocSync pass. QA, Security, Release Manager, and the
Orchestrator returned GO. This is checkpoint evidence only; the ticket and
F-017-S002 remain incomplete.

## Checkpoint B — Exact Git-Index Secret Coverage

- Add one shared thin scan seam used by both the CLI and pre-commit/tool policy.
- For staged content, reconcile NUL-delimited changes with stage-0 index path/OID entries and scan exact bytes from git cat-file. Include added, modified, copied, and renamed destinations plus tracked or force-added .harness/.env.local.
- Reconcile a staged deletion as having no resulting publishable index blob; never fall back to a replacement worktree file or block removal by rescanning the deleted HEAD secret.
- Full mode scans every tracked index blob, then dirty tracked worktree bytes and ordinary untracked non-ignored regular files through repofs. Skip .harness/.env.local only when it is genuinely untracked and ignored.
- Blob/read/unsupported-index failures remain blocked, never clean. Findings expose only repository-relative path, line, broad pattern, and [REDACTED], never candidate bytes or candidate-derived hashes.
- Focused temp-Git tests cover staged/worktree mismatch, index-only content, force-added and tracked local env, rename, deletion reconciliation, ignored untracked local env, injected read failure, and value-free text/JSON/errors.
- Commit and push this checkpoint.

Checkpoint B passed at exact commit
`b3b5b9808e001491da793e515cb71444655dbf22`. The shared CLI and tool-policy
scanner reads raw stage-0 blobs with Git replacement objects disabled,
reconciles rename destinations and deletion tombstones, scans tracked and
force-added local credentials, rejects nested worktree roots, and covers
present `assume-unchanged` and `skip-worktree` entries. Focused normal/race
tests, vet, documentation-consistency, DocSync, the staged scan, and the real
full-repository scan pass with zero findings. QA, Security, Release Manager,
and the Orchestrator returned GO. This remains checkpoint evidence only;
Checkpoint C, Checkpoint D, T-075, and F-017-S002 remain incomplete.

## Checkpoint C — Migrate Named Repository Surfaces

Deliver independently green sub-checkpoints:

1. Migrate ticket_create, tool_create, persona_create, workspace hygiene, record-decision/learnings, and other repo-backed internal/tools writers.
2. Migrate target init, upgrade, generated-target writes, and eject; contained removal rejects symlink parents/leaves.
3. Migrate model override and .harness credential-route readers/writers, bundle/context reads that can enter model input, release notes/backfill writers, and the remaining named repository-backed integration writer surfaces.

Sub-checkpoint 1 passed through exact commits
`88f7737bf9be3b804483f676507a193f68ffa7d4` and
`e30f207c9edc00a42a28b4d31b9ea5e52dba8a08`. Ticket, tool, persona,
workspace-hygiene, record-decision, and learnings persistence now use the
admitted repository descriptor; new files are exclusive, replacements are
atomic and preserve existing modes, and Git subprocesses verify repository
identity before and after execution. Focused `internal/learnings` and
`internal/tools` normal/race tests, package vet, the exact 26-case Git-admission
fixture regression, formatting, and diff checks pass. QA, Security, Release
Manager, and Orchestrator returned GO. Sub-checkpoint 2 is current;
sub-checkpoint 3, Checkpoint D, T-075, and F-017-S002 remain incomplete.

Use atomic replacement or exclusive creation as appropriate and preserve existing behavior and owner-only credential mode. Do not mediate shell_exec, Git's own internal writes, global MARS state, model downloads, databases, logs, traces, self-update installation, or validation-output directories; those are outside this repository boundary or owned by T-076/T-077.

## Checkpoint D — Closure

- Perform one mechanical source inventory proving no direct repo-root os.ReadFile/Open/WriteFile/MkdirAll/Rename/Remove remains in the named production surfaces. Record the result as evidence; do not add a permanent generalized linter.
- Sync only F-004, F-005, F-007, F-017, guardrails/tools guidance, mirrored CLI/generated doctrine, the active plan, goal, and this ticket where behavior changes.
- Run focused normal/race tests, docs-consistency and DocSync, four supported CGO-disabled builds, and one installed clean-target smoke proving symlink read/write rejection, exact index-only/force-added secret blocking with no value leakage, and ordinary ignored untracked .harness/.env.local omission.
- Obtain concurrent QA/Security review plus Dogfood and Release Manager sign-off.

## Acceptance

- No named repository read, create, atomic write, mkdir, chmod, rename, remove, stat, init, upgrade, eject, ticket/release writer, credential-route reader/writer, or generated-target operation can follow a hostile repository symlink outside the selected descriptor-bound root.
- Staged scanning uses exact index blob bytes; tracked, staged, force-added, renamed, and index-only content cannot be substituted by worktree bytes or skipped. Deletions are explicitly reconciled as tombstones with no resulting blob.
- Scanner errors and unsupported index states never report clean, and no candidate value appears in text, JSON, errors, logs, or traces.
- Darwin/Linux focused tests, cross-builds, installed smoke, DocSync, QA, Security, Dogfood, Release Manager, and Orchestrator pass.
- T-075 closes only the repository-path and secret-scan slice. F-017-S002 remains incomplete pending T-076 and resumed T-058; Primary Status remains primary_blocked.

## No-Go

Any external sentinel read or mutation, staged blob substitution, tracked/force-added .harness/.env.local exemption, silent scan skip/error, candidate-value rendering, custom audit/runtime/kernel-lab expansion, shell-containment claim, version/tag/Release/settings/visibility/announcement mutation, or claim that F-017-S002 is complete.
