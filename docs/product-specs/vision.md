# MARS Product Vision

**Status:** Accepted
**Updated:** 2026-05-02
**Owner:** MARS maintainers
**Sources:** [tenets](../design-docs/tenets.md), [Mars relationship](mars-relationship.md), [product surface](product-surface.md)

## Product Promise

MARS is a self-hosted autonomous AI delivery system. A user points it at a git checkout, runs one setup path, and gets an agent-operated delivery loop: planning, ticket creation, implementation, verification, documentation, scoring, trust management, guardrail enforcement, and visible operational telemetry.

The product must feel zero-config. It detects local hardware, chooses a sensible local inference profile, scaffolds a target harness when needed, drains existing in-progress work before new backlog work, and records decisions in the repo so future agents inherit the context.

## What The Product Must Be

### Local-first autonomous delivery

Inference runs through local open-weight models by default, with llama.cpp managed as a subprocess. Model, binary, and performance choices should be automatic for normal users and overrideable for advanced users.

Local-first is the default, not a claim that every configured path stays on the
machine. Cloud model routes send selected assembled context, model messages,
and tool data to the chosen provider under its terms. GitHub, JIRA, remote MCP,
updates, model downloads, and opt-in telemetry reporting make the network
requests required for their configured purpose.

The user should not need to tune thread counts, context sizes, model quantization, or parallel slots before seeing a useful run. Doctor checks explain what is missing and how to fix it.

### Strict trunk by design

The canonical unit of progress is a small semantic commit on `main`, followed by a push. Review, checks, comments, and statuses are signals around trunk work, not a separate delivery model.

Every role, generated target harness, trust capability, scoring rule, and product doc should align with that model.

### Repo as system of record

Plans, decisions, tickets, design docs, traces, generated guidance, and harness evolution must be recoverable from the repo and the harness database. Chat-only decisions are not product state.

Product specs describe what MARS promises. Design docs describe why key implementation decisions exist. Exec plans describe active delivery work. Tickets describe discrete work items.

### Mirrored source and target harnesses

MARS has two surfaces:

- the source harness in this repository
- the target harness emitted by `mars init` and refreshed by `mars upgrade`

Those surfaces must share the same doctrine: compact agent entrypoints, strict trunk workflow, ticket discipline, design decisions, references, and context routing. Target repos should not receive a thin manifest while this repo keeps the real operating model hidden in source-only docs.

### Context efficiency

Agents should receive a compact map and retrieve narrow supporting context on demand. `AGENTS.md`, `.harness/manifest.yaml`, `.harness/knowledge/`, `.harness/skills/`, tickets, design docs, references, and traces form the routing layer.

The product should avoid prompt-stuffing large manuals into every role. It should instead maintain glossary and knowledge-route files that tell the agent where to look, plus compact skills that teach reusable workflows.

### Self-reflective improvement

The harness grades both its own process and the feature work it builds. Outcomes such as completed commits, failed checks, guardrail blocks, timeouts, noops, stuck tickets, human follow-up, and reverts become scoring and telemetry signals.

The product must proactively triage those signals into improvement targets: prompt, skill, process, guardrail, context, inference, manifest, tool policy, scanner, ticket flow, or generated target guidance. Direct evolution is allowed only inside deterministic trust and safety bounds.

## Who It Serves

MARS is for developers and small teams who want autonomous delivery with local
inference as the default and explicit control over any hosted model or external
integration they enable. The primary user has a local machine with Apple
Silicon, NVIDIA, or AMD ROCm hardware and wants agents to operate normal
repository workflows with minimal ceremony.

Secondary users include teams that want optional remote-code-host telemetry, local dashboards, score history, trust controls, and durable documentation for autonomous work.

## What It Is Not

- Not an IDE autocomplete product.
- Not a hosted service.
- Not a generic chat wrapper over a repository.
- Not a replacement for tests, checks, or human product direction.
- Not Mars-specific. Mars is the prototype and first demanding customer; MARS is the reusable product.

## Product Success Measures

- A new target repo can be initialized, scanned, given starter tickets, and run by agents without hand-written harness setup.
- In-progress tickets are completed, explicitly blocked, or returned to a truthful state before new backlog work is claimed.
- `doctor` explains local inference, model, database, guardrail, workflow, and optional integration health in actionable terms.
- Scores and trust levels influence role autonomy and self-improvement work.
- Guardrails and safety checks are enforced mechanically before mutating actions.
- The generated target harness remains close enough to the source harness doctrine that agents behave consistently in both places.
- Product specs stay current through metadata, index coverage, and docs-consistency tests.

## North Star

MARS supersedes the original Mars meta-harness by turning Mars's proven operating habits into a local-first product: one repo-owned workflow, one autonomous queue, one visible scoring and trust system, one mirrored target harness, and a feedback loop that keeps improving the process instead of only completing individual tickets.
