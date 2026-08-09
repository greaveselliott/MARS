---
id: T-076
title: Harden execution profiles, child processes, state, and traces
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002"]
end_to_end_evidence: required
evidence_links: ["commit:9191182601d79b996f1848a1e867e50b7d6eaf1c", "commit:5c23f536fadd9ab18694e3f46ed9b10ca96594da"]
verified_by: "Checkpoints A and B1: QA, Security, Release Manager, and Orchestrator GO; T-076 closure still requires Checkpoints B2 through D and Dogfood"
owner: "foundation-maintainer"
last_attempt: "2026-08-09: Checkpoint B1 passed and was pushed at 5c23f536fadd9ab18694e3f46ed9b10ca96594da"
blocker: "none"
blocked_by: []
trace_id: "launch-execution-boundary:2026-08-09"
next_action: "Implement Checkpoint B2 only: replace global background-process tracking and unowned cleanup with job-owned bounded cleanup."
dedupe_key: "open-source:execution-profile-environment-state-trace"
metadata:
  classification: "foundation-owned"
  isolation_status: "unavailable"
  mutation_authority: "repository-source-tests-docs-only"
  primary_status: "primary_blocked"
  publication_authority: "denied"
source: MARS Launch-Complete Open-Source Delivery Plan — T-076
created: 2026-08-09
depends_on: [T-075]
---

# T-076: Harden execution profiles, child processes, state, and traces

## Context

T-075 closed the descriptor-bound repository-path and exact Git-index secret-scanning gate. Checkpoint A closed the independent execution-profile admission gap, and Checkpoint B1 now sanitizes the named reachable child-process environment seams. F-017-S002 remains current because background processes are globally tracked, startup can kill unowned processes, MARS-owned state is sometimes group/world-readable, and traces/logs persist raw values indefinitely. These are reachable launch blockers owned by Checkpoints B2 through D. The repository remains private and the launch version freeze remains in force.

## Outcome

Make observer the non-mutating default, require explicit acknowledgement for same-user host execution, fail isolated mode closed until an enforceable adapter exists, sanitize model-controlled child environments, scope child cleanup to the owning job, enforce owner-only MARS state, redact tested credential forms at display/persistence/export boundaries, and provide redacted trace export plus dry-run-first body retention. Use small standard-library seams and existing stores; do not build a container, sandbox framework, process supervisor, secret-vault/DLP engine, or new storage system.

## Checkpoint A — Explicit Execution-Profile Admission

**Status:** Complete at exact pushed commit
`9191182601d79b996f1848a1e867e50b7d6eaf1c`.

- Add one small observer|host|isolated execution-profile type and apply it consistently to run, start, serve, tools run, and mcp serve.
- Default every entry point to observer. Observer is an independent ceiling over manifest and stored progressive trust: read-only tools remain available, while shell_exec and every already-classified mutating tool are unavailable even to contributor/autonomous roles.
- Observer run and start against an uninitialized target fail before scaffold, commit, workspace-hygiene repair, or model-controlled host execution, with actionable guidance to run mars init deliberately or choose acknowledged host mode. Observer-safe tools run and mcp serve reads remain usable without a harness. An initialized observer may start managed inference and write owner-only operational database, log, and trace state while target mutation and host-capable tools remain unavailable.
- Host requires both --execution-profile host and --acknowledge-host-execution before target mutation or model-controlled host execution. Once acknowledged, existing observer/contributor/autonomous role trust continues to govern the allowed tool set.
- Isolated always returns one fixed unavailable error before state or subprocess work. Do not wire or describe the existing cwd/ulimit fallback as isolation.
- State truthfully that acknowledged host commands run with the current OS user's filesystem, network, process, keychain, and credential authority; profile admission is not containment.
- Commit and push this checkpoint before further runtime work.

The implementation defaults `run`, `start`, `serve`, `tools run`, and
`mcp serve` to observer, requires the independent host acknowledgement, rejects
unsupported isolation before effects, and centrally blocks direct server target
writers in observer while retaining owner-local bookkeeping. During final QA,
the acknowledged-host manual-run path was found to hard-code contributor trust,
which would have upgraded a manifest observer role. That regression was fixed
before push by parsing the role's configured trust with an observer fallback;
the focused live-command regression proves the model receives the observer
mutator rejection and the requested target file is not created.

Exact Checkpoint A gates passed:

- `go test ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars ./internal/scanner ./internal/docsconsistency ./internal/docsync`
- `go test -race ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars`
- `go vet ./internal/executionprofile ./internal/tools ./internal/serve ./cmd/mars ./internal/scanner`
- `go run ./cmd/mars docsync audit --repo .` (`362` files checked, `0` findings)
- `git diff --check`
- Final trust-regression delta: `go test ./cmd/mars -run '^TestRunHostAcknowledgementPreservesObserverRoleTrust$' -count=1`, `go test ./cmd/mars`, `go test -race ./cmd/mars`, `go vet ./cmd/mars`, and `git diff --check`

QA, Security, Release Manager, and Orchestrator returned GO for Checkpoint A.
This evidence completes only Checkpoint A. It did not change `VERSION=0.68.49`,
repository visibility, legal/rights or installed-App dispositions, settings,
tags, Releases, signing, publication, or announcement authority.

## Checkpoint B — Sanitized Children And Job-Owned Processes

**Status:** B1 complete at exact pushed commit
`5c23f536fadd9ab18694e3f46ed9b10ca96594da`; B2 is current, incomplete, and
the sole next implementation action.

Deliver two independently green semantic commits inside this checkpoint.

1. **B1 — Complete.** One thin standard-library child-environment package now preserves ordinary non-sensitive PATH, HOME, temporary-directory, locale, and toolchain/cache state while removing credential-like, MARS, GitHub, cloud/provider, delimiter-bounded auth, SSH, token, secret, password, API-key, private-key, and credential names by default. Parent-only `MARS_CHILD_ENV_ALLOWLIST` can restore an exact named variable but never reaches a child and cannot be widened by repository or model configuration. Every named reachable shell, MARS CLI, dependency, Git, code-intelligence, managed-inference, MCP stdio, Jira proxy, GitHub-auth, and source-update subprocess receives an explicit environment. Jira removes every repository-requested passthrough name from the sanitized base and fails actionably unless the same name is owner-allowlisted, without rendering its value. Filtering is name-based exposure reduction, not same-user containment.
2. **B2 — Current.** Replace the package-global background-process map with job/session-owned process records. Foreground/background termination and targeted kill may affect only the current job's recorded process groups. Manual run, MCP shutdown, and server jobs clean up only their own groups with bounded TERM then KILL. Remove serve startup/cleanup behavior that kills arbitrary port owners or processes named llama-server; a foreign listener is an actionable conflict, while owned inference stops through its existing manager.

The B1 real-child regressions prove foreground and background shell children
drop a poisoned GitHub credential, dependency sync retains PATH and cache state,
managed inference drops the poison, and Jira requires the repository request
and owner allowlist even for an otherwise benign variable. Exact B1 gates
passed:

- `go test ./internal/childenv ./internal/tools ./internal/inference ./internal/serve ./internal/codeintel ./internal/githubauth ./internal/selfupdate ./internal/mcpclient ./internal/jira ./cmd/mars -count=1`
- `GOCACHE=/private/tmp/mars-t076b1-root-race-cache go test -race ./internal/childenv ./internal/tools ./internal/inference ./internal/jira -run 'Test(Filter|ApplyWith|ShellExecSanitizes|DependencySyncChild|ServerManagedChild|ProxyEnvironment)' -count=1`
- `GOCACHE=/private/tmp/mars-t076b1-root-cache go vet ./internal/childenv ./internal/tools ./internal/inference ./internal/serve ./internal/codeintel ./internal/githubauth ./internal/selfupdate ./internal/mcpclient ./internal/jira ./cmd/mars`
- `go test ./internal/docsconsistency ./internal/docsync -count=1`
- `go run ./cmd/mars docsync audit --repo .` (`364` files checked, `0` findings)
- `git diff --check`

QA, Security, Release Manager, and Orchestrator returned GO for B1. This closes
only B1. B2, Checkpoints C and D, T-076, resumed T-058, and F-017-S002 remain
incomplete. The repository remains private at `VERSION=0.68.49` with Primary
Status `primary_blocked`; the legal/rights and installed-App no-gos remain, and
no Release, settings, visibility, signing, publication, or announcement
authority changed.

Prove that poisoned parent credentials and values are absent from foreground/background child output while ordinary builds still work; concurrent jobs cannot see or kill each other's children; standalone run leaves no owned child; and unrelated listeners or llama-server-named processes survive startup and cleanup. Do not add a generalized supervisor, descendant scanner, container, or namespace layer.

## Checkpoint C — Owner-Only State And Central Redaction

**Status:** Incomplete and not current.

Deliver two independently green semantic commits inside this checkpoint.

1. Add one thin permissions helper for MARS-owned canonical and recognized legacy state. Canonical state directories are 0700; config, credential/token, database, WAL/SHM, command/inference log, and trace/export files are 0600. Tighten safe existing MARS-owned modes or fail actionably before agent execution. Canonicalize shared serve state to ~/.mars/db/mars.db; legacy state is an explicit migration/read decision, not the default. Do not chmod arbitrary operator-selected parent directories.
2. Add one small standard-library redactor at the actual display/persistence/export choke points. Remove Authorization/Bearer remainder, credential URLs/query pairs, common token/key-value forms, and private-key bodies from trace turns/tool arguments/final errors, command/inference logs, CLI/TraceWriter and dashboard projections, and exports. Preserve ordinary actionable text and repository-relative locators, render [redacted], never hash candidate values, do not alter content sent to the active model, and do not build an exact-secret registry, home-path scrubber, or generalized DLP system. This is defense in depth, not a claim that acknowledged host code cannot read user files.

Prove exact modes under a permissive umask and after safe migration. Inject one synthetic credential form through assistant content, tool arguments, stdout/stderr, structured log attributes, and an error; it must be absent from every tested persisted JSONL/summary, log, CLI, dashboard, and export byte while ordinary text remains useful.

## Checkpoint D — Trace Export, Purge, Retention, And Closure

**Status:** Incomplete and not current. T-076 and F-017-S002 remain incomplete.

- Add mars traces export --repo <path> --output <path>. Export a deterministic redacted JSONL projection, require the output to be outside the target repository through the existing runtime-artifact path rule, create a new regular 0600 file exclusively, and reject in-repository, existing, or symlink outputs.
- Add mars traces purge --repo <path> --older-than <duration>. It is preview-only unless --apply is supplied; preview reports counts without mutation.
- Preserve the existing public contract with a hard maximum: full trace bodies retain for 30 days by default, a manifest trace_retention_days value in the bounded range 1 through 30 may only shorten retention, and summaries retain indefinitely. Per-repository startup pruning and purge affect only full trace bodies in the selected database. Because current command logs are not repository-attributable, age MARS-owned logs older than 30 days globally at logger startup; do not claim a per-repo purge can identify them.
- Synchronize only directly owning CLI, generated-target, feature/design, product, README/AGENTS, active-plan/goal, ticket, and DocSync surfaces.
- Run affected-package normal/race tests and vet, docs consistency/DocSync, four supported CGO-disabled builds, and one installed clean-target smoke covering observer non-mutation, pre-mutation host/isolated refusal, acknowledged-host sanitized child behavior, job-owned cleanup, exact state/export modes, redaction, and dry-run/applied retention.
- Obtain QA, Security, Dogfood, Release Manager, and Orchestrator GO before closing T-076. F-017-S002 remains incomplete pending the resumed T-058 browser proof.

## Acceptance

- run, start, serve, tools run, and mcp serve default to observer; progressive trust cannot bypass the observer ceiling.
- Host authority cannot begin without explicit acknowledgement, and isolated cannot fall back or create state.
- Observer does not initialize, commit, repair, or otherwise mutate an uninitialized or dirty target through command-owned lifecycle behavior.
- Model-controlled children do not inherit synthetic credential names or values; normal supported toolchains still execute.
- One job cannot terminate another job's child, standalone cleanup is bounded, and startup never kills an unowned listener/process.
- New and safely migrated MARS-owned state is owner-only; tested persisted/displayed/exported output contains no synthetic credential form.
- Export is redacted, outside the target repository, and 0600. Purge is dry-run-first; explicit apply deletes only eligible full bodies in the selected trace database while preserving summaries and recent/unrelated data. Global log aging touches only MARS-owned logs older than 30 days.
- Focused tests, four builds, installed smoke, DocSync, QA, Security, Dogfood, Release Manager, and Orchestrator pass.
- T-076 closes only this execution/environment/process/state/redaction/trace slice. Primary Status stays primary_blocked and F-017-S002 remains incomplete pending resumed T-058.

## No-Go

Any default target mutation; host execution without acknowledgement; isolated fallback or containment overclaim; child credential exposure; cross-job or unowned-process kill; group/world-readable MARS state; raw tested credential form in persisted/displayed/exported output; in-repository trace export; purge mutation without apply; retention beyond 30 days; summary/recent/unrelated deletion; generalized sandbox/supervisor/DLP/storage-framework expansion; T-075 reopening; or any VERSION, CHANGELOG, tag, Release, settings, visibility, signing, publication, or announcement action.
