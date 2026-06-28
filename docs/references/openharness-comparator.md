# Reference: OpenHarness Comparator

**Status:** Reference
**Date:** 2026-05-03
**Source:** [HKUDS/OpenHarness](https://github.com/HKUDS/OpenHarness)
**MARS relevance:** Runtime ergonomics comparator for role readiness,
context compaction, skill metadata, plugin boundaries, and permission design.

## Verdict

OpenHarness is a useful comparator, not a dependency or replacement. It is
better than MARS at interactive Claude Code-style runtime ergonomics:
dry-run preflight, context compaction, skills/plugins, and provider flexibility.
MARS remains better aligned with the source product promise: local-first
autonomous delivery, strict trunk, repo/database system of record, trust,
scoring, guardrails, generated target harnesses, and a single Go binary.

The adoption rule is:

- steal runtime ergonomics
- reject runtime architecture
- fold useful lessons into the current Mars parity tickets instead of creating
  a parallel OpenHarness roadmap

## Useful Bits To Steal

| OpenHarness idea | Mars-shaped adoption |
| --- | --- |
| Dry-run readiness report | Extend `mars run --dry-run` and/or `doctor` with `ready`, `warning`, or `blocked`, role resolution, tool allowlist validity, trust checks, model readiness, guardrail load status, context estimate, and concrete next actions. |
| Sophisticated context compaction | Replace blunt old tool-result pruning with trace-aware compaction that preserves task focus, recent work, tool-call/tool-result integrity, and auditable compact checkpoints. |
| Rich skill metadata | Extend `.harness/skills/*/SKILL.md` beyond `name` and `scope` with optional description, when-to-use, evidence, and freshness metadata. Use this for readiness output and skill routing. |
| Plugin discovery ergonomics | Borrow the catalog/discovery shape only. Mars skills should remain repo-visible procedural guidance unless promoted into formal built-in tools. |
| Lifecycle checkpoints | Translate into deterministic remediation recipes and typed policy checkpoints, especially under `MH-048`, not arbitrary command or model hooks. |

## Explicit Non-Adoptions

- Do not import OpenHarness code directly. It is a Python application, while
  MARS is constrained to Go, single-binary distribution, and no external
  runtime dependencies.
- Do not load arbitrary project plugin code by default. That conflicts with
  blast-radius containment and the formalized tool creation path.
- Do not copy broad tool sprawl. Mars should keep the mirrored universal tool
  surface small, audited, allowlisted, and trust-checked.
- Do not copy `full_auto` permission semantics. Mars already has the stronger
  `observer`, `contributor`, and `autonomous` trust model.
- Do not create hidden chat memory as product truth. Any compaction carryover
  must be trace-backed or repo-backed.
- Do not prioritize parallel tool execution now. Sequential execution remains
  the safer default for strict trunk and auditability; any future batching
  should be read-only and explicitly trace-recorded.

## Ticket Mapping

| Lesson | Mars destination |
| --- | --- |
| Canonical role/trust/tool inputs for readiness | `MH-043` checked role registry |
| Role readiness preview | New small slice after `MH-043`, or a scoped addition if `MH-043` grows a preview surface |
| Skill metadata and routing | Skill-evolution follow-up; update generated target guidance and scanner tests together |
| Deterministic lifecycle checkpoints | `MH-048` deterministic remediation recipes |
| Orchestrator survey and stuck-work signals | `MH-047` native Orchestrator survey loop |
| Dogfood comparison evidence | `MH-049` dogfood matrix supersession benchmark |
| Context compaction carryover | Dedicated agent-runtime hardening slice after readiness preview |

## Priority Recommendation

Keep the current active-plan order. `MH-047`, `MH-048`, and `MH-049` remain the
next named Mars parity tickets. OpenHarness should influence how those tickets
are implemented, but it should not jump the queue.

The highest-value standalone steal is role readiness preview because it improves
plug-and-play, execution truth, and safety without changing autonomous execution
behavior. Trace-aware compaction is valuable but higher-risk because it touches
the agent loop.

## Source Notes

The comparator review focused on:

- [README](https://github.com/HKUDS/OpenHarness/blob/main/README.md) for product
  positioning and feature surface
- [CLI](https://github.com/HKUDS/OpenHarness/blob/main/src/openharness/cli.py)
  for dry-run readiness behavior
- [query loop](https://github.com/HKUDS/OpenHarness/blob/main/src/openharness/engine/query.py)
  for tool execution and loop behavior
- [compaction service](https://github.com/HKUDS/OpenHarness/blob/main/src/openharness/services/compact/__init__.py)
  for micro/full compaction and carryover metadata
- [skill loader](https://github.com/HKUDS/OpenHarness/blob/main/src/openharness/skills/loader.py)
  and [plugin loader](https://github.com/HKUDS/OpenHarness/blob/main/src/openharness/plugins/loader.py)
  for skill/plugin metadata ergonomics
- [permission modes](https://github.com/HKUDS/OpenHarness/blob/main/src/openharness/permissions/modes.py)
  for contrast with Mars trust levels
