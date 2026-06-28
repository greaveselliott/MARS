# Reference: Harness Engineering - Leveraging Codex in an Agent-First World

**Source:** [openai.com/index/harness-engineering](https://openai.com/index/harness-engineering/)
**Author:** Ryan Lopopolo, Member of the Technical Staff at OpenAI
**Published:** February 11, 2026
**Source verified:** 2026-05-02
**Type:** Engineering article
**Carry-over source:** `../mars/docs/references/harness-engineering-agent-first.md`

## Summary

OpenAI's Harness team describes building an internal beta product where Codex wrote the product code, tests, CI, docs, observability, and internal tooling. A small human team steered the work by shaping the environment, documenting intent, enforcing constraints, and building feedback loops.

The article is foundational for MARS because it describes the same product shape this repository is trying to make self-hosted and reusable: an agent-first delivery system where humans set direction, the repository is the source of truth, and failures become better tools, rules, and context.

The article's PR-based workflow is not a MARS default. MARS translates those lessons into strict trunk-based development: semantic commits directly to `main`, push after each completed step, and optional GitHub status/check/comment integration.

## Key Concepts For MARS

### Humans steer, agents execute

The important shift is not "AI writes code" as a novelty. It is that human engineering time moves up a layer: specifying intent, designing constraints, reviewing outcomes, and converting failures into durable system improvements.

Harness implication: roles should not merely be prompted to "try harder." When a role fails, the system should ask what context, tool, guardrail, test, score signal, or deterministic remediation path was missing.

### AGENTS.md is a map, not the encyclopedia

The article argues for a short entrypoint that routes agents to deeper sources instead of stuffing every rule into one monolithic instruction file.

Harness implication:

- `AGENTS.md` stays compact and points to canonical docs.
- Generated target repos need their own compact `AGENTS.md`.
- Context assembly should use routes, manifests, tickets, traces, and design docs instead of dumping the whole repository into the prompt.
- Docs consistency checks should verify that the map still points to real, current sources.

### The repository is the system of record

Knowledge outside the repo is invisible to a running agent. Decisions, plans, failures, test evidence, quality scores, and operating rules need to live in versioned files or machine-readable state that the harness can retrieve.

Harness implication:

- Conversations become design docs, tickets, exec plans, or reference notes.
- Score and trust state should have repo-visible exports where useful.
- Failed jobs should produce traces and, when actionable, tickets.
- "Document all decisions" is not ceremony; it is how future agents inherit context.

### Progressive disclosure compounds

Agents need a stable map and then narrow, relevant detail on demand. The OpenAI article's docs layout directly influenced Mars and should continue into MARS.

Harness implication:

- Keep `docs/design-docs/`, `docs/exec-plans/`, `docs/tickets/`, `docs/references/`, and `docs/generated/` as navigable first-class areas.
- Add knowledge routes for Harness domains such as agent runtime, queue, trust, scoring, guardrails, inference, scanner, and dashboard.
- Generated bundles should carry the same progressive-disclosure pattern into target projects.

### Make the system legible to agents

OpenAI invested in making UI state, logs, metrics, traces, and local development environments accessible to Codex. The lesson is that agents can only validate what they can inspect.

Harness implication:

- Terminal job semantics, traces, logs, checks, queue state, and guardrail outcomes must be easy for roles to read.
- Dogfood runs should record enough evidence for the Engineer to reproduce and complete the task rather than opening more stale tickets.
- The dashboard and CLI should expose machine-readable state, not only human-friendly prose.

### Enforce invariants mechanically

The article's durable lesson is that documentation alone is not enough. Rules that matter should become linters, structural tests, policy checks, doctor checks, or tool-execution guardrails.

Harness implication:

- Hard guardrails must execute in tools, not only in prompts.
- Role tool allowlists must fail closed.
- Secret scans, out-of-root write checks, destructive git protection, and blast-radius limits should run before commit or push.
- Error messages should include remediation instructions that are useful to the next agent run.

### Failures feed back into the harness

When Codex struggles, the response should be to improve the environment. Mars adopted this as intervention debt; MARS should make it mechanical.

Harness implication:

- Human intervention, repeated job failure, stuck in-progress tickets, and guardrail blocks should create or update intervention-debt tickets.
- Planner and Orchestrator should promote repeated failures into roadmap work.
- Scoring should reward completed work and penalize noops, stale handoffs, timeouts, and regressions.

### Throughput changes integration philosophy

OpenAI's article describes short-lived PRs and cheap follow-up correction. MARS keeps the same small-change philosophy while replacing PRs with strict trunk.

Harness implication:

- The default unit of progress is a small semantic commit on `main`.
- Completed steps are pushed immediately.
- Review and checks remain important, but they should be trunk-native signals rather than branch/PR gates.
- Reverts, follow-up commits touching the same files, and failed checks become score signals.

## Direct MARS Requirements Derived From This Article

1. `AGENTS.md` remains a concise routing map.
2. Docs, tickets, traces, scores, and decisions are treated as system memory.
3. Context assembly prefers retrieval and routes over prompt stuffing.
4. Generated target repos receive agent-legible docs and ticket structure from day one.
5. Hard rules graduate from prose into mechanical checks.
6. Failures produce intervention debt, deterministic fixes, or updated guardrails.
7. Dogfood evidence must be reproducible by the role that fixes it.
8. Strict trunk is the canonical integration model for MARS.

## Notes For Future Work

- The OpenAI article assumes Codex and PR-oriented development. MARS must preserve the agent-first operating model while staying local-first, model-flexible, and strict-trunk by default.
- The article's observability and UI-legibility sections should inform future dashboard, trace, and browser-testing work.
- The article is the strongest external support for carrying references from Mars into MARS rather than treating them as historical baggage.
