# Tools Glossary

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** Mars Harness maintainers
**Mirrors:** Generated target `docs/design-docs/tools-glossary.md`

## Purpose

This glossary is first-class mirrored tool context. It tells LLM chats which
tools exist, when to use them, and which guardrails shape their use in both the
foundation harness and deployed harnesses.

Read this file whenever a task involves tool availability, tool selection,
tool allowlists, tool policy, or CLI operation. Keep it current when built-in
tools are added, removed, renamed, or materially change behavior.

## Availability Rules

- Tools are available only when registered in the built-in registry and included
  in the current role allowlist.
- Universal mirrored tools are part of the same built-in registry in the
  foundation harness and every deployed harness initialized or upgraded by
  `mars-harness`.
- Universal tools must be readily discoverable through `mars-harness tools list`
  and executable through `mars-harness tools run <name>` when an operator or
  external LLM shell is outside an active agent run.
- Universal tools must also be exposed through the standard MCP stdio surface:
  `mars-harness mcp serve --repo <path>`. This is the preferred integration
  point for MCP-compatible clients and local harness agents because it makes
  Mars Harness tools native tools instead of shell conventions.
- The universal tool surface is model-provider agnostic. Deployed harnesses use
  local models by default, and MCP/tool transport must not assume frontier cloud
  model access.
- During active agent runs, universal tools are still trust-gated and must appear
  in the role's allowlist; outside an agent run, `mars-harness tools run` uses
  the same executor, repo-root resolution, trust policy, and JSON argument path.
- Mirrored tools are valid in both the foundation harness and deployed harnesses.
- Mutating tools are blocked at observer trust.
- Prefer purpose-built tools over `shell_exec` when a deterministic tool exists.
- Prefer structured argv over shell strings unless shell features are required.

## Mirrored Built-In Tools

| Tool | Use When | Notes |
| --- | --- | --- |
| `file_read` | Read a known file path from the repository. | Non-mutating. Use before editing or reviewing code. |
| `file_write` | Create or replace a file under the repository root. | Mutating. Guardrails and secret scanning apply. New ticket markdown is blocked; use `ticket_create`. Done feature-ticket writes are blocked when required BDD evidence is empty or the same ticket still exists in another lifecycle directory. New `docs/features/F-NNN*.md` writes are blocked when another contract with the same `F-NNN` ID already exists, and feature files cannot contain duplicate scenario heading IDs; duplicate-scenario errors name heading line numbers and clarify that Scenario Schedule list references are allowed. COO may only write planning artifacts. Engineer product writes require a claimed in-progress product ticket when ordinary backlog work exists. |
| `file_search` | Find files by glob-style path patterns. | Non-mutating. Use for inventory before broad reads. |
| `grep` | Search file contents with a regex. | Non-mutating. Use to locate symbols, text, or repeated patterns. |
| `shell_exec` | Run a subprocess when no purpose-built tool fits. | Mutating. Prefer argv for a single executable and arguments; use `shell_command` when shell parsing, redirection, pipes, control operators, or shell builtins are required. Use background for long-running dev servers. Ticket moves into `docs/tickets/done/` are blocked until required feature evidence fields are populated, and copying tickets into `done/` is blocked so lifecycle completion uses `git mv`. Engineer product mutation through shell commands requires a claimed in-progress product ticket; `git mv` from backlog to in-progress remains allowed for the claim step. |
| `workspace_hygiene` | Audit generated dependency/build churn, ignore policy, tracked generated paths, and deletion risk before agent work or dependency sync. | Non-mutating. Returns `status`, `blocking`, `auto_repairable`, `findings`, `recipe_id`, `message`, and `next_action`; `serve` can auto-commit safe `.gitignore`-only repairs before model loading. |
| `github_auth_check` | Check private Mars Harness GitHub Release auth readiness. | Non-mutating. Returns `status`, `auth_source`, `repo_access`, `release_access`, `message`, and `next_action` without revealing token values. |
| `dependency_sync` | Run package-manager install or fetch through deterministic workspace hygiene preflight and postflight. | Mutating. Performs the same safe `.gitignore`-only repair when needed. Use instead of raw `npm install`, `npm ci`, `pnpm install`, `yarn install`, `bun install`, `go mod download`, `cargo fetch`, `pip install`, `bundle install`, or `composer install`. Engineer dependency sync requires a claimed in-progress product ticket when backlog product work exists. |
| `mars_harness_cli` | Read exhaustive CLI reference or run `mars-harness` commands with structured argv. | Mutating. Use for setup, init, upgrade, doctor, scan, run, start/serve, release, scores, trust, models, and update workflows. The resolver prefers `MARS_HARNESS_CLI_BIN`, then the active harness executable, then `PATH`, and stale binaries produce actionable update guidance. When CLI commands or flags change, sync the reference, repo-shortcut map, skills, and generated doctrine per [cli-tool-skill-sync.md](cli-tool-skill-sync.md). |
| `record_decision` | Persist durable decisions, trade-offs, and reusable learnings. | Mutating. Use when the reasoning should survive the chat. |
| `ticket_create` | Create or update deduped markdown tickets. | Mutating. Use instead of hand-writing ticket files. |
| `job_disposition_record` | Record the terminal outcome of a dispatch-mode agent job. | Mutating. Required before dispatch-mode jobs complete. Non-Orchestrator roles must commit repo changes before terminal dispositions that approve, complete, request changes, block, fail, or otherwise hand off work; runtime-only `.harness/learnings.yaml` convention metadata does not block by itself. Engineer successful dispositions with `ticket_id` require that ticket to live in `docs/tickets/done/`; `no_work` can finish without a ticket. |
| `tool_create` | Scaffold a new built-in Go tool and starter test. | Mutating. Follow with implementation, registration, trust policy, tests, and allowlist updates. |
| `persona_create` | Scaffold a repo-local persona manual, role prompt, registry row, and optional manifest role. | Mutating. Use for universal, foundation, or deployed persona proposals; foundation defaults still require adding the canonical Go entry in `internal/personas`. |
| `release_orchestrate` | Plan and preflight the full semantic commit, release notes, push, tag, workflow, and asset verification ritual. | Mutating workflow. Use before driving release state with `mars_harness_cli` and git tools. |
| `github_release_status` | Inspect the release-status workflow and decide whether to wait, rerun, verify, or record a blocker. | Non-mutating. Pairs local tag state with GitHub inspection commands. |
| `architecture_audit` | Check architecture docs against current CLI, generated harness layout, tool registry, and runtime boundaries. | Non-mutating. Use after architecture-affecting changes and before doc reviews. |
| `harness_doctrine_sync` | Check mirrored foundation and deployed harness doctrine for glossary, tools, operating-model, and generated-target consistency. | Non-mutating. Use when changing operating doctrine or mirrored definitions. |
| `docsync_audit` | Audit source files for `MarsDocSync` metadata and associated documentation pointers. | Non-mutating. Use before commits that touch code or when validating the no-stale-docs operating model in [documentation-sync-architecture.md](documentation-sync-architecture.md). |
| `git_release_guard` | Check git, tag, version, and release-note invariants around the release flow. | Non-mutating. Use before and after release-note generation. |
| `tool_inventory_audit` | Compare registered tools, mutating policy, tools glossary, generated target guidance, and role exposure. | Non-mutating. Use whenever tools are added, removed, renamed, or reclassified. |
| `tool_creation_guard` | Audit whether built-in tool creation followed the governed `tool_create` and `record_decision` path. | Non-mutating. Use when reviewing new tool work or exception handling. |
| `task_trace_summarize` | Summarize a recent work trace and identify repeated manual processes that should become formal tools. | Non-mutating. Use after multi-step work or recurring manual recovery. |
| `git_status` | Inspect repository state. | Non-mutating. Use before commits or risky operations. |
| `git_diff` | Inspect unstaged or staged changes. | Non-mutating. Use before review, commit, and release notes. |
| `git_commit` | Stage files and create a semantic commit. | Mutating. Requires meaningful diff and strict-trunk discipline. |
| `git_branch` | Create or switch a local branch. | Mutating. Use only for explicit branch workflows; trunk-based delivery normally stays on `main`. |
| `git_push` | Push committed changes. | Mutating. Strict trunk allows pushing `main`. |

## Selection Guide

- Need Mars Harness behavior, versioning, setup, release, score, trust, or target
  harness lifecycle operations: use `mars_harness_cli`.
- Need to verify private Mars Harness release access before update, release
  verification, install repair, or version-drift remediation: use
  `github_auth_check` or `mars-harness auth github check`.
- Need to add, remove, rename, or change a `mars-harness` CLI command or flag:
  update `mars_harness_cli`, generated skills, generated doctrine, and product
  docs using [cli-tool-skill-sync.md](cli-tool-skill-sync.md).
- Need to discover or invoke the universal tool surface from an operator shell
  or external LLM context: use `mars-harness tools list` and
  `mars-harness tools run <name> --args-json '{...}'`. Add
  `--trust contributor` only for deliberate mutating tool calls.
- Need an MCP-compatible client or local harness agent to see Mars Harness tools
  as native tools: configure it to launch
  `mars-harness mcp serve --repo <path> --trust observer|contributor`.
- Need to run or prepare the whole release ritual: use `release_orchestrate`,
  `git_release_guard`, and `github_release_status` before mutating state; use
  `mars_harness_cli` with `release backfill-notes --check` when auditing
  historical changelog narrative compliance.
- Need a durable repo-owned note: use `record_decision`.
- Need backlog, dogfood, dependency, or intervention-debt work item creation:
  use `ticket_create`. Do not hand-write new ticket markdown with
  `file_write`.
- Need ticket lifecycle movement after a ticket already exists: use `git mv`
  through `shell_exec` and commit it. Blast-radius policy permits the deletion
  side only when the same ticket ID is present in another lifecycle directory as
  the staged, untracked, or already-existing counterpart. For feature tickets,
  fill `evidence_links` and `verified_by` before moving to `done/`; tool policy
  blocks missing evidence before the move. Do not copy a ticket into `done/`
  and then delete the source; tool policy requires one lifecycle move.
- Need COO planning updates: use `file_write` only for `docs/exec-plans`,
  `docs/features`, or `docs/goals/observations.md`; product implementation
  files must wait for CTO tickets and Engineer delivery.
- Need a dispatch-mode handoff, blocker, review request, no-work outcome, or
  completed-work signal: use `job_disposition_record` after `git_status` is
  clean or after committing the produced work with `git_commit`. Runtime-only
  `.harness/learnings.yaml` metadata is ignored by the disposition gate and may
  be auto-committed by the server when it is the only dirty path, but any
  product, ticket, documentation, or source change must be committed first.
- Need a new deterministic capability: use `tool_create`, then finish the code
  and tests manually.
- Need a new or revised agent persona: use `persona_create`, then add canonical
  foundation entries to `internal/personas` when the persona is a foundation
  default.
- Need to decide whether repeated work deserves a tool: use
  `task_trace_summarize`, then create or update a ticket or tool.
- Need to keep documentation, doctrine, and tools mirrored: use
  `docsync_audit`, `architecture_audit`, `harness_doctrine_sync`,
  `tool_creation_guard`, and `tool_inventory_audit`.
- Need to inspect generated dependency/build churn before a job, commit, or
  package-manager operation: use `workspace_hygiene`. Missing ignore policy may
  be auto-repaired by `serve` as a `.gitignore`-only commit when generated paths
  are untracked and `.gitignore` has no user changes.
- Need dependency setup or package fetch/install: use `dependency_sync`, not raw
  package-manager commands through `shell_exec`.
- Need to know which docs must be checked after touching a code file: read the
  file's `MarsDocSync` block and run `docsync_audit` or
  `mars-harness docsync audit --repo .`.
- Need ordinary repository inspection: use `file_search`, `grep`, `file_read`,
  `git_status`, or `git_diff`.
- Need ordinary repository mutation: use `file_write`, `git_commit`, and
  `git_push` with the repository's operating rules. In local throwaway demos
  with no configured remote, `git_push` is a clean skip that leaves the commit
  local instead of creating a retry loop.
- Need a command outside the built-in tool surface: use `shell_exec`, keep the
  command narrow, and record any reusable gap as a tool improvement.
- Need Dogfood validation evidence: keep it observation-first. Dogfood may
  write `docs/reports/dogfood/*.md` and create target-owned tickets with
  `ticket_create`, but it must not edit product source, package manifests,
  lockfiles, config, or harness scaffold to make validation pass.

## Maintenance Rules

- New built-in tools must originate through `tool_create` before manual
  implementation. If an agent bypasses `tool_create`, it must first record a
  durable exception with `record_decision` and add design-doc rationale before
  the change is complete.
- Every newly created tool must extend this glossary in the same change that
  implements or exposes the tool.
- Update this glossary in the same change that removes, renames, or materially
  changes a built-in tool.
- Mirror changes into generated target defaults in `internal/scanner/init.go`.
- Update scanner tests so initialized harnesses keep this first-class tool
  context.
- Keep use cases short and action-oriented; deeper rationale belongs in design
  decisions.
