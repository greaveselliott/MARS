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
| `file_write` | Create or replace a file under the repository root. | Mutating. Guardrails and secret scanning apply. |
| `file_search` | Find files by glob-style path patterns. | Non-mutating. Use for inventory before broad reads. |
| `grep` | Search file contents with a regex. | Non-mutating. Use to locate symbols, text, or repeated patterns. |
| `shell_exec` | Run a subprocess when no purpose-built tool fits. | Mutating. Prefer argv; use background for long-running dev servers. |
| `mars_harness_cli` | Read exhaustive CLI reference or run `mars-harness` commands with structured argv. | Mutating. Use for setup, init, upgrade, doctor, scan, run, start/serve, release, scores, trust, models, and update workflows. |
| `record_decision` | Persist durable decisions, trade-offs, and reusable learnings. | Mutating. Use when the reasoning should survive the chat. |
| `ticket_create` | Create or update deduped markdown tickets. | Mutating. Use instead of hand-writing ticket files. |
| `tool_create` | Scaffold a new built-in Go tool and starter test. | Mutating. Follow with implementation, registration, trust policy, tests, and allowlist updates. |
| `release_orchestrate` | Plan and preflight the full semantic commit, release notes, push, tag, workflow, and asset verification ritual. | Mutating workflow. Use before driving release state with `mars_harness_cli` and git tools. |
| `github_release_status` | Inspect the release-status workflow and decide whether to wait, rerun, verify, or record a blocker. | Non-mutating. Pairs local tag state with GitHub inspection commands. |
| `architecture_audit` | Check architecture docs against current CLI, generated harness layout, tool registry, and runtime boundaries. | Non-mutating. Use after architecture-affecting changes and before doc reviews. |
| `harness_doctrine_sync` | Check mirrored foundation and deployed harness doctrine for glossary, tools, operating-model, and generated-target consistency. | Non-mutating. Use when changing operating doctrine or mirrored definitions. |
| `git_release_guard` | Check git, tag, version, and release-note invariants around the release flow. | Non-mutating. Use before and after release-note generation. |
| `tool_inventory_audit` | Compare registered tools, mutating policy, tools glossary, generated target guidance, and role exposure. | Non-mutating. Use whenever tools are added, removed, renamed, or reclassified. |
| `tool_creation_guard` | Audit whether built-in tool creation followed the governed `tool_create` and `record_decision` path. | Non-mutating. Use when reviewing new tool work or exception handling. |
| `task_trace_summarize` | Summarize a recent work trace and identify repeated manual processes that should become formal tools. | Non-mutating. Use after multi-step work or recurring manual recovery. |
| `git_status` | Inspect repository state. | Non-mutating. Use before commits or risky operations. |
| `git_diff` | Inspect unstaged or staged changes. | Non-mutating. Use before review, commit, and release notes. |
| `git_commit` | Stage files and create a semantic commit. | Mutating. Requires meaningful diff and strict-trunk discipline. |
| `git_push` | Push committed changes. | Mutating. Strict trunk allows pushing `main`. |

## Selection Guide

- Need Mars Harness behavior, versioning, setup, release, score, trust, or target
  harness lifecycle operations: use `mars_harness_cli`.
- Need to discover or invoke the universal tool surface from an operator shell
  or external LLM context: use `mars-harness tools list` and
  `mars-harness tools run <name> --args-json '{...}'`. Add
  `--trust contributor` only for deliberate mutating tool calls.
- Need an MCP-compatible client or local harness agent to see Mars Harness tools
  as native tools: configure it to launch
  `mars-harness mcp serve --repo <path> --trust observer|contributor`.
- Need to run or prepare the whole release ritual: use `release_orchestrate`,
  `git_release_guard`, and `github_release_status` before mutating state.
- Need a durable repo-owned note: use `record_decision`.
- Need backlog or intervention-debt work item creation: use `ticket_create`.
- Need a new deterministic capability: use `tool_create`, then finish the code
  and tests manually.
- Need to decide whether repeated work deserves a tool: use
  `task_trace_summarize`, then create or update a ticket or tool.
- Need to keep documentation, doctrine, and tools mirrored: use
  `architecture_audit`, `harness_doctrine_sync`, `tool_creation_guard`, and
  `tool_inventory_audit`.
- Need ordinary repository inspection: use `file_search`, `grep`, `file_read`,
  `git_status`, or `git_diff`.
- Need ordinary repository mutation: use `file_write`, `git_commit`, and
  `git_push` with the repository's operating rules.
- Need a command outside the built-in tool surface: use `shell_exec`, keep the
  command narrow, and record any reusable gap as a tool improvement.

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
