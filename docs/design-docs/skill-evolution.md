# Skill Evolution

**Status:** Accepted
**Date:** 2026-05-02
**Sources:** User direction, [tenets](tenets.md), [self-improvement](self-improvement.md), [self-reflective telemetry](self-reflective-telemetry.md), [mirrored harness and context glossary](mirrored-harness-and-context-glossary.md)
**Related:** [cli-tool-skill-sync.md](cli-tool-skill-sync.md)

## Context

MARS aims to remove the human from the loop by converting repeated human interventions and agent failures into durable system improvements. Not every improvement belongs in a role prompt. If a fix is reusable procedural knowledge, it should usually become a skill.

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

### AD-140: Recursive Improvement Loop Stays Doctrine, Release Publication Becomes A Foundation Skill

> **Superseded 2026-07-21 by T-065/F-018.** The bespoke publication command
> described below is no longer reachable. The source-only skill now governs
> pinned, publication-disabled GoReleaser/Syft snapshots; target repositories
> choose their own producer. The original rationale is retained as history.

The recursive improvement loop is a foundation operating-model rule, not a
single skill. The full loop spans live target replay, evidence review, bounded
source action, deterministic tests, rerun evidence, trunk publication, release
notes, GitHub Release publication, and blocker recording. Because it crosses
several roles, tools, gates, and doctrine surfaces, it remains design doctrine
in [delivery-operating-model.md](delivery-operating-model.md) and
[foundation-deployed-harness-architecture.md](foundation-deployed-harness-architecture.md)
rather than becoming a universal skill.

The release-publication ritual is different. It is a repeated, judgment-heavy
Release Manager procedure with clear stop conditions: generate release notes,
push the release commit and tag, confirm the GitHub Release object, create a
notes-only fallback release when the tag workflow cannot publish one, run asset
verification, and record a blocker when assets are missing. That procedure is
well suited to a compact **foundation skill** before a new deterministic
release-publish tool exists.

Decision:

- Keep the recursive improvement loop as operating doctrine and generated
  target guidance.
- Use `.harness/skills/release-publication/SKILL.md` as the foundation Release
  Manager skill for local release publication, optional GitHub mirroring, and
  missing-asset blocker recording.
- Do not create a universal skill yet; target repos already receive generic
  release discipline, and target release publication may not produce binaries.
- Do not mirror the foundation skill into generated targets yet. The source
  harness publishes cross-platform binary assets, while generated targets may
  publish npm packages, container images, app deployments, notes-only releases,
  or no GitHub Release at all. Mirroring waits until a generic target release
  publication contract exists.
- Keep deterministic publication in the source release tool surface through
  `mars release publish-assets`; the skill owns sequencing,
  optional-mirror judgment, and blocker truth.

### CLI Workflow Skills Stay Synchronized

When a `mars` CLI change affects a reusable workflow, the skill that
teaches that workflow changes in the same commit as the CLI and tool mapping.
The source operating model for this is
[cli-tool-skill-sync.md](cli-tool-skill-sync.md). Skills remain compact, but they
must not point agents at stale commands, flags, examples, or tool choices.

## Current Implementation

The bundle loader reads `.harness/skills/<name>/SKILL.md` with optional frontmatter and role scoping. `run` and `serve` inject loaded skills into the `## SKILLS` context section. The source harness now treats repeated loop and max-turn telemetry as a possible skill gap.

## Required Next Steps

- Add generated target skill templates for self-improvement and ticket completion.
- Add dashboard/API visibility for improvement proposals whose target is `skill`.
- Add evolution safety checks for creating or editing `.harness/skills/`.
