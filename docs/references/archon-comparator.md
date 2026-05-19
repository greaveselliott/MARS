# Reference: Archon Comparator

**Status:** Reference
**Date:** 2026-05-19
**Source:** [coleam00/Archon](https://github.com/coleam00/Archon)
**Mars Harness relevance:** Comparator for deterministic AI coding workflows,
workflow DAG ergonomics, worktree isolation, web control-plane UX, provider
adapter boundaries, and the risks of cloud-agent-first defaults.

## Verdict

Archon is a useful comparator, not a dependency or replacement. It is strongest
as a workflow engine for AI coding agents: YAML-defined plans, DAG execution,
loop nodes, bash nodes, provider-specific AI nodes, worktree isolation, web
chat, workflow monitoring, and workflow-builder UX.

Mars Harness remains aimed at a different product center: a local-first,
self-hosted autonomous delivery system with governed roles, trust and scoring,
BDD evidence, repo-owned operating doctrine, universal tools, release
discipline, and local inference by default.

The adoption rule is:

- steal workflow ergonomics and execution visibility
- keep Mars authority, trust, BDD, and release gates
- reject cloud-agent, PR-first, or arbitrary-script defaults that weaken the
  local-first software-factory promise

## Key Findings

| Finding | Why It Matters For Mars |
| --- | --- |
| Archon has a clear workflow DSL with `steps`, `loop`, and `nodes` modes. | Mars role dispatch is strong for authority boundaries, but deterministic repeatable procedures would benefit from a first-class workflow layer. |
| DAG nodes can mix AI command/prompt work with deterministic bash work. | Mars can model deterministic remediation recipes without asking one role prompt to remember every phase. |
| Independent DAG layers run concurrently and capture node outputs for later substitutions. | Useful for read-only audits, reviews, and fan-out validation, but Mars should gate mutation carefully. |
| Workflow routing can still be LLM-selected from descriptions. | Mars should prefer deterministic trigger/routing rules where safety or release state is involved. |
| Worktree isolation is a first-class execution concept. | Mars strict-trunk remains the default, but optional worktree execution could reduce blast radius for risky, concurrent, or review-before-merge work. |
| The Web UI exposes chat, workflow history, tool-call visualization, and workflow-builder concepts. | Mars dashboard can borrow run-graph visibility without changing the source-of-truth model. |
| Archon is Claude/Codex/Pi-provider oriented and Claude-heavy by default. | This conflicts with Mars Harness' local-open-model default and no data exfiltration promise. |
| Archon has useful validation ergonomics for workflows, commands, scripts, provider/model compatibility, and referenced resources. | Mars should make harness/workflow readiness errors actionable before model execution. |
| Archon docs show small drift: architecture prose referenced 7 database tables while database docs listed 8. | Reinforces Mars' no-stale-docs and docsync audit direction. |

## Useful Bits To Steal

| Archon idea | Mars-shaped adoption |
| --- | --- |
| YAML workflow DAGs | Add a `.harness/workflows/` layer for deterministic recipes that can invoke roles, tools, checks, and approval gates while preserving role authority boundaries. |
| Workflow node types | Support at least role/tool/check nodes before considering arbitrary shell nodes; keep mutating actions trust-gated. |
| Trigger rules and conditional nodes | Use for remediation recipes, release preflights, docsync gates, and review fan-out/fan-in without encoding every branch in prompts. |
| Node output references | Let later steps consume structured outputs from earlier deterministic tools or role dispositions. |
| Worktree isolation | Consider optional worktree-backed execution for high-risk or concurrent work, while strict trunk remains the default integration model. |
| Run graph UX | Extend the dashboard with workflow/role run graphs, current node, artifacts, traces, retries, blockers, and evidence links. |
| Resource validation | Add preflight checks for workflow files, referenced roles/tools/skills, model routes, guardrail files, and generated target drift. |
| Approval nodes | Recast as Mars trust/guardrail gates rather than generic human pause points. |

## Explicit Non-Adoptions

- Do not make Claude Code, Codex cloud, Pi, or any external coding assistant the
  default runtime. Mars Harness should keep local OpenAI-compatible inference as
  the default path.
- Do not switch the generated delivery model to branch/PR-first operation.
  Strict trunk remains the generated Mars default.
- Do not allow arbitrary project scripts or provider extensions to bypass
  Mars' tool registry, trust policy, guardrails, and secret scanning.
- Do not copy Archon's broad process dependencies as a runtime requirement.
  Mars' single Go binary and embedded SQLite constraint is a core product
  differentiator.
- Do not treat LLM workflow selection as authoritative for safety-critical,
  release, or mutating paths.
- Do not create a parallel Archon roadmap. Fold useful lessons into existing
  Mars parity, deterministic remediation, runtime, and dashboard tickets.

## Action Tracker

| Lesson | Possible Mars destination | Status |
| --- | --- | --- |
| First-class deterministic workflow recipes | `MH-048` deterministic remediation recipes, then a dedicated workflow-design slice if needed | Reference input only |
| Role/tool/check workflow nodes | Future F-006/F-005 scenario after current dispatch model stabilizes | Not scheduled |
| Optional worktree execution | Future blast-radius or concurrency design doc; must preserve strict-trunk default | Not scheduled |
| Run graph and workflow visibility | F-010 dashboard/control-plane follow-up | Not scheduled |
| Workflow/resource validation | `doctor --repo`, `run --dry-run`, and possible workflow validator | Not scheduled |
| Approval gates mapped to trust | F-007 guardrails and F-008 scoring/trust follow-up | Not scheduled |
| Docs drift comparator evidence | Docsync/no-stale-docs validation rationale | Captured here |

## Priority Recommendation

Keep the current Mars active-plan order. The highest-value Archon lesson for the
near term is deterministic workflow ergonomics for remediation and release
preflight paths. This should strengthen `MH-048` rather than displace it.

The next highest-value lesson is run visibility: users should be able to see
what the harness is doing, which evidence it produced, and which gate is
blocking progress without reading raw traces first.

## Source Notes

The comparator review focused on:

- [README](https://github.com/coleam00/Archon/blob/dev/README.md) for product
  positioning, setup, workflow examples, and Web UI surface.
- [CLAUDE.md](https://github.com/coleam00/Archon/blob/dev/CLAUDE.md) for
  operating rules, engineering constraints, git workflow, and validation
  commands.
- [Architecture](https://github.com/coleam00/Archon/blob/dev/packages/docs-web/src/content/docs/reference/architecture.md)
  for platform adapters, orchestrator, provider abstraction, isolation, and
  database shape.
- [Workflow YAML reference](https://github.com/coleam00/Archon/blob/dev/.claude/docs/workflow-yaml-reference.md)
  for steps, loops, DAG nodes, trigger rules, conditions, substitutions, and
  model/provider controls.
- [Default idea-to-PR workflow](https://github.com/coleam00/Archon/blob/dev/.archon/workflows/defaults/archon-idea-to-pr.yaml)
  for end-to-end planning, implementation, validation, PR finalization, review
  fan-out, synthesis, and fix phases.
- [Provider contract](https://github.com/coleam00/Archon/blob/dev/packages/providers/src/types.ts)
  for Claude, Codex, Pi, streaming chunks, tool events, and provider defaults.
- [Worktree provider](https://github.com/coleam00/Archon/blob/dev/packages/isolation/src/providers/worktree.ts)
  for git worktree isolation, branch lifecycle, cleanup, and ownership checks.
- [Database reference](https://github.com/coleam00/Archon/blob/dev/packages/docs-web/src/content/docs/reference/database.md)
  for SQLite/PostgreSQL state and workflow-run/event tables.
