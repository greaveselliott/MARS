---
id: T-075
title: Make repository paths descriptor-safe and scan exact Git index blobs
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002"]
end_to_end_evidence: required
evidence_links: ["commit:f9993b5941e2fcd4f8e77866526f8a9b81946d3f", "commit:b3b5b9808e001491da793e515cb71444655dbf22", "commit:88f7737bf9be3b804483f676507a193f68ffa7d4", "commit:e30f207c9edc00a42a28b4d31b9ea5e52dba8a08", "commit:66d7e412f0c0dade49c037752c8fa3f0000ee94e", "commit:c8c28cbcc709e12554236e92b7c2e7ba19006784", "commit:d67b04278db608c5fb39d61d3fa0b54c4909cbed", "commit:f99964e79047b3e71d3076d1a05c75b3df9c4e95", "commit:e08deb4bd118ff025abf131e7db8cf4eeb4cf333", "commit:228d859511fb2f7c93e0162424c2e6dc95107e44", "commit:7578549bdd0dde90857f9652e651832d484abdb2", "commit:ff69aaa1bab3680d169e8889866ba73cccb397c9", "commit:16b5527bbe48e8afea82bb70127d383d6f280ed7", "commit:c18030edb44d1b869a03d30e4339ff457641c6e4", "commit:9ba8156942a584f301888a3675942923739993d6", "commit:56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b", "binary-sha256:c1137731531fded59e600e36ba8f77cd7ef1d6759262ddc223a3b7235831a28f"]
verified_by: "QA, Security, Dogfood, Release Manager, and Orchestrator on 2026-08-09; post-closure test-fixture correction independently reviewed on 2026-08-24"
owner: "foundation-maintainer"
last_attempt: "2026-08-24: post-closure internal/agent fixture drift was corrected test-only at 56b8de3; exact nine and full package normal/race/vet gates passed without production-policy change"
blocker: "none"
blocked_by: []
trace_id: "launch-repository-boundary:2026-08-09"
next_action: "Create T-076 through ticket_create and begin the execution-profile, environment, state, and trace gate."
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
3. Migrate model override and .harness credential-route readers/writers, release notes/backfill controlled-file writers, and the remaining named repository-backed integration writer surfaces.

Sub-checkpoint 1 passed through exact commits
`88f7737bf9be3b804483f676507a193f68ffa7d4` and
`e30f207c9edc00a42a28b4d31b9ea5e52dba8a08`. Ticket, tool, persona,
workspace-hygiene, record-decision, and learnings persistence now use the
admitted repository descriptor; new files are exclusive, replacements are
atomic and preserve existing modes, and Git subprocesses verify repository
identity before and after execution. Focused `internal/learnings` and
`internal/tools` normal/race tests, package vet, the exact 26-case Git-admission
fixture regression, formatting, and diff checks pass. QA, Security, Release
Manager, and Orchestrator returned GO.

Sub-checkpoint 2 passed through exact commits
`66d7e412f0c0dade49c037752c8fa3f0000ee94e` and
`c8c28cbcc709e12554236e92b7c2e7ba19006784`. Init, upgrade, generated-target
mutation, metadata, and workspace-ignore writes retain one admitted descriptor;
automatic Git initialization verifies repository identity before and after the
subprocess. Eject preflights every target before mutation, rejects symlink
parents/leaves in dry-run and apply modes, removes through the descriptor, and
preserves application files and empty-directory pruning behavior. Focused
normal/race tests, scanner vet, formatting, and diff checks pass. QA, Security,
Release Manager, and Orchestrator returned GO. This was sub-checkpoint 2
evidence only.

Sub-checkpoint 3 passed through exact commits
`d67b04278db608c5fb39d61d3fa0b54c4909cbed`,
`f99964e79047b3e71d3076d1a05c75b3df9c4e95`, and
`e08deb4bd118ff025abf131e7db8cf4eeb4cf333`. Model overrides and local
credential fallback use descriptor reads and atomic or exclusive writes;
credential read failures remain distinct from missing credentials, local
credentials tighten to `0600`, and committed examples retain names without
values. Release notes/backfill retain one admitted descriptor for `VERSION`,
`CHANGELOG.md`, and `internal/buildinfo/version.go`, preserving modes and prior
dry-run/check/order behavior. Jira ticket creation is exclusive and Jira
reconciliation is atomic and mode-preserving after existing admission.
Focused normal/race tests, affected caller tests, package vet, formatting, and
diff checks pass; QA, Security, Release Manager, and Orchestrator returned GO.
Checkpoint C's named mutation/writer portion is complete. Checkpoint D, T-075,
and F-017-S002 remain incomplete.

Use atomic replacement or exclusive creation as appropriate and preserve existing behavior and owner-only credential mode. Do not mediate shell_exec, Git's own internal writes, global MARS state, model downloads, databases, logs, traces, self-update installation, or validation-output directories; those are outside this repository boundary or owned by T-076/T-077.

## Checkpoint D — Deferred Read Inventory And Closure

- Perform one mechanical source inventory of the deferred bundle/context, tools-policy, scanner, release, and Jira general read-side surfaces, migrate the reachable named repository reads through `repofs`, and prove no direct repo-root os.ReadFile/Open/WriteFile/MkdirAll/Rename/Remove remains in scope. Record the result as evidence; do not add a permanent generalized linter.
- Sync only F-004, F-005, F-007, F-017, guardrails/tools guidance, mirrored CLI/generated doctrine, the active plan, goal, and this ticket where behavior changes.
- Run focused normal/race tests, docs-consistency and DocSync, four supported CGO-disabled builds, and one installed clean-target smoke proving symlink read/write rejection, exact index-only/force-added secret blocking with no value leakage, and ordinary ignored untracked .harness/.env.local omission.
- Obtain concurrent QA/Security review plus Dogfood and Release Manager sign-off.

Checkpoint D passed through exact commits
`228d859511fb2f7c93e0162424c2e6dc95107e44`,
`7578549bdd0dde90857f9652e651832d484abdb2`,
`ff69aaa1bab3680d169e8889866ba73cccb397c9`,
`16b5527bbe48e8afea82bb70127d383d6f280ed7`, and
`c18030edb44d1b869a03d30e4339ff457641c6e4`. Repository prompt, skill,
integration, learning, ticket-state, model-policy, tools-policy, scanner,
release-evidence, and Jira inventory reads now retain the admitted repository
descriptor. Missing optional inputs keep their prior fallback behavior, while
non-skipped symlink inputs and inventory failures fail closed without reading
or reporting an outside target.

The final DocSync correction passed at exact commit
`9ba8156942a584f301888a3675942923739993d6`: source inventory, source reads,
foundation admission, and referenced-document checks retain one descriptor;
a symlinked source produces only a fixed repository-relative error, while a
symlinked referenced document produces the existing missing-document finding.
Full `internal/docsync` and `internal/docsconsistency` tests, focused race tests,
affected-package vet, formatting, and diff checks passed. The exact
CGO-disabled build hashes were:

- Darwin/amd64: `dedcecb5e05416fdb6614e7c9d8010f446a51ff3e4fbafe2429435fc46bf4ed0`
- Darwin/arm64: `c1137731531fded59e600e36ba8f77cd7ef1d6759262ddc223a3b7235831a28f`
- Linux/amd64: `5113f1b119a35c46a90fbb93a28877c61213527ee163319960961717ac7d290c`
- Linux/arm64: `844cd84754cc55ffd426a657549b104a89ce2b02a8dbab5ab47b739379bace06`

Dogfood installed the exact `c18030e` candidate with SHA-256
`7fde74b563c0b54f456e6fbecf31e4fecdc2fe75b9444191105ff60a3ee17bd6`
and passed clean init/Engineer dry-run, role-prompt and file-write symlink
rejection, exact index-only and force-added ignored-local-env blocking with
redacted output, and ordinary ignored-untracked-local-env omission. The final
Darwin/arm64 candidate was installed owner-only with SHA-256
`c1137731531fded59e600e36ba8f77cd7ef1d6759262ddc223a3b7235831a28f`,
Go `1.26.5`, revision `9ba8156942a584f301888a3675942923739993d6`,
and `vcs.modified=false`; its hostile source-link audit disclosed only the
relative source path, its referenced-doc link produced the missing-doc finding,
and its live repository audit exited zero with no findings. QA, Security,
Dogfood, Release Manager, and Orchestrator returned GO. T-075 is complete;
F-017-S002 remains incomplete pending T-076 and resumed T-058.

## Post-Closure Fixture Correction — 2026-08-24

T-077's source-compatibility replay exposed nine `internal/agent` tests whose
temporary worktrees had not adopted Checkpoint B's intentional fail-closed Git
admission boundary. Mutating tool calls and review commands reached the exact
Git-authoritative scanner in non-Git fixture directories, failed closed, and
exhausted their later response mocks. This was test-fixture drift, not a
production scanner regression.

Exact pushed commit `56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b`
changes only `internal/agent/loop_test.go` (`13` insertions, `1` deletion;
pre-commit file SHA-256
`76b8f68c285288d5c5a7f5bd7f5be586a684fe0f30f6ac8ef0f1661f9e98ef4d`).
The nine affected fixtures now create a deterministic clean Git baseline with
local identity, commit signing disabled, `git add -A`, and an allow-empty,
no-hook commit before the loop runs. The exact nine and full `internal/agent`
normal/race suites, package vet, repository-secret safety normal/race tests,
three named tool-policy secret tests, and diff check pass. QA and Security
independently returned GO. No production code, scanner policy, version, or
public behavior changed, so T-075 remains closed and F-017-S002 is not
reopened.

## Acceptance

- No named repository read, create, atomic write, mkdir, chmod, rename, remove, stat, init, upgrade, eject, ticket/release writer, credential-route reader/writer, or generated-target operation can follow a hostile repository symlink outside the selected descriptor-bound root.
- Staged scanning uses exact index blob bytes; tracked, staged, force-added, renamed, and index-only content cannot be substituted by worktree bytes or skipped. Deletions are explicitly reconciled as tombstones with no resulting blob.
- Scanner errors and unsupported index states never report clean, and no candidate value appears in text, JSON, errors, logs, or traces.
- Darwin/Linux focused tests, cross-builds, installed smoke, DocSync, QA, Security, Dogfood, Release Manager, and Orchestrator pass.
- T-075 closes only the repository-path and secret-scan slice. F-017-S002 remains incomplete pending T-076 and resumed T-058; Primary Status remains primary_blocked.

## No-Go

Any external sentinel read or mutation, staged blob substitution, tracked/force-added .harness/.env.local exemption, silent scan skip/error, candidate-value rendering, custom audit/runtime/kernel-lab expansion, shell-containment claim, version/tag/Release/settings/visibility/announcement mutation, or claim that F-017-S002 is complete.
