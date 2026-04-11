# Context Efficiency

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

How the harness keeps prompts **small, relevant, and bounded** while still giving roles enough signal to succeed—complements additive assembly in [agent-runtime.md](agent-runtime.md) (AD-006).

## Context

Local models have finite context and quality degrades with noise. The harness must avoid “stuffing” entire repos into every turn and should route **knowledge** deliberately. This doc covers retrieval strategy, per-role budgets, and how guardrails and routing interact.

Efficiency is a **product feature**: faster jobs, fewer failures, and lower hardware requirements—not only a cost optimization.

## Key Design Decisions

_(No baseline AD IDs; add AD-* rows when choices are frozen.)_

### Topics to resolve

- **Minimal base context:** smallest fixed preamble that every role needs (policies, repo identity, current task, manifest excerpt); everything else is opt-in per turn.
- **Tool-based retrieval vs context stuffing:** prefer tools (search, read file, symbol lookup) over preemptively embedding large trees; define default tool set per role template.
- **Context budgets per role:** token/time ceilings coordinated with agent-runtime max-turn and truncation rules; soft vs hard budget behavior documented.
- **Knowledge routing:** `.harness/knowledge-routes.yaml` maps task classes or paths to curated docs/snippets; validation and staleness checks on `harness doctor`.
- **Guardrail scoping:** which rules are injected into which roles to avoid over-exposing sensitive policies or doubling volume unnecessarily; align with [guardrails.md](guardrails.md) tiers.

### Metrics (future)

Capture **context bytes per turn** and **retrieval hit rate** in traces to guide routing improvements and model bundle sizing.

### Compatibility with AD-006

Additive assembly still applies: retrieval **appends** labeled sections rather than embedding opaque blobs; each block cites source path or tool id for audit.

When trimming, prefer dropping **oldest tool outputs** before dropping the task specification, unless a role policy explicitly allows task summarization.

Document default budgets in `.harness/` examples so new repos inherit sensible ceilings without tuning.

## Discoveries

_(None yet.)_
