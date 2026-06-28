# Context Efficiency

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** MARS contributors

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

### AD-288: Context Budgeting Clamps To The Served Window With Calibrated Estimation

The 2026-06-12 demo-12 (package-managed frontend) and demo-13 (existing-repo
maintenance) baselines both wedged deterministically in engineer: prompts of
33,281 / 32,883 / 32,923 served tokens against the balanced coding tier's
32,768-token window, rejected by llama.cpp as non-retryable
`exceed_context_size_error` (T-032). Trace forensics showed the agent loop's
pruner never fired: at the moment the server counted 33,281 tokens, the
character-heuristic estimator reported 26,188 (~3.15 chars/token actual on
package.json/tool-JSON/source content vs the assumed 4), so the loop believed
it was ~6.5k tokens under the window. A second latent defect compounded it:
the loop's window was taken from manifest `context_size` (defaulting to
32,768) and never from what the inference tier actually serves, so reasoning
roles were budgeted at a quarter of their served 131,072 window and any
profile change could silently invalidate the assumption.

Overflow is now impossible by construction through three cooperating rules:

1. **Conservative estimation.** `llm.EstimateTokens` uses ~3 chars/token,
   calibrated to over-estimate the measured worst case (the demo-12 wedge is
   encoded as a regression floor test). Budget math must over-estimate, never
   under-estimate.
2. **Serving-window wiring.** The inference router exposes the per-tier
   served context length (`ContextWindowForRoleModel`) and the server
   executor budgets the agent loop against it unless the manifest explicitly
   overrides `context_size`. Coding-tier roles budget at 32,768; reasoning
   roles at their real 131,072. When a validation or operator path starts a
   server with multiple parallel slots, the router scales the server's total
   context by the slot count so each slot still serves that advertised window.
3. **Server-reported clamp with prune-and-retry.** The LLM client returns a
   typed `ContextSizeError` carrying llama.cpp's `n_prompt_tokens`/`n_ctx`.
   The client and loop never retry the doomed request verbatim; instead the
   loop clamps its working window to the server-reported `n_ctx`, rescales
   the prune target by the measured estimate-to-served ratio, prunes oldest
   tool output first per AD-006 trimming order, and retries (bounded
   attempts). Only when nothing remains prunable does the job fail, with the
   typed error still classified `context_overflow` by telemetry.

The balanced coding-tier window was deliberately **not** raised in this
change: raising ctx without budget enforcement only moves the wedge, doubles
coding-tier KV-cache cost, and would have changed the AD-285 model identity
under the replays that validate this fix. If post-fix replays show engineer
convergence is starved by the ~14k-token system prompt share of the 32k
window, raising the window or slimming the assembled sections is a separate
measured slice. Known residual: the pruner never touches the system prompt
or the protected recent tail, so a system prompt larger than the served
window still fails (cleanly, with the typed error); the assembler-side
budget for that case remains a "Topics to resolve" item below.

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

- **2026-06-12 — T-032 overflow forensics:** the demo-12 engineer wedge (job
  `0b93881f`, trace `tr-1781225306000294000`) recorded a final turn estimate
  of 26,188 tokens while llama.cpp counted 33,281 for the same content —
  the first measured calibration of the chars/token heuristic against the
  Qwen3-Coder tokenizer on package-managed JS content (~3.15 chars/token).
  AD-288 derives its 3-chars/token floor and the server-reported clamp from
  this measurement.

- **2026-04-11 — MH-004 assembler:** `internal/context.Assemble` builds fixed-order sections (`## ROLE`, `## GUARDRAILS`, `## KNOWLEDGE ROUTES`, `## TRIGGER CONTEXT`, `## REPO SUMMARY`), omits optional blocks when empty, filters guardrails by `Scope` (empty/`all` = global), formats knowledge as bullet lines `When working on X, read Y`, and applies a token **budget** using `llm.EstimateTokens` by iteratively shrinking lowest-priority bodies first (`repo` < `trigger` < `knowledge`; `role` is never truncated; shrinking stops if the budget still cannot be met without editing the role text).
- **2026-05-02 — Context glossary default:** `mars init` now emits a target `AGENTS.md`, `docs/design-docs/context-glossary.md`, and `.harness/knowledge/context-glossary.yaml`. Default roles reference the glossary route file so the base prompt carries a compact map, not full project doctrine.
- **2026-05-21 — Run metadata grounding:** `internal/context.Assemble` can include a compact `## RUN METADATA` section with current date, timestamp, and timezone. Server jobs populate it from the executor clock, and budget trimming preserves it alongside the role prompt so dated artifacts cannot drift to stale model-memory dates when shell time commands are unavailable.
