# Reference: OpenHarness Review Follow-Up

**Status:** Reference for later action
**Date:** 2026-05-19
**Source:** [HKUDS/OpenHarness](https://github.com/HKUDS/OpenHarness)
**Related reference:** [openharness-comparator.md](openharness-comparator.md)
**MARS relevance:** Competitive review and adoption backlog for agent
runtime ergonomics, repo autopilot, extension compatibility, provider profiles,
and blast-radius policy.

## Summary

OpenHarness is a credible and fast-moving Python agent runtime. Its strongest
center of gravity is the interactive Claude Code/Codex-style assistant layer:
streaming model loop, broad provider support, skills, plugins, MCP client,
permissions, memory, TUI, background tasks, and chat-channel gateway through
`ohmo`.

It is not a direct MARS replacement. MARS is still positioned as
a self-hosted autonomous delivery system: local inference lifecycle, generated
target harnesses, role registry, queue, scheduler, guardrails, trust, scoring,
trace evidence, BDD contracts, release notes, and versioned repo state.

The meaningful overlap is OpenHarness's newer repo `autopilot` layer. That layer
scans GitHub issues and PRs, queues task cards, runs agent work in git
worktrees, verifies with policy commands, opens or updates PRs, waits for CI,
retries repair rounds, exports a static dashboard, and can merge through a
label-gated policy. This is the part to track most closely.

## Findings

### OpenHarness strengths

- **Interactive runtime polish:** The `oh` CLI and React/Ink TUI provide a
  smoother hands-on coding-agent experience than Mars currently exposes.
- **Provider flexibility:** Built-in profiles cover Anthropic-compatible,
  OpenAI-compatible, Claude subscription, Codex subscription, Copilot, Ollama
  style endpoints, and several named hosted providers.
- **Extension ergonomics:** Skills and plugins follow familiar Claude-style
  layouts and can be discovered from user, project, and plugin locations.
- **Runtime compaction:** The query loop carries task focus, active artifacts,
  recent file reads, skill invocations, and verified work across compaction.
- **Autopilot proof point:** The repo autopilot feature demonstrates a compact
  GitHub issue/PR intake, worktree execution, verification, PR, CI, and repair
  loop.
- **Distribution familiarity:** Python plus Pydantic makes tool and provider
  contribution easy for a broad community.

### MARS advantages

- **Local-first delivery system:** Mars owns setup, hardware detection,
  llama.cpp subprocess management, model catalog, and local default inference.
- **Single-binary product shape:** The Go binary/no external runtime dependency
  constraint remains a sharp operational advantage.
- **Governed operating model:** Mars ties execution to roles, trust, scoring,
  BDD contracts, tickets, quality evidence, release notes, and generated target
  doctrine.
- **Repo as system of record:** Durable artifacts are first-class. Conversation
  summaries, run outcomes, release evidence, and decisions are not left as
  hidden chat memory.
- **Stronger default path boundary:** Mars built-in file tools resolve paths
  through repo-root checks. OpenHarness file tools only apply project-boundary
  validation when its Docker sandbox is active.
- **Trust before autonomy:** Mars has explicit observer/contributor/autonomous
  trust semantics and scored progression instead of a broad `full_auto` mode as
  the primary autonomous path.

### Safety concern to monitor

OpenHarness has useful permission checks and sensitive-path deny patterns, but
its standard file tools accept resolved absolute paths unless Docker sandboxing
or explicit path rules are active. In `full_auto`, especially under autopilot,
that is a weaker default than MARS should accept.

Mars should keep blast-radius containment as a product-level differentiator:
repo-root file boundaries, explicit mutating-tool trust gates, auditable policy
decisions, and conservative defaults.

## Candidate Follow-Up Work

| Priority | Candidate | Mars-shaped action |
| --- | --- | --- |
| High | Dry-run readiness preview | Extend `run --dry-run`, `doctor`, or a small role-readiness command to report `ready`, `warning`, or `blocked`, with role, trust, tool allowlist, model, context, guardrail, and next-action checks. |
| High | Autopilot comparison | Compare OpenHarness `autopilot` against Mars queue, scheduler, ticket, trace, and dashboard plans. Decide whether any intake/dashboard ideas should become tickets. |
| High | Blast-radius positioning | Document Mars's stronger repo-root and trust-gated tool policy in public-facing copy and generated target guidance. |
| Medium | Skill compatibility adapters | Consider optional import/discovery of `.claude/skills`, `.agents/skills`, or Claude-style `SKILL.md` metadata without allowing arbitrary plugin code by default. |
| Medium | Provider profiles | Add a Mars-shaped profile layer for local and OpenAI-compatible endpoints while preserving local-first defaults. |
| Medium | Edit approval ergonomics | Borrow the unified-diff approval experience for future interactive operator workflows, especially mutating tool review. |
| Medium | Static dashboard export | Consider a repo-owned static dashboard snapshot for target projects, complementary to the live dashboard. |
| Low | Chat gateway | Track `ohmo` as a UX reference only. Mars should not chase chat-assistant breadth until the core delivery loop is stronger. |

## Non-Adoptions

- Do not reframe Mars as an open Claude Code clone.
- Do not import Python OpenHarness code into MARS.
- Do not adopt arbitrary project plugin execution by default.
- Do not loosen Mars's repo-root file boundary or trust gates to match
  OpenHarness `full_auto` ergonomics.
- Do not make hidden memory the system of record for delivery work.
- Do not broaden the universal tool surface without tool-creation rationale,
  tests, docs, and guardrails.

## Strategic Positioning

OpenHarness validates demand for an open "harness" category, but it occupies a
different layer of the stack. It is best understood as an open interactive agent
runtime plus emerging repo autopilot. MARS should position itself as the
governed autonomous delivery system: local inference, durable operating model,
role specialization, trust progression, scoring, release discipline, and target
repo lifecycle.

The best integration posture is interoperability, not convergence. Mars should
make `mars mcp serve` and the universal tool surface strong enough that
OpenHarness, Codex, Claude Code, and other MCP clients can use Mars-governed
tools without Mars adopting their runtime architecture.

## Source Notes

Review inputs:

- [README](https://github.com/HKUDS/OpenHarness/blob/main/README.md) for
  product positioning, provider support, feature surface, and test claims.
- [pyproject.toml](https://github.com/HKUDS/OpenHarness/blob/main/pyproject.toml)
  for package version, dependencies, and entry points.
- [CHANGELOG.md](https://github.com/HKUDS/OpenHarness/blob/main/CHANGELOG.md)
  for recent direction, including v0.1.8 and v0.1.9.
- `src/openharness/engine/query.py` for the tool-aware query loop, parallel tool
  execution, compaction behavior, and tool result carryover.
- `src/openharness/permissions/checker.py` for sensitive-path patterns,
  read-only handling, default confirmation, plan mode, and full-auto behavior.
- `src/openharness/tools/file_read_tool.py`,
  `src/openharness/tools/file_write_tool.py`, and
  `src/openharness/tools/file_edit_tool.py` for path-boundary behavior.
- `src/openharness/autopilot/service.py` for GitHub intake, worktree execution,
  verification, PR creation, CI wait, repair loops, merge policy, and dashboard
  export.
- `.github/workflows/autopilot-*.yml` for scheduled self-hosted autopilot scan
  and tick workflows.
