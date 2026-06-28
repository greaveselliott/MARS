# Role Customization

**Status:** Accepted
**Date:** 2026-05-02

## Context

MARS ships starter roles so a new target repo has an agent team on day one. Those roles are useful defaults, but they are not perfect universal agents. A project may need different roles, fewer roles, stricter tool access, different schedules, domain-specific prompts, or custom knowledge routes.

The product must therefore treat generated role prompts and manifests as user-owned configuration after initialization. Otherwise `mars upgrade` can erase the exact tuning users need to make the harness effective.

## Decisions

### AD-046: Shipped Roles Are Starter Agents

The default role set is a seed operating model, not a claim of universal correctness. Users may edit, replace, remove, or add roles in `.harness/manifest.yaml` and `.harness/roles/`.

### AD-047: Target Role Configuration Is User-Owned

Once generated into a target repo, role prompts, manifest configuration, knowledge routes, and guardrails are owned by that repo. Harness upgrades must preserve existing files by default.

### AD-048: Upgrade Fills Missing Defaults, It Does Not Retune Agents Silently

`mars upgrade` writes missing starter files but does not overwrite existing manifest, role prompts, knowledge routes, guardrails, or target docs. Adopting newer default prompt wording is an explicit user choice through comparison and deliberate edits.

## Implementation Requirements

- `scanner.Upgrade` preserves existing target harness files.
- Missing starter prompts and support files are created when absent.
- Bundle docs explain that shipped roles are configurable starter examples.
- Product specs describe role customization as part of the product contract.

## Consequences

- User tuning survives harness upgrades.
- The harness can still add missing scaffold files for older target repos.
- Prompt improvements require an explicit adoption path rather than silent replacement.
- Future work should add dry-run diff and backup support for users who want guided prompt updates.
