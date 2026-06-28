# Mirrored Harness And Context Glossary

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [Harness Engineering](../references/harness-engineering-agent-first.md), [Symphony reference](../references/open-source-codex-orchestration-symphony.md)

## Context

MARS has two harness surfaces:

1. The source harness in this repository, used to build and evolve MARS.
2. The initialized harness emitted into a target repository by `mars init` or `mars upgrade`.

Those surfaces must feel like the same operating system. A target repo should not receive only roles and a manifest while the source repo keeps the real doctrine in `AGENTS.md`, design docs, references, and context-routing rules. That mismatch causes agents to behave well in the harness repo and poorly in target repos.

OpenAI's Harness Engineering article also reinforces a context design rule: give the agent a compact map and route it to the right file, rather than loading a large manual into every prompt. MARS already has knowledge routes; the initialized target harness now needs a seed glossary that makes those routes useful from day one.

The Symphony article adds the orchestration pattern: task state is the control plane, every eligible task should have an agent/workspace, stalled sessions are restarted or reconciled, and the workflow contract is repo-owned.

## Decisions

### AD-034: Source And Initialized Harnesses Must Mirror Doctrine

`mars init` must create target repo guidance that mirrors the source harness doctrine:

- compact `AGENTS.md` as the first agent entrypoint
- strict trunk workflow
- tickets as repo-native work state
- in-progress work priority
- design decisions in `docs/design-docs/`
- references for agent-first workflow
- context routing via `.harness/knowledge/`

This does not mean every target repo receives all MARS internals. It means every target repo receives the same operating principles and enough structure for agents to work without chat-only context.

### AD-035: Context Glossary Is A Routing Layer

Every initialized target gets `docs/design-docs/context-glossary.md` plus `.harness/knowledge/context-glossary.yaml`.

The glossary is intentionally small. It defines project terms, points to canonical files, and gives agents a retrieval path. It is not a place to paste long architecture documents.

The knowledge route file injects hints such as "when terminology is unclear, read the glossary" into role prompts. The agent then uses file tools to retrieve only what it needs.

### AD-076: Harness Glossary Is Mirrored First-Class Context

The foundation harness and deployed harnesses share a first-class harness
glossary. Core terms such as `mars`, foundation harness, deployed
harness, harness definitions, mirrored harness definitions, tools, mirrored
tools, operating model, foundation operating model, deployed operating model,
tenets, first-class harness definitions, and contextual harness definitions
belong in top-level `AGENTS.md` so agents never have to infer the operating
vocabulary.

The expanded glossary lives in `docs/design-docs/harness-glossary.md`. It is
also a routing layer for contextual harness definitions written as "When doing
X include this: <path>". Agents should expand it autonomously when repeated
language appears in docs, tickets, traces, prompts, or user conversations.

### AD-080: Tools Glossary Is Mirrored First-Class Context

Tool availability and use cases live in `docs/design-docs/tools-glossary.md`.
The tools glossary is mirrored into initialized target harnesses so every LLM
chat can discover which tools exist, when to use them, and what policy applies
without inferring from memory or searching implementation files first.

Every newly created tool must extend the tools glossary in the same change that
implements or exposes it. Tool removals, renames, and material behavior changes
must update the same glossary, generated target defaults, and tests.

### AD-082: Repeated Process Becomes Formal Tool

When a process is repeated, risky, validation-heavy, likely to recur, or spans
foundation and deployed harness boundaries, it should become a formalized tool
instead of remaining ad hoc chat memory. Mirrored formal tools must be listed in
the tools glossary, exposed through generated target defaults where useful, and
covered by tests before roles depend on them.

New built-in tools must originate through `tool_create`. Bypassing
`tool_create` is allowed only as an explicit exception recorded with
`record_decision` and backed by design-doc rationale before implementation is
treated as complete. Shared implementation files are a refactor after
scaffolding, not a reason to skip the governed path.

### AD-139: Foundation And Deployed Harness Architecture

Foundation and deployed harnesses share reusable operating doctrine, but they
do not share every implementation duty. The foundation harness owns the
`mars` source repo, generated defaults, software-factory release
discipline, and runtime improvement loop. The deployed harness owns target
planning, target feature contracts, target tickets, target-specific skills, and
target product evidence.

The runtime substrate is the compiled `mars` binary and its internal
packages. It executes orchestration for both contexts, but it does not decide
doctrine by itself and it must not turn the source harness into the target of
its own agents during a target run.

Generated target guidance should mirror the reusable core: evidence-driven
planning, BDD contracts, ticket truth, feedback routing, tool/skill selection,
and the generic run-review-act-rerun improvement loop. Source-only mechanics,
including `demo-123` as the named source replay and `mars` binary
release asset publication, stay foundation-only unless a target deliberately
adopts an equivalent local policy.

### AD-036: Workflow Contracts Belong In The Repo

Symphony's `WORKFLOW.md` idea maps to MARS as repo-owned workflow artifacts:

- `AGENTS.md` for entrypoint discipline
- `docs/tickets/README.md` for ticket states and completion rules
- `docs/exec-plans/README.md` for plan hygiene
- `.harness/manifest.yaml` for role, tool, trigger, model, and knowledge routing
- `.harness/knowledge/context-glossary.yaml` for compact context routing

Future work may introduce an explicit `WORKFLOW.md`, but v1 should first make these existing artifacts complete and mechanically checked.

### AD-058: Operating Rules Inherit Into Target Harnesses

Operating rules added to the source harness apply to initialized target harnesses unless explicitly marked source-only.

An operating rule is any durable instruction that changes how agents should work: commit discipline, versioning, ticket flow, documentation rules, skill creation, guardrail policy, trust/scoring behavior, release behavior, or context-routing discipline.

When a source operating rule changes, the same task must update:

- source guidance such as `AGENTS.md`, `.cursor/rules/`, design docs, or product specs
- generated target guidance in `internal/scanner/init.go`
- scanner or docs-consistency tests proving the rule is mirrored

Source-only exceptions must be explicit in the rule text. Ambiguity defaults to mirroring.

### AD-061: Architecture And Product Changes Carry Their Rationale

Architectural changes and product features must be documented in repo-owned artifacts with the reason why, not just the description of what changed.

For MARS source changes, the owning artifact is usually a design doc in `docs/design-docs/` plus an index entry in `docs/design-docs/index.md`; user-visible product behavior also updates the relevant `docs/product-specs/` file. For initialized target repos, the same rule applies through generated `AGENTS.md`: architecture decisions belong in `docs/design-docs/`, and product-facing behavior must be captured in a product spec or the owning design doc.

This rule is mirrored because target agents need the same durable rationale trail as source agents. A future agent can only safely evolve the system if it can recover why a feature, guardrail, workflow, or trade-off exists without relying on chat history.

## Implementation Requirements

- `init` creates `AGENTS.md` unless it already exists.
- `init` creates `docs/design-docs/context-glossary.md` and indexes it.
- `init` creates `docs/references/README.md` and the OpenAI Harness Engineering summary.
- `init` creates `.harness/knowledge/context-glossary.yaml`.
- Starter role manifest entries load the glossary route file through `knowledge`.
- `upgrade` fills in missing default harness files while preserving existing target manifest, role prompts, knowledge routes, guardrails, and user docs.
- Repo docs created outside `.harness/` remain user content and are not overwritten by `upgrade`.
- Any new source operating rule is mirrored into generated target guidance unless the rule is explicitly source-only.
- Architecture changes and product features in source and target repos are documented with rationale in design docs or product specs.
- First-class harness glossary definitions are mirrored into source and generated target `AGENTS.md`, while the expanded glossary is mirrored through `docs/design-docs/harness-glossary.md`.

## Consequences

- New target repos receive useful agent guidance immediately, not just a manifest.
- Prompts stay smaller because glossary context is a pointer layer.
- Agents have a standard place to add terminology discovered during work.
- The source harness and initialized harness can evolve together through `mars upgrade`.
- Future rules do not silently improve source-agent behavior while leaving target agents behind.
- Future agents inherit the why behind architectural and product changes, not only the latest file state.

## Open Follow-Ups

- Add a role registry generated into both source and target repos.
- Add doctor checks for missing or stale mirrored harness files.
- Consider a first-class `WORKFLOW.md` once ticket, plan, and glossary artifacts are stable.
- Add scanner-generated glossary seed entries from detected frameworks, package managers, and test commands.
