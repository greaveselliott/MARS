---
id: T-076
title: Harden execution profiles, child processes, state, and traces
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002"]
end_to_end_evidence: required
evidence_links: ["commit:9191182601d79b996f1848a1e867e50b7d6eaf1c", "commit:5c23f536fadd9ab18694e3f46ed9b10ca96594da", "commit:473b829efe865630f4942b55af9e0108d7529d0c", "commit:9eb3f96d1de9f91ba54ee4f2dd70d0cdf98b8708", "commit:31b00b1ce01cce10df81fc0769f6bdbbc94ff1b5", "binary-sha256:dc8033d3024624ae182175fec80362a87e8585048f5cf9d17cb319f0a0420dbe"]
verified_by: "Checkpoint A and B1 reviewer gates, Checkpoint B2 QA/Security/Orchestrator GO, focused config/setup gates, and installed config-mode smoke on 2026-08-09"
owner: "foundation-maintainer"
last_attempt: "2026-08-09: risk-proportionate launch scope passed through 31b00b1; broader local-state lifecycle work deferred"
blocker: "none"
blocked_by: []
trace_id: "launch-execution-boundary:2026-08-09"
next_action: "Resume T-058's bounded installed browser proof; broader local-state redaction, retention, and permission work is post-launch backlog."
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

T-075 closed the descriptor-bound repository-path and exact Git-index secret-scanning gate. Checkpoint A closed the independent execution-profile admission gap, Checkpoint B1 sanitizes the named reachable child-process environment seams, and Checkpoint B2 scopes process records and cleanup to the owning job. A final narrow config correction closes the concrete local secret exposure: setup and config persistence can no longer leave a GitHub token in a group/world-readable config file. On 2026-08-09 the owner governor removed the remaining generalized local-state, redaction, export, and retention programme from the public-launch critical path. The repository remains private and the launch version freeze remains in force.

## Outcome

Make observer the non-mutating default, require explicit acknowledgement for same-user host execution, fail isolated mode closed until an enforceable adapter exists, sanitize model-controlled child environments, scope child cleanup to the owning job, and keep persisted GitHub credentials owner-only. Correct the public shared-database path without migrating state. Defer broader local database/log modes, centralized redaction, trace export, and retention lifecycle work rather than turning source publication into a general hardening programme.

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

**Status:** Complete at exact pushed commits
`5c23f536fadd9ab18694e3f46ed9b10ca96594da` (B1) and
`473b829efe865630f4942b55af9e0108d7529d0c` (B2).

Deliver two independently green semantic commits inside this checkpoint.

1. **B1 — Complete.** One thin standard-library child-environment package now preserves ordinary non-sensitive PATH, HOME, temporary-directory, locale, and toolchain/cache state while removing credential-like, MARS, GitHub, cloud/provider, delimiter-bounded auth, SSH, token, secret, password, API-key, private-key, and credential names by default. Parent-only `MARS_CHILD_ENV_ALLOWLIST` can restore an exact named variable but never reaches a child and cannot be widened by repository or model configuration. Every named reachable shell, MARS CLI, dependency, Git, code-intelligence, managed-inference, MCP stdio, Jira proxy, GitHub-auth, and source-update subprocess receives an explicit environment. Jira removes every repository-requested passthrough name from the sanitized base and fails actionably unless the same name is owner-allowlisted, without rendering its value. Filtering is name-based exposure reduction, not same-user containment.
2. **B2 — Complete.** Background process records require a job ID and are listed, killed, cleaned up, and admitted for direct helper process control only within that job. Unowned and cross-job targets, direct shell-form kill/pkill/killall process-control forms, and incomplete PID sets fail closed without an OS-wide fallback. Cleanup sends TERM to the recorded process group, waits at most two seconds, then sends KILL if the group remains. Server jobs, manual run, one-shot tools run, and MCP EOF/error clean up only their owned groups. Serve no longer uses lsof/pgrep to kill arbitrary port owners or llama-server-named processes; foreign listeners and unrelated llama-server-named processes survive.

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

QA, Security, Release Manager, and Orchestrator returned GO for B1.

The B2 ownership, policy, cleanup, lifecycle, and foreign-process regressions
passed at the exact pushed semantic checkpoint. Exact B2 gates passed:

- `go test ./internal/tools ./internal/mcpstdio ./internal/serve ./internal/scanner ./cmd/mars`
- `go test -race ./internal/tools ./internal/mcpstdio ./internal/serve ./cmd/mars`
- `go vet ./internal/tools ./internal/mcpstdio ./internal/serve ./internal/scanner ./cmd/mars`
- `go test ./internal/docsync ./internal/docsconsistency`
- `go run ./cmd/mars docsync audit --repo .` (`364` files checked, `0` findings)
- `git diff --check`

QA, Security, and Orchestrator returned GO for B2. Together with the bounded
config correction below, these checkpoints satisfy T-076's revised launch
scope. Resumed T-058 and F-017-S002 remain incomplete. The repository remains
private at `VERSION=0.68.49` with
Primary Status `primary_blocked`; the legal/rights and installed-App no-gos
remain, and no Release, settings, visibility, signing, publication, or
announcement authority changed.

Prove that poisoned parent credentials and values are absent from foreground/background child output while ordinary builds still work; concurrent jobs cannot see or kill each other's children; standalone run leaves no owned child; and unrelated listeners or llama-server-named processes survive startup and cleanup. Do not add a generalized supervisor, descendant scanner, container, or namespace layer.

## Checkpoint C — Persisted Credential Permission

**Status:** Complete at exact pushed commit
`9eb3f96d1de9f91ba54ee4f2dd70d0cdf98b8708`.

Setup now writes the default config through the same `0600` save path used by
later GitHub authentication. Config load/save tightens an existing loose
regular config leaf before reading or replacing it, rejects symlink and
non-regular leaves, and leaves custom parent-directory modes unchanged.
Focused config/setup normal, race, vet, formatting, and diff gates passed.

An installed Go 1.26.5 candidate from exact clean commit `9eb3f96` has SHA-256
`dc8033d3024624ae182175fec80362a87e8585048f5cf9d17cb319f0a0420dbe`.
Under `umask 000`, `setup --test-mode --skip-download --skip-github
--inference defer --yes --plain` completed and created
`~/.mars/config.yaml` with mode `0600`.

Exact pushed commit `31b00b1ce01cce10df81fc0769f6bdbbc94ff1b5`
also corrects the CLI/help/docs shared-serve default to the runtime's actual
legacy `~/.mars-harness/db/mars.db` path without moving or rewriting state.

## Deferred Post-Launch Hardening

The previously proposed database/WAL/SHM and log mode normalization,
centralized output redaction, trace export, purge, and automatic retention are
useful local product hardening but are not required to expose this source
repository or to close a demonstrated remote/publication path. They are
explicitly deferred; T-076 does not claim those features exist.

## Acceptance

- run, start, serve, tools run, and mcp serve default to observer; progressive trust cannot bypass the observer ceiling.
- Host authority cannot begin without explicit acknowledgement, and isolated cannot fall back or create state.
- Observer does not initialize, commit, repair, or otherwise mutate an uninitialized or dirty target through command-owned lifecycle behavior.
- Model-controlled children do not inherit synthetic credential names or values; normal supported toolchains still execute.
- One job cannot terminate another job's child, standalone cleanup is bounded, and startup never kills an unowned listener/process.
- Persisted config credentials are owner-only, and the installed permissive-umask setup smoke proves mode `0600`.
- Live CLI/help/docs name the actual shared serve database path without claiming migration.
- Focused tests, race, vet, installed config-mode smoke, and DocSync pass for the revised launch scope.
- T-076 closes only this execution/environment/process/config-secret slice. Primary Status stays primary_blocked and F-017-S002 remains incomplete pending resumed T-058.

## No-Go

Any default target mutation; host execution without acknowledgement; isolated fallback or containment overclaim; child credential exposure; cross-job or unowned-process kill; group/world-readable persisted config credential; false shared-database path claim; generalized sandbox/supervisor/DLP/storage-framework expansion; T-075 reopening; or any VERSION, CHANGELOG, tag, Release, settings, visibility, signing, publication, or announcement action.
