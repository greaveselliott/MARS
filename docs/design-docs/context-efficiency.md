# Context Efficiency

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

How the harness keeps prompts **small, relevant, and bounded** while still giving roles enough signal to succeed—complements additive assembly in [agent-runtime.md](agent-runtime.md) (AD-006).

## Context

Local models have finite context and quality degrades with noise. The harness must avoid “stuffing” entire repos into every turn and should route **knowledge** deliberately. This doc covers retrieval strategy, per-role budgets, and how guardrails and routing interact.

Efficiency is a **product feature**: faster jobs, fewer failures, and lower hardware requirements—not only a cost optimization.

## Key Design Decisions

### AD-208: Runtime Date Grounding Belongs In Non-Droppable Base Context

Live target validation on 2026-05-21 showed that review and dogfood roles can
invent stale report dates when validation policy blocks generic `date` shell
commands. Date guessing corrupts evidence paths and makes later release or
quality reviews look older or newer than the run that produced them.

The context assembler now supports a compact `## RUN METADATA` section supplied
by the executor for every server job. It includes `current_date`,
`current_time`, and timezone, plus an explicit instruction to use that date for
evidence files, reports, release entries, and ticket timestamps. The section is
treated like role identity during budget trimming: lower-value repo, trigger,
knowledge, skills, and guardrail bodies may shrink before the run date is
dropped. This keeps evidence temporally grounded without opening broad shell
discovery in review roles.

### Topics to resolve

- **Minimal base context:** smallest fixed preamble that every role needs (policies, run metadata, repo identity, current task, manifest excerpt); everything else is opt-in per turn.
- **Tool-based retrieval vs context stuffing:** prefer tools (search, read file, symbol lookup) over preemptively embedding large trees; define default tool set per role template.
- **Context budgets per role:** token/time ceilings coordinated with agent-runtime max-turn and truncation rules; soft vs hard budget behavior documented.
- **Knowledge routing:** `.harness/knowledge/*.yaml` maps task classes or paths to curated docs/snippets; validation and staleness checks on `harness doctor`.
- **Context glossary:** initialized repos receive `docs/design-docs/context-glossary.md` and `.harness/knowledge/context-glossary.yaml`; the injected prompt contains only route hints, while agents retrieve the glossary or deeper docs only when needed.
- **Guardrail scoping:** which rules are injected into which roles to avoid over-exposing sensitive policies or doubling volume unnecessarily; align with [guardrails.md](guardrails.md) tiers.

### Metrics (future)

Capture **context bytes per turn** and **retrieval hit rate** in traces to guide routing improvements and model bundle sizing.

### Compatibility with AD-006

Additive assembly still applies: retrieval **appends** labeled sections rather than embedding opaque blobs; each block cites source path or tool id for audit.

When trimming, prefer dropping **oldest tool outputs** before dropping the task specification, unless a role policy explicitly allows task summarization.

Document default budgets in `.harness/` examples so new repos inherit sensible ceilings without tuning.

## Discoveries

- **2026-04-11 — MH-004 assembler:** `internal/context.Assemble` builds fixed-order sections (`## ROLE`, `## GUARDRAILS`, `## KNOWLEDGE ROUTES`, `## TRIGGER CONTEXT`, `## REPO SUMMARY`), omits optional blocks when empty, filters guardrails by `Scope` (empty/`all` = global), formats knowledge as bullet lines `When working on X, read Y`, and applies a token **budget** using `llm.EstimateTokens` by iteratively shrinking lowest-priority bodies first (`repo` < `trigger` < `knowledge`; `role` is never truncated; shrinking stops if the budget still cannot be met without editing the role text).
- **2026-05-02 — Context glossary default:** `mars-harness init` now emits a target `AGENTS.md`, `docs/design-docs/context-glossary.md`, and `.harness/knowledge/context-glossary.yaml`. Default roles reference the glossary route file so the base prompt carries a compact map, not full project doctrine.
- **2026-05-21 — Run metadata grounding:** `internal/context.Assemble` can include a compact `## RUN METADATA` section with current date, timestamp, and timezone. Server jobs populate it from the executor clock, and budget trimming preserves it alongside the role prompt so dated artifacts cannot drift to stale model-memory dates when shell time commands are unavailable.
