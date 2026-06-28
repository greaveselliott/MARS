# Reference: An Open-Source Spec For Codex Orchestration: Symphony

**Source:** [openai.com/index/open-source-codex-orchestration-symphony](https://openai.com/index/open-source-codex-orchestration-symphony/)
**Authors:** Alex Kotliarskyi, Victor Zhu, and Zach Brock
**Published:** April 27, 2026
**Source verified:** 2026-05-02
**Type:** Engineering article and orchestration specification

## Summary

OpenAI describes Symphony, a minimal orchestration specification that turns an issue tracker into a control plane for coding agents. The article frames a shift from supervising individual coding sessions to supervising deliverables: tickets, states, workspaces, retries, and review packets.

MARS should treat Symphony as a reference for native orchestration, not as a dependency. The implementation remains local-first, strict-trunk, and repo-native, but the orchestration lessons map cleanly onto Harness queue, scheduler, ticket, trace, and dashboard primitives.

## Concepts For MARS

### Tickets are the control plane

Symphony uses task states as the work state machine. MARS already uses markdown tickets; the next step is to make ticket state mechanically drive eligibility, priority, retries, and completion.

Harness implication: in-progress tickets are not passive files. They are active orchestration state and should be drained before new backlog work.

### One eligible task should have one active workspace or run

Symphony guarantees active work receives an agent session and workspace. MARS should translate this into queue ownership: every eligible in-progress task has an assigned run, blocked state, or scheduled retry.

Harness implication: stale in-progress tickets, crashed runs, and silent handoffs are orchestrator failures, not normal backlog drift.

### Workflow belongs in the repository

Symphony captures workflow policy in repo-owned `WORKFLOW.md`. MARS currently spreads the contract across `AGENTS.md`, `.harness/manifest.yaml`, ticket docs, exec-plan docs, and knowledge routes.

Harness implication: those files must be generated together, checked together, and eventually may be summarized or formalized as a `WORKFLOW.md`.

### Agents need objectives, tools, and context

The article notes that rigid state-machine roles were too limiting. Agents performed better when given objectives plus the tools and context to complete the job.

Harness implication: keep role prompts goal-oriented, but make tool permissions, trust, guardrails, score signals, and context routes explicit and mechanically enforced.

### Failures become harness improvements

When Symphony-style work misses the mark, OpenAI describes adding guardrails, skills, end-to-end tests, smoke tests, and clearer documentation.

Harness implication: MARS should convert repeated failures into intervention-debt tickets, guardrails, deterministic remediation recipes, and generated guidance updates.

### App-server style orchestration is a useful future shape

The article references programmatic agent control through a documented JSON-RPC-like app-server mode. MARS already owns its runtime loop for local models; the lesson is the same: orchestration should talk to agents through structured events and status, not terminal scraping.

Harness implication: traces, tool calls, token accounting, run status, and ticket transitions should be structured APIs inside the harness.

## Strict-Trunk Translation

Symphony discusses PRs because OpenAI's workflow uses PRs. MARS translates the pattern:

| Symphony concept | MARS translation |
| --- | --- |
| Issue tracker as control plane | Markdown tickets plus queue state |
| Dedicated workspace per issue | Bounded run/session ownership per ticket and repo |
| PR links and review state | Commit, trace, check/status, and ticket completion state |
| Merging state | Semantic commit to `main`, push `main`, record score |
| Workflow prompt | `AGENTS.md`, manifest, ticket docs, exec-plan docs, knowledge routes |
| Stalled session restart | Orchestrator retry, blocker ticket, or intervention-debt ticket |

## Requirements Derived From This Reference

1. Treat in-progress tickets as the highest-priority orchestration state.
2. Detect stale active work and crashed or incomplete runs.
3. Keep workflow contracts in repo-owned files.
4. Prefer structured run events and metrics over terminal-only observation.
5. Feed orchestration misses back into guardrails, docs, tests, and deterministic recipes.
