# Mirrored Harness And Context Glossary

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [Harness Engineering](../references/harness-engineering-agent-first.md), [Symphony reference](../references/open-source-codex-orchestration-symphony.md)

## Context

Mars Harness has two harness surfaces:

1. The source harness in this repository, used to build and evolve Mars Harness.
2. The initialized harness emitted into a target repository by `mars-harness init` or `mars-harness upgrade`.

Those surfaces must feel like the same operating system. A target repo should not receive only roles and a manifest while the source repo keeps the real doctrine in `AGENTS.md`, design docs, references, and context-routing rules. That mismatch causes agents to behave well in the harness repo and poorly in target repos.

OpenAI's Harness Engineering article also reinforces a context design rule: give the agent a compact map and route it to the right file, rather than loading a large manual into every prompt. Mars Harness already has knowledge routes; the initialized target harness now needs a seed glossary that makes those routes useful from day one.

The Symphony article adds the orchestration pattern: task state is the control plane, every eligible task should have an agent/workspace, stalled sessions are restarted or reconciled, and the workflow contract is repo-owned.

## Decisions

### AD-034: Source And Initialized Harnesses Must Mirror Doctrine

`mars-harness init` must create target repo guidance that mirrors the source harness doctrine:

- compact `AGENTS.md` as the first agent entrypoint
- strict trunk workflow
- tickets as repo-native work state
- in-progress work priority
- design decisions in `docs/design-docs/`
- references for agent-first workflow
- context routing via `.harness/knowledge/`

This does not mean every target repo receives all Mars Harness internals. It means every target repo receives the same operating principles and enough structure for agents to work without chat-only context.

### AD-035: Context Glossary Is A Routing Layer

Every initialized target gets `docs/design-docs/context-glossary.md` plus `.harness/knowledge/context-glossary.yaml`.

The glossary is intentionally small. It defines project terms, points to canonical files, and gives agents a retrieval path. It is not a place to paste long architecture documents.

The knowledge route file injects hints such as "when terminology is unclear, read the glossary" into role prompts. The agent then uses file tools to retrieve only what it needs.

### AD-036: Workflow Contracts Belong In The Repo

Symphony's `WORKFLOW.md` idea maps to Mars Harness as repo-owned workflow artifacts:

- `AGENTS.md` for entrypoint discipline
- `docs/tickets/README.md` for ticket states and completion rules
- `docs/exec-plans/README.md` for plan hygiene
- `.harness/manifest.yaml` for role, tool, trigger, model, and knowledge routing
- `.harness/knowledge/context-glossary.yaml` for compact context routing

Future work may introduce an explicit `WORKFLOW.md`, but v1 should first make these existing artifacts complete and mechanically checked.

## Implementation Requirements

- `init` creates `AGENTS.md` unless it already exists.
- `init` creates `docs/design-docs/context-glossary.md` and indexes it.
- `init` creates `docs/references/README.md` and the OpenAI Harness Engineering summary.
- `init` creates `.harness/knowledge/context-glossary.yaml`.
- Starter role manifest entries load the glossary route file through `knowledge`.
- `upgrade` fills in missing default harness files while preserving existing target manifest, role prompts, knowledge routes, guardrails, and user docs.
- Repo docs created outside `.harness/` remain user content and are not overwritten by `upgrade`.

## Consequences

- New target repos receive useful agent guidance immediately, not just a manifest.
- Prompts stay smaller because glossary context is a pointer layer.
- Agents have a standard place to add terminology discovered during work.
- The source harness and initialized harness can evolve together through `mars-harness upgrade`.

## Open Follow-Ups

- Add a role registry generated into both source and target repos.
- Add doctor checks for missing or stale mirrored harness files.
- Consider a first-class `WORKFLOW.md` once ticket, plan, and glossary artifacts are stable.
- Add scanner-generated glossary seed entries from detected frameworks, package managers, and test commands.
