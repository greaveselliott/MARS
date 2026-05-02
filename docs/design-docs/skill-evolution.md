# Skill Evolution

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [tenets](tenets.md), [self-improvement](self-improvement.md), [self-reflective telemetry](self-reflective-telemetry.md), [mirrored harness and context glossary](mirrored-harness-and-context-glossary.md)

## Context

Mars Harness aims to remove the human from the loop by converting repeated human interventions and agent failures into durable system improvements. Not every improvement belongs in a role prompt. If a fix is reusable procedural knowledge, it should usually become a skill.

Skills sit between compact role prompts and executable tools:

- role prompts define identity, responsibility, and completion criteria
- skills define reusable workflows and judgment checklists
- tools perform bounded actions
- guardrails enforce rules that must not depend on model memory
- knowledge routes point agents to supporting context without stuffing prompts

## Decisions

### AD-052: Skills Are A First-Class Evolution Target

Self-improvement triage must consider `skill` as an explicit target alongside prompt, process, guardrail, context, inference, manifest, and tool policy.

Create or update a skill when evidence shows a role repeatedly lacks a reusable procedure, for example:

- the same human recovery steps are needed after multiple failed runs
- a role loops or exhausts turns because the workflow is unclear
- dogfood, QA, or checks repeatedly fail for the same class of avoidable setup or verification issue
- an in-progress ticket is repeatedly handed off without a truthful completion or blocker state
- a successful human or agent workaround should be reused by future roles
- a project-specific operating habit is useful across more than one task

Do not create a skill when the right fix is mechanical enforcement, a missing tool, a model/runtime issue, or a one-off ticket.

### AD-053: Skills Must Stay Compact And Scoped

Skills are procedural memory, not manuals. A skill should describe when to use it, the minimum workflow, stop conditions, and expected evidence.

Skill creation rules:

- keep the role prompt small; link or scope the skill instead of expanding the prompt
- prefer role-scoped skills when only one role needs the workflow
- prefer `scope: all` only for repo-wide operating rules
- include concrete completion evidence, not broad advice
- avoid duplicating content already available through knowledge routes
- create a design decision when a new skill changes workflow doctrine

### AD-054: Prompt, Skill, Tool, Guardrail, Or Knowledge Route

The evolution loop uses this decision matrix:

| Evidence | Target |
| --- | --- |
| Role goal, responsibility, or stop condition is wrong | Prompt |
| Reusable multi-step workflow is missing or repeatedly forgotten | Skill |
| Rule must be enforced regardless of model behavior | Guardrail |
| Deterministic action is repeated, error-prone, or needs validation | Tool |
| Agent needs to know where context lives | Knowledge route or glossary |
| Runtime, model, context size, or hardware profile is failing | Inference/config |
| Role schedule, tools, chains, or trust settings are wrong | Manifest/tool policy |
| Work itself is incomplete or blocked | Ticket or exec plan |

### AD-055: Mirrored Target Harnesses Need Skill Guidance

Initialized target repos must receive the same skill-evolution doctrine. The source harness and generated target harness should both teach agents when to add `.harness/skills/<name>/SKILL.md`, because target-specific skills are one of the safest ways to reduce future human intervention without bloating prompts.

## Current Implementation

The bundle loader reads `.harness/skills/<name>/SKILL.md` with optional frontmatter and role scoping. `run` and `serve` inject loaded skills into the `## SKILLS` context section. The source harness now treats repeated loop and max-turn telemetry as a possible skill gap.

## Required Next Steps

- Add generated target skill templates for self-improvement and ticket completion.
- Add dashboard/API visibility for improvement proposals whose target is `skill`.
- Add evolution safety checks for creating or editing `.harness/skills/`.
